package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

const postmanFixture = `{
  "info": {"name": "My API", "schema": "https://schema.getpostman.com/json/collection/v2.1.0/collection.json"},
  "item": [
    {"name": "List Users", "request": {"method": "GET", "url": {"raw": "https://api.example.com/users"}}},
    {"name": "Admin", "item": [
      {"name": "Ban user", "request": {"method": "POST", "url": {"raw": "https://api.example.com/admin/ban"}}}
    ]}
  ]
}`

const insomniaFixture = `{
  "_type": "export", "__export_format": 4,
  "resources": [
    {"_id": "wrk_1", "_type": "workspace", "name": "Shop", "parentId": ""},
    {"_id": "req_1", "_type": "request", "parentId": "wrk_1", "name": "get items", "method": "GET", "url": "https://shop.example.com/items"}
  ]
}`

func execImport(t *testing.T, args ...string) (string, error) {
	t.Helper()
	root := &cobra.Command{Use: "clrsig", SilenceErrors: true, SilenceUsage: true}
	root.AddCommand(importCmd())
	buf := &bytes.Buffer{}
	root.SetOut(buf)
	root.SetErr(buf)
	root.SetArgs(append([]string{"import"}, args...))
	err := root.Execute()
	return buf.String(), err
}

func TestImport_PostmanWritesRequestFiles(t *testing.T) {
	src := writeTempFile(t, "collection.json", postmanFixture)
	out := filepath.Join(t.TempDir(), "requests")

	text, err := execImport(t, "postman", src, "--output", out)
	if err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	if !strings.Contains(text, `Wrote 2 request files from "My API"`) {
		t.Errorf("summary:\n%s", text)
	}
	for _, want := range []string{"my-api/List-Users.yaml", "my-api/admin/Ban-user.yaml"} {
		data, err := os.ReadFile(filepath.Join(out, filepath.FromSlash(want)))
		if err != nil || !strings.Contains(string(data), "schemaVersion") {
			t.Errorf("%s was not written as a request file: %v", want, err)
		}
	}
}

func TestImport_DryRunWritesNothing(t *testing.T) {
	src := writeTempFile(t, "collection.json", postmanFixture)
	out := filepath.Join(t.TempDir(), "requests")

	text, err := execImport(t, "postman", src, "--output", out, "--dry-run")
	if err != nil || !strings.Contains(text, "Would write 2 request files") {
		t.Fatalf("%v\n%s", err, text)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("a dry run must not create the output folder")
	}
}

func TestImport_NeverOverwritesAndNumbersFiles(t *testing.T) {
	src := writeTempFile(t, "collection.json", postmanFixture)
	out := filepath.Join(t.TempDir(), "requests")
	if _, err := execImport(t, "postman", src, "-o", out); err != nil {
		t.Fatal(err)
	}
	text, err := execImport(t, "postman", src, "-o", out)
	if err != nil || !strings.Contains(text, "List-Users-2.yaml") {
		t.Errorf("a second import keeps the first files and numbers the new ones: %v\n%s", err, text)
	}
}

func TestImport_InsomniaAndJSON(t *testing.T) {
	src := writeTempFile(t, "insomnia.json", insomniaFixture)
	out := filepath.Join(t.TempDir(), "r")
	text, err := execImport(t, "insomnia", src, "-o", out, "--format", "json")
	if err != nil {
		t.Fatalf("%v\n%s", err, text)
	}
	var doc struct {
		Collection string `json:"collection"`
		DryRun     bool   `json:"dryRun"`
		Files      []struct {
			Path, Method, URL string
		} `json:"files"`
	}
	if err := json.Unmarshal([]byte(text), &doc); err != nil {
		t.Fatalf("not JSON: %v\n%s", err, text)
	}
	if doc.Collection != "Shop" || len(doc.Files) != 1 || doc.Files[0].Method != "GET" || doc.Files[0].URL != "https://shop.example.com/items" {
		t.Errorf("doc = %+v", doc)
	}
	if _, err := os.Stat(doc.Files[0].Path); err != nil {
		t.Errorf("the reported file should exist: %v", err)
	}
}

func TestImport_Errors(t *testing.T) {
	good := writeTempFile(t, "c.json", postmanFixture)
	bad := writeTempFile(t, "bad.json", `{"not": "a collection"}`)
	empty := writeTempFile(t, "empty.json", `{"info":{"name":"x"},"item":[]}`)

	cases := []struct {
		name string
		args []string
		want string
	}{
		{"unknown source", []string{"curl", good}, "unknown source"},
		{"missing file", []string{"postman", filepath.Join(t.TempDir(), "nope.json")}, "reading"},
		{"wrong content", []string{"postman", bad}, ""},
		{"no requests", []string{"postman", empty}, "no requests found"},
		{"bad format", []string{"postman", good, "--format", "yaml"}, "unknown --format"},
	}
	for _, c := range cases {
		_, err := execImport(t, c.args...)
		if err == nil || (c.want != "" && !strings.Contains(err.Error(), c.want)) {
			t.Errorf("%s: want error containing %q, got %v", c.name, c.want, err)
		}
	}
}
