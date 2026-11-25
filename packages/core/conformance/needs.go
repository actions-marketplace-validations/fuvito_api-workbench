package conformance

import (
	"slices"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// OperationNeeds says which values the request generated for one operation takes, and which of
// them a run would not have.
type OperationNeeds struct {
	OperationID string
	Method      string
	// Path and SpecPath are as in OperationResult.
	Path     string
	SpecPath string
	// Skipped: a filter or read-only leaves the operation out of this run.
	Skipped bool
	// Needs names every {{placeholder}} of the generated request, whatever the environment has.
	Needs []string
	// Credentials names the ones among Needs that authenticate the request.
	Credentials []string
	// Missing and Blocked are what a run would report: it would not send this operation.
	Missing []string
	Blocked string
}

// NeedsList is what a run would need, worked out without sending anything.
type NeedsList struct {
	SpecTitle   string
	SpecVersion string
	Operations  []OperationNeeds
	// FilterMatched counts the operations the tags and paths selected.
	FilterMatched int
}

// ListNeeds works out, for every operation, the values its request needs and which are missing.
// It uses the request generation, selection and substitution of Run, so the two agree on what
// would not be sent. No request is made.
func ListNeeds(spec *openapi.Spec, baseURL string, env *domain.Environment, opts Options) (*NeedsList, error) {
	title, version := spec.Info()
	if baseURL == "" {
		baseURL = spec.FirstServerURL()
	}
	reqs, err := openapi.GenerateRequests(spec, baseURL)
	if err != nil {
		return nil, err
	}
	list := &NeedsList{SpecTitle: title, SpecVersion: version}
	resolver := variables.NewResolver(env)
	for _, gr := range reqs {
		or := operationFor(gr, baseURL)
		selected, send := opts.selects(or, gr)
		if selected {
			list.FilterMatched++
		}
		n := OperationNeeds{
			OperationID: or.OperationID, Method: or.Method, Path: or.Path, SpecPath: or.SpecPath, Skipped: !send,
			Needs: UnresolvedPlaceholders(gr.Request), Credentials: credentialNames(gr.Request),
		}
		if send {
			if _, missing, blocked := prepare(resolver, gr); blocked != "" {
				n.Missing, n.Blocked = missing, blocked
			}
		}
		list.Operations = append(list.Operations, n)
	}
	return list, nil
}

// Names returns every value the operations of this run need, and every one that is missing, each
// once and in the order they first appear.
func (l *NeedsList) Names() (needs, missing []string) {
	seenNeed, seenMissing := map[string]bool{}, map[string]bool{}
	add := func(out *[]string, seen map[string]bool, names []string) {
		for _, n := range names {
			if !seen[n] {
				seen[n] = true
				*out = append(*out, n)
			}
		}
	}
	for _, op := range l.Operations {
		if op.Skipped {
			continue
		}
		add(&needs, seenNeed, op.Needs)
		add(&missing, seenMissing, op.Missing)
	}
	return needs, missing
}

// operationFor is the identity of a generated request, as a result and a listing show it.
func operationFor(gr openapi.GeneratedRequest, baseURL string) OperationResult {
	return OperationResult{
		OperationID: gr.OperationID,
		Summary:     gr.Summary,
		Tags:        gr.Tags,
		Method:      string(gr.Request.Method),
		Path:        pathOf(gr.Request.URL, baseURL),
		SpecPath:    gr.Path,
	}
}

// selects says whether the tag, path and changed-operation filters pick an operation, and whether
// a run would then send it.
func (o Options) selects(or OperationResult, gr openapi.GeneratedRequest) (selected, send bool) {
	selected = matchesTags(gr.Tags, o.Tags) && matchesPathPrefix(or.SpecPath, or.Path, o.Paths) && matchesOnly(o.OnlyOperations, or.Method, gr.Path)
	return selected, selected && (!o.ReadOnly || IsReadOnlyMethod(or.Method))
}

// prepare puts the values into a generated request. A non-empty blocked says why it must not be
// sent: a placeholder nothing provides, an empty value where one is needed, or a dynamic value that
// failed.
func prepare(resolver *variables.Resolver, gr openapi.GeneratedRequest) (resolved *domain.Request, missing []string, blocked string) {
	resolved = resolver.InterpolateRequest(gr.Request)
	missing, dynamicErrors := UnresolvedPlaceholders(resolved), resolver.TakeErrors()
	empty := emptyWhereNeeded(resolver, gr.Request)
	for _, name := range empty {
		if !slices.Contains(missing, name) {
			missing = append(missing, name)
		}
	}
	if len(missing) == 0 && len(dynamicErrors) == 0 {
		return resolved, nil, ""
	}
	blocked = blockedReason(missing, dynamicErrors)
	if len(empty) > 0 {
		blocked += " (defined but empty: " + braced(empty) + "; a path value or a credential cannot be empty)"
	}
	return resolved, missing, blocked
}

// emptyWhereNeeded names the values that are defined but empty where a generated request cannot
// mean it: in the path (/items/{{id}} would become /items/) and in a credential (the request would
// arrive unauthenticated). The answer, a declared 400, 401 or 404, could then pass for an
// operation that was never exercised. Anywhere else (query, header, cookie or body values) an
// empty text can be what is meant, and so can an empty basic password: some APIs take a key as the
// user name and nothing as the password.
func emptyWhereNeeded(resolver *variables.Resolver, template *domain.Request) []string {
	path, _, _ := strings.Cut(template.URL, "?")
	must := &domain.Request{URL: path}
	if a := template.Auth; a != nil {
		auth := &domain.Auth{Type: a.Type, Bearer: a.Bearer, APIKey: a.APIKey}
		if a.Basic != nil {
			auth.Basic = &domain.BasicAuth{Username: a.Basic.Username}
		}
		must.Auth = auth
	}
	names := UnresolvedPlaceholders(must)
	// An API key in a cookie travels in the Cookie header, beside ordinary cookie parameters.
	for _, h := range template.Headers {
		if strings.EqualFold(h.Name, "Cookie") && strings.Contains(h.Value, "{{"+openapi.APIKeyPlaceholder+"}}") {
			names = append(names, openapi.APIKeyPlaceholder)
		}
	}
	var empty []string
	for _, name := range names {
		if value, defined := resolver.Get(name); defined && value == "" && !slices.Contains(empty, name) {
			empty = append(empty, name)
		}
	}
	return empty
}

// credentialNames lists the placeholders in the parts of a request that authenticate it: the auth
// block, and the Cookie header an API key in a cookie goes into.
func credentialNames(req *domain.Request) []string {
	only := &domain.Request{Auth: req.Auth}
	for _, h := range req.Headers {
		if strings.EqualFold(h.Name, "Cookie") || strings.EqualFold(h.Name, "Authorization") {
			only.Headers = append(only.Headers, h)
		}
	}
	return UnresolvedPlaceholders(only)
}
