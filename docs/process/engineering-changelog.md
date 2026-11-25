# Engineering changelog

Dated entries for every significant change to Clear Signal, oldest first; add new entries at the end. Each says what changed and why, and names the tests that guard it. Entries up to 2026-11-24 were kept in section 10 of `CLAUDE.md`, which now only points here so the file every working session loads stays about the rules.

User-facing release notes are in the About dialog (`AboutDialog.tsx`) and on the GitHub Releases; design decisions are in `docs/enhancements/`.

### 2026-11-03
- CLAUDE.md created from `docs/product/Project.md` and `docs/product/Product Requirements, Architecture & Implementation Roadmap.md`.
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

- Created `docs/planning/next_steps/` for point-in-time planning documents (separate from `docs/enhancements/`, which holds feature design proposals). Convention in `docs/planning/next_steps/README.md`: `YYYY-MM-DD-next-steps.md`, new file per snapshot — do not edit old ones.
- First snapshot `2026-11-19-next-steps.md` orders post-MVP work: (1) code signing + notarization (Azure Trusted Signing, Apple Developer Program — release.yml currently produces unsigned binaries), (2) hosted demo server + everything workspace (`examples/server` exists locally; needs hosting and a kitchen-sink workspace wired into release checks), (3) `clrsig conform` spec-conformance command (per `docs/archive/contract-testing-roadmap.md`; needs an enhancements proposal), (4) NotebookLM video guides kept as regenerable script + screenshot artifacts, (5) traction (posts, shorts, forums), (6) backlog: failure-catalog repo, community scaffolding, JUnit output.

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

### 2026-11-19 — Open Conformance Workspace

- **Help → Dev Servers → Open Conformance Workspace** (`ConformanceServerService.OpenWorkspace`) starts the server and opens a temporary workspace (`clrsig-conformance-*`) with `specs/conformance-openapi.yaml` and an active `environments/local.yaml` (`id`, `name`, `baseUrl`), so the Check Conformance page runs with no setup; the Go test runs the page's service inside it and gets exactly the three known violations. Stopping the server (menu or status chip) removes the workspace, reopens the previous one and closes the tabs that belonged to it.
- **Guard:** only one temporary workspace at a time; each of the two Dev Servers refuses to open its workspace while the other's is open (the swap and restore are not nestable). The plain Start/Stop of the Conformance Server still opens no workspace.
- `lib/tabState.withoutTabsIn` is the shared "close the tabs of a temporary workspace" helper, now used by the Demo Server stop as well.

### 2026-11-19 — Check Conformance: unset `${NAME}` warnings

