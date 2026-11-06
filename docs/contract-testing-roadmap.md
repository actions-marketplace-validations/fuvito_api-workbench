# ApiDesk — Contract Testing: Gaps & Roadmap

_2026-11-06_

---

## What we have today

ApiDesk has the building blocks for contract validation but they are not yet assembled into a first-class conformance workflow. Four capabilities exist:

| Capability | Location | What it does |
| --- | --- | --- |
| OpenAPI spec parsing | `packages/core/openapi/parser.go` | Loads 3.0/3.1 YAML or JSON; validates structure via kin-openapi |
| Request generation | `packages/core/openapi/parser.go` | One `domain.Request` per operation with `{{param}}` placeholders, body stubs, Bearer auth hints |
| Response schema validation | `packages/core/openapi/validator.go` | `ValidateResponse` (by operationID) and `ValidateResponseByPath` (by method + path) check a response body against the spec's declared schema |
| Schema assertions | `packages/core/assertions/evaluator.go` | `AssertionSchema` wraps an inline YAML/JSON schema in a minimal spec and runs validation as a pass/fail assertion step |

The desktop UI imports a spec and writes one YAML request file per operation. `apidesk generate` does the same headlessly from a file or URL. Neither automatically validates responses — that must be wired manually via `schema` assertions in scenario YAML.

---

## Gap 1 — Automatic Spec Conformance Mode

**Problem:** There is no way to say "run all generated requests and validate every response against the spec" without manually authoring a scenario file with schema assertions for each step.

**Proposed solution:** A new `apidesk conform` CLI command.

```
apidesk conform <spec-file|url> \
  --base-url https://api.example.com \
  [--env env.yaml] \
  [--tags users,auth]      # filter by OpenAPI tag
  [--paths /users,/health] # filter by path prefix
  [--format text|json|junit] \
  [--output report.xml]
```

Execution model:
1. Parse the spec.
2. For each matching operation, generate a request (with `{{param}}` values left empty or supplied via `--params`).
3. Execute the request against `--base-url`.
4. Call `openapi.ValidateResponseByPath` on the response.
5. Report pass/fail per operation with the specific schema violations.

Exit codes: `0` = all conform, `1` = violations found, `2` = execution/parse error.

**Go implementation sketch:**
- New `conformance` package under `packages/core/` with a `Runner` that mirrors `scenario.Runner` but drives from a spec rather than a scenario file.
- Reuses the existing `executor.Executor`, `openapi.GenerateRequests`, and `openapi.ValidateResponseByPath`.
- New `reporting.Conformance(result)` function alongside the existing `Text` and `JSON` reporters.

**Desktop UI:** A "Validate conformance" button in the workspace panel (shown when a spec has been imported) that runs the equivalent flow and displays a per-endpoint pass/fail table.

---

## Gap 2 — Breaking Change Detection

**Problem:** No way to compare two versions of a spec to detect breaking changes (removed endpoints, added required fields, changed response schemas, narrowed enums, changed auth requirements).

**Proposed solution:** A new `apidesk diff` CLI command.

```
apidesk diff <old-spec> <new-spec> \
  [--format text|json] \
  [--breaking-only]
```

**Classification of changes:**

| Type | Example | Breaking? |
| --- | --- | --- |
| Endpoint removed | `DELETE /users/{id}` gone | Yes |
| Required request param added | new required `X-Tenant-ID` header | Yes |
| Response field removed | `user.email` no longer in `GET /users/{id}` | Yes |
| Response field type changed | `id: integer` → `id: string` | Yes |
| Enum value removed | status no longer allows `"pending"` | Yes |
| Auth requirement added | operation now requires `bearerAuth` | Yes |
| Endpoint added | new `POST /exports` | No |
| Optional field added to response | new `user.avatar` | No |
| Description / example changed | | No |

**Implementation options:**
- **oasdiff** (`github.com/oasdiff/oasdiff`) — mature Go library with breaking-change detection; MIT licensed. This is the recommended path: wraps it in a new `packages/core/openapi/diff.go` file and exposes a `Diff(oldSpec, newSpec *Spec) DiffResult` function.
- **Roll our own** — feasible but would take weeks to reach parity with oasdiff's coverage.

**CI use case:** Add a `diff` step in GitHub Actions that runs `apidesk diff main-spec.yaml pr-spec.yaml --breaking-only` and fails the build if breaking changes are found.

---

## Gap 3 — Request Validation

**Problem:** Only responses are validated against the spec. Outgoing requests in scenarios are never checked — a scenario could send a malformed request (missing required fields, wrong Content-Type, bad path param type) and still receive a response that happens to pass schema assertions.

**Proposed solution:** Add `ValidateRequest` to `packages/core/openapi/validator.go`.

```go
func ValidateRequest(s *Spec, req *domain.Request) []ValidationError
```

Uses kin-openapi's `openapi3filter.ValidateRequest` pipeline (the same library already used for response validation).

**Wiring options:**
1. **Scenario step flag** — add `validateRequest: true` to `ScenarioStep`; the runner calls `ValidateRequest` before executing the HTTP call and records violations as assertion failures.
2. **Scenario-level spec binding** (see Gap 5) — when a scenario is bound to a spec, all steps get request validation automatically.
3. **CLI `conform` command** — `apidesk conform` validates both the generated request and the response.

Option 2 is the most useful; option 1 is the cheapest to implement as a starting point.

---

