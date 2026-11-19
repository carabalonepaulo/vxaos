package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
)

// GetServerState returns one server_state value. Missing keys yield "".
func (s *Store) GetServerState(ctx context.Context, key string) (string, error) {
	var value string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT value FROM server_state WHERE state_key = ?`), key).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	if err != nil {
		return "", fmt.Errorf("store: get server state: %w", err)
	}
	return value, nil
}

// SetServerState writes one server_state value. The world goroutine is the
// only writer, so the simple update-then-insert sequence is enough and stays
// dialect independent.
func (s *Store) SetServerState(ctx context.Context, key, value string) error {
	res, err := s.db.ExecContext(ctx, s.q(`UPDATE server_state SET value = ? WHERE state_key = ?`), value, key)
	if err != nil {
		return fmt.Errorf("store: update server state: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("store: update server state: %w", err)
	}
	if n > 0 {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, s.q(`INSERT INTO server_state (state_key, value) VALUES (?, ?)`), key, value); err != nil {
		return fmt.Errorf("store: insert server state: %w", err)
	}
	return nil
}
