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
- [ ] **`Ctrl+1–9` direct tab jump** — cycling exists; index-based jumps don't. Nice-to-have for heavy keyboard users.
- [ ] **macOS Gatekeeper friction** — release builds are unsigned `.app` zips; first launch on macOS requires `xattr -d com.apple.quarantine` or right-click → Open. Worth a README note until signing exists — it's the first thing a macOS evaluator hits.

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
- [ ] **`clrsig diff` has no `junit` format** — text/json only. Fine for a pass/fail gate, but a JUnit report would let PR annotations show which change is breaking without parsing text. Low effort given `ConformJUnit` exists.
- [ ] **`conform` request validation is response-only** — generated requests aren't checked against the spec before sending (roadmap Gap 3, `ValidateRequest`). A malformed generated request can produce a misleading "server violation" instead of a "request didn't match spec" signal. Cheap to add — kin-openapi already does it.
- [ ] **`os.Exit` inside cobra `RunE`** — `main.go` calls `os.Exit(1)` mid-handler in `run`/`conform`/`diff`/`validate`. Output is flushed first so behaviour is correct, but any future `defer` (file close, cleanup) will silently not run. Prefer returning a sentinel error and exiting in `main`.
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

- [ ] **Docker image referenced but never published** — a root `Dockerfile` exists and the README documents `docker run ghcr.io/fuvito/api-workbench:latest`, but `release.yml` has no docker build/push job. **Anyone running the documented command gets nothing.** Either add a `docker` job to `release.yml` or remove the README section until it's real — a broken install path is worse than none.
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

- [ ] **Add a `docker` job to `release.yml`** (or cut the README Docker section) — fixes a broken documented install path today.
- [ ] **Homebrew tap + Scoop manifest** — the last real install friction; binary downloads are table stakes for a CLI pitching CI adoption.
- [ ] **`ValidateRequest` in conform/spec-bound steps** — closes the last roadmap gap that can produce misleading results.
- [ ] **Marketplace-publish `clrsig-action`** — extract or tag so `uses:` appears in Marketplace search; small effort, compounding visibility.
- [ ] **macOS signing (or README quarantine note)** — cheap interim: one line in the install docs.
- [ ] **Cross-run coverage aggregation via `history.db`** — the durable retention hook.
- [ ] **`clrsig diff --format junit`** — nice-to-have for PR annotation workflows.
- [ ] **Content wedge execution** — write the first two posts ("Contract testing your API in CI in 10 minutes", "Newman alternative") once packaging above is done, so the posts can end with a working `docker run`/`brew install` command.
- [ ] **Refactor `os.Exit` out of cobra handlers** — hygiene, not urgent.

---

## Verdict

The 11-07 report's action list is essentially complete — including items that were marked optional. Every verified fix checks out in source; nothing was marked done that isn't. The product has crossed from "well-built MVP" to "coherent conformance tool with a working distribution story." The remaining work is plumbing (publish channels, marketplace, signing) and one correctness gap (`ValidateRequest`). Execute the packaging + content wedge next — the product is finally worth the traffic.
