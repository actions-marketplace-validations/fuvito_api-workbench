package conformance_test

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/domain"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const writeOnlySpec = `openapi: "3.0.3"
info: {title: wo, version: "1"}
servers: [{url: "%s"}]
paths:
  /user:
    get:
      operationId: user
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [name]
                properties:
                  name: {type: string}
                  password: {type: string, writeOnly: true}
`

// F-022: a writeOnly property in a response has its own kind and the catalog ID CS-RESP-011.
func TestFindings_WriteOnlyLeakHasItsOwnKindAndCheckID(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"name":"a","password":"hunter2"}`))
	}))
	defer srv.Close()
	spec, err := openapi.ParseBytes([]byte(fmt.Sprintf(writeOnlySpec, srv.URL)), "yaml")
	if err != nil {
		t.Fatal(err)
	}
	res := conformance.New(executor.New(executor.Options{})).Run(context.Background(), spec, "", domain.NewEnvironment("e"), conformance.Options{})
	f := findingOf(t, res, "/user")
	if f.Kind != conformance.FindingWriteOnly || f.CheckID != "CS-RESP-011" || f.Where != "password" {
		t.Fatalf("kind=%s id=%s where=%q\n%+v", f.Kind, f.CheckID, f.Where, f)
	}
	if strings.Contains(f.Why, "does not match what the spec declares") || f.Fix == "" || f.Spec == "" {
		t.Errorf("the finding must explain why and how to fix, and say where in the spec: %+v", f)
	}
}
