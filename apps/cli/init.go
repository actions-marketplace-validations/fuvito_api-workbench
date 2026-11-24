package main

import (
	"errors"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/fuvito/api-workbench/packages/core/openapi"
	"github.com/fuvito/api-workbench/packages/core/projectconfig"
	"github.com/fuvito/api-workbench/packages/core/scaffold"
)

func initCmd() *cobra.Command {
	var specFlag, baseURL string
	var dryRun bool

	cmd := &cobra.Command{
		Use:   "init [folder]",
		Short: "Create a clrsig.yaml and the starter folders for a project",
		Long: `Create what a project needs to run clrsig with no flags: a project file (clrsig.yaml), a local and
a CI environment file, and empty scenarios/, requests/ and specs/ folders.

init only creates what is missing. It never overwrites or deletes: a file or folder that already
exists is left as it is and listed as kept, so it is safe to run again. Nothing is fetched from the
network. With --dry-run it prints what it would create and writes nothing.

To get a GitHub workflow as well, run: clrsig init github-actions`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			folder := "."
			if len(args) == 1 {
				folder = args[0]
			}
			absFolder, err := filepath.Abs(folder)
			if err != nil {
				return err
			}
			if st, err := os.Stat(absFolder); err == nil && !st.IsDir() {
				return fmt.Errorf("%s is a file, not a folder", folder)
			}
			if baseURL != "" {
				u, err := url.Parse(baseURL)
				if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
					return fmt.Errorf("--base-url %q must be a full http or https address, for example http://localhost:8080", baseURL)
				}
			}
			specValue, err := specForProjectFile(specFlag, absFolder)
			if err != nil {
				return err
			}

			entries := scaffold.ProjectEntries(scaffold.ProjectOptions{Spec: specValue, BaseURL: baseURL})
			out, err := scaffold.Apply(absFolder, entries, dryRun)
			if err != nil {
				return err
			}
			w := cmd.OutOrStdout()
			where := folder
			if folder == "." {
				where = "this folder"
			}
			if dryRun {
				fmt.Fprintf(w, "Dry run: nothing was written. In %s, clrsig init would:\n", where)
			}
			printList(w, createdHeading(dryRun, where), out.Created)
			printList(w, "Kept (already there, not changed):", out.Kept)
			if len(out.Created) == 0 && !dryRun {
				fmt.Fprintln(w, "\nNothing to create: this folder is already set up.")
			}
			fmt.Fprintln(w, "\nNext:")
			if folder != "." {
				fmt.Fprintf(w, "  cd %s\n", folder)
			}
			switch {
			case specValue != "":
				fmt.Fprintln(w, "  clrsig conform                  # checks the API against the spec, with the values from clrsig.yaml")
				if baseURL == "" {
					fmt.Fprintln(w, "  (no --base-url was given: conform uses the first servers entry of the spec, or set baseUrl: in clrsig.yaml)")
				}
			default:
				fmt.Fprintln(w, "  put your OpenAPI file in specs/ and set spec: in clrsig.yaml, then run: clrsig conform")
				fmt.Fprintln(w, "  or create request files from it: clrsig generate <spec>")
			}
			fmt.Fprintln(w, "  clrsig init github-actions      # prints a CI workflow (add --write to save it)")
			fmt.Fprintln(w, "\nTip: reports go to reports/ in the generated workflow; add it to .gitignore if you do not want to commit them.")
			return nil
		},
	}
	cmd.Flags().StringVar(&specFlag, "spec", "", "OpenAPI spec file or URL to write into clrsig.yaml (a file is checked to parse; a URL is not fetched)")
	cmd.Flags().StringVar(&baseURL, "base-url", "", "Server to check, written into clrsig.yaml and environments/local.yaml (default http://localhost:8080 in the environment)")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Print what would be created and write nothing")

	cmd.AddCommand(initGitHubActionsCmd())
	return cmd
}

func createdHeading(dryRun bool, where string) string {
	if dryRun {
		return "Would create:"
	}
	return "Created in " + where + ":"
}

