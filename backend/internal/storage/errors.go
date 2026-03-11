package storage

import (
	"database/sql"
	"errors"
)

// IsNotFound reports whether one storage call failed only because the target
// row does not exist.
//
// The helper currently normalizes to `sql.ErrNoRows` so existing SQLite code
// and future PostgreSQL code can share one storage-agnostic check.
func IsNotFound(err error) bool {
	return errors.Is(err, sql.ErrNoRows)
}
