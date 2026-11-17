package bundle

import (
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// Runnable is a bundle scenario ready to execute without importing the bundle: requests
// the scenario refers to are inlined and its spec is resolved.
type Runnable struct {
	// File is the bundle name the scenario would be saved under (without extension).
	File     string
	Scenario *domain.Scenario
}

// Materialize returns every scenario of the bundle in bundle order, ready to run.
// A step's requestRef that names a request in the bundle is replaced by that request.
// Any other requestRef is left for the caller to resolve (for example from disk).
// Each scenario's spec is its own specFile, else the bundle's; opts.SpecFile overrides both.
// The bundle itself is not modified.
func Materialize(b *serialization.Bundle, opts Options) []Runnable {
	requests := map[string]*domain.Request{}
	for _, br := range b.Requests {
		if stem, err := fileStem(br.File, ""); err == nil {
			requests[stem] = br.Request
		}
	}

	out := make([]Runnable, 0, len(b.Scenarios))
	for _, bs := range b.Scenarios {
		sc := *bs.Scenario
		sc.SpecFile = effectiveSpec(b, bs.Scenario, opts)
		sc.Steps = append([]domain.ScenarioStep(nil), bs.Scenario.Steps...)
		for i := range sc.Steps {
			step := &sc.Steps[i]
			if step.RequestRef == "" || step.Request != nil {
				continue
			}
			if req, ok := requests[requestKey(step.RequestRef)]; ok {
				copied := *req
				step.Request = &copied
				step.RequestRef = ""
			}
		}
		stem, err := fileStem(bs.File, bs.Scenario.Title)
		if err != nil {
			stem = bs.Scenario.Title
		}
		out = append(out, Runnable{File: stem, Scenario: &sc})
	}
	return out
}

// SelectEnvironment picks the environment to run a bundle with. name matches an
// environment's name or its file name, ignoring case. With no name, the only environment
// is used; a bundle with several needs one chosen. A bundle with none returns nil.
func SelectEnvironment(b *serialization.Bundle, name string) (*domain.Environment, error) {
	if name == "" {
		switch len(b.Environments) {
		case 0:
			return nil, nil
		case 1:
			return b.Environments[0].Environment, nil
		}
		return nil, fmt.Errorf("the bundle has several environments (%s); choose one with --env", envNames(b))
	}
	for _, be := range b.Environments {
		if strings.EqualFold(be.Environment.Name, name) {
			return be.Environment, nil
		}
		if stem, err := fileStem(be.File, be.Environment.Name); err == nil && strings.EqualFold(stem, name) {
			return be.Environment, nil
		}
	}
	if len(b.Environments) == 0 {
		return nil, fmt.Errorf("the bundle has no environments, so --env %q cannot be used (use --env-file)", name)
	}
	return nil, fmt.Errorf("the bundle has no environment %q (available: %s)", name, envNames(b))
}

func envNames(b *serialization.Bundle) string {
	names := make([]string, 0, len(b.Environments))
	for _, be := range b.Environments {
		names = append(names, be.Environment.Name)
	}
	return strings.Join(names, ", ")
}

// FilterRunnables keeps the scenarios whose bundle name or title matches one of the
// given selectors (ignoring case). An empty selector list keeps everything. It reports
// the selectors that matched nothing.
func FilterRunnables(all []Runnable, selectors []string) (kept []Runnable, unmatched []string) {
	if len(selectors) == 0 {
		return all, nil
	}
	matched := make([]bool, len(selectors))
	for _, r := range all {
		for i, sel := range selectors {
			if strings.EqualFold(r.File, sel) || strings.EqualFold(r.Scenario.Title, sel) {
				matched[i] = true
				kept = append(kept, r)
				break
			}
		}
	}
	for i, sel := range selectors {
		if !matched[i] {
			unmatched = append(unmatched, sel)
		}
	}
	return kept, unmatched
}
