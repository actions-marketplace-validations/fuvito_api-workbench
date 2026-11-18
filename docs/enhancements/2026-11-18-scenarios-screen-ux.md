# Scenarios screen: declutter (UX review)

**Status:** Accepted (direction); not started
**Date:** 2026-11-18
**Related:** `2026-11-18 Scenarios Screen UX Simplification Report.md` (earlier, code-independent report; this one is grounded in the components and takes a smaller route)

## 1. Summary

The Scenarios screen edits, runs and shows results in one place, which you like and which I would keep. It is now crowded because of **how much is stacked in one vertical column**, not because of how much it does. Counting from the code (`ScenarioViewer.tsx`, `ScenarioDetailPane.tsx`), before the user reaches the first editable field of a step there can be up to **eight stacked bands**:

| # | Band | Component | Shown when |
|---|---|---|---|
| 1 | Header (title, env, run, stats, save, discard, YAML, CLI) | `ScenarioHeaderBar` | always |
| 2 | Run progress / pass-fail bar | inline | during / after run |
| 3 | specFile warning | inline | after run, if spec error |
| 4 | Coverage banner (title, bar, "this run", "not yet covered", quality panel) | `CoverageBanner` | after run, or accumulated coverage exists |
| 5 | Description + Spec file | `ScenarioMetaSection` | always |
| 6 | Scenario variables | `ScenarioVariablesSection` | always |
| 7 | Run info (setup error, seed, generated variables) | `ScenarioRunInfo` | after run |
| 8 | Pane label "Step Editor" | `StepDetailPane` | always |

Inside the step pane the stack continues: result strip, then `StepExchange` (request section, response section), then the editor card (request, headers, params, body, assertions, extractions, advanced). A failed step with a large response pushes the editor below the fold.

## 2. Honest opinion

1. **Bands 3-7 are scenario-level and mostly static, yet they sit above the working area.** Description and spec file are set once. Variables are touched occasionally. Coverage and run info are run-level facts. All of them take full-width rows that never scroll away, so they permanently shrink the area the user actually works in. This is the biggest win and the cheapest fix.
2. **Result and definition of the same step are stacked, not related.** `StepResultStrip` + `StepExchange` + `StepEditorCard` are three views of one step in one scroll. After a run, the user wants "what happened" first; while authoring, "what I wrote". Showing both always makes the pane a long document.
3. **The step list already is the timeline.** It shows pass/fail per step, so a separate run summary page adds little.
4. **The earlier report's Build | Run | Inspect modes would work but cost too much.** They bring back the edit/run split you said you like, add a mode state to keep in sync with selection, and move the most common loop (fix assertion, rerun, look at response) across two modes. A per-step toggle gives most of the benefit with one small piece of state.
5. **Run All results already moved to the Runner tab** (changelog 2026-11-17), so the scenario screen should only carry results for *this* scenario's last run.

## 3. Recommendation

Keep one screen, no global modes. Three changes, in this order.

### A. A pinned "Setup" row replaces bands 3-7 (largest effect)

- First item in the step list, styled differently from steps (own icon, no number, not reorderable, not deletable, no enable/disable). It is **UI only**: it is not an entry in `steps:`, so the file format, `schemaVersion`, `format-spec.md`, CLI, bundles, importers, report indices and "run from step N" are untouched. It is deliberately not called "Step 0": it has no request and no pass/fail, and a literal step would invite "can I disable it?" questions.
- **Definition** tab: description, spec file, scenario variables (the existing `ScenarioMetaSection` and `ScenarioVariablesSection` content). Room for later scenario-level defaults (default headers, timeout, base URL).
- **Result** tab (after a run): seed and replay command, generated variable values, "Scenario could not start" error, specFile warning, and a one-line coverage summary (`5/8 operations, 62%`, with accumulated coverage and reset). Same `Definition | Result` pattern as steps (section B).
- "Always runs first, whatever step you start from" is already true: scenario variables are generated once per run before any step. Nothing changes at runtime.
- Failures stay visible without opening it: the Setup row shows a red dot, and a setup error also appears as a one-line alert under the header.
- A scenario with steps opens with step 1 selected; an empty one opens with Setup selected (see the ghost row below).
- Not in scope: setup that sends requests (log in first). That would be a before-all hook with its own failure semantics; today a login is step 1, and the OAuth2 pattern is documented that way.

#### Empty scenario: the "Start here" ghost row

When a scenario has zero steps (new, or the user deleted them all), the list shows Setup followed by a dashed, dimmed placeholder numbered `1`, titled **"Start here"**, with a `+` icon. The user sees that steps come after Setup.

