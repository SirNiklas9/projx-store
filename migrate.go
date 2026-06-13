package store

import (
	"database/sql"
	"fmt"
)

// migrations is the forward-only, ordered list of schema steps. Each entry is run
// inside a transaction when the DB's recorded version is below its index+1. There
// are no down-migrations: the schema only moves forward.
//
// Migration #1 creates the records table that backs the Store contract.
var migrations = []string{
	// 1: the records table. rkey (not key) because KEY is reserved in SQL.
	`CREATE TABLE records (
		id    TEXT PRIMARY KEY,
		kind  INTEGER,
		scope INTEGER,
		rkey  TEXT,
		body  TEXT
	)`,
}

// migrate brings db up to the latest schema version, applying any pending
// migrations in order. It is idempotent: a fully-migrated DB is a no-op. A failed
// migration rolls back its transaction and returns an error, so the DB is never
// left half-migrated.
func migrate(db *sql.DB) error {
	if _, err := db.Exec(
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
	); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	var version int
	row := db.QueryRow(`SELECT version FROM schema_version LIMIT 1`)
	switch err := row.Scan(&version); err {
	case sql.ErrNoRows:
		if _, err := db.Exec(`INSERT INTO schema_version (version) VALUES (0)`); err != nil {
			return fmt.Errorf("seed schema_version: %w", err)
		}
		version = 0
	case nil:
		// existing version read.
	default:
		return fmt.Errorf("read schema_version: %w", err)
	}

	for i := version; i < len(migrations); i++ {
		if err := applyMigration(db, i+1, migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
	}
	return nil
}

// applyMigration runs one migration's SQL and bumps the recorded version to v,
// both inside a single transaction so a failure leaves nothing behind.
func applyMigration(db *sql.DB, v int, stmt string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin: %w", err)
	}
	if _, err := tx.Exec(stmt); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("exec: %w", err)
	}
	if _, err := tx.Exec(`UPDATE schema_version SET version = ?`, v); err != nil {
		_ = tx.Rollback()
		return fmt.Errorf("bump version: %w", err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit: %w", err)
	}
	return nil
}
