package store

import (
	"context"
	"fmt"
)

// ListFriends returns a character's friend names.
func (s *Store) ListFriends(ctx context.Context, characterID int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT c.name FROM character_friends f
		JOIN characters c ON c.id = f.friend_character_id
		WHERE f.character_id = ? ORDER BY c.name`), characterID)
	if err != nil {
		return nil, fmt.Errorf("store: list friends: %w", err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("store: scan friend: %w", err)
		}
		out = append(out, name)
	}
	return out, rows.Err()
}

// AddFriend links two characters once (a -> b).
func (s *Store) AddFriend(ctx context.Context, characterID, friendID int64) error {
	var one int
	err := s.db.QueryRowContext(ctx, s.q(`SELECT 1 FROM character_friends WHERE character_id = ? AND friend_character_id = ?`), characterID, friendID).Scan(&one)
	if err == nil {
		return nil
	}
	if _, err := s.db.ExecContext(ctx, s.q(`INSERT INTO character_friends (character_id, friend_character_id) VALUES (?, ?)`), characterID, friendID); err != nil {
		return fmt.Errorf("store: add friend: %w", err)
	}
	return nil
}

// RemoveFriend unlinks a -> b.
func (s *Store) RemoveFriend(ctx context.Context, characterID, friendID int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM character_friends WHERE character_id = ? AND friend_character_id = ?`), characterID, friendID)
	if err != nil {
		return fmt.Errorf("store: remove friend: %w", err)
	}
	return nil
}
