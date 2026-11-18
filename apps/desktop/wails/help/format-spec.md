# Clear Signal — YAML Format Specification

This document describes every file format used by Clear Signal. All files are valid YAML or JSON and share the same schema.

---

## Variable Syntax

Use `{{variableName}}` anywhere in a URL, header value, query param value, or body content. Variable names are case-sensitive.

```yaml
url: "{{baseUrl}}/users/{{userId}}"
headers:
  - name: Authorization
    value: "Bearer {{token}}"
```

Variables are resolved in this order: values extracted by prior scenario steps, then the scenario's own `variables`, then the active environment. Unresolved references are left as-is.

### Dynamic values

`{{$name arg arg key=value}}` generates a value each time it is evaluated. Arguments are separated by spaces; use quotes for text with spaces (`format="MMM d, yyyy"`).

| Function | Result |
|---|---|
| `$guid`, `$uuid` | random UUID v4 |
| `$randomString [length]` | letters and digits (default 8) |
| `$randomAlpha [length]` / `$randomHex [length]` | letters only / hex digits |
| `$randomChoice a b c` | one of the given values |
| `$randomInt [min] [max]` | whole number (default 0–1000) |
| `$randomFloat [min] [max] [decimals=2]` | decimal number (default 0–1) |
| `$randomBool` | `true` or `false` |
| `$timestamp` / `$timestampMs` | Unix time in seconds / milliseconds |
| `$isoTimestamp` | current time, ISO 8601 (UTC) |
| `$date`, `$datetime`, `$time` | current date / date and time / time of day |
| `$calc <expression>` | arithmetic, see below |
| `$randomEmail`, `$randomFirstName`, `$randomLastName`, `$randomFullName`, `$randomPhone`, `$randomWord` | small built-in fake-data set (emails use `example.com`, phones a fictional `555-01xx` line) |

`$date`, `$datetime`, `$time` and `$isoTimestamp` accept an offset (`+7d`, `-2h`; units `s m h d w M y`), `format=` (`yyyy-MM-dd`, `HH:mm`, or `iso`, `rfc1123`, `unix`, `unixms`) and `tz=` (an IANA name such as `Europe/Berlin`; default UTC). Example: `{{$date +7d format="yyyy-MM-dd"}}`.

`$calc` evaluates numbers, variable names, `+ - * / %`, parentheses and `round(x, n)`, `floor`, `ceil`, `abs`, `min`, `max`; for example `{{$calc price * quantity}}` or `{{$calc round(total / 3, 2)}}`. Variable names are written bare (not in braces); their values must be numbers. Integers stay integers (`10 / 5` is `2`), decimals are trimmed (`0.1 * 3` is `0.3`). Division by zero or a non-numeric variable is an error. Nothing else can be called.

Dynamic values also work in environment variable values and in assertion `expected`, `path` and `name`. An unknown function or bad argument is an error (`clrsig validate` finds it; `clrsig run` refuses to send anything).

**When a value is generated**

- A scenario `variables` entry is generated **once per scenario run**; `{{email}}` is identical in every step and assertion of that run.
- An inline call (`{{$guid}}` in a URL) is generated **every time it appears**, so two inline calls differ.
- An environment variable that uses a function is generated **once per command** and shared by all scenarios run by that command (the desktop app generates it per scenario run).
- Random values come from a seed. The report prints it (`seed: N`) when random values were used; `clrsig run --seed N` (or `CLRSIG_SEED=N`) replays the same values. Date and time functions read the clock and are not affected by the seed.

---

## Request File

A single HTTP request saved as a `.yaml` or `.json` file.

```yaml
schemaVersion: "1"           # required — always "1" (quoted string)
title: Get User              # required — display name

method: GET                  # required — GET | POST | PUT | PATCH | DELETE | HEAD | OPTIONS
url: "{{baseUrl}}/users/{{userId}}"  # required

headers:                     # optional
  - name: Authorization
    value: "Bearer {{token}}"
  - name: Accept
    value: application/json

queryParams:                 # optional
  - name: page
    value: "1"
  - name: limit
    value: "20"

body:                        # optional
  type: json                 # none | json | raw | form-urlencoded | multipart
  content: '{"name":"Alice","email":"alice@example.com"}'

auth:                        # optional — alternative to manual auth headers
  type: bearer               # none | bearer | basic | apikey
  token: "{{token}}"         # bearer only

  # basic auth:
  # type: basic
  # username: "{{username}}"
  # password: "{{password}}"

  # api key auth:
  # type: apikey
  # key: X-API-Key            # header name or query param name
  # value: "{{apiKey}}"
  # location: header          # header | query

timeout: 5s                  # optional — per-request timeout as a duration string (500ms, 5s, 1m30s); omit to use the global default
followRedirects: false       # optional — false returns a 3xx response as it is instead of following it (default true)
```

