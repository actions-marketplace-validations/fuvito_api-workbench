// Package bundle imports a bundle file — one document holding environments, reusable
// requests and scenarios — into a workspace. It parses and validates the bundle, works
// out which files it would create (resolving name conflicts), and writes them atomically.
//
// A bundle is the single-file format an AI assistant is asked to produce. The same code
// serves the desktop import dialog and, later, the CLI.
package bundle

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// Severity classifies a validation finding. Errors block the import; warnings do not.
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
)

// Issue is one validation finding. Where locates it, e.g. `scenarios[1] "Smoke" › step 3 "Login"`.
type Issue struct {
	Severity Severity
	Where    string
	Message  string
}

func (i Issue) String() string {
	if i.Where == "" {
		return i.Message
	}
	return i.Where + ": " + i.Message
}

// Options adjust how a bundle is imported.
type Options struct {
	// SpecFile, when set, replaces the spec (path or URL) of every scenario — including
	// the bundle's own default. Use it to point imported scenarios at a local spec file.
	SpecFile string
}

// ParseText parses pasted or file text. It tolerates what chat assistants add around the
// content: a byte-order mark and a surrounding ``` code fence. JSON is detected by a
// leading brace; everything else is read as YAML.
func ParseText(text string) (*serialization.Bundle, error) {
	text = strings.TrimPrefix(text, string(rune(0xFEFF))) // UTF-8 byte-order mark
	text = strings.TrimSpace(strings.ReplaceAll(text, "\r\n", "\n"))
	text = stripFence(text)
	if text == "" {
		return nil, fmt.Errorf("bundle is empty")
	}
	format := "yaml"
	if strings.HasPrefix(text, "{") {
		format = "json"
	}
	return serialization.ParseBundle([]byte(text), format)
}

func stripFence(text string) string {
	if !strings.HasPrefix(text, "```") {
		return text
	}
	nl := strings.Index(text, "\n")
	if nl < 0 {
		return text
	}
	body := text[nl+1:]
	if end := strings.LastIndex(body, "```"); end >= 0 {
		body = body[:end]
	}
	return strings.TrimSpace(body)
}

var (
	slugRe    = regexp.MustCompile(`[^a-z0-9]+`)
	segmentRe = regexp.MustCompile(`[^A-Za-z0-9._-]+`)
	varRe     = regexp.MustCompile(`\{\{\s*([^}]+?)\s*\}\}`)
)

// fileStem turns a bundle "file" key (or a fallback title) into a safe relative path
// without extension. Folder separators are allowed; ".." and empty segments are not.
func fileStem(key, fallback string) (string, error) {
	raw := strings.TrimSpace(key)
	if raw == "" {
		slug := strings.Trim(slugRe.ReplaceAllString(strings.ToLower(fallback), "-"), "-")
		if slug == "" {
			return "", fmt.Errorf("a file name is required (set \"file\")")
		}
		return slug, nil
	}
	raw = strings.ReplaceAll(raw, "\\", "/")
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		raw = strings.TrimSuffix(raw, ext)
	}
	var parts []string
	for _, seg := range strings.Split(raw, "/") {
		seg = strings.TrimSpace(seg)
		if seg == "" || seg == "." {
			continue
		}
		if seg == ".." {
			return "", fmt.Errorf("file %q must not contain \"..\"", key)
		}
		seg = strings.Trim(segmentRe.ReplaceAllString(strings.ReplaceAll(seg, " ", "-"), ""), ".")
		if seg == "" {
			return "", fmt.Errorf("file %q has no usable name", key)
		}
		parts = append(parts, seg)
	}
	if len(parts) == 0 {
		return "", fmt.Errorf("file %q has no usable name", key)
	}
	return strings.Join(parts, "/"), nil
}

