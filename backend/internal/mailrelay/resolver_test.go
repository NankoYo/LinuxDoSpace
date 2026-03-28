package mailrelay

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"linuxdospace/backend/internal/model"
)

// fakeResolverStore keeps the mail relay unit tests focused on route
// resolution instead of a concrete SQL backend.
type fakeResolverStore struct {
	routes       map[string]model.EmailRoute
	targets      map[string]model.EmailTarget
	tokens       map[string]model.APIToken
	controls     map[int64]model.UserControl
	routeLookups int
}

// GetEmailRouteByAddress returns one in-memory route keyed by domain + prefix.
func (f *fakeResolverStore) GetEmailRouteByAddress(ctx context.Context, rootDomain string, prefix string) (model.EmailRoute, error) {
	f.routeLookups++
	key := strings.ToLower(strings.TrimSpace(rootDomain)) + "|" + strings.ToLower(strings.TrimSpace(prefix))
	item, ok := f.routes[key]
	if !ok {
		return model.EmailRoute{}, sql.ErrNoRows
	}
	return item, nil
}

// GetEmailTargetByEmail returns one in-memory target binding keyed by email.
func (f *fakeResolverStore) GetEmailTargetByEmail(ctx context.Context, email string) (model.EmailTarget, error) {
	key := strings.ToLower(strings.TrimSpace(email))
	item, ok := f.targets[key]
	if !ok {
		return model.EmailTarget{}, sql.ErrNoRows
	}
	return item, nil
}

// GetAPITokenByPublicID returns one in-memory API token keyed by public id.
func (f *fakeResolverStore) GetAPITokenByPublicID(ctx context.Context, publicID string) (model.APIToken, error) {
	key := strings.TrimSpace(publicID)
	item, ok := f.tokens[key]
	if !ok {
		return model.APIToken{}, sql.ErrNoRows
	}
	return item, nil
}

// GetUserControlByUserID returns one in-memory moderation control row.
func (f *fakeResolverStore) GetUserControlByUserID(ctx context.Context, userID int64) (model.UserControl, error) {
	item, ok := f.controls[userID]
	if !ok {
		return model.UserControl{}, sql.ErrNoRows
	}
	return item, nil
}

// TestDBResolverPrefersExactRoute verifies that an exact mailbox row is used
// before the domain catch-all route, even when both exist.
func TestDBResolverPrefersExactRoute(t *testing.T) {
	verifiedAt := time.Now().UTC()
	resolver := NewDBResolver(&fakeResolverStore{
		routes: map[string]model.EmailRoute{
			"linuxdo.space|alice": {
				ID:          1,
				OwnerUserID: 10,
				RootDomain:  "linuxdo.space",
				Prefix:      "alice",
				TargetEmail: "exact@example.com",
				Enabled:     true,
			},
			"linuxdo.space|catch-all": {
				ID:          2,
				OwnerUserID: 10,
				RootDomain:  "linuxdo.space",
				Prefix:      catchAllRoutePrefix,
				TargetEmail: "catchall@example.com",
				Enabled:     true,
			},
		},
		targets: map[string]model.EmailTarget{
			"exact@example.com": {
				ID:          11,
				OwnerUserID: 10,
				Email:       "exact@example.com",
				VerifiedAt:  &verifiedAt,
			},
			"catchall@example.com": {
				ID:          12,
				OwnerUserID: 10,
				Email:       "catchall@example.com",
				VerifiedAt:  &verifiedAt,
			},
		},
	}, "linuxdo.space", nil)

	result, err := resolver.ResolveRecipient(context.Background(), "Alice@LinuxDo.Space")
	if err != nil {
		t.Fatalf("resolve exact recipient: %v", err)
	}
	if result.TargetEmail != "exact@example.com" {
		t.Fatalf("expected exact target email, got %q", result.TargetEmail)
	}
	if result.UsedCatchAll {
		t.Fatalf("expected exact route, but resolver reported catch-all")
	}
}

// TestDBResolverFallsBackToCatchAll verifies that the relay still delivers mail
// for unmatched local-parts when a catch-all route exists for that domain.
func TestDBResolverFallsBackToCatchAll(t *testing.T) {
	verifiedAt := time.Now().UTC()
	resolver := NewDBResolver(&fakeResolverStore{
		routes: map[string]model.EmailRoute{
			"alice.linuxdo.space|catch-all": {
				ID:          2,
				OwnerUserID: 10,
				RootDomain:  "alice.linuxdo.space",
				Prefix:      catchAllRoutePrefix,
				TargetEmail: "catchall@example.com",
				Enabled:     true,
			},
		},
		targets: map[string]model.EmailTarget{
			"catchall@example.com": {
				ID:          12,
				OwnerUserID: 10,
				Email:       "catchall@example.com",
				VerifiedAt:  &verifiedAt,
			},
		},
	}, "linuxdo.space", nil)

	result, err := resolver.ResolveRecipient(context.Background(), "notice@alice.linuxdo.space")
	if err != nil {
		t.Fatalf("resolve catch-all recipient: %v", err)
	}
	if result.TargetEmail != "catchall@example.com" {
		t.Fatalf("expected catch-all target email, got %q", result.TargetEmail)
	}
	if !result.UsedCatchAll {
		t.Fatalf("expected catch-all route to be used")
	}
}

