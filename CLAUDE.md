# CLAUDE.md — API Workbench (Clear Signal)

This file is Claude Code's operating contract for this repository. Read it fully before touching any code.

---

## 1. Project Identity

**Name:** Clear Signal (working title: api-workbench)
**Primary type:** Open-source, local-first **API contract testing platform**
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
├── docs/
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

---

## 10. Changelog

### 2026-11-03
- CLAUDE.md created from `docs/Project.md` and `docs/Product Requirements, Architecture & Implementation Roadmap.md`.
- Project is at pre-implementation stage; no code exists yet.
- Engineering foundation (Phase 0) is the active work target.
- Clarified primary product identity: API contract validation and spec conformance testing platform first; full workbench second.
- Moved cross-platform OS credential store (Keychain / Windows Credential Manager / libsecret) to post-MVP scope. MVP uses env-var references for secrets.
- Locked canonical terminology: use "API Contract Validation" and "Spec Conformance Testing" throughout. Never use "contract testing" alone (Pact connotations).

### 2026-11-03 — Task 1: Monorepo + Go Workspace (Phase 0)
- Created full monorepo directory structure under `apps/`, `packages/core/`, `examples/`, `scripts/`.
- Initialized Go modules: `packages/core`, `apps/cli`, `apps/desktop/wails` (all `go 1.26`).
- Created `go.work` workspace tying all three modules together; `go.work.sum` excluded from git.
- Created root `package.json` and `pnpm-workspace.yaml` (workspace: `apps/desktop/frontend`).
- Installed Wails CLI v2.17.1 (ready for Task 2).
- Added `packages/core/doc.go` and `packages/core/core_test.go` smoke test (`TestCoreSanity` passes).
- Added `apps/cli/main.go` stub.
- Created `.github/workflows/ci.yml`: core tests → CLI build → desktop build on push/PR to main.
- Updated `.gitignore`: removed `go.work` exclusion (committed); added `node_modules/`, `dist/`, `build/bin/`.
- All modules build clean; core test passes.

### 2026-11-03 — Task 2: Wails + React application (Phase 0)
- Installed Wails v2.17.1 CLI; added Wails v2 as Go dependency to `apps/desktop/wails`.
- Created `apps/desktop/wails/main.go` (Wails app entry, `//go:embed all:frontend/dist`) and `app.go` (App struct + startup hook).
- Created `apps/desktop/wails/wails.json` with `frontend:dir: "../frontend"` pointing to the pnpm workspace frontend.
- Created Vite 8 + React 19 + TypeScript 7 app in `apps/desktop/frontend` via `create-vite --template react-ts`.
- Installed Tailwind CSS v4 (`@tailwindcss/vite` plugin — no config file needed).
- Installed `clsx`, `tailwind-merge`, `lucide-react`; wrote `components.json` for shadcn/ui.
- Wrote `src/lib/utils.ts` (cn utility), `src/index.css` (Tailwind + dark theme tokens), `src/App.tsx` (3-panel shell: sidebar / request editor / response viewer).
- Configured `vite.config.ts`: `outDir: '../wails/frontend/dist'` so Go embed works; `@` path alias.
- Added `apps/desktop/wails/frontend/dist/` to `.gitignore` (build artifact).
- Updated CI: desktop job now builds frontend first, then `go build` the Wails module.
- Frontend builds clean (177ms); `go build ./...` on desktop module succeeds.

