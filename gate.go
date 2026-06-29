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