- **Not a real step.** An empty step would fail validation (label and URL required), make a new scenario dirty on open and leave a junk step in the file. The ghost row is UI only and is never saved. It has nothing to delete, no drag handle and no menu.
- **Click creates a real step 1** and focuses its label field. The detail pane next to it shows "Add your first request" with quick actions: blank step, template (`ScenarioTemplateGallery`), saved request.
- **Deleting the last step is allowed**; the ghost row returns and Setup stays.
- **No duplicate add controls:** the bottom "+ Add step" button is shown only when at least one step exists.
- **Selection:** a new or empty scenario opens with Setup selected, the ghost row right below it.
- **To verify before building:** a zero-step scenario can be saved and loaded (validator, serializer), and Run is disabled with the tooltip "Add a step to run" (runner behaviour not yet checked).

#### Run from here (follow-up, optional)

Starting at step N skips earlier steps, so values they extract (a token, an id) do not exist. The Setup/step view should warn: "Starting here: `token`, `userId` are not set (extracted in steps 1-2)". This is the real pain point behind "setup always runs"; Setup alone does not solve it. Verify the runner's current behaviour before building.

### B. Per-step `Definition | Result` toggle in the detail pane

- Segmented control at the top of the pane, replacing the "Step Editor" label. The `Result` tab is disabled until the step has a result, and carries the status dot and code (`✕ 404`).
- `Definition` = `StepEditorCard`. `Result` = result strip contents + `StepExchange` + assertions and extractions.
- After a run, auto-select `Result` for the first failed step (or the selected step if it ran); any edit to that step switches back to `Definition`. Users can always click either.
- A one-line "Last run: FAILED · 404 · 92 ms · 1 assertion failed" stays above the editor in `Definition`, so the editor never hides the verdict.
- "Run from here" stays in both views.

### C. Calmer step editor

- Collapse empty Headers, Params, Extractions and Advanced by default (Advanced already opens when it has non-default values); show a count badge when collapsed. Assertions stay open.
- Result tab: response headers collapsed (already), request headers collapsed, bodies capped to about 20 lines with "expand".

## 4. Request views vs. step content

You asked that the request screen and the step content look similar but not identical. Proposal: one small read-only **`RequestSummary`** presentation (method + URL, headers, params, body in that order, same labels and colours as the editor sections) used by:

- the step `Result` tab (resolved request as sent, credentials masked, already produced by `exchangeRequest`);
- the main request screen's response area (the request that was sent).

Differences stay intentional: editors show templates (`{{baseUrl}}`), summaries show resolved values; summaries are read-only and copyable. An optional `Template | Resolved` toggle in the step `Result` tab is possible later, not needed now.

## 5. Phasing

| Phase | Delivers | Size |
|---|---|---|
| 1 | A (pinned Setup row) | Frontend only; move existing components into it, no format or runtime change |
| 2 | B + C (step toggle, calmer editor) | Frontend only; one state value per scenario view |
| 3 | `RequestSummary` shared with the request screen | Frontend only; refactor `StepExchange` |
| 4 (optional) | Run-from-here "values not set" warning | Frontend; needs the runner behaviour checked |
| Later, only if still crowded | Optional collapse of the step list, or Build/Run/Inspect modes from the earlier report | Reassess after 1-3 |

## 6. Requirement review

- Frontend-only; no Go, no serialized format change, no `format-spec.md` update, no new dependency.
- Architecture rules unaffected: the frontend still only renders what the Go core returns.
- Not a "major enhancement" by the CLAUDE.md definition if kept to phases 1-2, but this note exists so the direction is agreed before building.
- Existing tests (`ScenarioViewer.test.tsx`, `ScenarioStepResult`, `ScenarioDetailPane`, Playwright scenario specs) assert on the current layout and will need updating. Screenshots in `docs/images` that show the scenario screen need regenerating.

## 7. Risks

- Moving variables and coverage into Setup lowers their discoverability. Mitigation: the Setup row shows a variable count and a status dot, and errors also appear under the header.
- "Setup" could be mistaken for a step that sends requests. Mitigation: distinct icon, no number, no run status of its own.
- Auto-switching tabs can surprise. Mitigation: only auto-switch right after a run, never while typing.
- A failed run with the toggle on `Definition` could be missed. Mitigation: the "Last run" verdict line and the list status dot.

## 8. Open decisions

1. Auto-select `Result` after a run? *Recommend yes, for failed steps only.*
2. Go further with global modes now? *Recommend no; revisit after phases 1-2.*
3. Add the Run-from-here warning (phase 4)? *Recommend yes, after checking the runner.*

## 9. Decision log

| Date | Decision | Who |
|---|---|---|
| 2026-11-18 | Proposal written | Claude |
| 2026-11-18 | Replace the "Scenario details strip" with a pinned UI-only Setup row (not a real step, no format change); name it "Setup", not "Step 0"; defer global modes | Owner, Claude |
| 2026-11-18 | Empty scenario shows a UI-only "Start here" ghost row after Setup (not a real step, not saved); deleting the last step is allowed | Owner, Claude |