### 2026-11-03 — Task 3: Domain models (Phase 0)
- Created `packages/core/domain/` with 8 files covering the full domain model.
- `request.go` — `HTTPMethod` (7 consts + `IsValid()`), `Header`, `QueryParam`, `BodyType` (5 consts), `RequestBody`, `Request` (with `Validate()`).
- `auth.go` — `AuthType` (none/bearer/basic/apikey), `APIKeyLocation`, `BearerAuth`, `BasicAuth`, `APIKeyAuth`, `Auth`.
- `response.go` — `Response` with `IsSuccess()` and `BodyString()`.
- `assertion.go` — `AssertionType` (status/header/jsonpath/**schema**), `Operator` (6 ops), `Assertion`, `AssertionResult`. Schema assertion type is first-class for API Contract Validation.
- `variable.go` — `Variable`, `ExtractionSource` (jsonpath/header/status), `Extraction`.
- `environment.go` — `Environment` + `EnvironmentVariable` (with `Secret` flag for redaction), `NewEnvironment()`, `Lookup()`. Defines `CurrentSchemaVersion = "1"`.
- `collection.go` — `Collection`, `CollectionItem`, `Folder`.
- `scenario.go` — `Scenario`, `ScenarioStep`, `FailPolicy` (stop/continue/skip), `RetryPolicy`.
- `workspace.go` — `Workspace` marker type, `NewWorkspace()`.
- `domain_test.go` — 6 tests; all pass.

### 2026-11-03 — Task 4: HTTP executor + variable resolver (Phase 0)
- `packages/core/executor/executor.go` — `Executor` struct; `Execute(ctx, req)` → `(*Response, error)`. Per-request timeout via `context.WithTimeout`; falls back to `Options.DefaultTimeout` (30s). Response body capped at 50 MB. `applyAuth` handles Bearer, Basic, APIKey (header + query). Content-Type set automatically from `BodyType` when not explicitly provided.
- `packages/core/executor/executor_test.go` — 13 tests against `httptest.NewServer`: all methods, JSON body, query params, all 3 auth types, timeout, context cancellation, response size, invalid request. All pass against real local HTTP servers (no mocks).
- `packages/core/variables/resolver.go` — `Resolver` with `Interpolate(string)` and `InterpolateRequest(*Request) *Request`. Env variables loaded at construction; runtime variables (`Set`) override. Unresolved `{{refs}}` left unchanged. Original request never mutated.
- `packages/core/variables/resolver_test.go` — 4 tests including whitespace trimming, runtime override, full request interpolation (URL, headers, query params, Bearer token), nil env. All pass.
- Total: 27 tests passing across core, domain, executor, variables.

### 2026-11-03 — Phase 3: Scenario engine + CLI

- **`packages/core/assertions/evaluator.go`** — `Evaluator` with `Evaluate(Assertion, Response) AssertionResult` and `EvaluateAll`. Supports: status (eq/ne/gt/lt with numeric coercion), header (case-insensitive, all 6 operators), jsonpath (PaesslerAG/jsonpath, value-to-string comparison). Schema assertions return "requires Phase 4" message. 13 tests; all pass.
- **`packages/core/scenario/runner.go`** — `Runner` executes all steps sequentially. `StepResult` captures request, response, assertion results, extracted variables, pass/skip/error state. `FailPolicy` semantics: `stop`/`""` (default) skips remaining steps; `continue` keeps going; `skip` skips this step if prev failed. Variable extraction feeds `resolver.Set()` so subsequent steps see extracted values. `Result.Counts()` and `AssertionCounts()` for reporting. 6 tests; all pass.
- **`packages/core/serialization/scenario.go`** — `ScenarioFile`, `StepFile`, `AssertionFile`, `ExtractionFile` DTOs with YAML/JSON tags. `ParseScenario` / `ParseScenarioFile` / `WriteScenario`. Handles inline requests and `requestRef` paths. 3 new tests added to `serialization_test.go` (parse YAML, round-trip, requestRef); all pass.
- **`packages/core/reporting/report.go`** — `Text(result)` (plain ASCII, CI-safe) and `JSON(result) ([]byte, error)`. `assertionLabel` formats human-readable assertion descriptions (`status eq 200`, `header Content-Type contains json`, `jsonpath $.name eq Alice`). `JSONReport` / `JSONStep` / `JSONAssertion` structs for machine-readable output.
- **`apps/cli/main.go`** — Cobra CLI. `clrsig run <scenario-file> [--env <env-file>] [--format text|json]`. Exit code 0 = all pass, 1 = assertion failures, 2 = execution/parse error. Reads scenario and env files, runs via `scenario.Runner`, outputs `reporting.Text` or `reporting.JSON`.
- **`apps/cli/go.mod`** — added `replace github.com/fuvito/api-workbench/packages/core => ../../packages/core`; added `PaesslerAG/jsonpath`, `yaml.v3` as indirect deps.
- **UI: URL param extraction on paste** — pasting a URL containing `?` (e.g. `https://api.example.com/items?page=1&limit=20`) automatically strips the query string from the URL field and merges the extracted params into the Params tab. Same-named existing params are replaced; new ones are appended. No change to typing behavior.
- **Phase 3 acceptance criteria met**: multi-step scenarios run end-to-end; variable extraction propagates between steps; assertions evaluated with 6 operators; scenarios saved/loaded as YAML or JSON; `clrsig run` CLI command with text/JSON output and proper exit codes; 44+ Go tests all pass.

### 2026-11-03 — Phase 2: Workspace + API client

- **`packages/core/serialization`** — new package. `ParseRequest` / `ParseRequestFile` / `WriteRequest` for domain.Request; `ParseEnvironment` / `WriteEnvironment` for domain.Environment. Format detected from file extension (`.yaml`/`.yml`/`.json`). Full round-trip fidelity. 6 tests (YAML parse, JSON parse, file extension detect, request round-trip, environment parse, environment round-trip).
- **`packages/core/go.mod`** — added `gopkg.in/yaml.v3 v3.0.1` and `modernc.org/sqlite v1.61.1`.
- **`apps/desktop/wails/workspace_service.go`** — new Wails-bound service. Methods: `OpenWorkspace` (dialog → sets path), `NewWorkspace` (dialog + creates example structure if empty), `GetWorkspacePath`, `ListRequests` (file tree, skips `environments/` dir), `ListEnvironments` (scans `workspacePath/environments/`), `SelectEnvironment` (loads env, sets active), `GetActiveEnvironment`, `LoadRequest` (parse file → `RequestFileIO`), `SaveRequest` (Wails save dialog → atomic write), `OpenRequest` (Wails open dialog → `RequestFileIO`).
- **Config persistence** — `~/.clrsig/config.json` persists workspace path and active environment path across restarts (atomic write). Loaded on `NewWorkspaceService()` startup; active environment re-loaded from file at startup.
- **Variable interpolation** — `RequestService.Execute` now reads `WorkspaceService.ActiveEnvironment()` and passes it to `variables.NewResolver(env)`. Switching environments in the UI immediately affects variable resolution in the next request.
- **Example workspace** — `NewWorkspace` on an empty directory auto-creates `environments/local.yaml` and `requests/health-check.yaml` as a starting point.
- **UI: sidebar tabs** — `Workspace` and `History` tabs; Workspace tab shows the `WorkspacePanel` component; History tab shows the existing history list.
- **UI: WorkspacePanel** — recursive file tree (`TreeNode` component, expand/collapse directories), Open Workspace and New Workspace buttons when no workspace is set.
- **UI: environment selector** — `<select>` in the top bar; hidden when no workspace has environments; `No environment` option; selecting an env calls `selectEnvironment` on Go side.
- **UI: toolbar buttons** — Open (folder icon, opens file dialog), Save (saves current request via dialog), New (clears editor).
- **`bridge.ts`** — 10 new bridge functions wrapping the workspace service Wails bindings.
- All 37 Go tests pass. Full `wails build` succeeds producing `clrsig.exe`.
- **Phase 2 acceptance criteria met**: collections (file-based), environments (selectable, active env drives variable resolution), request history (Phase 1), restart-safe (config.json persists workspace + env between sessions).

### 2026-11-03 — Phase 1: HTTP execution engine + request editor fully wired

- **UI: Params tab** — `KeyValueEditor` component with checkbox-toggle rows, add/remove, header labels. Enabled params flow into `queryParams` on the outbound request; disabled or nameless rows are skipped.
- **UI: Headers tab** — same `KeyValueEditor` component, bound to `headers`. Passes custom headers to the Go executor.
- **UI: Body tab** — `BodyEditor` component: type selector (none/JSON/raw/form-urlencoded), textarea with per-type placeholder, Format button for JSON that pretty-prints in-place.
- **UI: Auth tab** — `AuthEditor` component: type selector (none/Bearer/Basic/APIKey) with conditional fields for each type. APIKey lets the user choose header vs. query-param injection location.
- Tab badges — tabs show a count badge (Params, Headers) or a dot (Body, Auth) when active, so the user can see at a glance what is configured.
- **History sidebar** — left sidebar now shows the last 100 requests (most recent first) with method badge, URL, status code, and duration. Clicking an entry pre-fills the method and URL in the request editor.
- **History state management** — `App` loads history on mount via `useEffect` and refreshes after every request via `useCallback`+`refreshHistory`.
- **`packages/core/history`** — new package. `Store` wraps a SQLite database (`modernc.org/sqlite` v1.61.1, pure Go). `Open(path)` → `migrate()` → `Record(...)` → `Recent(limit)` → `Close()`. Auth tokens and request bodies are **never written** to protect secrets. 4 tests: record + recent, limit, empty store, `DefaultPath` directory creation.
- **`history.DefaultPath()`** — resolves to `~/.clrsig/history.db`; creates the directory if absent.
- **`RequestService.GetHistory()`** — new Wails-bound method returning `[]HistoryEntryIO`; maps `history.Entry` to JSON-serializable struct.
- **`RequestService.recordHistory()`** — private helper called on every successful execution; silently skips on error so a broken history DB never blocks requests.
- **`app.shutdown`** hook added → calls `requestService.close()` → closes SQLite connection cleanly.
- **`bridge.ts`** — `AuthPayload` exported as a named type; `getHistory()` added, wraps `WailsGetHistory`.
- **`src/types.ts`** — new file: `KVPair`, `BodyState`/`BodyType`, `AuthState`/`AuthType` discriminated union.
- Wails build regenerates bindings with `GetHistory` and produces clean `clrsig.exe`.
- All 31 Go tests pass (`core`, `domain`, `executor`, `history`, `variables`).
- **Phase 1 acceptance criteria met**: all 7 HTTP methods work, custom params/headers/body/auth flow end-to-end through the UI, response status and duration displayed, executor tested against local mock servers (13 tests).

### 2026-11-03 — Task 5: React ↔ Go executor wired (Phase 0 complete)
- `apps/desktop/wails/request_service.go` — `RequestService` struct with `Execute(RequestInput) ResponseOutput`. Input/output types use JSON tags and are Wails-serializable. Maps domain types in both directions; network errors returned in `ResponseOutput.Error` (not Go error) so the UI always receives a consistent type.
- Updated `app.go` — `App` holds `RequestService`; app context propagated to service on `startup`.
- Updated `main.go` — `app.requestService` bound to Wails alongside `app`.
- Added `replace github.com/fuvito/api-workbench/packages/core => ../../../packages/core` to `apps/desktop/wails/go.mod` — required because Wails's binding generator runs in a temp directory outside the Go workspace and cannot see `go.work`.
- `wails build` runs successfully: generates TypeScript bindings → builds React → compiles Go → produces `build/bin/clrsig.exe`.
- Generated `wailsjs/` added to `.gitignore` (regenerated by `wails build` / `wails dev`).
- `src/lib/bridge.ts` — typed wrapper around Wails-generated bindings using plain interfaces (`RequestPayload`, `ResponseData`). Isolates the `as any` cast in one place; avoids Wails class types at call sites.
- Updated `tsconfig.app.json` — `erasableSyntaxOnly: false` (Wails-generated `models.ts` uses `namespace`, which TypeScript 7 otherwise rejects). Added `@wailsjs` path alias.
- Updated `vite.config.ts` — `@wailsjs` resolve alias pointing to `wailsjs/` directory.
- `src/App.tsx` — fully wired: method selector, URL input (Enter key submits), Send button → `executeRequest` → response panel (status badge, duration, size, pretty-printed JSON body, headers table, error state, loading state).
- **Phase 0 acceptance criteria met**: desktop app builds and launches; Go core callable from UI through Wails IPC; CI workflow validates core tests, CLI build, and desktop build. Run `wails dev` from `apps/desktop/wails/` to launch the full app.

### 2026-11-03 — Phase 4: OpenAPI integration

- **`packages/core/openapi/parser.go`** — `Spec` wraps `openapi3.T` from `github.com/getkin/kin-openapi v0.152.0`. `ParseBytes(data, format)` and `ParseFile(path)` for YAML/JSON input. `Info()` returns title + version. `GeneratedRequest` carries `domain.Request` plus `OperationID`, `Summary`, `Tags`. `GenerateRequests(s, baseURL)` iterates all paths/operations, substitutes `{{paramName}}` path placeholders, adds query params, sets `Authorization: Bearer {{token}}` hint for secured operations, and generates a JSON body stub for POST/PUT/PATCH with a schema body.
- **`packages/core/openapi/validator.go`** — `ValidateResponse(s, operationID, resp)` and `ValidateResponseByPath(s, method, path, statusCode, resp)` return `[]ValidationError`. Uses `kin-openapi` `ResponseValidationInput` pipeline. Schema assertions in `assertions/evaluator.go` now call this directly.
- **`packages/core/openapi/openapi_test.go`** — 9 tests all passing: parse valid/invalid spec, generate count, baseURL injection, path params, JSON body stub, response pass, response fail (missing field), unknown operation.
- **`packages/core/assertions/evaluator.go`** — schema assertion case now fully implemented via `openapi.ParseBytes` + `openapi.ValidateResponse`.
- **`examples/server/main.go`** — standalone Go HTTP server (users CRUD, login, protected, health). Pre-seeded with Alice (id=1) and Bob (id=2); `nextID=3`. Auth: POST `/auth/login` with `admin`/`secret` returns `demo-token-12345`. `examples/server/go.mod` created; `go.work` updated.
- **`examples/server/openapi.yaml`** — OpenAPI 3.0.3 spec with 8 operations and full response schemas.
- **`examples/scenarios/user-lifecycle.yaml`** — 4-step scenario (Login → Create user → Fetch user → Delete user); all 9 assertions verified against the example server.
- **`apps/desktop/wails/openapi_service.go`** — `OpenAPIService` with `ImportOpenAPISpec(baseURL)` (file dialog) and `ImportOpenAPISpecFromPath(path, baseURL)`. Saves one YAML request file per operation under `workspace/requests/<spec-title>/`. Sanitises directory and file names.
- **`apps/desktop/wails/helpers.go`** — shared `atomicWrite(path, data)` helper used by workspace, settings, and OpenAPI services.
- **UI: Import OpenAPI button** — `WorkspacePanel` gains an ⇡ button (shown when workspace is open); calls `importOpenAPISpec`; shows spinner while importing; reloads workspace tree on completion.
- **`bridge.ts`** — `ImportedRequest`, `ImportResult` types; `importOpenAPISpec` and `importOpenAPISpecFromPath` functions.
- All Go tests pass. `wails build` succeeds.
- **Phase 4 acceptance criteria met**: import OpenAPI spec from file, generate request files for all operations with path-param placeholders and body stubs, validate responses against schema, schema assertions wired into the assertion evaluator.

### 2026-11-03 — Phase 5: Public MVP

- **API timing breakdown** — `domain.Timing` struct (`DNSMs`, `ConnectMs`, `TLSMs`, `TTFBMs`, `TransferMs`, `TotalMs()`). `executor.go` attaches `net/http/httptrace.ClientTrace` to the request context; timestamps captured at DNS start/done, connect done, TLS done, first-byte, body read complete. `TimingIO` added to `ResponseOutput` in Wails service. `TimingBar` React component renders a 5-segment stacked bar (DNS=purple, Connect=blue, TLS=yellow, TTFB=emerald, Transfer=cyan) in the response panel.
- **TLS skip-verify and proxy support** — `executor.Options` gains `ProxyURL string` and `TLSSkipVerify bool`. `New()` builds an `http.Transport` with `crypto/tls.Config{InsecureSkipVerify: ...}` and `http.ProxyFromEnvironment` / `url.Parse` proxy. `RebuildExecutor()` on `RequestService` re-creates the transport when settings change.
- **Settings** — `domain.Settings` (`Theme`, `FontSize`, `DefaultTimeoutMs`, `ProxyURL`, `TLSSkipVerify`), `DefaultSettings()`. `SettingsService` persists to `~/.clrsig/settings.json` via `atomicWrite`; `load()` fills zero values with defaults; `onSave func()` callback triggers executor rebuild. `GetSettings`, `SaveSettings`, `ResetSettings` Wails-bound methods. `app.go` wires `ss.onSave = rs.RebuildExecutor`.
- **Settings UI** — `SettingsPanel` React component: modal overlay, Appearance section (theme select, font-size range), Network section (timeout, proxy URL, TLS skip checkbox). Gear icon in top bar → `setShowSettings(true)`. Settings loaded on mount via `getSettings()`.
- **Theme + font-size** — `applySettings(s)` sets `--font-size-code` CSS variable and toggles `document.documentElement.classList` between `dark` and `light`. `:root.light` block in `index.css` overrides all colour tokens for the light theme.
- **CI** — `.github/workflows/ci.yml` updated to `go test -v`.
- **Release workflow** — `.github/workflows/release.yml` triggers on `v*` tags. Four jobs: `release-windows` (windows-latest, `wails build -platform windows/amd64`), `release-macos` (macos-latest, arm64 + amd64), `release-linux` (ubuntu-latest, installs `libgtk-3-dev libwebkit2gtk-4.0-dev`), `cli-binaries` (matrix: win/mac/linux × amd64 + darwin/arm64). Final `publish` job downloads all artifacts and creates a GitHub Release via `softprops/action-gh-release@v2`; pre-release flag set automatically for tags containing `-`.
- **Phase 5 / MVP acceptance criteria met**: developer can import an OpenAPI spec, execute generated requests with timing breakdown, configure app settings (theme, font size, timeout, proxy, TLS), create and run multi-step scenarios through the desktop app and CLI, and ship cross-platform binaries via GitHub Actions on tag push.

### 2026-11-14 — Format spec + AI prompt (Help menu)

- **`apps/desktop/wails/help/format-spec.md`** — new canonical YAML format specification covering request, scenario, and environment file formats; assertion and extraction references; variable syntax; file naming conventions.
- **`help_service.go`** — added `GetFormatSpec()` and `GetAIPrompt()`. `GetAIPrompt()` returns a complete system prompt (4-step workflow: discovery survey → generate → self-verify → deliver; 8 best-practice rules; a 15-point verification checklist) followed by the full spec.
- **`HelpPanel.tsx`** — refactored to tabbed layout: **User Guide** | **Format Spec**. Format Spec tab has a "Copy AI Prompt" button (clipboard, 2 s flash) and a one-line hint explaining the AI workflow.
- **`CLAUDE.md` Section 8** — added rule: any serialization field change must update `format-spec.md` in the same commit.

### 2026-11-14 — Scenario narrative layer (label + notes)

- **Domain**: `ScenarioStep.Title` → `Label`, `ScenarioStep.Description` → `Notes`. Scenario-level `Title`/`Description` unchanged.
- **Serialization**: YAML/JSON step fields renamed `title`→`label`, `description`→`notes`. Existing files using `title:` on steps must be migrated.
- **Reporting + runner + validate**: all references updated; validation error messages updated to "step label is required" / "duplicate step label".
- **Wails service**: `StepDetailIO` and `StepResultIO` updated (`label`/`notes` JSON tags); new scenario starter YAML uses `label:`.
- **Frontend**: `StepDetail.label`/`notes`, `StepResult.label`; `ScenarioStepList` shows `notes` as a secondary italic line under the label; step editor uses "Step label" / "Notes (optional)" placeholders; `scenario-yaml.ts` serializes `label:`/`notes:`.
- **Templates + examples**: all 11 scenario template files and `examples/scenarios/user-lifecycle.yaml` migrated to `label:`/`notes:` with example notes added.

### 2026-11-16 — Bundle import (single-file AI output)

- **Format:** a *bundle* is one YAML/JSON document with `environments`, `requests` and `scenarios`, each item using the existing file format plus a `file` name; optional top-level `specFile` default. Decoded strictly (unknown fields are errors). Documented in `format-spec.md` ("Bundle File").
- **Core:** `serialization.ParseBundle` (strict decode) and the new `packages/core/bundle` package (`ParseText` tolerates ``` fences/BOM, `Validate` with located errors and warnings, `Prepare` resolves skip/overwrite/rename conflicts and rewrites `requestRef`s, `Plan.Write` writes atomically).
- **Desktop:** `BundleService` (`PickBundleFile`, `PreviewBundle`, `ImportBundle`) and `BundleImportDialog` under File → Import → Bundle.
- **Export:** File → Export → Bundle writes the whole workspace as one bundle (`bundle.Export`, `serialization.WriteBundle`): requestRefs become bundle names, a spec shared by all scenarios is hoisted, secret values are redacted to `CHANGE_ME`. Round-trip tested (export → import into a fresh workspace).
- **CLI:** `clrsig bundle run|validate|import|export`. `run` executes every scenario straight from the file (`bundle.Materialize` inlines bundle requests; `--env`, `--env-file`, `--spec`, `--scenario`, text/json/junit output; exit 0/1/2), `import`/`export` mirror the desktop dialogs. Shared helpers: `scenario.ResolveRequestRefs` (also fixes `clrsig run` ignoring `requestRef`), `reporting.JUnitMulti`, CLI `specLoader`.
- **Backlog:** a "copy CLI command" button in the desktop dialogs; honour step `retry`/`timeoutMs` in the runner; `bundle run --workspace` coverage accumulation.
- **Help:** the Help panel (wider) has four tabs — User Guide, Format Spec, Example Scenario, AI Prompt — each with Copy and Download (native save dialog) buttons. The Example Scenario is a runnable sample verified against the demo server by a Go test.
- **AI prompt:** `help/ai-prompt.md` holds the instructions (phased workflow, bundle-writing rules, self-check, sample bundle). `HelpService.GetAIPrompt()` assembles instructions + format spec + example scenario at copy time; the old hard-coded Go prompt is removed. A Go test imports and runs the prompt's sample bundle against the demo server.
- **Docs:** format spec corrected to match the code (request `timeout` not `timeoutMs`, `multipart`, `validateSpec`, `retry`/step `timeoutMs` noted as not applied, assertion `name`); user guide sections 7 and 14 updated.

### 2026-11-17 — README: Contracts page

- **README:** new "The Contracts page" section (per-spec cards, operation table, four quality signals, export, how to enable, `clrsig coverage` CLI) and Contracts images in the Screenshots gallery.
- **Screenshots:** `docs/images/10-12-contracts-*.png` (dark) and `*-light.png`, generated by `apps/desktop/e2e/tests/frontend/screenshots-contracts.spec.ts` (`pnpm test -- --grep @screenshot-contracts`). The e2e mock bridge now serves `ListAccumulatedCoverage` / `GetAccumulatedCoverage` / `ResetAccumulatedCoverage`.

### 2026-11-17 — HTML report format

- **Core:** `reporting.HTML`, `HTMLMulti`, `ConformHTML` (optional coverage and negative-probe sections) and `DiffHTML` in `packages/core/reporting/html.go`. One shared document model rendered with `html/template` (all untrusted text escaped); output is a single self-contained file with inline CSS, light/dark theme and print styles. Extracted variable values are deliberately omitted.
- **CLI:** `--format html` for `run`, `conform`, `diff` and `bundle run`; opt-in like `junit`, exit codes unchanged. The Action's `format` input description lists it.
- **Docs:** new "Output formats" section in the user guide (CLI usage); format table in `README-DEVELOPER.md`; one-line mention in `README.md`.
- **Fix:** `--spec` on `run`, `bundle run` and `bundle validate` is now resolved relative to the current directory (it was resolved against the scenario/bundle folder, so `--spec openapi.yaml` silently skipped schema validation and coverage with only a warning). A `specFile:` written inside a scenario or bundle is still relative to that file. The coverage key is unchanged: it stays the path as typed on the command line.

### 2026-11-17 — CLI Command Builder, phase 1 (design: `docs/enhancements/2026-11-17-cli-command-builder.md`)

- **Core:** `packages/core/clispec` (catalog of `run`, `conform`, `diff`: typed inputs, plain-language help, requirements, outputs, exit codes) and `packages/core/clibuild` (`Build` → argv, bash/PowerShell/cmd renderings, messages, needed environment variables, suggested report names, explanation; `DefaultRunFrom`, `SuggestOutput`, `Slug`). Paths are written relative to a run-from folder (default: repository root); files are opened with the same parsers as the CLI. Nothing is executed.
- **Anti-drift:** `TestCatalogMatchesCobra` compares the catalog with the cobra tree in both directions; `TestBuilderCommandsRun` runs builder output through the real command tree for every report format.
- **Desktop:** `CliBuilderService` (catalog, workspace file discovery, build, pickers), `CliBuilderTab` page (singleton `cli` page tab; Activity Rail terminal button; "Get CLI command" button on the scenario header), `isPageTab` helper for page-tab kinds.
- **Formats:** `format` and `output` are offered (text, json, junit, html) with a suggested `reports/<command>-<name>.<ext>`, overwrite/new-folder notes and a warning when html/junit would be printed to the screen.
- **Docs/images:** user guide section "The CLI Command Builder"; `docs/images/13-cli-builder-command*.png` and `14-cli-builder-problems*.png` (generated by `screenshots-cli-builder.spec.ts`).
- **Test debt cleared:** the four Playwright "Run All" tests (stale since Run All results moved from the sidebar to the Runner tab) now assert on the Run Results tab; the whole frontend e2e project passes (135 tests).
- **Not yet (later phases):** "Test it here" (run the command from the desktop), shipping the CLI with the app, `bundle run`/`impact`/`coverage`/`generate`/`request`/`validate`, Docker and GitHub Action targets, multi-step pipeline snippets.

### 2026-11-17 — Run scenarios from folders (design: `docs/enhancements/2026-11-17-run-scenarios-from-folders.md`)

- **CLI:** `clrsig run <file-or-folder>...` runs one or many scenarios. A folder is searched recursively (`--no-recursive` to opt out); `--fail-fast` stops after the first failing scenario. All scenarios are parsed before any request is sent (unreadable file or empty folder → exit 2); one combined report and one exit code for several scenarios; a single file is unchanged. `run` moved to `apps/cli/run.go` (load → execute → report).
- **Core:** `scenario.Discover` (files taken as given; scanned files need a top-level `steps` key, others are skipped and listed; hidden folders and `node_modules` skipped; de-duplicated, sorted). `reporting.MergeCoverage`, `CoverageText`, `TextMulti`, `JSONMulti`; `HTMLMulti` accepts coverage. `TextWithCoverage` and `JSONWithCoverage` now share those helpers (output unchanged).
- **Builder:** `run`'s input is "Scenario or folder" (`clispec.KindPath`) with `--fail-fast` and `--no-recursive`; `clibuild` previews what the CLI will find (count, skipped files, each broken scenario), the service lists workspace scenario folders, the page has separate "browse file" and "browse folder" buttons.
- **Docs:** user guide (CLI usage "Running many scenarios", builder, output formats), `README-DEVELOPER.md`, Action input description.

### 2026-11-17 — CI and parity batch (design: `docs/enhancements/2026-11-17-ci-and-parity-batch.md`)

- **Fix — `${NAME}` in environment files was never expanded** (documented in the guide and `CLAUDE.md` §7, absent in code). `variables.NewResolver` now expands it from the process environment when a run starts (never at parse/save time, so secrets are not written to disk); `$${NAME}` is literal; unset references are left as written and `run`, `conform`, `request` and `bundle run` warn once per variable.
- **Redaction (behaviour change):** secret environment values are replaced by `***` in every report (text, json, junit, html, markdown); extracted values whose names look secret (`token`, `password`, `apiKey`, `session`, …) are masked; `--show-extracted` (run, bundle run) shows those again but never secret environment values. Extracted values are now listed in name order. `scenario.Result` gains `Secrets` and `RevealExtracted`.
- **CLI:** `--report format=file` (repeatable) on run, conform, diff, bundle run; `--var name=value` (repeatable) on run, conform, bundle run; `--format markdown`; unknown `--format` is an error before anything runs; `diff --fail-on breaking|warning|info` and `diff --against <git-ref>`; `validate` and `impact` accept folders (`--no-recursive`); `clrsig import postman|insomnia`; `coverage list|quality --format csv|markdown|html --output`. `diff` moved to `diff.go`; shared report planning in `reports.go`.
- **Core:** `reporting.Markdown*`, `Contracts*` renderers (CSV, Markdown, HTML; HTML escapes all text), `openapi.FailOn`, `importer.Save` (shared by the CLI and the desktop import), `clispec.Input.Repeatable`.
- **Desktop:** the Contracts export builds nothing in the frontend any more (`ScenarioService.ExportCoverageReport` renders with the same Go code as the CLI); Run Results tab gets **Export report** (`ExportRunReport`: html, junit, json, markdown, text). "Run folder" already existed.
- **Catalog:** `fail-on` is offered in the builder; `report`, `var` and `against` are in the catalog as deferred (repeatable or single-positional modes the builder cannot express yet).


### 2026-11-17 — Dynamic values and scenario variables, phase 1 (design: `docs/enhancements/2026-11-17-dynamic-values.md`)

- **Syntax:** `{{$function arg key=value}}` next to `{{variable}}`. Functions: `$guid`/`$uuid`, `$randomString/Alpha/Hex/Choice`, `$randomInt/Float/Bool`, `$timestamp/$timestampMs/$isoTimestamp`, `$date/$datetime/$time` (offsets, `format=`, `tz=`). Evaluated by the Go core (`packages/core/variables`: `Engine`, `Check`, `ResolveEnvironment`).
- **Scenario `variables:`** (additive optional field, `schemaVersion` stays "1"; older clrsig rejects bundles containing it): generated once per scenario run, in order, may use functions, environment values and earlier variables; `secret: true` masks in reports.
- **Scopes:** extracted > scenario variables > environment. Environment values with functions are generated once per command (CLI) and shared by all scenarios; inline calls are generated at every occurrence.
- **Assertions:** `expected` (strings), `path` and `name` are now interpolated (previously never were).
- **Seed:** `clrsig run|bundle run --seed N` or `CLRSIG_SEED`; each scenario gets its own stream derived from the seed and its path, so values do not depend on run order. Reports show the seed (text, json, junit property, html, markdown) only when random values were used, plus generated scenario variables.
- **Validation:** unknown function, bad arguments/offsets/time zones, duplicate, self-referencing or out-of-order variables are errors in `clrsig validate`, bundle import and before `clrsig run` sends anything; a scenario that fails to start reports a setup error.
- **Importer:** Postman `$randomUUID`, `$randomBoolean`, `$randomAlphaNumeric` etc. are mapped; unsupported names are left as written and reported as warnings (stderr).
- **Desktop:** scenario `variables` survive visual-editor saves; editing UI, autocomplete and showing the seed in run results are phase 3. The desktop generates environment functions per scenario run.
- **Docs:** format spec, user guide. **Not yet:** `--seed` for `conform`, builder UI for `seed` beyond the catalog entry.

### 2026-11-17 — Dynamic values, phase 2

- **`$calc <expression>`:** own evaluator in `variables/calc.go` (numbers, bare variable names, `+ - * / %`, parentheses, `round floor ceil abs min max`); integers stay integers, floats trimmed, errors for division by zero, unknown or non-numeric variables; `validate` checks syntax without values. No dependency, cannot call anything else.
- **Fake data:** `$randomEmail`, `$randomFirstName`, `$randomLastName`, `$randomFullName`, `$randomPhone`, `$randomWord` from a small embedded set, following the seed. Postman names are mapped by the importer.

### 2026-11-17 — Dynamic values, phase 3 (desktop) and AI prompt

- **Function catalog from the core:** `variables.Functions()` (name, usage, description, example; a test checks every function is documented and its example validates) served by `ScenarioService.ListDynamicFunctions`; the frontend loads it once at startup (`lib/dynamicFunctions.ts`).
- **Autocomplete:** the existing `{{` suggestions (URL field, key/value editors, variable values) now also list `$` functions with their description and insert a working example.
- **Variables section** in the scenario editor (name, value with autocomplete, secret lock, add/remove); saved through the existing YAML serializer.
- **Run result:** `ScenarioResultIO` carries `seed` (only when random values were used), generated `variables` (secret values masked) and the setup error; the scenario view shows them with the `clrsig run --seed N` replay command.
- **AI prompt** now teaches scenario variables, dynamic functions and interpolated assertions (guard test updated).
- Playwright: 2 new tests; the 5 `smoke` tests still need a live `wails dev` server.


### 2026-11-17 — Postman gap review items A–E (design: `docs/enhancements/2026-11-17-postman-gap-review.md`)

- **B — redirect control:** request `followRedirects: false` returns a 3xx as it is (default unchanged); request files, scenario step requests, bundles, desktop request editor ("No redirects") and step editor. Finding (not changed): the response validator treats an undeclared status code as "nothing to validate".
- **A — `retry` and `timeoutMs` are real:** the runner applies step `timeoutMs` (overrides the request timeout) and `retry` (total attempts; re-sent after a network error or 5xx only, never after a failed assertion; max 10 attempts, delay up to 60 s, validated). `StepResult.Attempts` is shown in text, JSON, HTML, Markdown and the desktop; the "not applied" warnings and docs are gone.
- **D — OAuth2:** the client-credentials scenario pattern is documented (user guide, AI prompt) and covered by an end-to-end CLI test; no dedicated auth type.
- **E — client certificates (mTLS):** `executor.TLSFiles` (cert, key, extra CA; paths only, `${NAME}` expanded at run time); Settings panel fields; `--client-cert`, `--client-key`, `--ca-cert` (and `CLRSIG_CLIENT_CERT`, `CLRSIG_CLIENT_KEY`, `CLRSIG_CA_CERT`) on every CLI command that sends requests, validated before anything is sent. Encrypted private keys are not supported.
- **C — conform baseline (backlog 11):** `conform --write-baseline <file>` records violations, `--baseline <file>` fails only on new ones and reports fixed ones. New versioned file format (`schemaVersion: "1"`, documented in `format-spec.md`); violation identity = operation + kind + detail; no timestamps (stable diffs); secrets redacted; summary on stderr so machine formats stay pure.

### 2026-11-17 — Demo workspace: dynamic values

- `demo_workspace/scenarios/4-dynamic-values/` (unique-users, function-playground, calculations, dates-and-offsets), `requests/8-dynamic-values/echo-dynamic.yaml`, and a `runId` environment function in `environments/local.yaml`. A Go test runs every scenario five times (each run generates different values) and sends the request twice to check the values differ.

### 2026-11-17 — Scenario steps show the request and the response

- `StepResultIO` carries the request as sent (`requestMethod`, `requestUrl` with query parameters, `requestHeaders`, `requestBody`), built by `exchangeRequest` with credentials masked (sensitive header/parameter names, the auth block, and every secret value). New `StepExchange` component: blue Request and green Response sections (collapsible; response headers closed by default), shown for the selected step in the scenario editor and for each step in the run log.

### 2026-11-18 — Scenarios screen declutter, phase 1 (design: `docs/enhancements/2026-11-18-scenarios-screen-ux.md`)

- **Setup row:** a pinned item at the top of the step list (UI only; not an entry in `steps:`, no format change). **Definition** tab holds description, spec file and variables; **Result** tab holds the specFile warning, seed, generated variables, setup error and full coverage. The scenario bands that used to sit above the step list are gone, and so is the post-run pass/fail bar.
- **Header:** a compact coverage chip (`Coverage 5/8 · 62%`) next to the PASSED/FAILED badge opens Setup's Result tab; a setup error shows as one alert line under the header.
- **Empty scenarios:** a dashed "Start here" row replaces the "No steps yet" text and creates a real first step on click; the bottom "Add step" button shows only when steps exist; Run is disabled with "Add a step to run" (`scenario.Validate` is unchanged). Empty or last-step-removed scenarios select Setup.
- **Selection:** `ScenarioViewer` selection is `'setup' | number | null`; the Setup row is outside the `role="option"` set so arrow keys, scroll-into-view and Alt+Arrow reorder keep working on step indices; ArrowUp from step 1 selects Setup.
- **Tests:** viewer, step-list and Playwright tests updated, new tests for Setup, Start here, Run disabled and the keyboard path. Not yet: the "Add your first request" prompt (template / saved-request shortcuts) and focusing the new step's label, and regenerated scenario screenshots.

### 2026-11-18 — Scenarios screen declutter, phase 2

- **Step pane:** a `Definition | Result` toggle replaces the "Step Editor" label. Result is disabled until the step has a result and shows a status dot and code. After a run that ends on a failed step the pane switches to Result once; starting a run switches back to Definition.
- **Definition** shows a one-line "Last run: FAILED 404 92 ms 1 assertion failed" above the editor, with View result and Run from here. **Result** shows the result strip and `StepExchange` (request headers now start closed, like response headers).
- **Stale results:** editing a step after a run marks its result stale (tab badge, dimmed Result view, note) until the next run or save.
- **Calmer editor:** Headers, Query Params and Extractions are collapsible sections with a count badge; empty ones start closed and offer their add button in the header. The editor card is keyed by step index so each step opens the sections that have content.
- Tests updated and added (pane toggle, auto-switch, stale, editor sections); Playwright adjusted.

### 2026-11-18 — Scenarios screen declutter, phase 3 (partial)

- `RequestSummary` (in `StepExchange.tsx`) is the read-only "request as sent" view with neutral props (`method`, `url`, `headers`, `body`); `StepExchange` uses it for the step Result tab and the run log.
- **Request screen:** `RequestService.Execute` now returns the request as sent in `ResponseOutput` (`requestMethod`, `requestUrl`, `requestHeaders`, `requestBody`), built by the same `exchangeRequest` that masks scenario steps (sensitive header/parameter names, the auth block and every secret environment value). The response viewer gets a **Request** tab (and shows the request under a network error). It is returned to the UI only: history stores no bodies and the response cache writes its own fields, so the sent request is never persisted. Go tests cover masking and the error path.

### 2026-11-18 — Scenarios screen declutter, phase 4

- **Run-from-here warning:** `lib/runFrom.ts` (`missingOnRunFrom`) finds values extracted by steps before the start point and used by a step that still runs (URL, headers, params, body, assertions, extraction paths); scenario variables and names re-extracted before use are excluded. The step pane shows "Starting here skips earlier steps, so {{token}} is not set." next to Run from here (Definition and Result views). Frontend only.
- Scenario screenshots (`06`, `07` dark and light, `08`) regenerated for the new layout. Fixed a stray `0` after "Advanced" in the step editor (a falsy number rendered by `hasNonDefaultOptions &&`).

### 2026-11-18 — Request and scenario step editors made consistent

- **URL field:** the highlighted, autocompleting URL input is now one shared component (`HighlightedInput`) used by the request screen and the step editor. In a step, `{{` completes scenario variables, values extracted by earlier steps and the selected environment's variables (and `$` functions). Under the URL a line says where each `{{name}}` comes from (`= value`, `← extracted in step 1`) and flags unknown names or names that only a later step extracts (`lib/stepVars.ts`; secrets show `••••`). The environment is the one chosen for the run, read with `getEnvironment`.
- **Step rows:** header and param rows use the same value autocomplete, HTTP header-name suggestions and `Header` / `Parameter` / `Value` placeholders as the request screen.
- **Small alignments:** the step's method selector is coloured like the request screen; one shared `BODY_TYPES` (with labels such as "Form URL Encoded"; multipart stays out of step editors unless a step already uses it); the request tab is now **Query Params**, matching the `queryParams` field and the step editor.
- **Left as is, by design:** timeout placement (URL bar on the request screen, Advanced in a step); no per-row enable checkbox in steps. **Open:** step authentication (a step-level `auth` field is a file-format change; the alternative is default headers in Setup) needs a decision.
- **Step authentication (decided: the editor, not the format):** an inline step `request:` already accepted `auth:` (the file format and runner are unchanged; `schemaVersion` stays "1"), but the visual editor neither showed nor kept it, so saving a scenario in the desktop app silently dropped a step's auth. `StepDetailIO` now carries `auth` (shared `authDomainToIO`), `scenario-yaml.ts` writes it, and the step editor has an **Auth** section using the request screen's `AuthEditor`, with a hint to use `{{variables}}`. Not offered for `requestRef` steps (the referenced request owns its auth). `format-spec.md` now says an inline step request takes every request field including `auth`. Default headers in Setup were considered and left out: that is a new field with merge rules, and per-step auth covers the case without one.
- **Requests and History:** the rail tab **Explorer** is now **Requests**, and **History** left the rail. Both areas now have the same in-panel strip: **Requests | History** and **Scenarios | Run Log** (the choice is remembered in `ui.requestsView`; an old saved `history` rail tab opens Requests). The OS "Open in Explorer" actions are unchanged.
- **Scenarios list without a root node:** the `scenarios` root folder row is gone from the Scenarios panel, matching the Requests panel. Its actions (run all, refresh, template, new folder, new scenario) are a toolbar row at the top, always visible; top-level scenarios and folders start at the left edge. The empty space below the items is the root: dropping a scenario there moves it out of its folder, and right-clicking it opens the root menu (new scenario/folder, paste, run all).
- **Sidebar icon sizes (root cause):** `index.css` sizes every icon inside a `button` or `a` from the font-size setting (`button svg`, ~14px), overriding the `size` attribute. The Requests tree wraps its icons in buttons, so they rendered at 14px; the Scenarios tree uses div rows, so its icons kept their literal 10-12px and the earlier number changes had no visible effect. The rule now also covers `.ui-icon`, which the Scenarios tree puts on its chevron, folder and scenario icons: both trees render every icon at the same scaled size, rows are the same height (22px) and the same glyphs are used for "new folder" (`FolderPlus`) and "new item" (`FilePlus`). Measured in the browser, not read from the code.

### 2026-11-18 — Contracts page totals: four equal tiles

- **Why:** the header was one big coverage percentage with a full-width bar, so a spec at 100% looked healthy even when operations were failing; coverage only says an operation was called.
- **Now:** four equal tiles, each a count with its share of all operations: **Coverage** (called by a scenario), **Passing** (last result passed), **Failing** (last result failed), **Not covered** (no scenario calls them). Passing + Failing = Coverage, and Coverage + Not covered = all operations. Coverage is neutral (no green/amber/red judgement); Passing is green, Failing red and Not covered amber only when above zero, otherwise muted. Scenario run totals (runs, passed, failed, last run) moved to a "Scenario runs" line under the tiles so they are not confused with operation results. Covered operations without a recorded result (older data) get a note. Logic in `operationOutcomes` / `share` (`lib/coverage-report.ts`).
- Contracts screenshots (`10`-`12`, dark and light) regenerated. **Not changed:** the per-spec cards in the Contracts sidebar still show a coloured coverage bar.
- The fourth tile is called **Not covered**, not "Never tested": that wording clashed with the quality signal "Secured — never tested with auth" (an operation can be called, so covered, and still never have been tested with credentials).
- **Order and names of the tiles:** **Covered | Not covered | Passing | Failing**, grouped as two pairs with a gap between them. Covered + Not covered = all operations, Passing + Failing = Covered. "Coverage" became **Covered** so it reads as a count beside "Not covered" and matches the table's sections. Each tile still shows its share of all operations.

### 2026-11-18 — Contracts exports and CLI carry the same four totals

- **One definition:** `coverage.Report.Outcomes()` and `coverage.Share` (`packages/core/coverage/outcomes.go`) classify the operations (Covered + Not covered = Total; Passing + Failing = Covered; `NoResult` for covered operations with no recorded result). The Contracts page (`operationOutcomes`, which now also uses the report's own `covered` count), every export and the CLI use it, so they agree.
- **Exports (`reporting/contracts.go`, shared by the desktop Export menu and `clrsig coverage ... --format csv|markdown|html`):** the single-spec Markdown and HTML open with the four totals (the HTML as four equal tiles in the same two pairs, muted at zero; the big percentage and bar are gone) and a separate "Scenario runs" section; the per-operation CSV gets a trailing `Status` column (existing columns and their values unchanged); the per-spec list CSV gets trailing `Not Covered,Passing,Failing` columns; the per-spec Markdown and HTML tables show all four with their shares.
- **CLI:** `coverage list` and `coverage quality` text show one line `Covered 10 (71%) · Not covered 4 (29%) · Passing 9 (64%) · Failing 1 (7%) · of 14 operations` (`reporting.OutcomesLine`, plus `No result` when present) instead of the ASCII bar, which is removed; JSON gains additive fields (`notCovered`, `passing`, `failing`, `noResult` per spec in `list`; a `totals` object in `quality`); the MCP `coverage` tool prints the same line.
- **Left as is:** the single-run coverage in run reports and the scenario coverage banner (that is one run, not the accumulated tiles), `printCumulativeCoverage` (conform history), and the Contracts sidebar cards.

### 2026-11-18 — Requests and Scenarios trees made alike

- **Compared in the browser, not by eye:** folder names are weight 400, 13px and the same colour in both trees (the scenario folder names had `font-medium`, which made them look bold); the gap between a folder's chevron, icon and name is the same (`gap-1.5`); folder hover actions are in the same order (new item, new subfolder, then delete; scenarios keep their extra "run folder" first).
- **Scenarios gained what requests already had:** the scenario open in the active tab is highlighted (muted background and an accent left border, like the active request file) and a scenario with unsaved edits shows the amber dot. `Sidebar` takes `activeScenarioPath` and `dirtyScenarioPaths`, computed in `App` from the tabs.
- **Requests gained what scenarios already had:** a request folder row now toggles when you click anywhere on it (name or empty space), not only the chevron and folder icon.
- **Not changed:** requests show a method badge and scenarios a step count and play icon (different data); the Delete folder button shows only for empty request folders but always for scenario folders; scenario folder rows are 22px tall and request folder rows 21px.

### 2026-11-19 — Next-steps snapshots

- Created `docs/next_steps/` for point-in-time planning documents (separate from `docs/enhancements/`, which holds feature design proposals). Convention in `docs/next_steps/README.md`: `YYYY-MM-DD-next-steps.md`, new file per snapshot — do not edit old ones.
- First snapshot `2026-11-19-next-steps.md` orders post-MVP work: (1) code signing + notarization (Azure Trusted Signing, Apple Developer Program — release.yml currently produces unsigned binaries), (2) hosted demo server + everything workspace (`examples/server` exists locally; needs hosting and a kitchen-sink workspace wired into release checks), (3) `clrsig conform` spec-conformance command (per `docs/contract-testing-roadmap.md`; needs an enhancements proposal), (4) NotebookLM video guides kept as regenerable script + screenshot artifacts, (5) traction (posts, shorts, forums), (6) backlog: failure-catalog repo, community scaffolding, JUnit output.

### 2026-11-19 — Conformance Server and Dev Servers menu (phase 1.6 of `docs/enhancements/2026-11-19-desktop-conformance.md`)

- **Conformance Server:** `conformance_server.go`, `ConformanceServerService` (Start, Stop, IsRunning, GetPort, OpenSpecInBrowser), embedded `help/conformance-openapi.yaml`. Four operations conform, three break the spec (`getProduct`, `listCategories`, `listOrders`); `TestConformanceServerViolationsAreExactlyTheKnownOnes` pins the set. No workspace is opened; port from 8120; stopped on app shutdown.
- **Menu:** Help → **Dev Servers** holds the Demo Server and Conformance Server entries; status bar shows a `Conformance :port` chip.
- **Finding (not changed):** the response validator ignores a wrong `Content-Type` and missing response headers.
- **Not yet:** the Conformance page itself (phase 2) and the `conform` extraction (phase 1.5).

### 2026-11-19 — Spec conformance signal on scenario runs (phase 1 of `docs/enhancements/2026-11-19-desktop-conformance.md`)

- **Core:** `scenario.StepResult.Spec` (`SpecCheck`: `matched`, `violations` or `not-checked` with a reason: validation turned off, no matching operation in the spec, step did not complete) and `openapi.Spec.HasOperation`. Failed-assertion behaviour is unchanged. Tests cover every state and the no-spec case.
- **Desktop:** `StepResultIO` gains `specState`, `specReason`, `specViolations` (additive); the Setup row's Result tab shows `N of M responses match <spec>`, violations per step and the not-checked steps (`SpecConformance` component). No file format change.
- **Not yet:** the conformance line in the scenario header chip, user guide text and screenshots.

### 2026-11-19 — `conform` orchestration extracted (phase 1.5 of `docs/enhancements/2026-11-19-desktop-conformance.md`)

- **`packages/core/application`:** `RunConformance(ctx, ConformInput) ConformOutput` runs the conformance runner, optional negative probes and coverage, collects the violations and compares them with a baseline (`Failed` already accounts for it); `SplitList`; `DefaultConformRequestTimeoutMs` (10000). The CLI `conform` command and the MCP `conform` tool both call it; reading the spec and environment, the executor, history, rendering, `--write-baseline` and telemetry stay with the caller.
- **One behaviour change:** the MCP `conform` tool used the executor default timeout (30 s); it now uses the shared 10 s default like the CLI. A CLI `--request-timeout 0` now means the shared default instead of the executor default.
- Tests: new `application` tests (violation fails, tag filter, negative probes, baseline known and new); all existing CLI tests pass unchanged.

### 2026-11-19 — Check Conformance page (phase 2a of `docs/enhancements/2026-11-19-desktop-conformance.md`)

- **Core:** `conformance.Options.ReadOnly` and `conformance.IsReadOnlyMethod` (GET/HEAD only; other operations and negative probes are skipped and never sent); `application.ConformInput.ReadOnly`; `openapi.Spec.FirstServerURL`.
- **Desktop:** `ConformanceService` (`GetConformContext`, `InspectConformSpec`, `RunConformance`, `CancelConformance`), a thin wrapper over `application.RunConformance`. Read-only unless `AllowWrites`; a non-local base URL (including one taken from the spec's servers) is refused unless `ConfirmedRemote`, checked in Go as well as in the page. Singleton page tab `conformance` (`ConformanceTab`, Activity Rail shield button) with spec, environment, base URL, write opt-in with the write count, run/cancel, four totals (Covered + Not covered = all operations; Passing + Failing = Covered; skipped and unreachable operations are Not covered), per-operation results with violations, and Get CLI command.
- **Not yet (phase 2b):** options (tags, path prefixes, negative probes, fail fast, timeout), export, coverage accumulation into the Contracts store, progress events, screenshots and README line.

### 2026-11-19 — Check Conformance page, phase 2b

- **Options:** tags, path prefixes, request timeout (empty = shared 10 s default), negative probes (probe list with rejected / accepted / error / skipped), stop after the first failure. Read-only still applies to probes.
- **Export:** `ConformanceService.ExportConformReport` renders the latest run with `reporting.ConformHTML/JUnit/JSON/Markdown`, the CLI's renderers; the page saves or opens it like the Run Results export.
- **Coverage recording (opt-in, off by default):** adds the run to the Contracts store only when no tag or path filter is set (also enforced in Go); the spec is recorded under its workspace-relative path and the page says so. Scenario and conformance runs share `accumulateCoverage` (`coverage_store.go`). Operations skipped or without a response are not counted. If scenarios name the same spec with a different path string, Contracts shows a separate card for it (the store is keyed by that string).
- **Not done:** progress events during long runs, environment `{{$function}}` generation for these runs (CLI parity), screenshots and a README line, Playwright test for the page, phase 3 baselines (deferred in the proposal).

### 2026-11-19 — Conformance coverage uses the scenarios' spec name

- Scenarios write `specFile` as a path relative to their folder (or a URL), and the Contracts store is keyed by that exact string. Recording from the Check Conformance page now looks through the workspace's scenarios for one whose file-path `specFile` resolves to the chosen spec and reuses its spelling, so both feed one Contracts card; with no match it falls back to the workspace-relative path. URL specs (such as the demo's) cannot be matched from a file chosen in the page; the page names the entry it wrote. Test: `TestScenarioSpecRef_ReusesTheNameAScenarioUses`.

### 2026-11-19 — Conformance Server verified live; path-level parameters; configurable Dev Server ports

- **Live check:** `clrsig conform` against the running Conformance Server with its environment reports exactly three failures, each for its intended reason (`/categories` missing `name`, `/orders` `id` as a string, `/products/{id}` `price` as a string). The first live run showed 5 failures and exposed two problems, below.
- **Fix — path-level parameters were ignored:** `openapi.GenerateRequests` and the negative probes read only operation-level `parameters`, so a parameter declared once on the path item (the usual place for `{id}`) was never turned into a `{{id}}` placeholder: the request went out with a literal `{id}`. `withPathParameters` merges them (an operation's own parameter replaces a shared one with the same name and location). This also changes generated and imported requests for specs written that way (they now carry the placeholder). Reports show such operations with the placeholder path, so a conformance baseline recorded before this fix for such a spec will list them under the old path.
- **Conformance Server:** the spec's POST body now requires only `name` (the generated body stub makes every field a string placeholder, which cannot satisfy a numeric field); DELETE declares 404; `help/conformance-environment.yaml` (`id`, `name`) is shipped, and the server serves `/openapi.yaml` and `/environment.yaml`. The service test now asserts the cause of each violation, not only which operations fail (it had passed partly by accident).
- **Ports:** Settings → **Dev servers** sets a fixed port for the Demo Server and the Conformance Server (`demoServerPort`, `conformanceServerPort`; 0 = first free port from 8090 / 8120, valid range 1024-65535, the two must differ). A fixed port that is in use is an error shown in the app, never silently replaced. `choosePort`, `sanitizePort`; the Demo Server start error is now shown too.
- **Not done:** the Check Conformance page cannot pick the Conformance Server's spec until the user saves it into the workspace (the guide says how); a one-click "open it in a workspace" like the Demo Server's is not built.
