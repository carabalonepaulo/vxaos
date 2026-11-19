package store

import (
	"context"
	"database/sql"
	"encoding/base64"
	"fmt"
	"time"
)

// Guild is one row of the guilds table.
type Guild struct {
	ID                int64
	Name              string
	LeaderCharacterID int64
	Notice            string
	Flag              []byte
	CreatedAt         int64
}

// GuildMember is one character linked to a guild.
type GuildMember struct {
	CharacterID int64
	Name        string
}

// CreateGuild inserts a guild and assigns its generated id.
func (s *Store) CreateGuild(ctx context.Context, g *Guild) error {
	if g == nil || g.Name == "" {
		return fmt.Errorf("store: invalid guild")
	}
	if g.CreatedAt == 0 {
		g.CreatedAt = time.Now().Unix()
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	id, err := s.insertID(ctx, tx, `INSERT INTO guilds
		(name, name_key, leader_character_id, notice, flag, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		g.Name, NameKey(g.Name), g.LeaderCharacterID, g.Notice, encodeGuildFlag(g.Flag), g.CreatedAt)
	if err != nil {
		return fmt.Errorf("store: create guild: %w", err)
	}
	g.ID = id
	return tx.Commit()
}

// ListGuilds loads every guild.
func (s *Store) ListGuilds(ctx context.Context) ([]Guild, error) {
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT id, name, leader_character_id, notice, flag, created_at FROM guilds`))
	if err != nil {
		return nil, fmt.Errorf("store: list guilds: %w", err)
	}
	defer rows.Close()
	var out []Guild
	for rows.Next() {
		var g Guild
		var flag string
		if err := rows.Scan(&g.ID, &g.Name, &g.LeaderCharacterID, &g.Notice, &flag, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("store: scan guild: %w", err)
		}
		g.Flag = decodeGuildFlag(flag)
		out = append(out, g)
	}
	return out, rows.Err()
}

// GuildMembers loads the characters that belong to a guild.
func (s *Store) GuildMembers(ctx context.Context, guildID int64) ([]GuildMember, error) {
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT id, name FROM characters WHERE guild_id = ? ORDER BY id`), guildID)
	if err != nil {
		return nil, fmt.Errorf("store: guild members: %w", err)
	}
	defer rows.Close()
	var out []GuildMember
	for rows.Next() {
		var m GuildMember
		if err := rows.Scan(&m.CharacterID, &m.Name); err != nil {
			return nil, fmt.Errorf("store: scan guild member: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SaveGuild updates the mutable guild fields.
func (s *Store) SaveGuild(ctx context.Context, g *Guild) error {
	if g == nil || g.ID == 0 {
		return fmt.Errorf("store: invalid guild")
	}
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE guilds SET leader_character_id = ?, notice = ?, flag = ? WHERE id = ?`),
		g.LeaderCharacterID, g.Notice, encodeGuildFlag(g.Flag), g.ID)
	if err != nil {
		return fmt.Errorf("store: save guild: %w", err)
	}
	return nil
}

// DeleteGuild unlinks every member and removes the guild row.
func (s *Store) DeleteGuild(ctx context.Context, id int64) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, s.q(`UPDATE characters SET guild_id = 0 WHERE guild_id = ?`), id); err != nil {
		return fmt.Errorf("store: clear guild members: %w", err)
	}
	if _, err := tx.ExecContext(ctx, s.q(`DELETE FROM guilds WHERE id = ?`), id); err != nil {
		return fmt.Errorf("store: delete guild: %w", err)
	}
	return tx.Commit()
}

func encodeGuildFlag(flag []byte) string {
	if len(flag) == 0 {
		return ""
	}
	return base64.StdEncoding.EncodeToString(flag)
}

func decodeGuildFlag(value string) []byte {
	if value == "" {
		return nil
	}
	flag, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil
	}
	return flag
}

// GetGuildByName loads one guild by case-insensitive name.
func (s *Store) GetGuildByName(ctx context.Context, name string) (*Guild, error) {
	var g Guild
	var flag string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT id, name, leader_character_id, notice, flag, created_at FROM guilds WHERE name_key = ?`), NameKey(name)).
		Scan(&g.ID, &g.Name, &g.LeaderCharacterID, &g.Notice, &flag, &g.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, sql.ErrNoRows
	}
	if err != nil {
		return nil, fmt.Errorf("store: get guild: %w", err)
	}
	g.Flag = decodeGuildFlag(flag)
	return &g, nil
}
