package scenario

import (
	"os"
	"path/filepath"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// ResolveRequestRefs loads the request file behind every step that has a requestRef and
// no inline request, so the scenario can run. Each ref is looked up relative to every
// directory in dirs, in order; an absolute ref is used as it is. The step's RequestRef is
// left in place. It returns the refs that could not be loaded.
func ResolveRequestRefs(sc *domain.Scenario, dirs ...string) (unresolved []string) {
	for i := range sc.Steps {
		step := &sc.Steps[i]
		if step.Request != nil || step.RequestRef == "" {
			continue
		}
		candidates := make([]string, 0, len(dirs))
		if filepath.IsAbs(step.RequestRef) {
			candidates = append(candidates, step.RequestRef)
		} else {
			for _, d := range dirs {
				if d != "" {
					candidates = append(candidates, filepath.Join(d, step.RequestRef))
				}
			}
		}
		resolved := false
		for _, path := range candidates {
			data, err := os.ReadFile(path)
			if err != nil {
				continue
			}
			req, err := serialization.ParseRequestFile(data, filepath.Base(path))
			if err != nil {
				continue
			}
			step.Request = req
			resolved = true
			break
		}
		if !resolved {
			unresolved = append(unresolved, step.RequestRef)
		}
	}
	return unresolved
}
