package storage

import (
	"database/sql"
	"errors"
)

var (
	// ErrEmailCatchAllDailyLimitExceeded means one catch-all delivery would push
	// the user over the effective single-day maximum.
	ErrEmailCatchAllDailyLimitExceeded = errors.New("email catch-all daily limit exceeded")

	// ErrEmailCatchAllInsufficientRemainingCount means the user has no active
	// subscription and not enough prepaid remaining count to accept the mail.
	ErrEmailCatchAllInsufficientRemainingCount = errors.New("email catch-all remaining count is insufficient")

	// ErrMailForwardDailyLimitExceeded means one user has already consumed the
	// hidden per-day mailbox forwarding cap enforced by the local SMTP relay.
	ErrMailForwardDailyLimitExceeded = errors.New("mail forward daily limit exceeded")

	// ErrAllocationQuotaExceeded means one user attempted to allocate more
	// namespaces on one managed root than their effective quota allows.
	ErrAllocationQuotaExceeded = errors.New("allocation quota exceeded")

	// ErrEmailRouteOwnershipConflict means one caller attempted to overwrite an
	// existing email route owned by another user.
	ErrEmailRouteOwnershipConflict = errors.New("email route ownership conflict")

	// ErrEmailTargetVerificationExpired means the verification token existed but
	// expired before it was atomically consumed.
	ErrEmailTargetVerificationExpired = errors.New("email target verification expired")

	// ErrEmailTargetVerificationRateLimited means the caller exceeded the
	// persistent resend/send thresholds for the owner or target inbox.
	ErrEmailTargetVerificationRateLimited = errors.New("email target verification rate limited")

	// ErrPOWChallengeDailyLimitExceeded means the current user already claimed
	// the maximum number of proof-of-work rewards allowed for the UTC day.
	ErrPOWChallengeDailyLimitExceeded = errors.New("pow challenge daily limit exceeded")

	// ErrPOWChallengeNotActive means the submitted challenge was already claimed
	// or superseded before the caller attempted to redeem it.
	ErrPOWChallengeNotActive = errors.New("pow challenge is not active")
)

// IsNotFound reports whether one storage call failed only because the target
// row does not exist.
//
// The helper currently normalizes to `sql.ErrNoRows` so existing SQLite code
// and future PostgreSQL code can share one storage-agnostic check.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
