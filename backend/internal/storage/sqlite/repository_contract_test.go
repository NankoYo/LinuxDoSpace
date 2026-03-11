package sqlite

import (
	"context"
	"path/filepath"
	"testing"

	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/storage/storagetest"
)

// TestRepositoryBehaviorSuite reuses the storage-agnostic repository contract
// so SQLite and PostgreSQL are validated against the same behavioral baseline.
func TestRepositoryBehaviorSuite(t *testing.T) {
	storagetest.RunRepositoryBehaviorSuite(t, func(t *testing.T) storage.Backend {
		t.Helper()

		store, err := NewStore(filepath.Join(t.TempDir(), "linuxdospace-contract-test.sqlite"))
		if err != nil {
			t.Fatalf("new sqlite contract test store: %v", err)
		}
		t.Cleanup(func() {
			if closeErr := store.Close(); closeErr != nil {
				t.Fatalf("close sqlite contract test store: %v", closeErr)
			}
		})

		if err := store.Migrate(context.Background()); err != nil {
			t.Fatalf("migrate sqlite contract test store: %v", err)
		}

		return store
	})
}
