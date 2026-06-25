package store

import "fmt"

// migrations is the forward-only, ordered list of schema steps. Each entry is run
// when the DB's recorded version is below its index+1. There are no down-migrations:
// the schema only moves forward.
//
// Migration #1 creates the records table that backs the Store contract. #2 and #3 add
// the merge metadata (updated_at for last-write-wins, origin = which brain/machine wrote
// it) needed to consolidate two stores — see merge.go. Forward-only: each ALTER adds one
// column with a constant DEFAULT so existing rows back-fill safely (updated_at=0 = oldest,
// origin="" = unknown).
var migrations = []string{
	// 1: the records table. rkey (not key) because KEY is reserved in SQL.
	`CREATE TABLE records (
		id    TEXT PRIMARY KEY,
		kind  INTEGER,
		scope INTEGER,
		rkey  TEXT,
		body  TEXT
	)`,
	// 2: last-write-wins clock (unix milliseconds; 0 = unknown/oldest).
	`ALTER TABLE records ADD COLUMN updated_at INTEGER NOT NULL DEFAULT 0`,
	// 3: which brain/machine authored the record (for origin-aware merge; "" = unknown).
	`ALTER TABLE records ADD COLUMN origin TEXT NOT NULL DEFAULT ''`,
}

// migrate brings the connection up to the latest schema version, applying any
// pending migrations in order. It is idempotent: a fully-migrated DB is a no-op.
// It runs over the sqlConn seam (not *sql.DB), so it works identically whether the
// backend is local modernc or the Pulp host capability.
func migrate(c sqlConn) error {
	if err := c.exec(
		`CREATE TABLE IF NOT EXISTS schema_version (version INTEGER NOT NULL)`,
	); err != nil {
		return fmt.Errorf("create schema_version: %w", err)
	}

	rows, err := c.query(`SELECT version FROM schema_version LIMIT 1`)
	if err != nil {
		return fmt.Errorf("read schema_version: %w", err)
	}
	version := 0
	if len(rows) == 0 {
		if err := c.exec(`INSERT INTO schema_version (version) VALUES (0)`); err != nil {
			return fmt.Errorf("seed schema_version: %w", err)
		}
	} else if len(rows[0]) > 0 {
		version = asInt(rows[0][0])
	}

	for i := version; i < len(migrations); i++ {
		if err := c.exec(migrations[i]); err != nil {
			return fmt.Errorf("migration %d: %w", i+1, err)
		}
		if err := c.exec(`UPDATE schema_version SET version = ?`, int64(i+1)); err != nil {
			return fmt.Errorf("migration %d bump version: %w", i+1, err)
		}
	}
	return nil
}
