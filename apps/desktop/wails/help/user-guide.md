# Clear Signal User Guide

<!-- This file is embedded into the desktop binary via //go:embed.
     Keep it in sync with docs/user-guide.md — that copy is the canonical
     source for GitHub viewing. -->

**Version:** 1.0 (MVP)

---

## Table of Contents

1. [Opening or creating a workspace](#1-opening-or-creating-a-workspace)
2. [Working with requests](#2-working-with-requests)
3. [Understanding dirty state and saving](#3-understanding-dirty-state-and-saving)
4. [Organising requests into folders](#4-organising-requests-into-folders)
5. [Environments and variables](#5-environments-and-variables)
6. [Sending requests and reading responses](#6-sending-requests-and-reading-responses)
7. [Importing an OpenAPI specification](#7-importing-an-openapi-specification)
8. [Scenarios and contract testing](#8-scenarios-and-contract-testing)
9. [CLI usage](#9-cli-usage)
10. [CI/CD integration](#10-cicd-integration)
11. [Settings](#11-settings)
12. [File format reference](#12-file-format-reference)
13. [Keyboard shortcuts](#13-keyboard-shortcuts)
14. [Troubleshooting](#14-troubleshooting)

---

## 1. Opening or creating a workspace

A **workspace** is an ordinary folder on your filesystem. Clear Signal reads and writes files inside it; nothing is stored in a cloud account or a hidden database.

**To open an existing workspace:** click **Open Existing** in the sidebar, or use the folder icon in the workspace panel header.

**To create a new workspace:** click **New Workspace**, then pick an empty folder. Clear Signal creates a starter structure with a sample environment and a sample request so you can see the layout immediately.

The last-used workspace and active environment are remembered between sessions.

---

## 2. Working with requests

### Creating a new request

Click the `+` icon in the top toolbar (or press **Ctrl+N**). The editor clears and the header shows **New Request**.

### Naming a request

Click the **New Request** label (or any existing request name) in the top bar. It becomes an editable field. Type the name you want and press **Enter** or click away to confirm.

The name you type becomes the filename when you save:

| Name you type | File written |
|---|---|
| `List Users` | `list-users.yaml` |
| `POST /auth/login` | `post-auth-login.yaml` |
| `Health Check` | `health-check.yaml` |

Names are slugified: lowercased, spaces and underscores replaced with hyphens, special characters removed.

### Saving a request

**Ctrl+S**, or the save icon in the toolbar.

- **New request (no file yet):** saved to `{workspace}/requests/{slug}.yaml`. A workspace must be open.
- **Existing request, same name:** the file is overwritten in place.
- **Existing request, name changed:** the new slug file is written, then the old file is deleted — effectively a rename.
- **Name conflict:** if a file with the same slug already exists in the folder, saving is blocked and an error message appears next to the name. Rename the request to something unique and save again.

### Loading a request from the workspace

Click any file in the **Workspace** sidebar tab. If the current request has unsaved changes, a confirmation dialog appears.

### Opening a request file from anywhere

The file picker button in the toolbar opens a system file picker. The selected file is loaded into the editor and the workspace sidebar highlights it if it is inside the current workspace.

---

## 3. Understanding dirty state and saving

### The amber dot

Whenever the editor contains changes that have not been saved to disk, a small **amber dot** appears:

- In the tab strip, next to the tab title.
- On the file row in the workspace sidebar.

The dot disappears as soon as you save (Ctrl+S) or discard.

### Navigating away with unsaved changes

If you try to open a different file, create a new request, or close a tab while the amber dot is showing, a confirmation dialog appears with three options:

| Choice | What happens |
|---|---|
| **Save & Continue** | Saves the current request using the existing name/path, then proceeds. If the save fails (e.g. a name conflict), the dialog stays open so you can fix the error. |
| **Discard** | Abandons unsaved changes and proceeds immediately. |
| **Cancel** | Closes the dialog and returns you to the current request. |

### Tabs

Clear Signal supports multiple request tabs. Each tab maintains independent state (URL, method, body, headers, response). Click the `+` icon in the tab bar or press **Ctrl+N** to open a new tab. The close button appears on hover.

---

## 4. Organising requests into folders

Folders in the workspace sidebar are real subdirectories inside `{workspace}/requests/`. To create a folder, right-click anywhere in the workspace panel and choose **New Folder**, or create a subdirectory in your file manager.

Any `.yaml`/`.json` files inside a subfolder appear as requests under that folder in the sidebar.

When you save a request that was loaded from a subfolder, it is saved back to the same directory. The folder structure is preserved.

Requests imported from an OpenAPI spec are automatically placed in a subdirectory named after the spec title:

```
requests/
└── My API v1/
    ├── list-users.yaml
    ├── create-user.yaml
    └── get-user.yaml
```

---

## 5. Environments and variables

An **environment** is a named set of key/value variables stored in `{workspace}/environments/{name}.yaml`.

### Selecting an environment

The environment selector appears in the top bar whenever a workspace has at least one environment file. Switching environments takes effect on the next request you send.

### Creating or editing an environment

Click the pencil icon next to the environment selector. The environment editor opens as an overlay. Add, edit, or delete variables and click **Save**. The updated environment is selected automatically.

### Using variables in requests

Reference a variable anywhere in the URL, headers, query parameters, request body, or authentication fields using `{{variableName}}`:

```
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{authToken}}
```

Unresolved references are left as-is and appear literally in the outbound request, making missing configuration easy to spot.

### Secret variables

Variables marked `secret: true` in an environment file are redacted in the response history and CLI reports. For CI/CD, reference OS environment variables in secret fields:

```yaml
- name: apiToken
  value: "${API_TOKEN}"
  secret: true
```

The application substitutes `${VAR}` references from the process environment before treating the value as a secret.

---

## 6. Sending requests and reading responses

1. Set the HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) using the dropdown.
2. Enter the URL. Pasting a URL containing a query string (e.g. `https://api.example.com/items?page=1&limit=20`) automatically extracts the parameters into the **Params** tab.
3. Fill in the **Params**, **Headers**, **Body**, and **Auth** tabs as needed.
4. Press **Send** (or **Enter** in the URL bar). While the request is in progress, the button changes to **Cancel** — click it to abort.

### Request tabs

| Tab | Purpose |
|---|---|
| **Params** | Query parameters appended to the URL. Toggle rows on/off with the checkbox. |
| **Headers** | Request headers (key/value pairs). |
| **Body** | Request body — JSON, raw text, form URL-encoded, or multipart. JSON has a **Format** button that pretty-prints in place. |
| **Auth** | Authentication — Bearer token, Basic credentials, or API Key (header or query param). |

### Timeout

The per-request timeout field (bottom-left of the editor) overrides the global default from Settings. Leave it at `0` to use the global default.

### Response panel

The right panel shows:

- **Status code** — colour-coded: green (2xx), blue (3xx), amber (4xx), red (5xx). Error responses show **Network Error** in red.
- **Duration and size** — total round-trip time and response body size.
- **Timing breakdown** — a stacked bar showing DNS lookup, TCP connect, TLS handshake, time-to-first-byte, and body transfer time. Hover each segment for the exact millisecond value.
- **Body** — JSON responses are syntax-highlighted and pretty-printed. Click the toggle to switch to raw view. Click the copy icon to copy the body to the clipboard.
- **Headers** — full response header table.

### Request history

Click the **History** tab in the sidebar to see your last 100 requests. Clicking an entry re-populates the method and URL in the editor (the confirmation dialog appears first if there are unsaved changes). The **Clear history** button removes all entries.

---

## 7. Importing an OpenAPI specification

1. Open or create a workspace.
2. Click the **Import OpenAPI** button (upload icon) in the workspace panel header.
3. Select a `.yaml` or `.json` OpenAPI 3.0 or 3.1 file.
4. Clear Signal generates one request file per operation, placed in `requests/{spec-title}/`.

Path parameters become `{{paramName}}` placeholders. POST/PUT/PATCH operations get a JSON body stub based on the request schema. Operations with security requirements get an `Authorization: Bearer {{token}}` header hint.

After import, the workspace panel refreshes and you can open any generated request by clicking it in the tree.

---

## 8. Scenarios and contract testing

A **scenario** is a YAML or JSON file that runs a sequence of HTTP requests in order, extracts values from responses, passes them to later steps via `{{variable}}` interpolation, and asserts that each response matches the declared contract.

Scenarios are the primary tool for **Spec Conformance Testing**: proving that your API honours its OpenAPI specification end-to-end, in a repeatable, automatable way.

### Scenario file format

```yaml
schemaVersion: "1"
name: User lifecycle

steps:
  - name: Login
    request:
      method: POST
      url: "{{baseUrl}}/auth/login"
      body:
        type: json
        content: '{"username":"admin","password":"{{adminPassword}}"}'
    extractions:
      - name: token
        source: jsonpath
        path: "$.token"
    assertions:
      - type: status
        expected: 200
    failPolicy: stop   # stop | continue | skip

  - name: Create user
    request:
      method: POST
      url: "{{baseUrl}}/users"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
      body:
        type: json
        content: '{"name":"Alice","email":"alice@example.com"}'
    extractions:
      - name: userId
        source: jsonpath
        path: "$.id"
    assertions:
      - type: status
        expected: 201
      - type: jsonpath
        path: "$.name"
        operator: eq
        expected: Alice

  - name: Fetch user
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"
    assertions:
      - type: status
        expected: 200
      - type: schema
        schema: "#/components/schemas/User"
```

### Step fields

| Field | Description |
|---|---|
| `request` | Inline request definition. Alternatively, use `requestRef: path/to/file.yaml` to reference a saved request file. |
| `extractions` | Pull values from the response into runtime variables for subsequent steps. |
| `assertions` | Checks that must pass for the step to succeed. |
| `failPolicy` | What to do if an assertion fails: `stop` (default — skips remaining steps), `continue` (keeps going), `skip` (skips this step if a previous one failed). |
| `retry.maxAttempts` | Retry the step up to N times before marking it failed. |
| `retry.delay` | Wait duration between retries (e.g. `2s`, `500ms`). |

### Assertion types

| Type | Description | Fields |
|---|---|---|
| `status` | HTTP status code | `expected` (integer) |
| `header` | Response header value | `path` (header name), `operator`, `expected` |
| `jsonpath` | JSON value at a JSONPath expression | `path` (JSONPath), `operator`, `expected` |
| `schema` | Validates the response body against an OpenAPI response schema | `schema` (`"#/components/schemas/Name"` or inline JSON Schema) |

### Operators

| Operator | Meaning |
|---|---|
| `eq` (default) | Equal to |
| `ne` | Not equal to |
| `contains` | String contains |
| `matches` | Regular expression match |
| `gt` | Greater than (numeric) |
| `lt` | Less than (numeric) |

### Extraction sources

| Source | Description |
|---|---|
| `jsonpath` | Extracts from the response body using a JSONPath expression |
| `header` | Extracts a response header value |
| `status` | Extracts the numeric status code |

---

## 9. CLI usage

The CLI uses the same Go execution engine as the desktop application. Build it once and use it in any terminal or CI pipeline — no behavioural differences between interactive testing and CI.

### Build

```bash
cd apps/cli
go build -o clrsig .
```

Or run directly without building:

```bash
go run ./apps/cli run <scenario-file>
```

### `clrsig run`

Executes a scenario file and prints results.

```bash
clrsig run ./scenarios/user-lifecycle.yaml
clrsig run ./scenarios/user-lifecycle.yaml --env ./environments/dev.yaml
clrsig run ./scenarios/user-lifecycle.yaml --format json
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
Scenario: User lifecycle

  PASS  Login  200  43ms
         ✓ status eq 200
         → token = "demo-token-12345"
  PASS  Create user  201  18ms
         ✓ status eq 201
         ✓ jsonpath $.name eq Alice
         → userId = "3"
  PASS  Fetch user  200  11ms
         ✓ status eq 200
         ✓ schema #/components/schemas/User

PASSED  3 passed, 0 failed, 0 skipped  (5/5 assertions passed)  72ms
```

### JSON output

Use `--format json` for machine-readable output suitable for CI tooling:

```json
{
  "scenario": "User lifecycle",
  "passed": true,
  "durationMs": 72,
  "steps": [
    {
      "name": "Login",
      "passed": true,
      "statusCode": 200,
      "durationMs": 43,
      "extracted": { "token": "demo-token-12345" },
      "assertions": [
        { "type": "status", "operator": "eq", "expected": "200", "actual": "200", "passed": true }
      ]
    }
  ]
}
```

---

## 10. CI/CD integration

The CLI is the natural integration point for pipelines. Build it as part of your workflow, run your scenarios, and act on the exit code.

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

      - name: Build Clear Signal CLI
        run: go build -o clrsig ./apps/cli

      - name: Run contract tests
        env:
          API_BASE_URL: ${{ secrets.API_BASE_URL }}
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: |
          ./clrsig run ./scenarios/user-lifecycle.yaml \
            --env ./environments/ci.yaml \
            --format json > results.json

      - name: Upload results
        if: always()
        uses: actions/upload-artifact@v4
        with:
          name: contract-test-results
          path: results.json
```

### CI environment file

Create `environments/ci.yaml` (committed — no secrets):

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

## 11. Settings

Open via the gear icon in the toolbar.

| Setting | Description |
|---|---|
| **Theme** | Dark or light. Applied immediately; reverts if you cancel. |
| **Font size** | Scales all text in the editor and response viewer (10–20 px). |
| **Default timeout** | Request timeout in milliseconds (applies when the per-request timeout is 0). |
| **Proxy URL** | HTTP/HTTPS proxy for all outbound requests (e.g. `http://proxy:8080`). Leave empty to use the system proxy. |
| **TLS skip verify** | Disable certificate verification. Use only for local development with self-signed certificates. |

Changes preview immediately. Click **Save** to persist; **Cancel** reverts to the last saved values.

Settings are stored in `~/.clrsig/settings.json`.

---

## 12. File format reference

All project files are human-readable YAML (or JSON equivalents). Every file carries a `schemaVersion` field — the current version is `"1"`.

### Request file

```yaml
schemaVersion: "1"
name: Get user by ID

request:
  method: GET
  url: "{{baseUrl}}/users/{{userId}}"
  headers:
    - name: Accept
      value: application/json
  queryParams:
    - name: include
      value: orders
  auth:
    type: bearer
    token: "{{apiToken}}"
  timeout: 10000   # milliseconds; 0 = use global default

assertions:
  - type: status
    expected: 200
  - type: jsonpath
    path: "$.id"
    operator: eq
    expected: "{{userId}}"
  - type: schema
    schema: "#/components/schemas/User"
```

### Environment file

```yaml
schemaVersion: "1"
name: local

variables:
  - name: baseUrl
    value: http://localhost:8080

  - name: userId
    value: "1"

  - name: apiToken
    value: "${API_TOKEN}"
    secret: true
```

### Authentication types

```yaml
# Bearer token
auth:
  type: bearer
  token: "{{apiToken}}"

# Basic authentication
auth:
  type: basic
  username: admin
  password: "{{adminPassword}}"

# API key — in a header
auth:
  type: apikey
  key: X-API-Key
  value: "{{apiKey}}"
  location: header

# API key — as a query parameter
auth:
  type: apikey
  key: api_key
  value: "{{apiKey}}"
  location: query
```

### Scenario file

See [Section 8](#8-scenarios-and-contract-testing) for the full scenario format.

---

## 13. Keyboard shortcuts

| Shortcut | Action |
|---|---|
| **Enter** (URL field focused) | Send the current request |
| **Ctrl+N** | New request tab |
| **Ctrl+S** | Save the current request |

---

## 14. Troubleshooting

### The app does not open after `wails dev`

- Ensure a C compiler (`gcc`) is installed and on your PATH.
- On Windows, confirm WebView2 is installed (it ships with Windows 10 and 11; on older systems it may need a separate install from the Microsoft website).
- Check the terminal for Go compilation errors — a syntax error in any Go file prevents the app from starting.

### TypeScript binding errors after editing Go types

The `wailsjs/` directory is generated and gitignored. If you change a Go struct that is bound to Wails, stale TypeScript types may cause compilation errors. Restart `wails dev` to regenerate the bindings.

### `wails build` fails with "no matching versions for query"

This happens when the Wails binding generator runs outside the Go workspace context. The `replace` directive in `apps/desktop/wails/go.mod` normally handles this. If you see the error, verify the directive is present:

```
replace github.com/fuvito/api-workbench/packages/core => ../../../packages/core
```

### `wails dev` shows a blank screen

The frontend may have failed to compile. Check the terminal for Vite or TypeScript errors. The most common cause is a stale `wailsjs/` directory — restart `wails dev`.

### Go tests fail in CI but pass locally

The executor tests use `httptest.NewServer` and require a free port. No external network access is required; all tests run against `localhost`. If the CI runner restricts networking, the tests will fail — ensure the runner allows loopback connections.

### A `{{variable}}` appears literally in the sent request

The variable is not defined in the active environment. Check the environment editor (pencil icon in the top bar) and confirm the variable name matches exactly, including case.

### The workspace sidebar is empty after opening a workspace

The workspace root must contain a `requests/` directory. If you opened a folder that has request files at the top level (not in `requests/`), create a `requests/` subdirectory and move the files there.