### Notes

- `auth` and manual `Authorization` headers are mutually exclusive — use one or the other.
- `body.type: none` omits the body entirely; omitting the `body` block has the same effect.
- `Content-Type` is set automatically from `body.type` when not explicitly provided.
- `timeout` is a duration string such as `500ms`, `5s` or `1m`. Request files do **not** accept `timeoutMs`; an unknown field is silently ignored, so the timeout would not apply.
- `followRedirects: false` makes the client return the `301`/`302`/`307`/`308` response itself (with its `Location` header) instead of following it, so a redirect that the OpenAPI spec declares can be asserted and validated. Leave it out for the normal behaviour. It is also accepted on a scenario step's `request` and on a bundle request.
- `body.type: multipart` sets its own `multipart/form-data` Content-Type with a generated boundary. `content` is a JSON array of fields: `n` is the field name, `t` is `text` or `file`, and `v` is the text value or, for files, an absolute path.

  ```yaml
  body:
    type: multipart
    content: '[{"n":"title","t":"text","v":"Report"},{"n":"file","t":"file","v":"C:/data/report.pdf"}]'
  ```

  A plain `name=value&other=value` string is also accepted for text-only forms.

---

## Scenario File

A multi-step workflow for API Contract Validation and Spec Conformance Testing.

```yaml
schemaVersion: "1"           # required — always "1" (quoted string)
title: User Lifecycle        # required — display name
description: "Verifies CRUD operations for /users. Run after any changes to the users endpoint or auth middleware."
                             # optional — when to run it and what it proves

specFile: "/absolute/path/to/openapi.yaml"  # optional — absolute path or http(s) URL of an OpenAPI 3.x spec; enables schema assertions by operationId

variables:                   # optional — scenario variables, generated once per run, in order
  - name: email              # no spaces, braces or $
    value: "user-{{$randomString 10}}@example.com"   # may use functions, environment variables and earlier variables
  - name: apiToken
    value: "{{token}}"
    secret: true             # optional — masked as *** in reports

steps:
  - label: Login             # required — short action phrase ("Login", "Create order", not "Step 1")
    notes: "Obtain a bearer token. failPolicy stop — all subsequent steps depend on this."
                             # optional — why this step exists, dependencies, what it proves

    request:                 # required (unless requestRef is used)
      method: POST
      url: "{{baseUrl}}/auth/login"
      headers:
        - name: Content-Type
          value: application/json
      body:
        type: json
        content: '{"username":"{{username}}","password":"{{password}}"}'

    # requestRef: "requests/login.yaml"
    # Alternative to inline request — path relative to the scenario file.
    # Use requestRef OR request, never both.

    assertions:              # optional
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.token"
        operator: ne
        expected: ""

    extractions:             # optional — capture values for use in later steps
      - name: token
        source: jsonpath
        path: "$.token"

    failPolicy: stop         # optional — stop (default) | continue | skip
    disabled: false          # optional — true skips this step without counting as failure
    validateSpec: true       # optional — with specFile set, validate this step's response against the spec (default true); false opts the step out

    timeoutMs: 0             # optional — per-step timeout in ms; overrides the request's own timeout
    retry:                   # optional — re-send the step after a network error or a 5xx response
      maxAttempts: 3         # total attempts, 1–10 (1 = no retry)
      delayMs: 500           # wait between attempts, at most 60000

  - label: Create user
    notes: "POST to /users. Extracts the new user's ID for use in later steps."
    request:
      method: POST
      url: "{{baseUrl}}/users"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
      body:
        type: json
        content: '{"name":"Diana Prince","email":"diana@example.com"}'
    assertions:
      - type: status
        operator: eq
        expected: 201
      - type: jsonpath
        path: "$.id"
        operator: gt
        expected: 0
    extractions:
      - name: userId
        source: jsonpath
        path: "$.id"
    failPolicy: stop

  - label: Fetch user
    notes: "Verify the user is readable immediately after creation."
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.name"
        operator: eq
        expected: "Diana Prince"
    failPolicy: continue

  - label: Delete user
    notes: "Clean up — 204 No Content confirms the resource was removed."
    request:
      method: DELETE
      url: "{{baseUrl}}/users/{{userId}}"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
    assertions:
      - type: status
        operator: eq
        expected: 204
```

