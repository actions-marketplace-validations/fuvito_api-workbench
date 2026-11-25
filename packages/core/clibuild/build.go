// Package clibuild turns a user's choices into a correct, explained clrsig command line.
//
// It reads the clispec catalog, resolves and checks every path against the folder the
// command will be run from, opens the picked files with the same parsers the CLI uses,
// and renders the result for bash, PowerShell and cmd. It never executes anything.
package clibuild

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/clispec"
	"github.com/fuvito/api-workbench/packages/core/projectconfig"
)

// Program is the executable name used in generated commands.
const Program = "clrsig"

// Severity ranks a message. Any Error makes the command invalid.
type Severity string

const (
	Error   Severity = "error"
	Warning Severity = "warning"
	Info    Severity = "info"
)

// Message is a finding about the whole command (Field empty) or about one input.
type Message struct {
	Field    string
	Severity Severity
	Text     string
}

// Need is an environment variable the command depends on when it runs, for example a
// token that an environment file reads as ${API_TOKEN}.
type Need struct {
	Name   string
	Secret bool
	From   string // the environment variable that refers to it
}

// Part explains one piece of the command line.
type Part struct {
	Token   string
	Meaning string
}

// Rendering is the command written for one shell.
type Rendering struct {
	Shell     Shell
	Multiline string
	Oneline   string
}

// Explanation tells the user what the command does, what it expects and what to expect back.
type Explanation struct {
	Goal      string
	Summary   string
	When      string
	Parts     []Part
	Outputs   []string
	ExitCodes []clispec.ExitCode
}

// Request is what the user chose. Values are keyed by catalog input name. Booleans use
// "true". File values may be absolute or relative to the run-from folder.
type Request struct {
	Command string
	Values  map[string]string
}

// Result is the built command.
type Result struct {
	Command     string
	Valid       bool
	Argv        []string // Argv[0] is Program; this is exactly what gets executed
	Renderings  []Rendering
	Messages    []Message
	Needs       []Need
	Suggestions []Suggestion
	Explanation Explanation
}

// Suggestion proposes a value for an input the user has not filled in.
type Suggestion struct {
	Field string
	Value string
	Why   string
}

type arg struct {
	text     string
	path     bool // a local relative path, shown with backslashes for cmd.exe
	flagName bool
}

// DefaultRunFrom returns the folder commands are best run from: the nearest ancestor of
// workspace that contains .git (the repository root), or workspace itself.
func DefaultRunFrom(workspace string) string {
	abs, err := filepath.Abs(workspace)
	if err != nil {
		return workspace
	}
	for dir := abs; ; {
		if _, err := os.Stat(filepath.Join(dir, ".git")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return abs
		}
		dir = parent
	}
}

// Build validates the request and renders the command. runFrom is the folder the user will
// run the command from; relative paths in the result are relative to it.
func Build(req Request, runFrom string) Result {
	res := Result{Command: req.Command}
	add := func(m Message) { res.Messages = append(res.Messages, m) }

	cmd, ok := clispec.Get(req.Command)
	if !ok {
		add(Message{Severity: Error, Text: fmt.Sprintf("unknown command %q", req.Command)})
		return res
	}

	root, err := filepath.Abs(runFrom)
	if err != nil || runFrom == "" {
		add(Message{Field: "runFrom", Severity: Error, Text: "choose the folder you will run the command from"})
		return res
	}
	if st, err := os.Stat(root); err != nil || !st.IsDir() {
		add(Message{Field: "runFrom", Severity: Error, Text: fmt.Sprintf("%s is not a folder", root)})
		return res
	}

	var in inspection
	values := map[string]string{}
	args := []arg{{text: Program}, {text: cmd.Name}}
	var positional []arg
	var flags []arg

	for _, input := range cmd.Inputs {
		if input.Deferred {
			continue
		}
		raw := strings.TrimSpace(req.Values[input.Name])
		if raw == "" || (input.Kind == clispec.KindBool && raw != "true") {
			if input.Required {
				add(Message{Field: input.Name, Severity: Error, Text: input.Label + " is required"})
			}
			continue
		}

		text, isPath, abs, ok := resolveValue(input, raw, root, add)
		if !ok {
			continue
		}
		values[input.Name] = text

		if input.Kind == clispec.KindOutputFile {
			outputNotes(input.Name, text, root, add)
		}

		if abs != "" {
			display := text
			switch input.Role {
			case clispec.RoleScenario:
				if st, err := os.Stat(abs); err == nil && st.IsDir() {
					in.scenarioDir = &folderRef{abs: abs, display: display}
				} else {
					inspectScenario(abs, display, &in, add)
				}
			case clispec.RoleEnvironment:
				inspectEnvironment(abs, display, &in, add)
			case clispec.RoleSpec:
				inspectSpec(input.Name, abs, display, add)
			}
		}

		if input.Positional {
			positional = append(positional, arg{text: text, path: isPath})
			continue
		}
		if input.Kind == clispec.KindBool {
			flags = append(flags, arg{text: "--" + input.Name, flagName: true})
			continue
		}
		if input.Default != "" && text == input.Default {
			delete(values, input.Name)
			continue
		}
		flags = append(flags, arg{text: "--" + input.Name, flagName: true}, arg{text: text, path: isPath})
	}

	if in.scenarioDir != nil {
		inspectScenarioFolder(*in.scenarioDir, values["no-recursive"] != "true", &in, add)
	} else if values["scenario-file"] != "" {
		for _, flag := range []string{"fail-fast", "no-recursive"} {
			if values[flag] == "true" {
				add(Message{Field: flag, Severity: Warning, Text: "has no effect when you run a single scenario file; it only applies to folders and several scenarios"})
			}
		}
	}

	checkRequirements(cmd, values, in, add)
	checkVariables(in, add)
	noteProjectFile(req.Command, root, add)
	res.Suggestions = checkReport(cmd, values, add)

	res.Needs = in.needs
	res.Argv = argv(args, positional, flags)
	res.Renderings = renderAll(args, positional, flags, add)
	res.Explanation = explain(cmd, positional, flags, values)
	res.Valid = true
	for _, m := range res.Messages {
		if m.Severity == Error {
			res.Valid = false
			break
		}
	}
	return res
}

