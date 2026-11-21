# Rail sidebars for every icon, and a last tab per icon

**Status:** Accepted as written by the project owner on 2026-11-21, including the recommendations for decisions 4-7. Not built yet; planned for 0.7.0, phase by phase (section 5).
**Date:** 2026-11-21
**Author:** Claude Code, for the project owner
**Backlog:** B-081

## 1. Summary

Today the Activity Rail has two kinds of icon. Requests, Scenarios and Contracts open a **sidebar**; Workspace, Check Conformance and CLI Builder only open a **full-width page** and close the sidebar (`2026-11-21-workspace-dashboard.md`, then the rail rules added in 0.6.0). The owner finds this inconsistent: clicking different icons shows different layouts.

The idea: make every rail icon work the same way.

1. **Every icon has a sidebar.** Workspace, Check Conformance and CLI Builder get one, so all six icons open "a panel on the left and a tab on the right".
2. **Every icon remembers its last tab.** Clicking an icon shows its sidebar and brings back the tab that was last active under that icon (the last request tab for Requests, the last scenario or Run Results tab for Scenarios, and so on).

The rule "a page tab closes the sidebar" disappears, which simplifies the code added in 0.6.0: whether the sidebar is open becomes purely the user's choice.

## 2. Goals / non-goals

**Goals**
- One interaction model for all six icons: click opens the icon's sidebar and restores its last tab; clicking the lit icon again collapses the sidebar (as today).
- Switching tabs by hand still lights the icon the tab belongs to (kept from 0.6.0) and now also updates that icon's remembered tab.
- The sidebars are small and useful, not copies of their pages.
- No change to files, request execution or any Go package that runs requests or scenarios.

**Non-goals**
- No change to what the pages themselves show.
- No persistence of last tabs across app restarts (the app always starts on the landing page; see section 9, decision 4).
- Settings and Help stay what they are: modals, not rail panels.
- No new dependency.

## 3. Requirement review

- **Architecture:** consistent. This is frontend work. Sidebar content is read through the existing Wails services (`DashboardService.GetWorkspaceOverview`, `ConformanceService.GetConformContext`, `CliBuilderService` catalog); the frontend only renders (rule 1). Nothing is implemented in the UI that belongs in the core.
- **Locked stack / out of scope:** nothing new; nothing from the out-of-scope list.
- **File format contract:** unchanged. No new config field is required if last tabs are session-only (decision 4).
- **Reverses an earlier decision:** on 2026-11-21 the owner chose "Workspace is a full-width page with no sidebar" (option A) and, later the same day, the rail-follows-tabs rule that closes the sidebar for page tabs. This proposal replaces those rules. The code is small (`railFocus.ts`, a handful of lines in `App.tsx`), so the cost of reversing is low, but the 0.6.0 behaviour will have shipped first.
- **Size:** medium. Three new sidebar components, a small "last tab per icon" store, changes in `ActivityRail`, `Sidebar`, `App`, and tests. No release decision beyond the normal one.
- **Necessity:** the inconsistency is real (the owner hit it in two screenshots). The smaller alternative, already shipped in 0.6.0, is the "page vs panel" rule. **Recommendation: proceed**, because it is the only option that makes every icon behave identically; do it after 0.6.0 so 0.6.0 is not delayed.

## 4. Design

### Model

A **rail icon** is one of: `workspace`, `requests`, `scenarios`, `contracts`, `conformance`, `cli`. Every tab kind belongs to exactly one icon (`lib/railFocus.ts` already maps this):

| Tab kind | Icon |
|---|---|
| `workspace` | Workspace |
| `request` | Requests |
| `scenario`, `runner` | Scenarios |
| `coverage` | Contracts |
| `conformance` | Check Conformance |
| `cli` | CLI Builder |

State (session only, in `App`):

- `railIcon`: the icon whose sidebar is showing.
- `sidebarOpen`: whether any sidebar is showing (user preference, persisted as today in `ui.sidebarOpen`).
- `lastTabByIcon`: `Record<RailIcon, tabId | undefined>`.

Rules:

