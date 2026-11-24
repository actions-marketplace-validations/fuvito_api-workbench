package openapi_test

import (
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const changesBase = `openapi: "3.0.3"
info: {title: t, version: "1"}
components:
  schemas:
    User: {type: object, required: [id], properties: {id: {type: integer}, name: {type: string}}}
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200": {description: ok, content: {application/json: {schema: {type: array, items: {$ref: "#/components/schemas/User"}}}}}
  /users/{id}:
    get:
      operationId: getUser
      parameters: [{name: id, in: path, required: true, schema: {type: integer}}]
      responses:
        "200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/User"}}}}
  /health:
    get:
      operationId: health
      responses: {"200": {description: ok}}
  /untouched:
    get:
      operationId: untouched
      responses: {"200": {description: ok}}
  /old:
    get:
      operationId: old
      responses: {"200": {description: ok}}
`

const changesRevision = `openapi: "3.0.3"
info: {title: t, version: "2"}
servers: [{url: "http://other"}]
components:
  schemas:
    User: {type: object, required: [id, email], properties: {id: {type: integer}, name: {type: string}, email: {type: string}}}
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200": {description: ok, content: {application/json: {schema: {type: array, items: {$ref: "#/components/schemas/User"}}}}}
  /users/{id}:
    get:
      operationId: getUser
      parameters: [{name: id, in: path, required: true, schema: {type: integer}}]
      responses:
        "200": {description: ok, content: {application/json: {schema: {$ref: "#/components/schemas/User"}}}}
  /health:
    get:
      operationId: health
      responses: {"200": {description: ok}, "503": {description: down}}
  /untouched:
    get:
      operationId: untouched
      responses: {"200": {description: ok}}
  /new:
    post:
      operationId: create
      responses: {"201": {description: ok}}
`

func parseSpec(t *testing.T, text string) *openapi.Spec {
	t.Helper()
	s, err := openapi.ParseBytes([]byte(text), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	return s
}

func TestChangedOperations(t *testing.T) {
	cs, err := openapi.ChangedOperations(parseSpec(t, changesBase), parseSpec(t, changesRevision))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, op := range cs.Changed {
		got = append(got, op.String())
	}
	want := "GET /health,POST /new,GET /users,GET /users/{id}"
	if strings.Join(got, ",") != want {
		t.Errorf("changed = %v, want %s", got, want)
	}
	if cs.Added != 1 || cs.Modified != 3 || cs.Unchanged != 1 {
		t.Errorf("added=%d modified=%d unchanged=%d", cs.Added, cs.Modified, cs.Unchanged)
	}
	if len(cs.Removed) != 1 || cs.Removed[0].String() != "GET /old" {
		t.Errorf("a removed operation is listed, not checked: %v", cs.Removed)
	}
	keys := cs.Keys()
	if !keys["POST /new"] || keys["GET /untouched"] || keys["GET /old"] {
		t.Errorf("keys: %v", keys)
	}
}

// A schema shared by two operations is a change to both; the version and a servers list are changes
// no operation owns.
func TestChangedOperations_SharedSchemaAndSpecLevelChanges(t *testing.T) {
	cs, err := openapi.ChangedOperations(parseSpec(t, changesBase), parseSpec(t, changesRevision))
	if err != nil {
		t.Fatal(err)
	}
	keys := cs.Keys()
	if !keys["GET /users"] || !keys["GET /users/{id}"] {
		t.Errorf("both operations that use the changed schema are changed: %v", keys)
	}
	if cs.Unattributed < 1 {
		t.Errorf("the version and servers changes belong to no operation: %d", cs.Unattributed)
	}
}

func TestChangedOperations_NothingChanged(t *testing.T) {
	cs, err := openapi.ChangedOperations(parseSpec(t, changesBase), parseSpec(t, changesBase))
	if err != nil {
		t.Fatal(err)
	}
	if len(cs.Changed) != 0 || cs.Unchanged != 5 || len(cs.Removed) != 0 {
		t.Errorf("%+v", cs)
	}
}
