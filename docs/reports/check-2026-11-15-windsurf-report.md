# Clear Signal — Full Review (2026-11-15)

Read-only review of the desktop app (frontend + Wails backend), shared core usage, and `docs/`. No code was changed. Focus: bugs first, then improvements that make the app more attractive and useful.

## Verified bugs

### Backend (apps/desktop/wails)

- **Auth and per-request timeout are silently dropped on save — data loss.** `requestFileIOToDomain` (`workspace_service.go` ~line 880) maps Title/Method/URL/Headers/QueryParams/Body but never copies `req.Auth` or `req.TimeoutMs`. `LoadRequest` reads them back fine and `serialization.WriteRequest` would persist them — the domain request is simply built without them. Every save strips `auth:` and `timeout:` from the YAML. **Highest priority fix.**

- **`RequestService.Execute` cancels the previous in-flight request globally.** `s.cancelLast()` is invoked on every new `Execute` (`request_service.go` ~line 178). With tabs, sending from tab B silently aborts a running request in tab A (`context canceled`). Either queue per-tab or document the single-flight limitation.

- **Failed requests never reach History.** `recordHistory` returns early when `out.Error != ""` (`request_service.go` ~line 266). DNS failures, timeouts, and TLS errors leave no trace, which makes flaky-endpoint debugging harder. Consider recording errors with `statusCode = 0`.

- **`loadSpecBytes` uses bare `http.Get` with no timeout** (`scenario_service.go` ~line 1065). A slow or hanging spec URL can block a scenario run indefinitely; the frontend has no cancel path for scenario runs.

- **Imports silently overwrite existing files.** `saveGeneratedRequests` (OpenAPI) and `saveImportedRequests` (Postman/Insomnia) write `{slug}.yaml` via `atomicWrite` with no existence check — re-importing a collection clobbers user edits. Compare with `CreateFromTemplate`, which correctly appends `-2`, `-3`, ….

- **Cancelled OpenAPI dialog reported as an error.** `ImportOpenAPISpec` returns `Error: "no file selected"` on cancel (`openapi_service.go` ~line 61); the frontend then shows it as a real import error banner. Postman/Insomnia import errors are swallowed by the frontend instead — inconsistent handling in both directions.

- **`OpenWorkspaceByPath` accepts file paths.** `os.Stat` succeeds on a file (`workspace_service.go` ~line 761); the app then treats a file as a workspace and everything downstream fails silently. Check `IsDir`.

- **Workspace export bundles plaintext secrets.** `ExportWorkspace` packs `environments/` verbatim, including `secret: true` values, into the ZIP with no warning (`workspace_service.go` ~line 1097). Given env files are git-ignored by convention, export should warn or offer an exclude-secrets option.

- **`buildFileTree` scans the whole workspace root**, excluding only top-level `environments/` and `scenarios/` (`workspace_service.go` ~line 812). Any `.yaml`/`.json` elsewhere (e.g. `node_modules`, arbitrary folders) appears in the Explorer as a request, and `peekMethod` reads every listed file — slow on large trees.

- **`SaveRequest` case-only rename edge.** The conflict check compares `targetPath != req.FilePath` as strings (`workspace_service.go` ~line 331). On case-insensitive filesystems (Windows/macOS), renaming "Foo" → "foo" hits `os.Stat` success on the same file → false "already exists" error.

### Frontend (apps/desktop/frontend)

- **Native `alert()` regression.** `handleExportWorkspace` and `handleImportWorkspace` in `App.tsx` (~lines 180, 186) call `alert()`. The 2026-11-09 report stated zero `window.alert`/`window.confirm` remained — this is a regression. Also `handleApplyWorkspaceImport` uses `alert(res.error)` / `alert('Import complete...')`.

- **Windows-only hardcoded path separators.** `App.tsx` builds `${workspace.workspacePath}\\requests` for paste-to-root and matches `\\scenarios\\` + `replace(/\//g, "\\")` in `onDeleteScenarioFolder`. Paste-at-root and open-tab tracking after scenario-folder deletes break on macOS/Linux.

- **`ui.railTab` written but never read.** `localStorage.setItem('ui.railTab', tab)` on every rail click, but the initial state is hardcoded `useState('explorer')` — dead persistence; the active panel resets on every launch while `sidebarOpen`/`sidebarWidth`/`responseWidth`/`railExpanded` do restore.

