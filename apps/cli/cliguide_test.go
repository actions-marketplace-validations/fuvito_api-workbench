package main

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"testing"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

// guidePages names the page of the detailed CLI guide (docs/guides/cli/) that documents each command.
// Commands of one group share a page.
var guidePages = map[string]string{
	"bundle export":       "bundle.md",
	"bundle import":       "bundle.md",
	"bundle run":          "bundle.md",
	"bundle validate":     "bundle.md",
	"config show":         "config-show.md",
	"conform":             "conform.md",
	"coverage list":       "coverage.md",
	"coverage quality":    "coverage.md",
	"coverage reset":      "coverage.md",
	"diff":                "diff.md",
	"generate":            "generate.md",
	"impact":              "impact.md",
	"import":              "import.md",
	"init":                "init.md",
	"init github-actions": "init.md",
	"mcp":                 "mcp.md",
	"request":             "request.md",
	"run":                 "run.md",
	"telemetry disable":   "telemetry.md",
	"telemetry enable":    "telemetry.md",
	"telemetry status":    "telemetry.md",
	"validate":            "validate.md",
}

// guidePending lists the commands whose page is not written yet. It may only shrink: an entry whose
// page exists fails the test. Every command has a page now, so a new command needs one at once.
var guidePending = []string{}

// runnableCommands returns every command a user can run, by its name without "clrsig".
func runnableCommands() map[string]*cobra.Command {
	out := map[string]*cobra.Command{}
	var walk func(c *cobra.Command, name string)
	walk = func(c *cobra.Command, name string) {
		if name != "" && c.Runnable() {
			out[name] = c
		}
		for _, sub := range c.Commands() {
			if sub.Name() == "help" || sub.Name() == "completion" || sub.Hidden {
				continue
			}
			walk(sub, strings.TrimSpace(name+" "+sub.Name()))
		}
	}
	walk(newRootCmd(), "")
	return out
}

var reGuideFlag = regexp.MustCompile("`--([a-z][a-z0-9-]*)")
var reGuideShort = regexp.MustCompile("`-([a-zA-Z])`")

// optionRows reads the options table under a heading "## Options" (or, on a page for several
// commands, "## Options: `clrsig <command>`"): flag name -> the whole row.
func optionRows(page, command string, alone bool) (map[string]string, bool) {
	heading := "## Options: `clrsig " + command + "`"
	lines := strings.Split(strings.ReplaceAll(page, "\r\n", "\n"), "\n")
	start := slices.Index(lines, heading)
	if start < 0 && alone {
		start = slices.Index(lines, "## Options")
	}
	if start < 0 {
		return nil, false
	}
	rows := map[string]string{}
	for _, line := range lines[start+1:] {
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "| `-") {
			continue
		}
		first := strings.SplitN(line[2:], " |", 2)[0]
		for _, m := range reGuideFlag.FindAllStringSubmatch(first, -1) {
			rows[m[1]] = line
		}
	}
	return rows, true
}

// Every command has a page (or is still pending), and its options table names exactly the flags the
// command has, with their short forms and defaults.
func TestDetailedGuideCoversEveryCommandAndFlag(t *testing.T) {
	commands := runnableCommands()
	if len(commands) < 20 {
		t.Fatalf("only %d commands were found", len(commands))
	}
	dir := filepath.Join("..", "..", "docs", "guides", "cli")

	for _, name := range guidePending {
		if _, ok := commands[name]; !ok {
			t.Errorf("guidePending names %q, which is not a command", name)
		}
		if _, ok := guidePages[name]; ok {
			t.Errorf("%q has a page now: remove it from guidePending", name)
		}
	}
	for name := range guidePages {
		if _, ok := commands[name]; !ok {
			t.Errorf("guidePages names %q, which is not a command", name)
		}
	}

	perPage := map[string]int{}
	for _, page := range guidePages {
		perPage[page]++
	}
	index, err := os.ReadFile(filepath.Join(dir, "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	for name, cmd := range commands {
		file, ok := guidePages[name]
		if !ok {
			if !slices.Contains(guidePending, name) {
				t.Errorf("clrsig %s has no page in docs/guides/cli: write one and add it to guidePages", name)
			}
			continue
		}
		if !strings.Contains(string(index), "]("+file+")") {
			t.Errorf("the guide's index does not link to %s", file)
		}
		data, err := os.ReadFile(filepath.Join(dir, file))
		if err != nil {
			t.Errorf("clrsig %s: %v", name, err)
			continue
		}
		rows, found := optionRows(string(data), name, perPage[file] == 1)
		if !found {
			t.Errorf("%s has no options section for clrsig %s", file, name)
			continue
		}
		want := map[string]*pflag.Flag{}
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Name != "help" && !f.Hidden {
				want[f.Name] = f
			}
		})
		for flag, f := range want {
			row, ok := rows[flag]
			if !ok {
				t.Errorf("%s: the options table of clrsig %s does not list --%s", file, name, flag)
				continue
			}
			first := strings.SplitN(row[2:], " |", 2)[0]
			short := ""
			if m := reGuideShort.FindStringSubmatch(first); m != nil {
				short = m[1]
			}
			if short != f.Shorthand {
				t.Errorf("%s: --%s has the short form %q, the table says %q", file, flag, f.Shorthand, short)
			}
			if d := f.DefValue; d != "" && d != "false" && d != "0" && d != "[]" && !strings.Contains(row, "`"+d+"`") {
				t.Errorf("%s: the row of --%s does not give its default `%s`", file, flag, d)
			}
		}
		for flag := range rows {
			if _, ok := want[flag]; !ok {
				t.Errorf("%s: the options table lists --%s, which clrsig %s does not have", file, flag, name)
			}
		}
	}
}
