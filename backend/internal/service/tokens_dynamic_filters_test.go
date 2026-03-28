package service

import (
	"context"
	"testing"

	"linuxdospace/backend/internal/config"
	"linuxdospace/backend/internal/mailrelay"
	"linuxdospace/backend/internal/storage"
)

// TestUpdateEmailStreamMailboxFiltersRequiresApprovedCatchAll verifies that a
// live token stream cannot enable dynamic `-mail<suffix>` domains before the
// owner's catch-all permission has been approved.
func TestUpdateEmailStreamMailboxFiltersRequiresApprovedCatchAll(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 4001, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	hub := mailrelay.NewTokenStreamHub()
	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}
	defer subscription.Cancel()

	tokenService := NewTokenService(config.Config{
		Cloudflare: config.CloudflareConfig{
			DefaultRootDomain: "linuxdo.space",
		},
	}, store, hub)

	token, err := store.CreateAPIToken(ctx, storage.CreateAPITokenInput{
		OwnerUserID: user.ID,
		Name:        "mail-stream",
		PublicID:    "ldt_token",
		TokenHash:   "hash",
		Scopes:      []string{"email"},
	})
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	_, err = tokenService.UpdateEmailStreamMailboxFilters(ctx, token, user.Username, UpdateTokenEmailFiltersRequest{
		Suffixes: []string{"foo"},
	})
	if err == nil {
		t.Fatalf("expected dynamic mailbox filters to require approved catch-all permission")
	}
	if normalized := NormalizeError(err); normalized.StatusCode != 403 {
		t.Fatalf("expected forbidden error, got %+v", normalized)
	}
}

// TestUpdateEmailStreamMailboxFiltersRequiresAvailableCatchAllAccess verifies
// that an approved permission without usable delivery allowance still cannot
// activate dynamic `-mail<suffix>` domains.
func TestUpdateEmailStreamMailboxFiltersRequiresAvailableCatchAllAccess(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)
	admin := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 4002, "admin")
	user := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 4003, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	permissionService := NewPermissionService(config.Config{
		Cloudflare: config.CloudflareConfig{
			DefaultRootDomain: "linuxdo.space",
		},
	}, store, nil)
	if _, err := permissionService.SetPermissionForUser(ctx, admin, user.ID, PermissionKeyEmailCatchAll, AdminSetUserPermissionRequest{
		Status: "approved",
		Reason: "approve for token-filter guard test",
	}); err != nil {
		t.Fatalf("approve catch-all permission: %v", err)
	}

	hub := mailrelay.NewTokenStreamHub()
	subscription, err := hub.Subscribe("ldt_token")
	if err != nil {
		t.Fatalf("subscribe token stream: %v", err)
	}
	defer subscription.Cancel()

	tokenService := NewTokenService(config.Config{
		Cloudflare: config.CloudflareConfig{
			DefaultRootDomain: "linuxdo.space",
		},
	}, store, hub)

	token, err := store.CreateAPIToken(ctx, storage.CreateAPITokenInput{
		OwnerUserID: user.ID,
		Name:        "mail-stream",
		PublicID:    "ldt_token",
		TokenHash:   "hash",
		Scopes:      []string{"email"},
	})
	if err != nil {
		t.Fatalf("create api token: %v", err)
	}

	_, err = tokenService.UpdateEmailStreamMailboxFilters(ctx, token, user.Username, UpdateTokenEmailFiltersRequest{
		Suffixes: []string{"foo"},
	})
	if err == nil {
		t.Fatalf("expected dynamic mailbox filters to require available catch-all access")
	}
	if normalized := NormalizeError(err); normalized.StatusCode != 409 {
		t.Fatalf("expected conflict error, got %+v", normalized)
	}
}
