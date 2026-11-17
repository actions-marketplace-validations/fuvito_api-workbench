package main

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/importer"
)

const taggedSpec = `openapi: "3.0.3"
info: {title: Pet Store, version: "9"}
paths:
  /pets:
    get:
      operationId: listPets
      tags: [Pets Admin]
      responses: {"200": {description: ok}}
    post:
      operationId: createPet
      tags: [Pets Admin]
      requestBody:
        content:
          application/json:
            schema: {type: object, properties: {name: {type: string}}}
      responses: {"201": {description: ok}}
  /pets/{petId}:
    get:
      operationId: getPet
      parameters:
        - {name: petId, in: path, required: true, schema: {type: integer}}
      responses: {"200": {description: ok}}
  /:
    get:
      operationId: root
      responses: {"200": {description: ok}}
`

const swagger2Spec = `{"swagger":"2.0","info":{"title":"Legacy","version":"1"},"host":"legacy.example.com","basePath":"/v1","schemes":["https"],
"paths":{"/things":{"get":{"operationId":"listThings","responses":{"200":{"description":"ok"}}}}}}`

func TestOpenAPIImport_FromPath(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	svc := NewOpenAPIService(ws)
	spec := writeFile(t, filepath.Join(t.TempDir(), "spec.yaml"), taggedSpec)

	res := svc.ImportOpenAPISpecFromPath(spec, "http://localhost:8080")
	if res.Error != "" {
		t.Fatal(res.Error)
	}
	if res.SpecTitle != "Pet Store" || res.SpecVersion != "9" || len(res.Imported) != 4 {
		t.Fatalf("result: %+v", res)
	}

	byOp := map[string]ImportedRequestIO{}
	for _, ir := range res.Imported {
		byOp[ir.OperationID] = ir
		if _, err := os.Stat(ir.SavedPath); err != nil {
			t.Errorf("%s not saved: %v", ir.OperationID, err)
		}
		if !strings.HasPrefix(ir.URL, "http://localhost:8080") {
			t.Errorf("base URL not applied: %q", ir.URL)
		}
	}
	base := filepath.Join(ws.workspacePath, "requests", "pet-store")
	if got := byOp["listPets"].SavedPath; got != filepath.Join(base, "pets-admin", "get-pets.yaml") {
		t.Errorf("tagged request path: %q", got)
	}
	if got := byOp["getPet"].SavedPath; got != filepath.Join(base, "get-petId.yaml") {
		t.Errorf("untagged request path: %q", got)
	}
	if got := filepath.Base(byOp["root"].SavedPath); got != "root.yaml" {
		t.Errorf("a path with no segment falls back to the operationId: %q", got)
	}

	// Files are valid requests that the workspace can load back.
	loaded, err := ws.LoadRequest(byOp["createPet"].SavedPath)
	if err != nil || loaded.Method != "POST" || loaded.Body == nil {
		t.Errorf("saved request: %+v %v", loaded, err)
	}

	again := svc.ImportOpenAPISpecFromPath(spec, "http://localhost:8080")
	if len(again.Imported) != 4 || again.Imported[0].SavedPath == res.Imported[0].SavedPath {
		t.Errorf("re-import must not overwrite existing files: %+v", again.Imported)
	}
}

func TestOpenAPIImport_Errors(t *testing.T) {
	isolateHome(t)
	svc := NewOpenAPIService(newWS(t, t.TempDir()))

	if r := svc.ImportOpenAPISpecFromPath(filepath.Join(t.TempDir(), "none.yaml"), ""); !strings.Contains(r.Error, "reading file") {
		t.Errorf("missing file: %+v", r)
	}
	bad := writeFile(t, filepath.Join(t.TempDir(), "bad.yaml"), "openapi: [")
	if r := svc.ImportOpenAPISpecFromPath(bad, ""); !strings.Contains(r.Error, "parsing spec") {
		t.Errorf("bad spec: %+v", r)
	}
	badSwagger := writeFile(t, filepath.Join(t.TempDir(), "bad.json"), `{"swagger":"2.0","paths":`)
	if r := svc.ImportOpenAPISpecFromPath(badSwagger, ""); r.Error == "" {
		t.Errorf("bad swagger: %+v", r)
	}
	spec := writeFile(t, filepath.Join(t.TempDir(), "ok.yaml"), taggedSpec)
	noWS := NewOpenAPIService(newWS(t, ""))
	if r := noWS.ImportOpenAPISpecFromPath(spec, ""); !strings.Contains(r.Error, "no workspace") {
		t.Errorf("no workspace: %+v", r)
	}
}

func TestOpenAPIImport_Swagger2(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	spec := writeFile(t, filepath.Join(t.TempDir(), "legacy.json"), swagger2Spec)
	res := NewOpenAPIService(ws).ImportOpenAPISpecFromPath(spec, "")
	if res.Error != "" || res.SpecTitle != "Legacy" || len(res.Imported) != 1 {
		t.Fatalf("swagger 2: %+v", res)
	}
	if !strings.Contains(res.Imported[0].URL, "legacy.example.com") {
		t.Errorf("swagger 2 base URL: %q", res.Imported[0].URL)
	}
}

