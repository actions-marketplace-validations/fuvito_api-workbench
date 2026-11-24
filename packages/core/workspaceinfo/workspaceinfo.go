// Package workspaceinfo describes the state of a workspace folder: what it contains, which
// environment is active, which spec files its scenarios use, and what needs attention. It reads
// files with the same parsers the runners use and never writes anything.
package workspaceinfo

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/fuvito/api-workbench/packages/core/variables"
)

// Severity of an attention item.
const (
	SeverityWarning = "warning" // a run will fail or mislead
	SeverityInfo    = "info"    // worth knowing, nothing breaks
)

// Kinds of attention item (stable identifiers, used by the page and by tests).
const (
	ItemNoEnvironment   = "no-environment"
	ItemUndefinedValue  = "undefined-placeholder"
	ItemUnsetEnvVar     = "unset-env-var"
	ItemSpecMissing     = "spec-missing"
	ItemSpecOutside     = "spec-outside"
	ItemScenarioInvalid = "scenario-invalid"
	ItemRequestRef      = "request-ref-missing"
	ItemScenarioWeak    = "scenario-weak"

	ItemProjectFileInvalid = "project-file-invalid"
	ItemProjectFilePath    = "project-file-path"
)

// Spec file states.
const (
	SpecFound   = "found"
	SpecMissing = "missing"
	SpecOutside = "outside" // resolves outside the workspace; not read
	SpecURL     = "url"     // not fetched
)

// Target says where the fix is, so the page can offer a button that goes there.
type Target struct {
	Kind string `json:"kind"` // scenario, environment, spec, request
	Path string `json:"path"`
}

// Attention is one thing that needs the user's attention, written as what, where, why and fix.
type Attention struct {
	ID       string `json:"id"`
	Severity string `json:"severity"`
	What     string `json:"what"`
	Where    string `json:"where"`
	Why      string `json:"why"`
	Fix      string `json:"fix"`
	Target   Target `json:"target"`
}

// EnvVar describes an environment variable without ever carrying its value.
type EnvVar struct {
	Name   string `json:"name"`
	Secret bool   `json:"secret"`
	// Set is false when the value is empty or refers to a process variable (${NAME}) that is not set.
	Set bool `json:"set"`
	// UnsetRef names the unset process variable, when that is why Set is false.
	UnsetRef string `json:"unsetRef,omitempty"`
}

// Spec is a spec named by one or more scenarios.
type Spec struct {
	Ref string `json:"ref"` // the first spelling a scenario used
	// Refs holds every spelling scenarios used for this one spec (a path relative to different
	// folders, an absolute path), because stored coverage is keyed by the spelling.
	Refs      []string `json:"refs"`
	Path      string   `json:"path"` // resolved file path (empty for a URL)
	State     string   `json:"state"`
	Scenarios []string `json:"scenarios"` // scenario file paths that use it
}

// Report is the state of one workspace.
type Report struct {
	Root            string      `json:"root"`
	Requests        int         `json:"requests"`
	Scenarios       int         `json:"scenarios"`
	Environments    int         `json:"environments"`
	EnvironmentName string      `json:"environmentName,omitempty"`
	EnvironmentPath string      `json:"environmentPath,omitempty"`
	EnvVars         []EnvVar    `json:"envVars"`
	Specs           []Spec      `json:"specs"`
	Attention       []Attention `json:"attention"`
	ProjectFile     ProjectFile `json:"projectFile"`
}

// ProjectFile is what the workspace folder says about its clrsig.yaml: whether it has one, whether
// it can be used, and which settings it gives.
type ProjectFile struct {
	Found bool `json:"found"`
	// Path is the file relative to the workspace, with forward slashes.
	Path string `json:"path,omitempty"`
	// Error is why the file cannot be used (empty when it is fine).
	Error string `json:"error,omitempty"`
	// Keys are the settings the file gives, as written in it.
	Keys []string `json:"keys"`
}

var (
	placeholderRe = regexp.MustCompile(`\{\{\s*([A-Za-z_][A-Za-z0-9_.-]*)\s*\}\}`)
)

