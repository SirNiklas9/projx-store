package store

import "strings"

// DenyRules turns the store's gate rules into agent file-tool deny rules —
// "Read(glob)" / "Edit(glob)" — a trailing "/" becoming a recursive "/**". This
// is the SINGLE gate->deny definition shared by the engine, the engine cell, and
// the Workbench (each previously derived it on its own). setting/* rules are
// skipped — config/secrets are never a project gate.
func DenyRules(s Store) []string {
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
		out = append(out, "Read("+p+")", "Edit("+p+")")
	}
	return out
}
