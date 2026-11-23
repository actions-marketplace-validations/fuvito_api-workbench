package clispec

var specExts = []string{".yaml", ".yml", ".json"}

var reportFormats = []string{"text", "json", "junit", "html", "markdown"}

const (
	formatHelp = "How the report is written. text is for people reading a terminal or a CI log; " +
		"json is for scripts; junit is for CI test dashboards and pull-request annotations; " +
		"html is a single page you can open in a browser or keep as a build artifact; " +
		"markdown is for pull-request comments and the GitHub job summary."
	reportHelp = "name=file, repeatable: also write a report in another format from the same run, " +
		"for example junit=reports/run.xml, so you do not have to run twice."
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
				Name: "seed", Kind: KindInt, Default: "0",
				Label: "Seed for random values",
				Help: "Random and dynamic values ({{$randomInt 1 9}}, {{$guid}}, …) are generated from a seed. Leave this at 0 for a " +
					"fresh random seed each run (it is printed in the report); give a number to replay a run exactly.",
				Example: "12345",
				Group:   GroupAdvanced,
			},
			{
				Name: "show-extracted", Kind: KindBool, Default: "false",
				Label: "Show secret-looking extracted values",
				Help: "Reports hide extracted values whose names look secret (token, password, key, session …) and any " +
					"secret environment value. Turn this on only to debug locally; the values then appear in the report.",
				Group: GroupAdvanced,
			},
			{
				Name: "var", Kind: KindString, Repeatable: true,
				Label: "Variable overrides",
				Help: "name=value, repeatable: set or override one environment variable for this run only. " +
					"Do not put secrets here; command lines are visible to other processes. Use environment variables for secrets.",
				Example: "baseUrl=https://staging.example.com",
				Group:   GroupAdvanced, Deferred: true,
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
			{
				Name: "report", Kind: KindString, Repeatable: true,
				Label: "Also save as", Help: reportHelp, Example: "junit=reports/run.xml",
				Group: GroupAdvanced, Deferred: true,
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
		Goal:    "Spec Conformance Testing: check an API against its OpenAPI spec",
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
				Name: "var", Kind: KindString, Repeatable: true,
				Label: "Variable overrides",
				Help: "name=value, repeatable: set or override one environment variable for this run only. " +
					"Do not put secrets here; command lines are visible to other processes. Use environment variables for secrets.",
				Example: "token=abc",
				Group:   GroupAdvanced, Deferred: true,
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
				Name: "explain", Kind: KindBool, Default: "false",
				Label: "Explain failures",
				Help: "Under each failing operation, say in plain language what the spec expected, what the server sent, why that fails and what to change. " +
					"The JSON, HTML, Markdown and JUnit reports always include these explanations; the screen output only with this option, so existing logs do not change.",
				Group: GroupAdvanced,
			},
			{
				Name: "read-only", Kind: KindBool, Default: "false",
				Label: "Read requests only",
				Help: "Send only GET and HEAD requests. POST, PUT, PATCH and DELETE operations are skipped and never sent, " +
					"so the check cannot create, change or delete data. Use it against a shared or production-like server.",
				Group: GroupEssential,
			},
			{
				Name: "baseline", Kind: KindString,
				Label:   "Accept recorded violations",
				Help:    "Violations already listed in this baseline file are accepted; only new ones fail the run, and fixed ones are reported.",
				Example: "conform-baseline.yaml",
				Group:   GroupAdvanced, Deferred: true,
			},
			{
				Name: "write-baseline", Kind: KindString,
				Label:   "Record violations as a baseline",
				Help:    "Write today's violations to this file and exit 0, so a long-lived API can adopt the check and improve gradually.",
				Example: "conform-baseline.yaml",
				Group:   GroupAdvanced, Deferred: true,
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
				Name: "max-response-time", Kind: KindInt, Default: "0",
				Label:   "Slowest acceptable response (ms)",
				Help:    "Fail an operation whose response takes longer than this. The response is still compared with the spec. 0 means no limit. Limits are wall-clock: leave a margin.",
				Example: "500",
				Group:   GroupAdvanced,
			},
			{
				Name: "max-response-time-for", Kind: KindString, Repeatable: true,
				Label:   "Response-time limit for a path (ms)",
				Help:    "/prefix=ms, repeatable: a different limit for the operations under a path prefix. The longest matching prefix wins over the general limit; 0 turns the limit off there.",
				Example: "/reports=2000",
				Group:   GroupAdvanced, Deferred: true,
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
			{
				Name: "report", Kind: KindString, Repeatable: true,
				Label: "Also save as", Help: reportHelp, Example: "junit=reports/conform.xml",
				Group: GroupAdvanced, Deferred: true,
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
				Name: "fail-on", Kind: KindEnum, Enum: []string{"breaking", "warning", "info"}, Default: "breaking",
				Label: "Fail when a change is at least",
				Help: "How serious a change must be before the command fails. breaking (the default) fails only on " +
					"breaking changes; warning also fails on potentially breaking ones; info fails on any change. " +
					"It cannot be combined with Only breaking changes.",
				Group: GroupEssential,
			},
			{
				Name: "against", Kind: KindString,
				Label: "Compare against a git ref",
				Help: "Give one spec file and compare it with its version at this git ref (for example origin/main), " +
					"read through git. Needs git and a repository.",
				Example: "origin/main",
				Group:   GroupAdvanced, Deferred: true,
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
			{
				Name: "report", Kind: KindString, Repeatable: true,
				Label: "Also save as", Help: reportHelp, Example: "junit=reports/diff.xml",
				Group: GroupAdvanced, Deferred: true,
			},
		},
		Outputs: []string{
			"A list of changes, each marked ERR (breaking), WARN or INFO, on the screen (or in the file you choose).",
		},
		ExitCodes: []ExitCode{
			{0, "No change at or above the --fail-on level (breaking by default).", "The pipeline step succeeds: safe to ship."},
			{1, "One or more changes at or above the --fail-on level (breaking by default).", "The pipeline step fails: review before shipping."},
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