func printList(w interface{ Write([]byte) (int, error) }, heading string, items []string) {
	if len(items) == 0 {
		return
	}
	fmt.Fprintln(w, heading)
	for _, it := range items {
		fmt.Fprintf(w, "  %s\n", it)
	}
}

// specForProjectFile turns the --spec value into what clrsig.yaml should say: a URL as it is, a
// local file checked to parse and written relative to the project folder.
func specForProjectFile(spec, absFolder string) (string, error) {
	if spec == "" || strings.Contains(spec, "://") {
		return spec, nil
	}
	absSpec, err := filepath.Abs(spec)
	if err != nil {
		return "", err
	}
	data, format, err := fetchSpec(absSpec)
	if err != nil {
		return "", fmt.Errorf("--spec %s: %w", spec, err)
	}
	if _, err := openapi.ParseBytes(data, format); err != nil {
		return "", fmt.Errorf("--spec %s: %w", spec, err)
	}
	if rel, err := filepath.Rel(absFolder, absSpec); err == nil {
		return filepath.ToSlash(rel), nil
	}
	return filepath.ToSlash(absSpec), nil
}

func initGitHubActionsCmd() *cobra.Command {
	var write bool
	var output string

	cmd := &cobra.Command{
		Use:   "github-actions",
		Short: "Print a GitHub Actions workflow that runs clrsig conform",
		Long: `Print a ready workflow for GitHub Actions. It runs clrsig conform on pull requests with the spec and
environment from the project file (clrsig.yaml) when there is one, and keeps the reports as an artifact.

By default the workflow is only printed. --write saves it to .github/workflows/clrsig.yml (or the
file given with --output) and refuses to replace a file that exists. Run it from the repository root.`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if output != "" && !write {
				return fmt.Errorf("--output needs --write")
			}
			cwd, err := os.Getwd()
			if err != nil {
				return err
			}
			opts := scaffold.WorkflowOptions{Version: version, ReadOnly: true}
			home, _ := os.UserHomeDir()
			path, err := projectconfig.Discover(cwd, home)
			if err != nil {
				return err
			}
			if path != "" {
				cfg, err := projectconfig.Load(path)
				if err != nil {
					return fmt.Errorf("project file: %w", err)
				}
				opts.Spec = cfg.ForCommandLine(cfg.Spec, cwd)
				env := cfg.Environment
				if _, err := os.Stat(filepath.Join(cfg.Dir(), "environments", "ci.yaml")); err == nil {
					env = "environments/ci.yaml" // the CI file never holds a local secret
				}
				opts.Environment = cfg.ForCommandLine(env, cwd)
				if cfg.Conform != nil && cfg.Conform.ReadOnly != nil {
					opts.ReadOnly = *cfg.Conform.ReadOnly
				}
			}
			text := scaffold.GitHubWorkflow(opts)
			if !write {
				fmt.Fprint(cmd.OutOrStdout(), text)
				return nil
			}
			dest := output
			if dest == "" {
				dest = filepath.Join(".github", "workflows", "clrsig.yml")
			}
			if err := scaffold.WriteNew(dest, text); err != nil {
				if errors.Is(err, fs.ErrExist) {
					return fmt.Errorf("%s already exists and was not changed: choose another name with --output, or print the workflow without --write and merge it by hand", dest)
				}
				return err
			}
			w := cmd.OutOrStdout()
			fmt.Fprintf(w, "Wrote %s\n\nBefore it can run, in the repository settings:\n", filepath.ToSlash(dest))
			fmt.Fprintln(w, "  Variables: API_BASE_URL (the server to check)")
			fmt.Fprintln(w, "  Secrets:   API_TOKEN    (only if the API needs one; read as ${API_TOKEN} in environments/ci.yaml)")
			return nil
		},
	}
	cmd.Flags().BoolVar(&write, "write", false, "Save the workflow to .github/workflows/clrsig.yml instead of printing it")
	cmd.Flags().StringVar(&output, "output", "", "With --write: the file to create instead of .github/workflows/clrsig.yml")
	return cmd
}