1. **Click on an icon that is not lit:** `railIcon` becomes it, the sidebar opens, and the icon's last tab becomes active if it is still open. If it is not open any more (closed, or never opened), nothing changes for tabs: the sidebar shows, and the sidebar's own "open" actions (clicking a request, a scenario, a command) create the tab. The exception is the three singleton pages, which have an obvious default: Workspace, Check Conformance and CLI Builder open their page tab if none exists.
2. **Click on the lit icon:** the sidebar collapses; the tab stays.
3. **Click on a tab:** its icon becomes lit, the sidebar (if open) switches to that icon's panel, and the tab is recorded as that icon's last tab. The sidebar's open/closed state is never changed by a tab click.
4. **Opening a workspace:** Workspace icon, its sidebar open, Workspace tab first and active, first request opened behind it (unchanged from 0.6.0 except that the sidebar is the Workspace sidebar).
5. **Closing a tab:** if it was an icon's last tab, that icon's memory falls back to the nearest remaining tab of the same icon, or to nothing.

This removes the 0.6.0 code that closes the sidebar for page tabs and reopens it when returning (`lastWasPage`), and the `highlightedTab` / `workspaceActive` / `conformanceActive` / `cliActive` props: with every icon having a sidebar, `railIcon` alone decides the highlight.

### The three new sidebars

Each is a compact summary that opens or fills the page, never a second copy of it.

**Workspace sidebar** (decision 2)
- Header: workspace folder name, "temporary" badge, a refresh button.
- **Environment:** name and where the choice came from (remembered, chosen, temporary, none); a selector to change it (the same control that is in the top bar today, so it is not duplicated there once this ships).
- **Needs attention:** the attention items as a short list (one line each, severity icon); clicking one shows the Workspace page scrolled to its card, or opens the target (scenario, Contracts) as the card's button does today. A "nothing needs attention" line when empty.
- Counts as links: requests, scenarios, specs (these switch to the matching icon).
Data comes from `GetWorkspaceOverview`, the same call as the page; both read one shared snapshot so they never disagree.

**Check Conformance sidebar** (decision 2)
- **Specs:** the workspace's spec files (from `GetConformContext`), one row each with method counts if known; clicking one selects it in the page's form and shows the page.
- **Environments:** the workspace's environment files with the active one marked; clicking selects it for the check (the page's environment field).
- Last result summary line for the selected spec if a run exists in this session (Covered, Passing, Failing); clicking shows the report.
The page keeps all options; the sidebar is a faster way to pick the two inputs.

**CLI Builder sidebar** (decision 3)
- The commands from the catalog (`run`, `conform`, `diff`, ...), each with its one-line goal; clicking selects the command on the page and shows it. The selected command is marked.
- A "Recent" section is not part of this proposal.

### Architecture

```
ActivityRail (6 icons) → App (railIcon, sidebarOpen, lastTabByIcon) → Sidebar → { Requests | Scenarios | Contracts | Workspace | Conformance | CLI }Panel
```

- A small pure module `lib/railIcons.ts` (extends `railFocus.ts`) owns the tab-kind → icon map and the reducer for rules 1-5, so the behaviour is unit-testable without React.
- Panels call the existing bridge functions; no new Go is needed except possibly a cheap `GetAttentionCount` if the overview call proves heavy for a sidebar (measure first; section 8).
- Frontend state shared by the page and its sidebar (selected spec, selected command, overview snapshot) lives in `App` (or a small context), not duplicated in each (CLAUDE.md: do not duplicate server state in React state).

### Security and data

Read-only. No new persisted data, no new file access, nothing leaves the machine.

## 5. Phasing and acceptance criteria

**Phase 1: the model.** Six icons, `railIcon` + `lastTabByIcon` reducer, rules 1-5, the three singleton icons open their page; sidebars for the new icons show a placeholder header only. Removes `lastWasPage` and the page-specific highlight props.
*Done when:* the Playwright walkthrough (click each icon, open tabs under two icons, click back and forth) restores the right tab each time and lights exactly one icon.

**Phase 2: Workspace sidebar.** Environment, attention list, count links; shared overview snapshot.
*Done when:* an attention item clicked in the sidebar shows the same card the page shows, and changing the environment in the sidebar updates the page and the top bar.

