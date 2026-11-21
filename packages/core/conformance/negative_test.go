package conformance_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/fuvito/api-workbench/packages/core/conformance"
	"github.com/fuvito/api-workbench/packages/core/executor"
	"github.com/fuvito/api-workbench/packages/core/openapi"
)

// negativeSpec declares three operations that have distinct negative-probe targets:
//   - POST /users       — required fields (username string, age integer) → generates
//     drop-required-field and wrong-type probes
//   - DELETE /users/{id} — bearer security → generates missing-auth probe
//   - GET /items        — required query param "q" → generates missing-param probe
const negativeSpec = `openapi: "3.0.3"
info:
  title: "neg-api"
  version: "1.0.0"
components:
  securitySchemes:
    BearerAuth:
      type: http
      scheme: bearer
paths:
  /users:
    post:
      operationId: "createUser"
      requestBody:
        required: true
        content:
          application/json:
            schema:
              type: object
              required: [username, age]
              properties:
                username:
                  type: string
                age:
                  type: integer
      responses:
        "201":
          description: "Created"
        "400":
          description: "Bad Request"
        "422":
          description: "Unprocessable Entity"
  /users/{id}:
    delete:
      operationId: "deleteUser"
      security:
        - BearerAuth: []
      parameters:
        - name: id
          in: path
          required: true
          schema:
            type: string
      responses:
        "204":
          description: "No Content"
        "401":
          description: "Unauthorized"
  /items:
    get:
      operationId: "listItems"
      parameters:
        - name: q
          in: query
          required: true
          schema:
            type: string
      responses:
        "200":
          description: "OK"
        "400":
          description: "Bad Request"
`

func parseNegativeSpec(t *testing.T) *openapi.Spec {
	t.Helper()
	spec, err := openapi.ParseBytes([]byte(negativeSpec), "yaml")
	if err != nil {
		t.Fatalf("ParseBytes: %v", err)
	}
	return spec
}

// strictServer correctly rejects all contract violations.
func strictServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/users":
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, "bad json", http.StatusBadRequest)
				return
			}
			username, hasUsername := body["username"]
			age, hasAge := body["age"]
			if !hasUsername || !hasAge {
				http.Error(w, "missing required field", http.StatusUnprocessableEntity)
				return
			}
			// Type check: username must be string, age must be number
			if _, ok := username.(string); !ok {
				http.Error(w, "username must be string", http.StatusUnprocessableEntity)
				return
			}
			if _, ok := age.(float64); !ok { // JSON numbers are float64
				http.Error(w, "age must be integer", http.StatusUnprocessableEntity)
				return
			}
			w.WriteHeader(http.StatusCreated)

		case r.Method == http.MethodDelete && r.URL.Path != "" && len(r.URL.Path) > len("/users/"):
			if r.Header.Get("Authorization") == "" {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusNoContent)

		case r.Method == http.MethodGet && r.URL.Path == "/items":
			if r.URL.Query().Get("q") == "" {
				http.Error(w, "missing q", http.StatusBadRequest)
				return
			}
			w.WriteHeader(http.StatusOK)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// lenientServer accepts everything with 2xx — a broken server.
