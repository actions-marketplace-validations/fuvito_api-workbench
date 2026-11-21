# Clear Signal User Guide

**Version:** 0.6.0

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

**To open an existing workspace:** use **File → Open Workspace** in the menu bar, or click the folder icon at the top of the Requests panel in the sidebar.

**To create a new workspace:** use **File → New Workspace**, then pick an empty folder. Clear Signal creates a starter structure with a sample environment and a sample request so you can see the layout immediately.

When a workspace is opened, the first request file found in the tree is loaded automatically into the editor — no need to click a file manually to get started.

**Switching workspaces:** opening or switching to a workspace (via the File menu, the Open Workspace button, or a recent workspace entry) closes the current workspace and all its tabs first. If any tabs have unsaved changes, a confirmation dialog appears before the switch.

The app always starts on the landing page; recent workspaces are listed there. The environment you choose is remembered **per workspace** and restored when you open that workspace again (the temporary Demo and Conformance workspaces are never remembered).

**Recent workspaces:** The landing page (shown when no workspace is open) lists your recently used workspaces. If you have opened a workspace before, a **Resume** button appears at the top — it reopens the most recent workspace in one click. The File menu → Recent Workspaces submenu lists up to ten recent paths from anywhere in the app.

### The Workspace page

Opening a workspace (File → Open or New Workspace, a recent entry, or Help → Dev Servers) opens the **Workspace** page. It is the first item in the Activity Rail, so you can come back to it at any time; the tabs you already had open stay open behind it.

The page shows, at a glance:

- **A reminder line** at the top: which environment is active and where that choice came from (*remembered for this workspace*, *chosen this session*, *temporary workspace, not remembered* or *none chosen*), and how many things need attention. Dismiss it with the cross.
- **Needs attention:** things that would make a run fail or mislead. Each says what is wrong, where, why it matters and how to fix it, with a button that goes there. The items are: no environment selected although scenarios use placeholders only an environment can supply; a placeholder that a scenario uses but the selected environment does not define (and nothing in the scenario defines either); an environment value that refers to a process variable (`${NAME}`) that is not set; a scenario that cannot be parsed or fails validation; a `requestRef` whose request file is missing; a `specFile` that does not exist (or lies outside the workspace, shown as information); and a spec whose last recorded run had failing operations. The page informs; it never blocks a run.
- **Specs** that scenarios name, whether each file is found, missing, outside the workspace or a URL (URLs are never fetched here), and the Contracts totals recorded for it.
- **Last results:** the most recent scenario run in this workspace and how many scenarios failed the last time they ran. This comes from the run log kept on disk, so it survives a restart.
- **Contents and Environment:** counts of requests, scenarios, environments and specs, and the environment's variables as *set* or *unset*. A secret value is never shown.

Only what is sent counts as a use of a placeholder (URL, headers, query, body, auth and assertions, including the request files a step points at); notes, labels and descriptions do not. The page reads the workspace when it opens, whenever you come back to its tab, after a scenario run, when you change the environment, and when you press **Refresh**. It only reads; it never changes a file.

When you close or switch workspaces and have unsaved tabs, the confirmation also tells you what will be remembered (for example *The environment "Staging" will be remembered for this workspace.*).

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

The application substitutes `${VAR}` references from the process environment **when a run starts**, never when an environment file is read or saved, so the real value is never written back to disk. Write `$${VAR}` for the literal text `${VAR}`. If a referenced variable is not set on the machine, the text is left as written and `clrsig run`, `conform`, `request` and `bundle run` print one warning naming it (it is a warning, not an error, so an unused variable cannot fail a run).

Secret values are also kept out of **reports**: wherever one would appear (an assertion's actual value, an error message), it is replaced by `***`, in every format. See [Secrets in reports](#secrets-in-reports).

---

## 6. Sending requests and reading responses

1. Set the HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) using the dropdown.
2. Enter the URL. Pasting a URL containing a query string (e.g. `https://api.example.com/items?page=1&limit=20`) automatically extracts the parameters into the **Query Params** tab.
3. Fill in the **Query Params**, **Headers**, **Body**, and **Auth** tabs as needed.
4. Press **Send** (or **Enter** in the URL bar). While the request is in progress, the button changes to **Cancel** — click it to abort.

### Request tabs

| Tab | Purpose |
|---|---|
| **Query Params** | Query parameters appended to the URL. Toggle rows on/off with the checkbox. |
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

Open the **Requests** panel and switch to its **History** view (next to the request list, the same way **Run Log** sits next to the scenario list) to see your last 100 requests, grouped by date — **Today**, **Yesterday**, and older dates. Each entry shows the method badge, status code, duration, and URL. 

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

Path parameters become `{{paramName}}` placeholders. Operations with security requirements get an `Authorization: Bearer {{token}}` header hint.

POST/PUT/PATCH operations get a JSON body template built from the request schema (the same template `clrsig conform` sends):