### failPolicy values

| Value | Behaviour |
|---|---|
| `stop` | (default) Skip all remaining steps when this step fails |
| `continue` | Run the next step regardless of this step's outcome |
| `skip` | Skip this step if the previous step failed |

### Step timeout and retry

- `timeoutMs` overrides the request's `timeout` (and the global default) for that step. `0` or omitted means no override.
- `retry` re-sends the step when the attempt fails at the network level (connection refused, timeout) or the server answers `5xx`. `maxAttempts` is the **total** number of attempts (so `3` means one try and up to two retries); `delayMs` is the wait between them.
- A failed assertion, a `4xx` response or a spec violation never triggers a retry: those are real results and retrying would hide them. Assertions, extractions and spec validation run on the last attempt only.
- Reports show the attempt count when a step was re-sent (`(3 attempts)`; `attempts` in JSON). A step counts once in totals and coverage however many attempts it took.

### validateSpec

When the scenario sets `specFile`, every step's response is also validated against the matching operation in that spec, and a violation fails the step. Set `validateSpec: false` on a step that intentionally returns something the spec does not describe, such as a negative test.

---

## Environment File

Named sets of variables. Scenarios are environment-agnostic — the same scenario file runs against dev, staging, or prod by switching environments.

```yaml
schemaVersion: "1"           # required
name: local                  # required — display name

variables:
  - name: baseUrl
    value: "http://localhost:3000"

  - name: username
    value: admin

  - name: password
    value: secret
    secret: true             # optional — redacted in request history and reports

  - name: token
    value: ""                # leave empty; filled by a scenario extraction at runtime
    secret: true

  - name: apiKey
    value: "sk-live-abc123"
    secret: true
```

### Notes

- Variable names are case-sensitive and must be valid identifiers (letters, digits, underscores).
- `secret: true` redacts the value in all logs, history, and exported reports.
- Create one environment file per target (e.g. `local.yaml`, `staging.yaml`, `production.yaml`) with the **same variable names** so scenarios are portable across environments.

---

## Assertions Reference

Every assertion evaluates one aspect of the HTTP response.

### `type: status` — HTTP status code

```yaml
- type: status
  operator: eq              # eq | ne | gt | lt
  expected: 200
```

Every assertion may also carry an optional `name:` label. It is saved with the assertion and included in the JSON report (`--format json`); text and JUnit reports always describe the assertion from its type, path, operator and expected value.

### `type: jsonpath` — JSON response body field

```yaml
- type: jsonpath
  path: "$.user.name"       # JSONPath expression
  operator: eq              # eq | ne | gt | lt | contains | matches
  expected: "Alice"
```

`matches` uses Go regular expression syntax.

### `type: header` — Response header value

```yaml
- type: header
  path: Content-Type        # header name (case-insensitive)
  operator: contains        # eq | ne | gt | lt | contains | matches (gt and lt need numeric values)
  expected: application/json
```

### `type: schema` — OpenAPI schema validation

Requires `specFile` set at the scenario level.

```yaml
- type: schema
  schema: getUserById       # operationId from the linked OpenAPI spec
```

### Operator reference

| Operator | Applies to | Meaning |
|---|---|---|
| `eq` | status, jsonpath, header | Equal |
| `ne` | status, jsonpath, header | Not equal |
| `gt` | status, jsonpath, header | Greater than (numeric) |
| `lt` | status, jsonpath, header | Less than (numeric) |
| `contains` | jsonpath, header | String contains substring |
| `matches` | jsonpath, header | Matches regular expression |

---

## Extractions Reference

Capture a value from a response to use as a variable in later steps.

### `source: jsonpath` — from JSON response body

```yaml
- name: userId
  source: jsonpath
  path: "$.id"
```

### `source: header` — from a response header

```yaml
- name: requestId
  source: header
  path: X-Request-ID
```

### `source: status` — the HTTP status code

```yaml
- name: lastStatus
  source: status
  # no path needed
```

---

## Bundle File

A **bundle** is one document that holds several environments, reusable requests and scenarios. It exists so a whole test suite can be generated, shared or pasted as a single block and imported in one step (**File → Import → Bundle (environments + scenarios)…**). Every item uses exactly the formats described above; a bundle only adds a top-level wrapper and a `file` name per item.

