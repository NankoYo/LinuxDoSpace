package postgres

import (
	"context"
	"database/sql"
	"time"

	"linuxdospace/backend/internal/model"
	"linuxdospace/backend/internal/storage"
)

type CreateEmailTargetInput = storage.CreateEmailTargetInput
type UpdateEmailTargetInput = storage.UpdateEmailTargetInput
type PrepareEmailTargetVerificationSendInput = storage.PrepareEmailTargetVerificationSendInput

const emailTargetVerificationAttemptRetention = 30 * 24 * time.Hour

// ListEmailTargetsByOwner returns every forwarding destination currently bound
// to one local user.
func (s *Store) ListEmailTargetsByOwner(ctx context.Context, ownerUserID int64) ([]model.EmailTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE owner_user_id = ?
ORDER BY
    CASE WHEN verified_at IS NULL THEN 1 ELSE 0 END,
    updated_at DESC,
    id DESC
`, ownerUserID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]model.EmailTarget, 0, 8)
	for rows.Next() {
		item, scanErr := scanEmailTarget(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return items, nil
}

// GetEmailTargetByEmail loads one globally-unique forwarding destination by
// the external inbox address.
func (s *Store) GetEmailTargetByEmail(ctx context.Context, email string) (model.EmailTarget, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE email = ?
`, email)
	return scanEmailTarget(row)
}

// GetEmailTargetByVerificationTokenHash loads one pending forwarding target by
// the hashed verification token sent inside the platform-owned email message.
func (s *Store) GetEmailTargetByVerificationTokenHash(ctx context.Context, tokenHash string) (model.EmailTarget, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE verification_token_hash = ?
`, tokenHash)
	return scanEmailTarget(row)
}

// CreateEmailTarget inserts one new forwarding destination ownership row.
func (s *Store) CreateEmailTarget(ctx context.Context, input CreateEmailTargetInput) (model.EmailTarget, error) {
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
INSERT INTO email_targets (
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
	last_verification_sent_at,
	created_at,
	updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
RETURNING id
`,
		input.OwnerUserID,
		input.Email,
		input.CloudflareAddressID,
		input.VerificationTokenHash,
		formatNullableTime(input.VerificationExpiresAt),
		formatNullableTime(input.VerifiedAt),
		formatNullableTime(input.LastVerificationSentAt),
		formatTime(now),
		formatTime(now),
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return model.EmailTarget{}, err
	}
	return s.getEmailTargetByID(ctx, id)
}

// UpdateEmailTarget refreshes the Cloudflare synchronization state for one
// existing forwarding destination ownership row.
func (s *Store) UpdateEmailTarget(ctx context.Context, input UpdateEmailTargetInput) (model.EmailTarget, error) {
	now := time.Now().UTC()
	row := s.db.QueryRowContext(ctx, `
UPDATE email_targets
SET
    cloudflare_address_id = ?,
    verification_token_hash = ?,
    verification_expires_at = ?,
    verified_at = ?,
    last_verification_sent_at = ?,
    updated_at = ?
WHERE id = ?
RETURNING id
`,
		input.CloudflareAddressID,
		input.VerificationTokenHash,
		formatNullableTime(input.VerificationExpiresAt),
		formatNullableTime(input.VerifiedAt),
		formatNullableTime(input.LastVerificationSentAt),
		formatTime(now),
		input.ID,
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return model.EmailTarget{}, err
	}
	return s.getEmailTargetByID(ctx, id)
}

