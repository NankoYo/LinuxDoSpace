package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"

	"linuxdospace/backend/internal/storage"
	"linuxdospace/backend/internal/storage/storagetest"
)

// TestRepositoryBehaviorSuite runs the shared repository contract against a
// real PostgreSQL schema when the caller provides one integration DSN.
//
// The test stays opt-in so ordinary local `go test ./...` runs do not require a
// PostgreSQL service, but CI or an operator can enable it with
// `LINUXDOSPACE_TEST_POSTGRES_DSN`.
func TestRepositoryBehaviorSuite(t *testing.T) {
	storagetest.RunRepositoryBehaviorSuite(t, func(t *testing.T) storage.Backend {
		t.Helper()
		return newIntegrationTestStore(t)
	})
}

// newIntegrationTestStore opens one isolated PostgreSQL schema for the current
// test case, migrates it, and drops it after the test completes.
func newIntegrationTestStore(t *testing.T) *Store {
	t.Helper()

	baseDSN := strings.TrimSpace(os.Getenv("LINUXDOSPACE_TEST_POSTGRES_DSN"))
	if baseDSN == "" {
		t.Skip("set LINUXDOSPACE_TEST_POSTGRES_DSN to run PostgreSQL integration tests")
	}

	adminDB, err := sql.Open("pgx", baseDSN)
	if err != nil {
		t.Fatalf("open postgres integration database: %v", err)
	}
	t.Cleanup(func() {
		if closeErr := adminDB.Close(); closeErr != nil {
			t.Errorf("close postgres integration database: %v", closeErr)
		}
	})

	ctx := context.Background()
	schemaName := fmt.Sprintf("linuxdospace_test_%d", time.Now().UTC().UnixNano())
	if _, err := adminDB.ExecContext(ctx, `CREATE SCHEMA `+schemaName); err != nil {
		t.Fatalf("create postgres integration schema %q: %v", schemaName, err)
	}

	cfg, err := pgx.ParseConfig(baseDSN)
	if err != nil {
		t.Fatalf("parse postgres integration dsn: %v", err)
	}
	if cfg.RuntimeParams == nil {
		cfg.RuntimeParams = make(map[string]string)
	}
	cfg.RuntimeParams["search_path"] = schemaName

	store, err := NewStore(cfg.ConnString())
	if err != nil {
		t.Fatalf("open postgres store for schema %q: %v", schemaName, err)
	}
	t.Cleanup(func() {
		if closeErr := store.Close(); closeErr != nil {
			t.Errorf("close postgres store for schema %q: %v", schemaName, closeErr)
		}
		if _, dropErr := adminDB.ExecContext(context.Background(), `DROP SCHEMA IF EXISTS `+schemaName+` CASCADE`); dropErr != nil {
			t.Errorf("drop postgres integration schema %q: %v", schemaName, dropErr)
		}
	})

	if err := store.Migrate(ctx); err != nil {
		t.Fatalf("migrate postgres integration schema %q: %v", schemaName, err)
	}

	return store
}
