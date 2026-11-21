package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/serialization"
	"github.com/spf13/cobra"
)

func impactCmd() *cobra.Command {
	var format string
	var outputFile string
	var noRecursive bool

	cmd := &cobra.Command{
		Use:   "impact <base-spec> <revision-spec> <scenario-file-or-folder>...",
		Short: "Show which scenarios will break when a spec change ships",
		Long: `Compare two OpenAPI specs and cross-reference breaking changes against
scenario files. Reports which scenarios contain steps that match a breaking
change path, so you know what will stop passing before you ship the revision.

Exit codes:
  0  no scenarios are affected by breaking changes
  1  one or more scenario steps match a breaking change
  2  spec parse or configuration error`,
		Args: cobra.MinimumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			baseData, baseFmt, err := fetchSpec(args[0])
			if err != nil {
				return err
			}
			revData, revFmt, err := fetchSpec(args[1])
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

			diff, err := openapi.Diff(baseSpec, revSpec)
			if err != nil {
				return fmt.Errorf("diffing specs: %w", err)
			}

			var breaking []openapi.DiffChange
			for _, c := range diff.Changes {
				if c.Breaking && c.Method != "" && c.Path != "" {
					breaking = append(breaking, c)
				}
			}

			scenarioPaths, err := expandScenarioArgs(cmd, args[2:], noRecursive)
			if err != nil {
				return err
			}
			var affected []impactScenario
			for _, path := range scenarioPaths {
				data, err := os.ReadFile(path)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: %v\n", path, err)
					continue
				}
				sc, err := serialization.ParseScenarioFile(data, path)
				if err != nil {
					fmt.Fprintf(cmd.ErrOrStderr(), "warning: skipping %s: %v\n", path, err)
					continue
				}

				var hits []impactStep
				for i, step := range sc.Steps {
					if step.Request == nil {
						continue // requestRef steps need file resolution — skip
					}
					var matched []openapi.DiffChange
					for _, bc := range breaking {
						if strings.EqualFold(string(step.Request.Method), bc.Method) &&
							impactPathMatches(bc.Path, step.Request.URL) {
							matched = append(matched, bc)
						}
					}
					if len(matched) > 0 {
						hits = append(hits, impactStep{
							Index:   i,
							Label:   step.Label,
							Method:  string(step.Request.Method),
							URL:     step.Request.URL,
							Changes: matched,
						})
					}
				}
				if len(hits) > 0 {
					affected = append(affected, impactScenario{
						Path:  path,
						Title: sc.Title,
						Steps: hits,
					})
				}
			}

			var reportBytes []byte
			switch strings.ToLower(format) {
			case "json":
				type jsonChange struct {
					ID   string `json:"id"`
					Text string `json:"text"`
				}
				type jsonStep struct {
					Index   int          `json:"stepIndex"`
					Label   string       `json:"label,omitempty"`
					Method  string       `json:"method"`
					URL     string       `json:"url"`
					Changes []jsonChange `json:"changes"`
				}
				type jsonScenario struct {
					Path  string     `json:"path"`
					Title string     `json:"title,omitempty"`
					Steps []jsonStep `json:"steps"`
				}
				type jsonOut struct {
					BreakingChanges   int            `json:"breakingChanges"`
					AffectedScenarios int            `json:"affectedScenarios"`
					Scenarios         []jsonScenario `json:"scenarios"`
				}
				out := jsonOut{
					BreakingChanges:   len(breaking),
					AffectedScenarios: len(affected),
				}
				for _, a := range affected {
					js := jsonScenario{Path: a.Path, Title: a.Title}
					for _, s := range a.Steps {
						jstep := jsonStep{Index: s.Index, Label: s.Label, Method: s.Method, URL: s.URL}
						for _, c := range s.Changes {
							jstep.Changes = append(jstep.Changes, jsonChange{ID: c.ID, Text: c.Text})
						}
						js.Steps = append(js.Steps, jstep)
					}
					out.Scenarios = append(out.Scenarios, js)
				}
				b, _ := json.MarshalIndent(out, "", "  ")
				reportBytes = append(b, '\n')

			default:
				var sb strings.Builder
				if len(breaking) == 0 {
					fmt.Fprintln(&sb, "No breaking changes — all scenarios unaffected.")
				} else {
					fmt.Fprintf(&sb, "Breaking changes (%d):\n", len(breaking))
					for _, c := range breaking {
						fmt.Fprintf(&sb, "  [%s] %s %s — %s\n", c.Level, c.Method, c.Path, c.Text)
					}
					fmt.Fprintln(&sb)
					if len(affected) == 0 {
						fmt.Fprintln(&sb, "No scenario steps match the breaking change paths.")
					} else {
						fmt.Fprintf(&sb, "Affected scenarios (%d):\n", len(affected))
						for _, a := range affected {
							header := a.Path
							if a.Title != "" {
								header = fmt.Sprintf("%s  (%s)", a.Title, a.Path)
							}
							fmt.Fprintf(&sb, "  %s\n", header)
							for _, s := range a.Steps {
								label := s.Label
								if label == "" {
									label = fmt.Sprintf("step %d", s.Index+1)
								}
								for _, c := range s.Changes {
									fmt.Fprintf(&sb, "    ✗ %-30s  %s %s\n      └─ %s\n",
										label, s.Method, s.URL, c.Text)
								}
							}
						}
						fmt.Fprintln(&sb)
						fmt.Fprintf(&sb, "%d scenario(s) affected — review before shipping the spec revision\n",
							len(affected))
					}
				}
				reportBytes = []byte(sb.String())
			}

			if outputFile != "" {
				if err := os.MkdirAll(filepath.Dir(outputFile), 0o755); err != nil {
					return fmt.Errorf("creating output directory: %w", err)
				}
				if err := os.WriteFile(outputFile, reportBytes, 0o644); err != nil {
					return fmt.Errorf("writing output file: %w", err)
				}
			} else {
				fmt.Fprint(cmd.OutOrStdout(), string(reportBytes))
			}

			if len(affected) > 0 {
				return &exitError{1}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&noRecursive, "no-recursive", false, "Only look at the top level of each folder, not its sub-folders")
	cmd.Flags().StringVarP(&format, "format", "f", "text", "Output format: text or json")
	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Write report to file")
	return cmd
}

