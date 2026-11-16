# Clear Signal — Architectural Planning Notes

> **Historical document.** This is an edited transcript of the AI-assisted architectural planning session that established the technical baseline for Clear Signal. For the current authoritative project specification, see `CLAUDE.md` and `docs/Product Requirements, Architecture & Implementation Roadmap.md`.

---

## Summary of key decisions

Three important architectural decisions were made at the outset:

1. **Desktop-first:** Build a native desktop application using Wails, Go, and React.
2. **Dual-format project files:** Support YAML and JSON through one canonical internal data model.
3. **OpenAPI in MVP:** Make OpenAPI import and schema validation part of the initial release, not a future enhancement.

The CLI is retained as a first-class component even though desktop is the primary delivery method. Automated execution is one of the strongest differentiators for contract testing.

---

## 1. Product architecture

```
React UI (TypeScript, Vite, Tailwind, shadcn/ui)
         │
      Wails (native window and Go bindings)
         │
  ┌──────┴──────────────────────────────────┐
  │       Shared Go application core        │
  │  Request engine · Scenario engine       │
  │  Assertion engine · OpenAPI engine      │
  │  Variable manager · Authentication      │
  │  Environment manager · Report generator │
  └──────┬──────────────────────────────────┘
         │
  Storage & project formats
  (YAML · JSON · OpenAPI · SQLite history)
  Filesystem-based, local-first, Git-compatible
```

The CLI application uses the same Go core as an independent executable, providing CI/CD integration and machine-readable reports.

**Critical architecture rule:** The GUI and CLI must never implement their own HTTP execution, scenario evaluation, or assertion logic. They both call the same Go packages.

---

## 2. Technical decisions

| Area | Decision |
|---|---|
| Language | Go |
| Desktop framework | Wails |
| Frontend | React + TypeScript + Vite |
| UI | Tailwind CSS + shadcn/ui |
| CLI | Cobra |
| Core HTTP | net/http |
| Configuration | YAML + JSON |
| OpenAPI | 3.0 and 3.1 |
| JSON Schema | Draft 2020-12 support target |
| Project storage | Local filesystem |
| Local history | SQLite |
| Unit testing | Go testing |
| Frontend testing | Vitest + React Testing Library |
| E2E testing | Playwright |
| CI/CD | GitHub Actions |
| Distribution | GitHub Releases |

---

## 3. YAML and JSON strategy

Both formats are supported natively through a single internal representation:

```
request.yaml ──┐
               ├──> Domain Model ──> Execution Engine
request.json ──┘
```

There are no separate YAML and JSON execution implementations.

- YAML is the default for newly created project files.
- JSON is supported for import, export, and direct editing.
- Users can choose their preferred format per project.
- The GUI operates on the canonical domain model.
- Serialization preserves supported fields and schema versions.

---

## 4. Why SQLite for history?

Collections, environments, and scenarios belong in ordinary files. Request history, execution history, and recent project metadata are better suited to a local database. SQLite is appropriate because it requires no server and has a small operational footprint.

---

## 5. Desktop security principles

- Restrict Wails communication to trusted local application code.
- Never expose an unauthenticated local HTTP control server.
- Protect secrets using OS credential store where available.
- Redact credentials in request history and reports.
- Restrict file operations to the selected project and explicitly authorized locations.
- Provide warnings for destructive requests against production environments.

---

## 6. Functional modules

| Module | Responsibility |
|---|---|
| Workspace | Projects, collections, folders and files |
| API Client | Build and execute individual requests |
| Scenario Runner | Execute multi-step workflows |
| OpenAPI | Import specifications and validate requests/responses |
| Environment | Variables, credentials and authentication |
| Reporting | Execution results and CI/CD reports |

---

## 7. Build vertical slices, not disconnected modules

The first milestone must be a complete vertical slice — not a collection of empty packages:

1. Launch the Wails desktop application.
2. Open a project directory.
3. Create a GET request.
4. Execute it through the Go engine.
5. Display the response in React.
6. Save the request as YAML.
7. Load the same request through the CLI.

This proves the most important architectural assumption before investing in scenario editing, OpenAPI, and advanced UI.

---

## 8. Repository organisation

```
clrsig/
│
├── apps/
│   ├── desktop/
│   │   ├── frontend/
│   │   └── wails/
│   │
│   └── cli/
│
├── packages/
│   └── core/
│       ├── domain/
│       ├── application/
│       ├── executor/
│       ├── scenario/
│       ├── openapi/
│       └── reporting/
│
├── examples/
├── docs/
├── scripts/
│
├── go.work
├── package.json
├── pnpm-workspace.yaml
└── README.md
```

The core is kept as an independently compilable Go module; a Go workspace (`go.work`) is used for local development to avoid coupling the core to the desktop application lifecycle.

---

## Guiding principle

The Go engine is the actual product foundation; the desktop application is its most visible interface. If the engine is built correctly, additional interfaces, protocols, integrations, and automation workflows can be added without rewriting the core.
