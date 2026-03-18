package sqlite

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"linuxdospace/backend/internal/model"
	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/timeutil"
)

type UpsertEmailCatchAllAccessInput = storage.UpsertEmailCatchAllAccessInput
type ConsumeEmailCatchAllInput = storage.ConsumeEmailCatchAllInput
type RefundEmailCatchAllInput = storage.RefundEmailCatchAllInput

// GetEmailCatchAllAccessByUser loads the mutable catch-all delivery state for
// one user.
func (s *Store) GetEmailCatchAllAccessByUser(ctx context.Context, userID int64) (model.EmailCatchAllAccess, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    user_id,
    subscription_expires_at,
    remaining_count,
    temporary_reward_count,
    temporary_reward_expires_at,
    daily_limit_override,
    created_at,
    updated_at
FROM email_catch_all_access
WHERE user_id = ?
`, userID)
	return scanEmailCatchAllAccess(row)
}

// UpsertEmailCatchAllAccess inserts or updates one user's mutable catch-all
// delivery state.
func (s *Store) UpsertEmailCatchAllAccess(ctx context.Context, input UpsertEmailCatchAllAccessInput) (model.EmailCatchAllAccess, error) {
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
INSERT INTO email_catch_all_access (
    user_id,
    subscription_expires_at,
    remaining_count,
    temporary_reward_count,
    temporary_reward_expires_at,
    daily_limit_override,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    subscription_expires_at = excluded.subscription_expires_at,
    remaining_count = excluded.remaining_count,
    temporary_reward_count = excluded.temporary_reward_count,
    temporary_reward_expires_at = excluded.temporary_reward_expires_at,
    daily_limit_override = excluded.daily_limit_override,
    updated_at = excluded.updated_at
RETURNING
    user_id,
    subscription_expires_at,
    remaining_count,
    temporary_reward_count,
    temporary_reward_expires_at,
    daily_limit_override,
    created_at,
    updated_at
`,
		input.UserID,
		formatNullableTime(input.SubscriptionExpiresAt),
		input.RemainingCount,
		input.TemporaryRewardCount,
		formatNullableTime(input.TemporaryRewardExpiresAt),
		input.DailyLimitOverride,
		formatTime(now),
		formatTime(now),
	)
	return scanEmailCatchAllAccess(row)
}

