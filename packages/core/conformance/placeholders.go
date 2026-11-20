package conformance

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
)

var placeholderRef = regexp.MustCompile(`\{\{\s*([^{}\s][^{}]*?)\s*\}\}`)

// UnresolvedPlaceholders lists, in order and without repeats, the {{names}} still present in a
// request after variables were filled in: the ones nothing in the environment or the overrides
// provided. Dynamic functions ({{$name}}) are not listed; their failures are reported separately.
func UnresolvedPlaceholders(req *domain.Request) []string {
	var texts []string
	texts = append(texts, req.URL)
	for _, q := range req.QueryParams {
		texts = append(texts, q.Name, q.Value)
	}
	for _, h := range req.Headers {
		texts = append(texts, h.Name, h.Value)
	}
	if req.Body != nil {
		texts = append(texts, req.Body.Content)
	}
	if a := req.Auth; a != nil {
		if a.Bearer != nil {
			texts = append(texts, a.Bearer.Token)
		}
		if a.Basic != nil {
			texts = append(texts, a.Basic.Username, a.Basic.Password)
		}
		if a.APIKey != nil {
			texts = append(texts, a.APIKey.Key, a.APIKey.Value)
		}
	}
	var names []string
	seen := map[string]bool{}
	for _, text := range texts {
		for _, m := range placeholderRef.FindAllStringSubmatch(text, -1) {
			name := strings.TrimSpace(m[1])
			if strings.HasPrefix(name, "$") || seen[name] {
				continue
			}
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
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
