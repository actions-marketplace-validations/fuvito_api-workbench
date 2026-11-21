# Workspace dashboard and open/close reminders

**Status:** Done in 0.6.0 (all three phases built 2026-11-21; accepted the same day, all six decisions answered)
**Date:** 2026-11-21
**Author:** Claude Code, for the project owner
**Backlog:** B-055 (owner request, 2026-11-20)

## 1. Summary

When a workspace opens, the user has to work out its state for themselves: which environment is active and why, whether any `${NAME}` value is unset, whether a scenario names a spec file that is not there, and what the last run said. The app already knows all of this. It never says it in one place.

The idea is a **Workspace** page (a singleton page tab, like Check Conformance and Contracts) that answers "what state is this workspace in, and what needs attention?" at a glance, plus a short reminder when a workspace is opened or closed.

It reads and shows. It changes nothing about how requests, scenarios or environments run.

## 2. Goals / non-goals

**Goals**
- One page showing: folder path, number of requests and scenarios, the active environment and where the choice came from, the specs the workspace references, the last known results and coverage per spec, and a list of **attention items**.
- Attention items are concrete and each one has a button that goes to the fix (open the environment, open the scenario, choose a spec file).
- On open, a one-line summary of what was restored ("Environment: Staging, remembered for this workspace"); on close, a reminder of unsaved tabs and what will be remembered.
- All checks are computed by Go code that the CLI could also call (`packages/core/...`), so the same warnings are available to `clrsig validate` later. The page only displays them.
- Works the same for the temporary Demo and Conformance workspaces, which are never remembered.

**Non-goals**
- No change to request, scenario, environment or bundle file formats, and no new config fields (see section 4, Data).
- No new rules about what counts as a conformance violation.
- No run history store and no trend charts (that is B-017, Analytics).
- No editing from the page beyond navigation: it links to the existing editors.
- No telemetry, no network access, no new dependency.

## 3. Requirement review

- **Architecture:** consistent. The checks belong in a core package with no Wails dependency; a `DashboardService` in `apps/desktop/wails` is a thin wrapper, as `ConformanceService` is. The frontend only renders what the service returns (rule 1).
- **Locked stack / out of scope:** nothing new. No cloud, sync or collaboration features.
- **File format contract:** unchanged. The page reads existing files and the existing `~/.clrsig/config.json` fields (`workspaceEnvs`, `recentWorkspaces`). **If a later phase wants to remember "dismissed warnings" per workspace, that is a new config field and is deliberately left out of this proposal** (section 9, decision 4).
- **Security:** reading only. It must respect "restrict file operations to the selected project": spec-file existence is checked for paths inside the workspace; a `specFile` that is a URL is **not fetched** (the page says "URL, not checked"), and a `specFile` that resolves outside the workspace is reported as outside, not read. Secret environment values are never shown (only names, and whether a value is set).
- **Size:** medium. A new UI area (hence this document), one small core package, one Wails service, one page. No release decision is needed beyond the normal one.
- **Necessity:** the information exists but is scattered (landing page, environment selector, Contracts, scenario headers). The smaller alternative is the open/close reminder alone (a toast), which costs less but gives no overview and cannot show attention items. **Recommendation: build the page; phase 1 below is the smallest useful version, so the reminder alone is not a separate choice.**

## 4. Design

### User experience

**Where it lives.** An Activity Rail item "Workspace", the **first item in the rail**, opening a singleton page tab `workspace`. The page **opens automatically** when a workspace is opened (decision 1) and becomes the active tab; tabs the user already had open stay open behind it. Closing it is allowed and it can be reopened from the rail.