// requestKey normalises a requestRef so "check-health", "check-health.yaml" and
// "requests/check-health.yaml" all identify the same bundle request.
func requestKey(ref string) string {
	ref = strings.ReplaceAll(strings.TrimSpace(ref), "\\", "/")
	ref = strings.TrimPrefix(ref, "./")
	ref = strings.TrimPrefix(ref, "requests/")
	for _, ext := range []string{".yaml", ".yml", ".json"} {
		ref = strings.TrimSuffix(ref, ext)
	}
	return ref
}

func looksLikePath(ref string) bool {
	return strings.ContainsAny(ref, "/\\") || strings.HasSuffix(ref, ".yaml") || strings.HasSuffix(ref, ".yml") || strings.HasSuffix(ref, ".json")
}

// effectiveSpec is the spec a scenario will be bound to after import.
func effectiveSpec(b *serialization.Bundle, sc *domain.Scenario, opts Options) string {
	if opts.SpecFile != "" {
		return opts.SpecFile
	}
	if sc.SpecFile != "" {
		return sc.SpecFile
	}
	return b.SpecFile
}

// Validate checks a bundle without touching the filesystem. Errors block an import.
func Validate(b *serialization.Bundle, opts Options) []Issue {
	var issues []Issue
	add := func(sev Severity, where, msg string) { issues = append(issues, Issue{sev, where, msg}) }
	errf := func(where, format string, a ...any) { add(SeverityError, where, fmt.Sprintf(format, a...)) }
	warnf := func(where, format string, a ...any) { add(SeverityWarning, where, fmt.Sprintf(format, a...)) }

	if b.SchemaVersion != "1" {
		errf("", `schemaVersion is required and must be "1"`)
	}
	if len(b.Environments)+len(b.Requests)+len(b.Scenarios) == 0 {
		errf("", "the bundle contains no environments, requests or scenarios")
		return issues
	}

	definedVars := map[string]bool{}

	// Environments
	envFiles := map[string]string{}
	envNames := map[string]int{}
	for i, be := range b.Environments {
		where := fmt.Sprintf("environments[%d]", i)
		env := be.Environment
		if env.Name == "" {
			errf(where, "name is required")
		} else {
			where = fmt.Sprintf("environments[%d] %q", i, env.Name)
			if prev, dup := envNames[strings.ToLower(env.Name)]; dup {
				errf(where, "duplicate environment name (also environments[%d])", prev)
			}
			envNames[strings.ToLower(env.Name)] = i
		}
		if stem, err := fileStem(be.File, env.Name); err != nil {
			errf(where, "%v", err)
		} else if prev, dup := envFiles[stem]; dup {
			errf(where, "file %q is already used by %s", stem, prev)
		} else {
			envFiles[stem] = where
		}
		seen := map[string]bool{}
		for _, v := range env.Variables {
			switch {
			case strings.TrimSpace(v.Name) == "":
				errf(where, "a variable has no name")
			case seen[v.Name]:
				errf(where, "variable %q is defined twice", v.Name)
			default:
				seen[v.Name] = true
				definedVars[v.Name] = true
			}
		}
	}

	// Requests
	reqByKey := map[string]*domain.Request{}
	reqFiles := map[string]string{}
	for i, br := range b.Requests {
		where := fmt.Sprintf("requests[%d]", i)
		if strings.TrimSpace(br.File) == "" {
			errf(where, `"file" is required so scenarios can refer to it with requestRef`)
			continue
		}
		where = fmt.Sprintf("requests[%d] %q", i, br.File)
		stem, err := fileStem(br.File, "")
		if err != nil {
			errf(where, "%v", err)
			continue
		}
		if prev, dup := reqFiles[stem]; dup {
			errf(where, "file %q is already used by %s", stem, prev)
			continue
		}
		reqFiles[stem] = where
		reqByKey[stem] = br.Request
		issues = append(issues, validateRequest(where, br.Request)...)
	}

	// Scenarios
	scnFiles := map[string]string{}
	for i, bs := range b.Scenarios {
		sc := bs.Scenario
		where := fmt.Sprintf("scenarios[%d]", i)
		if sc.Title != "" {
			where = fmt.Sprintf("scenarios[%d] %q", i, sc.Title)
		}
		if stem, err := fileStem(bs.File, sc.Title); err != nil {
			errf(where, "%v", err)
		} else if prev, dup := scnFiles[stem]; dup {
			errf(where, "file %q is already used by %s (give one of them a different \"file\")", stem, prev)
		} else {
			scnFiles[stem] = where
		}

		for _, iss := range scenario.Validate(sc, "") {
			if iss.Step == 0 {
				errf(where, "%s", iss.Message)
				continue
			}
			errf(stepWhere(where, iss.Step, iss.Name), "%s", iss.Message)
		}

		spec := effectiveSpec(b, sc, opts)
		extracted := map[string]bool{}
		for si, step := range sc.Steps {
			sw := stepWhere(where, si+1, step.Label)

			req := step.Request
			if step.RequestRef != "" {
				if r, ok := reqByKey[requestKey(step.RequestRef)]; ok {
					req = r
				} else if looksLikePath(step.RequestRef) {
					warnf(sw, "requestRef %q is not part of this bundle, so it must already exist in the workspace", step.RequestRef)
				} else {
					errf(sw, "requestRef %q does not match any request in this bundle (known: %s)", step.RequestRef, knownKeys(reqByKey))
				}
			}
			if step.Request != nil {
				issues = append(issues, validateRequest(sw, step.Request)...)
			}

			if len(b.Environments) > 0 && req != nil {
				for _, name := range variablesIn(req) {
					if !definedVars[name] && !extracted[name] {
						warnf(sw, "{{%s}} is not defined in any environment of this bundle and is not extracted by an earlier step", name)
					}
				}
			}

			if step.Retry != nil || step.TimeoutMs > 0 {
				warnf(sw, "retry and timeoutMs are accepted but not applied when a scenario runs yet")
			}
			for _, a := range step.Assertions {
				if a.Type == domain.AssertionSchema && spec == "" && looksLikeOperationID(a.Schema) {
					warnf(sw, "schema assertion %q needs a spec: set specFile on the bundle or scenario", a.Schema)
				}
			}
			for _, e := range step.Extractions {
				extracted[e.Name] = true
			}
		}
	}

	return issues
}

