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

// SettingWorkerDirective keys the EDITABLE worker-role directive: the text prepended
// to a spawned worker's context (PROJX_ROLE=worker) so it reframes the trunk's
// "dispatch, don't mutate" law as not-its-own and does the task directly instead of
// re-dispatching. Seeded at floor time with DefaultWorkerDirective as its body — edit
// it with `store commit --kind convention --key setting/worker-directive --body "…"`,
// no recompile needed. Key starts with "setting/" so dropSettings excludes it from
// normal preamble rendering (only WorkerDirectiveText's explicit fetch surfaces it).
const SettingWorkerDirective = "setting/worker-directive"

// DefaultWorkerDirective is the SEED content for the worker directive, and the
// fallback WorkerDirectiveText returns when the store has no record yet (a legacy
// project that hasn't re-seeded, or the store is briefly unreachable) — so a worker
// is never left without this reframing just because the record is missing.
const DefaultWorkerDirective = "# YOUR ROLE: WORKER (executor) — READ THIS FIRST\n" +
	"You are a spawned worker agent, NOT the trunk. Your job is to COMPLETE this task yourself: " +
	"read, edit files, and run whatever tools are needed, then stop. Editing files is expected and permitted for you.\n" +
	"The project's \"dispatch, don't mutate\" convention below governs the TRUNK session ONLY — it does NOT apply to you. " +
	"Do NOT dispatch, delegate, spawn another agent, or ask to — just do the work directly.\n\n---\n\n"

// WorkerDirectiveText returns the declared worker directive from the store (the
// setting/worker-directive convention), or DefaultWorkerDirective if s is nil, the
// record is absent, or its body is blank.
func WorkerDirectiveText(s Store) string {
	if s != nil {
		for _, r := range s.List(OfKind(KConvention)) {
			if r.Key == SettingWorkerDirective {
				if body := strings.TrimSpace(r.Body); body != "" {
					return r.Body
				}
				break
			}
		}
	}
	return DefaultWorkerDirective
}

var mutatingTools = map[string]bool{
	"Edit": true, "Write": true, "MultiEdit": true, "NotebookEdit": true,
}

// IsMutatingTool reports whether a tool name writes to files.
func IsMutatingTool(name string) bool { return mutatingTools[strings.TrimSpace(name)] }

// SettingCageMode is the DECLARED, project-level default for OS-level agent
// confinement. Cage stays opt-in — this setting only lets a project turn it ON by
// default (seeded "off"); the PROJX_CAGE env var, when set, always overrides it
// explicitly for one launch. Not the gate/dispatcher-mode axis — orthogonal.
const SettingCageMode = "setting/cage-mode"

// CageModeOn reports whether a project has declared cage-mode ON by default (a
// setting/cage-mode gate-rule with an affirmative body); false (uncaged) if absent.
func CageModeOn(s Store) bool {
	if s == nil {
		return false
	}
	for _, r := range s.List(OfKind(KGateRule)) {
		if r.Key == SettingCageMode {
			switch strings.ToLower(strings.TrimSpace(r.Body)) {
			case "on", "true", "1", "yes":
				return true
			}
			return false
		}
	}
	return false
}

// SettingOverrideAuthority keys the HUMAN-CONTROLLED delegation flag that decides
// whether the AI may override a soft rule at all. Default OFF: the AI can REQUEST an
// override but must not self-grant one. The human delegates by setting this ON (which,
// like the override itself, only they can do out-of-band — the hook blocks the AI from
// flipping it). See doc/enforcement-follow-override-plan and the override-authority ADR.
const SettingOverrideAuthority = "setting/override-authority"

// OverrideAuthorityOn reports whether the human has delegated override authority to the
// AI (a setting/override-authority gate-rule with an affirmative body). Default false —
// absent means NOT delegated, so the AI cannot self-authorize a bypass.
func OverrideAuthorityOn(s Store) bool {
	if s == nil {
		return false
	}
	for _, r := range s.List(OfKind(KGateRule)) {
		if r.Key == SettingOverrideAuthority {
			switch strings.ToLower(strings.TrimSpace(r.Body)) {
			case "on", "true", "1", "yes":
				return true
			}
			return false
		}
	}
	return false
}

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