// Inspect reports on the workspace at root. env and envPath are the active environment (nil and
// "" when none is selected).
func Inspect(root string, env *domain.Environment, envPath string) Report {
	r := Report{Root: root, EnvVars: []EnvVar{}, Specs: []Spec{}, Attention: []Attention{}, ProjectFile: ProjectFile{Keys: []string{}}}
	r.Requests = countDataFiles(filepath.Join(root, "requests"))
	r.Environments = countDataFilesFlat(filepath.Join(root, "environments"))

	if env != nil {
		r.EnvironmentName = env.Name
		r.EnvironmentPath = envPath
		r.EnvVars = describeEnv(env)
		for _, ref := range variables.MissingEnv(env) {
			r.Attention = append(r.Attention, Attention{
				ID: ItemUnsetEnvVar, Severity: SeverityWarning,
				What:   fmt.Sprintf("The process variable %s is not set.", ref.Name),
				Where:  fmt.Sprintf("environment %q, variable %q refers to it as ${%s}", envLabel(env, envPath), ref.Var, ref.Name),
				Why:    "The value stays as written in every request, so requests fail for the wrong reason.",
				Fix:    fmt.Sprintf("Set %s in the environment that starts clrsig, or give %q a value in the environment file.", ref.Name, ref.Var),
				Target: Target{Kind: "environment", Path: envPath},
			})
		}
	}

	specs := map[string]*Spec{}
	var specOrder []string
	unresolved := map[string][]string{} // placeholder name -> scenarios that use it and nothing in them defines
	scenDir := filepath.Join(root, "scenarios")
	_ = filepath.WalkDir(scenDir, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !isDataFile(d.Name()) {
			return nil
		}
		r.Scenarios++
		inspectScenario(root, p, &r, specs, &specOrder, unresolved)
		return nil
	})
	for _, k := range specOrder {
		sp := specs[k]
		sort.Strings(sp.Scenarios)
		r.Specs = append(r.Specs, *sp)
		switch sp.State {
		case SpecMissing:
			r.Attention = append(r.Attention, Attention{
				ID: ItemSpecMissing, Severity: SeverityWarning,
				What:   fmt.Sprintf("The spec file %q does not exist.", sp.Ref),
				Where:  fmt.Sprintf("used by %s", scenarioList(root, sp.Scenarios)),
				Why:    "Responses are not checked against a spec, and conformance and coverage are not recorded.",
				Fix:    "Choose the spec file in the scenario's Setup row, or put the file back at " + sp.Path + ".",
				Target: Target{Kind: "scenario", Path: sp.Scenarios[0]},
			})
		case SpecOutside:
			r.Attention = append(r.Attention, Attention{
				ID: ItemSpecOutside, Severity: SeverityInfo,
				What:   fmt.Sprintf("The spec file %q is outside this workspace.", sp.Ref),
				Where:  fmt.Sprintf("used by %s", scenarioList(root, sp.Scenarios)),
				Why:    "It works on this computer, but the workspace is not self-contained and was not checked here.",
				Fix:    "Copy the spec into the workspace (for example under specs/) and point the scenario at it.",
				Target: Target{Kind: "scenario", Path: sp.Scenarios[0]},
			})
		}
	}

	inspectProjectFile(root, &r)

	names := make([]string, 0, len(unresolved))
	for n := range unresolved {
		names = append(names, n)
	}
	sort.Strings(names)
	switch {
	case env == nil && r.Environments > 0 && len(names) > 0:
		r.Attention = append([]Attention{{
			ID: ItemNoEnvironment, Severity: SeverityWarning,
			What:   "No environment is selected.",
			Where:  fmt.Sprintf("scenarios use %s, which nothing in them defines", nameList(names)),
			Why:    "Those placeholders stay unresolved, so requests go to the wrong address or fail.",
			Fix:    "Choose an environment in the top bar.",
			Target: Target{Kind: "environment", Path: ""},
		}}, r.Attention...)
	case env != nil:
		have := map[string]bool{}
		for _, v := range env.Variables {
			have[v.Name] = true
		}
		var missing []string
		users := map[string]bool{}
		for _, n := range names {
			if have[n] {
				continue
			}
			missing = append(missing, n)
			for _, sc := range unresolved[n] {
				users[sc] = true
			}
		}
		if len(missing) > 0 {
			var files []string
			for sc := range users {
				files = append(files, sc)
			}
			sort.Strings(files)
			r.Attention = append(r.Attention, Attention{
				ID: ItemUndefinedValue, Severity: SeverityWarning,
				What:   fmt.Sprintf("The environment %q does not define %s.", envLabel(env, envPath), nameList(missing)),
				Where:  "used by " + joinMore(files, 3, "scenarios"),
				Why:    "Nothing in those scenarios defines it either, so the placeholder stays as written and the request fails or goes to the wrong place.",
				Fix:    "Add the value to the environment, have an earlier step extract it, or pass it with --var when running from the command line.",
				Target: Target{Kind: "environment", Path: envPath},
			})
		}
	}
	return r
}

