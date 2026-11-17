package clispec

var specExts = []string{".yaml", ".yml", ".json"}

var reportFormats = []string{"text", "json", "junit", "html"}

const (
	formatHelp = "How the report is written. text is for people reading a terminal or a CI log; " +
		"json is for scripts; junit is for CI test dashboards and pull-request annotations; " +
		"html is a single page you can open in a browser or keep as a build artifact."
	outputHelp = "Write the report to this file instead of the screen. Missing folders are created. " +
		"Nothing is written to disk unless you set this."
	envHelp = "Variables used by {{placeholders}} in your requests, such as baseUrl and tokens. " +
		"Values written as ${NAME} are read from the machine's environment variables when the command runs."
)

var catalog = []Command{
	{
		Name:    "run",
		Goal:    "Run scenarios (a file or a whole folder)",
		Summary: "Runs scenario files step by step: sends each request, extracts values, and checks every assertion. Give it one file, several, or a folder to run everything inside it.",
		When:    "To check a workflow such as login → create → fetch → delete, or a whole suite, locally or as a CI gate.",
		Inputs: []Input{
			{
				// The name is kept for compatibility; the input takes a file or a folder.
				Name: "scenario-file", Positional: true, Required: true,
				Kind: KindPath, Role: RoleScenario, Accepts: specExts,
				Label: "Scenario or folder",
				Help: "A scenario file, or a folder to run every scenario inside it (and in its sub-folders, unless " +
					"you turn that off). All scenarios are read first, and nothing runs if one of them is broken. " +
					"With several scenarios you get one combined report and one exit code. Other files in the folder " +
					"(environments, requests, specs) are skipped and listed.",
				Example: "scenarios/",
				Group:   GroupEssential,
			},
			{
				Name: "env", Short: "e",
				Kind: KindFile, Role: RoleEnvironment, Accepts: specExts,
				Label:   "Environment",
				Help:    envHelp,
				Example: "environments/ci.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "spec", Short: "s",
				Kind: KindFileOrURL, Role: RoleSpec, Accepts: specExts,
				Label: "OpenAPI spec",
				Help: "The OpenAPI document to validate responses against. A relative path is relative to the folder " +
					"you run the command from. A specFile: written inside the scenario is relative to the scenario file, " +
					"and this option overrides it.",
				Example: "openapi.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "coverage", Kind: KindBool, Default: "false",
				Label:    "Coverage report",
				Help:     "After the run, list which operations of the spec the scenario exercised and which it never reached.",
				Requires: []Requirement{{Input: "spec", Or: ImplicitScenarioSpec}},
				Group:    GroupEssential,
			},
			{
				Name: "workspace", Short: "w",
				Kind: KindDir, Role: RoleWorkspace,
				Label: "Accumulate coverage in workspace",
				Help: "Record this run's coverage in a workspace so it appears on the desktop Contracts page and in " +
					"clrsig coverage. Coverage builds up across runs.",
				Example:  ".",
				Requires: []Requirement{{Input: "spec", Or: ImplicitScenarioSpec}},
				Group:    GroupAdvanced,
			},
			{
				Name: "fail-fast", Kind: KindBool, Default: "false",
				Label: "Stop at the first failing scenario",
				Help:  "When running several scenarios, stop after the first one that fails instead of running them all.",
				Group: GroupAdvanced,
			},
			{
				Name: "no-recursive", Kind: KindBool, Default: "false",
				Label: "Do not search sub-folders",
				Help:  "When you give a folder, only run the scenarios directly inside it, not those in its sub-folders.",
				Group: GroupAdvanced,
			},
			{
				Name: "timeout", Kind: KindInt, Default: "0",
				Label:   "Step timeout (ms)",
				Help:    "Give every step this many milliseconds before it is abandoned. 0 keeps each step's own timeout.",
				Example: "10000",
				Group:   GroupAdvanced,
			},
			{
				Name: "format", Short: "f", Kind: KindEnum, Enum: reportFormats, Default: "text",
				Label: "Report format", Help: formatHelp, Group: GroupEssential,
			},
			{
				Name: "output", Short: "o", Kind: KindOutputFile,
				Label: "Save report to", Help: outputHelp, Example: "reports/run.html",
				Group: GroupEssential,
			},
		},
		Outputs: []string{
			"A report of every step and assertion, on the screen (or in the file you choose). With several scenarios, one combined report with an overall summary.",
			"With a workspace: coverage recorded for the desktop Contracts page, scenario by scenario.",
		},
		ExitCodes: []ExitCode{
			{0, "Every step of every scenario passed every assertion.", "The pipeline step succeeds."},
			{1, "At least one assertion failed in at least one scenario.", "The pipeline step fails: the API no longer behaves as the scenarios expect."},
			{2, "The run could not happen: a file could not be read or parsed (so nothing ran), nothing was found in a folder, or a configuration error.", "The pipeline step fails: fix the command or the files before trusting any result."},
		},
	},
	{
		Name:    "conform",
		Goal:    "Check an API against its OpenAPI spec",
		Summary: "Sends a request to every operation in an OpenAPI spec and checks each response against the schema the spec declares.",
		When:    "To prove a running API matches its spec with no test files to write.",
		Inputs: []Input{
			{
				Name: "spec", Positional: true, Required: true,
				Kind: KindFileOrURL, Role: RoleSpec, Accepts: specExts,
				Label:   "OpenAPI spec",
				Help:    "The OpenAPI 3.0/3.1 (or Swagger 2.0) document, as a file or an http(s) address.",
				Example: "openapi.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "base-url", Short: "b", Kind: KindURL,
				Label:   "Server address",
				Help:    "Where the API is running. Overrides the first server listed in the spec.",
				Example: "http://localhost:8080",
				Group:   GroupEssential,
			},
			{
				Name: "env", Short: "e",
				Kind: KindFile, Role: RoleEnvironment, Accepts: specExts,
				Label:   "Environment",
				Help:    envHelp + " Use it to supply an auth token for secured operations.",
				Example: "environments/ci.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "coverage", Kind: KindBool, Default: "false",
				Label: "Coverage summary",
				Help:  "Append a summary of how many operations received a response.",
				Group: GroupEssential,
			},
			{
				Name: "negative", Kind: KindBool, Default: "false",
				Label: "Negative probes",
				Help: "Also send deliberately wrong requests (missing required fields, wrong types, no credentials, " +
					"missing parameters) and check the API rejects them with a 4xx.",
				Group: GroupAdvanced,
			},
			{
				Name: "fail-fast", Kind: KindBool, Default: "false",
				Label: "Stop at the first failure",
				Help:  "Stop after the first violation or execution error instead of checking every operation.",
				Group: GroupAdvanced,
			},
			{
				Name: "tags", Kind: KindString,
				Label:   "Only these tags",
				Help:    "Only check operations that carry one of these OpenAPI tags. Separate several with commas.",
				Example: "users,orders",
				Group:   GroupAdvanced,
			},
			{
				Name: "paths", Kind: KindString,
				Label:   "Only these path prefixes",
				Help:    "Only check operations whose path starts with one of these prefixes. Separate several with commas.",
				Example: "/users,/orders",
				Group:   GroupAdvanced,
			},
			{
				Name: "request-timeout", Kind: KindInt, Default: "10000",
				Label:   "Request timeout (ms)",
				Help:    "How long to wait for each request before giving up.",
				Example: "10000",
				Group:   GroupAdvanced,
			},
			{
				Name: "format", Short: "f", Kind: KindEnum, Enum: reportFormats, Default: "text",
				Label: "Report format", Help: formatHelp, Group: GroupEssential,
			},
			{
				Name: "output", Short: "o", Kind: KindOutputFile,
				Label: "Save report to", Help: outputHelp, Example: "reports/conform.html",
				Group: GroupEssential,
			},
		},
		Outputs: []string{
			"A pass/fail line per operation with the violations found, on the screen (or in the file you choose).",
			"The run is also recorded in your local history so coverage can build up across runs.",
			"With negative probes: a second section showing which invalid requests the API wrongly accepted.",
		},
		ExitCodes: []ExitCode{
			{0, "Every operation conformed to the spec.", "The pipeline step succeeds."},
			{1, "At least one response broke the spec, a request failed, or a negative probe was accepted.", "The pipeline step fails: the API and its spec disagree."},
			{2, "The spec could not be read or parsed, or a configuration error.", "The pipeline step fails: fix the command or the spec first."},
		},
	},
	{
		Name:    "diff",
		Goal:    "Detect breaking changes between two specs",
		Summary: "Compares two versions of an OpenAPI spec and classifies every change as breaking, potentially breaking, or informational.",
		When:    "Before shipping a new API version, to see what would break existing clients.",
		Inputs: []Input{
			{
				Name: "base-spec", Positional: true, Required: true,
				Kind: KindFileOrURL, Role: RoleSpec, Accepts: specExts,
				Label:   "Current spec",
				Help:    "The spec clients use today (the baseline).",
				Example: "openapi-v1.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "revision-spec", Positional: true, Required: true,
				Kind: KindFileOrURL, Role: RoleSpec, Accepts: specExts,
				Label:   "New spec",
				Help:    "The spec you are about to ship. It is compared against the current one.",
				Example: "openapi-v2.yaml",
				Group:   GroupEssential,
			},
			{
				Name: "breaking-only", Kind: KindBool, Default: "false",
				Label: "Only breaking changes",
				Help:  "Hide informational and potentially-breaking changes.",
				Group: GroupEssential,
			},
			{
				Name: "format", Short: "f", Kind: KindEnum, Enum: reportFormats, Default: "text",
				Label: "Report format", Help: formatHelp, Group: GroupEssential,
			},
			{
				Name: "output", Short: "o", Kind: KindOutputFile,
				Label: "Save report to", Help: outputHelp, Example: "reports/diff.html",
				Group: GroupEssential,
			},
		},
		Outputs: []string{
			"A list of changes, each marked ERR (breaking), WARN or INFO, on the screen (or in the file you choose).",
		},
		ExitCodes: []ExitCode{
			{0, "No breaking changes.", "The pipeline step succeeds: safe to ship."},
			{1, "One or more breaking changes.", "The pipeline step fails: review before shipping."},
			{2, "A spec could not be read or parsed, or a configuration error.", "The pipeline step fails: fix the command or the specs first."},
		},
	},
}

// All returns the catalogued commands.
func All() []Command {
	out := make([]Command, len(catalog))
	copy(out, catalog)
	return out
}

// Get returns the named command.
func Get(name string) (Command, bool) {
	for _, c := range catalog {
		if c.Name == name {
			return c, true
		}
	}
	return Command{}, false
}
