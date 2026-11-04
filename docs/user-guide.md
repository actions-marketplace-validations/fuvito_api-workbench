# EndpointDesk User Guide

## Table of Contents

1. [Opening or creating a workspace](#1-opening-or-creating-a-workspace)
2. [Working with requests](#2-working-with-requests)
3. [Understanding dirty state and saving](#3-understanding-dirty-state-and-saving)
4. [Organising requests into folders](#4-organising-requests-into-folders)
5. [Environments and variables](#5-environments-and-variables)
6. [Sending requests and reading responses](#6-sending-requests-and-reading-responses)
7. [Importing an OpenAPI specification](#7-importing-an-openapi-specification)
8. [Running scenarios from the CLI](#8-running-scenarios-from-the-cli)
9. [Settings](#9-settings)

---

## 1. Opening or creating a workspace

A **workspace** is an ordinary folder on your filesystem. EndpointDesk reads and writes files inside it; nothing is stored in a cloud account or a hidden database.

**To open an existing workspace:** File → Open Workspace… (or the folder icon in the sidebar header).

**To create a new workspace:** File → New Workspace… then pick an empty folder. EndpointDesk creates a starter structure with a sample environment and a sample request so you can see the layout immediately.

The last-used workspace is remembered between sessions.

---

## 2. Working with requests

### Creating a new request

File → New Request (or the `+` icon in the top toolbar). The editor clears and the header shows **New Request**.

### Naming a request

Click the **New Request** label (or any existing request name) in the top bar. It becomes an editable input. Type the name you want and press Enter or click away to confirm.

The name you type becomes the filename when you save:

| Name you type | File written |
|---|---|
| `List Users` | `list-users.yaml` |
| `POST /auth/login` | `post-auth-login.yaml` |
| `Health Check` | `health-check.yaml` |

Names are slugified: lowercased, spaces and underscores replaced with hyphens, special characters removed.

### Saving a request

**Ctrl+S**, File → Save Request, or the save icon in the toolbar.

- **New request (no file yet):** saved to `{workspace}/requests/{slug}.yaml`. A workspace must be open.
- **Existing request, same name:** the file is overwritten in place.
- **Existing request, name changed:** the new slug file is written, then the old file is deleted — effectively a rename.
- **Name conflict:** if a file with the same slug already exists in the folder (and it isn't the file you loaded), saving is blocked and an error message appears next to the name. Rename the request to something unique and save again.

### Loading a request from the workspace

Click any file in the **Workspace** sidebar tab. If the current request has unsaved changes, a confirmation dialog appears.

### Opening a request file from anywhere

File → Open Request… opens a system file picker. The selected file is loaded into the editor and the workspace sidebar highlights it if it is inside the current workspace.

---

## 3. Understanding dirty state and saving

### The amber dot

Whenever the editor contains changes that have not been saved to disk, a small **amber dot** appears:

- To the left of the method badge in the top bar.
- On the file row in the workspace sidebar.

The dot disappears as soon as you save (Ctrl+S) or discard.

### Navigating away with unsaved changes

If you try to open a different file, create a new request, open via dialog, or click a history entry while the amber dot is showing, a confirmation dialog appears with three options:

| Choice | What happens |
|---|---|
| **Save & Continue** | Saves the current request (using the existing name/path), then opens the new one. If the save fails (e.g. a name conflict), the dialog stays open so you can fix the error. |
| **Discard** | Abandons unsaved changes and opens the new request immediately. |
| **Cancel** | Closes the dialog and returns you to the current request. |

---

## 4. Organising requests into folders

Folders in the workspace sidebar are real subdirectories inside `{workspace}/requests/`. To create a folder, create a subdirectory there in your file manager or terminal. Any `.yaml`/`.json` files inside it appear as requests under that folder in the sidebar.

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

Click the pencil icon next to the environment selector. The environment editor opens as an overlay. Add, edit, or delete variables and click **Save**. The new environment is automatically selected.

### Using variables in requests

Reference a variable anywhere in the URL, headers, query parameters, or request body using `{{variableName}}`:

```
GET {{baseUrl}}/users/{{userId}}
Authorization: Bearer {{authToken}}
```

Unresolved references are left as-is and appear literally in the outbound request.

---

## 6. Sending requests and reading responses

1. Set the HTTP method (GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS) using the dropdown.
2. Enter the URL. Pasting a URL containing a query string (e.g. `https://api.example.com/items?page=1&limit=20`) automatically extracts the parameters into the **Params** tab.
3. Fill in the **Params**, **Headers**, **Body**, and **Auth** tabs as needed.
4. Press **Send** (or Enter in the URL bar).

### Response panel

The right panel shows:

- **Status code and text** — colour-coded green (2xx), blue (3xx), amber (4xx), red (5xx).
- **Duration and size** — total round-trip time and response body size.
- **Timing breakdown** — a stacked bar showing DNS, TCP connect, TLS handshake, time-to-first-byte, and transfer time.
- **Body** — JSON responses are syntax-highlighted and pretty-printed.
- **Headers** — full response header table.

### Request history

Click the **History** tab in the sidebar to see your last 100 requests. Clicking an entry re-populates the method and URL in the editor (the confirmation dialog appears first if there are unsaved changes).

---

## 7. Importing an OpenAPI specification

1. Open or create a workspace.
2. Click the **Import OpenAPI** button (upload icon) in the workspace panel header, or use File → Import OpenAPI Spec….
3. Select a `.yaml` or `.json` OpenAPI 3.0/3.1 file.
4. EndpointDesk generates one request file per operation, placed in `requests/{spec-title}/`.

Path parameters become `{{paramName}}` placeholders. POST/PUT/PATCH operations get a JSON body stub based on the request schema. Operations with security requirements get an `Authorization: Bearer {{token}}` header hint.

---

## 8. Running scenarios from the CLI

A **scenario** is a YAML or JSON file describing a sequence of HTTP requests with assertions and variable extractions between steps.

```bash
cd apps/cli
go run . run ../../examples/scenarios/user-lifecycle.yaml \
  --env ../../examples/server/local.yaml \
  --format text   # or json
```

Exit codes: `0` = all assertions passed, `1` = assertion failures, `2` = execution or parse error.

The same execution engine powers both the desktop app and the CLI — no behavioural differences between interactive testing and CI.

---

## 9. Settings

Open via File → Settings… or the gear icon in the toolbar.

| Setting | Description |
|---|---|
| **Theme** | Dark or light. Applied immediately; reverts if you cancel. |
| **Font size** | Scales all text in the editor and response viewer (10–20 px). |
| **Default timeout** | Request timeout in milliseconds (applies when the request does not set its own). |
| **Proxy URL** | HTTP/HTTPS proxy for all outbound requests (e.g. `http://proxy:8080`). Leave empty to use the system proxy. |
| **TLS skip verify** | Disable certificate verification. Use only for local development with self-signed certificates. |

Changes preview immediately. Click **Save** to persist; **Cancel** reverts to the last saved values.