func TestOpenAPIImport_FromURL(t *testing.T) {
	isolateHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/spec.yaml":
			fmt.Fprint(w, taggedSpec)
		case "/spec.json":
			fmt.Fprint(w, swagger2Spec)
		case "/typed":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, swagger2Spec)
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	svc := NewOpenAPIService(newWS(t, t.TempDir()))
	if r := svc.ImportOpenAPISpecFromURL(srv.URL+"/spec.yaml", ""); r.Error != "" || len(r.Imported) != 4 {
		t.Errorf("yaml url: %+v", r)
	}
	if r := svc.ImportOpenAPISpecFromURL(srv.URL+"/spec.json", ""); r.Error != "" || len(r.Imported) != 1 {
		t.Errorf("json url: %+v", r)
	}
	if r := svc.ImportOpenAPISpecFromURL(srv.URL+"/typed", ""); r.Error != "" || len(r.Imported) != 1 {
		t.Errorf("json content type: %+v", r)
	}
	if r := svc.ImportOpenAPISpecFromURL(srv.URL+"/missing", ""); !strings.Contains(r.Error, "404") {
		t.Errorf("404: %+v", r)
	}
	if r := svc.ImportOpenAPISpecFromURL("", ""); r.Error == "" {
		t.Error("empty url")
	}
	if r := svc.ImportOpenAPISpecFromURL("http://127.0.0.1:1/x", ""); !strings.Contains(r.Error, "fetching URL") {
		t.Errorf("unreachable: %+v", r)
	}
}

func TestPathBasedFilenameAndSanitizers(t *testing.T) {
	for _, c := range []struct{ method, path, want string }{
		{"GET", "/users", "get-users"},
		{"POST", "/user/{username}", "post-username"},
		{"DELETE", "/a/b/", "delete-b"},
		{"GET", "/", ""},
		{"GET", "", ""},
	} {
		if got := pathBasedFilename(c.method, c.path); got != c.want {
			t.Errorf("pathBasedFilename(%s,%q) = %q, want %q", c.method, c.path, got, c.want)
		}
	}
	if got := sanitizeDirName("My API_v2!"); got != "my-api-v2" {
		t.Errorf("sanitizeDirName: %q", got)
	}
	if got := sanitizeFilename("Get User/by id?"); got != "Get-Userby-id" {
		t.Errorf("sanitizeFilename: %q", got)
	}
}

func TestFindNonConflictingPath(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "a.yaml")
	writeFile(t, p, "x")
	first := findNonConflictingPath(p)
	writeFile(t, first, "x")
	second := findNonConflictingPath(p)
	if first == p || second == p || first == second {
		t.Errorf("paths must be distinct and free: %q %q", first, second)
	}
	for _, q := range []string{first, second} {
		if filepath.Dir(q) != dir || filepath.Ext(q) != ".yaml" {
			t.Errorf("unexpected path %q", q)
		}
	}
}

func TestSaveImportedRequests(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	svc := NewImportService(ws)
	mk := func(title, folder string) importer.ImportedRequest {
		return importer.ImportedRequest{
			FolderPath: folder,
			Request:    &domain.Request{Title: title, Method: domain.MethodGET, URL: "http://x/" + title},
		}
	}

	res := svc.saveImportedRequests("My Collection!", []importer.ImportedRequest{
		mk("Get One", ""),
		mk("Get One", ""),
		mk("Nested", "Auth/OAuth Flows"),
		mk("???", ""),
	})
	if res.Error != "" || res.CollectionName != "My Collection!" || len(res.Imported) != 4 || res.Skipped != 0 {
		t.Fatalf("result: %+v", res)
	}
	base := filepath.Join(ws.workspacePath, "requests", "my-collection")
	want := []string{
		filepath.Join(base, "Get-One.yaml"),
		filepath.Join(base, "auth", "oauth-flows", "Nested.yaml"),
		filepath.Join(base, "request.yaml"),
	}
	for _, w := range want {
		if _, err := os.Stat(w); err != nil {
			t.Errorf("expected %s: %v", w, err)
		}
	}
	if res.Imported[0].SavedPath == res.Imported[1].SavedPath {
		t.Error("duplicate titles must not overwrite each other")
	}

	if r := svc.saveImportedRequests("!!!", []importer.ImportedRequest{mk("a", "")}); r.Error != "" ||
		!strings.Contains(r.Imported[0].SavedPath, filepath.Join("requests", "imported")) {
		t.Errorf("unsluggable collection name: %+v", r)
	}
	if r := NewImportService(newWS(t, "")).saveImportedRequests("x", nil); !strings.Contains(r.Error, "no workspace") {
		t.Errorf("no workspace: %+v", r)
	}
}

func TestImportedPostmanEndToEnd(t *testing.T) {
	isolateHome(t)
	ws := newWS(t, t.TempDir())
	name, reqs, err := importer.ParsePostmanCollection([]byte(
		`{"info":{"name":"Shop"},"item":[{"name":"Items","item":[{"name":"List","request":{"method":"GET","url":{"raw":"https://x.com/items"}}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	res := NewImportService(ws).saveImportedRequests(name, reqs)
	if len(res.Imported) != 1 || !strings.HasSuffix(res.Imported[0].SavedPath, filepath.Join("shop", "items", "List.yaml")) {
		t.Errorf("postman import: %+v", res)
	}
	if _, err := ws.LoadRequest(res.Imported[0].SavedPath); err != nil {
		t.Errorf("imported file should load: %v", err)
	}
}

func TestHelpService(t *testing.T) {
	h := NewHelpService()
	guide, spec, prompt := h.GetUserGuide(), h.GetFormatSpec(), h.GetAIPrompt()
	if len(guide) < 100 || len(spec) < 100 {
		t.Fatalf("embedded docs look empty: guide=%d spec=%d", len(guide), len(spec))
	}
	if !strings.HasSuffix(prompt, spec) || len(prompt) <= len(spec) {
		t.Error("the AI prompt must end with the full format spec")
	}
	for _, want := range []string{"Step 4", "{{"} {
		if !strings.Contains(prompt, want) {
			t.Errorf("prompt missing %q", want)
		}
	}
}