// PrepareEmailTargetVerificationSend atomically reserves one verification-send
// slot, persists the fresh token, and updates the target row before the actual
// outbound email leaves the process.
func (s *Store) PrepareEmailTargetVerificationSend(ctx context.Context, input PrepareEmailTargetVerificationSendInput) (model.EmailTarget, error) {
	now := input.PreparedAt.UTC()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.EmailTarget{}, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE id = ?
FOR UPDATE
`, input.ID)
	item, err := scanEmailTarget(row)
	if err != nil {
		return model.EmailTarget{}, err
	}
	if item.OwnerUserID != input.OwnerUserID || item.Email != input.Email {
		return model.EmailTarget{}, sql.ErrNoRows
	}
	shortOwnerCount, dailyOwnerCount, shortTargetCount, dailyTargetCount, err := countEmailTargetVerificationAttemptsTx(ctx, tx, input.OwnerUserID, input.Email, input.ShortWindowStart.UTC(), input.DailyWindowStart.UTC())
	if err != nil {
		return model.EmailTarget{}, err
	}
	if input.OwnerShortLimit > 0 && shortOwnerCount >= input.OwnerShortLimit {
		return model.EmailTarget{}, storage.ErrEmailTargetVerificationRateLimited
	}
	if input.OwnerDailyLimit > 0 && dailyOwnerCount >= input.OwnerDailyLimit {
		return model.EmailTarget{}, storage.ErrEmailTargetVerificationRateLimited
	}
	if input.TargetShortLimit > 0 && shortTargetCount >= input.TargetShortLimit {
		return model.EmailTarget{}, storage.ErrEmailTargetVerificationRateLimited
	}
	if input.TargetDailyLimit > 0 && dailyTargetCount >= input.TargetDailyLimit {
		return model.EmailTarget{}, storage.ErrEmailTargetVerificationRateLimited
	}

	cleanupBefore := now.Add(-emailTargetVerificationAttemptRetention)
	if _, err := tx.ExecContext(ctx, `
DELETE FROM email_target_verification_attempts
WHERE prepared_at < ?
`, formatTime(cleanupBefore)); err != nil {
		return model.EmailTarget{}, err
	}

	if _, err := tx.ExecContext(ctx, `
INSERT INTO email_target_verification_attempts (
    owner_user_id,
    email,
    prepared_at,
    created_at
) VALUES (?, ?, ?, ?)
`,
		input.OwnerUserID,
		input.Email,
		formatTime(now),
		formatTime(now),
	); err != nil {
		return model.EmailTarget{}, err
	}

	row = tx.QueryRowContext(ctx, `
UPDATE email_targets
SET
    cloudflare_address_id = '',
    verification_token_hash = ?,
    verification_expires_at = ?,
    verified_at = NULL,
    last_verification_sent_at = ?,
    updated_at = ?
WHERE id = ?
RETURNING id
`,
		input.VerificationTokenHash,
		formatNullableTime(input.VerificationExpiresAt),
		formatTime(now),
		formatTime(now),
		input.ID,
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		return model.EmailTarget{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.EmailTarget{}, err
	}
	return s.getEmailTargetByID(ctx, id)
}

// countEmailTargetVerificationAttemptsTx returns owner- and target-scoped send
// counts across the short and daily windows used by resend protection.
func countEmailTargetVerificationAttemptsTx(ctx context.Context, tx *queryTx, ownerUserID int64, email string, shortWindowStart time.Time, dailyWindowStart time.Time) (int, int, int, int, error) {
	row := tx.QueryRowContext(ctx, `
SELECT
    COALESCE(SUM(CASE WHEN owner_user_id = ? AND prepared_at >= ? THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN owner_user_id = ? AND prepared_at >= ? THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN email = ? AND prepared_at >= ? THEN 1 ELSE 0 END), 0),
    COALESCE(SUM(CASE WHEN email = ? AND prepared_at >= ? THEN 1 ELSE 0 END), 0)
FROM email_target_verification_attempts
WHERE owner_user_id = ? OR email = ?
`,
		ownerUserID,
		formatTime(shortWindowStart),
		ownerUserID,
		formatTime(dailyWindowStart),
		email,
		formatTime(shortWindowStart),
		email,
		formatTime(dailyWindowStart),
		ownerUserID,
		email,
	)

	var shortOwnerCount int
	var dailyOwnerCount int
	var shortTargetCount int
	var dailyTargetCount int
	if err := row.Scan(&shortOwnerCount, &dailyOwnerCount, &shortTargetCount, &dailyTargetCount); err != nil {
		return 0, 0, 0, 0, err
	}
	return shortOwnerCount, dailyOwnerCount, shortTargetCount, dailyTargetCount, nil
}

// ConsumeEmailTargetVerificationToken atomically verifies one still-valid token
// exactly once and clears it so concurrent replays cannot both succeed.
func (s *Store) ConsumeEmailTargetVerificationToken(ctx context.Context, tokenHash string, verifiedAt time.Time) (model.EmailTarget, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.EmailTarget{}, err
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE verification_token_hash = ?
FOR UPDATE
`, tokenHash)
	item, err := scanEmailTarget(row)
	if err != nil {
		return model.EmailTarget{}, err
	}

	now := verifiedAt.UTC()
	if item.VerificationExpiresAt == nil || !item.VerificationExpiresAt.After(now) {
		if _, err := tx.ExecContext(ctx, `
UPDATE email_targets
SET
    verification_token_hash = '',
    verification_expires_at = NULL,
    updated_at = ?
WHERE id = ?
`,
			formatTime(now),
			item.ID,
		); err != nil {
			return model.EmailTarget{}, err
		}
		if err := tx.Commit(); err != nil {
			return model.EmailTarget{}, err
		}
		return model.EmailTarget{}, storage.ErrEmailTargetVerificationExpired
	}

	row = tx.QueryRowContext(ctx, `
UPDATE email_targets
SET
    cloudflare_address_id = '',
    verification_token_hash = '',
    verification_expires_at = NULL,
    verified_at = ?,
    updated_at = ?
WHERE id = ? AND verification_token_hash = ?
RETURNING id
`,
		formatTime(now),
		formatTime(now),
		item.ID,
		tokenHash,
	)

	var id int64
	if err := row.Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return model.EmailTarget{}, sql.ErrNoRows
		}
		return model.EmailTarget{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.EmailTarget{}, err
	}
	return s.getEmailTargetByID(ctx, id)
}

// getEmailTargetByID loads one stored forwarding destination by its local id.
func (s *Store) getEmailTargetByID(ctx context.Context, id int64) (model.EmailTarget, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
    verification_token_hash,
    verification_expires_at,
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE id = ?
`, id)
	return scanEmailTarget(row)
}

// scanEmailTarget maps one email-target row into the model package.
func scanEmailTarget(scanner interface{ Scan(dest ...any) error }) (model.EmailTarget, error) {
	var item model.EmailTarget
	var verificationExpiresAt sql.NullString
	var verifiedAt sql.NullString
	var lastVerificationSentAt sql.NullString
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&item.ID,
		&item.OwnerUserID,
		&item.Email,
		&item.CloudflareAddressID,
		&item.VerificationTokenHash,
		&verificationExpiresAt,
		&verifiedAt,
		&lastVerificationSentAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return model.EmailTarget{}, err
	}

	if item.VerificationExpiresAt, err = parseNullableTime(verificationExpiresAt); err != nil {
		return model.EmailTarget{}, err
	}
	if item.VerifiedAt, err = parseNullableTime(verifiedAt); err != nil {
		return model.EmailTarget{}, err
	}
	if item.LastVerificationSentAt, err = parseNullableTime(lastVerificationSentAt); err != nil {
		return model.EmailTarget{}, err
	}
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.EmailTarget{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.EmailTarget{}, err
	}
	return item, nil
}