// GetEmailCatchAllDailyUsage loads the Shanghai-day catch-all consumption row for
// one user.
func (s *Store) GetEmailCatchAllDailyUsage(ctx context.Context, userID int64, usageDate string) (model.EmailCatchAllDailyUsage, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    user_id,
    usage_date,
    used_count,
    created_at,
    updated_at
FROM email_catch_all_daily_usage
WHERE user_id = ? AND usage_date = ?
`, userID, usageDate)
	return scanEmailCatchAllDailyUsage(row)
}

// ConsumeEmailCatchAll atomically reserves catch-all delivery usage. Active
// subscriptions win first; otherwise the remaining prepaid count is decremented.
func (s *Store) ConsumeEmailCatchAll(ctx context.Context, input ConsumeEmailCatchAllInput) (model.EmailCatchAllConsumeResult, error) {
	if input.Count <= 0 {
		return model.EmailCatchAllConsumeResult{}, fmt.Errorf("catch-all consume count must be positive")
	}

	now := input.Now.UTC()
	usageDate := timeutil.ShanghaiDayKey(now)
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.EmailCatchAllConsumeResult{}, err
	}
	defer tx.Rollback()

	access, accessExists, err := getEmailCatchAllAccessTx(ctx, tx, input.UserID)
	if err != nil {
		return model.EmailCatchAllConsumeResult{}, err
	}
	if !accessExists {
		access = model.EmailCatchAllAccess{
			UserID:         input.UserID,
			RemainingCount: 0,
		}
	}
	if normalizedAccess, changed := normalizeTemporaryRewardAccess(access, now); changed {
		access = normalizedAccess
		if accessExists {
			if err := upsertEmailCatchAllAccessTx(ctx, tx, input.UserID, access.SubscriptionExpiresAt, access.RemainingCount, access.TemporaryRewardCount, access.TemporaryRewardExpiresAt, access.DailyLimitOverride, now); err != nil {
				return model.EmailCatchAllConsumeResult{}, err
			}
		}
	}

	usage, usageExists, err := getEmailCatchAllDailyUsageTx(ctx, tx, input.UserID, usageDate)
	if err != nil {
		return model.EmailCatchAllConsumeResult{}, err
	}
	if !usageExists {
		usage = model.EmailCatchAllDailyUsage{
			UserID:    input.UserID,
			UsageDate: usageDate,
			UsedCount: 0,
		}
	}

	effectiveDailyLimit := input.DefaultDailyLimit
	if access.DailyLimitOverride != nil {
		effectiveDailyLimit = *access.DailyLimitOverride
	}
	if effectiveDailyLimit <= 0 {
		effectiveDailyLimit = 1_000_000
	}
	if usage.UsedCount+input.Count > effectiveDailyLimit {
		return model.EmailCatchAllConsumeResult{}, storage.ErrEmailCatchAllDailyLimitExceeded
	}

	consumedMode := "subscription"
	consumedPermanentCount := int64(0)
	consumedTemporaryRewardCount := int64(0)
	consumedTemporaryRewardExpiresAt := access.ActiveTemporaryRewardExpiry(now)
	subscriptionActive := access.SubscriptionExpiresAt != nil && access.SubscriptionExpiresAt.After(now)
	if !subscriptionActive {
		remainingCountToConsume := input.Count
		activeTemporaryRewardCount := access.ActiveTemporaryRewardCount(now)
		if activeTemporaryRewardCount > 0 {
			consumedTemporaryRewardCount = minInt64(activeTemporaryRewardCount, remainingCountToConsume)
			remainingCountToConsume -= consumedTemporaryRewardCount
		}
		if access.RemainingCount < remainingCountToConsume {
			return model.EmailCatchAllConsumeResult{}, storage.ErrEmailCatchAllInsufficientRemainingCount
		}
		consumedPermanentCount = remainingCountToConsume
		access.RemainingCount -= consumedPermanentCount
		access.TemporaryRewardCount = activeTemporaryRewardCount - consumedTemporaryRewardCount
		if access.TemporaryRewardCount <= 0 {
			access.TemporaryRewardCount = 0
			access.TemporaryRewardExpiresAt = nil
		}
		switch {
		case consumedTemporaryRewardCount > 0 && consumedPermanentCount > 0:
			consumedMode = "mixed"
		case consumedTemporaryRewardCount > 0:
			consumedMode = "temporary_reward"
		default:
			consumedMode = "quantity"
		}
		if err := upsertEmailCatchAllAccessTx(ctx, tx, input.UserID, access.SubscriptionExpiresAt, access.RemainingCount, access.TemporaryRewardCount, access.TemporaryRewardExpiresAt, access.DailyLimitOverride, now); err != nil {
			return model.EmailCatchAllConsumeResult{}, err
		}
		access.UpdatedAt = now
	}

	usage, err = upsertEmailCatchAllDailyUsageTx(ctx, tx, input.UserID, usageDate, input.Count, effectiveDailyLimit, now)
	if err != nil {
		if err == sql.ErrNoRows {
			return model.EmailCatchAllConsumeResult{}, storage.ErrEmailCatchAllDailyLimitExceeded
		}
		return model.EmailCatchAllConsumeResult{}, err
	}

	if err := tx.Commit(); err != nil {
		return model.EmailCatchAllConsumeResult{}, err
	}

	return model.EmailCatchAllConsumeResult{
		Access:                           access,
		DailyUsage:                       usage,
		EffectiveDailyLimit:              effectiveDailyLimit,
		ConsumedMode:                     consumedMode,
		ConsumedPermanentCount:           consumedPermanentCount,
		ConsumedTemporaryRewardCount:     consumedTemporaryRewardCount,
		ConsumedTemporaryRewardExpiresAt: consumedTemporaryRewardExpiresAt,
	}, nil
}

// upsertEmailCatchAllDailyUsageTx atomically creates or increments one
// per-user Shanghai-day usage row while enforcing the effective daily limit
// inside the same write statement. This closes the read-then-update race that
// could otherwise surface duplicate inserts or cap overshoot under contention.
func upsertEmailCatchAllDailyUsageTx(ctx context.Context, tx *sql.Tx, userID int64, usageDate string, count int64, effectiveDailyLimit int64, now time.Time) (model.EmailCatchAllDailyUsage, error) {
	row := tx.QueryRowContext(ctx, `
INSERT INTO email_catch_all_daily_usage (
    user_id,
    usage_date,
    used_count,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?)
ON CONFLICT(user_id, usage_date) DO UPDATE SET
    used_count = email_catch_all_daily_usage.used_count + excluded.used_count,
    updated_at = excluded.updated_at
WHERE email_catch_all_daily_usage.used_count + excluded.used_count <= ?
RETURNING
    user_id,
    usage_date,
    used_count,
    created_at,
    updated_at
`,
		userID,
		usageDate,
		count,
		formatTime(now),
		formatTime(now),
		effectiveDailyLimit,
	)
	return scanEmailCatchAllDailyUsage(row)
}

// RefundEmailCatchAll rolls back one previously reserved catch-all usage unit.
// It is used when SMTP forwarding fails after quota was reserved successfully.
func (s *Store) RefundEmailCatchAll(ctx context.Context, input RefundEmailCatchAllInput) error {
	if input.Count <= 0 {
		return fmt.Errorf("catch-all refund count must be positive")
	}
	if input.UsageDate == "" {
		return fmt.Errorf("catch-all refund usage date is required")
	}

	now := input.Now.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	access, accessExists, err := getEmailCatchAllAccessTx(ctx, tx, input.UserID)
	if err != nil {
		return err
	}
	if !accessExists {
		return fmt.Errorf("catch-all access state does not exist")
	}
	if normalizedAccess, changed := normalizeTemporaryRewardAccess(access, now); changed {
		access = normalizedAccess
		if err := upsertEmailCatchAllAccessTx(ctx, tx, input.UserID, access.SubscriptionExpiresAt, access.RemainingCount, access.TemporaryRewardCount, access.TemporaryRewardExpiresAt, access.DailyLimitOverride, now); err != nil {
			return err
		}
	}

	usage, usageExists, err := getEmailCatchAllDailyUsageTx(ctx, tx, input.UserID, input.UsageDate)
	if err != nil {
		return err
	}
	if !usageExists || usage.UsedCount < input.Count {
		return fmt.Errorf("catch-all daily usage cannot be refunded")
	}

	shouldRestoreTemporaryReward := input.ConsumedTemporaryRewardCount > 0 &&
		input.TemporaryRewardExpiresAt != nil &&
		input.TemporaryRewardExpiresAt.After(now)
	if input.ConsumedPermanentCount > 0 {
		access.RemainingCount += input.ConsumedPermanentCount
	}
	if shouldRestoreTemporaryReward {
		currentExpiry := access.ActiveTemporaryRewardExpiry(now)
		switch {
		case currentExpiry == nil:
			expiry := input.TemporaryRewardExpiresAt.UTC()
			access.TemporaryRewardExpiresAt = &expiry
			access.TemporaryRewardCount = input.ConsumedTemporaryRewardCount
		case currentExpiry.Equal(input.TemporaryRewardExpiresAt.UTC()):
			access.TemporaryRewardCount += input.ConsumedTemporaryRewardCount
		default:
			return fmt.Errorf("temporary reward expiry mismatch during refund")
		}
	}
	if input.ConsumedPermanentCount > 0 || shouldRestoreTemporaryReward {
		if err := upsertEmailCatchAllAccessTx(ctx, tx, input.UserID, access.SubscriptionExpiresAt, access.RemainingCount, access.TemporaryRewardCount, access.TemporaryRewardExpiresAt, access.DailyLimitOverride, now); err != nil {
			return err
		}
	}

	if _, err := tx.ExecContext(ctx, `
UPDATE email_catch_all_daily_usage
SET used_count = used_count - ?, updated_at = ?
WHERE user_id = ? AND usage_date = ?
`, input.Count, formatTime(now), input.UserID, input.UsageDate); err != nil {
		return err
	}

	return tx.Commit()
}

// getEmailCatchAllAccessTx loads one access row inside an open transaction.
func getEmailCatchAllAccessTx(ctx context.Context, tx *sql.Tx, userID int64) (model.EmailCatchAllAccess, bool, error) {
	row := tx.QueryRowContext(ctx, `
SELECT
    user_id,
    subscription_expires_at,
    remaining_count,
    temporary_reward_count,
    temporary_reward_expires_at,
    daily_limit_override,
    created_at,
    updated_at
FROM email_catch_all_access
WHERE user_id = ?
`, userID)

	item, err := scanEmailCatchAllAccess(row)
	if err != nil {
		if storage.IsNotFound(err) {
			return model.EmailCatchAllAccess{}, false, nil
		}
		return model.EmailCatchAllAccess{}, false, err
	}
	return item, true, nil
}

// getEmailCatchAllDailyUsageTx loads one per-day usage row inside an open
// transaction.
func getEmailCatchAllDailyUsageTx(ctx context.Context, tx *sql.Tx, userID int64, usageDate string) (model.EmailCatchAllDailyUsage, bool, error) {
	row := tx.QueryRowContext(ctx, `
SELECT
    user_id,
    usage_date,
    used_count,
    created_at,
    updated_at
FROM email_catch_all_daily_usage
WHERE user_id = ? AND usage_date = ?
`, userID, usageDate)

	item, err := scanEmailCatchAllDailyUsage(row)
	if err != nil {
		if storage.IsNotFound(err) {
			return model.EmailCatchAllDailyUsage{}, false, nil
		}
		return model.EmailCatchAllDailyUsage{}, false, err
	}
	return item, true, nil
}

// scanEmailCatchAllAccess maps one access row into the model package.
func scanEmailCatchAllAccess(scanner interface{ Scan(dest ...any) error }) (model.EmailCatchAllAccess, error) {
	var item model.EmailCatchAllAccess
	var subscriptionExpiresAt sql.NullString
	var temporaryRewardExpiresAt sql.NullString
	var dailyLimitOverride sql.NullInt64
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&item.UserID,
		&subscriptionExpiresAt,
		&item.RemainingCount,
		&item.TemporaryRewardCount,
		&temporaryRewardExpiresAt,
		&dailyLimitOverride,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.EmailCatchAllAccess{}, err
	}

	if dailyLimitOverride.Valid {
		value := dailyLimitOverride.Int64
		item.DailyLimitOverride = &value
	}

	item.SubscriptionExpiresAt, err = parseNullableTime(subscriptionExpiresAt)
	if err != nil {
		return model.EmailCatchAllAccess{}, err
	}
	item.TemporaryRewardExpiresAt, err = parseNullableTime(temporaryRewardExpiresAt)
	if err != nil {
		return model.EmailCatchAllAccess{}, err
	}
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.EmailCatchAllAccess{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.EmailCatchAllAccess{}, err
	}
	return item, nil
}

// upsertEmailCatchAllAccessTx persists one fully normalized mutable catch-all
// access snapshot inside the current transaction so storage paths can share one
// consistent write contract.
func upsertEmailCatchAllAccessTx(ctx context.Context, tx *sql.Tx, userID int64, subscriptionExpiresAt *time.Time, remainingCount int64, temporaryRewardCount int64, temporaryRewardExpiresAt *time.Time, dailyLimitOverride *int64, now time.Time) error {
	_, err := tx.ExecContext(ctx, `
INSERT INTO email_catch_all_access (
    user_id,
    subscription_expires_at,
    remaining_count,
    temporary_reward_count,
    temporary_reward_expires_at,
    daily_limit_override,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(user_id) DO UPDATE SET
    subscription_expires_at = excluded.subscription_expires_at,
    remaining_count = excluded.remaining_count,
    temporary_reward_count = excluded.temporary_reward_count,
    temporary_reward_expires_at = excluded.temporary_reward_expires_at,
    daily_limit_override = excluded.daily_limit_override,
    updated_at = excluded.updated_at
`,
		userID,
		formatNullableTime(subscriptionExpiresAt),
		remainingCount,
		temporaryRewardCount,
		formatNullableTime(temporaryRewardExpiresAt),
		normalizeNullableInt64(dailyLimitOverride),
		formatTime(now),
		formatTime(now),
	)
	return err
}

// normalizeTemporaryRewardAccess clears expired temporary-reward state from the
// mutable access snapshot and reports whether a persistence write is needed.
func normalizeTemporaryRewardAccess(access model.EmailCatchAllAccess, now time.Time) (model.EmailCatchAllAccess, bool) {
	normalized := access.NormalizeTemporaryReward(now)
	changed := normalized.TemporaryRewardCount != access.TemporaryRewardCount
	if !changed && ((normalized.TemporaryRewardExpiresAt == nil) != (access.TemporaryRewardExpiresAt == nil)) {
		changed = true
	}
	return normalized, changed
}

// minInt64 keeps the mixed temporary/permanent balance deduction logic readable.
func minInt64(left int64, right int64) int64 {
	if left < right {
		return left
	}
	return right
}

// scanEmailCatchAllDailyUsage maps one per-day usage row into the model
// package.
func scanEmailCatchAllDailyUsage(scanner interface{ Scan(dest ...any) error }) (model.EmailCatchAllDailyUsage, error) {
	var item model.EmailCatchAllDailyUsage
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&item.UserID,
		&item.UsageDate,
		&item.UsedCount,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.EmailCatchAllDailyUsage{}, err
	}
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.EmailCatchAllDailyUsage{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.EmailCatchAllDailyUsage{}, err
	}
	return item, nil
}
