# CLAUDE.md — API Workbench (Clear Signal)

This file is Claude Code's operating contract for this repository. Read it fully before touching any code.

---

## 1. Project Identity

**Name:** Clear Signal (working title: api-workbench)
**Primary type:** Open-source, local-first **API Contract Validation platform**
**Secondary:** Full API development and testing workbench (interactive client, collections, environments, scenarios)
**Status:** Pre-implementation — engineering foundation phase (Phase 0)

The two canonical terms for this product's core purpose — use both consistently throughout the codebase, docs, and UI:

- **API Contract Validation** — the act of verifying that an API's actual behavior matches its declared contract (OpenAPI spec, schema assertions).
- **Spec Conformance Testing** — the broader practice of running tests that confirm an API conforms to its specification, including breaking change detection and CI/CD enforcement.

Do not use "contract testing" alone — it carries Pact-style consumer-driven connotations that do not apply here. Use "API Contract Validation" or "Spec Conformance Testing" instead.

The workbench features (request editor, collections, scenario runner) exist to support and complement these primary purposes.

---

## 2. Tech Stack (locked)

| Area | Decision |
|---|---|
| Core language | Go |
| Desktop framework | Wails |
| Frontend | React + TypeScript + Vite |
| UI components | Tailwind CSS + shadcn/ui |
| CLI | Cobra |
| Core HTTP | net/http |
| Configuration | YAML + JSON |
| OpenAPI | 3.0 and 3.1 |
| Project storage | Local filesystem |
| Local history | SQLite |
| Unit testing | Go testing |
| Frontend testing | Vitest + React Testing Library |
| E2E testing | Playwright |
| CI/CD | GitHub Actions |
| Distribution | GitHub Releases |

Do not introduce dependencies outside this list without explicit user approval.

---

## 3. Architecture Rules (non-negotiable)

1. **The Go core is the product.** The GUI and CLI are interfaces to it. Never implement HTTP execution, scenario evaluation, or assertion logic in the frontend or CLI — always call the shared Go packages.

2. **Dependency direction is strictly enforced:**
   ```
   Frontend → Wails bindings → Application services → Domain + engine packages
   CLI      → Application services → Domain + engine packages
   ```
   The domain and execution packages must never depend on Wails, React, or Cobra.

3. **Go core must be independent of Wails.** It must be compilable and testable without the desktop application.

4. **YAML and JSON are serialization formats, not separate models.** One canonical domain model; two serialization paths.

5. **Never expose an unauthenticated local HTTP control server.** Wails communication is restricted to trusted local application code.

6. **Credentials must be redacted** in request history, logs, and reports.

7. **Atomic writes only** for project file saves.

8. **Build vertical slices, not disconnected modules.** The first milestone must be a fully working end-to-end path, not a pile of empty packages.

---

## 4. Planned Directory Structure

```
clrsig/
├── apps/
│   ├── desktop/
│   │   ├── frontend/       # React + TypeScript + Vite
│   │   └── wails/          # Wails app bindings
│   └── cli/                # Cobra CLI
│
├── packages/
│   └── core/
│       ├── domain/
│       ├── application/
│       ├── workspace/
│       ├── serialization/
│       ├── request/
│       ├── executor/
│       ├── scenario/
│       ├── variables/
│       ├── assertions/
│       ├── environment/
│       ├── authentication/
│       ├── openapi/
│       ├── reporting/
│       └── history/
│
├── examples/
├── docs/                    # developer documents (see docs/README.md); documents the app serves live in apps/desktop/wails/help/
├── scripts/
├── go.work                 # Go workspace for local development
├── package.json
├── pnpm-workspace.yaml
└── README.md
```

This layout may be refined after the first working vertical slice — do not over-engineer it prematurely.

---

## 5. Implementation Phases

| Phase | Focus | Key Acceptance Criteria |
|---|---|---|
| 0 | Engineering foundation | App launches, Go core callable from UI, CI runs |
| 1 | HTTP execution engine | All HTTP methods work, response displayed, tested against local mock |
| 2 | Workspace + API client | Collections, environments, request history, restart-safe |
| 3 | Scenario engine + CLI | Multi-step scenarios, variable extraction, assertions, CLI parity |
| 4 | OpenAPI integration | Import spec, generate collections, validate responses |
| 5 | Public MVP | Cross-platform builds, docs, examples, GitHub Actions CI |