- Only the **required** properties are included, plus optional ones that have an `example` or a `default`. Properties marked `readOnly` are left out (the server assigns them). A schema that marks nothing as required gives `{}`.
- A property with an `example`, a `default` or an `enum` uses that value as written (the first enum value).
- Any other property becomes a `{{placeholder}}` named after it, **typed**: `"{{title}}"` for a string, and a bare `{{price}}` for a number, boolean, array or object, so that setting `price` to `9.99`, `inStock` to `true` or `tags` to `["a","b"]` in the environment produces valid JSON. Because a bare placeholder is not valid JSON until it is filled in, the **Format** button understands them.
- `allOf` schemas are merged (properties and required lists); for `oneOf` and `anyOf` the first branch is used; an array body becomes an array of the item template; nested objects and arrays stay one placeholder holding a JSON value.

If a placeholder has no value when a request runs, it is not sent (see "Operations that cannot be sent" under `conform`).

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

### Exporting a bundle

**File → Export → Bundle (environments + requests + scenarios)…** saves everything in the open workspace as one bundle file — handy for sharing a test suite, moving it to another workspace, or giving an AI assistant your existing tests as context.

- Choose where to save it. A confirmation lists how many environments, requests and scenarios were written.
- **Secret values are replaced with `CHANGE_ME`**, so the file is safe to share. The workspace is not changed, and the message tells you how many values were replaced.
- Files that cannot be read (for example a request with a YAML error) are skipped and listed in the message.
- Import the file in another workspace with **File → Import → Bundle…**; scenarios keep working with the requests they refer to.

Unlike **Export Workspace** (a ZIP of the raw files, secrets included), a bundle is a single readable text file.

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
| `variables` | Optional list of scenario variables (`name`, `value`, optional `secret`). Each is generated once per run and can use dynamic values such as `{{$randomString 10}}`. See "Dynamic values" below. |
| `specFile` | Path to an OpenAPI spec (relative to the workspace). When set, Clear Signal maps each executed request to the spec's operations and accumulates coverage data visible in the **Contracts** tab. |

### Dynamic values and scenario variables

Tests do not have to use fixed data. `{{$guid}}`, `{{$randomString 10}}`, `{{$randomInt 1 100}}`, `{{$randomBool}}` and date functions such as `{{$date +7d}}` generate a fresh value when the request is sent. They work in URLs, headers, bodies, environment values and assertion values.

To use one generated value in several places, put it in the scenario's `variables`:

```yaml
variables:
  - name: email
    value: "user-{{$randomString 10}}@example.com"
steps:
  - label: Create user
    request: {method: POST, url: "{{baseUrl}}/users", body: {type: json, content: '{"email": "{{email}}"}'}}
  - label: Fetch user
    request: {method: GET, url: "{{baseUrl}}/users?email={{email}}"}
    assertions:
      - {type: jsonpath, path: "$[0].email", operator: eq, expected: "{{email}}"}
```

`{{email}}` is the same in both steps; two inline `{{$guid}}` calls are different. A function used in an environment variable is generated once per `clrsig` command. Random values come from a seed that the report shows; `clrsig run --seed 12345` (or `CLRSIG_SEED`) replays a run exactly. `{{$calc price * quantity}}` does arithmetic, and `{{$randomEmail}}`, `{{$randomFullName}}` and similar give realistic fake data. In the desktop app, the **Variables** section of the scenario's **Setup** row (the first item in the step list) edits these (the lock marks a value as secret, so reports mask it). Typing `{{` in a URL, header, body or variable value lists your variables and every dynamic function with a one-line description; picking a function inserts a working example to adjust. After a run, Setup's **Result** tab shows the generated values and, when random values were used, the seed that replays them with `clrsig run --seed`; a scenario that cannot start (for example an unknown function) shows the reason under the header instead of running. To experiment, start the Demo Server: its workspace has a `4-dynamic-values` scenario folder (unique users, a playground that echoes every function, calculations, dates and offsets) and a `8-dynamic-values` request, all safe to edit and run repeatedly. The complete function list is in the Format Spec tab.

### Authenticating with OAuth2 (client credentials)

There is no separate OAuth2 auth type: for service-to-service APIs a scenario does the same thing in two steps. The first step asks the token endpoint for a token and extracts it; later steps use it as a bearer token.

```yaml
steps:
  - label: Get token
    request:
      method: POST
      url: "{{authUrl}}/oauth/token"
      body:
        type: form-urlencoded
        content: "grant_type=client_credentials&client_id={{clientId}}&client_secret={{clientSecret}}"
    assertions:
      - {type: status, operator: eq, expected: 200}
    extractions:
      - {name: accessToken, source: jsonpath, path: $.access_token}
  - label: Call the API
    request:
      method: GET
      url: "{{baseUrl}}/me"
      auth:
        type: bearer
        token: "{{accessToken}}"
    assertions:
      - {type: status, operator: eq, expected: 200}
```

Keep the client secret out of the file: in the environment, give `clientSecret` the value `${CLIENT_SECRET}` with `secret: true`, and set `CLIENT_SECRET` in your CI secrets. Reports mask secret environment values, and an extracted value whose name looks secret (`accessToken`) is masked too. The token is fetched once per scenario run; put the token step first in each scenario that needs it. Values that contain `&`, `=` or other special characters must be URL-encoded in a form body. Flows that need a browser (authorization code, PKCE) are not supported: paste a token into an environment variable instead.