// TestDBResolverRejectsMismatchedTargetOwnership verifies that the relay fails
// closed if a route points at a target inbox already bound to another user.
func TestDBResolverRejectsMismatchedTargetOwnership(t *testing.T) {
	verifiedAt := time.Now().UTC()
	resolver := NewDBResolver(&fakeResolverStore{
		routes: map[string]model.EmailRoute{
			"linuxdo.space|alice": {
				ID:          1,
				OwnerUserID: 10,
				RootDomain:  "linuxdo.space",
				Prefix:      "alice",
				TargetEmail: "shared@example.com",
				Enabled:     true,
			},
		},
		targets: map[string]model.EmailTarget{
			"shared@example.com": {
				ID:          11,
				OwnerUserID: 99,
				Email:       "shared@example.com",
				VerifiedAt:  &verifiedAt,
			},
		},
	}, "linuxdo.space", nil)

	_, err := resolver.ResolveRecipient(context.Background(), "alice@linuxdo.space")
	if !errors.Is(err, ErrTargetOwnershipMismatch) {
		t.Fatalf("expected target ownership mismatch, got %v", err)
	}
}

// TestBuildForwardMessageAddsTraceHeaders verifies that the outbound forwarder
// writes the relay marker and original envelope headers above the raw message.
func TestBuildForwardMessageAddsTraceHeaders(t *testing.T) {
	raw := []byte("From: sender@example.com\r\nSubject: Test\r\n\r\nhello")

	message, err := buildForwardMessage(raw, "bounce@example.com", []string{"alice@linuxdo.space"}, "signed-loop-token")
	if err != nil {
		t.Fatalf("build forward message: %v", err)
	}

	serialized := string(message)
	if !strings.Contains(serialized, "X-LinuxDoSpace-Relay: 1\r\n") {
		t.Fatalf("expected relay marker header, got %q", serialized)
	}
	if !strings.Contains(serialized, "X-LinuxDoSpace-Relay-Signature: signed-loop-token\r\n") {
		t.Fatalf("expected relay signature header, got %q", serialized)
	}
	if !strings.Contains(serialized, "X-LinuxDoSpace-Original-Envelope-From: bounce@example.com\r\n") {
		t.Fatalf("expected original envelope from header, got %q", serialized)
	}
	if !strings.Contains(serialized, "X-LinuxDoSpace-Original-Envelope-To: alice@linuxdo.space\r\n") {
		t.Fatalf("expected original envelope to header, got %q", serialized)
	}
	if !strings.Contains(serialized, "\r\nFrom: sender@example.com\r\n") {
		t.Fatalf("expected original message headers to remain after relay headers, got %q", serialized)
	}
}

// TestBuildForwardMessageRejectsRelayLoop verifies that the relay does not
// forward a message that already passed through LinuxDoSpace once.
func TestBuildForwardMessageRejectsRelayLoop(t *testing.T) {
	raw := []byte("X-LinuxDoSpace-Relay: 1\r\nX-LinuxDoSpace-Relay-Signature: signed-loop-token\r\nFrom: sender@example.com\r\n\r\nhello")

	_, err := buildForwardMessage(raw, "", []string{"alice@linuxdo.space"}, "signed-loop-token")
	if !errors.Is(err, ErrRelayLoopDetected) {
		t.Fatalf("expected relay loop detection, got %v", err)
	}
}

// TestBuildForwardMessageIgnoresSpoofedUnsignedRelayHeader verifies that an
// attacker cannot blackhole inbound mail merely by adding the visible relay
// marker without the secret signature value.
func TestBuildForwardMessageIgnoresSpoofedUnsignedRelayHeader(t *testing.T) {
	raw := []byte("X-LinuxDoSpace-Relay: 1\r\nFrom: sender@example.com\r\n\r\nhello")

	message, err := buildForwardMessage(raw, "", []string{"alice@linuxdo.space"}, "signed-loop-token")
	if err != nil {
		t.Fatalf("expected spoofed unsigned relay header to be ignored, got %v", err)
	}
	if !strings.Contains(string(message), "X-LinuxDoSpace-Relay-Signature: signed-loop-token\r\n") {
		t.Fatalf("expected signed relay header to be added, got %q", string(message))
	}
}

