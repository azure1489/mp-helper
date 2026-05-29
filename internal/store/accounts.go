package store

import (
	"database/sql"
	"errors"
)

// Account 是一个公众号配置。
type Account struct {
	ID        int64
	Name      string
	AppID     string
	AppSecret string
}

func (s *Store) CreateAccount(name, appid, secret string) (*Account, error) {
	res, err := s.db.Exec(
		`INSERT INTO accounts (name, appid, app_secret) VALUES (?, ?, ?)`,
		name, appid, secret)
	if err != nil {
		return nil, err
	}
	id, _ := res.LastInsertId()
	return &Account{ID: id, Name: name, AppID: appid, AppSecret: secret}, nil
}

func (s *Store) GetAccount(id int64) (*Account, error) {
	a := &Account{}
	err := s.db.QueryRow(
		`SELECT id, name, appid, app_secret FROM accounts WHERE id = ?`, id).
		Scan(&a.ID, &a.Name, &a.AppID, &a.AppSecret)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}

func (s *Store) ListAccounts() ([]*Account, error) {
	rows, err := s.db.Query(`SELECT id, name, appid, app_secret FROM accounts ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*Account
	for rows.Next() {
		a := &Account{}
		if err := rows.Scan(&a.ID, &a.Name, &a.AppID, &a.AppSecret); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// UpdateAccount 部分更新：仅非 nil 字段被更新。
// 用单条 COALESCE 语句原子完成，避免读改写竞态（nil 的 *string 绑定为 NULL → 保留原值）。
func (s *Store) UpdateAccount(id int64, name, appid, secret *string) error {
	res, err := s.db.Exec(
		`UPDATE accounts
		    SET name       = COALESCE(?, name),
		        appid      = COALESCE(?, appid),
		        app_secret = COALESCE(?, app_secret),
		        updated_at = CURRENT_TIMESTAMP
		  WHERE id = ?`,
		name, appid, secret, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Store) DeleteAccount(id int64) error {
	res, err := s.db.Exec(`DELETE FROM accounts WHERE id = ?`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}