**Phase 3: Check Conformance and CLI Builder sidebars.**
*Done when:* choosing a spec or an environment or a command in a sidebar fills the matching control on the page, and the page's own controls stay in sync with the sidebar's selection.

Each phase ships on its own.

## 6. Testing

- **Unit (vitest):** the icon reducer (every rule, including a closed last tab and an icon with no tab), `railIcons` mapping, each sidebar with mocked bridge data, selection sync.
- **Playwright:** the click-through of all six icons, the "last tab per icon" cases, collapse on the lit icon, opening a workspace, and that the Settings and Help buttons are unaffected. The existing rail tests from 0.6.0 are rewritten for the new rule, not deleted.
- **Screenshots:** regenerate the gallery; add one of the Workspace sidebar.
- **Real app:** this is mostly layout and state, which the browser tests cover with the mock bridge; one manual pass in the desktop app is still needed.

## 7. Documentation impact

- User guide: replace the "The Workspace page" rail paragraphs (full-width page, rail follows tabs) with the new single model, and document each new sidebar.
- `CLAUDE.md` changelog per phase; backlog B-081 ticked as phases ship; B-068 (naming and rail state) is largely absorbed.
- README: only the screenshots, if the owner wants them.
- `format-spec.md` and the catalog of checks: no change.

## 8. Risks

- **Duplication.** Sidebars that repeat their page train users to ignore one of them. Mitigation: each sidebar is a *picker or list that drives the page*, not a second report, and shares its data with the page.
- **Narrow windows.** Six sidebars all take the same width; on a small window the page gets squeezed. Mitigation: the existing resizable, collapsible sidebar is kept; the page tabs already scroll.
- **Remembering a tab that no longer means the same thing** (a scenario tab whose file was deleted). Mitigation: rule 5, and the reducer only restores a tab that is still open.
- **Overview cost.** The Workspace sidebar would read the overview as often as the page does. Mitigation: one shared snapshot, refreshed by the same triggers as the page (workspace change, run, environment change, tab activation); measure on a large workspace before adding a lighter call.
- **Churn.** This replaces rules shipped hours earlier in 0.6.0. Mitigation: it is a deliberate simplification (one rule instead of two kinds of icon), and the tests from 0.6.0 become the specification for the new tests.

## 9. Decisions

Answered by the project owner on 2026-11-21:

1. **Timing:** tag 0.6.0 first, build this for 0.7.0 after the proposal is reviewed.
2. **Sidebar content:** Workspace shows the environment and the Needs attention list; Check Conformance shows the workspace's specs and environments.
3. **CLI Builder:** yes, it gets a sidebar that lists the commands, so every icon works the same way.

Accepted by the project owner on 2026-11-21 as recommended:

4. **Remember last tabs across restarts?** *Recommendation: no.* The app always starts on the landing page and reopens nothing; tabs are not restored today either. Session-only memory is enough.
5. **Where does the environment selector live once the Workspace sidebar has one?** *Recommendation: keep it in the top bar as well for now* (it is useful from every icon), and let both write the same state; revisit removing the top-bar copy after use.
6. **Should clicking an icon with no remembered tab open a blank one** (a new request for Requests, a new scenario for Scenarios)? *Recommendation: no.* Show the sidebar; the sidebar's own "new" and "open" actions create tabs. Only the three singleton pages open themselves.
7. **Rail order.** *Recommendation: keep Workspace first, then Requests, Scenarios, Contracts, then Conformance and CLI Builder grouped above Settings and Help* as today, but put the divider between the "work" icons (first four) and the tool icons; with every icon alike, the divider is only a visual grouping.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-21 | Proposal drafted after the owner asked for sidebars on Workspace and Conformance and a last tab per rail icon | Claude Code |
| 2026-11-21 | Timing: 0.6.0 first; build for 0.7.0. Content: Workspace = environment + attention; Conformance = specs + environments. CLI Builder gets a commands sidebar too. Decisions 4-7 left open | Project owner |
| 2026-11-21 | Proposal approved as written: decisions 4-7 accepted as recommended (session-only last tabs; environment selector in the top bar and the Workspace sidebar; no blank tab for an icon without one; Workspace first with a divider after the four work icons) | Project owner |
