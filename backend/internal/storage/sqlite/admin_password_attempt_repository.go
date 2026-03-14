package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"linuxdospace/backend/internal/model"
)

// GetAdminPasswordAttempt loads one persisted admin-password limiter bucket.
func (s *Store) GetAdminPasswordAttempt(ctx context.Context, bucketType string, bucketKey string) (model.AdminPasswordAttempt, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    bucket_type,
    bucket_key,
    failure_count,
    blocked_until,
    last_seen_at,
    created_at,
    updated_at
FROM admin_password_attempts
WHERE bucket_type = ? AND bucket_key = ?
`, strings.TrimSpace(bucketType), strings.TrimSpace(bucketKey))
	return scanAdminPasswordAttempt(row)
}

// RegisterAdminPasswordFailure atomically increments one limiter bucket and
// starts a block window once the threshold is reached.
func (s *Store) RegisterAdminPasswordFailure(ctx context.Context, bucketType string, bucketKey string, maxFailures int, blockDuration time.Duration, now time.Time) (model.AdminPasswordAttempt, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	defer tx.Rollback()

	normalizedType := strings.TrimSpace(bucketType)
	normalizedKey := strings.TrimSpace(bucketKey)
	if normalizedType == "" || normalizedKey == "" {
		return model.AdminPasswordAttempt{}, sql.ErrNoRows
	}

	row := tx.QueryRowContext(ctx, `
SELECT
    bucket_type,
    bucket_key,
    failure_count,
    blocked_until,
    last_seen_at,
    created_at,
    updated_at
FROM admin_password_attempts
WHERE bucket_type = ? AND bucket_key = ?
`, normalizedType, normalizedKey)

	item, err := scanAdminPasswordAttempt(row)
	switch {
	case err == nil:
		if item.BlockedUntil != nil && !item.BlockedUntil.After(now) {
			item.BlockedUntil = nil
			item.FailureCount = 0
		}
	case IsNotFound(err):
		item = model.AdminPasswordAttempt{
			BucketType:   normalizedType,
			BucketKey:    normalizedKey,
			FailureCount: 0,
			LastSeenAt:   now.UTC(),
			CreatedAt:    now.UTC(),
			UpdatedAt:    now.UTC(),
		}
	default:
		return model.AdminPasswordAttempt{}, err
	}

	item.FailureCount++
	item.LastSeenAt = now.UTC()
	item.UpdatedAt = now.UTC()
	if item.FailureCount >= maxFailures {
		blockedUntil := now.UTC().Add(blockDuration)
		item.BlockedUntil = &blockedUntil
		item.FailureCount = 0
	}

	row = tx.QueryRowContext(ctx, `
INSERT INTO admin_password_attempts (
    bucket_type,
    bucket_key,
    failure_count,
    blocked_until,
    last_seen_at,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(bucket_type, bucket_key) DO UPDATE SET
    failure_count = excluded.failure_count,
    blocked_until = excluded.blocked_until,
    last_seen_at = excluded.last_seen_at,
    updated_at = excluded.updated_at
RETURNING
    bucket_type,
    bucket_key,
    failure_count,
    blocked_until,
    last_seen_at,
    created_at,
    updated_at
`, item.BucketType, item.BucketKey, item.FailureCount, formatNullableTime(item.BlockedUntil), formatTime(item.LastSeenAt), formatTime(item.CreatedAt), formatTime(item.UpdatedAt))

	item, err = scanAdminPasswordAttempt(row)
	if err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	if err := tx.Commit(); err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	return item, nil
}

// DeleteAdminPasswordAttempt clears one persisted limiter bucket.
func (s *Store) DeleteAdminPasswordAttempt(ctx context.Context, bucketType string, bucketKey string) error {
	_, err := s.db.ExecContext(ctx, `
DELETE FROM admin_password_attempts
WHERE bucket_type = ? AND bucket_key = ?
`, strings.TrimSpace(bucketType), strings.TrimSpace(bucketKey))
	return err
}

// DeleteStaleAdminPasswordAttempts prunes long-idle, currently unblocked
// limiter buckets so the persistence table stays bounded.
func (s *Store) DeleteStaleAdminPasswordAttempts(ctx context.Context, cutoff time.Time, now time.Time) error {
	_, err := s.db.ExecContext(ctx, `
DELETE FROM admin_password_attempts
WHERE last_seen_at < ?
  AND (blocked_until IS NULL OR blocked_until = '' OR blocked_until <= ?)
`, formatTime(cutoff.UTC()), formatTime(now.UTC()))
	return err
}

// scanAdminPasswordAttempt maps one limiter row into the model package.
func scanAdminPasswordAttempt(scanner interface{ Scan(dest ...any) error }) (model.AdminPasswordAttempt, error) {
	var item model.AdminPasswordAttempt
	var blockedUntil sql.NullString
	var lastSeenAt string
	var createdAt string
	var updatedAt string

	if err := scanner.Scan(
		&item.BucketType,
		&item.BucketKey,
		&item.FailureCount,
		&blockedUntil,
		&lastSeenAt,
		&createdAt,
		&updatedAt,
	); err != nil {
		return model.AdminPasswordAttempt{}, err
	}

	var err error
	item.BlockedUntil, err = parseNullableTime(blockedUntil)
	if err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	if item.LastSeenAt, err = parseTime(lastSeenAt); err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	if item.CreatedAt, err = parseTime(createdAt); err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	if item.UpdatedAt, err = parseTime(updatedAt); err != nil {
		return model.AdminPasswordAttempt{}, err
	}
	return item, nil
}