## Gap 4 — CLI Conformance Report (`apidesk conform`)

This is the CLI surface for Gap 1. Full specification:

```
apidesk conform <spec> [flags]

Flags:
  -b, --base-url string        Base URL to send requests to (overrides spec servers[0])
  -e, --env string             Environment file for variable resolution
      --tags string            Comma-separated list of OpenAPI tags to include
      --paths string           Comma-separated path prefixes to include
  -f, --format string          Output format: text (default), json, junit
  -o, --output string          Write report to file (stdout if omitted)
      --request-timeout int    Per-request timeout in ms (default 10000)
      --fail-fast              Stop after first violation
```

**Text output example:**

```
ApiDesk Example API  1.0.0
Base URL: http://localhost:8080

  PASS  POST   /auth/login        200  (application/json)
  PASS  GET    /health            200  (application/json)
  FAIL  GET    /users/{id}        200  missing required field "email"
  PASS  POST   /users             201  (application/json)
  SKIP  DELETE /users/{id}        —    no 2xx response declared

4 operations  ·  3 passed  ·  1 failed  ·  1 skipped
Exit code 1
```

**JUnit XML output** reuses the existing `reporting.JUnit()` infrastructure already built for scenario runs.

---

## Gap 5 — Scenario-Level Spec Binding

**Problem:** `AssertionSchema` requires embedding a full inline schema in every step. There is no way to bind a scenario to a spec file and have all steps automatically validated against the declared response schemas.

**Proposed solution:** Add an optional `specFile` field to the scenario YAML.

```yaml
schemaVersion: "1"
name: User lifecycle
specFile: ../../openapi.yaml     # relative to scenario file

steps:
  - name: Create user
    request:
      method: POST
      url: "{{baseUrl}}/users"
      body:
        type: json
        content: '{"name":"Alice"}'
    assertions:
      - type: status
        operator: eq
        expected: 201
    # No "schema" assertion needed — specFile binding validates automatically
```

**Runner behaviour when `specFile` is set:**
1. Load the spec once before the run.
2. After each step, call `openapi.ValidateResponseByPath(spec, method, path, statusCode, resp)`.
3. Any schema violations are appended to `sr.Assertions` as failed `AssertionSchema` results.
4. Steps can opt out per-step with `validateSpec: false`.

**Domain change:** Add `SpecFile string` and `ValidateSpec *bool` to `domain.Scenario` and `domain.ScenarioStep` respectively, plus the corresponding YAML tags in `serialization/scenario.go`.

---

## Gap 6 — Conformance Coverage Tracking

**Problem:** There is no visibility into which operations in a spec have been exercised by scenarios. A team can have a full scenario suite that misses entire endpoint groups.

**Proposed solution:** A coverage report emitted alongside the conformance or scenario run report.

**How it works:**
- Each `StepResult` already records the method and URL of the executed request.
- After a run (or a set of runs), compare the executed `(method, path)` pairs against the full set of operations in a bound spec.
- Report coverage as a percentage and list uncovered operations.

**CLI surface:**

```
apidesk conform openapi.yaml --base-url http://localhost:8080 --format json | \
  apidesk coverage openapi.yaml
```

Or as a flag on `conform`:

```
apidesk conform openapi.yaml --base-url http://localhost:8080 --coverage
```

**Coverage output:**

```
Coverage: 6 / 8 operations (75%)

Covered:
  POST   /auth/login
  GET    /health
  GET    /users
  POST   /users
  GET    /users/{id}
  PUT    /users/{id}

Not covered:
  DELETE /users/{id}
  GET    /protected
```

**Longer term:** Aggregate coverage across multiple scenario runs (store covered operation IDs in `history.db`) so a single run does not need to cover everything.

---

## Gap 7 — Additional Requirements

### Webhook / event-driven spec validation
When an API emits callbacks or webhook payloads, there is currently no way to record and validate an inbound payload against an OpenAPI `callbacks` or `webhooks` block. This is out of scope for the current CLI but worth noting for a future `apidesk listen` command.

### Partial / paginated response validation
`ValidateResponse` currently fails fast on the first schema error. For large responses, it may be more useful to report all violations at once. kin-openapi's `VisitJSON` already collects all errors — `parseSchemaErrors` just needs to unwrap them all, which is already implemented.

### Authentication for `apidesk conform`
`apidesk conform` sends requests against a live server. It needs to inject auth credentials (Bearer token, API key) sourced from the environment file, the same way scenarios resolve `{{token}}`. The `--env` flag + `variables.Resolver` already supports this; it just needs to be wired into the conform runner.

---

## Implementation priorities

Suggested order based on value-to-effort ratio:

| Priority | Item | Effort | Value |
| --- | --- | --- | --- |
| 1 | **Request validation** (`ValidateRequest`) | Low — kin-openapi already does it | Catches malformed test scenarios early |
| 2 | **Scenario spec binding** (`specFile`) | Medium — domain + serialization + runner changes | Removes manual `schema` assertion boilerplate |
| 3 | **`apidesk conform` command** | Medium — new conformance runner + reporter | First-class CI conformance gate |
| 4 | **Breaking change detection** (`apidesk diff`) | Low with oasdiff, high without | Prevents spec regressions in CI |
| 5 | **Coverage tracking** | Medium — history store + aggregation | Visibility into test completeness |

Items 1 and 2 can be developed in parallel since they touch different layers. Items 3 and 4 are independent. Item 5 depends on item 3 being in place.
