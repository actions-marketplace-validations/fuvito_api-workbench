package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	osexec "os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/reporting"
)

type diffOptions struct {
	format       string
	outputFile   string
	breakingOnly bool
	failOn       string
	against      string
	reports      []string
}

func diffCmd() *cobra.Command {
	var o diffOptions

	cmd := &cobra.Command{
		Use:   "diff <base-spec> <revision-spec>",
		Short: "Compare two OpenAPI specs and report breaking and non-breaking changes",
		Long: `Parse two OpenAPI 3.0/3.1 specs and classify every change as breaking
(ERR), potentially breaking (WARN), or informational (INFO).

With --against <git-ref>, give one spec file instead of two: its version at that git ref
(for example origin/main) is the base, and the file on disk is the revision.

  clrsig diff openapi.yaml --against origin/main

Exit codes:
  0  no change at or above the --fail-on level (breaking by default)
  1  one or more changes at or above the --fail-on level
  2  spec parse or configuration error`,
		Args: func(cmd *cobra.Command, args []string) error {
			if o.against != "" {
				return cobra.ExactArgs(1)(cmd, args)
			}
			return cobra.ExactArgs(2)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return runDiff(cmd, args, o)
		},
	}

	cmd.Flags().StringVarP(&o.format, "format", "f", "text", formatUsage)
	cmd.Flags().StringVarP(&o.outputFile, "output", "o", "", "Write report to file (parent directories created automatically)")
	cmd.Flags().StringArrayVar(&o.reports, "report", nil, "Also write a report to a file: format=file, for example junit=reports/diff.xml (repeatable)")
	cmd.Flags().BoolVar(&o.breakingOnly, "breaking-only", false, "Show only breaking changes")
	cmd.Flags().StringVar(&o.failOn, "fail-on", "breaking", "Fail when a change is at least this serious: breaking, warning, or info")
	cmd.Flags().StringVar(&o.against, "against", "", "Compare against the spec at this git ref (needs git); give one spec file")
	return cmd
}

func runDiff(cmd *cobra.Command, args []string, o diffOptions) error {
	failOn, err := openapi.ParseFailOn(o.failOn)
	if err != nil {
		return err
	}
	if o.breakingOnly && failOn != openapi.FailOnBreaking {
		return fmt.Errorf("--breaking-only hides the changes --fail-on %s would fail on; use one or the other", failOn)
	}
	plan, err := planReports(o.format, o.reports, o.outputFile)
	if err != nil {
		return err
	}

	var baseData, revData []byte
	var baseFmt, revFmt string
	if o.against != "" {
		if strings.HasPrefix(args[0], "http://") || strings.HasPrefix(args[0], "https://") {
			return fmt.Errorf("--against needs a local spec file, not a URL")
		}
		revData, revFmt, err = fetchSpec(args[0])
		if err != nil {
			return err
		}
		baseData, baseFmt, err = gitSpecAt(args[0], o.against)
	} else {
		if baseData, baseFmt, err = fetchSpec(args[0]); err == nil {
			revData, revFmt, err = fetchSpec(args[1])
		}
	}
	if err != nil {
		return err
	}

	baseSpec, err := openapi.ParseBytes(baseData, baseFmt)
	if err != nil {
		return fmt.Errorf("parsing base spec: %w", err)
	}
	revSpec, err := openapi.ParseBytes(revData, revFmt)
	if err != nil {
		return fmt.Errorf("parsing revision spec: %w", err)
	}
	result, err := openapi.Diff(baseSpec, revSpec)
	if err != nil {
		return fmt.Errorf("diffing specs: %w", err)
	}

	changes := result.Changes
	if o.breakingOnly {
		filtered := make([]openapi.DiffChange, 0, len(changes))
		for _, c := range changes {
			if c.Breaking {
				filtered = append(filtered, c)
			}
		}
		changes = filtered
	}

	render := func(format string) ([]byte, error) {
		switch format {
		case "junit":
			b, err := reporting.DiffJUnitOn(result, failOn)
			if err != nil {
				return nil, fmt.Errorf("generating JUnit report: %w", err)
			}
			return b, nil
		case "html":
			b, err := reporting.DiffHTMLOn(result, changes, failOn)
			if err != nil {
				return nil, fmt.Errorf("generating HTML report: %w", err)
			}
			return b, nil
		case "markdown":
			return reporting.DiffMarkdown(result, changes, failOn), nil
		case "json":
			return diffJSON(result, changes, failOn)
		}
		return []byte(diffText(result, changes, o.breakingOnly, failOn)), nil
	}
	if err := plan.emit(cmd, o.outputFile, render); err != nil {
		return err
	}

	if result.Fails(failOn) {
		return &exitError{1}
	}
	return nil
}

