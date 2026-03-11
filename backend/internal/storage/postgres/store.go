package postgres

import (
	"bytes"
	"context"
	"database/sql"
	"embed"
	"fmt"
	"sort"
	"strconv"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

// migrationsFS embeds the PostgreSQL-specific schema migrations so the backend
// binary can bootstrap and upgrade the database without extra files on disk.
//
//go:embed migrations/*.sql
var migrationsFS embed.FS

var utf8BOM = []byte{0xef, 0xbb, 0xbf}

// Store is the PostgreSQL persistence entry point. The repository methods reuse
// the existing database/sql flow, while the wrapper transparently rewrites
// SQLite-style `?` placeholders into PostgreSQL `$1...$n` placeholders.
type Store struct {
	db  *queryDB
	raw *sql.DB
}

// queryDB wraps sql.DB so repository code can keep its existing `?`-based SQL
// strings while PostgreSQL still receives valid numbered placeholders.
type queryDB struct {
	inner *sql.DB
}

// queryTx wraps sql.Tx for the same placeholder rebinding behaviour inside
// transactional repository methods.
type queryTx struct {
	inner *sql.Tx
}

// NewStore opens one PostgreSQL-backed store using the provided DSN.
func NewStore(dsn string) (*Store, error) {
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("open postgres database: %w", err)
	}

	// PostgreSQL is the production-oriented backend, so keep a real connection
	// pool instead of the SQLite single-connection limit.
	db.SetMaxOpenConns(20)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)
	db.SetConnMaxIdleTime(15 * time.Minute)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}

	return &Store{
		db:  &queryDB{inner: db},
		raw: db,
	}, nil
}

// Close closes the underlying PostgreSQL connection pool.
func (s *Store) Close() error {
	if s == nil || s.raw == nil {
		return nil
	}
	return s.raw.Close()
}

// DB exposes the raw database/sql handle for narrow advanced use-cases.
func (s *Store) DB() *sql.DB {
	if s == nil {
		return nil
	}
	return s.raw
}

// Migrate applies the embedded PostgreSQL schema migrations in filename order.
func (s *Store) Migrate(ctx context.Context) error {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return fmt.Errorf("read migration directory: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		script, err := migrationsFS.ReadFile("migrations/" + entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		script = bytes.TrimPrefix(script, utf8BOM)
		if _, err := s.raw.ExecContext(ctx, string(script)); err != nil {
			return fmt.Errorf("execute migration %s: %w", entry.Name(), err)
		}
	}

	return nil
}

// QueryRowContext executes one rebinding query expected to return at most one row.
func (db *queryDB) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return db.inner.QueryRowContext(ctx, rebindQuery(query), args...)
}

// QueryContext executes one rebinding query expected to stream multiple rows.
func (db *queryDB) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return db.inner.QueryContext(ctx, rebindQuery(query), args...)
}

// ExecContext executes one rebinding statement without returning rows.
func (db *queryDB) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return db.inner.ExecContext(ctx, rebindQuery(query), args...)
}

// BeginTx opens one wrapped PostgreSQL transaction.
func (db *queryDB) BeginTx(ctx context.Context, opts *sql.TxOptions) (*queryTx, error) {
	tx, err := db.inner.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &queryTx{inner: tx}, nil
}

// QueryRowContext executes one rebinding query inside the current transaction.
func (tx *queryTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.inner.QueryRowContext(ctx, rebindQuery(query), args...)
}

// QueryContext executes one rebinding row query inside the current transaction.
func (tx *queryTx) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return tx.inner.QueryContext(ctx, rebindQuery(query), args...)
}

// ExecContext executes one rebinding non-row statement inside the current transaction.
func (tx *queryTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.inner.ExecContext(ctx, rebindQuery(query), args...)
}

// Commit commits the wrapped transaction.
func (tx *queryTx) Commit() error {
	return tx.inner.Commit()
}

// Rollback rolls back the wrapped transaction.
func (tx *queryTx) Rollback() error {
	return tx.inner.Rollback()
}

// rebindQuery converts `?` placeholders into PostgreSQL numbered placeholders.
func rebindQuery(query string) string {
	index := 1
	buffer := bytes.NewBuffer(make([]byte, 0, len(query)+8))
	inSingleQuotedString := false
	inDoubleQuotedIdentifier := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(query); i++ {
		current := query[i]
		next := byte(0)
		if i+1 < len(query) {
			next = query[i+1]
		}

		switch {
		case inLineComment:
			buffer.WriteByte(current)
			if current == '\n' {
				inLineComment = false
			}
			continue
		case inBlockComment:
			buffer.WriteByte(current)
			if current == '*' && next == '/' {
				buffer.WriteByte(next)
				i++
				inBlockComment = false
			}
			continue
		case inSingleQuotedString:
			buffer.WriteByte(current)
			if current == '\'' {
				if next == '\'' {
					buffer.WriteByte(next)
					i++
				} else {
					inSingleQuotedString = false
				}
			}
			continue
		case inDoubleQuotedIdentifier:
			buffer.WriteByte(current)
			if current == '"' {
				if next == '"' {
					buffer.WriteByte(next)
					i++
				} else {
					inDoubleQuotedIdentifier = false
				}
			}
			continue
		}

		if current == '-' && next == '-' {
			buffer.WriteByte(current)
			buffer.WriteByte(next)
			i++
			inLineComment = true
			continue
		}
		if current == '/' && next == '*' {
			buffer.WriteByte(current)
			buffer.WriteByte(next)
			i++
			inBlockComment = true
			continue
		}
		if current == '\'' {
			buffer.WriteByte(current)
			inSingleQuotedString = true
			continue
		}
		if current == '"' {
			buffer.WriteByte(current)
			inDoubleQuotedIdentifier = true
			continue
		}
		if current != '?' {
			buffer.WriteByte(current)
			continue
		}

		buffer.WriteByte('$')
		buffer.WriteString(strconv.Itoa(index))
		index++
	}

	return buffer.String()
}
