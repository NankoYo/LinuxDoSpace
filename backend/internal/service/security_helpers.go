package service

import (
	"log"
	"time"
)

// AdminVerificationIsFresh reports whether one administrator password check is
// still inside the configured freshness window.
func AdminVerificationIsFresh(verifiedAt *time.Time, ttl time.Duration, now time.Time) bool {
	if verifiedAt == nil {
		return false
	}
	if ttl <= 0 {
		return false
	}
	return verifiedAt.UTC().Add(ttl).After(now.UTC())
}

// logAuditWriteFailure downgrades post-success audit write failures to operator
// logs so successful user-visible mutations do not turn into false 500 errors.
func logAuditWriteFailure(action string, err error) {
	if err == nil {
		return
	}
	log.Printf("audit log write failed for %s: %v", action, err)
}
