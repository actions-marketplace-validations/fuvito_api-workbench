package scaffold

import (
	"fmt"
	"regexp"
	"strings"
)

// WorkflowOptions are the values the generated GitHub workflow takes from the project.
type WorkflowOptions struct {
	// Version is the clrsig release the workflow pins (for example v0.9.0). A build that is not a
	// release (dev) pins the main branch and says so.
	Version string
	// Spec is the spec path as seen from the repository root; empty leaves a placeholder.
	Spec string
	// Environment is the environment file for CI; empty leaves environments/ci.yaml.
	Environment string
	// ReadOnly sends only GET and HEAD requests.
	ReadOnly bool
}

var reRelease = regexp.MustCompile(`^v\d+\.\d+\.\d+$`)

// GitHubWorkflow returns a workflow that runs clrsig conform on pull requests and keeps the report.
func GitHubWorkflow(o WorkflowOptions) string {
	ref, pinNote := o.Version, ""
	if !reRelease.MatchString(ref) {
		ref = "main"
		pinNote = "  # this clrsig is not a release build: pin a release tag for reproducible runs"
	}
	spec := o.Spec
	specNote := ""
	if spec == "" {
		spec = "specs/openapi.yaml"
		specNote = "  # the path of your OpenAPI file"
	}
	env := o.Environment
	if env == "" {
		env = "environments/ci.yaml"
	}
	var b strings.Builder
	b.WriteString("name: API conformance\n\n")
	b.WriteString("on:\n  pull_request:\n  workflow_dispatch:\n\n")
	// The job only reads the repository; without this it gets the repository's default token rights.
	b.WriteString("permissions:\n  contents: read\n\n")
	b.WriteString("jobs:\n  conform:\n    runs-on: ubuntu-latest\n    steps:\n")
	b.WriteString("      - uses: actions/checkout@v7\n\n")
	fmt.Fprintf(&b, "      - uses: fuvito/api-workbench@%s%s\n", ref, pinNote)
	b.WriteString("        with:\n")
	b.WriteString("          command: conform\n")
	fmt.Fprintf(&b, "          spec: %s%s\n", yamlScalar(spec), specNote)
	b.WriteString("          base-url: ${{ vars.API_BASE_URL }}   # set under Settings > Secrets and variables > Actions > Variables\n")
	fmt.Fprintf(&b, "          env: %s\n", yamlScalar(env))
	if o.ReadOnly {
		b.WriteString("          read-only: 'true'\n")
	}
	b.WriteString("          report: |\n            junit=reports/conform.xml\n            html=reports/conform.html\n")
	b.WriteString("        env:\n          API_TOKEN: ${{ secrets.API_TOKEN }}   # read as ${API_TOKEN} in environments/ci.yaml\n\n")
	b.WriteString("      - uses: actions/upload-artifact@v7\n        if: always()\n        with:\n          name: conformance-report\n          path: reports/\n")
	return b.String()
}
