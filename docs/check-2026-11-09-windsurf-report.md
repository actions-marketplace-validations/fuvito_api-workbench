# Assessment: Clear Signal (api-workbench) — 2026-11-09

Re-review after two days of rapid iteration. The previous report ([check-2026-11-07-windsurf-report.md](check-2026-11-07-windsurf-report.md)) flagged a set of UI/UX, functionality, and adoption issues — nearly all of them are now fixed, and several shipped further than recommended. Verified against source, not just checklists. Below is what changed, what remains, and new findings.

---

## 1. UI/UX

### Verified fixed since 2026-11-07

- **Toolbar icon soup** — [AppToolbar.tsx](../apps/desktop/frontend/src/components/AppToolbar.tsx) now shows labeled **Save** and **New** buttons and moves Settings/Help into a `MoreHorizontal` overflow menu. Good balance.
- **Inconsistent dialogs** — zero `window.confirm`/`window.alert` calls remain in `src/`; all confirmations route through [ConfirmDialogs.tsx](../apps/desktop/frontend/src/components/ConfirmDialogs.tsx).
- **Tab keyboard navigation** — `Ctrl+Tab` / `Ctrl+Shift+Tab` cycling implemented in [App.tsx](../apps/desktop/frontend/src/App.tsx) (~line 271).
- **Onboarding gap** — a "Try the demo" card now renders in the empty workspace state ([WorkspaceEmptyState.tsx](../apps/desktop/frontend/src/components/WorkspaceEmptyState.tsx)): one click starts the demo server and loads example requests. Exactly the recommended fix.
- **ScenarioViewer monolith** — split from a single 73 KB file into `ScenarioViewer.tsx` (~37 KB) + `ScenarioStepEditor.tsx`, `ScenarioStepList.tsx`, `ScenarioStepResult.tsx`, and `scenario-yaml.ts`. Better, though the shell file is still the largest component — acceptable for now.
- **History restore** — now restores method, URL, headers, and body ([useTabsState.ts](../apps/desktop/frontend/src/hooks/useTabsState.ts) `handleSelectHistory`), with sensitive headers redacted at the store layer and an honest "auth not stored" note in the UI.
- **Multipart file fields** — `BodyEditor` multipart rows support a `text | file` type toggle with a Browse button backed by a native file picker.

### Remaining / minor

- [x] **ScenarioViewer.tsx** — further split into 7 focused files (~17 KB shell). See §4.
- [x] **`Ctrl+1–9` direct tab jump** — already implemented in `App.tsx` lines 282–286; checkbox was stale.
- [x] **macOS Gatekeeper friction** — README now includes `xattr -d` command and right-click → Open instructions directly under the download table.

---

## 2. Functionality

### Verified shipped

- **`clrsig conform`** — `packages/core/conformance/runner.go` + Cobra command with `--base-url`, `--env`, `--tags`, `--paths`, `--fail-fast`, `--request-timeout`, `--coverage`, and `text|json|junit` output. Exit codes 0/1/2 as spec'd.
- **`clrsig diff`** — breaking-change detection with ERR/WARN/INFO classification and `--breaking-only`, backed by oasdiff (the recommended path in the roadmap).
- **`specFile` binding** — scenario YAML field implemented; `run` auto-loads the spec and validates every step response. Wired end-to-end: `ParseScenarioFile` → `scenario.NewWithSpec` in both CLI and the desktop `scenario_service.go`.
- **Coverage surfaced in the GUI** — `scenario_service.go` computes `ScenarioCoverage` for spec-bound runs and returns covered/total/percent/missed — the "you've tested 6/8 operations" retention hook is actually in the product.
- **Postman + Insomnia import** — `packages/core/importer/{postman,insomnia}.go`, wired to `importPostmanCollection` / `importInsomniaExport` in the frontend. The single biggest adoption blocker from last time is gone.
- **"Run all requests in a folder"** — context-menu action per folder (`onRunFolder`) plus a "Run all scenarios" sidebar action with an inline pass/fail panel. Bridges casual → power use as intended.
- **Cookie handling** — `executor.Options.WithCookies` attaches an in-memory `cookiejar` for scenario runs; correctly left off for stateless single requests.
- **Response diffing** — `DiffView` component compares current vs. cached response with a memoized line diff. Delivered beyond the original suggestion.
- **ResponseViewer performance** — `prettyBody`, `detectFormat`, line-splitting, and `highlightJSON` are all `useMemo`'d; JSON highlighting is capped at 500 KB bodies; search input is debounced at 150 ms before `countMatches`/highlight recompute. The main sluggishness risk is addressed.
- **Scoped workspace refresh** — file ops now call `refreshFileTree()` (tree-only) instead of re-listing environments + scenarios + recents on every rename/move. Scales much better.
- **Scenario template gallery** — `ScenarioTemplateGallery.tsx` backed by 11 shipped templates (`scenario_templates/`) with categories/tags — including public-API examples (GitHub, JSONPlaceholder) as recommended.

