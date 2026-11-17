package clibuild

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

var (
	placeholderRe = regexp.MustCompile(`\{\{\s*([A-Za-z0-9_.\-]+)\s*\}\}`)
	envRefRe      = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
)

// inspection is what the builder learns by opening the files the user picked, using the
// same parsers and checks as the CLI itself.
type inspection struct {
	scenarioDir      *folderRef // set when a folder (not a single file) was chosen
	scenarioSpecFile string
	usedVariables    []string // {{names}} used by inline requests, minus names the scenario extracts itself
	envVariables     map[string]bool
	haveEnv          bool
	needs            []Need
}

func inspectScenario(abs, display string, in *inspection, add func(Message)) {
	data, err := os.ReadFile(abs)
	if err != nil {
		add(Message{Field: "scenario-file", Severity: Error, Text: fmt.Sprintf("cannot read %s: %v", display, err)})
		return
	}
	sc, err := serialization.ParseScenarioFile(data, abs)
	if err != nil {
		add(Message{Field: "scenario-file", Severity: Error, Text: fmt.Sprintf("%s is not a valid scenario: %v", display, err)})
		return
	}
	issues := scenario.Validate(sc, filepath.Dir(abs))
	for _, iss := range issues {
		add(Message{Field: "scenario-file", Severity: Error, Text: iss.String()})
	}
	in.scenarioSpecFile = sc.SpecFile
	in.usedVariables = usedVariables(sc)
	if len(issues) == 0 {
		text := fmt.Sprintf("valid, %d step%s", len(sc.Steps), plural(len(sc.Steps)))
		if sc.SpecFile != "" {
			text += fmt.Sprintf("; checks responses against %s (relative to the scenario file)", sc.SpecFile)
		}
		add(Message{Field: "scenario-file", Severity: Info, Text: text})
	}
}

// usedVariables lists the {{names}} the scenario's own requests use that the scenario does
// not define by extracting them from an earlier response.
func usedVariables(sc *domain.Scenario) []string {
	defined := map[string]bool{}
	for _, st := range sc.Steps {
		for _, ex := range st.Extractions {
			defined[ex.Name] = true
		}
	}
	seen := map[string]bool{}
	for _, st := range sc.Steps {
		if st.Request == nil {
			continue
		}
		raw, err := json.Marshal(st.Request)
		if err != nil {
			continue
		}
		for _, m := range placeholderRe.FindAllStringSubmatch(string(raw), -1) {
			if !defined[m[1]] {
				seen[m[1]] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for k := range seen {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func inspectEnvironment(abs, display string, in *inspection, add func(Message)) {
	data, err := os.ReadFile(abs)
	if err != nil {
		add(Message{Field: "env", Severity: Error, Text: fmt.Sprintf("cannot read %s: %v", display, err)})
		return
	}
	env, err := serialization.ParseEnvironmentFile(data, abs)
	if err != nil {
		add(Message{Field: "env", Severity: Error, Text: fmt.Sprintf("%s is not a valid environment: %v", display, err)})
		return
	}
	in.haveEnv = true
	in.envVariables = map[string]bool{}
	seenNeed := map[string]bool{}
	for _, v := range env.Variables {
		in.envVariables[v.Name] = true
		for _, m := range envRefRe.FindAllStringSubmatch(v.Value, -1) {
			if seenNeed[m[1]] {
				continue
			}
			seenNeed[m[1]] = true
			in.needs = append(in.needs, Need{Name: m[1], Secret: v.Secret, From: v.Name})
		}
	}
	add(Message{Field: "env", Severity: Info, Text: fmt.Sprintf("valid, %d variable%s", len(env.Variables), plural(len(env.Variables)))})
}

func inspectSpec(field, abs, display string, add func(Message)) {
	data, err := os.ReadFile(abs)
	if err != nil {
		add(Message{Field: field, Severity: Error, Text: fmt.Sprintf("cannot read %s: %v", display, err)})
		return
	}
	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
	if format == "yml" {
		format = "yaml"
	}
	spec, err := openapi.ParseBytes(data, format)
	if err != nil {
		add(Message{Field: field, Severity: Error, Text: fmt.Sprintf("%s is not a valid OpenAPI document: %v", display, err)})
		return
	}
	title, version := spec.Info()
	ops := 0
	if reqs, err := openapi.GenerateRequests(spec, ""); err == nil {
		ops = len(reqs)
	}
	add(Message{Field: field, Severity: Info, Text: fmt.Sprintf("%s %s, %d operation%s", title, version, ops, plural(ops))})
}

func plural(n int) string {
	if n == 1 {
		return ""
	}
	return "s"
}

type folderRef struct {
	abs     string
	display string
}

const maxListed = 5

// inspectScenarioFolder finds the scenarios the CLI would run for a folder (same discovery as
// clrsig run) and checks each of them, so the user sees the count, the skipped files and any
// broken scenario before running anything.
func inspectScenarioFolder(ref folderRef, recursive bool, in *inspection, add func(Message)) {
	const field = "scenario-file"
	found, err := scenario.Discover([]string{ref.abs}, recursive)
	if err != nil {
		for _, line := range strings.Split(err.Error(), "\n") {
			add(Message{Field: field, Severity: Error, Text: strings.TrimSpace(line)})
		}
		return
	}

	used := map[string]bool{}
	withSpec := 0
	var broken []string
	for _, file := range found.Files {
		label := filepath.ToSlash(relOrSelf(ref.abs, file))
		data, err := os.ReadFile(file)
		if err != nil {
			broken = append(broken, fmt.Sprintf("%s: cannot read: %v", label, err))
			continue
		}
		sc, err := serialization.ParseScenarioFile(data, file)
		if err != nil {
			broken = append(broken, fmt.Sprintf("%s: not a valid scenario: %v", label, err))
			continue
		}
		if issues := scenario.Validate(sc, filepath.Dir(file)); len(issues) > 0 {
			broken = append(broken, fmt.Sprintf("%s: %s", label, issues[0].String()))
			continue
		}
		if sc.SpecFile != "" {
			withSpec++
			if in.scenarioSpecFile == "" {
				in.scenarioSpecFile = sc.SpecFile
			}
		}
		for _, v := range usedVariables(sc) {
			used[v] = true
		}
	}
	for v := range used {
		in.usedVariables = append(in.usedVariables, v)
	}
	sort.Strings(in.usedVariables)

	for i, b := range broken {
		if i == maxListed {
			add(Message{Field: field, Severity: Error, Text: fmt.Sprintf("and %d more broken scenario%s", len(broken)-maxListed, plural(len(broken)-maxListed))})
			break
		}
		add(Message{Field: field, Severity: Error, Text: b})
	}

	n := len(found.Files)
	scope := "in " + ref.display
	if recursive {
		scope += " and its sub-folders"
	}
	text := fmt.Sprintf("%d scenario%s found %s", n, plural(n), scope)
	if len(broken) == 0 {
		text = fmt.Sprintf("%d scenario%s found %s, all valid", n, plural(n), scope)
	}
	if k := len(found.Skipped); k > 0 {
		text += fmt.Sprintf("; %d other file%s skipped (not scenarios)", k, plural(k))
	}
	add(Message{Field: field, Severity: Info, Text: text})

	if withSpec > 0 && withSpec < n-len(broken) {
		add(Message{Field: field, Severity: Warning, Text: fmt.Sprintf("only %d of %d scenarios name a specFile:, so only those are checked against a spec unless you set one here", withSpec, n-len(broken))})
	}
}
