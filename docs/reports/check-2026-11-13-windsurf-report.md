# Assessment: Clear Signal (api-workbench) — 2026-11-13

Focus: **UI decluttering + navigation restructure.** The current interface is getting crowded — requests, scenarios, and the demo-server indicator all compete for the same sidebar/toolbar space, and simulations + analytical screens are planned next. This report defines the proposed layout, backed by a **clickable HTML wireframe**, for implementation by a follow-up agent.

## Deliverable

- **Wireframe (open in any browser, no build/server needed):**
  `C:\Temp\api-workbench-wireframe\index.html`
- Self-contained HTML/CSS/JS single file. Dark/light theme toggle works live (View → Toggle Theme). Every surface described below is clickable — verify the proposal interactively before implementing.
- **No project source files were changed** — this is a design artifact only.

---

## 1. Proposed layout

```
┌ File  View  Tools  Help ────────────────────────── menubar (unchanged)
├────┬───────────┬─────────────────────────────────
│ R  │           │ [tab][tab][tab]        +         tab strip (unified)
│ A  │  sidebar  ├─────────────────────────────────
│ I  │  (panel   │  tab-scoped toolbar              no env picker here
│ L  │   swaps   ├─────────────────┬───────────────
│    │   per     │  request editor │ response
│ ⚙  │   rail)   │                 │ (existing split)
│ ?  │           │                 │
├────┴───────────┴─────────────────┴───────────────
│ env: [local]  ● :8080 demo        my-workspace · 1 sim running   ← status bar (NEW)
```

### Activity rail (NEW, ~46 px, leftmost column)

- **Explorer** — request file tree + filter + import (today's `WorkspacePanel`, minus the scenario section).
- **Scenarios** — dedicated panel: scenario list with step counts, last-run status, "Run all", template gallery entry point. Gets the room the current bottom-of-tree section lacks.
- **Simulations** — cards for demo server, spec-derived mocks, fault-injection/chaos proxies; Start/Stop per card + "Recent hits" traffic log.
- **Analytics** — rail icon opens a **page tab** in the main tab strip (not a sidebar panel): stat cards, pass-rate trend, slowest endpoints, spec-conformance donut, violations feed.
- **History** — today's `HistoryPanel`, unchanged.
- **Settings / Help** — pinned at the bottom of the rail (replacing the toolbar overflow entries).

### Collapsible / expandable rail behavior (REQUIRED)

- Clicking the **active** rail icon collapses the secondary sidebar to zero width; the rail itself always stays visible. Clicking again (or any other icon) expands it.
- `Ctrl+B` toggles the sidebar the same way (VS Code convention).
- When collapsed, content gets the full width; the active rail icon keeps its accent indicator only while the panel is open.
- Persist `sidebarOpen` in app state alongside the rail selection — the existing `collapsed`/`sidebarOpen` logic in `Sidebar.tsx` is the model.
- Rail icons carry status dots (green pulse on the Simulations icon whenever any simulation is running) so collapsed panels don't hide live state.

### Unified tab strip

- Requests, scenarios, **and pages** share one tab strip — extend `TabInfo.kind` (`TabBar.tsx`) from `'request' | 'scenario'` to `'request' | 'scenario' | 'page'`. Page tabs get a distinct badge (Analytics uses a chart glyph) and are singletons (re-clicking the rail icon focuses the existing tab via a stable `key`).
- Dirty dots, close buttons, `+` new-request, and `Ctrl+1–9`/`Ctrl+Tab` shortcuts continue to work across all kinds.

### Status bar (NEW, ~24 px, bottom)

- **Left:** environment selector (moved out of `AppToolbar`), running-simulation chips (`● :8080 demo` — click copies `http://localhost:PORT`).
- **Right:** workspace name, running-sim count.
- Rationale: environment and running servers are **global context**, not tab actions. Every future simulation type gets a persistent home for its indicator instead of stacking into the toolbar like `demoRunning` does today in `AppToolbar.tsx`.

### Toolbar

- Becomes tab-scoped: file/method name, Save, + New, Run (scenario), overflow menu. Env picker and demo-server indicator move to the status bar.

---

## 2. What the wireframe demonstrates interactively

- Rail switching + collapse-on-reclick; Ctrl+B.
- Explorer: live filter, collapsible folders, click-to-open request tabs.
- Request tab: Send → fake latency → response body + status + Body/Headers/**Timing** sub-tabs with a DNS/Connect/TLS/TTFB/Transfer bar breakdown.
- Scenario tab: ▶ Run animates steps (idle → running → pass) with a summary banner (steps · assertions · spec coverage %).
- Simulations: Start/Stop toggles; running sims feed status-bar chips, the rail dot, and a live "Recent hits" list.
- Analytics page tab: cards (runs, pass rate, p95 TTFB, coverage), trend bars, slowest-endpoint bars, coverage donut, violations list (ERR/WARN/INFO — matches `clrsig diff` severities).
- History panel: method/status/latency rows, click restores a request tab.
- Menus (File/View/Tools/Help), Settings dialog, Help, About — all wired.

---

## 3. Implementation notes for the follow-up agent

- **Component mapping:**
  - `Sidebar.tsx` → becomes rail + secondary panel; `SidebarTab` union gains `'scenarios' | 'sims'` (and `'analytics'`/`settings`/`help` route to actions, not panels).
  - `WorkspacePanel.tsx` → drop `WorkspaceScenarioSection`; scenarios move to a new `ScenariosPanel` that also hosts the template-gallery entry and run-all results.
  - New `ActivityRail.tsx`, `StatusBar.tsx`, `ScenariosPanel.tsx`, `SimulationsPanel.tsx`, `AnalyticsPage.tsx`.
  - `AppToolbar.tsx` → remove env `<select>` and `demoRunning` block (move to `StatusBar`); keep tab actions.
  - `App.tsx` → add rail/panel/status state; main-area branch gains `activeTab.kind === 'page'` → `AnalyticsPage`. Keep the existing mount-preserving `display` toggling for scenario viewers.
- **Data sources already exist:** sim state ← `useDemo`/future simulation services; analytics ← `history.db` scenario-run rows + `ScenarioCoverage` (already returned by `scenario_service.go`); violations feed ← conformance `OperationResult` severities.
- **Order suggested:** (1) rail + collapsible sidebar + status bar skeleton with existing panels, (2) scenarios extraction to its own panel, (3) tab `kind: 'page'` + Analytics shell, (4) simulations panel once the backend sim service lands.
- **Intentionally not mocked:** request/response drag-resize (exists in `App.tsx`), scenario step *editing* (wireframe shows run results only), real charting lib for analytics — pick Recharts or keep CSS bars for the first pass.

## Verdict

The layout keeps every current surface reachable in one click while giving each upcoming feature class (simulations, analytics) a first-class home — and the collapsible rail preserves maximal editor width on narrow windows. Review the wireframe at `C:\Temp\api-workbench-wireframe\index.html`, then implement in the order above.
