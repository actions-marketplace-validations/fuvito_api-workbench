package main

import (
	"fmt"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// changedOperationScope compares the spec with its version at a git ref and returns the operations
// conform should check, with the description reports show. The spec must be a local file.
func changedOperationScope(src, ref string, spec *openapi.Spec) (map[string]bool, *conformance.Scope, error) {
	data, format, err := gitSpecAt("--changed-against", src, ref)
	if err != nil {
		return nil, nil, err
	}
	base, err := openapi.ParseBytes(data, format)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing the spec at %s: %w", ref, err)
	}
	changes, err := openapi.ChangedOperations(base, spec)
	if err != nil {
		return nil, nil, fmt.Errorf("comparing the spec with %s: %w", ref, err)
	}
	scope := &conformance.Scope{
		ChangedAgainst: ref,
		Checked:        []string{},
		Added:          changes.Added,
		Modified:       changes.Modified,
		Unchanged:      changes.Unchanged,
		Unattributed:   changes.Unattributed,
	}
	for _, op := range changes.Changed {
		scope.Checked = append(scope.Checked, op.String())
	}
	for _, op := range changes.Removed {
		scope.Removed = append(scope.Removed, op.String())
	}
	only := changes.Keys()
	if only == nil {
		only = map[string]bool{}
	}
	return only, scope, nil
}
