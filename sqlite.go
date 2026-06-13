package store

import (
	"database/sql"
	"fmt"

	_ "modernc.org/sqlite" // pure-Go SQLite driver, registered as "sqlite" (no CGo).
)

// SQLite is a disk- or memory-backed Store. It behaves identically to Mem — same
// insert-or-replace Put, same (Record, bool) Get, no-op Delete, and ID-sorted
// List — but persists through database/sql on the pure-Go modernc.org/sqlite
// driver. Open runs migrations; the backing schema is a single records table.
type SQLite struct {
	db *sql.DB
}

// Open opens (or creates) a SQLite-backed Store at path and runs any pending
// migrations. path may be a file path or ":memory:" for an ephemeral store. The
// returned *SQLite must be closed with Close.
func Open(path string) (*SQLite, error) {
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, fmt.Errorf("store: open %q: %w", path, err)
	}
	// One connection keeps an in-memory DB (which is per-connection) coherent and
	// serializes writes; this store is small and not a hot path.
	db.SetMaxOpenConns(1)
	if err := migrate(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("store: migrate %q: %w", path, err)
	}
	return &SQLite{db: db}, nil
}

// Close releases the underlying database handle.
func (s *SQLite) Close() error {
	if err := s.db.Close(); err != nil {
		return fmt.Errorf("store: close: %w", err)
	}
	return nil
}

// Put inserts or replaces a record by ID.
func (s *SQLite) Put(r Record) error {
	if r.ID == "" {
		return ErrNoID
	}
	if _, err := s.db.Exec(
		`INSERT INTO records (id, kind, scope, rkey, body) VALUES (?, ?, ?, ?, ?)
		 ON CONFLICT(id) DO UPDATE SET kind = excluded.kind, scope = excluded.scope,
		 rkey = excluded.rkey, body = excluded.body`,
		r.ID, int(r.Kind), int(r.Scope), r.Key, r.Body,
	); err != nil {
		return fmt.Errorf("store: put %q: %w", r.ID, err)
	}
	return nil
}

// Get returns the record with the given ID, if present.
func (s *SQLite) Get(id string) (Record, bool) {
	var (
		r          Record
		kind, scop int
	)
	row := s.db.QueryRow(
		`SELECT id, kind, scope, rkey, body FROM records WHERE id = ?`, id,
	)
	if err := row.Scan(&r.ID, &kind, &scop, &r.Key, &r.Body); err != nil {
		return Record{}, false
	}
	r.Kind, r.Scope = Kind(kind), Scope(scop)
	return r, true
}

// Delete removes a record by ID. Deleting a missing ID is a no-op (not an error).
func (s *SQLite) Delete(id string) error {
	if _, err := s.db.Exec(`DELETE FROM records WHERE id = ?`, id); err != nil {
		return fmt.Errorf("store: delete %q: %w", id, err)
	}
	return nil
}

// List returns all records matching the filter, sorted by ID. The filter is
// applied as SQL WHERE clauses so results match Mem.List exactly.
func (s *SQLite) List(f Filter) []Record {
	query := `SELECT id, kind, scope, rkey, body FROM records`
	var (
		clauses []string
		args    []any
	)
	if f.Scope != nil {
		clauses = append(clauses, "scope = ?")
		args = append(args, int(*f.Scope))
	}
	if f.Kind != nil {
		clauses = append(clauses, "kind = ?")
		args = append(args, int(*f.Kind))
	}
	for i, c := range clauses {
		if i == 0 {
			query += " WHERE "
		} else {
			query += " AND "
		}
		query += c
	}
	query += " ORDER BY id ASC"

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil
	}
	defer rows.Close()

	var out []Record
	for rows.Next() {
		var (
			r          Record
			kind, scop int
		)
		if err := rows.Scan(&r.ID, &kind, &scop, &r.Key, &r.Body); err != nil {
			return nil
		}
		r.Kind, r.Scope = Kind(kind), Scope(scop)
		out = append(out, r)
	}
	if rows.Err() != nil {
		return nil
	}
	return out
}

// compile-time assertion that SQLite satisfies Store.
var _ Store = (*SQLite)(nil)