func lenientServer(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		if r.Method == http.MethodPost {
			w.WriteHeader(http.StatusCreated)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv
}

// ---- NegativeResult.Counts ----

func TestNegativeResult_Counts(t *testing.T) {
	r := &conformance.NegativeResult{
		Probes: []conformance.NegativeProbeResult{
			{StatusCode: 422},             // passed
			{StatusCode: 401},             // passed
			{StatusCode: 200},             // failed
			{Error: "connection refused"}, // errored
			{Skipped: true},               // skipped
		},
	}
	passed, failed, skipped, errored := r.Counts()
	if passed != 2 {
		t.Errorf("passed: got %d, want 2", passed)
	}
	if failed != 1 {
		t.Errorf("failed: got %d, want 1", failed)
	}
	if skipped != 1 {
		t.Errorf("skipped: got %d, want 1", skipped)
	}
	if errored != 1 {
		t.Errorf("errored: got %d, want 1", errored)
	}
}

// ---- NegativeRunner.Run against strict server ----

func TestNegativeRunner_StrictServer_AllPass(t *testing.T) {
	srv := strictServer(t)
	spec := parseNegativeSpec(t)
	runner := conformance.NewNegative(executor.New(executor.Options{}))

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{})

	passed, failed, skipped, errored := result.Counts()
	t.Logf("passed=%d failed=%d skipped=%d errored=%d", passed, failed, skipped, errored)

	// DELETE /users/{id} has an unresolved {id} placeholder, so those probes are skipped.
	// All other probes should pass (strict server returns 4xx for bad requests).
	if failed > 0 {
		for _, p := range result.Probes {
			if !p.Passed() && !p.Skipped && p.Error == "" {
				t.Errorf("probe FAILED: %s  status=%d", p.Description, p.StatusCode)
			}
		}
		t.Errorf("want 0 failed probes, got %d", failed)
	}
	if errored > 0 {
		t.Errorf("want 0 errored probes, got %d", errored)
	}
	if passed == 0 {
		t.Error("want at least one passed probe from strict server")
	}
}

// ---- NegativeRunner.Run against lenient server ----

func TestNegativeRunner_LenientServer_ProbesFail(t *testing.T) {
	srv := lenientServer(t)
	spec := parseNegativeSpec(t)
	runner := conformance.NewNegative(executor.New(executor.Options{}))

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{})

	_, failed, _, _ := result.Counts()
	if failed == 0 {
		t.Error("want at least one failed probe against lenient server (server accepts invalid requests)")
	}
}

// ---- Probe generation sanity ----

func TestGenerateNegativeProbes_ExpectedKinds(t *testing.T) {
	spec := parseNegativeSpec(t)
	probes, err := openapi.GenerateNegativeProbes(spec, "http://example.com")
	if err != nil {
		t.Fatalf("GenerateNegativeProbes: %v", err)
	}

	kinds := make(map[openapi.ProbeKind]int)
	for _, p := range probes {
		kinds[p.Kind]++
	}

	// POST /users has 2 required fields → 2 drop-field + 2 wrong-type probes
	if kinds[openapi.ProbeDropRequiredField] < 2 {
		t.Errorf("drop_required_field: got %d, want >=2", kinds[openapi.ProbeDropRequiredField])
	}
	if kinds[openapi.ProbeWrongFieldType] < 2 {
		t.Errorf("wrong_field_type: got %d, want >=2", kinds[openapi.ProbeWrongFieldType])
	}
	// DELETE /users/{id} has bearer security → 1 missing-auth probe
	if kinds[openapi.ProbeMissingAuth] < 1 {
		t.Errorf("missing_auth: got %d, want >=1", kinds[openapi.ProbeMissingAuth])
	}
	// GET /items has required query param "q" → 1 missing-param probe
	if kinds[openapi.ProbeMissingRequiredParam] < 1 {
		t.Errorf("missing_required_param: got %d, want >=1", kinds[openapi.ProbeMissingRequiredParam])
	}
}

func TestNegativeRunner_FailFast(t *testing.T) {
	srv := lenientServer(t)
	spec := parseNegativeSpec(t)
	runner := conformance.NewNegative(executor.New(executor.Options{}))

	result := runner.Run(context.Background(), spec, srv.URL, nil, conformance.Options{FailFast: true})

	executed := 0
	for _, p := range result.Probes {
		if !p.Skipped && p.Error == "" {
			executed++
		}
	}
	// FailFast should stop after the first non-passing, non-skipped probe.
	if executed > 1 {
		t.Errorf("FailFast: expected <=1 executed probes, got %d", executed)
	}
}
