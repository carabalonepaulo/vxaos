package store

import (
	"context"
	"database/sql"
	"fmt"
)

// Ban is one row of the bans table. Kind is "account" or "ip"; Value is the
// account id or IP address.
type Ban struct {
	Kind      string
	Value     string
	Reason    string
	ExpiresAt int64
	CreatedAt int64
	CreatedBy int64
}

// GetActiveBan returns the active ban for a kind/value pair. Missing or expired
// rows yield (nil, nil).
func (s *Store) GetActiveBan(ctx context.Context, kind, value string, now int64) (*Ban, error) {
	row := s.db.QueryRowContext(ctx, s.q(`SELECT kind, value, reason, expires_at, created_at, created_by
		FROM bans WHERE kind = ? AND value = ?`), kind, value)
	var b Ban
	if err := row.Scan(&b.Kind, &b.Value, &b.Reason, &b.ExpiresAt, &b.CreatedAt, &b.CreatedBy); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("store: load ban: %w", err)
	}
	if b.ExpiresAt > 0 && b.ExpiresAt <= now {
		return nil, nil
	}
	return &b, nil
}

// SaveBan inserts or updates a ban row.
func (s *Store) SaveBan(ctx context.Context, b Ban) error {
	if b.Kind == "" || b.Value == "" {
		return fmt.Errorf("store: invalid ban")
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("store: begin ban: %w", err)
	}
	defer tx.Rollback()
	var one int
	err = tx.QueryRowContext(ctx, s.q(`SELECT 1 FROM bans WHERE kind = ? AND value = ?`), b.Kind, b.Value).Scan(&one)
	switch {
	case err == sql.ErrNoRows:
		if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO bans (kind, value, reason, expires_at, created_at, created_by)
			VALUES (?, ?, ?, ?, ?, ?)`), b.Kind, b.Value, b.Reason, b.ExpiresAt, b.CreatedAt, b.CreatedBy); err != nil {
			return fmt.Errorf("store: insert ban: %w", err)
		}
	case err != nil:
		return fmt.Errorf("store: find ban: %w", err)
	default:
		if _, err := tx.ExecContext(ctx, s.q(`UPDATE bans SET reason = ?, expires_at = ?, created_at = ?, created_by = ?
			WHERE kind = ? AND value = ?`), b.Reason, b.ExpiresAt, b.CreatedAt, b.CreatedBy, b.Kind, b.Value); err != nil {
			return fmt.Errorf("store: update ban: %w", err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("store: commit ban: %w", err)
	}
	return nil
}

// DeleteBan removes a ban row.
func (s *Store) DeleteBan(ctx context.Context, kind, value string) error {
	if _, err := s.db.ExecContext(ctx, s.q(`DELETE FROM bans WHERE kind = ? AND value = ?`), kind, value); err != nil {
		return fmt.Errorf("store: delete ban: %w", err)
	}
	return nil
}
