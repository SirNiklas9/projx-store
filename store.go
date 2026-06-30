// Package store is ProjX's declared-knowledge layer — the second deterministic
// root (core = facts from code; store = facts you declare). Plain data only: no
// AI, no UI, no logic beyond records and a read/write API.
//
// INTERFACE-FIRST by design: this contract is what context/workflow/graph/verify
// depend on. The concrete SQLite schema is deferred until real records teach its
// shape (maximal interface, minimal obligation). An in-memory impl backs it today.
package store

import (
	"errors"
	"strings"
	"sync"
	"time"
)

// Scope identifies which physical file a record lives in. THREE scopes across
// TWO files:
//
//	Global, Workspace -> YOUR store   (one file, travels with you between machines)
//	Project           -> PROJECT store (one per project, stays with the repo)
type Scope int

const (
	// ScopeGlobal: how you work everywhere — recipes, conventions, style. Agnostic.
	ScopeGlobal Scope = iota
	// ScopeWorkspace: this machine's cockpit state — repo list, default gate posture.
	ScopeWorkspace
	// ScopeProject: what's true about this codebase — ADRs, architecture, history,
	// this project's gate rules.
	ScopeProject
)

var scopeNames = map[Scope]string{
	ScopeGlobal: "global", ScopeWorkspace: "workspace", ScopeProject: "project",
}

func (s Scope) String() string {
	if n, ok := scopeNames[s]; ok {
		return n
	}
	return "scope?"
}

// Owner reports which of the two files a scope belongs to: "yours" (global +
// workspace) or "project". Callers use this to route a write to the right file.
func (s Scope) Owner() string {
	if s == ScopeProject {
		return "project"
	}
	return "yours"
}

// Kind is the typed record vocabulary. The store holds records; the engines give
// them meaning.
type Kind int

const (
	KRecipe            Kind = iota // a workflow recipe (global scope only)
	KConvention                    // a style/behavior rule baked into context
	KADR                           // an architecture decision record
	KDoc                           // an explanation / subsystem note
	KHistory                       // an append-only change record (commit output)
	KGateRule                      // a gate/redaction rule
	KDeclaredStructure             // declared module/system grouping for the graph
	KRoute                         // a capability class -> agent launch command (model tier)
)

var kindNames = map[Kind]string{
	KRecipe: "recipe", KConvention: "convention", KADR: "adr", KDoc: "doc",
	KHistory: "history", KGateRule: "gate-rule", KDeclaredStructure: "declared-structure",
	KRoute: "route",
}

func (k Kind) String() string {
	if n, ok := kindNames[k]; ok {
		return n
	}
	return "kind?"
}

// Record is one typed entry. Body is an opaque payload (text/JSON); its concrete
// schema is deferred. Key is a human handle, unique within (Scope, Kind) —
// e.g. a recipe name or an ADR id.
type Record struct {
	ID    string
	Kind  Kind
	Scope Scope
	Key   string
	Body  string
	// UpdatedAt is the last-write clock (0 = unknown), unix-MILLIS-based but guaranteed
	// strictly increasing per store (see stamp). Millis (not seconds) + the monotonic
	// guarantee mean two edits in the same instant never tie — important because the wasm
	// cell's wall clock can be second-resolution. merge/import preserve an incoming value
	// so last-write-wins holds across machines. Origin = which brain/machine wrote it.
	UpdatedAt int64
	Origin    string
}

// nowMillis returns current unix milliseconds. A package var so tests can pin it.
var nowMillis = func() int64 { return time.Now().UnixMilli() }

// stamp returns a write timestamp that is STRICTLY INCREASING within this process, based
// on nowMillis but never repeating or going backwards. This is what makes last-write-wins
// reliable even when the underlying clock is coarse (e.g. the wasm cell's second-resolution
// walltime): two rapid writes get t, t+1 rather than an identical value that would tie and
// force a manual reconcile. Cross-machine, values stay wall-clock-aligned so a genuinely
// newer edit still wins.
var (
	stampMu   sync.Mutex
	lastStamp int64
)

func stamp() int64 {
	stampMu.Lock()
	defer stampMu.Unlock()
	t := nowMillis()
	if t <= lastStamp {
		t = lastStamp + 1
	}
	lastStamp = t
	return t
}

// Filter selects records. The zero value matches everything; a non-nil/non-empty
// field narrows. Pointers distinguish "unset" from "the zero Scope/Kind".
//
// KeyPrefix and Text power task-sliced retrieval (the "query, don't dump" pillar):
// KeyPrefix matches the Key as a path prefix ("minecraft/login" → minecraft/login,
// minecraft/login/backend, …); Text is a case-insensitive substring of Key OR Body.
// Both are case-insensitive so a query needn't know exact casing. SQLite mirrors
// these as WHERE clauses (see SQLite.List) so results match Mem.List exactly.
type Filter struct {
	Scope     *Scope
	Kind      *Kind
	KeyPrefix string
	Text      string
}

// InScope is a convenience filter for one scope.
func InScope(s Scope) Filter { return Filter{Scope: &s} }

// OfKind is a convenience filter for one kind.
func OfKind(k Kind) Filter { return Filter{Kind: &k} }

func (f Filter) match(r Record) bool {
	if f.Scope != nil && r.Scope != *f.Scope {
		return false
	}
	if f.Kind != nil && r.Kind != *f.Kind {
		return false
	}
	if f.KeyPrefix != "" && !strings.HasPrefix(strings.ToLower(r.Key), strings.ToLower(f.KeyPrefix)) {
		return false
	}
	if f.Text != "" {
		t := strings.ToLower(f.Text)
		if !strings.Contains(strings.ToLower(r.Key), t) && !strings.Contains(strings.ToLower(r.Body), t) {
			return false
		}
	}
	return true
}

// Store is the read/write contract every backend satisfies. Both YOUR store and
// the PROJECT store are Store values; the caller picks which by Record.Scope's
// Owner().
type Store interface {
	Put(Record) error
	Get(id string) (Record, bool)
	List(Filter) []Record
	Delete(id string) error
}

// ErrNoID is returned by Put when a record has no ID.
var ErrNoID = errors.New("store: record has no ID")
