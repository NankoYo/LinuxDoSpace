package service

import (
	"context"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"linuxdospace/backend/internal/storage"
)

// TestCreateMyEmailTargetCreatesOwnedPendingBinding verifies that a new target
// inbox is bound to the current user and that LinuxDoSpace sends its own
// verification email instead of relying on Cloudflare destination addresses.
func TestCreateMyEmailTargetCreatesOwnedPendingBinding(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 101, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, mailer := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	item, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "Owner@Example.com"})
	if err != nil {
		t.Fatalf("create email target: %v", err)
	}
	if item.Email != "owner@example.com" {
		t.Fatalf("expected normalized target email, got %q", item.Email)
	}
	if item.Verified {
		t.Fatalf("expected the new target to stay pending until the verification link is opened")
	}
	if len(mailer.sent) != 1 {
		t.Fatalf("expected one verification email to be sent, got %d", len(mailer.sent))
	}
	if mailer.sent[0].TargetEmail != "owner@example.com" {
		t.Fatalf("expected verification email target owner@example.com, got %q", mailer.sent[0].TargetEmail)
	}
	if !strings.Contains(mailer.sent[0].VerificationURL, "/v1/public/email-targets/verify?token=") {
		t.Fatalf("expected verification callback url, got %q", mailer.sent[0].VerificationURL)
	}

	stored, err := store.GetEmailTargetByEmail(ctx, "owner@example.com")
	if err != nil {
		t.Fatalf("load stored email target: %v", err)
	}
	if stored.OwnerUserID != user.ID {
		t.Fatalf("expected owner user id %d, got %d", user.ID, stored.OwnerUserID)
	}
	if strings.TrimSpace(stored.VerificationTokenHash) == "" {
		t.Fatalf("expected stored verification token hash to be persisted")
	}
	if stored.LastVerificationSentAt == nil {
		t.Fatalf("expected verification send timestamp to be persisted")
	}
}

// TestCreateMyEmailTargetRejectsOtherUsersTarget verifies that the same target
// email cannot be rebound by a second user.
func TestCreateMyEmailTargetRejectsOtherUsersTarget(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	userA := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 201, "alice")
	userB := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 202, "bob")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, _ := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	if _, err := service.CreateMyEmailTarget(ctx, userA, CreateMyEmailTargetRequest{Email: "shared@example.com"}); err != nil {
		t.Fatalf("create first email target: %v", err)
	}

	_, err := service.CreateMyEmailTarget(ctx, userB, CreateMyEmailTargetRequest{Email: "shared@example.com"})
	if err == nil {
		t.Fatalf("expected rebinding another user's target email to fail")
	}

	normalized := NormalizeError(err)
	if normalized.Code != "forbidden" {
		t.Fatalf("expected forbidden error, got %s: %v", normalized.Code, err)
	}
}

// TestCreateMyEmailTargetKeepsRecentPendingBindingDuringCooldown verifies that
// pressing the add button again during the resend cooldown returns the existing
// pending row instead of sending another verification email immediately.
func TestCreateMyEmailTargetKeepsRecentPendingBindingDuringCooldown(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 203, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	expiresAt := time.Now().UTC().Add(time.Hour)
	sentAt := time.Now().UTC()
	existing, err := store.CreateEmailTarget(ctx, storage.CreateEmailTargetInput{
		OwnerUserID:            user.ID,
		Email:                  "pending@example.com",
		CloudflareAddressID:    "",
		VerificationTokenHash:  "existing-hash",
		VerificationExpiresAt:  &expiresAt,
		VerifiedAt:             nil,
		LastVerificationSentAt: &sentAt,
	})
	if err != nil {
		t.Fatalf("seed recent pending email target: %v", err)
	}

	service, mailer := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	item, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "pending@example.com"})
	if err != nil {
		t.Fatalf("repeat create for recent pending target: %v", err)
	}
	if item.ID != existing.ID {
		t.Fatalf("expected existing local target id %d, got %d", existing.ID, item.ID)
	}
	if len(mailer.sent) != 0 {
		t.Fatalf("expected no additional verification email during cooldown, got %d sends", len(mailer.sent))
	}
}

