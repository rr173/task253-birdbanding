package store

import (
	"database/sql"
	"fmt"
	"time"

	_ "modernc.org/sqlite"
)

// DB 封装 SQLite 连接与建表迁移。使用纯 Go 驱动，CGO 无关。
type DB struct {
	*sql.DB
}

// Open 打开（或创建）SQLite 数据库并执行建表迁移。
func Open(path string) (*DB, error) {
	// 多连接 + 外键 + 忙碌重试，保证并发导入安全。
	dsn := fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&cache=shared", path)
	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	conn.SetMaxOpenConns(4)
	conn.SetMaxIdleConns(4)
	if err := conn.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	db := &DB{conn}
	if err := db.migrate(); err != nil {
		return nil, fmt.Errorf("migrate: %w", err)
	}
	return db, nil
}

// Close 关闭底层连接。
func (db *DB) Close() error {
	return db.DB.Close()
}

func (db *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS batches (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  published_at TEXT
);
CREATE TABLE IF NOT EXISTS locations (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  lat REAL NOT NULL,
  lon REAL NOT NULL,
  precision_m REAL NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS individuals (
  id TEXT PRIMARY KEY,
  ring_code TEXT NOT NULL UNIQUE,
  species TEXT NOT NULL,
  created_at TEXT NOT NULL
);
CREATE TABLE IF NOT EXISTS events (
  id TEXT PRIMARY KEY,
  batch_id TEXT NOT NULL REFERENCES batches(id),
  individual_id TEXT REFERENCES individuals(id),
  ring_code TEXT NOT NULL,
  type TEXT NOT NULL,
  location_id TEXT NOT NULL REFERENCES locations(id),
  event_date TEXT NOT NULL,
  status TEXT NOT NULL,
  fingerprint TEXT NOT NULL UNIQUE,
  error_reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_events_fingerprint ON events(fingerprint);
CREATE INDEX IF NOT EXISTS idx_events_individual ON events(individual_id);
CREATE INDEX IF NOT EXISTS idx_events_batch ON events(batch_id);
CREATE TABLE IF NOT EXISTS edges (
  id TEXT PRIMARY KEY,
  individual_id TEXT NOT NULL REFERENCES individuals(id),
  from_event_id TEXT NOT NULL REFERENCES events(id),
  to_event_id TEXT NOT NULL REFERENCES events(id),
  distance_km REAL NOT NULL,
  days REAL NOT NULL,
  speed_km_day REAL NOT NULL,
  status TEXT NOT NULL,
  reason TEXT NOT NULL DEFAULT '',
  created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_edges_individual ON edges(individual_id);
CREATE TABLE IF NOT EXISTS versions (
  id TEXT PRIMARY KEY,
  individual_id TEXT NOT NULL REFERENCES individuals(id),
  name TEXT NOT NULL,
  status TEXT NOT NULL,
  created_at TEXT NOT NULL,
  frozen_at TEXT
);
CREATE TABLE IF NOT EXISTS version_edges (
  version_id TEXT NOT NULL REFERENCES versions(id),
  edge_id TEXT NOT NULL REFERENCES edges(id),
  seq INTEGER NOT NULL,
  PRIMARY KEY (version_id, edge_id)
);
`
	if _, err := db.Exec(schema); err != nil {
		return err
	}
	return nil
}

// nowISO 返回 UTC 的 ISO 时间戳字符串。
func nowISO() string {
	return time.Now().UTC().Format(time.RFC3339)
}
