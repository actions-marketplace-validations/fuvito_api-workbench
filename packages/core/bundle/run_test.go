package bundle_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/bundle"
)

func TestMaterialize_InlinesBundleRequestsAndResolvesSpecs(t *testing.T) {
	b := parse(t, good)
	runs := bundle.Materialize(b, bundle.Options{})
	if len(runs) != 2 || runs[0].File != "smoke-test" || runs[1].File != "flows/second" {
		t.Fatalf("runnables: %+v", runs)
	}

	smoke := runs[0].Scenario
	for i, wantURL := range []string{"{{baseUrl}}/health", "{{baseUrl}}/login"} {
		st := smoke.Steps[i]
		if st.Request == nil || st.Request.URL != wantURL || st.RequestRef != "" {
			t.Errorf("step %d should carry its request inline: %+v", i, st)
		}
	}
	if smoke.Steps[0].Request.Timeout.Seconds() != 5 {
		t.Error("request fields such as timeout must come along")
	}
	if smoke.SpecFile != "https://api.example.com/openapi.json" || runs[1].Scenario.SpecFile != "/local/other.yaml" {
		t.Errorf("specs: %q / %q", smoke.SpecFile, runs[1].Scenario.SpecFile)
	}

	over := bundle.Materialize(b, bundle.Options{SpecFile: "C:/spec.yaml"})
	if over[0].Scenario.SpecFile != "C:/spec.yaml" || over[1].Scenario.SpecFile != "C:/spec.yaml" {
		t.Error("an override replaces every scenario's spec")
	}

	if b.Scenarios[0].Scenario.Steps[0].RequestRef != "check-health" || b.Scenarios[0].Scenario.SpecFile != "" {
		t.Error("the parsed bundle must not be modified")
	}
}

func TestMaterialize_ExternalRefsAreLeftAlone(t *testing.T) {
	b := parse(t, "schemaVersion: \"1\"\nscenarios:\n  - title: S\n    steps:\n      - {label: a, requestRef: ../requests/outside.yaml}\n")
	st := bundle.Materialize(b, bundle.Options{})[0].Scenario.Steps[0]
	if st.Request != nil || st.RequestRef != "../requests/outside.yaml" {
		t.Errorf("an unknown ref stays for the caller to resolve: %+v", st)
	}
}

func TestSelectEnvironment(t *testing.T) {
	one := parse(t, good)
	if env, err := bundle.SelectEnvironment(one, ""); err != nil || env.Name != "Staging" {
		t.Errorf("a single environment is the default: %v %v", env, err)
	}
	for _, name := range []string{"Staging", "staging", "STAGING"} {
		if env, err := bundle.SelectEnvironment(one, name); err != nil || env.Name != "Staging" {
			t.Errorf("%q: %v %v", name, env, err)
		}
	}

	two := parse(t, "schemaVersion: \"1\"\nenvironments:\n  - {name: Staging, file: stg, variables: []}\n  - {name: Production, variables: []}\n")
	if _, err := bundle.SelectEnvironment(two, ""); err == nil || !strings.Contains(err.Error(), "Staging, Production") {
		t.Errorf("several environments need a choice: %v", err)
	}
	if env, err := bundle.SelectEnvironment(two, "stg"); err != nil || env.Name != "Staging" {
		t.Errorf("selecting by file name: %v %v", env, err)
	}
	if env, err := bundle.SelectEnvironment(two, "production"); err != nil || env.Name != "Production" {
		t.Errorf("file name derived from the name: %v %v", env, err)
	}
	if _, err := bundle.SelectEnvironment(two, "dev"); err == nil || !strings.Contains(err.Error(), "available: Staging, Production") {
		t.Errorf("unknown name: %v", err)
	}

	none := parse(t, "schemaVersion: \"1\"\nscenarios:\n  - {title: S, steps: [{label: a, request: {method: GET, url: \"http://x\"}}]}\n")
	if env, err := bundle.SelectEnvironment(none, ""); env != nil || err != nil {
		t.Errorf("no environment is fine: %v %v", env, err)
	}
	if _, err := bundle.SelectEnvironment(none, "x"); err == nil || !strings.Contains(err.Error(), "--env-file") {
		t.Errorf("asking for a name when there are none: %v", err)
	}
}

func TestFilterRunnables(t *testing.T) {
	all := bundle.Materialize(parse(t, good), bundle.Options{})
	if kept, un := bundle.FilterRunnables(all, nil); len(kept) != 2 || un != nil {
		t.Errorf("no selector keeps everything: %v %v", kept, un)
	}
	kept, un := bundle.FilterRunnables(all, []string{"SMOKE TEST", "flows/second", "nope"})
	if len(kept) != 2 || len(un) != 1 || un[0] != "nope" {
		t.Errorf("by title and by file, reporting misses: %+v %v", kept, un)
	}
	if kept, _ := bundle.FilterRunnables(all, []string{"smoke-test", "Smoke test"}); len(kept) != 1 {
		t.Errorf("the same scenario selected twice runs once: %d", len(kept))
	}
}
