package store

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// CreateAccount inserts an account and its empty bank aggregate atomically.
func (s *Store) CreateAccount(ctx context.Context, a *Account) error {
	if a == nil {
		return fmt.Errorf("store: nil account")
	}
	if a.CreatedAt == 0 {
		a.CreatedAt = time.Now().Unix()
	}
	if a.UsernameKey == "" {
		a.UsernameKey = UsernameKey(a.Username)
	}
	if a.PasswordAlgo == "" || len(a.PasswordSalt) == 0 || len(a.PasswordVerifier) == 0 || a.PasswordIterations <= 0 {
		return fmt.Errorf("store: account has incomplete password verifier")
	}
	bank := DefaultBankState()
	bankData, err := EncodeBankState(bank)
	if err != nil {
		return err
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	id, err := s.insertID(ctx, tx, `INSERT INTO accounts
		(username, username_key, email, password_algo, password_salt, password_iterations, password_verifier, group_id, vip_until, created_at, last_login_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		a.Username, a.UsernameKey, a.Email, a.PasswordAlgo, a.PasswordSalt, a.PasswordIterations,
		a.PasswordVerifier, a.GroupID, a.VIPUntil, a.CreatedAt, a.LastLoginAt)
	if err != nil {
		return fmt.Errorf("store: create account: %w", err)
	}
	a.ID = id
	if _, err := tx.ExecContext(ctx, s.q(`INSERT INTO bank_state (account_id, version, gold, data, updated_at) VALUES (?, ?, ?, ?, ?)`),
		id, bank.Version, bank.Gold, bankData, a.CreatedAt); err != nil {
		return fmt.Errorf("store: create bank: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	return nil
}

// GetAccountByUsername loads an account by case-insensitive username.
func (s *Store) GetAccountByUsername(ctx context.Context, username string) (*Account, error) {
	return s.getAccount(ctx, `WHERE username_key = ?`, UsernameKey(username))
}

// GetAccountByID loads an account by primary key.
func (s *Store) GetAccountByID(ctx context.Context, id int64) (*Account, error) {
	return s.getAccount(ctx, `WHERE id = ?`, id)
}

func (s *Store) getAccount(ctx context.Context, where string, arg any) (*Account, error) {
	row := s.db.QueryRowContext(ctx, s.q(`SELECT id, username, username_key, email, password_algo, password_salt, password_iterations, password_verifier, group_id, vip_until, created_at, last_login_at
		FROM accounts `+where), arg)
	var a Account
	if err := row.Scan(&a.ID, &a.Username, &a.UsernameKey, &a.Email, &a.PasswordAlgo, &a.PasswordSalt,
		&a.PasswordIterations, &a.PasswordVerifier, &a.GroupID, &a.VIPUntil, &a.CreatedAt, &a.LastLoginAt); err != nil {
		return nil, err
	}
	return &a, nil
}

// AccountExists reports whether username is taken.
func (s *Store) AccountExists(ctx context.Context, username string) (bool, error) {
	var one int
	err := s.db.QueryRowContext(ctx, s.q(`SELECT 1 FROM accounts WHERE username_key = ?`), UsernameKey(username)).Scan(&one)
	if err == nil {
		return true, nil
	}
	if err == sql.ErrNoRows {
		return false, nil
	}
	return false, err
}

// UpdateAccountLogin sets the last login time.
func (s *Store) UpdateAccountLogin(ctx context.Context, id, at int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE accounts SET last_login_at = ? WHERE id = ?`), at, id)
	return err
}

// UpdateAccountVIP sets the VIP expiry (Unix seconds).
func (s *Store) UpdateAccountVIP(ctx context.Context, id, until int64) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE accounts SET vip_until = ? WHERE id = ?`), until, id)
	return err
}

// UpdateAccountPassword replaces the stored password verifier.
func (s *Store) UpdateAccountPassword(ctx context.Context, id int64, algo string, salt []byte, iterations int, verifier []byte) error {
	_, err := s.db.ExecContext(ctx, s.q(`UPDATE accounts SET password_algo = ?, password_salt = ?, password_iterations = ?, password_verifier = ? WHERE id = ?`),
		algo, salt, iterations, verifier, id)
	return err
}

// insertID inserts a row and returns an auto-increment id on all supported
// dialects.
func (s *Store) insertID(ctx context.Context, tx *sql.Tx, query string, args ...any) (int64, error) {
	if s.driver == Postgres {
		var id int64
		if err := tx.QueryRowContext(ctx, s.q(query)+" RETURNING id", args...).Scan(&id); err != nil {
			return 0, err
		}
		return id, nil
	}
	res, err := tx.ExecContext(ctx, s.q(query), args...)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}