// resolveValue validates one input value and returns the text to put in the command line.
// For local files it also returns the absolute path so the caller can open it.
func resolveValue(input clispec.Input, raw, root string, add func(Message)) (text string, isPath bool, abs string, ok bool) {
	fail := func(format string, a ...any) (string, bool, string, bool) {
		add(Message{Field: input.Name, Severity: Error, Text: fmt.Sprintf(format, a...)})
		return "", false, "", false
	}

	switch input.Kind {
	case clispec.KindBool:
		return "true", false, "", true

	case clispec.KindInt:
		n, err := strconv.Atoi(raw)
		if err != nil || n < 0 {
			return fail("%s must be a whole number of 0 or more", input.Label)
		}
		return strconv.Itoa(n), false, "", true

	case clispec.KindEnum:
		for _, v := range input.Enum {
			if v == raw {
				return raw, false, "", true
			}
		}
		return fail("%s must be one of: %s", input.Label, strings.Join(input.Enum, ", "))

	case clispec.KindURL:
		u, err := url.Parse(raw)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return fail("%s must be an http:// or https:// address", input.Label)
		}
		return raw, false, "", true

	case clispec.KindString:
		return raw, false, "", true

	case clispec.KindFileOrURL:
		if strings.HasPrefix(raw, "http://") || strings.HasPrefix(raw, "https://") {
			if u, err := url.Parse(raw); err != nil || u.Host == "" {
				return fail("%s is not a valid address", input.Label)
			}
			add(Message{Field: input.Name, Severity: Info, Text: "remote document; it is fetched when the command runs and is not checked here"})
			return raw, false, "", true
		}
	}

	// Local path (file, dir, fileOrURL given as a path, outputFile).
	full := raw
	if !filepath.IsAbs(full) {
		full = filepath.Join(root, raw)
	}
	full = filepath.Clean(full)

	switch input.Kind {
	case clispec.KindFile, clispec.KindFileOrURL:
		st, err := os.Stat(full)
		if err != nil {
			return fail("%s not found: %s", input.Label, raw)
		}
		if st.IsDir() {
			return fail("%s must be a file, but %s is a folder", input.Label, raw)
		}
		if len(input.Accepts) > 0 && !hasExt(full, input.Accepts) {
			add(Message{Field: input.Name, Severity: Warning, Text: fmt.Sprintf("expected a %s file", strings.Join(input.Accepts, " / "))})
		}
	case clispec.KindPath:
		st, err := os.Stat(full)
		if err != nil {
			return fail("%s not found: %s", input.Label, raw)
		}
		if !st.IsDir() && len(input.Accepts) > 0 && !hasExt(full, input.Accepts) {
			add(Message{Field: input.Name, Severity: Warning, Text: fmt.Sprintf("expected a %s file", strings.Join(input.Accepts, " / "))})
		}
	case clispec.KindDir:
		st, err := os.Stat(full)
		if err != nil || !st.IsDir() {
			return fail("%s must be an existing folder: %s", input.Label, raw)
		}
	case clispec.KindOutputFile:
		if st, err := os.Stat(full); err == nil && st.IsDir() {
			return fail("%s must be a file name, but %s is a folder", input.Label, raw)
		}
	}

	rel, err := filepath.Rel(root, full)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		add(Message{Field: input.Name, Severity: Warning, Text: "this path is outside the folder you run the command from, so it is written as an absolute path and will not exist on another machine or a CI runner"})
		return full, false, absIfExisting(input, full), true
	}
	return filepath.ToSlash(rel), true, absIfExisting(input, full), true
}

func absIfExisting(input clispec.Input, full string) string {
	if input.Kind == clispec.KindOutputFile {
		return ""
	}
	return full
}

func hasExt(path string, exts []string) bool {
	ext := strings.ToLower(filepath.Ext(path))
	for _, e := range exts {
		if ext == e {
			return true
		}
	}
	return false
}

