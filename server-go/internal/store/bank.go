package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// GetBank loads an account's bank aggregate. Missing rows yield a default.
func (s *Store) GetBank(ctx context.Context, accountID int64) (*BankState, error) {
	var gold int64
	var data string
	err := s.db.QueryRowContext(ctx, s.q(`SELECT gold, data FROM bank_state WHERE account_id = ?`), accountID).Scan(&gold, &data)
	if err == sql.ErrNoRows {
		st := DefaultBankState()
		return &st, nil
	}
	if err != nil {
		return nil, err
	}
	st, err := DecodeBankState(data)
	if err != nil {
		return nil, err
	}
	st.Gold = gold
	return &st, nil
}

// SaveBank updates the bank aggregate atomically.
func (s *Store) SaveBank(ctx context.Context, accountID int64, st *BankState) error {
	if st == nil {
		return fmt.Errorf("store: nil bank state")
	}
	if st.Version == 0 {
		st.Version = StateVersion
	}
	data, err := EncodeBankState(*st)
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
	err = tx.QueryRowContext(ctx, s.q(`SELECT 1 FROM bank_state WHERE account_id = ?`), accountID).Scan(&one)
	if err == sql.ErrNoRows {
		if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO bank_state (account_id, version, gold, data, updated_at) VALUES (?, ?, ?, ?, ?)`),
			accountID, st.Version, st.Gold, data, now); err != nil {
			return fmt.Errorf("store: insert bank: %w", err)
		}
	} else if err != nil {
		return err
	} else {
		if _, err := tx.ExecContext(ctx, s.q(`UPDATE bank_state SET version = ?, gold = ?, data = ?, updated_at = ? WHERE account_id = ?`),
			st.Version, st.Gold, data, now, accountID); err != nil {
			return fmt.Errorf("store: update bank: %w", err)
		}
	}
	return tx.Commit()
}
