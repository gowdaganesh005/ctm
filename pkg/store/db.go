package store

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

type DB struct {
	conn *sql.DB
}

func NewDB(dbPath string) (*DB, error) {
	conn, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite db at %s: %w", dbPath, err)
	}

	// Set connection pool settings for SQLite WAL mode
	conn.SetMaxOpenConns(1) // Avoid table lock contention in modernc.org/sqlite
	conn.SetMaxIdleConns(1)

	db := &DB{conn: conn}
	if err := db.initSchema(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return db, nil
}

func (db *DB) Close() error {
	return db.conn.Close()
}

func (db *DB) Conn() *sql.DB {
	return db.conn
}

func (db *DB) initSchema() error {
	queries := []string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,

		`CREATE TABLE IF NOT EXISTS resources (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			uid TEXT UNIQUE NOT NULL,
			namespace TEXT NOT NULL,
			kind TEXT NOT NULL,
			name TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);`,

		`CREATE TABLE IF NOT EXISTS resource_revisions (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			resource_id INTEGER REFERENCES resources(id) ON DELETE CASCADE,
			resource_version TEXT NOT NULL,
			timestamp TIMESTAMP NOT NULL,
			actor_type TEXT NOT NULL,
			manager_name TEXT NOT NULL,
			change_type TEXT NOT NULL,
			raw_json TEXT NOT NULL
		);`,

		`CREATE TABLE IF NOT EXISTS incidents (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			title TEXT NOT NULL,
			severity TEXT NOT NULL,
			namespace TEXT NOT NULL,
			resource_kind TEXT NOT NULL,
			resource_name TEXT NOT NULL,
			detected_at TIMESTAMP NOT NULL,
			description TEXT,
			ranked_changes_json TEXT
		);`,

		`CREATE INDEX IF NOT EXISTS idx_revisions_lookup ON resource_revisions(resource_id, timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_revisions_timestamp ON resource_revisions(timestamp);`,
		`CREATE INDEX IF NOT EXISTS idx_revisions_actor ON resource_revisions(actor_type);`,
		`CREATE INDEX IF NOT EXISTS idx_resources_lookup ON resources(namespace, kind, name);`,
		`CREATE INDEX IF NOT EXISTS idx_incidents_time ON incidents(detected_at);`,
	}

	for _, q := range queries {
		if _, err := db.conn.Exec(q); err != nil {
			return fmt.Errorf("error executing schema statement [%s]: %w", q, err)
		}
	}

	log.Printf("Cluster Time Machine SQLite schema initialized successfully")
	return nil
}