```yaml
schemaVersion: "1"                 # required — always "1"
title: Pet Store test suite        # optional — shown in the import preview
description: Smoke and lifecycle tests for the Pet Store API   # optional
specFile: "https://petstore3.swagger.io/api/v3/openapi.json"   # optional — default OpenAPI spec (path or URL) for every scenario

environments:                      # optional — one or more
  - name: staging                  # required
    file: staging                  # optional — saved as environments/staging.yaml (default: derived from name)
    variables:
      - name: baseUrl
        value: "https://staging.example.com"
      - name: apiKey
        value: CHANGE_ME
        secret: true

requests:                          # optional — reusable requests
  - file: check-health             # required — saved as requests/check-health.yaml; scenarios refer to it by this name
    title: Check health
    method: GET
    url: "{{baseUrl}}/health"
    timeout: 5s

scenarios:                         # optional — one or more
  - title: Smoke test              # required
    file: smoke-test               # optional — saved as scenarios/smoke-test.yaml (default: derived from title)
    description: Quick check that the API is up
    specFile: "https://other.example.com/openapi.json"   # optional — overrides the bundle default for this scenario
    steps:
      - label: Health
        requestRef: check-health   # a request from this bundle
        assertions:
          - type: status
            operator: eq
            expected: 200
```

### Rules

- At least one of `environments`, `requests` or `scenarios` must be present.
- Each item takes the same fields as the matching standalone file, except `schemaVersion` (which belongs to the bundle) and the extra `file` key.
- **Unknown fields are errors.** A bundle is checked strictly, so a typo such as `queryParam:` or a misplaced `timeoutMs` on a request is reported at import instead of being silently ignored.
- `file` is a name without extension; it may contain folders (`auth/login`). It must not contain `..`. Names must be unique within each kind.
- `requestRef` in a bundle scenario names a request from the same bundle by its `file` (`check-health`, `check-health.yaml` and `requests/check-health.yaml` all match). On import it is rewritten to the saved location, for example `../requests/check-health.yaml`. A `requestRef` that is not in the bundle and looks like a path is left unchanged and must already exist in the workspace; any other unknown name is an error.
- `specFile` resolves per scenario: the scenario's own value, otherwise the bundle's. At import time the user can override it for all scenarios, for example to point at a local file. A URL is valid; secrets and tokens must never be put in a spec URL.
- Secret values in environments should be placeholders (`CHANGE_ME`) with `secret: true`; never include real credentials.

### Import behaviour

- The bundle is validated before anything is written. Any error stops the import; warnings (for example a `{{variable}}` that no environment defines) do not. A `retry.maxAttempts` above 10 or a negative `timeoutMs` is an error.
- Files that already exist can be skipped, overwritten, or kept by saving the new file as `name-2.yaml`. When a request is renamed this way, scenarios that refer to it are updated to match.
- Files are written atomically. The result is ordinary request, scenario and environment files; the bundle itself is not stored.
- A bundle may be YAML or JSON. Text pasted from a chat window may include a surrounding ``` code fence, which is ignored.
- The same bundle can be used from the command line: `clrsig bundle run` executes every scenario without importing, `clrsig bundle import` saves the files into a workspace, and `clrsig bundle validate` / `export` check and produce bundles.

### Export behaviour

**File → Export → Bundle (environments + requests + scenarios)…** saves the open workspace as one bundle in the format above:

- Every environment, request and scenario file is included (YAML or JSON, any depth below `requests/` and `scenarios/`). Hidden files and folders are skipped, as are files that cannot be parsed; skipped files are listed after the export.
- `file` is the path under its folder without the extension, for example `auth/login`. Scenario `requestRef`s that point at an exported request are rewritten to that name, so the bundle imports cleanly into another workspace.
- If every scenario uses the same `specFile`, it is written once at the top of the bundle.
- **Secret values are replaced with `CHANGE_ME`** so the file is safe to share or paste into a chat. The workspace itself is never changed.

---

## File Naming Conventions

| File type | Convention | Example |
|---|---|---|
| Request | `kebab-case.yaml` | `get-user.yaml`, `create-order.yaml` |
| Scenario | `kebab-case.yaml` | `user-lifecycle.yaml`, `checkout-flow.yaml` |
| Environment | `name.yaml` | `local.yaml`, `staging.yaml`, `production.yaml` |

Place requests under `requests/`, scenarios under `scenarios/`, environments under `environments/` within your workspace. A bundle is an import format only and is not stored in the workspace.

---

## Schema Version

All files carry `schemaVersion: "1"` (a quoted string). This field is required and enables future format migration. Always use the value `"1"` for the current format.
