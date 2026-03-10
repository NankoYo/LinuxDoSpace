package model

import "time"

// EmailTarget stores one forwarding destination email that has been bound to a
// specific LinuxDoSpace user. The binding is global on the email address so a
// second user cannot silently reuse the same external inbox.
type EmailTarget struct {
	ID                          int64      `json:"id"`
	OwnerUserID                 int64      `json:"owner_user_id"`
	Email                       string     `json:"email"`
	CloudflareAddressID         string     `json:"cloudflare_address_id"`
	VerifiedAt                  *time.Time `json:"verified_at,omitempty"`
	LastVerificationSentAt      *time.Time `json:"last_verification_sent_at,omitempty"`
	CreatedAt                   time.Time  `json:"created_at"`
	UpdatedAt                   time.Time  `json:"updated_at"`
}
