# Spec Conformance Testing in the desktop app

**Status:** Proposed
**Date:** 2026-11-19
**Author:** Claude Code, for the project owner

## 1. Summary

Spec Conformance Testing is proving that a live API behaves as its OpenAPI spec declares. The product already does this, but the desktop app never says so.

- **Scenarios with `specFile`** validate every step's response against the matching operation (`validateSpec`, default true). A violation fails the step, but the Result tab only shows PASSED or FAILED. It does not say "7 of 7 responses match `openapi.yaml`".
- **`clrsig conform`** (`packages/core/conformance`) calls every operation in a spec and validates each response, with negative probes, coverage and baselines. It exists only in the CLI. The desktop has no way to start it.

The idea is that a user can answer yes to "is this conformance testing?" from the screen: the app runs the API with the spec beside it and reports how well the two agree.

## 2. Goals / non-goals

**Goals**
- Show a clear conformance result for any scenario run that has a spec attached.
- Let a desktop user run a full spec conformance check (the `conform` command) without a scenario or a terminal.
- Use the same Go code and the same report as the CLI, so desktop and CLI results are identical.
- Use the canonical terms "API Contract Validation" and "Spec Conformance Testing" in the UI.

**Non-goals**
- No change to scenario, request, environment or bundle file formats.
- No new conformance rules. What counts as a violation stays whatever `openapi.ValidateResponse` and `conformance.Runner` decide today.
- No CI or Action changes, and no new dependencies.
- No in-app editing of baselines in phase 1.

## 3. Requirement review

- **Architecture:** consistent. `conformance` is a core package with no Wails dependency; the desktop calls it through an application-level service, like the CLI. Nothing is implemented in the frontend (rule 1).
- **Locked stack / out-of-scope:** no new dependency; nothing from the out-of-scope list (no load testing or mock servers).
- **File format contract:** unchanged. Baseline files are an existing versioned format, and phase 2 only reads and writes them through existing code.
- **Size:** phase 1 is small (frontend plus a few fields on an existing result type). Phase 2 is a new UI area (a Conformance page), which is why this document exists.
- **Necessity:** the capability exists. This only makes it visible and reachable. The smaller alternative is phase 1 alone, which delivers the "yes, this is conformance testing" signal for scenarios. **Recommendation: build phase 1 now; build phase 2 once this proposal is accepted.**

## 4. Design

### Phase 1: conformance signal on scenario runs

When the scenario has a `specFile` (and at least one step validates against it), the Result tab of the Setup row and the scenario header show:

- A **Spec conformance** line: `7 of 7 responses match openapi.yaml`, or `5 of 7 match, 2 violations`.
- Under it, the violations grouped by step: operation, kind (status, header, body schema) and detail, using the messages the validator already produces.
- Steps with `validateSpec: false` show as "not checked" and are not counted as conforming. A step that failed on a network error shows as "not checked", not as a violation.

Without a spec the line is absent. The Result tab keeps showing the spec-file warning it shows today.

Data: `StepResultIO` already carries the step result. It gains additive fields `specChecked` (bool) and `specViolations` (list of `{kind, detail}`), filled from the validator output the runner already computes. No format change.

### Phase 2: "Check conformance" page

A singleton page tab (like the CLI Command Builder), opened from the Activity Rail or from the Contracts page. The user picks:

1. A spec (from the workspace or imported specs).
2. A base URL (default: the spec's `servers[0]`) and an environment.
3. Options that mirror the CLI: tags, path prefixes, "include negative probes", fail fast, request timeout.

Run executes `conformance.Runner` via a new `ConformanceService` (Wails-bound, a thin wrapper over the core). The result is a table per operation (passed, failed, skipped; expandable violations), a summary of the same four totals used on the Contracts page, and an **Export report** menu (html, junit, json, markdown) using `reporting.Conform*`. A **Get CLI command** button reuses the CLI Command Builder (`conform` is already in the catalog) so the user can repeat the check in CI.

Accumulated coverage feeds the existing coverage store, so the Contracts page reflects conformance runs.

### Security
- Credentials come from the selected environment and are redacted in results exactly as in scenario runs (secret environment values never shown).
- "Warn before destructive requests against production" applies: `conform` sends POST/PUT/DELETE, so the page shows a confirmation when the selected environment is marked production, or when the base URL is not localhost and the run includes write operations. This is the main safety risk of phase 2.
- No new network listener; IPC stays Wails-only.

### Dependency direction
Frontend → Wails `ConformanceService` → `conformance` and `openapi` core packages. No change to the core's dependencies.

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| 1 | Conformance line and violation list on scenario Result; step-level "not checked" states | A scenario with a spec against the demo server shows "N of N match"; breaking a response (changed field type) shows the violation under its step; a scenario without a spec shows no line |
| 2 | Conformance page and `ConformanceService`, export, coverage accumulation, production warning | Running the page against the demo server gives the same per-operation outcome and exit-equivalent summary as `clrsig conform` with the same options |
| 3 (optional) | Baseline support in the page (record, compare, show new and fixed) | Matches CLI `--write-baseline` / `--baseline` output |

## 6. Testing

- Go: `ConformanceService` tests against `examples/server` over `httptest`, including a deliberately wrong response; parity test comparing service output with the CLI path for the same options; production-warning rule.
- Vitest and React Testing Library: conformance line states (all match, violations, not checked, no spec); the page's options form and result table.
- Playwright: scenario Result shows the line (mock bridge gains the new fields); page flow with the mock bridge.

## 7. Documentation impact

- User guide: a "Spec Conformance Testing in the desktop app" section and a pointer from the Scenarios section.
- `format-spec.md`: no change (no serialized field changes).
- `README.md`: one line plus a screenshot in the gallery; `README-DEVELOPER.md`: service and package map.
- `CLAUDE.md` changelog entry when each phase ships; screenshots via the Playwright screenshot specs.

## 8. Risks

- **Destructive calls from the desktop.** `conform` really sends writes. Mitigation: the confirmation described above and a visible "will send N write requests" count before Run.
- **Two ways to do the same thing** (scenario with spec versus conform page) could confuse users. Mitigation: wording. A scenario is hand-written flows with the spec as a check; the page is "every operation in the spec".
- **Long runs block the UI.** Mitigation: run asynchronously with cancellation and progress events, as scenario runs do.
- **Result type growth.** Phase 1 adds fields to `StepResultIO`; keep them additive so existing frontend and tests keep working.

## 9. Open decisions

1. **Scope now?** Recommendation: approve phase 1 immediately; approve phase 2 with this document.
2. **Where does the page live?** Recommendation: Activity Rail entry next to Contracts, since both are about the spec.
3. **Negative probes in the desktop by default?** Recommendation: off by default, as in the CLI.
4. **Baselines (phase 3):** build now or wait for demand? Recommendation: wait.

## 10. Decision log

| Date | Decision | Who |
|---|---|---|
| 2026-11-19 | Proposal written after the owner asked how conformance testing works on the desktop | Claude Code |
