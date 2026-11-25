package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/fuvito/api-workbench/packages/core/scenario"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

// People copy the example scenarios, so each one must pass clrsig validate and hold no key the
// format does not have. A scenario file with an unknown key still loads and runs: ten of these once
// named their steps with title: instead of label:, and the names were silently dropped.
func TestExampleScenarioFilesAreValid(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "scenarios", "*.yaml"))
	if err != nil || len(files) < 10 {
		t.Fatalf("example scenarios: %v %v", files, err)
	}
	for _, file := range files {
		t.Run(filepath.Base(file), func(t *testing.T) {
			data, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			strict := yaml.NewDecoder(bytes.NewReader(data))
			strict.KnownFields(true)
			if err := strict.Decode(&serialization.ScenarioFile{}); err != nil {
				t.Errorf("a key the scenario format does not have: %v", err)
			}
			sc, err := serialization.ParseScenarioFile(data, file)
			if err != nil {
				t.Fatal(err)
			}
			for _, issue := range scenario.Validate(sc, filepath.Dir(file)) {
				t.Errorf("%+v", issue)
			}
		})
	}
}
