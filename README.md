# EndpointDesk

**API Contract Validation and Spec Conformance Testing — with a full API development workbench.**

EndpointDesk is an open-source, local-first desktop application that lets you verify APIs conform to their declared contracts (OpenAPI specifications, schema assertions, and response validation), detect breaking changes, and enforce conformance in CI/CD pipelines — all without a cloud account.

> **Status:** Phases 0–3 complete. The desktop app, full HTTP client, workspace management, and scenario engine + CLI are all working. See the [roadmap](#roadmap) for what is coming next.

---

## Why EndpointDesk

Most API tools focus on exploration. EndpointDesk focuses on **contract verification**: does your API actually honour its OpenAPI specification? Can you prove it, repeatably, in a pipeline?

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

### Run the CLI

```bash
cd apps/cli
go run . run ./examples/scenario.yaml --env ./examples/local.yaml
```

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
# Binary: apps/desktop/wails/build/bin/endpointdesk.exe  (Windows)
#         apps/desktop/wails/build/bin/endpointdesk       (macOS / Linux)
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
│   │   └── wails/          # Wails app, Go bindings, RequestService
│   └── cli/                # Cobra CLI (Phase 3)
│
├── packages/
│   └── core/               # Shared Go execution engine
│       ├── domain/         # Canonical domain types (Request, Response, Assertion…)
│       ├── executor/       # net/http executor — the only place HTTP I/O happens
│       ├── variables/      # {{variable}} interpolation and runtime resolver
│       ├── assertions/     # Assertion evaluation (Phase 2)
│       ├── scenario/       # Scenario runner (Phase 3)
│       ├── openapi/        # OpenAPI 3.0 / 3.1 parser (Phase 4)
│       └── …               # environment, serialization, reporting, history
│
├── docs/                   # Architecture and product documentation
├── examples/               # Example project files
├── scripts/
├── go.work                 # Go workspace (ties core + cli + desktop)
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
| 4 | OpenAPI integration — import specs, generate collections, validate responses | Planned |
| 5 | Public MVP — cross-platform packaging, docs, examples, GitHub Actions | Planned |

**MVP definition:** a developer can import an OpenAPI specification, explore and execute generated requests in the desktop app, create a multi-step scenario, save it as YAML or JSON, and run it through the CLI and GitHub Actions.

---

## Contributing

The project is in early development. Contributions, feedback, and issue reports are welcome.

Please read [CLAUDE.md](CLAUDE.md) before contributing — it describes the architecture rules, coding standards, and what is explicitly out of scope.

---

## License

[MIT](LICENSE)
