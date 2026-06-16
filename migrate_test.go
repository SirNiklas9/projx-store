package store

import "testing"

// TestMigrationUpgradeV1toCurrent proves the REAL-WORLD upgrade path: a store created at
// schema v1 (the original 5-column records table) self-upgrades to the current schema on
// next Open, back-filling the new columns (updated_at=0, origin="") without losing data —
// and that fresh writes afterward get stamped/origin'd normally.
func TestMigrationUpgradeV1toCurrent(t *testing.T) {
	c, err := openConn(":memory:")
	if err != nil {
		t.Fatalf("openConn: %v", err)
	}
	defer c.close()
	must := func(q string) {
		if e := c.exec(q); e != nil {
			t.Fatalf("setup %q: %v", q, e)
		}
	}
	// simulate a v1 database: original 5-column schema, version pinned at 1, one row.
	must(`CREATE TABLE schema_version (version INTEGER NOT NULL)`)
	must(`INSERT INTO schema_version (version) VALUES (1)`)
	must(`CREATE TABLE records (id TEXT PRIMARY KEY, kind INTEGER, scope INTEGER, rkey TEXT, body TEXT)`)
	must(`INSERT INTO records (id, kind, scope, rkey, body) VALUES ('old', 1, 0, 'k', 'v1body')`)

	// run the CURRENT migrator -> applies migrations 2 (updated_at) and 3 (origin).
	if err := migrate(c); err != nil {
		t.Fatalf("migrate upgrade v1->current: %v", err)
	}

	s := &SQLite{c: c}
	got, ok := s.Get("old")
	if !ok {
		t.Fatal("v1 record vanished after upgrade")
	}
	if got.Body != "v1body" || got.Key != "k" || got.Kind != KConvention || got.Scope != ScopeGlobal {
		t.Fatalf("v1 fields lost/changed: %+v", got)
	}
	if got.UpdatedAt != 0 || got.Origin != "" {
		t.Fatalf("back-fill wrong (want 0/\"\"): %+v", got)
	}

	// a fresh write after upgrade gets stamped + carries origin
	defer pinClock(777)()
	if err := s.Put(Record{ID: "new", Kind: KConvention, Scope: ScopeGlobal, Key: "n", Body: "b", Origin: "brainA"}); err != nil {
		t.Fatalf("Put after upgrade: %v", err)
	}
	n, _ := s.Get("new")
	if n.UpdatedAt != 777 || n.Origin != "brainA" {
		t.Fatalf("post-upgrade write metadata wrong: %+v", n)
	}
}
