package store

import (
	"database/sql"
	"errors"
	"time"
)

// ArticleMeta 是人工补录的文章信息；PublishedAt 为北京时间 "YYYY-MM-DD HH:MM"。
type ArticleMeta struct {
	MsgID       string
	PublishedAt string
	Category    string
	Note        string
	UpdatedAt   string
}

// UpsertArticleMeta 新建或部分更新：nil 字段保留原值。
func (s *Store) UpsertArticleMeta(accountID int64, msgid string, publishedAt, category, note *string) error {
	_, err := s.db.Exec(
		`INSERT INTO article_meta (account_id, msgid, published_at, category, note)
		 VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (account_id, msgid) DO UPDATE SET
		     published_at = COALESCE(excluded.published_at, published_at),
		     category     = COALESCE(excluded.category, category),
		     note         = COALESCE(excluded.note, note),
		     updated_at   = CURRENT_TIMESTAMP`,
		accountID, msgid, publishedAt, category, note)
	return err
}

// ListArticleMeta 列出某账号的全部补录信息，按 msgid 排序。
func (s *Store) ListArticleMeta(accountID int64) ([]ArticleMeta, error) {
	rows, err := s.db.Query(
		`SELECT msgid, COALESCE(published_at, ''), COALESCE(category, ''), COALESCE(note, ''), updated_at
		   FROM article_meta WHERE account_id = ? ORDER BY msgid`, accountID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []ArticleMeta
	for rows.Next() {
		var m ArticleMeta
		if err := rows.Scan(&m.MsgID, &m.PublishedAt, &m.Category, &m.Note, &m.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// GetCachedDataCube 取缓存的统计响应；不存在时返回 ErrNotFound。
func (s *Store) GetCachedDataCube(accountID int64, api, refDate string) ([]byte, time.Time, error) {
	var body []byte
	var ts int64
	err := s.db.QueryRow(
		`SELECT body, fetched_at FROM datacube_cache WHERE account_id = ? AND api = ? AND ref_date = ?`,
		accountID, api, refDate).Scan(&body, &ts)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, time.Time{}, ErrNotFound
	}
	if err != nil {
		return nil, time.Time{}, err
	}
	return body, time.Unix(ts, 0), nil
}

// PutCachedDataCube 写入（覆盖）一条统计响应缓存。
func (s *Store) PutCachedDataCube(accountID int64, api, refDate string, body []byte, fetchedAt time.Time) error {
	_, err := s.db.Exec(
		`INSERT INTO datacube_cache (account_id, api, ref_date, body, fetched_at) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT (account_id, api, ref_date) DO UPDATE SET body = excluded.body, fetched_at = excluded.fetched_at`,
		accountID, api, refDate, body, fetchedAt.Unix())
	return err
}
