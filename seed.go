package store

import "strings"

// Floor — the universal contract seeded into every project so no one starts from
// zero. This is THE single definition: both the native engine and the engine cell
// seed from it (no duplication). Records are project-scoped (they stay with the
// repo) and tagged origin "seed:floor".

// SeedRec is a key/body pair awaiting a Kind + Scope at seed time.
type SeedRec struct{ Key, Body string }

// FloorConventions are the behaviour rules baked into every project's contract.
// This is the standing LAW: it is re-sent in full on EVERY message (see
// AgentContextFloor / AgentContextDelta), so each entry is paid per-turn — keep
// bodies terse and keep the set to rules that are correct in ANY repo. Per-language
// rules belong in stacks (profiles.go); per-repo rules in a project seed.toml.
var FloorConventions = []SeedRec{
	{"dispatch don't mutate", "The main session is a DISPATCHER, not a worker. Do not edit files directly from the trunk — route each task to its tier and spawn an agent to do the work (`projx-engine dispatch --run \"<task>\"`). The trunk reads, plans, dispatches, and VERIFIES the returned diff; spawned agents do the mutation. Tight iterative work = keep messaging one spawned agent rather than re-spawning. When dispatcher-mode is on this is enforced by a gate, not left to willpower."},
	{"read before acting", "Read this store contract first. The store is authoritative project knowledge — not any README or .md file. Never act before reading it."},
	{"commit what you learn", "When you decide or learn something durable, commit it to the store (convention/adr) — not a markdown file."},
	{"deterministic first", "Prefer deterministic ops (verify, store, tests) over agent reasoning whenever a tool can do the job."},
	{"secrets by codename", "Never read, edit, or print secret material. Reference secrets only by codename."},
	{"verify before done", "Don't report a task complete until you've exercised the change and seen it work — tests green, or the affected flow driven end-to-end. \"Should work\" is not done; run it and confirm before you claim it."},
	{"smallest change on task", "Change only what the task needs. Don't refactor, rename, or reformat unrelated code — keep the diff small and reviewable so the intent is obvious. Note tangents; don't fold them in."},
	{"fix don't disable", "Never delete, comment out, weaken, or skip a feature or test just to make something pass. Fix the root cause; if you can't, stop and report it rather than hide it."},
	{"match the surroundings", "New code reads like the code around it — same naming, structure, and idioms. The existing code is the style guide; don't impose a different one."},
	{"no irreversible ops unprompted", "Never run destructive or outward-facing commands (force-push, hard reset, history rewrite, mass delete, publish, deploy) without explicit confirmation for that specific action."},
}

// FloorGates are the off-limits paths every project denies by default.
var FloorGates = []SeedRec{
	{"secrets dir", "secret/**"},
	{"dotenv files", ".env*"},
	{"private keys", "**/*.key"},
	{"pem material", "**/*.pem"},
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
	// Dispatcher-mode ON by default (the trunk-dispatch discipline, proven e2e): the
	// trunk is denied file mutation and routes work to tier-agents. One setting flips it
	// off (`store commit --kind gate-rule --key setting/dispatcher-mode --body off`).
	if s.Put(Record{
		ID: KGateRule.String() + "/" + seedSlug(SettingDispatcherMode), Kind: KGateRule,
		Scope: ScopeProject, Key: SettingDispatcherMode, Body: "on", Origin: "seed:floor",
		Enforcement: EnforcementSoft, // deny-by-default, overridable with a logged reason
	}) == nil {
		n++
	}
	// The default provider integration — Claude Code, as replaceable DATA. Override by
	// declaring your own integration (seed.toml [[integration]]) and marking it active.
	if s.Put(IntegrationRecord(DefaultIntegration)) == nil {
		n++
	}
	if s.Put(IntegrationActiveRecord(DefaultIntegration.Name)) == nil {
		n++
	}
	// Cage stays OPT-IN (seeded "off" so the setting is discoverable via `store list`,
	// not just implicitly absent) — flip per-project with `store commit --kind gate-rule
	// --key setting/cage-mode --body on`, or override per-launch with PROJX_CAGE.
	if s.Put(Record{
		ID: KGateRule.String() + "/" + seedSlug(SettingCageMode), Kind: KGateRule,
		Scope: ScopeProject, Key: SettingCageMode, Body: "off", Origin: "seed:floor",
	}) == nil {
		n++
	}
	// The worker directive — EDITABLE, not hardcoded: `store commit --kind convention
	// --key setting/worker-directive --body "…"` changes what a spawned worker is told,
	// no recompile.
	if s.Put(Record{
		ID: KConvention.String() + "/" + seedSlug(SettingWorkerDirective), Kind: KConvention,
		Scope: ScopeProject, Key: SettingWorkerDirective, Body: DefaultWorkerDirective, Origin: "seed:floor",
	}) == nil {
		n++
	}
	// The classifier's keyword vocabulary — seeded so it's a real, editable rule (see
	// ClassifyStore): after this, adding/removing a word here actually changes routing.
	for _, kw := range FloorKeywordSeeds {
		key := settingRouteKeywords + "/" + kw.Key
		if s.Put(Record{
			ID: KRoute.String() + "/" + seedSlug(key), Kind: KRoute,
			Scope: ScopeProject, Key: key, Body: kw.Body, Origin: "seed:floor",
		}) == nil {
			n++
		}
	}
	return n
}

func floorRecord(kind Kind, r SeedRec) Record {
	// The off-limits gate floor is HARD (a wall, never overridable); conventions are
	// advisory context (left empty → derived). dispatcher-mode is seeded separately as
	// soft in SeedFloor. This bakes the tier into the DATA for fresh stores; existing
	// stores derive the same tier via Tier() until re-seeded.
	enf := ""
	if kind == KGateRule {
		enf = EnforcementHard
	}
	return Record{
		ID:          kind.String() + "/" + seedSlug(r.Key),
		Kind:        kind,
		Scope:       ScopeProject,
		Key:         r.Key,
		Body:        r.Body,
		Origin:      "seed:floor",
		Enforcement: enf,
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