type impactScenario struct {
	Path  string
	Title string
	Steps []impactStep
}

type impactStep struct {
	Index   int
	Label   string
	Method  string
	URL     string
	Changes []openapi.DiffChange
}

var placeholderRe = regexp.MustCompile(`\{\{[^}]*\}\}`)

// impactPathMatches returns true if the OpenAPI path pattern (e.g. /users/{id})
// matches the path component of a scenario step URL (e.g. {{baseUrl}}/users/42).
// A placeholder inside the path stands for one path segment.
func impactPathMatches(specPath, scenarioURL string) bool {
	urlPath := impactExtractPath(scenarioURL)
	if urlPath == "" {
		return false
	}
	urlPath = placeholderRe.ReplaceAllString(urlPath, "_")
	// Convert /users/{id} → ^/users/[^/]+$
	escaped := regexp.QuoteMeta(specPath)
	pattern := regexp.MustCompile(`\\\{[^}]+\\\}`).ReplaceAllString(escaped, `[^/]+`)
	matched, _ := regexp.MatchString("^"+pattern+"$", urlPath)
	return matched
}

// impactExtractPath strips the scheme+host (or a leading {{baseUrl}}-style placeholder) from a
// step URL and returns the path, keeping any placeholders inside the path (for example /users/{{id}}).
func impactExtractPath(rawURL string) string {
	rest := rawURL
	if i := strings.IndexAny(rest, "?#"); i >= 0 {
		rest = rest[:i]
	}
	switch {
	case strings.HasPrefix(rest, "http://") || strings.HasPrefix(rest, "https://"):
		rest = rest[strings.Index(rest, "://")+3:]
		slash := strings.IndexByte(rest, '/')
		if slash < 0 {
			return ""
		}
		rest = rest[slash:]
	case strings.HasPrefix(rest, "{{"):
		end := strings.Index(rest, "}}")
		if end < 0 {
			return ""
		}
		rest = rest[end+2:]
	}
	if !strings.HasPrefix(rest, "/") {
		return ""
	}
	return rest
}
