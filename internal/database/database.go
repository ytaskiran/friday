package database

import (
	"database/sql"
	"fmt"
	"sync"

	_ "github.com/mattn/go-sqlite3"
)

// DB wraps the SQLite connection and provides thread-safe database operations.
type DB struct {
	conn *sql.DB
	mu   sync.RWMutex
}

func New(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite3", dbPath+"?_foreign_keys=on")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	db := &DB{conn: conn}

	if err := db.migrate(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return db, nil
}

func (db *DB) migrate() error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS message_drafts (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			title       TEXT NOT NULL,
			content     TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_drafts_updated ON message_drafts(updated_at DESC)`,

		`CREATE TABLE IF NOT EXISTS contact_attributes (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			jid         TEXT NOT NULL,
			key         TEXT NOT NULL,
			value       TEXT NOT NULL,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(jid, key)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_attrs_jid ON contact_attributes(jid)`,
		`CREATE INDEX IF NOT EXISTS idx_attrs_key ON contact_attributes(key)`,

		`CREATE TABLE IF NOT EXISTS contact_groups (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			name        TEXT NOT NULL UNIQUE,
			created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
		)`,
		`CREATE INDEX IF NOT EXISTS idx_groups_name ON contact_groups(name)`,

		`CREATE TABLE IF NOT EXISTS group_members (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			group_id   INTEGER NOT NULL,
			jid        TEXT NOT NULL,
			added_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (group_id) REFERENCES contact_groups(id) ON DELETE CASCADE,
			UNIQUE(group_id, jid)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_members_group ON group_members(group_id)`,
		`CREATE INDEX IF NOT EXISTS idx_members_jid ON group_members(jid)`,

		`CREATE TABLE IF NOT EXISTS batch_runs (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			draft_id        INTEGER NOT NULL,
			group_id        INTEGER NOT NULL,
			group_name      TEXT NOT NULL,
			draft_title     TEXT NOT NULL,
			status          TEXT NOT NULL DEFAULT 'queued',
			total_count     INTEGER NOT NULL DEFAULT 0,
			sent_count      INTEGER NOT NULL DEFAULT 0,
			failed_count    INTEGER NOT NULL DEFAULT 0,
			error_message   TEXT,
			started_at      DATETIME,
			completed_at    DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (draft_id) REFERENCES message_drafts(id),
			FOREIGN KEY (group_id) REFERENCES contact_groups(id)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_runs_status ON batch_runs(status)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_runs_created ON batch_runs(created_at DESC)`,

		`CREATE TABLE IF NOT EXISTS batch_messages (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			batch_run_id    INTEGER NOT NULL,
			jid             TEXT NOT NULL,
			contact_name    TEXT,
			status          TEXT NOT NULL DEFAULT 'pending',
			template_content TEXT NOT NULL,
			sent_content    TEXT,
			error_message   TEXT,
			sent_at         DATETIME,
			created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY (batch_run_id) REFERENCES batch_runs(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_messages_run ON batch_messages(batch_run_id)`,
		`CREATE INDEX IF NOT EXISTS idx_batch_messages_status ON batch_messages(status)`,
	}

	for _, migration := range migrations {
		if _, err := db.conn.Exec(migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	return nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) Lock() {
	db.mu.Lock()
}

func (db *DB) Unlock() {
	db.mu.Unlock()
}

func (db *DB) RLock() {
	db.mu.RLock()
}

func (db *DB) RUnlock() {
	db.mu.RUnlock()
}