// TestUpsertMyDefaultEmailRouteRejectsUnverifiedOwnedTarget verifies that the
// default mailbox can only use targets that have completed LinuxDoSpace's own
// verification flow.
func TestUpsertMyDefaultEmailRouteRejectsUnverifiedOwnedTarget(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 301, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, _ := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	target, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "pending@example.com"})
	if err != nil {
		t.Fatalf("create pending email target: %v", err)
	}

	_, err = service.UpsertMyDefaultEmailRoute(ctx, user, UpsertMyDefaultEmailRouteRequest{
		TargetEmail: target.Email,
		Enabled:     true,
	})
	if err == nil {
		t.Fatalf("expected unverified target email to be rejected")
	}

	normalized := NormalizeError(err)
	if normalized.Code != "conflict" {
		t.Fatalf("expected conflict error, got %s: %v", normalized.Code, err)
	}
	if !strings.Contains(normalized.Message, "尚未完成平台验证") {
		t.Fatalf("expected platform verification guidance in error message, got %q", normalized.Message)
	}
}

// TestVerifyEmailTargetMarksBindingUsableForForwarding verifies that consuming
// the emailed token marks the target as verified and immediately unlocks the
// default mailbox save flow.
func TestVerifyEmailTargetMarksBindingUsableForForwarding(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 401, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, mailer := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	target, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "verified@example.com"})
	if err != nil {
		t.Fatalf("create email target: %v", err)
	}

	token := mustExtractVerificationToken(t, mailer.sent[0].VerificationURL)
	if _, err := service.VerifyEmailTarget(ctx, token); err != nil {
		t.Fatalf("verify email target: %v", err)
	}

	view, err := service.UpsertMyDefaultEmailRoute(ctx, user, UpsertMyDefaultEmailRouteRequest{
		TargetEmail: target.Email,
		Enabled:     true,
	})
	if err != nil {
		t.Fatalf("save default email route with verified target: %v", err)
	}
	if !view.Configured || !view.Enabled {
		t.Fatalf("expected configured enabled default route, got %+v", view)
	}
	if view.TargetEmail != "verified@example.com" {
		t.Fatalf("expected verified target email to be saved, got %q", view.TargetEmail)
	}
}

// TestResendMyEmailTargetVerificationRejectsRapidRepeat verifies that the
// resend endpoint still protects the target inbox from tight retry loops.
func TestResendMyEmailTargetVerificationRejectsRapidRepeat(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 501, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, _ := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	item, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "rapid@example.com"})
	if err != nil {
		t.Fatalf("create pending email target: %v", err)
	}

	_, err = service.ResendMyEmailTargetVerification(ctx, user, item.ID)
	if err == nil {
		t.Fatalf("expected rapid resend to be throttled")
	}

	normalized := NormalizeError(err)
	if normalized.Code != "too_many_requests" {
		t.Fatalf("expected too_many_requests, got %s: %v", normalized.Code, err)
	}
}

