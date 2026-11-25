package conformance

import (
	"fmt"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// UnresolvedPlaceholders lists, in order and without repeats, the {{names}} still present in a
// request after variables were filled in: the ones nothing in the environment or the overrides
// provided. Dynamic functions ({{$name}}) are not listed; their failures are reported separately.
func UnresolvedPlaceholders(req *domain.Request) []string {
	return variables.UnresolvedPlaceholders(req)
}

// braced writes names as the placeholders they are: {{a}}, {{b}}.
func braced(names []string) string {
	refs := make([]string, len(names))
	for i, n := range names {
		refs[i] = "{{" + n + "}}"
	}
	return strings.Join(refs, ", ")
}

// blockedReason is the sentence recorded when an operation is not sent.
func blockedReason(missing []string, dynamicErrors []error) string {
	var parts []string
	if len(missing) > 0 {
		parts = append(parts, "no value for "+braced(missing))
	}
	for _, e := range dynamicErrors {
		parts = append(parts, e.Error())
	}
	return fmt.Sprintf("not sent: %s", strings.Join(parts, "; "))
}
