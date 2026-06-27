package store

import "strings"

// Floor — the universal contract seeded into every project so no one starts from
// zero. This is THE single definition: both the native engine and the engine cell
// seed from it (no duplication). Records are project-scoped (they stay with the
// repo) and tagged origin "seed:floor".

// SeedRec is a key/body pair awaiting a Kind + Scope at seed time.
type SeedRec struct{ Key, Body string }

// FloorConventions are the behaviour rules baked into every project's contract.
var FloorConventions = []SeedRec{
	{"read before acting", "Read this store contract first. The store is authoritative project knowledge — not any README or .md file. Never act before reading it."},
	{"commit what you learn", "When you decide or learn something durable, commit it to the store (convention/adr) — not a markdown file."},
	{"deterministic first", "Prefer deterministic ops (verify, store, tests) over agent reasoning whenever a tool can do the job."},
	{"secrets by codename", "Never read, edit, or print secret material. Reference secrets only by codename."},
}

// FloorGates are the off-limits paths every project denies by default.
var FloorGates = []SeedRec{
	{"secrets dir", "secret/**"},
	{"dotenv files", ".env*"},
	{"private keys", "**/*.key"},
	{"ssh material", "**/.ssh/**"},
}

// SeedFloor writes the floor contract (conventions + gate rules) into s as
// project-scoped records. Put replaces by ID, so re-seeding is harmless. Returns
// the number of records written.
func SeedFloor(s Store) int {
	n := 0
	for _, c := range FloorConventions {
		if s.Put(floorRecord(KConvention, c)) == nil {
			n++
		}
	}
	for _, g := range FloorGates {
		if s.Put(floorRecord(KGateRule, g)) == nil {
			n++
		}
	}
	for _, rt := range FloorRoutes {
		if s.Put(floorRoute(rt)) == nil {
			n++
		}
	}
	return n
}

func floorRecord(kind Kind, r SeedRec) Record {
	return Record{
		ID:     kind.String() + "/" + seedSlug(r.Key),
		Kind:   kind,
		Scope:  ScopeProject,
		Key:    r.Key,
		Body:   r.Body,
		Origin: "seed:floor",
	}
}

// seedSlug normalises a key into an id-safe token (lowercase, alphanumerics,
// dashes). Matches the engine/CLI slug so UI-, CLI-, and seed-created records
// share one ID scheme.
func seedSlug(s string) string {
	var b strings.Builder
	prevDash := false
	for _, r := range strings.ToLower(strings.TrimSpace(s)) {
		switch {
		case r >= 'a' && r <= 'z', r >= '0' && r <= '9':
			b.WriteRune(r)
			prevDash = false
		default:
			if !prevDash {
				b.WriteByte('-')
				prevDash = true
			}
		}
	}
	return strings.Trim(b.String(), "-")
}
