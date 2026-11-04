# ApiDesk

**API Contract Validation and Spec Conformance Testing — with a full API development workbench.**

ApiDesk is an open-source, local-first desktop application that lets you verify APIs conform to their declared contracts (OpenAPI specifications, schema assertions, and response validation), detect breaking changes, and enforce conformance in CI/CD pipelines — all without a cloud account.

> **Status:** Phases 0–5 complete — public MVP. The desktop app, full HTTP client, workspace management, OpenAPI import, scenario engine + CLI, settings, timing breakdown, and cross-platform release workflow are all working.

---

## Why ApiDesk

Most API tools focus on exploration. ApiDesk focuses on **contract verification**: does your API actually honour its OpenAPI specification? Can you prove it, repeatably, in a pipeline?

- **API Contract Validation** — validate that actual HTTP responses match declared OpenAPI schemas.
- **Spec Conformance Testing** — run scenario-based tests that enforce your API contract across a full user workflow.
- **Local-first** — no mandatory account, no cloud sync, no data leaving your machine.
- **Git-native** — all project definitions are plain YAML or JSON files. Version-control them like code.
- **CLI and GUI parity** — the same Go execution engine powers both. Run the same tests interactively or in GitHub Actions.

---

## Tech Stack

| Layer | Technology |
|---|---|
| Core engine | Go 1.26 |
| Desktop | Wails v2 |
| Frontend | React 19 + TypeScript 7 + Vite 8 |
| UI | Tailwind CSS v4 + shadcn/ui |
| CLI | Cobra |
| Project files | YAML + JSON |
| API specification | OpenAPI 3.0 / 3.1 |
| History | SQLite |
| Package manager | pnpm 9 (frontend), Go workspace (backend) |

---

## Prerequisites

| Tool | Version |
|---|---|
| Go | 1.26+ |
| Node.js | 22+ |
| pnpm | 9+ |
| Wails CLI | v2 (`go install github.com/wailsapp/wails/v2/cmd/wails@latest`) |

