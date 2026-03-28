package service

import (
	"context"
	"testing"

	"linuxdospace/backend/internal/config"
)

// TestAdminCreateAllocationRejectsDynamicMailAliasPrefix verifies that the
// administrator allocation path now shares the same dynamic `-mail<suffix>`
// reservation boundary as the public availability and purchase flows.
func TestAdminCreateAllocationRejectsDynamicMailAliasPrefix(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	actor := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3101, "admin")
	owner := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3102, "other-owner")
	seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3103, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service := NewAdminService(config.Config{
		Cloudflare: config.CloudflareConfig{
			DefaultRootDomain: "linuxdo.space",
		},
	}, store, nil)

	_, err := service.CreateAllocation(ctx, actor, CreateAdminAllocationRequest{
		OwnerUserID: owner.ID,
		RootDomain:  "linuxdo.space",
		Prefix:      "alice-mailfoo",
		IsPrimary:   false,
		Source:      "manual",
		Status:      "active",
	})
	if err == nil {
		t.Fatalf("expected dynamic -mail alias prefix to be rejected for admin allocation creation")
	}
	normalized := NormalizeError(err)
	if normalized.StatusCode != 403 {
		t.Fatalf("expected forbidden error, got %+v", normalized)
	}
}

// TestAdminCreateAllocationStillAllowsUnrelatedPrefix verifies that the shared
// reservation rule remains narrow and does not reject ordinary manual grants.
func TestAdminCreateAllocationStillAllowsUnrelatedPrefix(t *testing.T) {
	ctx := context.Background()
	store := newAuthTestStore(t)

	actor := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3201, "admin")
	owner := seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3202, "other-owner")
	seedPermissionEmailTestUserWithLinuxDOID(t, ctx, store, 3203, "alice")
	seedPermissionEmailManagedDomain(t, ctx, store)

	service := NewAdminService(config.Config{
		Cloudflare: config.CloudflareConfig{
			DefaultRootDomain: "linuxdo.space",
		},
	}, store, nil)

	item, err := service.CreateAllocation(ctx, actor, CreateAdminAllocationRequest{
		OwnerUserID: owner.ID,
		RootDomain:  "linuxdo.space",
		Prefix:      "alice-project",
		IsPrimary:   false,
		Source:      "manual",
		Status:      "active",
	})
	if err != nil {
		t.Fatalf("expected unrelated prefix to remain allocatable, got %v", err)
	}
	if item.FQDN != "alice-project.linuxdo.space" {
		t.Fatalf("expected fqdn alice-project.linuxdo.space, got %q", item.FQDN)
	}
	if item.UserID != owner.ID {
		t.Fatalf("expected owner user id %d, got %d", owner.ID, item.UserID)
	}
}
