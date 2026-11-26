package scaffold_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/scaffold"
	"github.com/fuvito/api-workbench/packages/core/serialization"
)

func environmentFile(t *testing.T, o scaffold.ProjectOptions, path string) string {
	t.Helper()
	for _, e := range scaffold.ProjectEntries(o) {
		if e.Path == path {
			return e.Content
		}
	}
	t.Fatalf("no entry %s", path)
	return ""
}

// variablesOf parses an environment file and returns name -> value.
func variablesOf(t *testing.T, content string) map[string]string {
	t.Helper()
	env, err := serialization.ParseEnvironment([]byte(content), "yaml")
	if err != nil {
		t.Fatalf("%v\n%s", err, content)
	}
	out := map[string]string{}
	for _, v := range env.Variables {
		out[v.Name] = v.Value
		if v.Secret {
			out[v.Name+" (secret)"] = ""
		}
	}
	return out
}

// uncomment does what the file tells the reader to do: remove the # from a variable's lines.
func uncomment(content string) string {
	return strings.ReplaceAll(strings.ReplaceAll(content, "  #- ", "  - "), "  #  ", "    ")
}

var needed = []scaffold.NeededValue{
	{Name: "id", UsedBy: []string{"GET /a/{id}", "PUT /a/{id}", "DELETE /a/{id}", "GET /b/{id}", "GET /c/{id}"}},
	{Name: "token", Secret: true, UsedBy: []string{"GET /a/{id}"}},
	{Name: "apiKey", Secret: true},
	{Name: "X-Trace: id #1"},
}

// As written, nothing new is defined: a defined but empty value would be sent as empty.
func TestNeededValuesAreCommentsUntilUncommented(t *testing.T) {
	o := scaffold.ProjectOptions{Spec: "s.yaml", Needed: needed}
	local := environmentFile(t, o, "environments/local.yaml")
	if got := variablesOf(t, local); len(got) != 1 || got["baseUrl"] == "" {
		t.Fatalf("only baseUrl is defined: %v\n%s", got, local)
	}
	if !strings.Contains(local, "# used by: GET /a/{id}, PUT /a/{id}, DELETE /a/{id} and 2 more") {
		t.Errorf("a long list is cut:\n%s", local)
	}
	got := variablesOf(t, uncomment(local))
	for _, want := range []string{"id", "token", "token (secret)", "apiKey", "apiKey (secret)", "X-Trace: id #1"} {
		if _, ok := got[want]; !ok {
			t.Errorf("after uncommenting, %q is missing: %v\n%s", want, got, uncomment(local))
		}
	}

	ci := environmentFile(t, o, "environments/ci.yaml")
	if got := variablesOf(t, ci); len(got) != 3 { // baseUrl, token and its secret mark
		t.Fatalf("ci defines what it always did: %v\n%s", got, ci)
	}
	if strings.Count(ci, "name: token") != 1 {
		t.Errorf("token is already defined in ci.yaml:\n%s", ci)
	}
	if got := variablesOf(t, uncomment(ci)); got["apiKey"] != "${API_KEY}" || got["id"] != "" {
		t.Errorf("a secret reads a process variable in CI: %v", got)
	}
}

func TestNeededValuesFromAHostileSpecCannotAddLiveLines(t *testing.T) {
	hostile := []scaffold.NeededValue{
		{Name: "a\n  - name: injected\n    value: x"},
		{Name: "b\r- name: injected2"},
		{Name: "c\u2028  - name: injected3"},
		{Name: "d\u0085  - name: injected4"},
		{Name: "ok", UsedBy: []string{"GET /x\n  - name: injected5\n    value: y", "GET /y\u2028  - name: injected6"}},
	}
	o := scaffold.ProjectOptions{Needed: hostile}
	for _, path := range []string{"environments/local.yaml", "environments/ci.yaml"} {
		content := environmentFile(t, o, path)
		for name := range variablesOf(t, content) {
			if strings.Contains(name, "injected") {
				t.Errorf("%s: %q became a variable:\n%s", path, name, content)
			}
		}
	}
	if got := scaffold.Listable(hostile); len(got) != 1 || got[0].Name != "ok" {
		t.Errorf("names that do not fit on a line are left out: %+v", got)
	}
}

func TestNoNeededValuesLeavesTheFilesAsTheyWere(t *testing.T) {
	for _, path := range []string{"environments/local.yaml", "environments/ci.yaml"} {
		if c := environmentFile(t, scaffold.ProjectOptions{}, path); strings.Contains(c, "#- name") {
			t.Errorf("%s:\n%s", path, c)
		}
	}
}