// TestDBResolverRejectsBannedOwner verifies that banned users cannot keep
// receiving mail through already-configured routes.
func TestDBResolverRejectsBannedOwner(t *testing.T) {
	verifiedAt := time.Now().UTC()
	resolver := NewDBResolver(&fakeResolverStore{
		routes: map[string]model.EmailRoute{
			"linuxdo.space|alice": {
				ID:          1,
				OwnerUserID: 10,
				RootDomain:  "linuxdo.space",
				Prefix:      "alice",
				TargetEmail: "exact@example.com",
				Enabled:     true,
			},
		},
		targets: map[string]model.EmailTarget{
			"exact@example.com": {
				ID:          11,
				OwnerUserID: 10,
				Email:       "exact@example.com",
				VerifiedAt:  &verifiedAt,
			},
		},
		controls: map[int64]model.UserControl{
			10: {
				UserID:   10,
				IsBanned: true,
			},
		},
	}, "linuxdo.space", nil)

	_, err := resolver.ResolveRecipient(context.Background(), "alice@linuxdo.space")
	if !errors.Is(err, ErrRouteDisabled) {
		t.Fatalf("expected banned owner to disable routing, got %v", err)
	}
}

// TestDBResolverResolvesActiveDynamicMailboxDomain verifies that one live
// token-mailbox alias under `<username>-mail<suffix>.<root>` bypasses the
// database route table and resolves directly to the connected token stream.
func TestDBResolverResolvesActiveDynamicMailboxDomain(t *testing.T) {
	hub := NewTokenStreamHub()
	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}
	defer subscription.Cancel()
	if err := hub.UpdateMailboxDomains("ldt_token", 42, "alice", []string{"alice-mailfoo.linuxdo.space"}); err != nil {
		t.Fatalf("register dynamic mailbox domain: %v", err)
	}

	store := &fakeResolverStore{}
	resolver := NewDBResolver(store, "linuxdo.space", hub)

	result, err := resolver.ResolveRecipient(context.Background(), "notice@alice-mailfoo.linuxdo.space")
	if err != nil {
		t.Fatalf("resolve dynamic mailbox recipient: %v", err)
	}
	if result.TargetKind != model.EmailRouteTargetKindAPIToken || result.TargetTokenPublicID != "ldt_token" {
		t.Fatalf("expected api token dynamic mailbox route, got %+v", result)
	}
	if !result.UsedCatchAll || result.RoutePrefix != catchAllRoutePrefix {
		t.Fatalf("expected dynamic mailbox route to reuse catch-all accounting semantics, got %+v", result)
	}
	if store.routeLookups != 1 {
		t.Fatalf("expected one exact-route lookup before dynamic mailbox resolution, got %d", store.routeLookups)
	}
}

// TestDBResolverFastRejectsInactiveDynamicMailboxDomain verifies that random
// `-mail<suffix>` aliases do not hit the database when no live token stream
// registered that exact dynamic domain.
func TestDBResolverFastRejectsInactiveDynamicMailboxDomain(t *testing.T) {
	store := &fakeResolverStore{}
	resolver := NewDBResolver(store, "linuxdo.space", NewTokenStreamHub())

	_, err := resolver.ResolveRecipient(context.Background(), "notice@alice-mailzzz.linuxdo.space")
	if !errors.Is(err, ErrNoMatchingRoute) {
		t.Fatalf("expected ErrNoMatchingRoute for inactive dynamic mailbox alias, got %v", err)
	}
	if store.routeLookups != 2 {
		t.Fatalf("expected inactive dynamic alias to fall back to exact+catch-all route lookups, got %d", store.routeLookups)
	}
}

// TestDBResolverRejectsBannedOwnerForDynamicMailboxDomain verifies that the
// dynamic token-mailbox fast path still fails closed once the owner is banned.
func TestDBResolverRejectsBannedOwnerForDynamicMailboxDomain(t *testing.T) {
	hub := NewTokenStreamHub()
	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}
	defer subscription.Cancel()
	if err := hub.UpdateMailboxDomains("ldt_token", 42, "alice", []string{"alice-mailfoo.linuxdo.space"}); err != nil {
		t.Fatalf("register dynamic mailbox domain: %v", err)
	}

	resolver := NewDBResolver(&fakeResolverStore{
		controls: map[int64]model.UserControl{
			42: {UserID: 42, IsBanned: true},
		},
	}, "linuxdo.space", hub)

	_, err = resolver.ResolveRecipient(context.Background(), "notice@alice-mailfoo.linuxdo.space")
	if !errors.Is(err, ErrRouteDisabled) {
		t.Fatalf("expected banned dynamic-mail owner to be rejected, got %v", err)
	}
}
