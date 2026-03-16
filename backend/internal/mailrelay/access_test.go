package mailrelay

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/storage/sqlite"
)

// TestDBCatchAllAccessManagerMapsStorageState verifies that the SMTP-facing
// access manager translates the underlying mutable quota state into stable
// mailrelay errors.
func TestDBCatchAllAccessManagerMapsStorageState(t *testing.T) {
	ctx := context.Background()
	store := newAccessTestStore(t)

	user, err := store.UpsertUser(ctx, sqlite.UpsertUserInput{
		LinuxDOUserID:  9991,
		Username:       "alice",
		DisplayName:    "alice",
		AvatarURL:      "https://example.com/avatar.png",
		TrustLevel:     2,
		IsLinuxDOAdmin: false,
		IsAppAdmin:     false,
	})
	if err != nil {
		t.Fatalf("upsert user: %v", err)
	}

	dailyLimitOverride := int64(1)
	if _, err := store.UpsertEmailCatchAllAccess(ctx, storage.UpsertEmailCatchAllAccessInput{
		UserID:             user.ID,
		RemainingCount:     1,
		DailyLimitOverride: &dailyLimitOverride,
	}); err != nil {
		t.Fatalf("upsert email catch-all access: %v", err)
	}

	manager := NewDBCatchAllAccessManager(store)
	// 15:59 UTC equals 23:59 in Asia/Shanghai, so adding two minutes crosses
	// the service's real daily-reset boundary instead of merely adding 24 hours.
	fixedNow := time.Date(2026, 3, 12, 15, 59, 0, 0, time.UTC)
	manager.now = func() time.Time { return fixedNow }

	reservation, err := manager.Reserve(ctx, user.ID, 1)
	if err != nil {
		t.Fatalf("reserve first catch-all delivery: %v", err)
	}
	if reservation.ConsumedMode != "quantity" {
		t.Fatalf("expected quantity reservation mode, got %q", reservation.ConsumedMode)
	}

	if err := manager.Release(ctx, reservation); err != nil {
		t.Fatalf("release first catch-all delivery reservation: %v", err)
	}

	reusedReservation, err := manager.Reserve(ctx, user.ID, 1)
	if err != nil {
		t.Fatalf("reserve second catch-all delivery after release: %v", err)
	}

	if _, err := manager.Reserve(ctx, user.ID, 1); !errors.Is(err, ErrCatchAllDailyLimitExceeded) {
		t.Fatalf("expected daily limit exceeded error, got %v", err)
	}

	manager.now = func() time.Time { return fixedNow.Add(2 * time.Minute) }
	if _, err := manager.Reserve(ctx, user.ID, 1); !errors.Is(err, ErrCatchAllAccessUnavailable) {
		t.Fatalf("expected access unavailable after remaining count exhaustion, got %v", err)
	}

	if err := manager.Release(ctx, reusedReservation); err != nil {
		t.Fatalf("release reused reservation: %v", err)
	}
}

// newAccessTestStore creates one migrated SQLite store for access-manager
// tests.
func newAccessTestStore(t *testing.T) *sqlite.Store {
	t.Helper()

	store, err := sqlite.NewStore(filepath.Join(t.TempDir(), "mailrelay-access-test.sqlite"))
	if err != nil {
		t.Fatalf("new mailrelay access test store: %v", err)
	}
	t.Cleanup(func() {
		if err := store.Close(); err != nil {
			t.Fatalf("close mailrelay access test store: %v", err)
		}
	})

	if err := store.Migrate(context.Background()); err != nil {
		t.Fatalf("migrate mailrelay access test store: %v", err)
	}
	return store
}