**Layout** (two columns, reusing the Check Conformance page's pattern: a summary column on the right, content on the left):

| Block | Content |
|---|---|
| Header | Workspace folder name and full path, a "temporary" badge for the Demo and Conformance workspaces, buttons: Open folder in file manager, Close workspace |
| Contents | `12 requests · 5 scenarios · 2 environments · 1 spec` (each a link that opens its panel) |
| Environment | Active environment name and **where the choice came from**: `remembered for this workspace`, `chosen this session`, or `none chosen`. A list of its variables: name, secret lock, and `set` / `unset ${NAME}` (never the value of a secret) |
| Specs | Each spec the scenarios reference (`specFile`), with: found / missing / URL (not checked) / outside the workspace, how many scenarios use it, and its last coverage (Covered, Passing, Failing, Not covered, from the Contracts store) |
| Last results | Last scenario run (time, passed/failed counts) while the app is open; otherwise the stored coverage's `lastRunAt` per spec. Plain "no run yet" when there is none |
| **Needs attention** | The attention items, most important first (below) |

**Attention items** (each: what, where, how to fix, in the same "what / where / why / how to fix" voice as the conformance findings; see the failure-reporting rule):

1. No environment selected, while a scenario or request uses `{{variables}}` that only an environment can supply.
2. An environment variable that references an unset `${NAME}` (shared with the CLI's warning text through `variables.MissingEnv`).
3. A scenario names a `specFile` that does not exist, or is outside the workspace.
4. A scenario fails to parse or validate (the parse error and the file).
5. A `requestRef` that points at a missing request file.
6. A spec in the Contracts store whose last result had failing operations.

Items 1, 3 and 4 are things the user would otherwise only learn by running; item 2 and 5 are the "run fails for the wrong reason" class we already guard against elsewhere.

**Open reminder.** Because the page opens by itself, the reminder is the first line of the page, dismissible: `Environment: Staging (remembered) · 2 things need attention`. When the user closes the page and opens another workspace later, it opens again with the new summary. It names the environment source, because per-workspace memory is invisible today. Never a modal.

**Close reminder.** File → Close Workspace already asks about unsaved tabs; the confirmation text gains one line: `Staging will be remembered for this workspace` (or `No environment will be remembered`). No new dialog.

### Architecture

```
Frontend (WorkspaceTab) → Wails binding → DashboardService → core `workspaceinfo` (new, pure Go)
```

- **`packages/core/workspaceinfo`** (new, small): `Inspect(root, activeEnvPath string, env *domain.Environment) Report` walks the workspace using the existing parsers (`serialization`, `scenario.Discover`, `variables.MissingEnv`, `openapi.ParseFile` only to confirm the file parses) and returns counts plus `[]Attention` items (`Severity`, `Where`, `What`, `Fix`, `Target`). No Wails, no React, no Cobra: it compiles and tests without the desktop.
- **`DashboardService`** (Wails): gathers the app-side facts the core cannot know (environment source, temporary-workspace flag, last run in memory, the Contracts store's `ListAccumulatedCoverage`) and merges them with the core report. It is a read-only service: no methods that write files.
- **Environment source** needs one new in-memory field on `WorkspaceService` (`activeEnvSource`: `remembered`, `chosen`, `none`) set in `enterWorkspace` and `SelectEnvironment`. It is **not persisted**; it is derived each time.
- **CLI later:** `clrsig validate` can print the same attention items without any change to the core package (out of scope here, listed as a follow-up).

### Data and file formats

No new files, no new fields in `~/.clrsig/config.json`, no `schemaVersion` impact. `format-spec.md` is unchanged. The report type is a runtime structure returned over Wails, not persisted.

Cost: `Inspect` parses every scenario and request, so it runs on demand (page open, refresh button, workspace open) and not on every keystroke. A workspace of a few hundred files should take well under a second; the phase 1 test measures it on the demo workspace and a generated 500-file one.

## 5. Phasing and acceptance criteria

**Phase 1: the page, read-only.** Core `workspaceinfo` with counts, environment block and attention items 1-5; `DashboardService`; `WorkspaceTab` page and rail item; Refresh button.
*Done when:* opening the Demo workspace shows its counts and an empty attention list; deleting its spec file and refreshing shows item 3 with the scenario name, the missing path and "choose the spec file in the scenario's Setup row"; an environment with an unset `${NAME}` shows item 2; a workspace with no environment chosen shows item 1.

**Phase 2: reminders.** The open summary line, the close-confirmation line and the automatic opening of the page (first rail item); environment source tracked in `WorkspaceService`.
*Done when:* opening a workspace shows the page as the active tab with the other tabs still open; one whose environment was remembered says so, opening a new one says `none chosen`, and the Demo and Conformance workspaces say `temporary, not remembered`.

**Phase 3: results and coverage.** Specs block with Contracts coverage, last scenario run summary, item 6.
*Done when:* running a scenario updates the last-results block without reopening the page, and a spec with failing operations produces item 6 linking to Contracts.

Each phase ships on its own and is useful alone.

## 6. Testing

- **Go (core):** table tests for `Inspect` over temporary workspaces (empty; valid; missing spec; spec outside the workspace; URL spec; unparsable scenario; missing `requestRef`; environment with an unset `${NAME}`; secret values never in the report; no environment). Real files, no mocks (`CLAUDE.md` §6).
- **Go (service):** `DashboardService` over a temp workspace, the environment-source states, temporary workspaces, and that no method writes to disk.
- **Frontend (vitest):** the page renders each block and each attention item with its fix button; empty and "no workspace" states; secrets show a lock and never a value.
- **Playwright:** open a workspace, see the page; break something, refresh, see the item and follow its link. Screenshots for the README only if the owner wants them.

## 7. Documentation impact

- User guide: a "The Workspace page" section and the new open/close messages.
- `CLAUDE.md` changelog entry per phase; backlog B-055 ticked as each phase ships.
- README: a short mention only if the owner wants it (product pitch stays separate from developer notes).
- `format-spec.md`: no change. Catalog (`what-we-check.md`): no change, because nothing here checks API behaviour. The "attention items" are workspace hygiene, not conformance checks, and should not get `CS-` IDs.

## 8. Risks

- **Noise.** Too many warnings train users to ignore the page. Mitigation: only items that make a run fail or mislead (the six above); no style advice. Item 1 must not fire for workspaces that use no environment variables at all.
- **Staleness.** The page is a snapshot. Mitigation: refresh on open, on environment change, after a scenario run, and a Refresh button; show the time of the snapshot.
- **A second source of truth.** If `Inspect` re-implements parsing rules, it can disagree with the runners. Mitigation: use only the existing parsers and validators; a test runs `Inspect` and `clrsig validate` over the same fixtures and compares their errors.
- **Slow on large workspaces.** Mitigation: on-demand only, measured in phase 1.
- **Scope creep toward Analytics.** Trends, history and charts belong to B-017; the page shows the latest values only.

## 9. Decisions

All answered by the project owner on 2026-11-21.

1. **Open automatically after a workspace opens:** yes, and the Workspace item is the **first item in the Activity Rail**. (Replaces the recommendation to show only a reminder.) Consequence: existing tabs are kept, the page is added and focused; the temporary Demo and Conformance workspaces open it too.
2. **Name:** "Workspace".
3. **Attention items block nothing:** the page informs; the runners keep reporting these cases themselves.
4. **Dismissing a warning per workspace:** left out for now (it would need a persisted per-workspace setting). Warnings must therefore stay accurate and few.
5. **Last results across restarts:** show the Contracts store's last-run time after a restart and say so; no run-history store (B-017).
6. **CLI parity:** yes, as a follow-up: `clrsig validate` prints the same attention items (backlog B-071).

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-21 | Proposal drafted from B-055 | Claude Code |
| 2026-11-21 | Accepted. Open automatically and first in the rail; name "Workspace"; no blocking; no dismissing for now; stored last-run time after restart; CLI parity as a follow-up (B-071) | Project owner |
| 2026-11-21 | Built, all three phases. Two deviations: (1) the scenario run log is already stored on disk (SQLite), so "last results" survive a restart and decision 5 is better met than the proposal assumed; (2) the page opens through the existing `onWorkspaceOpened` hook (open, new, recent, Demo Server), not by watching the workspace path, so a workspace already open when the app reloads does not steal focus; the Conformance Server opens the Workspace page first and then Check Conformance, so Check Conformance stays the active tab. Close reminder is appended to the existing unsaved-tabs confirmation only, as proposed | Claude Code |
