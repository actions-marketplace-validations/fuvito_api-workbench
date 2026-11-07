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
