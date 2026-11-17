package scenario_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/scenario"
)

func TestResolveRequestRefs(t *testing.T) {
	root := t.TempDir()
	scnDir := filepath.Join(root, "scenarios")
	reqDir := filepath.Join(root, "requests")
	os.MkdirAll(scnDir, 0o755)
	os.MkdirAll(filepath.Join(reqDir, "auth"), 0o755)
	os.WriteFile(filepath.Join(reqDir, "ping.yaml"), []byte("method: GET\nurl: http://x/ping\n"), 0o644)
	os.WriteFile(filepath.Join(reqDir, "auth", "login.yaml"), []byte("method: POST\nurl: http://x/login\n"), 0o644)
	os.WriteFile(filepath.Join(scnDir, "local.yaml"), []byte("method: GET\nurl: http://x/local\n"), 0o644)
	os.WriteFile(filepath.Join(reqDir, "broken.yaml"), []byte("method: ["), 0o644)

	sc := &domain.Scenario{Steps: []domain.ScenarioStep{
		{Label: "relative to scenario", RequestRef: "../requests/ping.yaml"},
		{Label: "relative to requests", RequestRef: "auth/login.yaml"},
		{Label: "next to scenario", RequestRef: "local.yaml"},
		{Label: "absolute", RequestRef: filepath.Join(reqDir, "ping.yaml")},
		{Label: "inline wins", Request: &domain.Request{Method: domain.MethodGET, URL: "http://inline"}, RequestRef: "../requests/ping.yaml"},
		{Label: "no ref", Request: &domain.Request{Method: domain.MethodGET, URL: "http://x"}},
		{Label: "missing", RequestRef: "nope.yaml"},
		{Label: "unparseable", RequestRef: "broken.yaml"},
	}}
	unresolved := scenario.ResolveRequestRefs(sc, scnDir, reqDir)

	wantURL := map[int]string{0: "http://x/ping", 1: "http://x/login", 2: "http://x/local", 3: "http://x/ping", 4: "http://inline"}
	for i, want := range wantURL {
		if sc.Steps[i].Request == nil || sc.Steps[i].Request.URL != want {
			t.Errorf("step %d (%s): %+v", i, sc.Steps[i].Label, sc.Steps[i].Request)
		}
	}
	if sc.Steps[6].Request != nil || sc.Steps[7].Request != nil {
		t.Error("unresolvable refs must not get a request")
	}
	if len(unresolved) != 2 || unresolved[0] != "nope.yaml" || unresolved[1] != "broken.yaml" {
		t.Errorf("unresolved: %v", unresolved)
	}
	if sc.Steps[0].RequestRef != "../requests/ping.yaml" {
		t.Error("the ref itself is left in place")
	}

	if got := scenario.ResolveRequestRefs(&domain.Scenario{Steps: []domain.ScenarioStep{{Label: "x", RequestRef: "ping.yaml"}}}); len(got) != 1 {
		t.Errorf("with no directories nothing can resolve: %v", got)
	}
	if got := scenario.ResolveRequestRefs(&domain.Scenario{Steps: []domain.ScenarioStep{{Label: "x", RequestRef: "ping.yaml"}}}, "", reqDir); len(got) != 0 {
		t.Errorf("empty directories are ignored: %v", got)
	}
}
