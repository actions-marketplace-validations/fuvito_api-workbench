package openapi

import (
	"sort"
	"strings"
)

// ChangeSet says which operations of a revision of a spec are new or were changed since a base
// version. It uses the same notion of change as Diff (oasdiff), so a change to a schema that
// several operations share counts for every operation that uses it.
type ChangeSet struct {
	// Changed are the operations to check: added and modified ones that exist in the revision,
	// sorted by path then method.
	Changed  []CoveredOperation
	Added    int
	Modified int
	// Removed operations exist only in the base version; they cannot be sent.
	Removed []CoveredOperation
	// Unchanged is how many operations of the revision were not touched.
	Unchanged int
	// Unattributed counts changes that name no operation (a different servers list, the info block,
	// global security). They do not widen the set.
	Unattributed int
}

// Keys returns the changed operations as METHOD path, the form OperationKey writes.
func (c *ChangeSet) Keys() map[string]bool {
	keys := make(map[string]bool, len(c.Changed))
	for _, op := range c.Changed {
		keys[OperationKey(op.Method, op.Path)] = true
	}
	return keys
}

// OperationKey names an operation by its method and the path as the spec writes it.
func OperationKey(method, path string) string { return strings.ToUpper(method) + " " + path }

// ChangedOperations compares two versions of a spec and returns what to check in the revision.
func ChangedOperations(base, revision *Spec) (*ChangeSet, error) {
	result, err := Diff(base, revision)
	if err != nil {
		return nil, err
	}
	inBase := map[string]bool{}
	for _, op := range base.AllOperations() {
		inBase[op.String()] = true
	}
	revOps := revision.AllOperations()
	inRevision := map[string]bool{}
	for _, op := range revOps {
		inRevision[op.String()] = true
	}

	touched := map[string]bool{}
	cs := &ChangeSet{}
	for _, c := range result.Changes {
		if c.Method == "" || c.Path == "" {
			cs.Unattributed++
			continue
		}
		touched[OperationKey(c.Method, c.Path)] = true
	}
	for _, op := range revOps {
		key := op.String()
		switch {
		case !inBase[key]:
			cs.Added++
			cs.Changed = append(cs.Changed, op)
		case touched[key]:
			cs.Modified++
			cs.Changed = append(cs.Changed, op)
		default:
			cs.Unchanged++
		}
	}
	for _, op := range base.AllOperations() {
		if !inRevision[op.String()] {
			cs.Removed = append(cs.Removed, op)
		}
	}
	sort.Slice(cs.Changed, func(i, j int) bool {
		if cs.Changed[i].Path != cs.Changed[j].Path {
			return cs.Changed[i].Path < cs.Changed[j].Path
		}
		return cs.Changed[i].Method < cs.Changed[j].Method
	})
	return cs, nil
}