Wails also requires a C compiler on the host. On Windows, [TDM-GCC](https://jmeubank.github.io/tdm-gcc/) or [MSYS2](https://www.msys2.org/) works. See the [Wails prerequisites guide](https://wails.io/docs/gettingstarted/installation) for platform-specific setup.

---

## Quick Start

```bash
git clone https://github.com/fuvito/api-workbench.git
cd api-workbench

# Install frontend dependencies
pnpm install --dir apps/desktop/frontend

# Launch the desktop app (dev mode with hot reload)
cd apps/desktop/wails
wails dev
```

The application opens automatically. Type a URL in the request bar and press **Send** or **Enter**.

### Run the example server locally

```bash
# In a separate terminal
cd examples/server
go run .
# Listening on :8080 — includes /health, /users CRUD, /auth/login, /protected
```

### Run a scenario with the CLI

```bash
cd apps/cli
go run . run ../../examples/scenarios/user-lifecycle.yaml --env ../../examples/server/local.yaml
```

### Import an OpenAPI spec

Open the desktop app → open a workspace → click **Import OpenAPI** in the workspace panel → select `examples/server/openapi.yaml`. All operations are saved as individual request files and appear in the workspace tree.

---

## How Workspaces Work

ApiDesk organises requests as **plain YAML files on disk** — not in a proprietary binary format, not in a database. This means your collection is fully version-controllable with git, diffable in any editor, and portable across machines.

### Folder structure

```
my-workspace/
├── environments/
│   ├── local.yaml          # Environment variables for local development
│   └── staging.yaml
└── requests/
    ├── health-check.yaml   # Individual request files
    ├── auth/
    │   ├── login.yaml
    │   └── refresh-token.yaml
    └── users/
        ├── list-users.yaml
        └── create-user.yaml
```

- `environments/` — variable sets (base URL, auth tokens, etc.). Select the active environment from the top bar.
- `requests/` — one YAML file per request; subdirectories are folders in the sidebar tree.

### Name = filename

**The request's name is its filename.** When you save a request named `List Users`, it writes to `list-users.yaml`. Renaming the request (click the name in the header, type a new name, Ctrl+S) renames the file automatically — no separate "Save As" dialog, no filename-vs-name mismatch.

Uniqueness is enforced at save time: two requests in the same folder cannot share a name, because they would map to the same file. If a conflict is detected, an error is shown inline in the header and no file is written.

### Dirty state

An **amber dot** appears in the header whenever the current request has unsaved changes. The same dot appears on the file in the workspace sidebar. Navigating away (opening another file, clicking New Request, switching history) triggers a confirmation dialog with three choices: **Save & Continue**, **Discard**, or **Cancel**.

### File format

Each request file is human-readable YAML:

```yaml
schemaVersion: "1"
name: List Users
method: GET
url: "{{baseUrl}}/users"
headers:
  - name: Accept
    value: application/json
queryParams:
  - name: page
    value: "1"
```

You can edit these files directly in any text editor. `{{variable}}` references are resolved against the active environment at request time.

---

## Development

### Run all Go tests

```bash
# From the repo root
go test ./packages/core/...
```

### Build the desktop app (production binary)

```bash
cd apps/desktop/wails
wails build
# Binary: apps/desktop/wails/build/bin/apidesk.exe  (Windows)
#         apps/desktop/wails/build/bin/apidesk       (macOS / Linux)
```

### Regenerate TypeScript bindings

Bindings are generated automatically by `wails build` or `wails dev`. They are gitignored and must not be committed.

---

## Project Structure

```
api-workbench/
├── apps/
│   ├── desktop/
│   │   ├── frontend/       # React + TypeScript + Vite (pnpm workspace)
│   │   └── wails/          # Wails app, Go services (request, workspace, openapi, settings)
│   └── cli/                # Cobra CLI — apidesk run
│
├── packages/
│   └── core/               # Shared Go execution engine
│       ├── domain/         # Canonical domain types (Request, Response, Assertion, Settings…)
│       ├── executor/       # net/http executor with httptrace timing breakdown
│       ├── variables/      # {{variable}} interpolation and runtime resolver
│       ├── assertions/     # Assertion evaluation (status, header, jsonpath, schema)
│       ├── scenario/       # Multi-step scenario runner with FailPolicy and variable extraction
│       ├── openapi/        # OpenAPI 3.0/3.1 parser, request generator, response validator
│       ├── serialization/  # YAML/JSON round-trip for requests, environments, scenarios
│       ├── reporting/      # Text and JSON report output for CLI
│       └── history/        # SQLite-backed request history (credentials never stored)
│
├── examples/
│   ├── server/             # Local Go HTTP server for end-to-end testing
│   │   └── openapi.yaml    # OpenAPI 3.0.3 spec for the example server
│   └── scenarios/          # Sample scenarios (user-lifecycle.yaml)
│
├── docs/                   # Architecture and product documentation
├── .github/workflows/
│   ├── ci.yml              # Push/PR: core tests → CLI build → desktop build
│   └── release.yml         # Tag push: cross-platform builds → GitHub Release
├── go.work                 # Go workspace (ties core + cli + desktop + examples/server)
├── pnpm-workspace.yaml
└── CLAUDE.md               # Engineering rules and changelog
```

### Architecture rule

The GUI and CLI must never implement their own HTTP execution, assertion, or scenario logic. Both call the same packages in `packages/core`. See [CLAUDE.md](CLAUDE.md) for the full engineering contract.

---

## Roadmap

| Phase | Focus | Status |
|---|---|---|
| 0 | Engineering foundation — monorepo, Wails app, domain models, HTTP executor | **Complete** |
| 1 | HTTP execution — all body types, auth, error handling, request history | **Complete** |
| 2 | Workspace and API client — collections, environments, restart-safe config | **Complete** |
| 3 | Scenario engine + CLI — multi-step flows, variable extraction, assertions, text/JSON reports | **Complete** |
| 4 | OpenAPI integration — import specs, generate collections, validate responses | **Complete** |
| 5 | Public MVP — settings, timing breakdown, cross-platform release workflow, examples | **Complete** |

**MVP complete.** A developer can import an OpenAPI specification, explore and execute generated requests in the desktop app, inspect detailed timing breakdowns (DNS / Connect / TLS / TTFB / Transfer), create a multi-step scenario, save it as YAML or JSON, and run it through the CLI and GitHub Actions.

---

## Contributing

The project is in early development. Contributions, feedback, and issue reports are welcome.

Please read [CLAUDE.md](CLAUDE.md) before contributing — it describes the architecture rules, coding standards, and what is explicitly out of scope.

---

## License

[MIT](LICENSE)
