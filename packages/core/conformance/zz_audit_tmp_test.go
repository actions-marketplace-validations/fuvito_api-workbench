package conformance

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

const e2eSpec = `openapi: 3.0.3
info: {title: t, version: "1"}
SERVERS
paths:
  /users:
    get:
      operationId: listUsers
      responses:
        "200":
          description: ok
          content:
            application/json:
              schema:
                type: object
                required: [total]
                properties: {total: {type: integer}}
`

func TestAuditConformEndToEnd(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprint(w, `{"total":"not-an-integer"}`) // violates the schema
	}))
	defer srv.Close()
	for _, servers := range []string{"", "servers:\n  - url: " + srv.URL + "\n", "servers:\n  - url: http://localhost:8080\n"} {
		s, err := openapi.ParseBytes([]byte(strings.Replace(e2eSpec, "SERVERS\n", servers, 1)), "yaml")
		if err != nil {
			t.Fatal(err)
		}
		res := New(executor.New(executor.Options{})).Run(context.Background(), s, srv.URL, nil, Options{})
		for _, op := range res.Operations {
			t.Logf("AUDIT servers=%q => %s %s status=%d violations=%d passed=%v", strings.TrimSpace(servers), op.Method, op.Path, op.StatusCode, len(op.Violations), op.Passed())
		}
	}
}
