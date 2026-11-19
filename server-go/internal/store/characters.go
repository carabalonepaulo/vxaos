package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

const characterColumns = `id, account_id, slot, name, name_key, class_id, sex, level, exp, hp, mp, points,
	mhp, mmp, atk, defense, mat, mdef, agi, luk,
	character_name, character_index, face_name, face_index,
	map_id, x, y, direction, revive_map_id, revive_x, revive_y,
	gold, guild_id, created_at, last_login_at`

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCharacter(row rowScanner) (*Character, error) {
	var c Character
	if err := row.Scan(&c.ID, &c.AccountID, &c.Slot, &c.Name, &c.NameKey, &c.ClassID, &c.Sex,
		&c.Level, &c.Exp, &c.HP, &c.MP, &c.Points,
		&c.MHP, &c.MMP, &c.ATK, &c.DEF, &c.MAT, &c.MDF, &c.AGI, &c.LUK,
		&c.CharacterName, &c.CharacterIndex, &c.FaceName, &c.FaceIndex,
		&c.MapID, &c.X, &c.Y, &c.Direction, &c.ReviveMapID, &c.ReviveX, &c.ReviveY,
		&c.Gold, &c.GuildID, &c.CreatedAt, &c.LastLoginAt); err != nil {
		return nil, err
	}
	return &c, nil
}

// CreateCharacter inserts a character and an empty state aggregate atomically.
func (s *Store) CreateCharacter(ctx context.Context, c *Character) error {
	if c == nil {
		return fmt.Errorf("store: nil character")
	}
	if c.CreatedAt == 0 {
		c.CreatedAt = time.Now().Unix()
	}
	if c.NameKey == "" {
		c.NameKey = NameKey(c.Name)
	}
	state := DefaultCharacterState()
	data, err := EncodeState(state)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	id, err := s.insertID(ctx, tx, `INSERT INTO characters
		(account_id, slot, name, name_key, class_id, sex, level, exp, hp, mp, points,
		 mhp, mmp, atk, defense, mat, mdef, agi, luk,
		 character_name, character_index, face_name, face_index,
		 map_id, x, y, direction, revive_map_id, revive_x, revive_y,
		 gold, guild_id, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		c.AccountID, c.Slot, c.Name, c.NameKey, c.ClassID, c.Sex, c.Level, c.Exp, c.HP, c.MP, c.Points,
		c.MHP, c.MMP, c.ATK, c.DEF, c.MAT, c.MDF, c.AGI, c.LUK,
		c.CharacterName, c.CharacterIndex, c.FaceName, c.FaceIndex,
		c.MapID, c.X, c.Y, c.Direction, c.ReviveMapID, c.ReviveX, c.ReviveY,
		c.Gold, c.GuildID, c.CreatedAt, c.LastLoginAt)
	if err != nil {
		return fmt.Errorf("store: create character: %w", err)
	}
	c.ID = id
	if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO character_state (character_id, version, data, updated_at) VALUES (?, ?, ?, ?)`),
		id, state.Version, data, c.CreatedAt); err != nil {
		return fmt.Errorf("store: create character state: %w", err)
	}
	return tx.Commit()
}