// TestResendMyEmailTargetVerificationIssuesFreshTokenAfterCooldown verifies
// that a pending target gets a fresh token and a second outbound message once
// the resend cooldown has elapsed.
func TestResendMyEmailTargetVerificationIssuesFreshTokenAfterCooldown(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 601, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, mailer := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	item, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "pending@example.com"})
	if err != nil {
		t.Fatalf("create pending email target: %v", err)
	}
	firstStored, err := store.GetEmailTargetByEmail(ctx, item.Email)
	if err != nil {
		t.Fatalf("load first stored email target: %v", err)
	}

	agedSentAt := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := store.UpdateEmailTarget(ctx, storage.UpdateEmailTargetInput{
		ID:                     firstStored.ID,
		CloudflareAddressID:    "",
		VerificationTokenHash:  firstStored.VerificationTokenHash,
		VerificationExpiresAt:  firstStored.VerificationExpiresAt,
		VerifiedAt:             nil,
		LastVerificationSentAt: &agedSentAt,
	}); err != nil {
		t.Fatalf("age target resend timestamp: %v", err)
	}

	resent, err := service.ResendMyEmailTargetVerification(ctx, user, item.ID)
	if err != nil {
		t.Fatalf("resend email target verification: %v", err)
	}
	if resent.Verified {
		t.Fatalf("expected resent target to stay pending")
	}
	if len(mailer.sent) != 2 {
		t.Fatalf("expected two verification emails total after resend, got %d", len(mailer.sent))
	}

	refreshed, err := store.GetEmailTargetByEmail(ctx, item.Email)
	if err != nil {
		t.Fatalf("reload resent email target: %v", err)
	}
	if refreshed.VerificationTokenHash == firstStored.VerificationTokenHash {
		t.Fatalf("expected resend to mint a fresh verification token hash")
	}
	if refreshed.LastVerificationSentAt == nil || !refreshed.LastVerificationSentAt.After(agedSentAt) {
		t.Fatalf("expected resend timestamp to advance, got %+v", refreshed.LastVerificationSentAt)
	}
}

// TestVerifyEmailTargetRejectsExpiredToken verifies that stale verification
// links are rejected and cleared so they cannot be retried indefinitely.
func TestVerifyEmailTargetRejectsExpiredToken(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 701, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	token, tokenHash, err := generateEmailTargetVerificationToken()
	if err != nil {
		t.Fatalf("generate verification token: %v", err)
	}
	expiredAt := time.Now().UTC().Add(-time.Minute)
	if _, err := store.CreateEmailTarget(ctx, storage.CreateEmailTargetInput{
		OwnerUserID:            user.ID,
		Email:                  "expired@example.com",
		CloudflareAddressID:    "",
		VerificationTokenHash:  tokenHash,
		VerificationExpiresAt:  &expiredAt,
		VerifiedAt:             nil,
		LastVerificationSentAt: nil,
	}); err != nil {
		t.Fatalf("seed expired email target: %v", err)
	}

	service, _ := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	_, err = service.VerifyEmailTarget(ctx, token)
	if err == nil {
		t.Fatalf("expected expired verification token to fail")
	}
	normalized := NormalizeError(err)
	if normalized.Code != "validation_failed" {
		t.Fatalf("expected validation_failed, got %s: %v", normalized.Code, err)
	}

	stored, err := store.GetEmailTargetByEmail(ctx, "expired@example.com")
	if err != nil {
		t.Fatalf("reload expired email target: %v", err)
	}
	if strings.TrimSpace(stored.VerificationTokenHash) != "" {
		t.Fatalf("expected expired verification token hash to be cleared")
	}
}

// TestVerifyEmailTargetConsumesTokenOnlyOnceUnderConcurrency verifies that
// concurrent opens of the same verification link cannot both succeed.
func TestVerifyEmailTargetConsumesTokenOnlyOnceUnderConcurrency(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 801, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, mailer := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	if _, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "concurrent@example.com"}); err != nil {
		t.Fatalf("create email target: %v", err)
	}
	token := mustExtractVerificationToken(t, mailer.sent[0].VerificationURL)

	type verifyResult struct {
		err error
	}

	results := make(chan verifyResult, 2)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for attempt := 0; attempt < 2; attempt++ {
		waitGroup.Add(1)
		go func() {
			defer waitGroup.Done()
			<-start
			_, verifyErr := service.VerifyEmailTarget(ctx, token)
			results <- verifyResult{err: verifyErr}
		}()
	}

	close(start)
	waitGroup.Wait()
	close(results)

	successCount := 0
	notFoundCount := 0
	for item := range results {
		if item.err == nil {
			successCount++
			continue
		}
		if NormalizeError(item.err).Code == "not_found" {
			notFoundCount++
			continue
		}
		t.Fatalf("expected concurrent loser to see not_found, got %v", item.err)
	}

	if successCount != 1 {
		t.Fatalf("expected exactly one verification success, got %d", successCount)
	}
	if notFoundCount != 1 {
		t.Fatalf("expected exactly one verification miss, got %d", notFoundCount)
	}
}