func inspectScenario(root, path string, r *Report, specs map[string]*Spec, order *[]string, unresolved map[string][]string) {
	data, err := os.ReadFile(path)
	rel := relTo(root, path)
	if err != nil {
		r.Attention = append(r.Attention, scenarioProblem(path, rel, "cannot be read: "+err.Error()))
		return
	}
	sc, err := serialization.ParseScenarioFile(data, filepath.Base(path))
	if err != nil {
		r.Attention = append(r.Attention, scenarioProblem(path, rel, "cannot be parsed: "+err.Error()))
		return
	}
	for _, iss := range scenario.Validate(sc, "") {
		r.Attention = append(r.Attention, scenarioProblem(path, rel, iss.String()))
	}

	scDir := filepath.Dir(path)
	used := map[string]bool{}
	collect := func(texts []string) {
		for _, t := range texts {
			for _, m := range placeholderRe.FindAllStringSubmatch(t, -1) {
				used[m[1]] = true
			}
		}
	}
	for i := range sc.Steps {
		step := &sc.Steps[i]
		switch {
		case step.Request != nil:
			if !step.Disabled {
				collect(scenario.StepTexts(step))
			}
		case step.RequestRef != "":
			refPath := findRequestRef(root, scDir, step.RequestRef)
			if refPath == "" {
				label := step.Label
				if label == "" {
					label = fmt.Sprintf("step %d", i+1)
				}
				r.Attention = append(r.Attention, Attention{
					ID: ItemRequestRef, Severity: SeverityWarning,
					What:   fmt.Sprintf("The request file %q does not exist.", step.RequestRef),
					Where:  fmt.Sprintf("%s, step %q", rel, label),
					Why:    "The step cannot be sent, so the scenario fails when it reaches it.",
					Fix:    "Choose an existing request in the step, or restore the file under requests/.",
					Target: Target{Kind: "scenario", Path: path},
				})
				continue
			}
			if step.Disabled {
				continue
			}
			if data, err := os.ReadFile(refPath); err == nil {
				if req, err := serialization.ParseRequestFile(data, filepath.Base(refPath)); err == nil {
					collect(scenario.StepTexts(&domain.ScenarioStep{Request: req}))
				}
			}
		}
	}

	if sc.SpecFile != "" {
		addSpec(root, path, sc.SpecFile, specs, order)
	}

	// valid but weak: what clrsig validate --strict reports. Names nothing defines are left out
	// here, because the environment check below reports those with the environment in view.
	scenario.ResolveRequestRefs(sc, scDir, filepath.Join(root, "requests"), filepath.Join(scDir, "..", "requests"))
	if weak := scenario.Lint(sc, scenario.LintOptions{}); len(weak) > 0 {
		r.Attention = append(r.Attention, weakScenario(path, rel, weak))
	}

	defined := map[string]bool{}
	for _, v := range sc.Variables {
		defined[v.Name] = true
		collect([]string{v.Value})
	}
	for _, step := range sc.Steps {
		for _, e := range step.Extractions {
			defined[e.Name] = true
		}
	}
	for name := range used {
		if !defined[name] {
			unresolved[name] = append(unresolved[name], rel)
		}
	}
}