### Step fields

| Field | Description |
|---|---|
| `label` | Short action phrase for the step — e.g. "Login", "Create order". Shown in the runner UI and reports. |
| `notes` | Optional prose explaining the step's purpose and dependencies — e.g. "Requires the token from the Login step". |
| `request` | Inline request definition. Alternatively, use `requestRef: path/to/file.yaml` to reference a saved request file. Use one or the other, never both. |
| `extractions` | Pull values from the response into runtime variables for subsequent steps. |
| `assertions` | Checks that must pass for the step to succeed. |
| `failPolicy` | What to do if an assertion fails: `stop` (default — skips remaining steps), `continue` (keeps going), `skip` (skips this step if a previous one failed). |
| `disabled` | `true` skips the step without counting it as a failure. |
| `validateSpec` | With `specFile` set, validate this step's response against the spec (default `true`). `false` opts the step out. |
| `timeoutMs` | Timeout for this step in milliseconds; overrides the request's own `timeout` and the global default. |
| `retry.maxAttempts`, `retry.delayMs` | Re-send the step after a network error or a `5xx` response, up to `maxAttempts` attempts in total (1–10) with `delayMs` between them (at most 60000). A failed assertion or a `4xx` is never retried. Reports show the attempt count. |
| request `followRedirects: false` | Return a `3xx` response as it is instead of following it, so a redirect declared in the spec can be checked. In the app: **No redirects** next to the timeout, or **Don't follow redirects** in a step's advanced options. |

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

After a run, select a step to see exactly what happened: a blue **Request** section shows the request as it was sent (the URL with its query parameters, the headers and the body, with variables and dynamic values filled in) and a green **Response** section shows the status, time, headers and body that came back. Credentials are masked in the request: `Authorization`, cookies, API keys, tokens and any value of a secret variable appear as `***`. The same two sections appear for each step in the run log.

### Running all scenarios (Run All)

Click **Run All** in the Scenarios sidebar or in the scenario header bar. All scenarios in the workspace execute in sequence. A dedicated **Runner** tab opens (or becomes active) and shows:

- A progress bar across the top while running.
- A table with one row per scenario — pass/fail icon, scenario name, and total duration.
- A live counter in the header ("Running 2 / 5…").
- When finished: a summary line ("All passed" or "N failed") with totals.
- **Stop** button (amber) while running — cancels the current and remaining scenarios.
- **Run Again** and **Clear** buttons after the run completes.

Click any row in the runner table to open that scenario's full result in an editor tab.

When a run has finished, **Export report** saves its results as an HTML page (opened in your browser), JUnit XML, JSON, Markdown or plain text, built by the same code as the CLI's reports (so secrets are left out the same way). The export covers the scenarios listed in the table, using their latest results. To run only part of the workspace, use **Run folder** from a scenario folder's context menu; its results appear in the same tab.

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

At the top, four equal tiles give the totals, each as a count and its share of all operations in the spec:

- **Covered**: operations that at least one scenario calls.
- **Not covered**: operations no scenario calls (the same list as the *Not covered* section of the table).
- **Passing**: covered operations whose most recent result passed.
- **Failing**: covered operations whose most recent result failed.

They come in two pairs that add up: Covered plus Not covered make up the whole spec, and Passing plus Failing make up Covered. Being covered only says an operation was called, so a spec at 100% coverage can still show failures. Below the tiles, the **Scenario runs** line counts whole scenario runs (runs, passed, failed, last run), which is a different thing from operation results.

The detail view then shows a table with one row per operation:

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
  - **Export CSV** — comma-separated file with one row per operation: method, path, coverage, last result, last run time, quality flags, missing status codes, and a **Status** column (*Passing*, *Failing*, *Covered (no result)* or *Not covered*) so you can count the four totals in a spreadsheet.
  - **Export Markdown** — Markdown report suitable for pasting into a wiki, PR description, or README. Starts with the same four totals as the page (Covered, Not covered, Passing, Failing, each with its share of all operations), then the scenario run counts, the operations table, and all quality gap sections.
  - **Export HTML** — full HTML report opened in your default browser, ready for print-to-PDF. Includes the same content as the Markdown report with styling.

The command line produces the same reports, and its `coverage list` and `coverage quality` output (text and JSON) carries the same four totals: in JSON, `covered`, `notCovered`, `passing`, `failing` and `noResult` (covered operations whose last result was not recorded), with `total`. The per-spec CSV has the extra columns `Not Covered`, `Passing` and `Failing` after the original ones.

### Auto-refresh

The tab refreshes automatically whenever you switch to it and after any scenario run completes (individual or run-all).

---

## 10. CLI usage

The CLI uses the same Go execution engine as the desktop application. Build it once and use it in any terminal or CI pipeline — no behavioural differences between interactive testing and CI.

### The CLI Command Builder

You do not have to assemble a command by hand. The **CLI Command Builder** is a screen in the desktop app that builds the command for you, checks it, and explains every part.

