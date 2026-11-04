# EndpointDesk — User Guide

**Version:** pre-MVP (Phases 0–3 complete)  
**Last updated:** 2026-11-03

---

## Table of Contents

1. [What is EndpointDesk](#1-what-is-endpointdesk)
2. [Installing for development](#2-installing-for-development)
3. [Running the application](#3-running-the-application)
4. [The request editor](#4-the-request-editor)
5. [The response viewer](#5-the-response-viewer)
6. [Project file format](#6-project-file-format)
7. [Variable interpolation](#7-variable-interpolation)
8. [Authentication](#8-authentication)
9. [Environments](#9-environments)
10. [Collections](#10-collections)
11. [Scenarios and contract testing](#11-scenarios-and-contract-testing)
12. [CLI usage](#12-cli-usage)
13. [CI/CD integration](#13-cicd-integration)
14. [Keyboard shortcuts](#14-keyboard-shortcuts)
15. [Troubleshooting](#15-troubleshooting)

---

## 1. What is EndpointDesk

EndpointDesk is an open-source, local-first API development and testing workbench. Its primary purpose is **API Contract Validation** and **Spec Conformance Testing**: verifying that your APIs actually honour the contracts they declare — OpenAPI specifications, response schemas, and assertion-based rules — and proving that in a repeatable, automatable way.

It is also a full API client. You can explore endpoints interactively, build collections, manage environments, and run multi-step scenarios. But unlike general-purpose API clients, every feature is designed to support the testing and validation use case.

### Core principles

- **Local-first.** No account required. All data lives on your machine.
- **Git-native.** Project definitions are plain YAML or JSON files. Commit them, diff them, review them.
- **CLI and GUI parity.** The same Go engine runs in both. A test that passes in the GUI passes in a pipeline.
- **OpenAPI first.** OpenAPI 3.0 and 3.1 specifications are a first-class input, not an afterthought.

### Terminology

| Term | Meaning |
|---|---|
| **API Contract Validation** | Verifying that an API's actual HTTP responses match its declared contract (schema, status codes, headers). |
| **Spec Conformance Testing** | Running a complete test suite that proves an API conforms to its specification across all endpoints and flows. |
| **Workspace** | A directory on disk that contains your project files (collections, environments, scenarios). |
| **Collection** | A named, hierarchical set of saved requests. |
| **Environment** | A named set of variables (e.g. `baseUrl`, `apiKey`) that can be switched without modifying requests. |
| **Scenario** | A multi-step workflow that executes requests in sequence, extracts values, and asserts contract conformance. |

---

## 2. Installing for development

> **Note:** Pre-built installers are not yet available. The application is built from source during early development.

### Prerequisites

| Tool | Minimum version | Installation |
|---|---|---|
| Go | 1.26 | https://go.dev/dl |
| Node.js | 22 | https://nodejs.org |
| pnpm | 9 | `npm install -g pnpm` |
| Wails CLI | v2 (latest) | `go install github.com/wailsapp/wails/v2/cmd/wails@latest` |
| C compiler | — | Platform-specific (see below) |

### C compiler setup

Wails requires a C compiler to build the WebView bridge.

**Windows:** Install [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or the GCC toolchain via [MSYS2](https://www.msys2.org/). After installation, ensure `gcc` is on your PATH.

**macOS:** Install Xcode Command Line Tools: `xcode-select --install`

**Linux:** Install `gcc` and WebKit development headers:
```bash
# Ubuntu / Debian
sudo apt install gcc libgtk-3-dev libwebkit2gtk-4.1-dev

# Fedora
sudo dnf install gcc gtk3-devel webkit2gtk4.1-devel
```

### Clone and install

```bash
git clone https://github.com/fuvito/api-workbench.git
cd api-workbench

# Install frontend dependencies
pnpm install --dir apps/desktop/frontend
```

---

## 3. Running the application

### Development mode (recommended for contributors)

```bash
cd apps/desktop/wails
wails dev
```

This starts the application with hot-reload. Changes to Go files restart the backend; changes to TypeScript/React files update the UI instantly without restarting the app.

Wails will also regenerate the TypeScript bindings in `apps/desktop/frontend/wailsjs/` on startup. These files are gitignored — do not commit them.

### Production build

```bash
cd apps/desktop/wails
wails build
```

The binary is written to:
- **Windows:** `apps/desktop/wails/build/bin/endpointdesk.exe`
- **macOS:** `apps/desktop/wails/build/bin/endpointdesk`
- **Linux:** `apps/desktop/wails/build/bin/endpointdesk`

### Running Go tests

```bash
# From the repo root — tests all core packages
go test ./packages/core/...

# Verbose output
go test ./packages/core/... -v

# A specific package
go test ./packages/core/executor/...
```

---

## 4. The request editor

The request editor is on the left side of the main panel.

### URL bar

```
[ METHOD ▼ ] [ URL input field                                    ] [ Send ]
```

- **Method selector** — choose from GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS.
- **URL field** — enter the full URL. Supports `{{variable}}` references (see [Variable interpolation](#7-variable-interpolation)). Press **Enter** to send. Pasting a URL that includes a query string (e.g. `?page=1&limit=20`) automatically strips the query string from the URL field and populates the Params tab.
- **Send button** — sends the request. Disabled if the URL is empty or a request is in progress.

### Tabs

| Tab | Purpose |
|---|---|
| **Params** | Query parameters appended to the URL. Toggle rows on/off with the checkbox. |
| **Headers** | Request headers (key/value pairs). |
| **Body** | Request body — JSON, raw text, or form URL-encoded. |
| **Auth** | Authentication — Bearer token, Basic credentials, or API Key. |

### Request body types

| Type | Content-Type set automatically |
|---|---|
| `json` | `application/json` |
| `form-urlencoded` | `application/x-www-form-urlencoded` |
| `raw` | None (set manually in the Headers tab) |
| `multipart` | None (Phase 2) |
| `none` | Nothing sent |

If you set `Content-Type` explicitly in the Headers tab, it overrides the automatic value.

### Timeout

Requests use a default timeout of 30 seconds. Per-request timeouts can be set in the project file format (see [Project file format](#6-project-file-format)). A configurable UI control is planned for Phase 2.

---

## 5. The response viewer

The response viewer is on the right side of the main panel.

### Status bar

When a response is received, the status bar shows:

```
[ 200 ]  200 OK                                    142 ms   1.2 KB
```

- **Status code** — colour-coded: green (2xx), blue (3xx), amber (4xx), red (5xx).
- **Status text** — the full HTTP status line from the server.
- **Duration** — time from sending the request to receiving the last byte of the response body.
- **Size** — size of the response body.

### Body tab

The response body is displayed as pretty-printed JSON if the content is valid JSON, or as raw text otherwise.

> Syntax highlighting and search are planned for Phase 2.

### Headers tab

Lists all response headers as a key/value table. Multiple values for the same header name appear as separate rows.

### Error state

If the request fails before a response is received (network error, DNS failure, timeout, context cancellation), the error is displayed in the response panel instead of a status code.

---

## 6. Project file format

All project definitions are saved and loaded through the UI (Open / Save toolbar buttons) or by clicking files in the Workspace panel. Files can also be authored by hand and opened directly.

All project definitions are saved as YAML (default) or JSON. Every file carries a `schemaVersion` field. Version 1 is the current schema.

### Request file

```yaml
schemaVersion: "1"
name: Get customer by ID
description: Retrieves a single customer record.

request:
  method: GET
  url: "{{baseUrl}}/customers/{{customerId}}"

  headers:
    - name: Accept
      value: application/json

  queryParams:
    - name: include
      value: orders

  auth:
    type: bearer
    token: "{{apiToken}}"

  timeout: 30s

assertions:
  - type: status
    expected: 200

  - type: jsonpath
    path: "$.id"
    operator: eq
    expected: "{{customerId}}"

  - type: schema
    # Validates the response body against the referenced OpenAPI response schema (Phase 4)
    schema: "#/components/schemas/Customer"
```

**JSON equivalent:**

```json
{
  "schemaVersion": "1",
  "name": "Get customer by ID",
  "request": {
    "method": "GET",
    "url": "{{baseUrl}}/customers/{{customerId}}",
    "headers": [
      { "name": "Accept", "value": "application/json" }
    ],
    "auth": {
      "type": "bearer",
      "token": "{{apiToken}}"
    }
  },
  "assertions": [
    { "type": "status", "expected": 200 }
  ]
}
```

### Assertion types

| Type | Description | Required fields |
|---|---|---|
| `status` | HTTP status code | `expected` (integer) |
| `header` | Response header value | `path` (header name), `expected`, `operator` |
| `jsonpath` | JSON value at a JSONPath expression | `path` (JSONPath), `expected`, `operator` |
| `schema` | Response body validates against a JSON Schema | `schema` (inline schema or `$ref`) |

### Operators

| Operator | Meaning |
|---|---|
| `eq` (default) | Equal to |
| `ne` | Not equal to |
| `contains` | String contains |
| `matches` | Regular expression match |
| `gt` | Greater than (numeric) |
| `lt` | Less than (numeric) |

### Environment file

```yaml
schemaVersion: "1"
name: local

variables:
  - name: baseUrl
    value: http://localhost:8080

  - name: customerId
    value: "42"

  - name: apiToken
    value: "${API_TOKEN}"   # references an OS environment variable
    secret: true            # value is redacted in logs and reports
```

### Collection file

```yaml
schemaVersion: "1"
name: Customer API

auth:
  type: bearer
  token: "{{apiToken}}"

defaultHeaders:
  - name: Accept
    value: application/json

items:
  - type: folder
    folder:
      name: Customers
      items:
        - type: request
          request:
            name: List customers
            method: GET
            url: "{{baseUrl}}/customers"
        - type: request
          request:
            name: Get customer
            method: GET
            url: "{{baseUrl}}/customers/{{customerId}}"
```

### Scenario file

```yaml
schemaVersion: "1"
name: Create and retrieve customer

steps:
  - name: Create customer
    request:
      method: POST
      url: "{{baseUrl}}/customers"
      body:
        type: json
        content: '{"name": "Alice", "email": "alice@example.com"}'
    extractions:
      - name: createdId
        source: jsonpath
        path: "$.id"
    assertions:
      - type: status
        expected: 201
    failPolicy: stop

  - name: Retrieve customer
    request:
      method: GET
      url: "{{baseUrl}}/customers/{{createdId}}"
    assertions:
      - type: status
        expected: 200
      - type: jsonpath
        path: "$.name"
        expected: Alice
      - type: schema
        schema: "#/components/schemas/Customer"
```

---

## 7. Variable interpolation

Variables are written as `{{variableName}}` anywhere a string value appears: URLs, headers, query parameters, request body content, and authentication values.

### Resolution order

Variables are resolved in this order (later sources override earlier ones):

1. **Environment variables** — loaded from the active environment file.
2. **Runtime variables** — extracted from earlier scenario steps using `extractions`.

### Unresolved references

If a variable is referenced but has no value in any source, the `{{variableName}}` placeholder is left unchanged in the request. This makes it easy to spot missing configuration.

### Secret variables

Variables marked `secret: true` in an environment file are:
- Redacted as `[REDACTED]` in the response viewer history.
- Redacted in exported reports.
- Not echoed in CLI output.

For CI/CD, reference OS environment variables in secret fields:

```yaml
- name: apiToken
  value: "${API_TOKEN}"
  secret: true
```

The application substitutes `${VAR}` references from the process environment before treating the value as a secret.

---

## 8. Authentication

Authentication is configured per request or inherited from a collection.

### Bearer token

```yaml
auth:
  type: bearer
  token: "{{apiToken}}"
```

Adds `Authorization: Bearer <token>` to the request.

### Basic authentication

```yaml
auth:
  type: basic
  username: admin
  password: "{{adminPassword}}"
```

Adds a standard `Authorization: Basic <base64>` header.

### API key

```yaml
auth:
  type: apikey
  key: X-API-Key
  value: "{{apiKey}}"
  location: header   # or: query
```

- `location: header` — adds the key/value as a request header.
- `location: query` — appends the key/value as a query parameter.

### Inheritance

Requests inherit authentication from their parent collection if they do not define their own `auth` block. Set `auth: { type: none }` on a request to explicitly disable inherited auth.

---

## 9. Environments

Environments let you switch between configurations (local, dev, staging, production) without modifying request definitions.

### Switching environments

The environment selector appears in the top bar whenever a workspace contains environment files. Selecting an environment applies its variables immediately — the next request will use the new values without restarting the app. The active environment is persisted across restarts.

### Environment file location

By convention, environment files live in an `environments/` directory inside the workspace:

```
my-project/
├── environments/
│   ├── local.yaml
│   ├── dev.yaml
│   └── staging.yaml
├── collections/
└── scenarios/
```

### Keeping secrets out of version control

Add environment files that contain real secrets to `.gitignore`. Provide a `*.example.yaml` template with placeholder values that is committed:

```
# .gitignore
environments/local.yaml
environments/staging.yaml

# Commit this:
environments/local.example.yaml
```

---

## 10. Collections

Collections organise requests into a named, hierarchical structure. They are stored as a single YAML or JSON file, or as a directory tree where each file is one request.

The Workspace sidebar shows all request files in a collapsible tree. Click any file to load it into the request editor. The Open Workspace and New Workspace buttons appear when no workspace is set.

### Directory-based collections

For Git-friendly workflows, you can store each request as a separate file:

```
my-project/
└── collections/
    └── customer-api/
        ├── collection.yaml      # collection metadata
        ├── list-customers.yaml  # individual request
        ├── get-customer.yaml
        └── create-customer.yaml
```

The application discovers all `.yaml` and `.json` files in the workspace and assembles the collection tree from them.

---

## 11. Scenarios and contract testing

Scenarios are the primary tool for **Spec Conformance Testing**. A scenario:

1. Executes requests in sequence.
2. Extracts values from responses (e.g. a created resource ID).
3. Passes extracted values to subsequent steps via `{{variable}}` interpolation.
4. Asserts that each response matches the declared contract.

### Writing a scenario

See the [Scenario file format](#scenario-file) in the previous section. Key fields:

| Field | Description |
|---|---|
| `steps[].request` | Inline request definition. Alternatively, use `steps[].requestRef` to reference a saved collection request. |
| `steps[].extractions` | Pull values from the response into runtime variables. |
| `steps[].assertions` | Checks that must pass for the step to succeed. |
| `steps[].failPolicy` | What to do if an assertion fails: `stop` (default), `continue`, or `skip`. |
| `steps[].retry.maxAttempts` | Retry the step up to N times before marking it failed. |
| `steps[].retry.delay` | Wait duration between retries (e.g. `2s`, `500ms`). |

### Assertion types for contract testing

For contract validation, the most important assertion types are:

**`status`** — verify the HTTP status code matches the OpenAPI operation's declared response code.

```yaml
assertions:
  - type: status
    expected: 200
```

**`schema`** — validate the response body against a JSON Schema (inline or referenced from an OpenAPI document). This is the core of API Contract Validation.

```yaml
assertions:
  - type: schema
    schema: |
      {
        "type": "object",
        "required": ["id", "name"],
        "properties": {
          "id":   { "type": "integer" },
          "name": { "type": "string" }
        }
      }
```

**`jsonpath`** — assert a specific value at a JSONPath expression.

```yaml
assertions:
  - type: jsonpath
    path: "$.status"
    operator: eq
    expected: active
```

**`header`** — assert a response header value.

```yaml
assertions:
  - type: header
    path: Content-Type
    operator: contains
    expected: application/json
```

Assertion evaluation is fully implemented for `status`, `header`, and `jsonpath` types. `schema` assertions are reserved for Phase 4 (OpenAPI integration).

---

## 12. CLI usage

The CLI uses the same Go execution engine as the desktop application. Build it once and use it in any terminal or CI pipeline.

### Build

```bash
cd apps/cli
go build -o endpointdesk .
```

Or run directly without building:

```bash
go run ./apps/cli run <scenario-file>
```

### `endpointdesk run`

Executes a scenario file and prints results.

```bash
endpointdesk run ./scenarios/create-and-retrieve.yaml
endpointdesk run ./scenarios/create-and-retrieve.yaml --env ./environments/dev.yaml
endpointdesk run ./scenarios/create-and-retrieve.yaml --format json
```

| Flag | Default | Description |
|---|---|---|
| `--env`, `-e` | (none) | Path to an environment file (YAML or JSON). |
| `--format`, `-f` | `text` | Output format: `text` (human-readable) or `json` (machine-readable). |

### Exit codes

| Code | Meaning |
|---|---|
| `0` | All steps passed all assertions. |
| `1` | One or more assertions failed. |
| `2` | A network, parse, or configuration error occurred. |

### Text output example

```
Scenario: Create and retrieve customer
          Verifies the full create → fetch workflow

  PASS  Create customer  201  43ms
         ✓ status eq 201
         → customerId = "42"
  PASS  Retrieve customer  200  12ms
         ✓ status eq 200
         ✓ jsonpath $.name eq Alice

PASSED  2 passed, 0 failed, 0 skipped  (3/3 assertions passed)  55ms
```

### JSON output

Use `--format json` to get a machine-readable report suitable for CI tooling:

```bash
endpointdesk run ./scenarios/flow.yaml --env ./environments/ci.yaml --format json
```

```json
{
  "scenario": "Create and retrieve customer",
  "passed": true,
  "durationMs": 55,
  "steps": [
    {
      "name": "Create customer",
      "passed": true,
      "statusCode": 201,
      "durationMs": 43,
      "extracted": { "customerId": "42" },
      "assertions": [
        { "type": "status", "operator": "eq", "expected": "201", "actual": "201", "passed": true }
      ]
    }
  ]
}
```

---

## 13. CI/CD integration

The CLI produces a machine-readable JSON report suitable for any CI system.

### GitHub Actions example

```yaml
name: API Contract Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  contract-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Build EndpointDesk CLI
        run: go build -o endpointdesk ./apps/cli

      - name: Run contract tests
        env:
          API_BASE_URL: ${{ secrets.API_BASE_URL }}
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: |
          ./endpointdesk run ./scenarios/create-and-retrieve.yaml \
            --env ./environments/ci.yaml \
            --format json > test-results.json
```

### CI environment file

Create `environments/ci.yaml` (committed, no secrets):

```yaml
schemaVersion: "1"
name: ci

variables:
  - name: baseUrl
    value: "${API_BASE_URL}"

  - name: apiToken
    value: "${API_TOKEN}"
    secret: true
```

Secrets are injected from environment variables at runtime. No secrets appear in committed files.

---

## 14. Keyboard shortcuts

| Shortcut | Action |
|---|---|
| `Enter` (URL field focused) | Send the current request |

---

## 15. Troubleshooting

### The app does not open after `wails dev`

- Ensure a C compiler (`gcc`) is installed and on your PATH.
- On Windows, confirm WebView2 is installed (it ships with Windows 10 and 11; on older systems it may need a separate install).
- Check the terminal for Go compilation errors — a syntax error in any Go file prevents the app from starting.

### TypeScript binding errors after editing Go types

The `wailsjs/` directory is generated and gitignored. If you change a Go struct that is bound to Wails, the stale TypeScript types in memory will conflict. Restart `wails dev` to regenerate the bindings.

### `wails build` fails with "no matching versions for query"

This happens when the Wails binding generator runs outside the Go workspace context. The `replace` directive in `apps/desktop/wails/go.mod` normally fixes this. If you see the error, verify the directive is present:

```
replace github.com/fuvito/api-workbench/packages/core => ../../../packages/core
```

### Go tests fail on the CI but pass locally

The executor tests use `httptest.NewServer` and require a free port. If the CI runner restricts networking, the tests will fail. No external network access is required; all tests run against `localhost`.

### `wails dev` shows a blank screen

The frontend may have failed to compile. Check the terminal for Vite or TypeScript errors. The most common cause is a stale `wailsjs/` directory — restart `wails dev`.

---

*EndpointDesk is in active development. This guide is updated with each completed phase.*
