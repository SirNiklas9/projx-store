package store

import "sort"

// Workspace ties the two physical stores together as one logical Store, realizing
// the "two files, three scopes" design: global + workspace records live in YOUR
// store (the portable file that travels with you), project records live in the
// PROJECT store (one per repo). It routes every operation to the owning store by
// Record.Scope's Owner(), so callers see a single Store regardless of which file
// a record physically lives in.
type Workspace struct {
	// Yours holds global- and workspace-scoped records (the portable file).
	Yours Store
	// Project holds project-scoped records (stays with the repo).
	Project Store
}

// NewWorkspace returns a Workspace over the two physical stores: yours holds
// global + workspace records, project holds project records.
func NewWorkspace(yours, project Store) *Workspace {
	return &Workspace{Yours: yours, Project: project}
}

// Put routes the record to the owning store by Scope.Owner(): "yours" (global +
// workspace) goes to Yours, "project" goes to Project.
func (w *Workspace) Put(r Record) error {
	if r.Scope.Owner() == "project" {
		return w.Project.Put(r)
	}
	return w.Yours.Put(r)
}

// Get returns the record with the given ID. Project is checked first, then Yours;
// the first hit wins.
func (w *Workspace) Get(id string) (Record, bool) {
	if r, ok := w.Project.Get(id); ok {
		return r, true
	}
	return w.Yours.Get(id)
}

// Delete removes the record with the given ID from both stores. Deleting a
// missing ID is a no-op (not an error), matching the backing stores.
func (w *Workspace) Delete(id string) error {
	if err := w.Project.Delete(id); err != nil {
		return err
	}
	return w.Yours.Delete(id)
}

// List returns records matching the filter. When the filter pins a Scope, only
// the store owning that scope is queried; otherwise results from BOTH stores are
// merged and sorted by ID ascending so output is deterministic. The kind filter
// (if any) is applied to both stores via the filter itself.
func (w *Workspace) List(f Filter) []Record {
	if f.Scope != nil {
		if f.Scope.Owner() == "project" {
			return w.Project.List(f)
		}
		return w.Yours.List(f)
	}
	var out []Record
	out = append(out, w.Yours.List(f)...)
	out = append(out, w.Project.List(f)...)
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// compile-time assertion that Workspace satisfies Store.
var _ Store = (*Workspace)(nil)