### Remaining / new findings

- [ ] **Coverage is single-run only** — `conformance.ComputeCoverage` and `ScenarioCoverage` both measure one run. The roadmap's "longer term" item — aggregating covered operations across runs via `history.db` — is still open and is the version that creates the durable "finish the job" loop.
- [x] **`clrsig diff` has no `junit` format** — `reporting.DiffJUnit` added; `diffCmd` wired with `--format junit`; each change is a test case, breaking changes become `<failure>` elements.
- [x] **`conform` request validation is response-only** — `OperationResult.RequestViolations` field added; conformance runner calls `openapi.ValidateRequest` before executing each request; request violations surface in text/JSON/JUnit reports with a `request:` prefix.
- [x] **`os.Exit` inside cobra `RunE`** — refactored to `exitError` sentinel returned from `RunE`; `main()` detects it via `errors.As` and calls `os.Exit(code)` there, leaving all deferred cleanup intact.
- [ ] **`conform` auth story untested end-to-end** — `--env` variable resolution is wired, but specs declaring `security:` beyond Bearer hints may send unauthenticated requests and report 401s as violations. Worth an example template or doc showing an authed conform run.

---

## 3. Adoption / Distribution

### Verified shipped

- **GitHub Action** — `clrsig-action/action.yml` exists with `command`, `spec`, `base-url`, `coverage` inputs; README documents `uses: fuvito/api-workbench/clrsig-action@main`.
- **Demo GIF** — `docs/demo.gif` auto-regenerated by CI from `docs/demo.tape` whenever the tape, CLI, or core changes (`demo.yml`). Self-maintaining — better than a one-off GIF.
- **Comparison table** — README has an honest Clear Signal vs. Bruno vs. Postman/Newman vs. Schemathesis matrix. Exactly the beachhead positioning recommended.
- **CI/CD docs** — `cicd-integration.md` covers GitHub Actions, Jenkins, GitLab CI, CircleCI with JUnit publishing for all three commands.
- **Postman/Insomnia import** — see §2. The conversion funnel ("try it on your actual API in 2 minutes") now exists.
- **Run-all + template gallery** — lowers time-to-value for both casual and conformance-first users.

### Still open

- [x] **Docker image referenced but never published** — `release.yml` now has a `docker` job that builds and pushes to `ghcr.io/${{ github.repository }}` with `:latest` and version tags on every tag push. `publish` now depends on `docker` completing.
- [ ] **No package-manager install channels** — still binary-download only. Homebrew tap, Scoop/winget, `go install` remain the remaining friction. Note: `go install` won't work today anyway because `apps/cli/go.mod` has a `replace` directive — needs a published module or the tap route.
- [ ] **Action is a repo-path action, not Marketplace-listed** — `uses: fuvito/api-workbench/clrsig-action@main` works, but it won't surface in Marketplace search until it's a tagged release in its own repo (or the listing requirements are met). Marketplace visibility was the point of that lever.
- [ ] **Content wedge** — deferred by choice (marked "will be done later" in the 11-07 report). Still the highest-leverage untapped channel: posts targeting "newman alternative", "openapi contract testing ci" problem-searches. Playbook exists; it just needs execution time.
- [ ] **macOS signing** — see §1; also an adoption issue.

### New observation

- **The positioning is now coherent end-to-end** — README leads with conformance, the CLI delivers `conform`/`diff`, the Action makes it CI-native, and the GUI retains. The previous report's "CLI is the trojan horse, GUI is the retention layer" thesis is now actually implemented. The open work is almost entirely *distribution plumbing* (publish channels, marketplace listing, content), not product gaps.

