package store

import "strings"

// GatePatterns returns the normalized off-limits glob patterns declared by the
// store's gate rules: each rule's Body (falling back to its Key), with a trailing
// "/" expanded to a recursive "/**". setting/* rules are skipped — config/secrets
// are never a project gate. This is the SINGLE source of the gate pattern set:
// DenyRules renders these as agent Read()/Edit() denies, and a path-matching gate
// check tests file paths against them (so the deny globs and the check can never
// drift).
func GatePatterns(s Store) []string {
	var out []string
	for _, r := range s.List(OfKind(KGateRule)) {
		if strings.HasPrefix(r.ID, "setting/") || strings.HasPrefix(r.Key, "setting") {
			continue
		}
		p := strings.TrimSpace(r.Body)
		if p == "" {
			p = strings.TrimSpace(r.Key)
		}
		if p == "" {
			continue
		}
		if strings.HasSuffix(p, "/") {
			p += "**"
		}
		out = append(out, p)
	}
	return out
}

// DenyRules turns the store's gate rules into agent file-tool deny rules —
// "Read(glob)" / "Edit(glob)" — over the GatePatterns set. This is the SINGLE
// gate->deny definition shared by the engine, the engine cell, and the Workbench
// (each previously derived it on its own).
func DenyRules(s Store) []string {
	pats := GatePatterns(s)
	out := make([]string, 0, len(pats)*2)
	for _, p := range pats {
		out = append(out, "Read("+p+")", "Edit("+p+")")
	}
	return out
}

// ── Trunk-dispatch (dispatcher-mode) ─────────────────────────────────────────
// The interaction law: the main session is a DISPATCHER, never a worker. When
// dispatcher-mode is ON, the trunk is denied file-mutating tools so every change is
// routed to a spawned tier-agent; a projx-spawned worker (PROJX_ROLE=worker) is
// exempt. This is a policy gate — NOT the cage/sandbox (which stays separately
// opt-in). Stored as a setting gate-rule (skipped by GatePatterns → never a deny glob).
const SettingDispatcherMode = "setting/dispatcher-mode"

// WorkerDirective is injected at the TOP of a spawned worker's context. A worker is
// the executor (the hands), not the trunk — but it still receives the project floor,
// which includes the "dispatch, don't mutate" trunk law. Without this override the
// worker reads that law and refuses to edit / tries to re-dispatch (role recursion).
// This reframes it: the worker does the task directly. Injected only when PROJX_ROLE=worker.
const WorkerDirective = "# YOUR ROLE: WORKER (executor) — READ THIS FIRST\n" +
	"You are a spawned worker agent, NOT the trunk. Your job is to COMPLETE this task yourself: " +
	"read, edit files, and run whatever tools are needed, then stop. Editing files is expected and permitted for you.\n" +
	"The project's \"dispatch, don't mutate\" convention below governs the TRUNK session ONLY — it does NOT apply to you. " +
	"Do NOT dispatch, delegate, spawn another agent, or ask to — just do the work directly.\n\n---\n\n"

var mutatingTools = map[string]bool{
	"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true,
}

// IsMutatingTool reports whether a tool name writes to files.
func IsMutatingTool(name string) bool { return mutatingTools[strings.TrimSpace(name)] }

// DispatcherModeOn reports whether the trunk-dispatch discipline is enabled (a
// setting/dispatcher-mode gate-rule with an affirmative body).
func DispatcherModeOn(s Store) bool {
	if s == nil {
		return false
	}
	for _, r := range s.List(OfKind(KGateRule)) {
		if r.Key == SettingDispatcherMode {
			switch strings.ToLower(strings.TrimSpace(r.Body)) {
			case "on", "true", "1", "yes":
				return true
			}
			return false
		}
	}
	return false
}
