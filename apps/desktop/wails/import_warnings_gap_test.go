package main

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/importer"
)

// Pins CS-GAP-026: what an import could not convert does not reach the app.
func TestDesktopImport_DropsWarnings_KnownGap(t *testing.T) {
	isolateHome(t)
	name, requests, err := importer.ParsePostmanCollection([]byte(`{"info": {"name": "c"}, "item": [{"name": "a",
		"request": {"method": "GET", "url": "http://x"},
		"event": [{"listen": "test", "script": {"exec": ["pm.test('ok')"]}}]}]}`))
	if err != nil || len(requests) != 1 || len(requests[0].Warnings) != 1 {
		t.Fatalf("the core should warn once: %v %+v", err, requests)
	}
	result := NewImportService(newWS(t, t.TempDir())).saveImportedRequests(name, requests)
	data, err := json.Marshal(result)
	if err != nil || len(result.Imported) != 1 {
		t.Fatalf("%v %s", err, data)
	}
	if strings.Contains(string(data), "test script") {
		t.Error("the app receives the warnings now: update CS-GAP-026 and close B-148")
	}
}
