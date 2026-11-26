package main

import "testing"

// Pins CS-GAP-027: only the project file and a baseline refuse a schemaVersion they do not know.
func TestValidate_AnUnsupportedSchemaVersionIsAccepted_KnownGap(t *testing.T) {
	file := writeTemp(t, "v.yaml", "schemaVersion: \"9\"\ntitle: v\nsteps:\n  - label: a\n    request: {method: GET, url: \"http://localhost:1\"}\n    assertions:\n      - {type: status, operator: eq, expected: 200}\n")
	if err := runQuietly(t, "validate", file); err != nil {
		t.Errorf("validate reports it now (%v): update CS-GAP-027, troubleshooting.md, and close B-150", err)
	}
}