- **Correction:** the earlier note that the page lacked CLI parity for environment functions was wrong for `conform`: the CLI does not pre-generate environment functions for it (only scenario runs do, via `ResolveEnvironment`); `conform` applies `--var` overrides and warns about unset `${NAME}` references. The page already behaves the same for functions (the resolver generates them at each use).
- **The real gap, now closed:** the page left an unset `${NAME}` as written without saying so, so requests failed for the wrong reason. `variables.EnvRef.Message()` (shared with the CLI's warning text, unchanged) feeds `ConformRunOutputIO.Warnings`, shown above the results. Test: `TestConformanceService_WarnsAboutUnsetEnvironmentReferences`; the CLI warning tests pass unchanged.
- **Still not offered on the page:** `--var` overrides.

### 2026-11-19 — Conformance test gaps closed

- **Parity:** `TestConformanceParity_ServiceMatchesTheCLI` builds the real CLI and compares `clrsig conform --format json` with `ConformanceService` operation by operation (method, path, outcome, status), each against its own fresh server (the Conformance Server keeps state: its DELETE removes product 1). It also pins the CLI-level claim: exactly `listCategories`, `listOrders`, `getProduct` fail and the exit code is 1. Skipped with `go test -short`.
- **Cancel:** `TestConformanceService_CancelStopsARunInProgress` (a run waiting on a slow server stops at once on `CancelConformance`, counts nothing as covered, and the service runs again afterwards).
- **Playwright (170 tests, all pass):** `conformance.spec.ts` (read-only run with totals and violations; write opt-in and write count; remote base URL confirmation; negative probes; Get CLI command) and two scenario tests (the spec conformance summary with violations; no section without a spec). The mock bridge's `ConformanceService` now behaves like the real one (read-only skips writes, remote needs confirmation) and its CLI catalog has `conform`. The negative probe list is now a labelled `region`.
- **Still untested:** the UI in a running desktop app (`wails dev`), and the 5 `smoke` tests need that server.

### 2026-11-19 — README and screenshots for conformance

- **README:** a "Check conformance from the desktop" section (safe-by-default reads, the four totals, options and export, scenario spec conformance, the Conformance Server), a feature bullet and two gallery entries. Product pitch only; developer detail stays in the guide and this file.
- **Screenshots:** `15-conformance-results` and `16-scenario-spec-conformance` (dark and light), generated by `screenshots-conformance.spec.ts` (`pnpm test -- --grep @screenshot-conformance`) from data that mirrors the Conformance Server. The other existing screenshots were not regenerated, so they do not show the new shield button in the rail.

### 2026-11-19 — Check Conformance: summary panel and failure-first report (desktop page only)

- **Layout:** two columns. The right-hand **Summary** panel holds **Export report** (disabled until a run) and the four totals as a 2×2 grid; the left holds the form and the report.
- **Report:** a headline (`3 of 7 operations break the spec`) with chips for why they fail, then **Problems** (a card per failing operation: expected, got, why, fix, and the validator's message under *Technical detail*), **Passing** (open) and **Not run** (collapsed, with the reason). Without findings (older data) a card falls back to the raw messages.
- **Core:** `conformance.Diagnose(OperationResult) []Finding` classifies the validator's free-text messages (missing field, wrong type, invalid generated request, no answer, other; JSON pointers shown as `[0].id`). Unrecognised messages become `other` with the raw text, so a validator wording change degrades to less explanation, not to a failure. `ConformanceService` exposes them as `findings` on each operation.
- **Scope:** only the desktop page uses `Diagnose`. The CLI text output and the exported HTML, JUnit, JSON and Markdown reports are unchanged.
- **Known gap:** the page's **Get CLI command** opens the builder with `conform` selected but does not carry the page's choices over, and `clrsig conform` has no read-only option, so a read-only page run cannot be reproduced by a command today.

### 2026-11-19 — `clrsig conform --read-only`; Get CLI command carries the page's choices

- **CLI:** `clrsig conform --read-only` sends only GET and HEAD (other operations and their negative probes are skipped and never sent). Default off, so existing commands and CI behave as before. It is in the command catalog (`clispec`, Essential group), so the CLI Command Builder offers it; `TestCatalogMatchesCobra` guards the pairing. Same `ReadOnly` option the desktop page already used.
- **Desktop:** **Get CLI command** on the Check Conformance page now opens the builder with the page's choices filled in (`cliValues`: spec, base URL, environment, tags, paths, negative probes, fail fast, timeout, and `read-only` unless writes were ticked). "Active environment" is expressed as its file (`ConformContext.activeEnvironment`). Not carried over: "Add to Contracts coverage" (no flag; `conform` always records to history) and the page's own coverage option.
- **Tests:** `TestConformReadOnlySendsNoWrites`, `TestBuilderReadOnlyConformCommand` (the command the builder produces for a read-only choice runs and sends no writes), `cliValues` and button tests, two Playwright tests.

### 2026-11-19 — Project backlog

- **`docs/planning/backlog.md`** is the one living list of open work (checkbox, ID `B-###`, size, what it needs, source). It links to `docs/planning/next_steps/` (point-in-time plans, never edited to stay current) and `docs/enhancements/` (one proposal per major feature) instead of copying them. Tick items there when they ship and add new ones with the next free ID.

### 2026-11-19 — Backlog items B-001, B-004, B-005, B-011 (conformance)

- **B-001 explanations everywhere:** `reporting` redacts and renders `conformance.Diagnose` findings: `ConformJSON` gains an additive `findings` array per operation, `ConformHTML`/`ConformMarkdown` and `ConformJUnit` add one `→ … Why: … Fix: …` line per failure (JUnit's failure `message` still counts violations). The screen output stays as it was unless `clrsig conform --explain` (new, opt-in, in the command catalog) is given: scripts and CI logs rely on it. `reporting.ConformTextExplained`; `ConformText` is unchanged.
- **B-004 progress:** `conformance.Options.OnOperation` / `application.ConformInput.Progress` / Wails event `conformance:progress` (`ConformProgressEvent`), shown on the page as a progress bar and the current operation. Negative probes report no progress yet.
- **B-005 variable overrides:** `variables.WithOverrides` (moved out of the CLI, which keeps `applyVars` as a wrapper with the same `--var …` error text); `ConformRunInput.Vars`; a "Variable overrides" box in the page's Options (this run only, never saved, not in Get CLI command).
- **B-011 sample workspace content:** the Conformance workspace now has `scenarios/product-lifecycle.yaml` and four requests.
- **Finding, not changed (B-009):** a URL with an unresolved `{{placeholder}}` is still sent and a declared 404 passes, so such an operation shows as Passing without being compared. Recorded in the backlog for a decision.

### 2026-11-20 — Conformance Server starts like the Demo Server

- Help → Dev Servers → **Start Conformance Server** is now the single entry: it starts the server and opens its temporary workspace (spec, local environment, sample scenario and requests), exactly as **Start Demo Server** does; the same item becomes **Stop Conformance Server** (the status-bar chip stops it too), and stopping closes the workspace and reopens the previous one. The separate "Open Conformance Workspace" item is gone; there was never a product reason for two (it was added after a plain start-only item). It is disabled, with the reason, while the Demo Server's workspace is open, and `requestWorkspaceSwitch` asks about unsaved tabs first, as for the demo.
- `ConformanceServerService.Start` (server only) is still there for tests and API users but the app no longer calls it; `startConformanceServer` was removed from `bridge.ts`.
- Test: a Playwright test drives Help → Dev Servers → Start Conformance Server (page opens, status chip, Stop entry).

### 2026-11-20 — B-009: operations with a missing placeholder value are not sent

- **Behaviour change (CLI and desktop):** `conformance.Runner` no longer sends a request that still contains a `{{placeholder}}` after variables were filled in (URL, query, headers, auth, body), or whose dynamic value failed. The operation gets `Blocked` (reason, `not sent: no value for {{id}}`) and `Missing` (names); `Passed()` is false and `Counts()` counts it as failed, so the run exits 1 and it can never show as passing; coverage treats it as not covered. Before, `/items/{{id}}` was sent literally and a declared 404 counted as a match.
- **Unused variables:** a variable that is defined but never used is ignored and not reported (environments are shared by many operations; the existing warning for an unset `${NAME}` reference stays). Only a *used and missing* value blocks.
- **Reporting:** one aggregated line under the report names every missing value once (`conformance.MissingValuesAdvice`, `ConformResult.BlockedSummary`); JSON gains `blocked`/`missing` per operation and `missingValues`; `Diagnose` gives a `missing-value` finding. The desktop page shows NOT SENT cards, a headline that separates "break the spec" from "could not be checked", and a **Missing values** box whose button adds the names to Variable overrides.
- **Risk:** CI that ran `conform` without supplying every value now fails (the repo's own `examples/environments/demo-conform.yaml` supplies them all, and the example-server CI step still passes). Secured operations need `token`; generated bodies need a value per property (see backlog B-003).

### 2026-11-20 — B-019: Run folder (requests) can be stopped

- Every run in the desktop can now be stopped: a request (Cancel), a scenario (Cancel), Run All / Run folder for scenarios (Stop on the Run Results tab), the Check Conformance page (Cancel) and now **Run all requests in a folder**: `ScenarioService.CancelFolderRun` cancels the context of `RunFolder`, the request in flight is dropped (it is not a result), later requests never start, and the finished results are returned; the folder results show **Stop** while running and "stopped" after. `FolderRunState.stopped`; `cancelFolderRun` in `bridge.ts`.

### 2026-11-20 — Conformance accuracy, part A (B-002; `docs/enhancements/2026-11-20-conformance-accuracy.md`)

- **Bug fixed (behaviour change, CLI, scenarios and desktop):** `openapi.ValidateResponseByPath` forwarded only `Content-Type` to kin-openapi and dropped `ResponseError.Reason` when `Err` was nil. So a response whose operation declared *any* response header never had its body checked, and an undeclared status, a wrong Content-Type or a missing required header passed silently. Now every real response header is forwarded and every `Reason` is reported. New violation kinds on `openapi.ValidationError.Kind` (`undeclared-status`, `wrong-content-type`, `missing-header`, `bad-header`, plus `body`), with `Subject` (status, content type or header name). A `default` response counts as declared; `validateSpec: false` per step still skips. Only the first problem is reported (header problems come before body problems). The `type: schema` assertion by operationId (`ValidateResponse`) is unchanged.
- **What / why / where / how to fix (owner rule, saved as a memory note):** every message names the operation, status and thing that is wrong (`GET /a 200: the response header "X-Total-Count" is required by the spec but was not sent`), and `conformance.Finding` gained `Part` (response body, response status, response header, request, network), `Where` (field path, header name, status) and `Spec` (the place in the spec, for example `GET /a → response 200 → headers → X-Total-Count`) next to summary, expected, got, why, fix and the raw detail. `OperationResult.SpecPath` carries the spec's own path template. The explanation line in the text (`--explain`), HTML, Markdown and JUnit reports now reads `Summary. Why: … Where: … (spec: …). Fix: …`; JSON findings gain `part` and `spec`; the page shows Where and In the spec rows.
- **Conformance Server:** three more deliberate defects, six in all (`GET /widgets` text/plain, `GET /inventory` missing `X-Total-Count`, `GET /status` answers 503), ten operations (8 reads, 2 writes). Tests, the CLI parity test, docs and the screenshot data (captured from a real run) updated.
- **Risk:** CI that was silently green because of the bug can turn red. Adopt gradually with `clrsig conform --write-baseline` / `--baseline`. The repo's own CI steps (`conform` and `run` against the example server, with and without `--spec`) still pass.
- **Part B (generated request bodies, B-003) is accepted but not built.**

### 2026-11-20 — Conformance accuracy, part B (B-003; `docs/enhancements/2026-11-20-conformance-accuracy.md`)

- **One shared request body generator** (`packages/core/openapi/bodystub.go`, replacing the OpenAPI 3.x and Swagger 2.0 ones): only **required** properties (plus optional ones with an `example` or `default`), `readOnly` properties left out, a property with an example, default or enum uses that value, any other property is a **typed** placeholder (`"{{title}}"` for a string, bare `{{price}}` for number, boolean, array and object so a filled-in value is valid JSON), `allOf` merged (also through `$ref` in Swagger 2.0), first `oneOf`/`anyOf` branch, array bodies as an array of the item template, nested structures one placeholder. Negative probes are unchanged (they build their own typed bodies).
- **Consequence for imports:** imported request files now contain fewer fields (a schema with nothing required imports as `{}`); the optional fields are no longer visible as a scaffold. Backlog B-047 tracks offering a full scaffold at import time.
- **Request-side validator fixed like the response side:** `ValidateRequest` dropped the parameter name, the "request body" context and the `Reason` (and reported nothing for a `RequestError` without an inner error). New kinds `request-parameter` (with `Subject` and `In`) and `request-body`; messages name the operation, the parameter and where. `conformance.Diagnose` explains request problems with what / expected / got / where / spec location / why / fix and names the environment value to set.
- **Format JSON** (request body editor and scenario step editor) understands bare `{{placeholders}}` (`lib/jsonFormat.ts`) and now says when the body is not JSON instead of silently doing nothing.
- **Tests:** the generator (each type, required-only, literals, allOf, oneOf, arrays, Swagger 2.0), "a filled-in generated request passes `ValidateRequest`", request findings end to end, a CLI end-to-end test where the server decodes real numbers, booleans and arrays, and the Format JSON helper. Three older stub tests were updated to the new rules.

### 2026-11-20 — Catalog of what we check (B-048; `docs/what-we-check.md`)

- **`docs/what-we-check.md`** lists every failure, warning and "not checked" outcome with a provisional ID (`CS-RESP`, `CS-REQ`, `CS-RUN`, `CS-PROBE`, `CS-COV`, `CS-DIFF`, `CS-IMP`, `CS-NC`, `CS-GAP`), title, description, example, state (caught / partly / not caught / not checked), severity today and evidence, and the known gaps with the reason. The response checks are listed for the two checkers separately (A: step spec check and `conform`; B: the `schema` assertion, which only compares the body). Terms: API Contract Validation (section 1) and Spec Conformance Testing (section 2).
- **Evidence is enforced:** `packages/core/openapi/what_we_check_test.go` pins current behaviour including gaps (`KnownGap`), and `packages/core/conformance/what_we_check_doc_test.go` fails if the catalog cites a test that does not exist, defines an ID twice, or if the docs or any test comment mention a `CS-` ID the catalog does not define.
- **Rule (owner):** every new failure, gap, false pass or false failure we learn about is added to the catalog in the same change; fixing a gap moves its row and flips its `KnownGap` test.
- **Found while measuring (now in the backlog):** `impact` never matches a step whose URL path contains a placeholder (B-051, a bug); a declared `application/xml` response is reported as a violation (B-052); `format: email` / `uuid` are not validated (B-053); the `schema` assertion ignores status, Content-Type, headers and `writeOnly`, and does not check an empty body.
- `docs/archive/contract-testing-roadmap.md` is retired with a pointer (B-030); the README and the user guide link the catalog.

### 2026-11-20 — The catalog is shown in the app (Help, What We Check)

- The catalog moved from `docs/what-we-check.md` to **`apps/desktop/wails/help/what-we-check.md`** (embedded files must live under the Go package, and `help/` is where every document the app serves already lives: user guide, format spec, example scenario, AI prompt). It is still the file the README links to and the one the guard tests (`what_we_check_doc_test.go`) read; the 2026-11-20 entry above that says `docs/what-we-check.md` is history.
- **Help panel:** a fifth tab, **What We Check** (menu Help, What We Check…), loaded on demand through `HelpService.GetWhatWeCheck`, with Download and Copy; the panel is wider on that tab for the seven-column tables, and a check ID (`CS-RESP-008`) never wraps so it can be quoted.
- Tests: Go guard tests on the new path, component tests for the tab (lazy load, table, copy and download, load failure, ID not wrapping), a Playwright flow through the Help menu.
- Next step for B-049: turn the tab into a searchable table by ID when the catalog becomes data in Go; today it renders the Markdown.
- **Help panel width:** the panel is now 90% of the window wide (capped at 1800 px) on every tab, instead of a fixed width, so guides, specs and the catalog's wide tables use the room there is (`help-panel.spec.ts` measures it at several window sizes).

### 2026-11-20 — `docs/` reorganized (backlog B-054; map in `docs/README.md`)

- **Two places.** Documents the **app serves** stay in `apps/desktop/wails/help/` (the Go embed requires it): user guide, format spec, example scenario, AI prompt, `what-we-check.md`, the sample specs. **Developer documents** live in `docs/`: `enhancements/` (design proposals, path unchanged), `planning/` (`backlog.md`, `next_steps/`, product requirement documents), `reports/` (`check-YYYY-MM-DD.md` audits, comparisons), `guides/` (CI/CD integration, authenticated conformance runs), `process/` (release process, manual steps), `product/` (founding notes, requirements and roadmap, presentation), `archive/` (retired). `docs/images/`, `docs/demo.gif` and `docs/demo.tape` stay where they are: the README, the demo workflow and the screenshot tests use those paths.
- **Rules now:** the backlog is `docs/planning/backlog.md`; next-step snapshots go in `docs/planning/next_steps/`; audit files are `docs/reports/check-YYYY-MM-DD.md` (a new file per date, old ones kept); a user-facing reference the app should show goes in `help/` with a Help tab; `docs/README.md` says where a new document goes.
- **Moves:** 19 files with `git mv`, the dead `docs/UserGuide.md` stub deleted, `ci-integration.md` and `cicd-integration.md` merged into one rewritten `docs/guides/ci-integration.md` (it fixes advice that was wrong: environment files use `${NAME}`, not `{{NAME}}`, for process variables; it adds the current flags, the Action's inputs, and a section on running conformance in CI). Relative links and plain-text mentions were rewritten; a link check over every markdown file finds none broken. Older changelog entries above still name the old paths; they are history.

### 2026-11-20 — Always start on the landing page

- **Behaviour change (owner request):** the app no longer reopens the last workspace or active environment on start or restart; it opens on the landing page. The last workspaces stay in the Recent list only.
- **No residue:** at startup stale `clrsig-demo-*` and `clrsig-conformance-*` temp workspaces (left by a crash or kill) are deleted; temp workspaces are never added to the Recent list, and Recent entries for folders that no longer exist are dropped. Tests: `TestWorkspaceConfig_StartsOnLandingPage`, `TestWorkspaceConfig_TempWorkspacesLeaveNoResidue`.

### 2026-11-20 — Environment remembered per workspace

- The selected environment is stored per workspace (`workspaceEnvs` in `~/.clrsig/config.json`, path relative to the workspace, including "no environment") and restored when that workspace is opened (`enterWorkspace`); a workspace no longer inherits the previous one's environment. Temporary demo and conformance workspaces are not remembered. File → Close Workspace (already present) returns to the landing page. Test: `TestWorkspaceEnvironment_RememberedPerWorkspace`. Dashboard and open/close reminders: backlog B-055.

### 2026-11-21 — 0.6.0 release batch (from `docs/reports/check-2026-11-20.md`)

- **Fixed:** `clrsig impact` matches steps whose URL path contains a placeholder (B-051); required query and header parameters are generated as the spec's example/default/first enum value or a `{{name}}` placeholder, so a missing value is "not sent" (B-056, OpenAPI 3.x and Swagger 2.0); a declared media type with no decoder (XML, CSV, ...) is "not checked" (CS-NC-009) instead of a violation, in both response checkers (B-052); the Check Conformance page carries the run verdict (`failed`), its headline separates break the spec / got no answer / could not be checked and says when nothing was reached, and skipped operations get their own line (B-057, B-066); the CLI conform text report prints the spec path (B-060) and one "No operation got an answer" block for a dead server; a second `RunConformance` is refused while one is active (B-062); `gofmt -w` over every Go file plus a "Check formatting" step in CI (B-059).
- **Catalog:** CS-GAP-007 removed (fixed), CS-RESP-012 and CS-GAP-004 now "not checked", CS-NC-009 added.
- **Backlog:** the remaining findings of the review are B-061, B-063, B-064, B-065 (hardening) and B-067 to B-070 (UI); B-058 (Action inputs) is still open.
- **Release:** version 0.6.0 in `AboutDialog.tsx` (with release notes) and the user guide.

### 2026-11-21 — Workspace page (B-055; design: `docs/enhancements/2026-11-21-workspace-dashboard.md`)

- **Core:** `packages/core/workspaceinfo` (`Inspect(root, env, envPath) Report`): counts, environment variables as set/unset (never a value), the specs scenarios name (found, missing, outside, URL; URLs are never fetched) and attention items written as what / where / why / fix: no environment while scenarios need one, unset `${NAME}`, unparsable or invalid scenario, missing `requestRef`, missing or outside spec. Uses the existing parsers and `scenario.Validate`; reads only.
- **Desktop:** `DashboardService.GetWorkspaceOverview` adds the environment source (`remembered`, `chosen`, `temporary`, `none`; tracked in `WorkspaceService`, never persisted), stored Contracts totals per spec, the last scenario run from the on-disk run log, and a "spec-failing" item. `WorkspaceService.GetCloseReminder` feeds one extra line in the unsaved-tabs confirmation.
- **UI:** singleton `workspace` page tab (`WorkspaceTab`), first item in the Activity Rail, opened by `onWorkspaceOpened` when a workspace is opened (the Conformance Server then focuses Check Conformance). Opening summary line is dismissible. No file format or config change; informs, blocks nothing.
- **Docs:** user guide section "The Workspace page" (and the outdated "last workspace is remembered" sentence corrected). **Not done:** screenshots; `clrsig validate` printing the same items (B-071); dismissing a warning per workspace (deliberately left out).

### 2026-11-21 — Demo workspace works with Check Conformance; sidebar scrollbar fix

- **Demo Server:** its scenarios name the spec by URL, but the Check Conformance page lists spec *files* in the workspace, so the page's spec list was empty for the Demo Server (the Conformance Server ships a file). `extractDemoWorkspace` now also writes `specs/openapi.yaml` (the embedded demo spec with its `servers` URL set to the port the server runs on), and the demo `local` environment gains `id`, `code` and `token`, which the generated requests use as placeholders. Result: a read-only check passes all 15 read operations with nothing not sent; the 9 write operations are skipped. The demo scenarios now name that file (`../../specs/openapi.yaml`) instead of the URL, so a scenario run and a conformance run recorded to Contracts feed **one** card (with the URL spelling they made two cards named `openapi.yaml`, which was confusing). Tests: `TestDemoWorkspace_CheckConformancePageListsAndRunsTheSpec`, `TestDemoWorkspace_ScenarioAndConformanceRunsShareOneContractsCard`. A URL `specFile` is still supported and documented; B-010 remains for workspaces whose scenarios use one.
- **Scenarios panel:** the root drop zone was `min-h-full` below the toolbar and filter rows, so the list always overflowed by their height and showed a scrollbar; it now fills the remaining height. Regression tests in `sidebar-scroll.spec.ts`.
- **Contracts sidebar:** every card shows the spec's location (the exact `specFile` string: a URL or the path the scenarios wrote) under its name, so two specs with the same file name can be told apart. Contracts screenshots (`10`-`12`) regenerated.
- **Fix — Contracts could not open a spec named by a relative path:** the Contracts card is keyed by the `specFile` string as the scenarios wrote it, which is relative to the scenario's folder, but opening the card and exporting its report loaded it with no folder, so it was looked up from the app's working directory (`loading spec: open ..\..\specs\openapi.yaml: The system cannot find the path specified.`). It only worked for URLs and absolute paths. `loadWorkspaceSpec` now tries the folder of each scenario that names the spec, then the workspace root, and says what it tried when the file is missing. This affects every workspace with a relative `specFile`, not only the Demo Server. Tests: `workspace_spec_test.go`, and the demo Contracts test opens and exports the card.
- **Fix — Run All ran the folders:** the scenario list holds a placeholder entry per folder (so empty folders can be drawn), and Run All iterated every entry, so each folder was "run" as a scenario: a failed row with no name (`read ...\scenarios-full-coverage: Incorrect function.`) and an empty scenario tab when it was clicked, and the total counted folders. Run All now runs only scenarios (Run Folder, Quick Open and the sidebar already filtered). Playwright test: `Run All runs only scenarios, not the folders that hold them`.

### 2026-11-21 — Pre-release check items (`docs/reports/check-2026-11-21.md`)

- **Release artefacts:** the Docker image now reports its version (`ARG VERSION`, `-X main.version`, `build-args` in `release.yml`; verified by a local build); the release workflow pins `wails@v2.17.1` in its three jobs (B-072, B-073).
- **Workspace page:** a "the environment does not define {{x}}" card (placeholder use is read from parsed, sent fields only, including the request files a step points at; notes and labels do not count); coverage looked up by the Contracts store's own key and shown once per stored card; rows keyed by resolved path; the page reads again whenever its tab becomes active (B-074, B-075, B-076). Sample workspaces are pinned to need no attention.
- **Hardening:** `DeleteFolder` refuses without a workspace, the workspace folder itself and any folder holding a hidden folder; all 14 `HasPrefix(rel, "..")` guards use `outsideRel` (B-077).
- **Backlog:** B-078 (symlinks), B-079 (reminder returns), B-080 (Contracts store key collisions, found on the way) are open and none blocks 0.6.0.
- **Rail consistency (owner request):** the Workspace page is a full-width page, not a side panel. Showing it (its icon, or opening a workspace) closes the sidebar and lights only its icon; a panel icon (Requests, Scenarios, Contracts) reopens the sidebar and takes the highlight, so the rail never lights a page and a panel at once. The sidebar has `data-testid="sidebar"`. Conformance and CLI Builder still follow the older behaviour (B-068).
- **Rail follows the tabs (owner request; part of B-068):** `lib/railFocus.ts` maps a tab kind to its panel (request to Requests, scenario and runner to Scenarios, coverage to Contracts) or full-width page (workspace, conformance, cli). Switching tabs points the sidebar at the tab's panel and closes it for a page tab; coming back from a page tab to a panel tab reopens it (the page closed it, not the user), while between panel tabs open or closed is kept; the lit icon is the open panel, or with the sidebar closed the icon of the active tab. Check Conformance and CLI Builder icons now light too. A rail click wins until the next tab switch; the first render is skipped so a saved panel is not overwritten. Screenshots regenerated.
- **Tab order on opening a workspace (owner request):** the Workspace tab is created leftmost (`openWorkspaceTab` inserts it at the front), the first request opens after it, and the Workspace tab stays active. Page tabs (Workspace, Run Results, ...) outlive a workspace, so "open the first request" now ignores them when it checks that the user has no work open; before, a surviving page tab would have stopped the first request from opening in the next workspace. Tests: hook tests for both, and a Playwright test of the whole order.

### 2026-11-21 — v0.6.0 released

- **Tag `v0.6.0` on `88569eb`** (CI green on that exact commit). Release workflow green; GitHub Release (13 files), Docker `ghcr.io/fuvito/api-workbench:v0.6.0` (prints `clrsig version v0.6.0`), Homebrew and Scoop bumped by the `chore: bump tap to v0.6.0` commit.
- **Docs for the release:** README (Workspace page section and screenshot `17-workspace-page`, feature bullet), `README-DEVELOPER.md` (status, pinned Wails, tracked `wailsjs/runtime`, the rail/tab/Workspace page architecture, pre-release checklist), `docs/process/release-process.md` (tag the CI-tested SHA, `git pull --rebase` after bot commits, Docker version argument, pinned Wails), user guide (version 0.6.0, the Workspace page and rail behaviour), the GitHub Release notes, a new next-steps snapshot `docs/planning/next_steps/2026-11-21-next-steps.md`.
- **Known gap:** the 0.6.0 release notes inside the app (`AboutDialog.tsx`) predate the last UI fixes; the GitHub Release carries the complete notes.
- **Next:** B-081 (a sidebar and a last tab on every rail icon; proposal `docs/enhancements/2026-11-21-rail-sidebars-and-last-tab.md`) is planned for 0.7.0.

### 2026-11-21 — Rail sidebars and last tab per icon, phase 1 (B-081; design: `docs/enhancements/2026-11-21-rail-sidebars-and-last-tab.md`, approved as written)

- **One model for all six icons:** `lib/railIcons.ts` (`iconForTab`, `recordTab`, `tabToRestore`; replaces `railFocus.ts`). Clicking an icon shows its sidebar and brings back the last tab opened under it (this session only); clicking the lit icon collapses the sidebar; switching tabs lights the tab's icon and records it but never opens or closes the sidebar. Workspace, Check Conformance and CLI Builder open their page when nothing is open under them; Requests, Scenarios and Contracts do not create a tab. Opening a workspace shows the Workspace tab with the Workspace sidebar open.
- **Removed:** the 0.6.0 rules that closed the sidebar for page tabs and reopened it on return (`lastWasPage`, `showWorkspacePage`) and the `highlightedTab` / `workspaceActive` / `conformanceActive` / `cliActive` rail props.
- **Not yet:** the Workspace sidebar content (phase 2) and the Check Conformance and CLI Builder sidebars (phase 3); those three sidebars show only their title. Playwright: the 0.6.0 rail tests were rewritten for the new rule, plus two new ones (every icon opens its sidebar; each icon restores its last tab).

### 2026-11-21 — Workspace sidebar, phase 2 of B-081

- **Workspace sidebar** (`WorkspaceSidebarPanel`): workspace name and temporary badge, an environment selector that calls the same `handleEnvChange` as the status bar, the environment's source, the **Needs attention** list (clicking an item opens the Workspace tab and scrolls to that card, `data-attention-index`), and counts that switch to the Requests or Scenarios icon.
- **One snapshot:** `hooks/useWorkspaceOverview` reads `GetWorkspaceOverview` for both the Workspace page and the sidebar (while either is on screen, and again after a run, a Contracts change or an environment change), so they never disagree. `WorkspaceTab` now takes the snapshot as props instead of reading it itself.
- **Not yet:** the Check Conformance and CLI Builder sidebars (phase 3). The environment selector stays in the status bar as well.

### 2026-11-21 — Check Conformance and CLI Builder sidebars, phase 3 of B-081 (all three phases built)

- **Check Conformance sidebar** (`ConformanceSidebarPanel`): the workspace's specs and environments (plus *Active environment*), the choice on the page marked, and a *Last result* card (Covered, Passing, Failing of total; click to show the page). A click applies the choice on the page (`ConformRequest`) and shows it; the page reports its selection and its run totals back (`onSelectionChange`, `onResult`), kept in `App` for the session.
- **CLI Builder sidebar** (`CliBuilderSidebarPanel`): the catalog's commands with their goals; a click selects the command on the page (the existing `preset` path) and the selected one is marked (`onCommandChange`).
- **Both are pickers:** every option stays on the page; the sidebar lists are named *Workspace specs* and *Workspace environments* so they are not confused with the page's own fields.
- **B-081 is done:** the model (phase 1), the Workspace sidebar (phase 2) and these two. Not changed: the environment selector also stays in the status bar. Screenshots of the new sidebars are not generated yet.
- **Screenshots:** the whole gallery was regenerated for the rail with a sidebar on every icon; new `18-conformance-sidebar` and `19-cli-builder-sidebar` (dark and light, from `screenshots-workspace.spec.ts`), and `17-workspace-page` now shows the Workspace sidebar. README gallery updated.
- **In-app release notes:** `AboutDialog.tsx` has a 0.7.0 entry marked `unreleased` (the rail sidebars). At release time set its date and bump the version text in `AboutDialog.tsx` (and its two tests) and `help/user-guide.md`.

### 2026-11-21 — Pre-release batch for 0.7.0 (hardening and small items)

- **GitHub Action (B-058):** new inputs `fail-on`, `against`, `read-only`, `negative`, `explain`, `baseline`, `write-baseline`, `tags`, `paths`, `request-timeout`, `var` and `report` (one value per line). Every input now reaches the script as an environment variable instead of being pasted into it, so a value with quotes or shell characters cannot change what runs. `action.yml` had mixed line endings and is now LF. The CI guide lists the inputs.
- **Check Conformance reads only the open workspace (B-064, behaviour change):** `InspectConformSpec` and `RunConformance` refuse a spec or environment file that is outside the open workspace (links resolved) or when no workspace is open, with a message that says to copy the file into the workspace. The page only offers workspace files, so normal use is unchanged. `workspaceinfo.within` resolves symbolic links too (B-078): a link out of the workspace is reported as "outside".
- **Dev servers (B-063):** the Demo and Conformance Servers no longer send CORS headers and refuse a `Host` that is not `localhost`, `127.0.0.1` or `[::1]` and an `Origin` that is not their own, so a web page in the browser cannot read or change their sample data.
- **Temporary workspaces (B-061):** every temporary workspace carries `.clrsig-owner`, refreshed every 30 seconds by the copy of the app that opened it; startup and Start Demo Server remove only folders nobody has refreshed for 3 minutes (or, with no owner file, untouched for an hour). A second running copy no longer loses its workspace.
- **Workspace page (B-079):** a dismissed opening reminder stays dismissed for the session, per workspace, when the tab is closed and reopened.
- **Other:** `.gitattributes` (B-065); the screenshot gallery was regenerated (B-012).
- **Scenario result explains violations (B-067):** `StepResultIO.specFindings` carries `conformance.Diagnose` findings for each step with spec violations (what the spec expected, what came back, where, in the spec, why, fix, raw detail under Technical detail); the Setup row's Result tab renders them with the same `FindingView` component the Check Conformance page uses (now its own file). The raw validator lines are the fallback when a step has no findings. No file format change.
- **UI consistency and polish (B-068, B-069, B-070):** one name per feature (rail, tab and page say *Check Conformance* and *CLI Command Builder*; the tab and rail say *Contracts*, and its empty text names both sources); tabs carry the rail's icons instead of `SCN`/`CHK`/`CLI`/`WS`/`COV`/`RUN`; the request URL field keeps a usable width and the timeout and redirect options wrap beneath it (measured at 1280 px); the CLI Builder shows no required-field error before the first value, unselected choices look clickable and the explanation stacks under its term; the Check Conformance Base URL hint and asterisk wait for a spec; Settings shows *Automatic* for ports; the scenario toolbar line shows the file path instead of repeating the title; the step body-type dropdown is labelled; the `conform` goal in the CLI Builder reads *Spec Conformance Testing: check an API against its OpenAPI spec*. Test stability: Testing Library's async timeout is 5 s (the busy full suite made `findBy*` queries time out now and then). **Not done:** one entry point per place for Settings and Help, shorter Settings placeholders. **B-080 (Contracts store key collisions) left open on purpose:** a hash of the exact spelling would stop two spellings of one file from sharing a card, which is behaviour the demo relies on; the proper fix keys the store by the resolved file and needs your decision.

### 2026-11-21 — v0.7.0 released

- **Tag `v0.7.0` on `2599620`** (CI green on that exact commit). Release workflow green (all ten jobs); GitHub Release with 13 files and the full notes; Docker `ghcr.io/fuvito/api-workbench:v0.7.0` prints `clrsig version v0.7.0`; Scoop updated to 0.7.0 by the `chore: bump tap to v0.7.0` commit and installed on Windows (`clrsig version v0.7.0`).
- **Not verified:** the Homebrew install (no Mac) and a run of the Action at `@v0.7.0` (its `action.yml` parses and its script was tested locally). **Open:** B-080 and code signing (B-035); see `docs/planning/next_steps/2026-11-21-next-steps-after-0.7.0.md`.

### 2026-11-21 — Defect in the released Action (found after tagging 0.7.0)

- **What:** `action.yml` as tagged in `v0.7.0` (and on `main` since the B-058 commit) contained a lone carriage return inside a shell script. YAML reads it as a line break, so the file was not valid YAML and `uses: fuvito/api-workbench@v0.7.0` could not run. The binaries, Docker image, Scoop manifest and desktop builds are not affected. It came from my own scripted edit; the earlier local test ran the extracted script, not the YAML.
- **Fixed on `main`** (the character is now the two characters `` in the script) and **guarded:** `apps/cli/action_test.go` (`TestActionFileIsValid`) rejects a lone carriage return, parses the YAML, checks that the inputs the script reads are declared and runs `bash -n` on every script; it fails on the released file.

### 2026-11-21 — 0.7.0 prepared

- Version 0.7.0 in `AboutDialog.tsx` (with its complete release notes), its two tests and the user guide; the user guide says the Check Conformance page reads only workspace files. Release notes for the GitHub Release are drafted. **Not tagged yet:** waiting for the owner's check of the desktop app and a green CI on the final commit.
- **Left open on purpose:** B-080 (Contracts store keys; needs a decision, see the pre-release batch entry), B-006/B-007/B-008 and the other older items, code signing (B-035). **Not verified:** the Homebrew install and the GitHub Action at the new tag (they can only be tried after the release exists).

### 2026-11-21 — v0.7.1 released

- **Tag `v0.7.1` on `71e6d46`** (CI green on that exact commit). Release workflow green (all ten jobs); GitHub Release (13 files) is the latest release and carries the notes; the `v0.7.0` page has a "superseded" note about the Action. Docker `ghcr.io/fuvito/api-workbench:v0.7.1` prints `clrsig version v0.7.1`; Scoop updated to 0.7.1 by the tap-bump commit and installed on Windows (`clrsig version v0.7.1`); `action.yml` at the tag parses (25 inputs) and `TestActionFileIsValid` passed in CI on that commit.
- **Still not verified:** the Homebrew install (no Mac) and a real workflow run of the Action at `@v0.7.1` (the file is validated, the script was run locally). **Open:** B-080 and code signing (B-035); see `docs/planning/next_steps/2026-11-21-next-steps-after-0.7.0.md`.

### 2026-11-21 — 0.7.1 prepared

- Version 0.7.1 in `AboutDialog.tsx` (a 0.7.1 entry that says what was fixed), its two tests and the user guide. Release notes for the GitHub Release are drafted. Decision (owner): a patch release instead of moving the `v0.7.0` tag. The 0.7.0 release page gets a note that the Action must be used at `@v0.7.1`.

### 2026-11-22 — Findings from external testing, batch 1 (0.7.2)

- **Source:** the sandbox repository's `docs/clrsig-findings.md` (F-001 to F-021, found with v0.7.1); the whole list is tracked as B-082. F-001 and F-002 were already fixed in 0.7.x.
- **F-010 (false pass, the serious one):** a spec with `servers` matched nothing by path, so conformance skipped every response check, scenario steps were "not checked" and coverage showed 0%. `Spec.pathRouter` now builds the router without the servers and `Spec.relativePath` removes a server's base path (`/v1`) from the path, in the response check, the request check, `HasOperation` and coverage. The text run report adds `Spec: N of M responses were not compared with the spec (reason: n)` when any step was not compared. **Risk:** runs that were green because nothing was compared can now fail.
- **Generated requests (F-005, F-008, F-009, F-015, F-003, F-021):** credentials follow the operation's security scheme (`{{token}}`, `{{basicUser}}` / `{{basicPass}}`, `{{apiKey}}`; optional security adds none; global security is honoured), required cookie parameters become a `Cookie` header, array parameters follow `style` and `explode`, a media-type or schema-level `example` (and 3.1 `examples`) is the body, and form and multipart bodies are generated (the request check builds a real multipart document). Catalog: CS-REQ-003 added, CS-GAP-006 narrowed to XML bodies, CS-NC-001 notes the base path.
- **Not in this release:** batches 2 and 3 and the two decisions (F-018, F-020); see B-082.

### 2026-11-22 — Findings from external testing, batch 2 (0.7.3)

- **Negative probes:** `--tags` and `--paths` now limit the probes (a probe carries its operation's tags and path; filtered probes are left out, not listed as skipped); a required header gets a probe like a required query parameter; every `oneOf` / `anyOf` branch of a JSON body is probed with its own required fields (`... of branch N`); an operation whose authentication is optional (an empty security alternative) gets no "no credentials" probe. **Not changed:** the happy-path request still sends only the first `oneOf` branch.
- **Reports:** `ConformJSON` and `ConformJUnit` take the negative result (the CLI and the desktop export pass it): JSON gets a `probes` array and `passed` is false when a probe failed; JUnit gets one test case per probe, counted in tests and failures.
- **F-019:** `domain.Response` has `Truncated` and `ContentLength`; the executor trims the body to the limit and sets them; the response check reports kind `body-too-large` (limit and announced size) before any schema comparison, except for binary media types. Catalog: CS-RESP-013 and CS-PROBE-005 added, CS-PROBE-001 and 003 widened.
- **Open:** batch 3 (F-011, F-012, F-016, F-017, F-007) and the decisions F-018 and F-020 (B-082).

### 2026-11-22 — Findings from external testing, batch 3 (0.7.4)

- **Finding kinds (F-011):** the response check classifies body problems that are about the body as a whole: `body-empty`, `body-not-json`, `body-branch` (none or several `oneOf` / `anyOf` shapes, with the count; before it was reported as a missing field of the first shape) in `openapi`, and `conformance.Diagnose` turns them into `empty-body`, `invalid-body`, `no-branch-matched`; a type mismatch at the root is `wrong-type` like one at a field; value-rule breaks are `constraint`, `additionalProperties: false` breaks are `unexpected-property`; a failed `{{$function}}` is `dynamic-value-failed`, no longer `missing-value`. `body-too-large` has a finding too.
- **Check IDs (F-012):** `conformance.Finding.CheckID` (mapped from the kind in `checkIDFor`) appears as `checkId` in the JSON report, as `[CS-…]` at the start of the `--explain`, HTML, Markdown and JUnit explanation lines, and as a Check row in the desktop finding view. The default text output is unchanged. `other` findings have no ID.
- **F-017:** `conformance.invalidBodyValues` explains a JSON body made invalid by an unquoted environment value: field, `{{variable}}`, value and declared type (`Spec.BodyFieldType`); `ValidationError` gained `Expected`, `Got`, `Variable`.
- **F-007:** the runners use the spec's first server as the base URL when none is given, so the report and the "no operation got an answer" line name it.
- **F-016 (documented, not changed):** a bad dynamic function in a scenario is found before any request is sent (exit 2) by design; it is now CS-RUN-007 in the catalog, next to CS-RUN-002 for `conform`.
- **F-020 (behaviour change):** `clrsig coverage quality` exits 0 whether or not it finds gaps; `--fail-on-gaps` restores exit 1. CI steps that relied on the exit code must add the flag.
- **F-018:** a design proposal, `docs/enhancements/2026-11-22-response-time-assertions.md` (status Proposed); nothing built.

### 2026-11-22 — Response-time assertions, phase 1 (0.8.0; design: `docs/enhancements/2026-11-22-response-time-assertions.md`, accepted)

- **New scenario assertion `type: duration`** (additive, `schemaVersion` stays "1"; an older clrsig rejects a file that uses it): compares the response time in milliseconds (`Response.Duration`, the attempt that produced the response when `retry` is used) with `expected` using `eq`, `ne`, `gt` or `lt`; the failure says `the response took N ms; expected < L ms`. `scenario.Validate` requires a number of milliseconds (or a `{{variable}}`) and rejects `contains` / `matches`. Evaluated by `assertions.Evaluator`; reports label it `duration lt 500 ms`.
- **Desktop:** the step editor offers `duration` (numeric operators, a "max ms" hint). `format-spec.md`, the user guide, the AI prompt (only on request, with a margin) and the catalog (CS-GAP-005 now covers `conform` only) updated.
- **Not built:** phase 2 (`conform --max-response-time`) and phase 3 (recording durations in the Contracts store and baselines).
- **F-016 decision (owner):** a bad dynamic function in a scenario keeps stopping the run before any request (exit 2), documented as CS-RUN-007.

### 2026-11-22 — Response-time assertions, phase 2 (0.8.1)

- **`clrsig conform --max-response-time <ms>`** (default 0 = off): `conformance.Options.MaxResponseTimeMs` / `application.ConformInput.MaxResponseTimeMs`. An operation whose response took longer gets a `slow-response` violation **in addition to** its spec comparison (the response is kept). The violation's message holds the limit, not the measured time, so baselines stay stable; the measured time is in `DurationMs`, the finding's Got and the `--explain` text. Finding `slow-response`, check CS-RESP-014. Probes are not timed.
- **Wired through:** the CLI flag, the CLI Command Builder catalog (`max-response-time`, parity test), the GitHub Action input `max-response-time`, `ConformanceService` (`maxResponseTimeMs`), and the Check Conformance page (*Slowest acceptable response*, carried into Get CLI command).
- **Not built:** a per-operation limit (shape left open in the proposal) and recording durations in the Contracts store and baselines (phase 3). The MCP `conform` tool does not take the limit yet.

### 2026-11-22 — Response-time assertions, phase 3 (0.8.2)

- **Contracts store (additive):** `coverage.OperationResult` gains `lastDurationMs` and `slowestDurationMs` (omitted when zero, so older data and files stay valid and show a dash). `openapi.ExecutedRequest.DurationMs` feeds `ScenarioCoverage`, whose `ExercisedResult` carries the last and the slowest of the run (only requests that got a response count); the three producers (CLI `run`, desktop scenario runs, desktop conformance recording) pass the response time, and both accumulators merge with `coverage.MergeDuration` (last: this run's when a response arrived, else the stored one) and `max` for the slowest.
- **Shown:** the Contracts operation table (*Last / slowest*), the CSV (trailing `Last ms,Slowest ms`; existing columns unchanged), and the Markdown and HTML operation tables. Not in `coverage list` or `quality` text/JSON.
- **Baselines:** unchanged on purpose; a `slow-response` violation holds the limit, not the measured time.
- **Not built:** a per-operation response-time limit, and any failure on a trend (the proposal's non-goals). The proposal is complete otherwise.

### 2026-11-22 — Findings F-022 (writeOnly finding classified) and F-016 (bad function in a step) (0.8.3)

- A `writeOnly` property in a response (CS-RESP-011) was reported as kind `other` with no `checkId` and the generic "does not match what the spec declares". `conformance.Diagnose` now matches the validator message and gives it kind `write-only-property`, check ID CS-RESP-011, `where` (the property path), and its own why and fix (a write-only value may leak). Test: `TestFindings_WriteOnlyLeakHasItsOwnKindAndCheckID`; the catalog row cites it. Additive: the new kind value appears in JSON `findings` and the desktop finding view.
- **F-016 (reverses the earlier decision; owner request):** a bad `{{$function}}` in a scenario *step* no longer stops `clrsig run` before any request. That step fails when it is reached (not sent, error in the report, exit 1) and the others run per their fail policy; a folder run still runs the other scenarios. Problems in a scenario's own `variables:` still stop it before anything is sent (exit 2): `scenario.ScenarioLevel` filters `CheckDynamic`'s issues (Step 0). `clrsig validate` still reports every kind. Catalog CS-RUN-007 rewritten; tests `TestRun_UnknownFunctionInAStepFailsThatStepOnly`, `TestRun_BadFunctionInAScenarioVariableStopsBeforeAnythingIsSent`. **Risk:** a CI job that expected exit 2 for a typo in a step now sees exit 1.

### 2026-11-22 — Findings F-023, F-024, F-025 

- **F-023 (behaviour change):** a scenario step whose `{{name}}` has no value is **not sent** and fails with `not sent: no value for {{name}}` (URL, query, headers, auth and body, the same fields `conform` checks); later steps follow the fail policy. Before, the request went out with the braces in the URL. `variables.UnresolvedPlaceholders` is now the shared helper (`conformance.UnresolvedPlaceholders` wraps it). **Risk:** a scenario that relied on a literal `{{…}}` being sent (for example a body for a server-side template) now fails; a value that is meant literally can be given by a variable. The catalog row CS-RUN-001 now covers steps, and the user guide sentence ("a placeholder with no value is not sent") is now true for scenarios.
- **F-024:** `--seed` no longer depends on how the scenario file is spelled. The random values of a scenario come from the seed and a key made of its title (file name without extension when it has none; a repeated title gets `#2`, `#3` in run order), `seedKeys` in `apps/cli/run.go`; before, the key was the path as typed. **Behaviour change:** the values a given `--seed` produces differ from 0.8.3's (they now match the desktop, which already keyed by title). `bundle run` keys by the bundle's own file name, as before.
- **F-025:** `TRACE` is a valid method (`domain.MethodTRACE`, format spec) and an OpenAPI 3 path item's `trace` operation is generated, run by `conform`, probed and counted like the others; it is not read-only, so `--read-only` lists it as skipped. Swagger 2.0 has no `trace`.
- **Tests:** `TestRun_StepWithAMissingValueIsNotSent`, `TestRun_SeedGivesTheSameValuesForEverySpellingOfThePath`, `TestConform_RunsTraceOperations`.

### 2026-11-22 — Findings F-026 to F-032 (0.8.5)

- **F-026, F-027 (false passes):** `ValidateResponseByPath` checks a JSON body strictly after the library does (`strictJSONProblem`): bytes that are not UTF-8 and text after the value are `body-not-json` (`invalid-body`, CS-RESP-008). A status or header problem still comes first. `jsonpath` assertions already rejected trailing text and still do.
- **F-028:** a number beyond a 64-bit float (`1e400`) has its own finding, `number-out-of-range`, CS-RESP-015, that names the number and says it is a Clear Signal limit (it was kind `other` with a Go error text). A large integer for `type: integer` passes.
- **F-029:** the `--negative` summary says `FAILED` when any probe errored (CS-PROBE-006), as the exit code did, and says why.
- **F-030:** `domain.Response.HeaderValue` joins every value of a repeated header with `, `; header assertions and header extractions use it.
- **F-031:** `assertions.DecodeJSON` / `FormatValue`: jsonpath assertions and extractions keep an integer above 2^53 (or beyond int64) as its digits, and print a number in full (`1234567`, not `1.234567e+06`; this also fixes plain numbers of 7 or more digits).
- **F-032 (behaviour change):** a failed extraction fails the step and the run (exit 1, `passed: false`, headline FAILED). **Risk:** a scenario whose extraction silently found nothing used to exit 0 and now exits 1.
- **Tests:** `hostile_test.go`, `edge_assertions_test.go`, `TestNegativeText_ErroredProbesAreNotPassed`; catalog rows CS-RESP-015, CS-RUN-008, CS-RUN-009, CS-PROBE-006.

### 2026-11-22 — Findings F-033, F-034, F-035 (0.8.6)

- **F-033 (a 0.8.5 regression):** the strict single-document check added for F-026/F-027 only applies to `application/json` and `+json` (`isJSONDocumentType`); `application/x-ndjson`, `application/json-seq` and other types that merely contain "json" are not parsed as one document. A `+json` body with text after the value still fails.
- **F-034:** `ParseBytes` sets aside the `example` / `examples` of media types that are not JSON while the library validates the document (`setAsideNonJSONExamples`), so a valid XML example string no longer makes the spec invalid (exit 2); they are put back afterwards. A wrong example for a JSON media type is still rejected.
- **F-035:** the "no credentials" probe also removes an API key carried in a cookie (`dropCookiePair`), keeping other cookies; before, `Cookie: session=...` stayed and a correct server failed the probe.
- **Tests:** `TestConform_NDJSONIsNotOneDocument`, `TestParse_AcceptsAnExampleForANonJSONMediaType`, `TestParse_StillRejectsAWrongJSONExample`, `TestProbes_NoCredentialsDropsACookieAPIKey`; catalog rows CS-NC-009 and CS-PROBE-004 updated.

### 2026-11-22 — MCP `conform` takes `maxResponseTime` (0.8.7)

- The MCP `conform` tool accepts `maxResponseTime` (milliseconds, 0 = off, negative is an error) and passes it to `application.ConformInput.MaxResponseTimeMs`, like `clrsig conform --max-response-time`. Test: `TestMCP_ConformMaxResponseTime`.

### 2026-11-22 — Per-path response-time limits; Action install fix (0.8.8)

- **`clrsig conform --max-response-time-for /prefix=ms`** (repeatable): another limit for the operations under a path prefix. The longest matching prefix wins over `--max-response-time`; `0` switches the limit off there; a path limit works without a general limit. `conformance.Options.PathLimits`, `conformance.ParsePathLimits`, `application.ConformInput.PathLimits`. Carried by the GitHub Action (`max-response-time-for`, one per line), the MCP `conform` tool (`maxResponseTimeFor`, an array), the Check Conformance page (*Response-time limits for paths*) and the command catalog (deferred: the builder cannot express repeatable values yet, so Get CLI command does not carry it). The proposal's tag selectors are not built; the path prefix covers the stated need. Catalog: CS-RESP-014 extended, CS-GAP-005 is now only about trends.
- **Fix — the Action wrote the binary into the workspace:** `uses: fuvito/api-workbench@v0.8.7` downloaded `clrsig` to the current directory, so a repository with its own `clrsig/` folder (the sandbox has one) failed with `curl: (23)`. Found by the first real run of the Action (the sandbox's `action-smoke.yml`). It now installs into `$RUNNER_TEMP/clrsig-bin`; `TestActionInstallsOutsideTheWorkspace` guards it.
- **B-080 not built:** a hash in the Contracts store key would also split a spec that two scenarios spell differently (`../specs/api.yaml` and `../../specs/api.yaml`), which the Workspace page and the demo workspace rely on; waiting for the owner's choice (see the backlog).

### 2026-11-22 — CLI Builder: file name for the markdown format (not yet released)

- The builder's suggested report file (`clibuild.SuggestOutput`) had no extension for `markdown` and fell back to `.txt`; it now suggests `.md` and describes the format in the output note. `TestEveryReportFormatHasAnExtension` fails for any format the CLI offers that would fall back to `.txt`.

### 2026-11-22 — CLI Builder: browse buttons open where the field points (not yet released)

- The file and folder buttons next to a path field now open their dialog in the workspace when the field is empty, otherwise in the folder shown or the folder that holds the file shown (`pickerStart`: a relative value is read from the run-from folder; a value that does not exist yet opens its nearest existing parent; a web address opens the workspace). The run-from button does the same with its own value. `PickCliFile` and `PickCliFolder` take the current value and the run-from folder (Wails bindings regenerated). Tests: `TestPickerStart` and the builder tests.

### 2026-11-22 — CLI Builder: Server address from a running server (not yet released)

- When the Demo Server or the Conformance Server is running in the app, an empty **Server address** box (`base-url`) is filled with `http://localhost:<port>` and says where it came from. It is filled once per command and server, so a box the user clears or types in is never refilled or overwritten; the address is dropped again when the server stops, but only if it is still the one the builder wrote. `App` passes `runningServers` to `CliBuilderTab`. Tests: `Server address from a running server` in `CliBuilderTab.test.tsx`.

### 2026-11-22 — Findings from writing the Academy (0.8.9; source: `api-workbench-web/docs/clrsig-findings-from-academy.md`)

- **F1 (behaviour change):** `--tags` or `--paths` that selects no operation is an error: the run prints `NOTHING CHECKED`, names the filter, says how `--paths` matches and (on Windows Git Bash) that `MSYS_NO_PATHCONV=1` is needed, and exits `2`. A run that checked nothing for another reason (a spec with no operations, every operation skipped by `--read-only`) prints `NOTHING CHECKED` with the exit code unchanged. `ConformResult.FilterMatched`, `ConformOutput.NothingSelected` (also a warning on the Check Conformance page). **Risk:** a CI step with a mistyped filter used to be green and now exits 2. Catalog CS-RUN-010.
- **F2:** the fix for an undeclared status follows the status: 404 and 405 point at the route or method on the server; other 4xx say to read the answer; `OperationResult.BodySnippet` keeps the first 200 characters of a 4xx or 5xx body and the finding shows it (`Got`, and "The server's answer" in `--explain`).
- **F3:** `conformance.NetworkErrorKind` (unreachable, timeout, redirect loop, broken body, TLS) chooses the why and fix of CS-RUN-003; every operation keeps its own error unless the server is plainly down (all connection refused).
- **F4:** with `--baseline`, a run whose only failures are known ones prints `PASSED with N known violations` (the exit code was already 0). **F5:** a request-side finding is marked `(the request does not fit the spec; the answer was not judged)` on the first line. **F7:** a failed `schema` assertion shows `[CS-RESP-…]`, why and fix in the text report and `checkId` in JSON. **F8:** CS-RESP-007 names the rule and the offending value (`status must be one of [...]`, date-time without the regex). **F9, F10:** empty-spec headline, pluralised probe summary and an authentication-specific line for CS-PROBE-004. **F11:** `StepResult.SkipReason`, printed as `skipped: …` (failPolicy stop tells how to use `continue`). **F6:** documented in `--paths` help, the guide and the builder catalog (an exact-match form is not built).
- **Not changed:** the Docker and Homebrew checks; fixture and README fixes (D1–D4) belong to the sandbox and failure-catalog repositories.

### 2026-11-22 — v0.8.9 released

- **Tag `v0.8.9` on `34885b2`** (CI green on that exact commit). Release workflow green (all ten jobs); GitHub Release (13 files) is the latest and carries the notes; Scoop updated to 0.8.9 by the `chore: bump tap to v0.8.9` commit and installed on Windows; the installed binary shows `--paths /zzz` giving `NOTHING CHECKED` and exit 2 (and, in Git Bash, the path rewrite the message warns about). It also carries the three CLI Builder fixes listed above as unreleased.
- **Other repositories:** failure catalog `79c2901` (thirty READMEs used renamed folder names; `fm-020-01` and `fm-025-04` scenarios now run both steps), sandbox `5ee0206` (the failed-extraction description).
- **Not verified:** the Docker image run, the Homebrew install, the Action at `@v0.8.9`, and the sandbox's pinned expectations (`expectations/*.yaml`), several of which record 0.8.8 wording and will flip.

### 2026-11-24 — Check catalog as data, phase A of `docs/enhancements/2026-11-24-severity-and-check-ids.md`

- **`packages/core/checks`:** the checks that produce findings or "not checked" notes as data (`ID`, `Title`, default `Severity` failure/warning/note, `Area`); `All`, `ByID`, `SeverityOf` (an unknown ID is a failure). No dependency on other packages.
- **`conformance.Finding.Severity`** is set in `Diagnose` from that list; JSON conform findings and the desktop `ConformFindingIO` gain an additive `severity` (every existing check is a failure, so no outcome or exit code changes).
- **Guard tests:** every finding kind in `diagnose.go` must map to an ID in the list; every ID in the list must have a row in `what-we-check.md`; every row in that file must be in the list or in the test's documentation-only list (COV, DIFF, IMP, GAP and the run-level rows). Checked by adding a kind and an ID: both fail.
- **Not yet:** severity levels in reports and exit codes (phase B), baseline v2 (C), format validators (D).

### 2026-11-24 — Severity levels in reports and exit codes, phase B of `docs/enhancements/2026-11-24-severity-and-check-ids.md`

- **Core:** `conformance.OperationResult.Passed()` ignores findings whose check is not a failure; `Warnings()`, `ConformResult.OpPassed` / `WarningCount` / `FailOn`, `conformance.ViolationsFor` and `ViolationSeverity`; `application.ConformInput.FailOn` and `ConformOutput.Warnings`. `checks.WithSeverity` (tests and a later per-project setting) and `checks.ParseFailOn`. With a baseline, a warning counts as a violation only under `--fail-on warning`.
- **CLI, MCP, Action, builder:** `clrsig conform --fail-on failure|warning` (default failure; anything else is refused before a request is sent), MCP `failOn`, Action input `conform-fail-on`, catalog entry `fail-on` for conform (parity test).
- **Reports:** text shows `WARN`, a `!` mark and `PASSED with N warnings`; JSON adds `warnings` (report and operation) and `severity` on findings; JUnit makes a warning a passing case with `system-out`; HTML and Markdown show `WARN`; the explanation line starts with `(warning)`. Desktop: `failOnWarning` in the run input, `warnings` on operations, `withWarnings` in the output, a Fail on warnings option (carried into Get CLI command), a Warning chip and the warnings listed under Passing.
- **No behaviour change by default:** every existing check is a failure, so no result or exit code differs from 0.8.9 until a warning-level check exists (phase D / later). Tests lower a check with `checks.WithSeverity`.
- **Not yet:** baseline v2 (phase C), format validators (phase D); Playwright coverage for the warning chip.

### 2026-11-24 — Baseline v2, phase C of `docs/enhancements/2026-11-24-severity-and-check-ids.md`

- **Breaking change of the baseline file (`schemaVersion` "2"):** a violation is identified by `operation + check + where + rule`; `summary` is only for reading. `conformance.Violation` carries both identities (`Key` and `LegacyKey`), so one run can be compared with either format. Constraint violations keep their sub-rule (`rule`); a violation with no check ID is `other` with the validator text as `rule` (the one wording-sensitive identity left); probes map to CS-PROBE-001 to 004.
- **Version 1 is still read** (`Baseline.Legacy`): compared by the old wording, flagged in the baseline summary ("old format ... rewrite it with --write-baseline"), never written back (`Marshal` refuses). `--write-baseline` always writes version 2. A version 2 file is refused by older clrsig ("not supported"); a newer-than-2 file is refused here.
- **Evidence:** `apps/desktop/wails/testdata/baseline-v1-0.8.9.yaml` was produced by the pre-change code against the Conformance Server; `TestBaselineWrittenBy089StillWorks` runs it live (six known, none new, none fixed) and records and checks a version 2 baseline. Tests for rewording, constraints, probes, v1 compare, parse errors and a CLI round trip (v1 works, upgrade by `--write-baseline`).
- **Policy:** `docs/process/schema-migration-policy.md` section 5 gained the clause that machine-written regenerable files migrate by regenerating (no `clrsig migrate`).
- **Risk:** a CI job that compares a version 1 baseline keeps working until the validator wording changes; the baseline summary now says when to upgrade. `format-spec.md` and the user guide updated.

### 2026-11-24 — Format validators, phase D of `docs/enhancements/2026-11-24-severity-and-check-ids.md` (B-053)

- **Behaviour change (can turn a green run red):** `format: email`, `uuid`, `ipv4` and `ipv6` are now checked (`date`, `date-time` and `byte` already were); `openapi.SupportedFormats` lists them. A mismatch is a CS-RESP-007 constraint failure (`where` = the field, the finding names the format). `email` accepts a single address without display name, spaces or comments and allows a domain without a dot; `uuid` follows RFC 9562 (versions 1 to 8, nil and max). Unknown formats (`uri`, `hostname`, ...) are accepted as the OpenAPI specification allows: CS-GAP-001 is narrowed to them and pinned by a test.
- **Global effect:** the validation library keeps format validators in package-level maps, so the check also applies to generated requests and to the `schema` assertion, in every interface.
- **A spec still loads when its own example breaks one of these formats** (`Spec.IgnoredExampleProblem` keeps the message; examples are then not validated at all). Without that rule a spec with `example: "123"` on a uuid would have turned from valid into a parse error (exit 2). A wrong-typed example is still refused. Surfacing the ignored example to users is part of the spec lint (B-086).
- **Evidence:** `TestFormats_*` (every supported format, valid and invalid values, the message, the example rule), the catalog rows flipped from KnownGap to caught for email and uuid; the repository's CI steps against the example server still pass (checked locally).
- **Not done:** `uri`, `uri-reference`, `hostname`, `time`, `duration`; the release notes for 0.9.0 (phase E).

### 2026-11-24 — 0.9.0 prepared (phase E), not tagged

- Version 0.9.0 in `AboutDialog.tsx` (entry dated `unreleased`), its tests and the user guide; draft GitHub notes in `docs/process/release-notes-0.9.0-draft.md` with the two changes that can turn CI red first (formats, baseline format 2). **Waiting for:** the owner's answers to section 9 of the proposal and a green CI on the final commit; then set the date and tag.

### 2026-11-24 — v0.9.0 released

- **Tag `v0.9.0` on `19956dd`** (CI green on that exact commit). Release workflow green (all ten jobs); GitHub Release (13 files) with the notes published (formats and baseline format 2 first); Scoop updated by the tap-bump commit and installed on Windows (`clrsig version v0.9.0`); the released CLI shows `--fail-on`. The draft notes file was deleted after publishing.
- **Not verified:** the Docker image run (no daemon on the machine), the Homebrew install (no Mac) and the Action at `@v0.9.0`.

### 2026-11-24 — Project file `clrsig.yaml`, phase 1 of `docs/enhancements/2026-11-24-init-and-project-config.md`

- **Core:** `packages/core/projectconfig` (`Discover`, `Load`, `Parse`, `ForCommandLine`): `schemaVersion: "1"`, YAML or JSON through one model, keys checked exactly (unknown keys are errors with a "did you mean", `x-` kept), values checked with the flags' own functions (`checks.ParseFailOn`, `conformance.ParsePathLimits`), `${...}` rejected, two files in one folder is an error, search stops at the first file / `.git` / the home folder.
- **CLI:** persistent `--config` and `--no-config`; `applyProjectConfig` (`apps/cli/projectconfig.go`) fills only flags and arguments the user did not give: `conform` (spec, environment, baseUrl, the `conform` keys, reports), `run` (environment, `run.scenarios`, `run.failFast`, reports), `validate` (`run.scenarios`). Paths are relative to the file and shown relative to the current folder; stderr says `Using clrsig.yaml (...)`. A missing argument now fails with "conform needs a spec: pass one, or set spec: in a clrsig.yaml" instead of cobra's argument-count message.
- **Clarified:** `spec` does not apply to `run`; `impact` does not read the file yet.
- **Not yet:** `clrsig init` and `init github-actions` (phase 2), `config show` (phase 3), desktop (phase 4). Docs: `format-spec.md` "Project File", user guide.

### 2026-11-24 — `clrsig init` and `init github-actions`, phase 2 of `docs/enhancements/2026-11-24-init-and-project-config.md` (B-021)

- **Core:** `packages/core/scaffold` (`ProjectEntries`, `Apply`, `WriteNew`, `GitHubWorkflow`). Everything is created only when missing (hard link of a temporary file, so a reader never sees half a file and nothing is replaced); every path is checked to stay inside the folder, links resolved, before anything is written; a folder that exists is not touched (no `.gitkeep` added); another project file name (`clrsig.yml`, `.json`) counts as the project file.
- **CLI:** `clrsig init [folder] [--spec <file-or-url>] [--base-url <url>] [--dry-run]` creates `clrsig.yaml` (read-only by default), `environments/local.yaml`, `environments/ci.yaml` (token and base URL as `${NAME}` references), and `scenarios/`, `requests/`, `specs/`; it prints created and kept files and the next commands. A local `--spec` must parse; a URL is never fetched. `clrsig init github-actions` prints a workflow to stdout (spec and environment from the project file, the running release pinned); `--write [--output file]` saves it and refuses an existing file.
- **Evidence:** parsers read back what `init` writes (project file, environment files); a test checks every `with:` key of the generated workflow against `action.yml`; end to end: empty folder, `init --spec ... --base-url ...`, then `clrsig conform` with no flags passes read-only; idempotence, no overwrite, dry run, path and symlink escapes, no network.
- **Not yet:** `clrsig config show` (phase 3), desktop reading the file (phase 4), the CLI Command Builder note. Docs: user guide, CI guide.
- **Action:** `action.yml` no longer passes an empty `spec` or `scenario` input as an empty argument (for `run`, `conform`, `validate`), so a `clrsig.yaml` in the checkout can supply it; an input still wins.

### 2026-11-24 — `clrsig config show` and the builder note, phase 3 of `docs/enhancements/2026-11-24-init-and-project-config.md`

- **One table:** `configSettings` (`apps/cli/projectconfig.go`) says which key supplies which flag or argument on which command; applying the file and `config show` both read it. `projectconfig.Keys()` lists the format's keys and `TestConfigSettingsCoverEveryKeyAndRealFlag` fails if the two differ or a named flag does not exist. The refactor changed no behaviour (the phase 1 tests pass unchanged).
- **`clrsig config show [conform|run|validate]`:** for each command, every setting, its value and `file` or `default`; no project file, `--no-config` and a bad command name each have their own message. It applies the file to a fresh copy of the command, so it reports what the command would use.
- **CLI Command Builder:** `clibuild.Build` adds an info message when a `clrsig.yaml` applies where the command is run ("the command shown is written in full and does not depend on it; run clrsig config show <command>") for conform, run and validate; the command itself is unchanged.
- **Not yet:** the desktop (phase 4): Check Conformance and the Workspace page reading the file, and a card that suggests `init`.

### 2026-11-24 — Desktop reads the project file, phase 4 of `docs/enhancements/2026-11-24-init-and-project-config.md`

- **Core:** `projectconfig.InFolder` (only that folder, no walk up) and `Config.SetKeys`; `workspaceinfo.Report.ProjectFile` (found, path, error, keys) and two attention items: `project-file-invalid` (what / where / why / fix, with the parser's message) and `project-file-path` (the spec or environment it names does not exist).
- **Desktop:** `DashboardService` carries `projectFile`; `ConformanceService.GetConformContext` carries `projectDefaults` (spec, environment, base URL, tags, paths, negative, fail on warning, timeouts and limits). A path is used only when it is inside the open workspace and exists; `conform.readOnly: false` is not applied and says so. The Workspace page has a **Project file** card (what the file gives, or the `clrsig init` command to run in a terminal when there is none); Check Conformance preselects the file's values once and says where they came from.
- **Tests:** core, service and component tests; Playwright (173) passes against a dev server on a free port.
- **Done:** all four phases of the proposal. **Not done:** a button that runs `init` from the app (the card shows the command; writing files stays an explicit act in a terminal).

### 2026-11-24 — `clrsig conform --setup` (B-084)

- **Core:** `application.RunSetup` runs a scenario against the conform environment and returns the environment to use: the input plus what the steps extracted (later steps win, `--var` names are kept as typed). A value whose name looks like a credential (`reporting.IsSecretName`) is marked secret, so every report masks it. A scenario that fails, or cannot start, returns the environment unchanged and `Passed: false`.
- **CLI:** `clrsig conform --setup <scenario>`: the scenario is validated and run before the check; its `{{baseUrl}}` is filled from `--base-url` when the environment has none (and that value stays out of the check's environment); a failure prints the setup's report and stops with exit 2 before any operation is sent; the note names the values made available, never the values. `conform.setup` in `clrsig.yaml`, the Action input `setup`, the builder catalog entry.
- **Desktop:** Check Conformance → Options → **Run a scenario first** (the workspace's scenarios), `setupScenario` in the run input, `setupNote` in the result, preselected from `conform.setup` and carried into Get CLI command. The scenario must be inside the workspace.
- **Tests:** a login feeding a secured operation (and the same run without `--setup` reporting it not sent), no leak in text, JSON, HTML, Markdown or JUnit, a failed login stops before the check (exit 2, no operation sent), `--var` wins, invalid and missing scenario files, the project file key and `config show`; service test with a failed login and a path outside the workspace; component tests.

### 2026-11-24 — `clrsig validate --strict` (B-020)

- **Core:** `scenario.Lint(sc, LintOptions{Env})`: warnings for a step with no assertions that extracts nothing, an extraction no later step uses, and (with an environment) a `{{name}}` nothing defines (scenario variables, earlier extractions, the environment; `$functions` are not names). Disabled steps are ignored. A missing `requestRef` was already an error in `Validate`.
- **CLI:** `validate --strict [--env file]`: warnings print as `warning: ...` under the file; with `--strict` a warning makes the file not valid (exit 1, the summary and JSON `valid` agree, JSON gains `warnings`); without it nothing changes. `environment:` in `clrsig.yaml` now also supplies `--env` for `validate`. Action inputs `strict` (and `env` with it).
- **Not done:** the same findings as Workspace page attention items (B-071), which would reuse `scenario.Lint`.

### 2026-11-24 — 0.9.1 prepared, not tagged

- Version 0.9.1 in `AboutDialog.tsx` (entry dated `unreleased`), its tests and the user guide; draft GitHub notes in `docs/process/release-notes-0.9.1-draft.md`. It carries the project file, `init`, `config show`, `conform --setup`, `validate --strict` and the desktop reading the file; no existing result or exit code changes without a `clrsig.yaml` or a new flag. **Waiting for:** a green CI on the final commit, then set the date and tag.

### 2026-11-24 — v0.9.1 released

- **Tag `v0.9.1` on `bbd4350`** (CI green on that exact commit). Release workflow green (all ten jobs); GitHub Release (13 files) with the notes published (no result or exit code changes without a `clrsig.yaml` or a new flag); Scoop updated by the tap-bump commit and installed on Windows (`clrsig version v0.9.1`); the released CLI shows `validate --strict` and `init`. The draft notes file was deleted after publishing.
- **Not verified:** the Docker image run (no daemon on the machine), the Homebrew install (no Mac) and the Action at `@v0.9.1` (its new `setup` and `strict` inputs and the empty-argument fix are the part worth trying in a real workflow).

### 2026-11-24 — Weak scenarios on the Workspace page (B-071, page half)

- **Core:** `workspaceinfo.Inspect` runs `scenario.Lint` (the findings of `validate --strict`, without the environment rule, which the page already reports with the environment in view) after resolving `requestRef` files, and adds one **information** item per weak scenario (`scenario-weak`: what, where, why = one line per finding, fix). The page renders a multi-line Why. It adds to "Needs attention", so a workspace of scenarios without assertions shows one item each.
- **Fix (also in the CLI, `validate --strict`):** a value used only inside a request file a step references was reported as an extraction nobody uses; both now resolve `requestRef` files (next to the scenario and in the sibling `requests/` folder) before linting. Released 0.9.1 has the false positive.
- **Not done:** the other half of B-071: `clrsig validate` printing the workspace's other attention items.

### 2026-11-24 — `clrsig conform --changed-against` (B-085; design: `docs/enhancements/2026-11-24-conform-changed-operations.md`)

- **Core:** `openapi.ChangedOperations(base, revision)` (`ChangeSet`: added, modified, removed, unchanged, unattributed spec-level changes; a changed shared schema counts for each operation that uses it, as `Diff` reports it); `conformance.Options.OnlyOperations` (requests and negative probes, combined with tags and paths), `conformance.Scope` (`Line()`, shown in the text, HTML and Markdown reports and as `scope` in the JSON report); `application.ConformInput.OnlyOperations` and `Scope`.
- **CLI:** `conform <spec> --changed-against <ref>` (local file and git, as `diff --against`; `gitSpecAt` now names the flag in its errors). Only new or changed operations are sent; the rest show as skipped. Nothing changed: `NOTHING CHECKED`, exit 0 (not the exit 2 of a filter that matches nothing). The builder catalog entry and the Action input `changed-against` exist.
- **Bug fix for every limited run:** a baseline entry was reported as fixed when its operation had simply not been sent (`--tags`, `--paths`, `--read-only`). `conformance.CompareChecked` / `CheckedOperations` now call an entry fixed only when its operation was sent.
- **Behaviour change:** `--write-baseline` is refused together with `--tags`, `--paths` or `--changed-against` (it would drop the entries of the operations left out); record a baseline with a full run.
- **Limits, documented:** it follows the spec only (a server fault with no spec change is invisible) and unattributed spec-level changes do not widen the check; keep a full run on the main branch.

### 2026-11-24 — `conform --format github`: pull-request annotations (B-087)

- **Core:** `reporting.ConformGitHub(result, neg, locator)` renders GitHub workflow commands: an error per failing finding (a warning-level check is a warning), titled `<check ID> <METHOD> <path>`, with what / where / why / fix in the message, text redacted like every report, `%`, line breaks (and `:` `,` in properties) escaped; accepted negative probes are errors; one closing notice with the counts and the scope line. `openapi.OperationLines` finds the line of each operation in a YAML or JSON spec (yaml.v3 nodes), so an annotation points at the operation; a URL spec, or a file outside the current folder, leaves the annotation on the run.
- **CLI:** `--format github` is a screen format of `conform` only (`planReportsWith`): not a `--report` file, not offered by `run`, `diff` or `bundle run`. The Action's `format: github` works for `conform`.
- **Not done:** annotations for scenario runs (`run`: step lines in the scenario file) and `diff`.

### 2026-11-24 — Backlog batch: B-071 (CLI half), B-090, B-092, B-093, B-094, B-095; B-088 and B-091 reviewed

- **B-071 (CLI half):** `validate --strict` also warns when a scenario's `specFile` does not exist, and with `--env` for every `${NAME}` the environment needs that no process variable sets (printed to stderr, fails the strict check). With the Workspace page items this covers what the page showed.
- **B-090 (CI gaps):** CI now runs `go vet` (core, CLI, desktop), `go test` for the CLI and the desktop Go module (neither ran in CI before: only their builds did), the frontend Vitest suite, and the Playwright browser tests against the mock bridge in their own job. A test (`packages/core/doccheck`) checks every relative Markdown link. **Not added:** a vulnerability scanner (a tool outside the locked list; needs an owner decision). If the new CI steps fail on Linux for a reason that only shows there, fix the test, not the step.
- **B-092:** a **Copy as issue** button on every finding (Check Conformance and the scenario result): the check ID, summary, operation, expected, got, where, spec place, why and fix, and the version, as plain text for a bug tracker. Findings shown in the desktop now go through the same redaction as the reports (`reporting.Explain`): before, a secret that came back in a response could appear in a finding the page showed. Not done: an anchor and button in the HTML report.
- **B-093:** `doccheck.TestFormatSpecNamesEveryField` reflects over the request, scenario, environment, bundle and project file types and fails if a serialized field is not named in `format-spec.md`.
- **B-094 (partly):** the app's version is one constant (`APP_VERSION`, the newest release-notes entry in `AboutDialog.tsx`) used by the dialog and its tests, and `doccheck.TestVersionsAgree` fails if the user guide header says another version. Not done: rendering the notes from a `docs/CHANGELOG.md`.
- **B-095:** the engineering changelog moved from section 10 of `CLAUDE.md` to this file (`docs/process/engineering-changelog.md`); `CLAUDE.md` keeps the rules and a pointer (about 200 lines instead of 990 loaded in every session). New entries go here.
- **B-088 (data-driven scenarios):** not built. The earlier analysis (future enhancements 17, "think twice": a large change to the scenario format and the runner that moves toward load and data testing, which is out of scope, with no evidence of demand) still holds. The backlog row says so.
- **B-091 (actions on findings):** proposal written (`docs/enhancements/2026-11-24-fix-actions.md`), recommending to defer: only one finding has a button today, and attention items already carry a target.

### 2026-11-24 — 0.10.0 prepared, not tagged

- Version 0.10.0 in `AboutDialog.tsx` (dated 2026-11-24), the user guide (`TestVersionsAgree` checks they agree); draft GitHub notes in `docs/process/release-notes-0.10.0-draft.md`. The notes lead with three behaviour changes: `--write-baseline` refused with `--tags`/`--paths`/`--changed-against`, a baseline entry is fixed only when its operation was sent, and the stricter `validate --strict`. CI now runs the CLI and desktop Go tests, `go vet`, Vitest and the browser tests; the first run on Linux found two test-only problems (the parity test built the CLI with an isolated module cache, a temp workspace test set only the Windows temp variables), fixed in `ed9f16b`.

### 2026-11-24 — v0.10.0 released

- **Tag `v0.10.0` on `255aa92`** (CI green on that exact commit, including the new CLI, desktop Go, Vitest and browser steps). Release workflow green (all ten jobs); GitHub Release (13 files) with the notes published (the three behaviour changes first); Scoop updated by the tap-bump commit and installed on Windows (`clrsig version v0.10.0`); the released CLI shows `--changed-against` and `--format github`. The draft notes file was deleted after publishing.
- **Not verified:** the Docker image run (no daemon on the machine), the Homebrew install (no Mac) and the Action at `@v0.10.0` (new inputs `changed-against`, `setup`, `strict`; `format: github`).
- **Open:** B-091 (proposal recommends deferring), B-088 (reviewed, not proposed), the vulnerability scan of B-090, the HTML report anchor of B-092, the rendered `CHANGELOG.md` of B-094, annotations for `run` and `diff`.

### 2026-11-24 — GitHub Action self-test in CI

- A CI job (`action-selftest`) runs `action.yml` from the commit (`uses: ./`) against the example server: `conform` with an environment and a JUnit report must pass, `validate` must pass, and a run that fails (a `paths` filter that matches nothing, exit 2) must hand its exit code on through the `result` output. It uses the latest *released* clrsig, so it may only use inputs that release already has; the static check in `apps/cli/action_test.go` covers inputs added since. This is the first time the Action script runs in CI; before, only its YAML and shell syntax were tested.

### 2026-11-24 — Baselines in the Check Conformance page (B-006)

- **Service:** `ConformRunInput.BaselinePath` (inside the workspace; the file is read as `clrsig conform --baseline` does, version 1 and 2) and `ConformRunOutputIO.Baseline` (`known`, `new`, `fixed` with operation, check, where and summary, and `legacy`); the run verdict (`Failed`) already accounted for the baseline. `RecordConformBaseline(path)` writes the latest run as a version 2 baseline atomically, refusing before a run, after a run limited by tags or paths (`lastFiltered`), outside the workspace and for a name that is not .yaml, .yml or .json. `ConformContextIO.Baselines` lists workspace files named like a baseline; `conform.baseline` in `clrsig.yaml` preselects one.
- **Page:** Options → **Baseline**; a Baseline region with the three lists (new first, red); "all known to the baseline" in the headline when only known violations fail; **Record as baseline** in the Summary panel, with a replace confirmation when the file exists, then the file is chosen for later runs.
- **Tests:** service test (record, replace, compare, limited run refused, outside the workspace, bad extension, a 0.8.9 file accepted and flagged) and component tests. Not carried into Get CLI command: the builder catalog has no baseline mode yet.

### 2026-11-24 — Spec lint (B-086; decision D4: part of `validate` and a card on the page)

- **Core:** `Spec.Lint()` (`packages/core/openapi/lint.go`): CS-SPEC-001 operation without an operationId (warning), CS-SPEC-002 `required` names a property the schema does not declare (error; schemas that compose with allOf/oneOf/anyOf or allow additional properties are not judged; request bodies, responses and component schemas are walked), CS-SPEC-003 an example that broke a checked format (the ignored example of 0.9.0, warning). A duplicate operationId and an undeclared path parameter are not listed: the OpenAPI parser already refuses such a spec. The three checks are in `checks` and in the catalog (section 2.5).
- **CLI:** `clrsig validate --spec <file>` (scenarios optional); errors fail, warnings are notes and fail only with `--strict`; JSON gains `notes`. Action: `spec` for `validate`. **Desktop:** `InspectConformSpec` returns `lint` and Check Conformance shows a Spec health region as soon as a spec is chosen.
- **Our own specs** (example server, Conformance Server, demo) are clean.
