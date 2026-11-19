package store

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*/*.sql
var migrationsFS embed.FS

// Migrate applies all embedded migrations for the store's dialect.
func (s *Store) Migrate(ctx context.Context) error {
	if _, err := s.db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS schema_migrations (
		version INTEGER PRIMARY KEY,
		applied_at BIGINT NOT NULL
	)`); err != nil {
		return fmt.Errorf("store: create schema_migrations: %w", err)
	}
	dir := "migrations/" + string(s.driver)
	entries, err := fs.ReadDir(migrationsFS, dir)
	if err != nil {
		return fmt.Errorf("store: migrations for %s: %w", s.driver, err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		version, err := migrationVersion(entry.Name())
		if err != nil {
			return err
		}
		var one int
		err = s.db.QueryRowContext(ctx, s.q(`SELECT 1 FROM schema_migrations WHERE version = ?`), version).Scan(&one)
		if err == nil {
			continue
		}
		if !IsNotFound(err) {
			return fmt.Errorf("store: check migration %d: %w", version, err)
		}
		data, err := fs.ReadFile(migrationsFS, dir+"/"+entry.Name())
		if err != nil {
			return err
		}
		statements, err := splitSQL(string(data))
		if err != nil {
			return fmt.Errorf("store: %s: %w", entry.Name(), err)
		}
		tx, err := s.db.BeginTx(ctx, nil)
		if err != nil {
			return err
		}
		for i, stmt := range statements {
			if _, err := tx.ExecContext(ctx, s.q(stmt)); err != nil {
				tx.Rollback()
				return fmt.Errorf("store: migration %s statement %d: %w", entry.Name(), i+1, err)
			}
		}
		if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?)`), version, time.Now().Unix()); err != nil {
			tx.Rollback()
			return fmt.Errorf("store: record migration %d: %w", version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("store: commit migration %d: %w", version, err)
		}
	}
	return nil
}

func migrationVersion(name string) (int64, error) {
	idx := strings.IndexByte(name, '_')
	if idx <= 0 {
		return 0, fmt.Errorf("store: migration %q has no numeric prefix", name)
	}
	v, err := strconv.ParseInt(name[:idx], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("store: migration %q: %w", name, err)
	}
	return v, nil
}

// splitSQL splits a migration file on semicolons outside quotes and comments.
// The migrations contain only DDL, so this intentionally does not implement
// dollar-quoted PostgreSQL bodies or MySQL DELIMITER blocks.
func splitSQL(src string) ([]string, error) {
	var out []string
	var b strings.Builder
	inSingle, inDouble, inLine, inBlock := false, false, false, false
	for i := 0; i < len(src); i++ {
		c := src[i]
		if inLine {
			if c == '\n' {
				inLine = false
				b.WriteByte(c)
			}
			continue
		}
		if inBlock {
			if c == '*' && i+1 < len(src) && src[i+1] == '/' {
				inBlock = false
				i++
			}
			continue
		}
		if inSingle {
			b.WriteByte(c)
			if c == '\'' {
				if i+1 < len(src) && src[i+1] == '\'' {
					b.WriteByte(src[i+1])
					i++
				} else {
					inSingle = false
				}
			}
			continue
		}
		if inDouble {
			b.WriteByte(c)
			if c == '"' {
				inDouble = false
			}
			continue
		}
		switch c {
		case '\'':
			inSingle = true
			b.WriteByte(c)
		case '"':
			inDouble = true
			b.WriteByte(c)
		case '-':
			if i+1 < len(src) && src[i+1] == '-' {
				inLine = true
				i++
			} else {
				b.WriteByte(c)
			}
		case '/':
			if i+1 < len(src) && src[i+1] == '*' {
				inBlock = true
				i++
			} else {
				b.WriteByte(c)
			}
		case ';':
			if stmt := strings.TrimSpace(b.String()); stmt != "" {
				out = append(out, stmt)
			}
			b.Reset()
		default:
			b.WriteByte(c)
		}
	}
	if inSingle || inDouble || inBlock {
		return nil, fmt.Errorf("unterminated quote or block comment")
	}
	if stmt := strings.TrimSpace(b.String()); stmt != "" {
		out = append(out, stmt)
	}
	return out, nil
}