// TestResendMyEmailTargetVerificationHonorsPersistentTargetBurstLimit verifies
// that the resend endpoint is throttled by persisted send history, not only the
// in-memory cooldown on the current process.
func TestResendMyEmailTargetVerificationHonorsPersistentTargetBurstLimit(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 802, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service, _ := newPermissionServiceWithTestMailer(newPermissionEmailTestConfig(), store, nil)

	item, err := service.CreateMyEmailTarget(ctx, user, CreateMyEmailTargetRequest{Email: "burst@example.com"})
	if err != nil {
		t.Fatalf("create pending email target: %v", err)
	}
	stored, err := store.GetEmailTargetByEmail(ctx, item.Email)
	if err != nil {
		t.Fatalf("load created email target: %v", err)
	}

	for _, offset := range []time.Duration{-9 * time.Minute, -5 * time.Minute, -2 * time.Minute} {
		preparedAt := time.Now().UTC().Add(offset)
		expiresAt := preparedAt.Add(emailTargetVerificationTokenLifetime)
		if _, err := store.PrepareEmailTargetVerificationSend(ctx, storage.PrepareEmailTargetVerificationSendInput{
			ID:                    stored.ID,
			OwnerUserID:           user.ID,
			Email:                 stored.Email,
			VerificationTokenHash: "seed-" + preparedAt.Format("150405"),
			VerificationExpiresAt: &expiresAt,
			PreparedAt:            preparedAt,
			ShortWindowStart:      preparedAt.Add(-10 * time.Minute),
			DailyWindowStart:      preparedAt.Add(-24 * time.Hour),
			OwnerShortLimit:       99,
			OwnerDailyLimit:       99,
			TargetShortLimit:      99,
			TargetDailyLimit:      99,
		}); err != nil {
			t.Fatalf("seed verification attempt at %s: %v", preparedAt.Format(time.RFC3339), err)
		}
	}

	agedSentAt := time.Now().UTC().Add(-2 * time.Minute)
	if _, err := store.UpdateEmailTarget(ctx, storage.UpdateEmailTargetInput{
		ID:                     stored.ID,
		CloudflareAddressID:    "",
		VerificationTokenHash:  stored.VerificationTokenHash,
		VerificationExpiresAt:  stored.VerificationExpiresAt,
		VerifiedAt:             nil,
		LastVerificationSentAt: &agedSentAt,
	}); err != nil {
		t.Fatalf("age resend cooldown timestamp: %v", err)
	}

	_, err = service.ResendMyEmailTargetVerification(ctx, user, stored.ID)
	if err == nil {
		t.Fatalf("expected persistent target burst limit to block resend")
	}
	if NormalizeError(err).Code != "too_many_requests" {
		t.Fatalf("expected too_many_requests from persistent target burst limit, got %v", err)
	}
}

// mustExtractVerificationToken pulls the token query value back out of the test
// mailer's verification URL so the service can be exercised end-to-end.
func mustExtractVerificationToken(t *testing.T, verificationURL string) string {
	t.Helper()

	parsed, err := url.Parse(verificationURL)
	if err != nil {
		t.Fatalf("parse verification url: %v", err)
	}
	token := strings.TrimSpace(parsed.Query().Get("token"))
	if token == "" {
		t.Fatalf("expected verification token in url %q", verificationURL)
	}
	return token
}