// ListCharacters returns the account's characters ordered by slot.
func (s *Store) ListCharacters(ctx context.Context, accountID int64) ([]*Character, error) {
	rows, err := s.db.QueryContext(ctx, s.q(`SELECT `+characterColumns+` FROM characters WHERE account_id = ? ORDER BY slot`), accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Character
	for rows.Next() {
		c, err := scanCharacter(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCharacter loads a character and its aggregate state.
func (s *Store) GetCharacter(ctx context.Context, id int64) (*Character, *CharacterState, error) {
	c, err := scanCharacter(s.db.QueryRowContext(ctx, s.q(`SELECT `+characterColumns+` FROM characters WHERE id = ?`), id))
	if err != nil {
		return nil, nil, err
	}
	st, err := s.GetCharacterState(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	return c, st, nil
}

// GetCharacterByName loads a character by case-insensitive name.
func (s *Store) GetCharacterByName(ctx context.Context, name string) (*Character, error) {
	return scanCharacter(s.db.QueryRowContext(ctx, s.q(`SELECT `+characterColumns+` FROM characters WHERE name_key = ?`), NameKey(name)))
}

// CharacterExists reports whether a character name is taken.
func (s *Store) CharacterExists(ctx context.Context, name string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, s.q(`SELECT 1 FROM characters WHERE name_key = ?`), NameKey(name)).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

// GetCharacterState loads the aggregate, returning defaults when the row is
// absent.
func (s *Store) GetCharacterState(ctx context.Context, id int64) (*CharacterState, error) {
	var data string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT data FROM character_state WHERE character_id = ?`), id).Scan(&data)
	if err == sql.ErrNoRows {
		st := DefaultCharacterState()
		return &st, nil
	}
	if err != nil {
		return nil, err
	}
	st, err := DecodeState(data)
	if err != nil {
		return nil, err
	}
	return &st, nil
}

// SaveCharacter updates the core row and the aggregate atomically.
func (s *Store) SaveCharacter(ctx context.Context, c *Character, st *CharacterState) error {
	if c == nil || st == nil {
		return fmt.Errorf("store: nil character or state")
	}
	if c.NameKey == "" {
		c.NameKey = NameKey(c.Name)
	}
	if st.Version == 0 {
		st.Version = StateVersion
	}
	data, err := EncodeState(*st)
	if err != nil {
		return err
	}
	now := time.Now().Unix()
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var one int
	if err := tx.QueryRowContext(ctx, s.q(`SELECT 1 FROM characters WHERE id = ? AND account_id = ?`), c.ID, c.AccountID).Scan(&one); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("store: character %d not found for account %d", c.ID, c.AccountID)
		}
		return err
	}
	if _, err := tx.ExecContext(ctx, s.q(`UPDATE characters SET
		class_id = ?, sex = ?, level = ?, exp = ?, hp = ?, mp = ?, points = ?,
		mhp = ?, mmp = ?, atk = ?, defense = ?, mat = ?, mdef = ?, agi = ?, luk = ?,
		character_name = ?, character_index = ?, face_name = ?, face_index = ?,
		map_id = ?, x = ?, y = ?, direction = ?, revive_map_id = ?, revive_x = ?, revive_y = ?,
		gold = ?, guild_id = ?, last_login_at = ?
		WHERE id = ? AND account_id = ?`),
		c.ClassID, c.Sex, c.Level, c.Exp, c.HP, c.MP, c.Points,
		c.MHP, c.MMP, c.ATK, c.DEF, c.MAT, c.MDF, c.AGI, c.LUK,
		c.CharacterName, c.CharacterIndex, c.FaceName, c.FaceIndex,
		c.MapID, c.X, c.Y, c.Direction, c.ReviveMapID, c.ReviveX, c.ReviveY,
		c.Gold, c.GuildID, c.LastLoginAt, c.ID, c.AccountID); err != nil {
		return fmt.Errorf("store: update character: %w", err)
	}
	err = tx.QueryRowContext(ctx, s.q(`SELECT 1 FROM character_state WHERE character_id = ?`), c.ID).Scan(&one)
	if err == sql.ErrNoRows {
		if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO character_state (character_id, version, data, updated_at) VALUES (?, ?, ?, ?)`),
			c.ID, st.Version, data, now); err != nil {
			return fmt.Errorf("store: insert character state: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		if _, err := tx.ExecContext(ctx, s.q(`UPDATE character_state SET version = ?, data = ?, updated_at = ? WHERE character_id = ?`),
			st.Version, data, now, c.ID); err != nil {
			return fmt.Errorf("store: update character state: %w", err)
		}
	}
	return tx.Commit()
}

// DeleteCharacter removes the character and cascades its state.
func (s *Store) DeleteCharacter(ctx context.Context, id int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`DELETE FROM characters WHERE id = ?`), id)
	return err
}

// SetCharacterGuild updates only the guild link.
func (s *Store) SetCharacterGuild(ctx context.Context, id, guildID int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE characters SET guild_id = ? WHERE id = ?`), guildID, id)
	return err
}