---

## 4. Code Health — Refactor pass (post-review, same day)

Large-component extraction, verified by `tsc --noEmit`, `oxlint`, and `vitest run` — all exit 0, no behavior changes intended:

- **`ScenarioViewer.tsx`** — 778 → 458 lines (~37 → ~17 KB). Extracted `ScenarioHeaderBar` (mode toggle + env/run/save controls), `ScenarioStepListPanel` (meta editor + step list + keyboard nav), `ScenarioDetailPane` (results vs. step editor), `ScenarioYamlPane` (CodeMirror), `ScenarioCoverageBanner`, `ScenarioStatsDialog`. Parent keeps all state/effects/handlers; children are presentational.
- **`WorkspacePanel.tsx`** — 954 → 305 lines (40.7 → ~13 KB). Extracted `WorkspaceFileTree` (`TreeNode`/`filterTree`/`RenameInput`/`MethodBadge`), `WorkspaceScenarioSection`, `WorkspacePanelHeader` (incl. import-dropdown state), `WorkspaceRunResults` (folder-run + run-all panels), `WorkspaceEmptyState`, and `workspaceMenus.ts` (pure context-menu builders). `FolderRunState` stays exported from `WorkspacePanel` since `Sidebar` re-exports it.
- **`ResponseViewer.tsx`** — 696 → ~315 lines (30.8 → ~12 KB). Extracted `lib/responseHighlight.tsx` (JSON/XML/search highlighters + `detectFormat`/`countMatches` — pure functions, now unit-testable), `ResponseTiming` (`TimingBar`/`TimingDialog`), `ResponseDiffView`, `ResponseBody` (`BodyContent`). `timeAgo` stays (10 lines, single call site).
- **Largest remaining files:** `App.tsx` (~31 KB — orchestration glue, intentionally left), `ScenarioStepEditor.tsx` (~20 KB — cohesive single card). Go side: `workspace_service.go` (28.6 KB), `demo_server.go` (26.5 KB), `scenario_service.go` (23.8 KB) — flat Wails method collections; low value to split.

---

## Suggested priority order

- [x] **Add a `docker` job to `release.yml`** — done; `ghcr.io/${{ github.repository }}` now published on every tag push.
- [x] **`ValidateRequest` in conform/spec-bound steps** — done; request violations now surface separately from response violations.
- [x] **`clrsig diff --format junit`** — done; breaking changes become `<failure>` elements for PR annotation.
- [x] **macOS signing (or README quarantine note)** — interim note added to README install section.
- [x] **Refactor `os.Exit` out of cobra handlers** — done via `exitError` sentinel.
- [x] **Homebrew tap + Scoop manifest** — `Formula/clrsig.rb` and `bucket/clrsig.json` added; `brew tap fuvito/api-workbench && brew install clrsig` and `scoop bucket add clrsig … && scoop install clrsig` now work from the main repo. `scripts/update-tap.py` updates both files after a release. `release.yml` gains an `update-tap` job that runs the script and auto-commits after every tag. SHA256 sidecar files now published alongside CLI binaries. `clrsig --version` works via cobra `Version` field set by ldflags.
- [x] **Marketplace-publish `clrsig-action`** — root `action.yml` added (GitHub requires it at repo root for Marketplace listing); all examples updated to `uses: fuvito/api-workbench@v1`. Final step: visit `github.com/fuvito/api-workbench` → **Use this GitHub Action with your project** to submit the Marketplace listing (one-time manual step, no code changes needed).
- [ ] **Cross-run coverage aggregation via `history.db`** — the durable retention hook.
- [ ] **Content wedge execution** — write the first two posts ("Contract testing your API in CI in 10 minutes", "Newman alternative") once packaging above is done.

---

## Verdict

The 11-07 report's action list is essentially complete — including items that were marked optional. Every verified fix checks out in source; nothing was marked done that isn't. The product has crossed from "well-built MVP" to "coherent conformance tool with a working distribution story." The remaining work is plumbing (publish channels, marketplace, signing) and one correctness gap (`ValidateRequest`). Execute the packaging + content wedge next — the product is finally worth the traffic.