func knownKeys(m map[string]*domain.Request) string {
	if len(m) == 0 {
		return "none"
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return strings.Join(keys, ", ")
}

func stepWhere(scenarioWhere string, step int, label string) string {
	if label != "" {
		return fmt.Sprintf("%s › step %d %q", scenarioWhere, step, label)
	}
	return fmt.Sprintf("%s › step %d", scenarioWhere, step)
}

func validateRequest(where string, r *domain.Request) []Issue {
	var out []Issue
	add := func(format string, a ...any) {
		out = append(out, Issue{SeverityError, where, fmt.Sprintf(format, a...)})
	}
	if err := r.Validate(); err != nil {
		add("%v", err)
	}
	if r.Body != nil {
		switch r.Body.Type {
		case domain.BodyNone, domain.BodyJSON, domain.BodyRaw, domain.BodyFormURLEncoded, domain.BodyMultipart:
		default:
			add("unknown body type %q (want none, json, raw, form-urlencoded or multipart)", r.Body.Type)
		}
	}
	if r.Auth != nil {
		switch r.Auth.Type {
		case domain.AuthNone, domain.AuthBearer, domain.AuthBasic, domain.AuthAPIKey:
		default:
			add("unknown auth type %q (want none, bearer, basic or apikey)", r.Auth.Type)
		}
		if r.Auth.Type == domain.AuthAPIKey && r.Auth.APIKey != nil {
			if loc := r.Auth.APIKey.Location; loc != domain.APIKeyInHeader && loc != domain.APIKeyInQuery {
				add("apikey auth location must be header or query, got %q", loc)
			}
		}
	}
	return out
}

// variablesIn lists the distinct {{variable}} names used by a request.
func variablesIn(r *domain.Request) []string {
	texts := []string{r.URL}
	for _, h := range r.Headers {
		texts = append(texts, h.Value)
	}
	for _, q := range r.QueryParams {
		texts = append(texts, q.Value)
	}
	if r.Body != nil {
		texts = append(texts, r.Body.Content)
	}
	if a := r.Auth; a != nil {
		if a.Bearer != nil {
			texts = append(texts, a.Bearer.Token)
		}
		if a.Basic != nil {
			texts = append(texts, a.Basic.Username, a.Basic.Password)
		}
		if a.APIKey != nil {
			texts = append(texts, a.APIKey.Value)
		}
	}
	seen := map[string]bool{}
	var out []string
	for _, t := range texts {
		for _, m := range varRe.FindAllStringSubmatch(t, -1) {
			if !seen[m[1]] {
				seen[m[1]] = true
				out = append(out, m[1])
			}
		}
	}
	return out
}

// looksLikeOperationID mirrors the assertion engine: a schema value without braces,
// colons or newlines is an operationId rather than an inline schema.
func looksLikeOperationID(s string) bool {
	return s != "" && !strings.ContainsAny(s, "{\n\r:")
}

// ---- planning and writing ----

// Strategy decides what happens when a file the bundle would create already exists.
type Strategy string

const (
	StrategySkip      Strategy = "skip"
	StrategyOverwrite Strategy = "overwrite"
	StrategyRename    Strategy = "rename"
)

// Kind is the type of file an Entry produces.
type Kind string

const (
	KindEnvironment Kind = "environment"
	KindRequest     Kind = "request"
	KindScenario    Kind = "scenario"
)

// Action says what importing an Entry will do.
type Action string

const (
	ActionCreate    Action = "create"
	ActionOverwrite Action = "overwrite"
	ActionRename    Action = "rename"
	ActionSkip      Action = "skip"
)

// Entry is one file a bundle import produces.
type Entry struct {
	Kind Kind
	// Name is the display name: the environment name, request title or scenario title.
	Name string
	// RelPath is where the bundle wants the file, relative to the workspace, "/"-separated.
	RelPath string
	// FinalPath is where it will be written after conflict handling (equals RelPath unless renamed).
	FinalPath string
	Action    Action
	Content   []byte
}

// Plan is the result of preparing a bundle: the validation findings and the files to write.
type Plan struct {
	Title   string
	Issues  []Issue
	Entries []Entry
}

// HasErrors reports whether any validation error blocks the import.
func (p *Plan) HasErrors() bool {
	for _, i := range p.Issues {
		if i.Severity == SeverityError {
			return true
		}
	}
	return false
}

// Prepare validates the bundle and works out the files it would write under workspaceDir.
// It never writes anything. Entries are only produced when validation finds no errors.
func Prepare(b *serialization.Bundle, workspaceDir string, opts Options, strategy Strategy) (*Plan, error) {
	if strategy == "" {
		strategy = StrategyRename
	}
	switch strategy {
	case StrategySkip, StrategyOverwrite, StrategyRename:
	default:
		return nil, fmt.Errorf("unknown conflict strategy %q (want skip, overwrite or rename)", strategy)
	}

	plan := &Plan{Title: b.Title, Issues: Validate(b, opts)}
	if plan.HasErrors() {
		return plan, nil
	}

	taken := map[string]bool{}
	finalize := func(rel string) (string, Action) {
		abs := filepath.Join(workspaceDir, filepath.FromSlash(rel))
		_, err := os.Stat(abs)
		exists := err == nil
		if !exists && !taken[rel] {
			taken[rel] = true
			return rel, ActionCreate
		}
		switch strategy {
		case StrategySkip:
			return rel, ActionSkip
		case StrategyOverwrite:
			taken[rel] = true
			return rel, ActionOverwrite
		}
		ext := path.Ext(rel)
		stem := strings.TrimSuffix(rel, ext)
		for n := 2; ; n++ {
			candidate := fmt.Sprintf("%s-%d%s", stem, n, ext)
			if _, err := os.Stat(filepath.Join(workspaceDir, filepath.FromSlash(candidate))); err != nil && !taken[candidate] {
				taken[candidate] = true
				return candidate, ActionRename
			}
		}
	}

	// Requests first: scenarios need their final locations to build requestRefs.
	requestFinal := map[string]string{}
	for _, br := range b.Requests {
		stem, _ := fileStem(br.File, "")
		rel := "requests/" + stem + ".yaml"
		data, err := serialization.WriteRequest(br.Request, "yaml")
		if err != nil {
			return nil, fmt.Errorf("serialising request %q: %w", br.File, err)
		}
		final, action := finalize(rel)
		requestFinal[stem] = final
		name := br.Request.Title
		if name == "" {
			name = br.File
		}
		plan.Entries = append(plan.Entries, Entry{Kind: KindRequest, Name: name, RelPath: rel, FinalPath: final, Action: action, Content: data})
	}

	for _, be := range b.Environments {
		stem, _ := fileStem(be.File, be.Environment.Name)
		rel := "environments/" + stem + ".yaml"
		data, err := serialization.WriteEnvironment(be.Environment, "yaml")
		if err != nil {
			return nil, fmt.Errorf("serialising environment %q: %w", be.Environment.Name, err)
		}
		final, action := finalize(rel)
		plan.Entries = append(plan.Entries, Entry{Kind: KindEnvironment, Name: be.Environment.Name, RelPath: rel, FinalPath: final, Action: action, Content: data})
	}

	for _, bs := range b.Scenarios {
		stem, _ := fileStem(bs.File, bs.Scenario.Title)
		rel := "scenarios/" + stem + ".yaml"
		final, action := finalize(rel)

		sc := *bs.Scenario
		sc.SpecFile = effectiveSpec(b, bs.Scenario, opts)
		sc.Steps = append([]domain.ScenarioStep(nil), bs.Scenario.Steps...)
		for i := range sc.Steps {
			ref := sc.Steps[i].RequestRef
			if ref == "" {
				continue
			}
			if target, ok := requestFinal[requestKey(ref)]; ok {
				sc.Steps[i].RequestRef = relativeRef(final, target)
			}
		}
		data, err := serialization.WriteScenario(&sc, "yaml")
		if err != nil {
			return nil, fmt.Errorf("serialising scenario %q: %w", sc.Title, err)
		}
		plan.Entries = append(plan.Entries, Entry{Kind: KindScenario, Name: sc.Title, RelPath: rel, FinalPath: final, Action: action, Content: data})
	}
	return plan, nil
}

// relativeRef returns the path to target as seen from the directory of from, both
// workspace-relative with "/" separators.
func relativeRef(from, target string) string {
	rel, err := filepath.Rel(filepath.FromSlash(path.Dir(from)), filepath.FromSlash(target))
	if err != nil {
		return target
	}
	return filepath.ToSlash(rel)
}

// Result summarises a completed import.
type Result struct {
	Created     int
	Overwritten int
	Renamed     int
	Skipped     int
	// Written lists the workspace-relative paths that were written.
	Written []string
}

// Write saves the plan's files under workspaceDir. It refuses to run if the plan has
// validation errors, and each file is written atomically (temp file, then rename).
func (p *Plan) Write(workspaceDir string) (*Result, error) {
	if p.HasErrors() {
		return nil, fmt.Errorf("the bundle has validation errors and was not imported")
	}
	res := &Result{}
	for _, e := range p.Entries {
		if e.Action == ActionSkip {
			res.Skipped++
			continue
		}
		abs := filepath.Join(workspaceDir, filepath.FromSlash(e.FinalPath))
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			return res, fmt.Errorf("creating folder for %s: %w", e.FinalPath, err)
		}
		if err := atomicWrite(abs, e.Content); err != nil {
			return res, fmt.Errorf("writing %s: %w", e.FinalPath, err)
		}
		res.Written = append(res.Written, e.FinalPath)
		switch e.Action {
		case ActionOverwrite:
			res.Overwritten++
		case ActionRename:
			res.Renamed++
		default:
			res.Created++
		}
	}
	return res, nil
}

func atomicWrite(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
