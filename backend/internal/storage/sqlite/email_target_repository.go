package sqlite

import (
	"context"
	"database/sql"
	"time"

	"linuxdospace/backend/internal/model"
)

// CreateEmailTargetInput describes one new user-owned forwarding destination
// email that should be bound to a local account.
type CreateEmailTargetInput struct {
	OwnerUserID            int64
	Email                  string
	CloudflareAddressID    string
	VerifiedAt             *time.Time
	LastVerificationSentAt *time.Time
}

// UpdateEmailTargetInput describes the mutable synchronization state kept for
// one stored forwarding destination.
type UpdateEmailTargetInput struct {
	ID                     int64
	CloudflareAddressID    string
	VerifiedAt             *time.Time
	LastVerificationSentAt *time.Time
}

// ListEmailTargetsByOwner returns every forwarding destination currently bound
// to one local user.
func (s *Store) ListEmailTargetsByOwner(ctx context.Context, ownerUserID int64) ([]model.EmailTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
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
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
FROM email_targets
WHERE email = ?
`, email)
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
    verified_at,
    last_verification_sent_at,
    created_at,
    updated_at
) VALUES (?, ?, ?, ?, ?, ?, ?)
RETURNING id
`,
		input.OwnerUserID,
		input.Email,
		input.CloudflareAddressID,
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
    verified_at = ?,
    last_verification_sent_at = ?,
    updated_at = ?
WHERE id = ?
RETURNING id
`,
		input.CloudflareAddressID,
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

// getEmailTargetByID loads one stored forwarding destination by its local id.
func (s *Store) getEmailTargetByID(ctx context.Context, id int64) (model.EmailTarget, error) {
	row := s.db.QueryRowContext(ctx, `
SELECT
    id,
    owner_user_id,
    email,
    cloudflare_address_id,
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
	var verifiedAt sql.NullString
	var lastVerificationSentAt sql.NullString
	var createdAt string
	var updatedAt string

	err := scanner.Scan(
		&item.ID,
		&item.OwnerUserID,
		&item.Email,
		&item.CloudflareAddressID,
		&verifiedAt,
		&lastVerificationSentAt,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
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
