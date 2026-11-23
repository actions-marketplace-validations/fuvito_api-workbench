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

// blockedReason is the sentence recorded when an operation is not sent.
func blockedReason(missing []string, dynamicErrors []error) string {
	var parts []string
	if len(missing) > 0 {
		refs := make([]string, len(missing))
		for i, n := range missing {
			refs[i] = "{{" + n + "}}"
		}
		parts = append(parts, "no value for "+strings.Join(refs, ", "))
	}
	for _, e := range dynamicErrors {
		parts = append(parts, e.Error())
	}
	return fmt.Sprintf("not sent: %s", strings.Join(parts, "; "))
}