**Current phase: 5 (complete — public MVP)**

MVP definition: A developer can import an OpenAPI specification, explore and execute generated requests in the desktop application, create a multi-step scenario, save it as YAML or JSON, and execute the same scenario through the CLI and GitHub Actions.

---

## 6. Coding Standards

### Go
- Keep packages small and focused; one responsibility per package.
- Use interfaces at package boundaries to enable testing without mocks of infrastructure.
- Test with real local HTTP servers (httptest), not mocked HTTP clients.
- Return errors explicitly; never panic in library code.
- All exported types must have schema versioning awareness.

### TypeScript / React
- Strict TypeScript; no `any` unless wrapping an external untyped boundary.
- Components call Wails-generated bindings only — no direct HTTP from the frontend.
- State lives in the appropriate layer; do not duplicate server state in React state.

### General
- No comments that describe *what* the code does — only *why* when it's non-obvious.
- No TODO/FIXME left in committed code — convert to tracked issues.
- No feature flags, backwards-compat shims, or half-finished implementations.
- Prefer editing existing files to creating new ones.
- Do not introduce abstractions beyond what the current task requires.


### Requirement review (verify before building)

Before implementing a **major or breaking request**, check it and tell the user what you find *before* writing code, so they can decide whether to do it, change it, or drop it. Raise it when the request:

- **conflicts with an existing requirement** — this file, the locked tech stack, the architecture rules, the file format contract, out-of-scope items, or something decided earlier;
- is **inconsistent** with other requirements or with how the product already behaves;
- is **larger than it looks** — touches many packages, changes a serialized format, needs a new dependency, or needs a migration or a release;
- seems **unnecessary** — it duplicates something that exists, or a smaller change would meet the goal.

Say what the concern is in a few lines, give a recommendation (proceed, shrink it, or skip), and wait for the answer. Do not refuse or quietly change the request; once the user has decided, follow their decision. Small, clearly consistent requests need no review — just do them.

For a **major enhancement** (new UI area, new command family, format change, several packages), write a design proposal first in `docs/enhancements/` — named `YYYY-MM-DD-short-name.md`, following `docs/enhancements/README.md` — and get it agreed before building. The first one is `2026-11-17-cli-command-builder.md`.

---

## 7. Security Requirements

- Restrict Wails IPC to trusted local application code.
- Never expose an unauthenticated local HTTP control server.
- Protect secrets via env-var references for CI/CD. OS credential store is a post-MVP enhancement.
- Redact credentials in request history and all reports.
- Restrict file operations to the selected project and explicitly authorized locations.
- Warn before destructive requests against production environments.

---

## 8. File Format Contract

Every persisted project definition must:
- Carry a `schemaVersion` field.
- Be validated before execution.
- Support both YAML and JSON equivalently.
- Use atomic writes.
- Preserve unknown supported extension fields where possible.
- Have documented migration rules for future schema versions.

**Format spec rule:** Any change to a serialized field name, type, valid value set, or structure in a request, scenario, or environment file **must** update `apps/desktop/wails/help/format-spec.md` in the same commit. The spec is the canonical human reference and the source document for the AI prompt — it must stay in sync with the code.

---

## 9. What is Explicitly Out of Scope (MVP)

Cloud accounts, team sync, hosted collaboration, GraphQL, WebSockets, gRPC, load testing, mock servers, plugin marketplace, AI-generated tests, cross-platform OS credential store (Keychain / Windows Credential Manager / libsecret).

Do not implement or stub any of these without explicit user direction.

Commercial or enterprise-edition ideas, and anything else that is out of scope for this open-source project, are written as dated notes in `docs/enterprise/` (see its README) instead of being built, stubbed or added to the backlog.

---

## 10. Changelog

The engineering changelog is `docs/process/engineering-changelog.md` (dated entries, oldest first, newest at the end). Add an entry there for every significant change: what changed and why, what guards it, what is not done, and any behaviour change or risk. Do not add changelog entries to this file; it holds the rules every session must read.
