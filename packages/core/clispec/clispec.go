// Package clispec is the machine-readable description of the clrsig command line:
// which commands exist, what each input means and what kind of value it takes, what a
// command produces and what its exit codes mean.
//
// The desktop CLI Command Builder and the documentation read this catalog. A test in
// apps/cli compares it with the real cobra command tree, so the two cannot drift apart.
package clispec

// Kind says what sort of value an input takes, so a UI can offer the right picker and a
// builder can validate it.
type Kind string

const (
	KindFile       Kind = "file"       // an existing local file
	KindFileOrURL  Kind = "fileOrURL"  // an existing local file or an http(s) URL
	KindDir        Kind = "dir"        // an existing folder
	KindPath       Kind = "path"       // an existing file or folder
	KindOutputFile Kind = "outputFile" // a file the command will create
	KindURL        Kind = "url"        // an http(s) URL
	KindEnum       Kind = "enum"       // one of a fixed set of values
	KindInt        Kind = "int"
	KindBool       Kind = "bool"
	KindString     Kind = "string"
)

// Role is what a file means in Clear Signal, used to list the right files in a picker.
type Role string

const (
	RoleNone        Role = ""
	RoleScenario    Role = "scenario"
	RoleEnvironment Role = "environment"
	RoleSpec        Role = "spec"
	RoleWorkspace   Role = "workspace"
)

// Group separates the inputs a user nearly always sets from the optional ones.
type Group string

const (
	GroupEssential Group = "essential"
	GroupAdvanced  Group = "advanced"
)

// Requirement says an input only makes sense when another input is also given.
// Or names an implicit alternative; the only one today is "scenario.specFile", meaning a
// specFile: written inside the scenario file also satisfies the requirement.
type Requirement struct {
	Input string
	Or    string
}

// ImplicitScenarioSpec is the Requirement.Or value for a spec named inside the scenario.
const ImplicitScenarioSpec = "scenario.specFile"

// Input is one positional argument or one flag of a command.
type Input struct {
	// Name is the long flag name without dashes ("env"), or the argument name ("scenario-file").
	Name  string
	Short string // one-letter flag, empty if none

	Positional bool
	Required   bool
	// Repeatable inputs may be given several times (--var a=1 --var b=2). The builder does not
	// offer them yet, so they are also Deferred.
	Repeatable bool

	Kind    Kind
	Role    Role
	Accepts []string // accepted file extensions for file kinds, e.g. ".yaml"
	Enum    []string // allowed values for KindEnum
	Default string   // the CLI default as text; empty when the flag has none

	Label   string // short field label for a form
	Help    string // plain-language explanation of what the input is and what is expected
	Example string

	Requires []Requirement
	Group    Group

	// Deferred inputs are part of the real CLI (and the parity test) but are not yet
	// offered by the builder UI.
	Deferred bool
}

// ExitCode is one process exit status and what it means.
type ExitCode struct {
	Code    int
	Meaning string
	ForCI   string // what a pipeline does with it
}

// Command describes one clrsig command.
type Command struct {
	Name    string // as typed after clrsig, e.g. "run"
	Goal    string // what the user wants to do, in plain words
	Summary string // one or two sentences on what the command does
	When    string // when to use it

	Inputs    []Input
	Outputs   []string // what the command prints or writes
	ExitCodes []ExitCode
}

// Input returns the named input and whether it exists.
func (c Command) Input(name string) (Input, bool) {
	for _, in := range c.Inputs {
		if in.Name == name {
			return in, true
		}
	}
	return Input{}, false
}

// Positionals returns the positional arguments in order.
func (c Command) Positionals() []Input {
	var out []Input
	for _, in := range c.Inputs {
		if in.Positional {
			out = append(out, in)
		}
	}
	return out
}

// Flags returns the flag inputs in catalog order.
func (c Command) Flags() []Input {
	var out []Input
	for _, in := range c.Inputs {
		if !in.Positional {
			out = append(out, in)
		}
	}
	return out
}

// ExitCodeMeaning returns the catalog entry for code, or false if it is not documented.
func (c Command) ExitCodeMeaning(code int) (ExitCode, bool) {
	for _, e := range c.ExitCodes {
		if e.Code == code {
			return e, true
		}
	}
	return ExitCode{}, false
}