- **Ctrl+S is a no-op on scenario tabs.** The global shortcut calls `saveTabById`, which returns `false` for `kind === 'scenario'` (`useTabsState.ts` ~line 216). `ScenarioViewer` has its own Save button, but the muscle-memory shortcut silently does nothing.

- **Alt+Arrow step-reorder fires in *every* mounted scenario tab.** `ScenarioViewer` registers a `document`-level `keydown` listener that calls `moveVisualStep` without checking whether its tab is active (`ScenarioViewer.tsx` ~line 137). Inactive scenario tabs stay mounted (`display: none`), so Alt+↑/↓ reorders the selected step in **all** open scenario tabs simultaneously and marks hidden tabs dirty. It also fires while typing in other tabs' inputs.

- **Run executes the on-disk file, not the unsaved draft.** `handleRun`/`handleRunFrom` call `runScenario(scenarioPath, …)` while `visualDirty` may be true — edits to steps made in the visual editor are silently ignored by the run. No warning tells the user the run reflected the *saved* version.

- **YAML dialog discards edits on backdrop/X close.** `ScenarioYamlDialog` tracks `isDirty` but `onClick={onClose}` on the backdrop and the X button close without confirmation — unsaved YAML changes are silently lost.

- **Nested `<button>` inside `<button>`.** `StepResultStrip` renders a "Run from here" button inside the expandable header button (`ScenarioDetailPane.tsx` ~line 65) — invalid HTML; behavior depends on browser quirks.

- **Silent failure paths.** `handlePasteToRequestFolder`, `handlePasteToScenarioFolder`, `handleMoveScenario` use empty `catch {}` blocks; `useWorkspace.refreshWorkspace` does `catch { return [] }` — a backend failure leaves stale UI with no error state. Clipboard-copy in `StatusBar` also swallows errors.

- **Demo workspace lifecycle gaps.** `handleStartDemo` swaps the workspace with no dirty-tab check (unlike `requestCloseWorkspace`/`requestOpenRecent`). `handleStopDemo` only closes tabs whose *URL* starts with `localhost:{port}` — tabs opened on request/scenario *files* inside the temp demo workspace keep `filePath`s pointing into a deleted temp dir and will fail to save later. Crashed sessions also leak `clrsig-demo-*` temp dirs forever.

- **No cancel for scenario runs.** Neither the single-scenario Run button nor the sequential batch loops (`handleRunAll`, `handleRunScenarioFolder` — ~200 lines duplicated in `App.tsx`) can be stopped mid-run; `RunScenario` has no cancellation path at all.

- **Ctrl+Enter dead inside focused inputs.** The shortcut handler skips `INPUT`/`TEXTAREA` targets, so the Send button's "Ctrl+Enter" tooltip doesn't work in the places a user most often is — the URL field (where plain Enter works instead). Minor.

- **`handleSelectHistory` drops query params into the URL.** History restore fills the URL (with its `?query`) but leaves the Params table empty, while the URL-paste path extracts params into rows. Inconsistent editing experience for the same request shape.

- **`handleUrlPaste` hijacks any paste containing `?`.** Pasting a query-string fragment into a partially-filled URL replaces the entire URL — the intercept checks `text.includes('?')` unconditionally.

- **`CoverageTab.refresh` has no catch** — a `getAccumulatedCoverage` rejection is an unhandled promise rejection and the UI shows "No data" rather than an error.

- **`exportCSV` emits no quoting** — operation paths containing commas produce malformed CSV (`CoverageTab.tsx` ~line 242).

- **Double-bound Contracts rail item.** Clicking the bar-chart rail icon both opens the sidebar spec list *and* forces open a Contracts main tab, even when the user only wanted the panel. Surprising dual action.

- **Minor inconsistencies:** `StatusBar` uses a literal `✕` character instead of a Lucide icon; the "New Environment" pencil only renders inside the `environments.length > 0` block, so there's no status-bar entry point to create the *first* environment; `MenuBar` has no keyboard navigation (Alt/arrows); document title is set twice (React effect + Go `syncTitle`).

## Attractiveness & usefulness (main target)

