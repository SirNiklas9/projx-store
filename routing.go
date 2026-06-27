package store

import "strings"

// Routing — the auto model-tier policy, declared in the store. KRoute records map
// a capability class to the agent launch command (model tier). Classify is the
// deterministic task->class triage (no LLM). Route ties them together. This is the
// SINGLE routing definition shared by the native engine and the engine cell.

// FloorRoutes are the default class -> launch-command tiers seeded into every
// project. The model IDs live HERE, once — not hardcoded in the engine binary, so
// updating a tier is a store edit, not a recompile.
var FloorRoutes = []SeedRec{
	{"cheap-fast", "claude --model claude-haiku-4-5-20251001"},     // mechanical: moves, grep, format, classify
	{"default", "claude --model claude-sonnet-4-6"},                // standard: code, tests, review
	{"deep-reasoning", "claude --model claude-opus-4-8"},           // hard: architecture, multi-file, debugging
}

// Classify maps a task to a capability class by keyword. Deterministic; no LLM.
// Priority: deep-reasoning > cheap-fast > default.
func Classify(task string) string {
	t := strings.ToLower(task)
	if containsAny(t, "design", "architect", "architecture", "refactor", "why", "plan",
		"debug", "diagnose", "analyse", "analyze", "redesign", "strategy", "tradeoff", "trade-off") {
		return "deep-reasoning"
	}
	if containsAny(t, "rename", "typo", "format", "comment", "small", "trivial",
		"one-liner", "oneliner", "quick fix", "quickfix", "spelling") {
		return "cheap-fast"
	}
	return "default"
}

// Route classifies task and resolves the launch command from the store's KRoute
// records. Returns the class and the command (cmd is "" if no record for the class).
func Route(s Store, task string) (class, cmd string) {
	class = Classify(task)
	for _, r := range s.List(OfKind(KRoute)) {
		if strings.EqualFold(r.Key, class) {
			return class, r.Body
		}
	}
	return class, ""
}

// floorRoute builds a project-scoped KRoute seed record.
func floorRoute(r SeedRec) Record {
	return Record{
		ID:     KRoute.String() + "/" + seedSlug(r.Key),
		Kind:   KRoute,
		Scope:  ScopeProject,
		Key:    r.Key,
		Body:   r.Body,
		Origin: "seed:floor",
	}
}

func containsAny(s string, toks ...string) bool {
	for _, t := range toks {
		if strings.Contains(s, t) {
			return true
		}
	}
	return false
}