**Open it** from the terminal icon (**CLI Command Builder**) near the bottom of the Activity Rail, or from the terminal button in a scenario's header bar, which opens it with that scenario already chosen.

1. **Pick what you want to do** — run scenarios (one file or a whole folder), check an API against its OpenAPI spec, or detect breaking changes between two specs.
2. **Fill in the fields.** Each one says in plain words what it is and what is expected. Choose files from the **Workspace…** list (it shows the scenarios, environments and OpenAPI documents found in your workspace), browse for any other file, or type a path or web address. Less common options are under **More options**.
3. **Read the result** on the right: the command, a list of anything to fix, and an explanation.
4. **Copy** it into your terminal or CI pipeline. **Copy** stays disabled while there is a problem to fix.

What the builder does for you:

- **Relative paths, written for where you run the command.** Paths in the command are relative to the **Run from folder** (by default your repository root, found by looking for `.git`). That is why a command copied from here works the same on your machine and on a CI runner that checks out the repository. A path outside that folder is written as an absolute path, with a warning that it will not exist on another machine.
- **The right shell.** Choose **bash**, **PowerShell** or **cmd**; the quoting and line continuation are adjusted for each. **One line** switches off the line breaks.
- **Checks with the real parsers.** The scenario, environment and spec you pick are opened the same way the CLI opens them, so you see "valid, 4 steps", the OpenAPI title and number of operations, or the exact problem, before anything runs.
- **Files or whole folders.** For **Scenario or folder**, pick one scenario, or a folder (the list starts with the workspace's scenario folders and how many scenarios each holds). The builder shows what the CLI will find — "3 scenarios found in scenarios and its sub-folders, all valid; 1 other file skipped" — and names any broken scenario, because a broken one would stop the run. **Stop at the first failing scenario** and **Do not search sub-folders** are under More options.
- **Missing pieces.** It warns when a scenario uses a `{{variable}}` that the environment does not define, and when `--coverage` or `--workspace` has no spec to work with (a `specFile:` inside the scenario counts).
- **Variables to set in CI.** If your environment file reads values such as `${API_TOKEN}`, the builder lists them under **Set these where the command runs** and marks the secret ones. Secret values are never shown or stored.
- **Report format and file name.** Choose `text`, `json`, `junit` or `html`. For a non-text format, the builder suggests a file such as `reports/run-user-lifecycle.html`; click **Use …** to take it. It tells you when a file will be overwritten or a folder created. See [Output formats](#output-formats).
- **An explanation of every part.** *Part by part* says what each piece of the command means, *What you get* says what is printed or written, and *Exit codes* says what each exit code means and what a pipeline does with it.

The builder only builds and explains; it does not run anything. In a pipeline, give each `clrsig` command its own step with its own report file, so every result keeps its own log and exit code, and upload the reports together.

### Build

Or run directly without building:

```bash
go run ./apps/cli run <scenario-file>
```

### `clrsig run`

Runs scenario files and prints the results. Give it one file, several, or a folder.

```bash
clrsig run ./scenarios/user-lifecycle.yaml
clrsig run ./scenarios/user-lifecycle.yaml --env ./environments/dev.yaml
clrsig run ./scenarios/user-lifecycle.yaml --format json
clrsig run ./scenarios/user-lifecycle.yaml --seed 12345   # replay the same random values
clrsig run ./scenarios/user-lifecycle.yaml --spec openapi.yaml --coverage --workspace .

# every scenario in a folder and its sub-folders
clrsig run ./scenarios --env ./environments/ci.yaml
```

#### Running many scenarios

- **A folder runs everything inside it**, including sub-folders. Add `--no-recursive` to look only at the folder itself.
- **Several arguments** work too (`clrsig run a.yaml b.yaml ./more`); a scenario named twice runs once. Scenarios run one after another in path order.
- **Other files are skipped, not run.** Environment, request and spec files in the same folder are not scenarios; they are listed in a note, never counted as failures. A file you name directly is always treated as a scenario.
- **Nothing runs if anything is broken.** All scenarios are read first. If one cannot be read or parsed, the command stops with exit code `2` and names the file, before a single request is sent. A folder with no scenarios is also an error.
- **One combined report and one exit code.** The text report shows each scenario and then `Run PASSED 2/3 scenarios passed …`. `junit` has one test suite per scenario, `html` one section per scenario, and `json` a `scenarios` list. With `--coverage`, coverage is combined per spec, and `--workspace` records every scenario's run.
- **`--fail-fast`** stops after the first scenario that fails; the rest are listed as not run.
- The options `--env`, `--spec`, `--coverage`, `--workspace` and `--timeout` apply to every scenario. A scenario's own `specFile:` is still relative to that scenario file.
- **One file** behaves exactly as before: the same report and the same exit codes.

| Flag | Default | Description |
|---|---|---|
| `--env`, `-e` | (none) | Path to an environment file (YAML or JSON). |
| `--format`, `-f` | `text` | Output format: `text`, `json`, `junit` or `html` — see [Output formats](#output-formats). |
| `--output`, `-o` | (stdout) | Write the report to a file; parent directories are created. |
| `--spec`, `-s` | (none) | OpenAPI spec file or URL. Overrides the scenario's `specFile`. A relative path is relative to your current directory (a `specFile:` inside the scenario is relative to the scenario file). |
| `--coverage` | off | Report which spec operations the scenario exercised (needs a spec). |
| `--workspace`, `-w` | (none) | Accumulate coverage into this workspace, so it appears on the Contracts page. |
| `--timeout` | `0` | Global step timeout in milliseconds; overrides per-step values. `0` uses the per-step value. |
| `--fail-fast` | off | With several scenarios, stop after the first one that fails. |
| `--no-recursive` | off | With a folder, only run the scenarios directly inside it, not in its sub-folders. |
| `--var` | (none) | `name=value`, repeatable: set or override one environment variable for this run only. Put secrets in environment variables, not here (command lines are visible to other processes). |
| `--report` | (none) | `format=file`, repeatable: also write a report in another format from the same run. See [Several reports from one run](#several-reports-from-one-run). |
| `--show-extracted` | off | Show extracted values whose names look secret. See [Secrets in reports](#secrets-in-reports). |

### Client certificates (mutual TLS)

For APIs that require a client certificate, every command that sends requests (`run`, `request`, `conform`, `bundle run`, `mcp`) accepts:

```bash
clrsig run ./scenarios --client-cert client.pem --client-key client.key --ca-cert private-ca.pem
```

or the environment variables `CLRSIG_CLIENT_CERT`, `CLRSIG_CLIENT_KEY` and `CLRSIG_CA_CERT` (a flag wins). Give paths, not contents: in CI, write the certificate from your secret store to a temporary file first. A wrong or half-configured path stops the command before any request is sent (exit 2). The key must be unencrypted PEM.

### Exit codes

| Code | Meaning |
|---|---|
| `0` | All steps passed all assertions (in every scenario, when running several). |
| `1` | One or more assertions failed (in any scenario). |
| `2` | A network, parse, or configuration error occurred. When running several scenarios this includes a scenario that cannot be read (nothing runs) and a folder with no scenarios. |

### Output formats

Report commands choose their format with `--format` (`-f`) and print to the screen unless you give `--output` (`-o`) a file, in which case the folders are created for you. Nothing is written to disk by default, and the exit code never depends on the format.

| Format | Best for | Commands |
|---|---|---|
| `text` (default) | Reading in a terminal or CI log | all report commands |
| `json` | Scripts and other tools | `run`, `conform`, `diff`, `impact`, `validate`, `generate`, `request`, `coverage list`, `coverage quality`, `bundle run` |
| `junit` | CI test dashboards and pull-request annotations (Jenkins, GitHub, GitLab, CircleCI) | `run`, `conform`, `diff`, `bundle run` |
| `html` | A report a person can open in a browser, attach to a ticket or keep as a CI artifact | `run`, `conform`, `diff`, `bundle run` |
| `markdown` | Pull-request comments and the GitHub job summary | `run`, `conform`, `diff`, `bundle run` |
| `csv` | Spreadsheets (the Contracts coverage data) | `coverage list`, `coverage quality` (which also take `markdown` and `html`) |

```bash
clrsig run scenarios/user-lifecycle.yaml --env environments/ci.yaml -f junit -o reports/run.xml
clrsig run scenarios/user-lifecycle.yaml --env environments/ci.yaml -f html  -o reports/run.html
clrsig run scenarios --env environments/ci.yaml -f junit -o reports/all.xml   # a whole folder, one report
```

The HTML report is one self-contained file (no external scripts, fonts or images), follows your light or dark system theme, and prints cleanly. It shows the same results as the text report — pass/fail per step, assertions, violations, coverage — but leaves out **extracted variable values**, so a token pulled from a login response does not end up in a file you share. 
The Markdown report uses the same content, with ✅ and ❌ marks and escaped text, so it can be pasted into a pull-request comment or appended to the GitHub job summary:

```bash
clrsig run scenarios --env environments/ci.yaml -f markdown >> "$GITHUB_STEP_SUMMARY"
```

An unknown `--format` is an error and stops the command before anything runs.

#### Several reports from one run

`--report format=file` (repeatable) writes extra reports from the **same run**, so you do not send every request twice to get both a JUnit file and an HTML page:

```bash
clrsig run scenarios --env environments/ci.yaml \
  -f markdown -o reports/summary.md \
  --report junit=reports/all.xml --report html=reports/all.html
```

`--format` and `--output` still decide what goes to the screen or to one file; each `--report` adds a file (folders are created). A malformed value, an unknown format, or the same file used twice is reported before the run starts. Available on `run`, `conform`, `diff` and `bundle run`.

#### Secrets in reports

Reports are meant to be shared, so they never contain credentials:

- the value of any environment variable marked `secret: true` is replaced by `***` wherever it would appear (an assertion's actual value, an error message), when it is at least 4 characters long;
- an extracted value whose variable name looks secret (`token`, `password`, `apiKey`, `session`, `cookie`, `authorization`, …) is shown as `***`; other extracted values, such as an `id`, stay visible;
- `--show-extracted` (on `run` and `bundle run`) shows the secret-looking extracted values again, for debugging on your own machine. It never reveals the values of secret environment variables.

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

### `clrsig bundle`

A **bundle** (see [section 7](#importing-a-bundle-environments-scenarios)) can be run, checked, imported and exported from the command line, with the same behaviour as the desktop app.

```bash
# Run every scenario in a bundle, straight from the file — no workspace needed
clrsig bundle run bundle.yaml --env staging

# Check a bundle without running or importing anything
clrsig bundle validate bundle.yaml

# Save its files into a workspace
clrsig bundle import bundle.yaml --workspace ./my-workspace

# Write a workspace as one bundle (secrets become CHANGE_ME)
clrsig bundle export --workspace ./my-workspace -o bundle.yaml
```

**`bundle run`** validates the bundle first; any error stops the run before a single request is sent. It then runs every scenario in order, using the requests, spec and environment from the bundle.

| Option | Meaning |
|---|---|
| `--env <name>` | Which of the bundle's environments to use. Needed only when the bundle has more than one; a single environment is used automatically. |
| `--env-file <file>` | Use an environment file instead — handy in CI, where credentials come from the pipeline rather than the bundle. |
| `--spec <path-or-url>` | Replace the spec of every scenario. |
| `--scenario <title-or-file>` | Run only this scenario. Repeat the option for several. |
| `--format text\|json\|junit\|html` | Report format. `junit` produces one document with a test suite per scenario; `html` one page covering every scenario. |
| `--output <file>` | Write the report to a file. |

Exit codes: `0` every scenario passed, `1` at least one failed, `2` the bundle is invalid or could not be read.

**`bundle validate`** prints errors and warnings with their location and exits `1` if there are errors. **`bundle import`** accepts `--on-conflict rename|skip|overwrite` (default `rename`), `--spec-file` and `--dry-run` to preview the files without writing them.

`bundle export` also accepts `--keep-secrets` to write secret values as they are instead of `CHANGE_ME` — only use it for files that will not be shared.

`clrsig run` also loads the request files that a scenario's `requestRef` points at — looked up next to the scenario, in the `requests/` folder beside its `scenarios/` folder, and in `<workspace>/requests` when `--workspace` is given.

### `clrsig conform`

Runs **every operation declared in an OpenAPI spec** against a live server and validates each response against the spec — no scenario file needed.

```bash
clrsig conform openapi.yaml --base-url http://localhost:8080
clrsig conform openapi.yaml --base-url http://localhost:8080 --env environments/local.yaml --coverage
clrsig conform openapi.yaml --base-url http://localhost:8080 --negative --format junit -o conform.xml
```

| Flag | Description |
|---|---|
| `--base-url`, `-b` | Server to send requests to (overrides the spec's `servers[0]`). |
| `--env`, `-e` | Environment file for variable resolution (for example an auth token). |
| `--tags`, `--paths` | Only run operations with these comma-separated tags, or whose path starts with one of these prefixes. |
| `--negative` | Also send negative probes: dropped required fields, wrong types, missing auth, missing parameters. |
| `--explain` | Under each failing operation, say in plain language what the spec expected, what the server sent, why that fails and what to change. Off by default, so existing logs do not change; the JSON (`findings`), HTML, Markdown and JUnit reports always include it. |
| `--read-only` | Send only GET and HEAD requests. POST, PUT, PATCH and DELETE operations (and their negative probes) are skipped and never sent, so the check cannot create, change or delete data. Off by default: without it `conform` sends every operation. |
| `--coverage` | Append a coverage summary. |
| `--fail-fast` | Stop after the first violation or execution error. |
| `--request-timeout` | Per-request timeout in milliseconds (default `10000`). |
| `--format`, `-f` / `--output`, `-o` | `text`, `json`, `junit`, `html` or `markdown`; write the report to a file. |
| `--report` | `format=file`, repeatable: also write a report in another format from the same run. |
| `--var` | `name=value`, repeatable: set or override an environment variable for this run. |
| `--write-baseline` | Record today's violations in a file (YAML, or JSON for a `.json` name) and exit `0`. |
| `--baseline` | Accept the violations recorded in that file: only **new** ones fail the run, and fixed ones are reported. |

Exit codes: `0` all operations conformed, `1` violations or execution failures, `2` spec parse or configuration error.

**What a response is checked against.** For every operation the response must match what the spec declares for the status the server answered: the **status** itself (a status the spec does not declare, other than through a `default` response, is a violation), the **Content-Type** (a type the spec does not declare is a violation, and the body is then not compared), every **header** the spec declares (a required header that was not sent, or a header whose value does not fit its type) and the **body** against its schema. Each violation says what is wrong, where it is (the part of the response, such as the header name, the status or the body field, and the matching place in the spec), why it counts as a failure and how to fix it; the screen output shows that explanation with `--explain`, the JSON, HTML, Markdown and JUnit reports always include it. The first problem found is reported: a response with both a header problem and a body problem shows the header problem first, so fix it and run again to see the next. The complete list of what is checked, with an ID, an example and the known gaps for each, is in the **What We Check** tab of this Help panel (menu Help, What We Check). Versions before this release did not look at declared headers at all, and a response whose operation declared any header could pass with a broken body; if a previously green run now fails, record the current state with `--write-baseline` and let `--baseline` fail only on new violations while you fix them.

**Operations that cannot be sent.** `conform` builds each request from the spec and fills `{{placeholders}}` (path parameters such as `{{id}}`, `{{token}}` for secured operations, body fields such as `{{name}}`) from the environment and `--var`. If a request still has a placeholder nothing provides, it is **not sent**: sending `/items/{{id}}` would only test how the server handles that literal text, and a declared `404` would then count as a match. The operation is reported as failed with `not sent: no value for {{id}}`, the run exits `1`, and one line under the report lists every missing name once (add them to the environment, or pass `--var name=value`). A variable that is defined but never used is not an error and is not reported. A dynamic value that fails (an unknown `{{$function}}`) blocks the operation the same way.

#### Adopting `conform` on an API that already has violations

Pointing `conform` at a long-lived API often reports many violations at once. Record them, commit the file, and let CI fail only on **new** ones:

```bash
clrsig conform openapi.yaml -b http://localhost:8080 --write-baseline conform-baseline.yaml   # once
clrsig conform openapi.yaml -b http://localhost:8080 --baseline conform-baseline.yaml         # in CI
```

The baseline summary (`Baseline: 12 known, 1 new, 3 fixed`, with each new and fixed violation) is printed to stderr, so `--format json` on stdout stays machine-readable. When violations are fixed, run `--write-baseline` again to remove them, so quality can only improve; a regression of a fixed one then fails. A violation is identified by its operation (`GET /users/{id}`), its kind (`request`, `response`, `error`, `probe`) and the field and rule that broke, not by ids or timestamps, so the file does not change between runs. The JUnit/HTML/JSON reports still list every violation; only the exit code follows the baseline. Secrets never appear in the file.

### `clrsig diff` and `clrsig impact`

```bash
clrsig diff openapi-v1.yaml openapi-v2.yaml --breaking-only
clrsig diff openapi-v1.yaml openapi-v2.yaml --fail-on warning
clrsig diff openapi.yaml --against origin/main
clrsig impact openapi-v1.yaml openapi-v2.yaml scenarios
```

`diff` classifies every change between two specs as breaking (`ERR`), potentially breaking (`WARN`) or informational (`INFO`); `--format text|json|junit|html|markdown`, `--output` and `--report` are supported.

- **`--fail-on breaking|warning|info`** (default `breaking`) sets how serious a change must be to fail the command: `warning` also fails on potentially breaking changes, `info` on any change at all. The exit code, the JUnit failures and the HTML and Markdown verdict all follow it. It cannot be combined with `--breaking-only`, which hides the changes a stricter level would fail on.
- **`--against <git-ref>`** takes **one** spec file and compares it with its committed version at that ref (for example `origin/main`), read through git. It needs the `git` executable and a repository, and explains what is wrong when git is missing, the file is not in a repository, the ref is unknown (in CI, a shallow clone may need `fetch-depth: 0`), or the file does not exist at that ref.

`impact` goes one step further: it reports which of your **scenario steps** match a breaking change, so you know what will stop passing before the new spec ships. Scenario arguments can be files or folders. It exits `1` when any scenario is affected. All three commands exit `2` on a parse error.

### `clrsig generate`, `request` and `validate`

```bash
clrsig generate openapi.yaml --output requests/ [--base-url https://api.example.com] [--dry-run]
clrsig request requests/health-check.yaml --env environments/local.yaml [--timeout 5000]
clrsig validate scenarios
```

`generate` writes one request file per spec operation (the same files the desktop **Import OpenAPI** button creates). `request` executes one saved request file and prints the response. `validate` checks scenario files for errors without sending anything; like `run`, it accepts files and folders (sub-folders included unless `--no-recursive`), skipping files that are not scenarios. All three accept `--format text|json`.

### `clrsig import`

```bash
clrsig import postman collection.json --output requests/
clrsig import insomnia export.json --output requests/ --dry-run
```

Converts a Postman collection (v2.1) or an Insomnia export (v4) into request files, in the same layout the desktop app's **Import** menu uses: `<output>/<collection>/<folders>/<request>.yaml` (default output folder: `requests`). It never overwrites: a name that is taken gets `-2`, `-3` … `--dry-run` lists the files that would be written. `--format json` gives the list for scripts. Exit codes: `0` converted, `1` some requests could not be written, `2` the file is unreadable or not a supported export.

### `clrsig coverage`

Reads the same accumulated data the **Contracts** tab shows.

```bash
clrsig coverage list    --workspace .
clrsig coverage quality openapi.yaml --workspace .
clrsig coverage quality openapi.yaml --workspace . -f html -o reports/contracts.html
clrsig coverage reset   openapi.yaml --workspace .
```

`quality` prints the four quality signals from [section 9](#9-contracts--coverage-tracking) and exits `1` when any gap is found. Both `list` and `quality` also write `--format csv|markdown|html` (with `--output`); these are the same reports the desktop Contracts **Export** menu produces, because both use the same code.

### `clrsig mcp`

```bash
clrsig mcp
```

Starts a Model Context Protocol server on stdin/stdout, exposing the tools `conform`, `run_scenario`, `diff` and `coverage` so AI agents such as Claude Desktop or Cursor can run Clear Signal for you. It opens no network port.

### `clrsig telemetry`

```bash
clrsig telemetry status
clrsig telemetry enable
clrsig telemetry disable
```

Anonymous usage counters are **off by default**. They never include request contents or personal data. The desktop app has the same switch in Settings.

---

## 11. CI/CD integration

The CLI is the natural integration point for pipelines. Build it as part of your workflow, run your scenarios, and act on the exit code.

### The Clear Signal GitHub Action

The quickest route needs no Go toolchain — use the published Action:

```yaml
- uses: fuvito/api-workbench@v1
  with:
    command: conform
    spec: openapi.yaml
    base-url: http://localhost:8080
    format: junit
    output: test-results/conform.xml
```

### Building the CLI yourself

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
| **Client certificate file**, **Client key file** | Mutual TLS: the PEM certificate and private key sent to servers that require a client certificate. Only the file *paths* are stored (a `${NAME}` reference is expanded when a request is sent); the key must not be password-protected. Both are needed together. |
| **CA certificate file** | A PEM file of extra certificate authorities to trust (added to the system ones), for servers signed by a private CA. |
| **Anonymous telemetry** | Opt-in usage counters (off by default). No request contents or personal data. Same switch as `clrsig telemetry enable`. |

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
| **Check Conformance** (Activity Rail, shield icon) | Spec Conformance Testing without a terminal: choose a spec, a base URL (it starts as the spec's first server) and an environment, then **Run conformance check**. Every operation is called once and its response is checked against the spec. The Summary panel on the right holds **Export report** and the four totals (Covered, Not covered, Passing, Failing). An operation whose request still has a `{{placeholder}}` with no value is not sent: it shows as **NOT SENT**, is counted under Not covered, never as Passing, the headline says "N could not be checked", and a **Missing values** box lists every name once with a button that adds them to *Variable overrides*. The report lists the operations that break the spec first, each with what the spec expected, what the server sent, why that fails and a suggested fix (the validator's own message is under *Technical detail*), then the passing operations, then those that were not run. By default only read requests (GET, HEAD) are sent; tick **Also send write requests** to include POST, PUT, PATCH and DELETE, and the page tells you how many writes that is first. A base URL that is not on your computer needs an explicit confirmation. While it runs, a progress bar shows how many operations are done and which one is current. Under **Options** you can limit the run to tags or path prefixes, set the request timeout, override environment variables for this run (one `name=value` per line, never saved, not part of Get CLI command), add negative probes (malformed or unauthorized requests the server should refuse; the page lists any it accepted), stop after the first failure, and add the run to Contracts coverage (only offered when no tag or path filter is set, so a partial run cannot skew the totals). **Export report** saves the latest result as HTML, JUnit XML, JSON or Markdown, made by the same code as `clrsig conform`. **Get CLI command** opens the CLI Command Builder with this page's choices already filled in (spec, base URL, environment, filters, probes, timeout, and `--read-only` unless you ticked writes), so the command it shows does what the page just did. |
| **Dev Servers** (Help menu) | **Demo Server** is a well-behaved sample API; its temporary workspace includes its spec as `specs/openapi.yaml` (which its scenarios use too, so scenario runs and conformance runs share one Contracts card) and an environment that supplies every value the spec needs, so **Check Conformance** can choose that spec and run (reads only, all of which pass). **Conformance Server** is a deliberately imperfect one for trying Spec Conformance Testing: its spec (`View API Spec`) is correct, but six operations break it, one for each kind of problem the check finds: `GET /products/{id}` returns `price` as a string, `GET /categories` omits the required `name`, `GET /orders` returns `id` as a string, `GET /widgets` is served as `text/plain` although the spec says JSON, `GET /inventory` never sends the required `X-Total-Count` header, and `GET /status` answers `503`, which the spec does not declare. Download `/openapi.yaml` and `/environment.yaml` from the server's address (shown in the status bar) and run `clrsig conform openapi.yaml --env environment.yaml --base-url http://localhost:PORT`: exactly those six fail. The environment sets `id` and `name`, which the generated requests use as placeholders. For the desktop, choose **Help → Dev Servers → Start Conformance Server**: like the Demo Server, it starts the server and opens a temporary workspace with the spec and a `local` environment, with a sample scenario and requests, so **Check Conformance** and the Scenarios panel work at once (stopping the server removes that workspace and reopens your previous one; stop the Demo Server first, since only one temporary workspace can be open). If a server's usual port is taken, **Settings → Dev servers** lets you set a fixed port for each (0 picks a free one automatically); a fixed port that is already in use stops the server and says so. |
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

The `wailsjs/` directory is generated by Wails (`models.ts` and `runtime/` are gitignored; the per-service `go/main/*.js` and `*.d.ts` files are committed). If you change a Go struct that is bound to Wails, stale TypeScript types may cause compilation errors. Restart `wails dev` to regenerate the bindings.

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