- **Quick Open only covers request files.** `QuickOpen.tsx` flattens `fileTree` (requests only). Scenarios — a major app surface — plus environments are unreachable by keyboard. Evolving it into a command palette ("Run scenario…", "New request", "Switch environment") is the single biggest power-user win.

- **Landing page is good but thin.** `LandingPage.tsx` is clean; consider "Resume last workspace" as a first-class button, and warn that "Try the demo" swaps the current workspace.

- **History sidebar lacks context.** Entries show method/status/URL but no grouping by request file or re-run-in-place action; "Clear history" appears to have no confirmation. Restoring auth is documented as missing — consider storing secret references (`{{var}}`) rather than values to close the gap safely.

- **Tab bar context menu is sparse.** Missing "Close all tabs", "Copy path", "Reveal in file explorer" (the `openInExplorer` backend already exists).

- **Scenario editor is now strong — polish the gaps.** The refactored viewer (split step list + editor, per-run env picker, run-from-step, YAML dialog, stats dialog, coverage banner) is the app's best screen. Remaining: no save-before-run prompt (runs the stale file), no cancel, and `handleUseTemplate` ignores `result.error` — a failed template creation shows nothing (`ScenariosPanel.tsx` ~line 73).

- **Batch-run UX.** Run-all results still render inline in the sidebar — promote them to the main area (a "Runner" tab or reusing the scenario result strip) so pass/fail summary and per-step detail get room to breathe.

- **Response viewer polish.** Headers table lacks a copy button; there is no "Save as example/expectation" or one-click promotion of a response into a scenario step — a compelling path from manual request → contract test.

- **Demo discoverability.** The demo is the best onboarding asset but only reachable from the landing page and a small rail dot. A persistent "Stop demo" affordance in more places and a guided first-run ("send this request") would improve activation.

- **No way to create an environment from the editor.** When a `{{var}}` shows the amber "not found" hint, offering "create variable in {env}" inline would close the loop.

- **Accessibility.** `MenuBar`, tab strip, and rail lack complete keyboard/ARIA interaction models; menu has no arrow-key navigation.

## Docs findings

- **`docs/product/Project.md` is badly formatted** — it reads like a raw chat export: numbered lists restart at 1, the decision table is flattened into unlabeled lines ("Area Decision / Language Go"), and ASCII art is garbled. Either reformat as proper markdown or mark it as a historical transcript; as the first doc a contributor sees, it undercuts polish.

- **`docs/user-guide.md` is stale against the rail UI.** It references "Open Existing in the sidebar", "save icon in the toolbar", "gear icon in the toolbar", "pencil icon in the top bar", and the `?` icon "in the toolbar" — all moved to the Activity Rail / Status Bar / menu bar. The shortcuts table is missing Ctrl+W, Ctrl+Tab, Ctrl+B, Ctrl+Shift+S, Ctrl+1–9, and Ctrl+F (response search).

- **`docs/UserGuide.md`** is a correct redirect stub — fine as-is.

- **Prior check report claim is now stale:** `check-2026-11-09-windsurf-report.md` stated no `window.alert`/`window.confirm` calls remained — regressed (see verified bugs).

## Suggested priorities

1. **Fix the save data loss** — map `auth` + `timeoutMs` in `requestFileIOToDomain` and add a regression test (highest impact, smallest change).
2. **Kill the silent failures** — surface file-op/import/copy/move errors via the existing toast/dialog patterns; remove `alert()` calls.
3. **Cross-platform paths** — replace `\\` literals in `App.tsx` with separator-agnostic path joins.
4. **Restore `ui.railTab`** — one-line localStorage read on init.
5. **Protect imports from overwrite** — conflict-check or auto-rename in the two import save loops.
6. **Cancelable scenario runs** — add a cancel token to `RunScenario`/`RunFolder` and a stop button for run-all; add timeout to `loadSpecBytes`.
7. **Scenario-editor correctness** — scope the Alt+Arrow handler to the active tab (or move it to the step list's own `onKeyDown`), and warn/offer save when running with unsaved edits.
8. **Upgrade Quick Open → command palette** (requests + scenarios + actions).
9. **Refresh `docs/user-guide.md` and fix `docs/product/Project.md` formatting** to match the current UI.