func weakScenario(path, rel string, weak []scenario.Issue) Attention {
	lines := make([]string, len(weak))
	for i, w := range weak {
		lines[i] = w.String()
	}
	what := fmt.Sprintf("The scenario passes its checks but is weak: %d thing%s to improve.", len(weak), map[bool]string{true: "", false: "s"}[len(weak) == 1])
	return Attention{
		ID: ItemScenarioWeak, Severity: SeverityInfo,
		What:   what,
		Where:  rel,
		Why:    strings.Join(lines, "\n"),
		Fix:    "Add an assertion (a status check is enough), use or remove the extracted value, then run clrsig validate --strict to check it again.",
		Target: Target{Kind: "scenario", Path: path},
	}
}

func scenarioProblem(path, rel, msg string) Attention {
	return Attention{
		ID: ItemScenarioInvalid, Severity: SeverityWarning,
		What:   "The scenario has a problem: " + msg,
		Where:  rel,
		Why:    "Running it reports a setup error instead of results.",
		Fix:    "Open the scenario and fix the problem named above.",
		Target: Target{Kind: "scenario", Path: path},
	}
}

func addSpec(root, scenarioPath, ref string, specs map[string]*Spec, order *[]string) {
	key := ref
	if !strings.HasPrefix(ref, "http://") && !strings.HasPrefix(ref, "https://") {
		key = resolveSpec(filepath.Dir(scenarioPath), ref)
	}
	sp, ok := specs[key]
	if !ok {
		sp = &Spec{Ref: ref, Refs: []string{}}
		switch {
		case strings.HasPrefix(ref, "http://") || strings.HasPrefix(ref, "https://"):
			sp.State = SpecURL
		case !within(root, key):
			sp.Path, sp.State = key, SpecOutside
		default:
			sp.Path = key
			if info, err := os.Stat(key); err != nil || info.IsDir() {
				sp.State = SpecMissing
			} else {
				sp.State = SpecFound
			}
		}
		specs[key] = sp
		*order = append(*order, key)
	}
	sp.Scenarios = append(sp.Scenarios, scenarioPath)
	for _, r := range sp.Refs {
		if r == ref {
			return
		}
	}
	sp.Refs = append(sp.Refs, ref)
}

func resolveSpec(scenarioDir, ref string) string {
	if filepath.IsAbs(ref) {
		return filepath.Clean(ref)
	}
	return filepath.Join(scenarioDir, ref)
}

// findRequestRef returns the file a step's requestRef points at, looked up the way the runner
// does (the scenario's folder, then the workspace's requests/), or "" when it is missing.
func findRequestRef(root, scenarioDir, ref string) string {
	candidates := []string{ref}
	if !filepath.IsAbs(ref) {
		candidates = []string{filepath.Join(scenarioDir, ref), filepath.Join(root, "requests", ref)}
	}
	for _, c := range candidates {
		if info, err := os.Stat(c); err == nil && !info.IsDir() {
			return c
		}
	}
	return ""
}

func describeEnv(env *domain.Environment) []EnvVar {
	unset := map[string]string{}
	for _, ref := range variables.MissingEnv(env) {
		if _, ok := unset[ref.Var]; !ok {
			unset[ref.Var] = ref.Name
		}
	}
	out := make([]EnvVar, 0, len(env.Variables))
	for _, v := range env.Variables {
		ev := EnvVar{Name: v.Name, Secret: v.Secret, Set: strings.TrimSpace(v.Value) != ""}
		if name, bad := unset[v.Name]; bad {
			ev.Set, ev.UnsetRef = false, name
		}
		out = append(out, ev)
	}
	return out
}

func envLabel(env *domain.Environment, path string) string {
	if env.Name != "" {
		return env.Name
	}
	return filepath.Base(path)
}

// nameList renders placeholder names as {{name}}, at most five, then "and N more".
func nameList(names []string) string {
	shown, more := names, ""
	if len(shown) > 5 {
		more = fmt.Sprintf(" and %d more", len(shown)-5)
		shown = shown[:5]
	}
	out := make([]string, len(shown))
	for i, n := range shown {
		out[i] = "{{" + n + "}}"
	}
	return strings.Join(out, ", ") + more
}

