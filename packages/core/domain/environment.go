package domain

const CurrentSchemaVersion = "1"

// Environment holds named variables that are substituted into requests at runtime.
// Switching environments changes variable values without modifying request definitions.
type Environment struct {
	SchemaVersion string
	Name          string
	Variables     []EnvironmentVariable
}

type EnvironmentVariable struct {
	Name   string
	Value  string
	Secret bool // redacted in UI, logs, and reports; value should reference an env var in CI/CD
}

func NewEnvironment(name string) *Environment {
	return &Environment{
		SchemaVersion: CurrentSchemaVersion,
		Name:          name,
	}
}

// Lookup returns the value for the given variable name and whether it was found.
func (e *Environment) Lookup(name string) (string, bool) {
	for _, v := range e.Variables {
		if v.Name == name {
			return v.Value, true
		}
	}
	return "", false
}