func checkRequirements(cmd clispec.Command, values map[string]string, in inspection, add func(Message)) {
	for _, input := range cmd.Inputs {
		if input.Deferred || values[input.Name] == "" {
			continue
		}
		for _, r := range input.Requires {
			if values[r.Input] != "" {
				continue
			}
			if r.Or == clispec.ImplicitScenarioSpec && in.scenarioSpecFile != "" {
				continue
			}
			need, _ := cmd.Input(r.Input)
			text := fmt.Sprintf("--%s needs %s", input.Name, strings.ToLower(need.Label))
			if r.Or == clispec.ImplicitScenarioSpec {
				text += " (set it here, or add specFile: to the scenario)"
			}
			add(Message{Field: input.Name, Severity: Error, Text: text})
		}
	}
}

func checkVariables(in inspection, add func(Message)) {
	if len(in.usedVariables) == 0 {
		return
	}
	if !in.haveEnv {
		add(Message{Field: "env", Severity: Warning, Text: fmt.Sprintf("the scenario uses %s but no environment is selected, so they stay unresolved", braces(in.usedVariables))})
		return
	}
	var missing []string
	for _, v := range in.usedVariables {
		if !in.envVariables[v] {
			missing = append(missing, v)
		}
	}
	if len(missing) > 0 {
		add(Message{Field: "env", Severity: Warning, Text: fmt.Sprintf("the scenario uses %s, which this environment does not define", braces(missing))})
	}
}

func braces(names []string) string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "{{" + n + "}}"
	}
	return strings.Join(out, ", ")
}

func argv(head, positional, flags []arg) []string {
	var out []string
	for _, group := range [][]arg{head, positional, flags} {
		for _, a := range group {
			out = append(out, a.text)
		}
	}
	return out
}

func renderAll(head, positional, flags []arg, add func(Message)) []Rendering {
	// A line group is one flag with its value, or the program, command and positionals.
	first := append(append([]arg{}, head...), positional...)
	var groups [][]arg
	groups = append(groups, first)
	for i := 0; i < len(flags); i++ {
		if flags[i].flagName && i+1 < len(flags) && !flags[i+1].flagName {
			groups = append(groups, flags[i:i+2])
			i++
		} else {
			groups = append(groups, flags[i:i+1])
		}
	}

	warned := map[string]bool{}
	var out []Rendering
	for _, shell := range Shells {
		var lines []string
		var all []string
		for _, g := range groups {
			var parts []string
			for _, a := range g {
				text := a.text
				if shell == Cmd && a.path && !filepath.IsAbs(text) {
					text = strings.ReplaceAll(text, "/", `\`)
				}
				q, warn := quote(shell, text)
				if warn != "" && !warned[warn] {
					warned[warn] = true
					add(Message{Severity: Warning, Text: fmt.Sprintf("%s: %q %s", shell, text, warn)})
				}
				parts = append(parts, q)
			}
			line := strings.Join(parts, " ")
			lines = append(lines, line)
			all = append(all, line)
		}
		out = append(out, Rendering{
			Shell:     shell,
			Multiline: strings.Join(lines, continuation(shell)+"\n  "),
			Oneline:   strings.Join(all, " "),
		})
	}
	return out
}

func explain(cmd clispec.Command, positional, flags []arg, values map[string]string) Explanation {
	ex := Explanation{
		Goal: cmd.Goal, Summary: cmd.Summary, When: cmd.When,
		Outputs: cmd.Outputs, ExitCodes: cmd.ExitCodes,
	}
	ex.Parts = append(ex.Parts,
		Part{Token: Program, Meaning: "The Clear Signal command-line program."},
		Part{Token: cmd.Name, Meaning: cmd.Summary},
	)
	for i, p := range cmd.Positionals() {
		if i < len(positional) {
			ex.Parts = append(ex.Parts, Part{Token: positional[i].text, Meaning: p.Label + ": " + p.Help})
		}
	}
	for _, f := range cmd.Flags() {
		v, set := values[f.Name]
		if !set || f.Deferred {
			continue
		}
		token := "--" + f.Name
		if f.Kind != clispec.KindBool {
			token += " " + v
		}
		ex.Parts = append(ex.Parts, Part{Token: token, Meaning: f.Label + ": " + f.Help})
	}
	return ex
}

// noteProjectFile tells the user when a clrsig.yaml would also apply where they run the command.
// The command built here is always written in full, so it does not depend on the file.
func noteProjectFile(command, root string, add func(Message)) {
	switch command {
	case "conform", "run", "validate":
	default:
		return
	}
	home, _ := os.UserHomeDir()
	path, err := projectconfig.Discover(root, home)
	if err != nil || path == "" {
		return
	}
	add(Message{Severity: Info, Text: fmt.Sprintf(
		"A project file (%s) applies where you run this command. The command shown is written in full and does not depend on it; to see what the file supplies when a value is left out, run: clrsig config show %s",
		filepath.Base(path), command)})
}