// joinMore lists at most max items, then "and N more <noun>".
func joinMore(items []string, max int, noun string) string {
	if len(items) <= max {
		return strings.Join(items, ", ")
	}
	return fmt.Sprintf("%s and %d more %s", strings.Join(items[:max], ", "), len(items)-max, noun)
}

func scenarioList(root string, paths []string) string {
	rel := make([]string, len(paths))
	for i, p := range paths {
		rel[i] = relTo(root, p)
	}
	if len(rel) > 3 {
		return fmt.Sprintf("%s and %d more scenarios", strings.Join(rel[:3], ", "), len(rel)-3)
	}
	return strings.Join(rel, ", ")
}

func relTo(root, p string) string {
	if rel, err := filepath.Rel(root, p); err == nil {
		return filepath.ToSlash(rel)
	}
	return p
}

// within reports whether p is inside root after resolving symbolic links on both, so a spec reached
// through a link that points out of the workspace is reported as outside, not found.
func within(root, p string) bool {
	resolve := func(x string) string {
		if real, err := filepath.EvalSymlinks(x); err == nil {
			return real
		}
		return x
	}
	rel, err := filepath.Rel(resolve(root), resolve(p))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

func isDataFile(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".yaml", ".yml", ".json":
		return true
	}
	return false
}

func countDataFiles(dir string) int {
	n := 0
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err == nil && !d.IsDir() && isDataFile(d.Name()) {
			n++
		}
		return nil
	})
	return n
}

func countDataFilesFlat(dir string) int {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if !e.IsDir() && isDataFile(e.Name()) {
			n++
		}
	}
	return n
}

// inspectProjectFile reads the clrsig.yaml in the workspace folder (never one above it) and reports
// on it. A file that cannot be used, or that names a spec or environment that is not there, is
// something to fix.
func inspectProjectFile(root string, r *Report) {
	path, err := projectconfig.InFolder(root)
	if err != nil {
		r.ProjectFile = ProjectFile{Found: true, Error: err.Error(), Keys: []string{}}
		r.Attention = append(r.Attention, projectFileProblem("", err.Error(), "Keep one project file in the workspace folder."))
		return
	}
	if path == "" {
		return
	}
	r.ProjectFile.Found, r.ProjectFile.Path = true, relTo(root, path)
	cfg, err := projectconfig.Load(path)
	if err != nil {
		r.ProjectFile.Error = err.Error()
		r.Attention = append(r.Attention, projectFileProblem(path, err.Error(),
			"Correct the key the message names, or remove the file. From a terminal, clrsig config show reads it the same way."))
		return
	}
	r.ProjectFile.Keys = cfg.SetKeys()
	check := func(key, value, kind string) {
		if value == "" || strings.Contains(value, "://") {
			return
		}
		full := value
		if !filepath.IsAbs(full) {
			full = filepath.Join(cfg.Dir(), filepath.FromSlash(value))
		}
		if _, err := os.Stat(full); err == nil {
			return
		}
		r.Attention = append(r.Attention, Attention{
			ID: ItemProjectFilePath, Severity: SeverityWarning,
			What:   fmt.Sprintf("%s names a %s that does not exist: %q.", r.ProjectFile.Path, kind, value),
			Where:  fmt.Sprintf("%s, key %s", r.ProjectFile.Path, key),
			Why:    "Commands that read the project file (clrsig conform with no arguments, for example) fail until the file is there.",
			Fix:    fmt.Sprintf("Put the %s at that path, or change %s in %s.", kind, key, r.ProjectFile.Path),
			Target: Target{Kind: "projectfile", Path: path},
		})
	}
	check("spec", cfg.Spec, "spec")
	check("environment", cfg.Environment, "environment file")
}

func projectFileProblem(path, why, fix string) Attention {
	where := "the workspace folder"
	if path != "" {
		where = filepath.Base(path)
	}
	return Attention{
		ID: ItemProjectFileInvalid, Severity: SeverityWarning,
		What:   "The project file cannot be used.",
		Where:  where,
		Why:    why,
		Fix:    fix,
		Target: Target{Kind: "projectfile", Path: path},
	}
}
