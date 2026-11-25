package main

import (
	"fmt"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

// lintSpecFile checks an OpenAPI document itself and returns its errors and warnings. A spec that
// cannot be read or parsed is the error.
func lintSpecFile(src string) (errs, warns []scenario.Issue, err error) {
	data, format, err := fetchSpec(src)
	if err != nil {
		return nil, nil, err
	}
	spec, err := openapi.ParseBytes(data, format)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing spec: %w", err)
	}
	for _, i := range spec.Lint() {
		issue := scenario.Issue{Message: fmt.Sprintf("[%s] %s Why: %s Fix: %s", i.Check, i.What, i.Why, i.Fix)}
		if i.Severity == openapi.SpecError {
			errs = append(errs, issue)
		} else {
			warns = append(warns, issue)
		}
	}
	return errs, warns, nil
}
