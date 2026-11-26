package importer

import (
	"strings"
	"testing"
)

func TestPostmanScriptsThatAreLeftOutAreReported(t *testing.T) {
	const collection = `{
	"info": {"name": "c"},
	"item": [
		{"name": "own test", "request": {"method": "GET", "url": "http://x"},
		 "event": [{"listen": "test", "script": {"exec": ["pm.test('ok', () => {})"]}}]},
		{"name": "one string", "request": {"method": "GET", "url": "http://x"},
		 "event": [{"listen": "prerequest", "script": {"exec": "pm.environment.set('a', 1)"}}]},
		{"name": "folder", "event": [{"listen": "test", "script": {"exec": ["pm.response.to.have.status(200)"]}}],
		 "item": [{"name": "inherits", "request": {"method": "GET", "url": "http://x"}}]},
		{"name": "empty ones", "request": {"method": "GET", "url": "http://x"},
		 "event": [{"listen": "test", "script": {"exec": ["", "  // nothing yet"]}},
		           {"listen": "prerequest", "script": {"exec": ""}},
		           {"listen": "test", "disabled": true, "script": {"exec": ["pm.test('off')"]}}]},
		{"name": "none", "request": {"method": "GET", "url": "http://x"}}
	]}`
	_, requests, err := ParsePostmanCollection([]byte(collection))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]string{
		"own test": "test script", "one string": "pre-request script", "inherits": "test script",
		"empty ones": "", "none": "",
	}
	if len(requests) != len(want) {
		t.Fatalf("%d requests", len(requests))
	}
	for _, r := range requests {
		got := strings.Join(r.Warnings, "\n")
		expected, known := want[r.Request.Title]
		switch {
		case !known:
			t.Errorf("unexpected request %q", r.Request.Title)
		case expected == "" && got != "":
			t.Errorf("%s: warned about nothing: %s", r.Request.Title, got)
		case expected != "" && (len(r.Warnings) != 1 || !strings.Contains(got, expected)):
			t.Errorf("%s: want one warning about its %s, got %q", r.Request.Title, expected, got)
		}
	}
}

func TestPostmanCollectionScriptsApplyToEveryRequest(t *testing.T) {
	const collection = `{"info": {"name": "c"},
	"event": [{"listen": "test", "script": {"exec": ["pm.test('all')"]}}],
	"item": [{"name": "f", "item": [{"name": "deep", "request": {"method": "GET", "url": "http://x"}}]}]}`
	_, requests, err := ParsePostmanCollection([]byte(collection))
	if err != nil || len(requests) != 1 || len(requests[0].Warnings) != 1 {
		t.Fatalf("%v %+v", err, requests)
	}
}
