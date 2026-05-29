package store

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
)

// APIKey 是一把业务 API Key 的元数据（不含明文）。
type APIKey struct {
	ID        int64
	AccountID int64
	Prefix    string
	Label     string
	CreatedAt string
	RevokedAt sql.NullString
}

// GenerateKey 生成明文 key，形如 mpk_<32 hex>。
func GenerateKey() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return "mpk_" + hex.EncodeToString(b), nil
}

// HashKey 返回明文 key 的 sha256 hex。
func HashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

// CreateKey 为账号创建一把 key，返回明文（仅此一次）。
func (s *Store) CreateKey(accountID int64, label string) (id int64, plaintext, prefix string, err error) {
	if _, err = s.GetAccount(accountID); err != nil {
		return 0, "", "", err
	}
	plaintext, err = GenerateKey()
	if err != nil {
		return 0, "", "", err
	}
	prefix = plaintext[:12] // "mpk_" + 前 8 位 hex
	res, err := s.db.Exec(
		`INSERT INTO api_keys (account_id, key_hash, prefix, label) VALUES (?, ?, ?, ?)`,
		accountID, HashKey(plaintext), prefix, label)
	if err != nil {
		return 0, "", "", err
	}
	id, _ = res.LastInsertId()
	return id, plaintext, prefix, nil
}

func (s *Store) ListKeys() ([]*APIKey, error) {
	rows, err := s.db.Query(
		`SELECT id, account_id, prefix, COALESCE(label, ''), created_at, revoked_at
		   FROM api_keys ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*APIKey
	for rows.Next() {
		k := &APIKey{}
		if err := rows.Scan(&k.ID, &k.AccountID, &k.Prefix, &k.Label, &k.CreatedAt, &k.RevokedAt); err != nil {
			return nil, err
		}
		out = append(out, k)
	}
	return out, rows.Err()
}

func (s *Store) RevokeKey(id int64) error {
	res, err := s.db.Exec(
		`UPDATE api_keys SET revoked_at=CURRENT_TIMESTAMP WHERE id=? AND revoked_at IS NULL`, id)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

// ResolveAccountByKey 用明文 key 查到未吊销的账号。
func (s *Store) ResolveAccountByKey(plaintext string) (*Account, error) {
	a := &Account{}
	err := s.db.QueryRow(
		`SELECT a.id, a.name, a.appid, a.app_secret
		   FROM api_keys k JOIN accounts a ON a.id = k.account_id
		  WHERE k.key_hash = ? AND k.revoked_at IS NULL`,
		HashKey(plaintext)).
		Scan(&a.ID, &a.Name, &a.AppID, &a.AppSecret)
	if err == sql.ErrNoRows {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return a, nil
}
