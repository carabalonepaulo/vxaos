// Package store is the SQL persistence layer. It uses database/sql with
// dialect-aware queries and embedded migrations. SQLite is the default; MySQL
// and PostgreSQL use the same repositories.
package store

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
	_ "github.com/jackc/pgx/v5/stdlib"
	sqliteDriver "modernc.org/sqlite"
)

func init() {
	// These are per-connection in SQLite, so install them on every pooled
	// connection instead of relying on a single PRAGMA after Open.
	sqliteDriver.RegisterConnectionHook(func(conn sqliteDriver.ExecQuerierContext, dsn string) error {
		ctx := context.Background()
		if _, err := conn.ExecContext(ctx, "PRAGMA foreign_keys = ON", nil); err != nil {
			return err
		}
		if _, err := conn.ExecContext(ctx, "PRAGMA busy_timeout = 5000", nil); err != nil {
			return err
		}
		return nil
	})
}

// Driver selects a SQL dialect.
type Driver string

const (
	SQLite   Driver = "sqlite"
	MySQL    Driver = "mysql"
	Postgres Driver = "postgres"
)

// Config configures the connection pool.
type Config struct {
	Driver          Driver
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
}

// Store owns the SQL pool.
type Store struct {
	db     *sql.DB
	driver Driver
}

// Open opens a pool and verifies connectivity.
func Open(ctx context.Context, cfg Config) (*Store, error) {
	if cfg.Driver == "" {
		cfg.Driver = SQLite
	}
	if cfg.DSN == "" {
		return nil, fmt.Errorf("store: empty DSN")
	}
	var driverName string
	switch cfg.Driver {
	case SQLite:
		driverName = "sqlite"
	case MySQL:
		driverName = "mysql"
	case Postgres:
		driverName = "pgx"
	default:
		return nil, fmt.Errorf("store: unsupported driver %q", cfg.Driver)
	}
	db, err := sql.Open(driverName, cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("store: open: %w", err)
	}
	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	} else if cfg.Driver == SQLite {
		// One writer keeps SQLite's locking predictable; the game world is
		// single-threaded and the persistence worker is serialized.
		db.SetMaxOpenConns(1)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	} else if cfg.Driver == SQLite {
		db.SetMaxIdleConns(1)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("store: ping: %w", err)
	}
	s := &Store{db: db, driver: cfg.Driver}
	if cfg.Driver == SQLite {
		// WAL is a database property, not per-connection. It is harmless to
		// repeat on open; memory databases do not support it.
		if _, err := db.ExecContext(ctx, "PRAGMA journal_mode = WAL"); err != nil {
			// Keep going: the connection hook already applied the important
			// per-connection pragmas and some test DSNs cannot use WAL.
		}
	}
	return s, nil
}

// Close closes the pool.
func (s *Store) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}

// Driver returns the configured dialect.
func (s *Store) Driver() Driver { return s.driver }

// DB exposes the pool for tests and migrations. Repositories should use the
// Store methods instead.
func (s *Store) DB() *sql.DB { return s.db }

// q rewrites '?' placeholders for PostgreSQL.
func (s *Store) q(query string) string {
	if s.driver != Postgres {
		return query
	}
	var b strings.Builder
	n := 0
	inSingle := false
	for i := 0; i < len(query); i++ {
		c := query[i]
		if c == '\'' {
			inSingle = !inSingle
			b.WriteByte(c)
			continue
		}
		if c == '?' && !inSingle {
			n++
			b.WriteByte('$')
			b.WriteString(fmt.Sprint(n))
			continue
		}
		b.WriteByte(c)
	}
	return b.String()
}

// IsNotFound reports the standard missing-row error.
func IsNotFound(err error) bool {
	return err == sql.ErrNoRows
}
