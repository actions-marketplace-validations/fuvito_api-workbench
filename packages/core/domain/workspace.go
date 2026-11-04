package domain

// Workspace marks a directory as an EndpointDesk project.
// Collections, scenarios, and environments are discovered by scanning the workspace directory.
// The workspace file itself carries only identity metadata.
type Workspace struct {
	SchemaVersion string
	Name          string
	Description   string
}

func NewWorkspace(name string) *Workspace {
	return &Workspace{
		SchemaVersion: CurrentSchemaVersion,
		Name:          name,
	}
}
