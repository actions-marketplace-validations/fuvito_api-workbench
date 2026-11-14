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

Variables are resolved from the active environment first, then from values extracted by prior scenario steps. Unresolved references are left as-is.

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
  type: json                 # none | json | raw | form-urlencoded
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

timeoutMs: 5000              # optional — per-request timeout in ms (0 = use global default)
```

### Notes

- `auth` and manual `Authorization` headers are mutually exclusive — use one or the other.
- `body.type: none` omits the body entirely; omitting the `body` block has the same effect.
- `Content-Type` is set automatically from `body.type` when not explicitly provided.

---

## Scenario File

A multi-step workflow for API Contract Validation and Spec Conformance Testing.

```yaml
schemaVersion: "1"           # required — always "1" (quoted string)
title: User Lifecycle        # required — display name
description: "Verifies CRUD operations for /users. Run after any changes to the users endpoint or auth middleware."
                             # optional — when to run it and what it proves

specFile: "/absolute/path/to/openapi.yaml"  # optional — absolute path to OpenAPI spec; enables schema assertions by operationId

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
    timeoutMs: 0             # optional — 0 = use global default
    disabled: false          # optional — true skips this step without counting as failure

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
  operator: contains        # eq | ne | contains | matches
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
| `gt` | status, jsonpath | Greater than (numeric) |
| `lt` | status, jsonpath | Less than (numeric) |
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

## File Naming Conventions

| File type | Convention | Example |
|---|---|---|
| Request | `kebab-case.yaml` | `get-user.yaml`, `create-order.yaml` |
| Scenario | `kebab-case.yaml` | `user-lifecycle.yaml`, `checkout-flow.yaml` |
| Environment | `name.yaml` | `local.yaml`, `staging.yaml`, `production.yaml` |

Place requests under `requests/`, scenarios under `scenarios/`, environments under `environments/` within your workspace.

---

## Schema Version

All files carry `schemaVersion: "1"` (a quoted string). This field is required and enables future format migration. Always use the value `"1"` for the current format.
