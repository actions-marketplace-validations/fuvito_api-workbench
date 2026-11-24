# One "what next" mechanism: actions on findings

- **Status:** Proposed
- **Date:** 2026-11-24
- **Backlog:** B-091 (ideas 20 and 49 of `docs/reports/check-2026-11-24-improvement-ideas.md`)

## 1. Summary

Findings already say **what, where, why and how to fix** in words (`conformance.Finding`: `Summary`, `Where`, `Spec`, `Why`, `Fix`). Two places also offer a **button** that does the fix:

- the Workspace page's attention items carry a `Target{Kind, Path}` (`workspaceinfo.Attention`), and the page maps the kind to a button (open scenario, show environment file, open Contracts, show project file);
- the Check Conformance page has one hand-written button, "Add these to Variable overrides", for "not sent: no value for `{{id}}`".

The idea in the backlog was one shared `Action` type (`kind`, `label`, `target`) in the Go core, next to `Finding` and the attention items, so every finding that can have a button gets it from the core instead of from frontend special cases.

## 2. Requirement review (`CLAUDE.md` section 6)

| Question | Finding |
|---|---|
| Conflicts with a requirement? | No, and it supports rule 1: the knowledge "this finding is fixed by opening that file" would move from React into Go. |
| Larger than it looks? | The full version touches `conformance`, `workspaceinfo`, two Wails services, the Workspace page, the Check Conformance page, the scenario result view, and every report format if they print the action. |
| Necessary? | **Only partly.** Attention items already have `Target`. Findings have one real button today. The other candidate buttons (open the environment editor on an unset variable, pick a missing spec) have no place to land: the app has no environment editor action that takes a variable name, and a spec is chosen in a select that already shows the problem. Building a generic type for one button is the kind of abstraction `CLAUDE.md` says not to add before the task needs it. |
| Smaller alternative | Yes: section 4. |

## 3. Options

**A. The full type (the backlog row).** `action.Action{Kind, Label, Target}` in a new core package; `Finding.Actions` and `Attention.Actions`; every interface renders them; the CLI prints them as "Next:" lines. About a week with the page work, and the CLI "Next:" line duplicates the `Fix:` text already printed.

**B. Narrow: `Finding.Fix` stays text; add `Finding.Action *Action` for the cases that have a button today, and make `Attention.Target` the same type.** One small type (`kind`, `label`, `target`), two producers, one place in each page that renders it. About two days. Kinds today: `add-overrides` (names), `open-scenario`, `open-environment`, `open-contracts`, `open-project-file`.

**C. Do nothing.** Keep the hand-written button and the attention targets.

## 4. Recommendation

**B, later, and only when a second finding needs a button.** Today there is exactly one, so the generic type has nothing to generalise. The backlog row stays open with this note; it becomes worth building when either (1) the environment editor can open on a named variable, or (2) a second finding kind gets a button. Until then option C costs nothing and the words in `Fix` already tell the user what to do.

If you want it built now anyway, build B in this order: the shared `Action` type and the two existing producers (attention targets, the overrides button) so behaviour is unchanged and tests pin it; then add actions to findings one kind at a time.

## 5. Open decisions (owner)

1. Defer (C now, B when a second button is needed), build B now, or build A. **Recommend defer.**

## 6. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-24 | Proposal written; recommendation: defer | Claude, awaiting owner |