func diffJSON(result *openapi.DiffResult, changes []openapi.DiffChange, failOn openapi.FailOn) ([]byte, error) {
	type jsonChange struct {
		ID       string `json:"id"`
		Level    string `json:"level"`
		Breaking bool   `json:"breaking"`
		Method   string `json:"method,omitempty"`
		Path     string `json:"path,omitempty"`
		Text     string `json:"text"`
	}
	type jsonOut struct {
		HasBreaking bool         `json:"hasBreaking"`
		FailOn      string       `json:"failOn"`
		Failed      bool         `json:"failed"`
		Changes     []jsonChange `json:"changes"`
	}
	out := jsonOut{HasBreaking: result.HasBreaking, FailOn: string(failOn), Failed: result.Fails(failOn)}
	for _, c := range changes {
		out.Changes = append(out.Changes, jsonChange{ID: c.ID, Level: c.Level, Breaking: c.Breaking, Method: c.Method, Path: c.Path, Text: c.Text})
	}
	b, err := json.MarshalIndent(out, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

func diffText(result *openapi.DiffResult, changes []openapi.DiffChange, breakingOnly bool, failOn openapi.FailOn) string {
	var sb strings.Builder
	if len(changes) == 0 {
		if breakingOnly {
			fmt.Fprintln(&sb, "No breaking changes.")
		} else {
			fmt.Fprintln(&sb, "No changes.")
		}
		return sb.String()
	}
	for _, c := range changes {
		mark := " "
		if c.Breaking {
			mark = "!"
		}
		loc := ""
		if c.Method != "" && c.Path != "" {
			loc = fmt.Sprintf("  %s %s", c.Method, c.Path)
		}
		fmt.Fprintf(&sb, "  [%s] %s  %s%s\n", c.Level, mark, c.Text, loc)
	}
	fmt.Fprintln(&sb)
	breaking := 0
	for _, c := range changes {
		if c.Breaking {
			breaking++
		}
	}
	switch {
	case failOn != openapi.FailOnBreaking:
		fmt.Fprintf(&sb, "%d change(s)  ·  %d breaking  ·  %d at or above %q\n", len(changes), breaking, result.FailCount(failOn), string(failOn))
	case result.HasBreaking:
		fmt.Fprintf(&sb, "%d change(s)  ·  %d breaking\n", len(changes), breaking)
	default:
		fmt.Fprintf(&sb, "%d change(s)  ·  no breaking changes\n", len(changes))
	}
	return sb.String()
}

// gitSpecAt returns the contents of specPath as committed at ref, found through the git
// repository that contains the file.
func gitSpecAt(specPath, ref string) ([]byte, string, error) {
	// The ref becomes part of one git argument (<ref>:<path>); a ref that starts with "-" would be
	// read by git as an option (for example --output=<file>), so it is refused.
	if strings.HasPrefix(ref, "-") || strings.ContainsAny(ref, "\x00\n\r") {
		return nil, "", fmt.Errorf("--against %q is not a valid git ref (it must not start with '-')", ref)
	}
	if strings.HasPrefix(specPath, "http://") || strings.HasPrefix(specPath, "https://") {
		return nil, "", fmt.Errorf("--against needs a local spec file, not a URL")
	}
	if _, err := osexec.LookPath("git"); err != nil {
		return nil, "", fmt.Errorf("--against needs git, which is not installed or not on PATH")
	}
	abs, err := filepath.Abs(specPath)
	if err != nil {
		return nil, "", err
	}
	dir := filepath.Dir(abs)

	top, err := runGit(dir, "rev-parse", "--show-toplevel")
	if err != nil {
		return nil, "", fmt.Errorf("--against needs the spec to be in a git repository: %s is not in one", dir)
	}
	rel, err := filepath.Rel(filepath.FromSlash(strings.TrimSpace(string(top))), abs)
	if err != nil {
		return nil, "", fmt.Errorf("working out the path of %s inside the repository: %w", specPath, err)
	}

	data, err := runGit(dir, "show", ref+":"+filepath.ToSlash(rel))
	if err != nil {
		msg := err.Error()
		switch {
		case strings.Contains(msg, "invalid object name") || strings.Contains(msg, "unknown revision") || strings.Contains(msg, "bad revision"):
			return nil, "", fmt.Errorf("git ref %q was not found; in a CI checkout the ref may be missing from a shallow clone (fetch more history, for example fetch-depth: 0)", ref)
		case strings.Contains(msg, "does not exist") || strings.Contains(msg, "exists on disk, but not in"):
			return nil, "", fmt.Errorf("%s does not exist at git ref %q (it may be new on this branch)", filepath.ToSlash(rel), ref)
		}
		return nil, "", fmt.Errorf("reading %s at %q from git: %s", filepath.ToSlash(rel), ref, strings.TrimSpace(msg))
	}

	format := strings.TrimPrefix(strings.ToLower(filepath.Ext(abs)), ".")
	if format == "yml" {
		format = "yaml"
	}
	return data, format, nil
}

func runGit(dir string, args ...string) ([]byte, error) {
	cmd := osexec.Command("git", append([]string{"-C", dir}, args...)...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("%s", strings.TrimSpace(stderr.String()+" "+err.Error()))
	}
	return stdout.Bytes(), nil
}
