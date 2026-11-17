# Clear Signal User Guide

**Version:** 0.3.0

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
9. [Contracts — coverage tracking](#9-contracts--coverage-tracking)
10. [CLI usage](#10-cli-usage)
11. [CI/CD integration](#11-cicd-integration)
12. [Settings](#12-settings)
13. [File format reference](#13-file-format-reference)
14. [Creating files with AI assistance](#14-creating-files-with-ai-assistance)
15. [Keyboard shortcuts](#15-keyboard-shortcuts)
16. [Troubleshooting](#16-troubleshooting)

---

## 1. Opening or creating a workspace

A **workspace** is an ordinary folder on your filesystem. Clear Signal reads and writes files inside it; nothing is stored in a cloud account or a hidden database.

**To open an existing workspace:** use **File → Open Workspace** in the menu bar, or click the folder icon at the top of the Explorer panel in the sidebar.

**To create a new workspace:** use **File → New Workspace**, then pick an empty folder. Clear Signal creates a starter structure with a sample environment and a sample request so you can see the layout immediately.

When a workspace is opened, the first request file found in the tree is loaded automatically into the editor — no need to click a file manually to get started.

**Switching workspaces:** opening or switching to a workspace (via the File menu, the Open Workspace button, or a recent workspace entry) closes the current workspace and all its tabs first. If any tabs have unsaved changes, a confirmation dialog appears before the switch.

The last-used workspace and active environment are remembered between sessions.

**Recent workspaces:** The landing page (shown when no workspace is open) lists your recently used workspaces. If you have opened a workspace before, a **Resume** button appears at the top — it reopens the most recent workspace in one click. The File menu → Recent Workspaces submenu lists up to ten recent paths from anywhere in the app.

---

## 2. Working with requests

### Creating a new request

Click the `+` icon in the top toolbar (or press **Ctrl+N**). The editor clears and the header shows **New Request**.

### Naming a request

Click the **New Request** label (or any existing request name) in the request toolbar above the URL bar. It becomes an editable field. Type the name you want and press **Enter** or click away to confirm.

The name you type becomes the filename when you save:

| Name you type | File written |
|---|---|
| `List Users` | `list-users.yaml` |
| `POST /auth/login` | `post-auth-login.yaml` |
| `Health Check` | `health-check.yaml` |

Names are slugified: lowercased, spaces and underscores replaced with hyphens, special characters removed.

### Saving a request

**Ctrl+S**, or the **Save** button in the request editor toolbar.

- **New request (no file yet):** saved to `{workspace}/requests/{slug}.yaml`. A workspace must be open.
- **Existing request, same name:** the file is overwritten in place.
- **Existing request, name changed:** the new slug file is written, then the old file is deleted — effectively a rename.
- **Name conflict:** if a file with the same slug already exists in the folder, saving is blocked and an error message appears next to the name. Rename the request to something unique and save again.

### Loading a request from the workspace

Click any file in the **Workspace** sidebar tab. If the current request has unsaved changes, a confirmation dialog appears.

### Opening a request file from anywhere

The folder icon in the request toolbar opens a system file picker. The selected file is loaded into the editor and the workspace sidebar highlights it if it is inside the current workspace.

**Tab context menu:** right-click any tab to get additional actions: **Close**, **Close Others**, **Close to the Right**, **Close All Tabs**, **Copy Path** (copies the file path to the clipboard), and **Reveal in Explorer** (opens the file's containing folder in the system file manager).

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

When more tabs are open than fit in the tab bar, gradient fade indicators appear at the left and right edges. The active tab scrolls into view automatically whenever you switch tabs.

**Quick open / command palette (Ctrl+P):** press **Ctrl+P** to open the command palette. It lists every request file, scenario, and environment in the current workspace, plus two commands (New Request, New Scenario). Type any part of a name to filter; use the arrow keys to navigate; press **Enter** or click to open. Selecting an environment switches it immediately. Press **Escape** to dismiss.

Type **`>`** at the start of the query to enter command mode — only commands and environments are shown, so you can switch environments or trigger actions without leaving the keyboard. The footer hint at the bottom of the palette reminds you of the `>` shortcut.

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

The environment selector appears in the **status bar** at the bottom of the window whenever a workspace has at least one environment file. Switching environments takes effect on the next request you send.

### Creating or editing an environment

Click the pencil icon in the status bar (bottom of the window) next to the active environment name. The environment editor opens as an overlay. Add, edit, or delete variables and click **Save**. The updated environment is selected automatically.

### Using variables in requests

Reference a variable anywhere in the URL, headers, query parameters, request body, or authentication fields using `{{variableName}}`:

```
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{authToken}}
```

**Autocomplete:** typing `{{` in the URL bar or in any value field triggers a dropdown that lists all variables defined in the active environment. Use the arrow keys to navigate the list and press **Enter** or click an item to insert it. The dropdown closes automatically if no environment is active or no variables are defined.

Unresolved references are left as-is and appear literally in the outbound request, making missing configuration easy to spot.

**Missing variable hint:** when a `{{variable}}` in the URL bar cannot be resolved against the active environment, an amber **⚠ varName** badge appears below the URL. Click the badge to open the environment editor with that variable name pre-filled as a new row — just enter the value and save.

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
- **Body** — JSON responses are syntax-highlighted and pretty-printed. Click the toggle to switch between pretty and raw view. The search icon (Ctrl+F) opens an in-panel find bar. The copy icon copies the body to the clipboard.
- **Diff view** — after sending the same request more than once, a diff icon (⇄) appears in the toolbar. Click it to compare the current and previous response bodies side by side.
- **New scenario from request** — the list icon in the toolbar opens a dialog. If you have no existing scenarios in the workspace, it creates a new scenario file with the current request as the first step. If scenarios already exist, the dialog shows two tabs: **New** (create a new scenario file) and **Add to existing** (pick a scenario from the list to append the current request as a new step). The scenario opens immediately in an editor tab either way.
- **Headers** — full response header table with a **Copy all** button that copies every header as `name: value` lines to the clipboard.

**Collapsing the response panel:** click the chevron icon on the left edge of the status bar to collapse the response panel to a narrow strip, giving the request editor more room. Click the chevron again to restore it.

### Request history

Click the **History** rail tab (clock icon) to see your last 100 requests, grouped by date — **Today**, **Yesterday**, and older dates. Each entry shows the method badge, status code, duration, and URL. 

- **Click an entry** to restore its method, URL, headers, body, and auth into the editor. A confirmation dialog appears first if the current tab has unsaved changes. Auth is restored only when its values were `{{variable}}` references at the time the request was sent — literal tokens are never written to the history database.
- **Re-run button (↺):** hover over any entry to reveal a re-run button on the right edge. Clicking it restores the request *and* immediately sends it, so you can replay a previous call in one click.
- **Clear history** shows a confirmation prompt before removing all entries.

Network errors (DNS failure, connection refused, TLS errors, timeout) are recorded in history with a **Network Error** status and 0ms duration, so failed requests appear alongside successful ones.

---

## 7. Importing an OpenAPI specification

1. Open or create a workspace.
2. Click the **Import OpenAPI** button (upload icon) in the workspace panel header.
3. Select a `.yaml` or `.json` OpenAPI 3.0 or 3.1 file.
4. Clear Signal generates one request file per operation, placed in `requests/{spec-title}/`.

Path parameters become `{{paramName}}` placeholders. POST/PUT/PATCH operations get a JSON body stub based on the request schema. Operations with security requirements get an `Authorization: Bearer {{token}}` header hint.

After import, the workspace panel refreshes and you can open any generated request by clicking it in the tree.

### Importing a bundle (environments + scenarios)

A **bundle** is one document that holds environments, reusable requests and several scenarios — typically the reply from an AI assistant (see [section 14](#14-creating-files-with-ai-assistance)). To import one:

1. Open or create a workspace.
2. Choose **File → Import → Bundle (environments + scenarios)…**.
3. Paste the bundle into the box, or click **Open file…**. A surrounding code fence copied from a chat is fine.
4. Review the preview. It lists every file that will be created, plus any **problems** (these block the import) and **warnings** (these do not). Problems say exactly where they are, for example `scenarios[0] "Flow" › step 2 "Login"`.
5. Optionally enter a **Spec file** (a path or URL). It replaces the spec in every imported scenario — use it to point the schema checks at a local copy of the OpenAPI document.
6. If some files already exist, choose **Rename** (keep both, saved as `name-2.yaml`), **Skip** or **Overwrite**.
7. Click **Import**.

The result is ordinary request, scenario and environment files under `requests/`, `scenarios/` and `environments/`. Nothing is written until you click Import, and a bundle with any problem is not imported at all. Select the new environment afterwards. The bundle format is described in the Format Spec under **Bundle File**.

---

## 8. Scenarios and contract testing

A **scenario** is a YAML or JSON file that runs a sequence of HTTP requests in order, extracts values from responses, passes them to later steps via `{{variable}}` interpolation, and asserts that each response matches the declared contract.

Scenarios are the primary tool for **Spec Conformance Testing**: proving that your API honours its OpenAPI specification end-to-end, in a repeatable, automatable way.

### Scenario file format

```yaml
schemaVersion: "1"
title: User lifecycle
description: "Verifies CRUD operations for /users. Run after any changes to the users endpoint or auth middleware."

steps:
  - label: Login
    notes: "Obtain a bearer token. failPolicy stop — all subsequent steps depend on this."
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
        operator: eq
        expected: 200
    failPolicy: stop   # stop | continue | skip

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
        content: '{"name":"Alice","email":"alice@example.com"}'
    extractions:
      - name: userId
        source: jsonpath
        path: "$.id"
    assertions:
      - type: status
        operator: eq
        expected: 201
      - type: jsonpath
        path: "$.name"
        operator: eq
        expected: Alice

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
      - type: schema
        schema: getUserById
    failPolicy: continue
```

### Scenario-level fields

| Field | Description |
|---|---|
| `title` | Display name shown in the UI and reports. |
| `description` | Optional prose describing the overall scenario purpose. |
| `specFile` | Path to an OpenAPI spec (relative to the workspace). When set, Clear Signal maps each executed request to the spec's operations and accumulates coverage data visible in the **Contracts** tab. |

### Step fields

| Field | Description |
|---|---|
| `label` | Short action phrase for the step — e.g. "Login", "Create order". Shown in the runner UI and reports. |
| `notes` | Optional prose explaining the step's purpose and dependencies — e.g. "Requires the token from the Login step". |
| `request` | Inline request definition. Alternatively, use `requestRef: path/to/file.yaml` to reference a saved request file. Use one or the other, never both. |
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

### Running a scenario

Open a scenario tab and click **Run** in the scenario header bar. While the scenario is running the button changes to **Stop** (amber square icon) — click it at any time to cancel the in-flight run. Steps that have already completed are preserved in the results view; the remaining steps are marked cancelled.

### Running all scenarios (Run All)

Click **Run All** in the Scenarios sidebar or in the scenario header bar. All scenarios in the workspace execute in sequence. A dedicated **Runner** tab opens (or becomes active) and shows:

- A progress bar across the top while running.
- A table with one row per scenario — pass/fail icon, scenario name, and total duration.
- A live counter in the header ("Running 2 / 5…").
- When finished: a summary line ("All passed" or "N failed") with totals.
- **Stop** button (amber) while running — cancels the current and remaining scenarios.
- **Run Again** and **Clear** buttons after the run completes.

Click any row in the runner table to open that scenario's full result in an editor tab.

---

## 9. Contracts — coverage tracking

The **Contracts** tab tracks which operations in an OpenAPI specification have been exercised by your scenarios, what the last test result was for each one, and surfaces quality signals that highlight what you have *not yet proven*.

### Enabling coverage tracking

Add a `specFile` field to any scenario pointing at an OpenAPI spec file (relative to the workspace root):

```yaml
schemaVersion: "1"
title: User lifecycle
specFile: requests/My API v1/openapi.yaml

steps:
  - label: Login
    ...
```

After each run, Clear Signal matches every request the scenario made against the spec's path patterns and records whether each matched operation passed or failed.

### The Contracts sidebar

Click the **Contracts** icon (bar chart) in the Activity Rail to open the sidebar. Each spec that has been exercised appears as a card showing:

- Overall coverage percentage and progress bar
- Number of operations covered out of total
- Total runs, passed runs, and failed runs

Click a spec card to view its full detail in the main area.

### The Contracts tab (main area)

The detail view shows a table with one row per operation:

| Column | Meaning |
|---|---|
| **Method** | HTTP method (GET, POST, PUT, …) |
| **Path** | URL path pattern from the spec |
| **Coverage** | ✓ Covered (called at least once) or — (never called) |
| **Last Result** | ✓ Pass or ✗ Fail from the most recent run that hit this operation |
| **Last Run** | Relative time of the most recent run |

Operations are grouped into **Covered** and **Not covered** collapsible sections, sorted by path then method.

### Quality signals

Below the operations table, Clear Signal shows four collapsible quality-signal sections that highlight gaps in what you have proven:

| Signal | Meaning |
|---|---|
| **Happy-path only** | Only 2xx responses have been observed — no error paths have been tested |
| **Error responses never triggered** | The spec declares non-2xx status codes for this operation, but none have ever been returned |
| **Secured — never tested with auth** | The operation requires authentication (per the spec's security schemes) but has always been called without credentials |
| **Never exercised** | These operations have not been reached by any recorded run |

A compact summary of the same gaps also appears inline in the scenario result view after any run that has a `specFile`.

### Reset and export

- **Reset data** — clears all accumulated coverage for the selected spec.
- **Export** (dropdown button) — three formats:
  - **Export CSV** — comma-separated file with one row per operation: method, path, coverage, last result, last run time, quality flags, and missing status codes.
  - **Export Markdown** — Markdown report suitable for pasting into a wiki, PR description, or README. Includes the coverage summary, operations table, and all quality gap sections.
  - **Export HTML** — full HTML report opened in your default browser, ready for print-to-PDF. Includes the same content as the Markdown report with styling.

### Auto-refresh

The tab refreshes automatically whenever you switch to it and after any scenario run completes (individual or run-all).

---

## 10. CLI usage

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

## 11. CI/CD integration

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

## 12. Settings

Open via the gear icon at the bottom of the Activity Rail (the icon bar on the far left of the window).

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

## 13. File format reference

All project files are human-readable YAML (or JSON equivalents). Every file carries a `schemaVersion` field — the current version is `"1"`.

### Request file

```yaml
schemaVersion: "1"
title: Get user by ID

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

timeout: 10s   # omit to use the global default
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

## 14. Creating files with AI assistance

The Help panel (the `?` icon at the bottom of the Activity Rail, or the **Help** menu) has four tabs:

| Tab | What it is |
|---|---|
| **User Guide** | This guide. |
| **Format Spec** | The complete file format specification, including the bundle format. **Copy Format Spec** and **Download** put it on your clipboard or save it as a Markdown file. |
| **Example Scenario** | A complete, runnable scenario that uses every feature of the format, with its environment and a reusable request. It runs against the built-in Demo Server. **Copy scenario** and **Download** give you just the scenario YAML. |
| **AI Prompt** | Instructions for an AI assistant that turns an OpenAPI or Swagger document into a ready-to-import bundle. **Copy AI Prompt** and **Download** give you the complete prompt (instructions + format spec + example scenario). |

### Generating tests from an OpenAPI document

1. Open the **AI Prompt** tab.
2. Click **Copy AI Prompt** and paste it into a new conversation in ChatGPT, Claude, Gemini or any other assistant. Or click **Download** and attach the saved Markdown file to the chat instead.
3. Give the assistant your API: paste the OpenAPI/Swagger document, or give it a URL to the document. (If the assistant cannot open web pages it will ask you to paste the document.)
4. Answer its few questions — which environments you want, how authentication works, whether tests may create and delete data, and what to focus on. It then proposes a list of scenarios; reply `go` to generate them.
5. The assistant replies with **one code block**: a bundle containing an environment, optional reusable requests and several scenarios, followed by notes on what to fill in.
6. Copy the code block and import it with **File → Import → Bundle (environments + scenarios)…** ([section 7](#importing-a-bundle-environments-scenarios)).
7. Select the new environment and fill in its placeholder values (base URL, credentials, and so on), then run the scenarios.

### What the assistant is told to do

- Read the spec and summarise it, then ask at most five questions with sensible defaults.
- Write scenarios only from what the document defines — it must not invent endpoints, fields or operation IDs.
- Derive assertions from the spec: declared status codes, required response fields, and `schema` checks by `operationId`.
- Chain steps with extracted values (for example a login token or a created ID), and clean up any data a scenario creates.
- Keep real secrets out of the file: credentials are placeholders marked `secret: true`.
- Check its own output against a checklist before sending it.

### Tips

- If the OpenAPI document is OpenAPI 3.x and you gave a URL, the bundle binds scenarios to it, and every response is also validated against the spec. If you pasted the document instead, save it to a file and enter its path in the **Spec file** box of the import dialog.
- Swagger 2.0 documents work for generating requests and assertions, but cannot be bound for schema validation.
- Import problems are reported with their location. Paste them back to the assistant and ask it to fix exactly those.
- Run the generated scenarios against a test environment first; they create and delete data.
- For several environments (dev, staging, production), tell the assistant up front. It generates one environment per target with identical variable names so the same scenarios run against each.

---

## 15. Keyboard shortcuts

| Shortcut | Action |
|---|---|
| **Enter** (URL field focused) | Send the current request |
| **Ctrl+Enter** | Send the current request (works from the URL bar and most input fields; blocked only inside multiline text areas so you can type newlines in the body) |
| **Ctrl+N** | New request tab |
| **Ctrl+W** | Close the current tab |
| **Ctrl+S** | Save the current request or scenario |
| **Ctrl+Shift+S** | Save all dirty tabs |
| **Ctrl+Tab** | Switch to the next tab |
| **Ctrl+Shift+Tab** | Switch to the previous tab |
| **Ctrl+1** – **Ctrl+9** | Switch directly to tab N |
| **← / →** (tab bar focused) | Navigate between tabs with the arrow keys |
| **Delete / Backspace** (tab bar focused) | Close the focused tab |
| **Ctrl+B** | Toggle sidebar open/closed |
| **Ctrl+P** | Quick open — fuzzy-search all requests and scenarios |
| **Ctrl+F** | Search in the response body (when response panel is focused) |

---

## 16. Troubleshooting

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

### `Dev server request rejected: Host "wails.localhost" is not an allowed host`

The Wails devserver (port 34115) allows only `localhost`, IP literals, and its own bind host; the WebView requests `wails.localhost`. Set `WAILS_DEV_ALLOWED_HOSTS=wails.localhost` in the shell before `wails dev` (or persist it with `setx WAILS_DEV_ALLOWED_HOSTS wails.localhost` on Windows, then open a new terminal).

### Go tests fail in CI but pass locally

The executor tests use `httptest.NewServer` and require a free port. No external network access is required; all tests run against `localhost`. If the CI runner restricts networking, the tests will fail — ensure the runner allows loopback connections.

### A `{{variable}}` appears literally in the sent request

The variable is not defined in the active environment. Check the environment editor (pencil icon in the status bar) and confirm the variable name matches exactly, including case.

### The workspace sidebar is empty after opening a workspace

The workspace root must contain a `requests/` directory. If you opened a folder that has request files at the top level (not in `requests/`), create a `requests/` subdirectory and move the files there.
