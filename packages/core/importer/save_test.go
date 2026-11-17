package importer_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/importer"
)

func req(title, folder string) importer.ImportedRequest {
	return importer.ImportedRequest{
		FolderPath: folder,
		Request:    &domain.Request{SchemaVersion: "1", Title: title, Method: domain.MethodGET, URL: "https://api.example.com/" + title},
	}
}

func rel(root string, saved []importer.Saved) string {
	var out []string
	for _, s := range saved {
		r, _ := filepath.Rel(root, s.Path)
		out = append(out, filepath.ToSlash(r))
	}
	return strings.Join(out, ",")
}

func TestSaveWritesFilesIntoCollectionAndFolders(t *testing.T) {
	root := t.TempDir()
	res := importer.Save(root, "My API v1", []importer.ImportedRequest{
		req("List Users", ""),
		req("Create user!", "Admin/Users"),
	}, false)

	if got := rel(root, res.Saved); got != "my-api-v1/List-Users.yaml,my-api-v1/admin/users/Create-user.yaml" {
		t.Errorf("paths = %s", got)
	}
	for _, s := range res.Saved {
		data, err := os.ReadFile(s.Path)
		if err != nil || !strings.Contains(string(data), "schemaVersion") {
			t.Errorf("%s was not written as a request file: %v", s.Path, err)
		}
	}
	if res.Skipped != 0 || res.Collection != "My API v1" {
		t.Errorf("result = %+v", res)
	}
}

func TestSaveNeverOverwritesAndNumbersDuplicates(t *testing.T) {
	root := t.TempDir()
	first := importer.Save(root, "api", []importer.ImportedRequest{req("Ping", "")}, false)
	again := importer.Save(root, "api", []importer.ImportedRequest{req("Ping", ""), req("Ping", "")}, false)

	if got := rel(root, first.Saved); got != "api/Ping.yaml" {
		t.Errorf("first = %s", got)
	}
	if got := rel(root, again.Saved); got != "api/Ping-2.yaml,api/Ping-3.yaml" {
		t.Errorf("a second import keeps the first file and numbers the new ones: %s", got)
	}
}

func TestSaveDryRunWritesNothingButPlansTheSamePaths(t *testing.T) {
	root := t.TempDir()
	reqs := []importer.ImportedRequest{req("Ping", ""), req("Ping", ""), req("", "")}
	dry := importer.Save(root, "api", reqs, true)
	if entries, _ := os.ReadDir(root); len(entries) != 0 {
		t.Errorf("a dry run must not write anything, found %d entries", len(entries))
	}
	real := importer.Save(root, "api", reqs, false)
	if rel(root, dry.Saved) != rel(root, real.Saved) {
		t.Errorf("dry run planned %s but the real import wrote %s", rel(root, dry.Saved), rel(root, real.Saved))
	}
	if got := rel(root, dry.Saved); got != "api/Ping.yaml,api/Ping-2.yaml,api/request.yaml" {
		t.Errorf("paths = %s (an empty title becomes request.yaml)", got)
	}
}

func TestSaveEmptyCollectionNameBecomesImported(t *testing.T) {
	root := t.TempDir()
	res := importer.Save(root, "!!!", []importer.ImportedRequest{req("A", "")}, true)
	if got := rel(root, res.Saved); got != "imported/A.yaml" {
		t.Errorf("paths = %s", got)
	}
}

func TestSanitizers(t *testing.T) {
	if got := importer.SanitizeDirName("My Cool_API v2!"); got != "my-cool-api-v2" {
		t.Errorf("SanitizeDirName = %q", got)
	}
	if got := importer.SanitizeFilename("Get /users/{id}"); got != "Get-usersid" {
		t.Errorf("SanitizeFilename = %q", got)
	}
}
