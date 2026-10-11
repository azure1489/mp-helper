CREATE TABLE IF NOT EXISTS accounts (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    name        TEXT NOT NULL UNIQUE,
    appid       TEXT NOT NULL,
    app_secret  TEXT NOT NULL,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS api_keys (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    account_id  INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    key_hash    TEXT NOT NULL UNIQUE,
    prefix      TEXT NOT NULL,
    label       TEXT,
    created_at  DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at  DATETIME
);

-- key_hash 的 UNIQUE 约束已隐式建唯一索引，无需再显式建。
CREATE INDEX IF NOT EXISTS idx_api_keys_account ON api_keys(account_id);

-- 人工补录的文章信息（推送时刻、分类），按 msgid（"<msg_data_id>_<篇序>"）关联统计数据。
CREATE TABLE IF NOT EXISTS article_meta (
    account_id   INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    msgid        TEXT NOT NULL,
    published_at TEXT,
    category     TEXT,
    note         TEXT,
    updated_at   DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (account_id, msgid)
);

-- 统计接口单日响应缓存（目前只缓存 getarticletotaldetail，分析接口按天逐日调用它）。
CREATE TABLE IF NOT EXISTS datacube_cache (
    account_id INTEGER NOT NULL REFERENCES accounts(id) ON DELETE CASCADE,
    api        TEXT NOT NULL,
    ref_date   TEXT NOT NULL,
    body       BLOB NOT NULL,
    fetched_at INTEGER NOT NULL,
    PRIMARY KEY (account_id, api, ref_date)
);
