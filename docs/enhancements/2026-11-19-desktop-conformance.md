# Spec Conformance Testing in the desktop app

**Status:** Proposed
**Date:** 2026-11-19
**Author:** Claude Code, for the project owner

## 1. Summary

Spec Conformance Testing is proving that a live API behaves as its OpenAPI spec declares. The product already does this, but the desktop app never says so.

- **Scenarios with `specFile`** validate every step's response against the matching operation (`validateSpec`, default true). A violation fails the step, but the Result tab only shows PASSED or FAILED. It does not say "7 of 7 responses match `openapi.yaml`".
- **`clrsig conform`** (`packages/core/conformance`) calls every operation in a spec and validates each response, with negative probes, coverage and baselines. It exists only in the CLI. The desktop has no way to start it.

The idea is that a user can answer yes to "is this conformance testing?" from the screen: the app runs the API with the spec beside it and reports how well the two agree.

*Devin: Accurate, and the framing is right. One precision: today's scenario violation is only visible as a failed step; confirm the validator output is kept per step (not just a pass/fail flag) in `scenario.Runner`, since phase 1 depends on it.*

*Claude: Agreed, and checked. `runner.go` only appends a failed `spec conformance` assertion per violation; it keeps no checked or not-checked state. Phase 1 now starts with a `SpecCheck` on `StepResult` in the core.*

## 2. Goals / non-goals

**Goals**
- Show a clear conformance result for any scenario run that has a spec attached.
- Let a desktop user run a full spec conformance check (the `conform` command) without a scenario or a terminal.
- Use the same Go code and the same report as the CLI, so desktop and CLI results are identical. This requires moving the `conform` orchestration out of `apps/cli/main.go` first (phase 1.5).
- Scenario-with-spec results and the Conformance page share one violation display component.
- Use the canonical terms "API Contract Validation" and "Spec Conformance Testing" in the UI.

**Non-goals**
- No change to scenario, request, environment or bundle file formats (so no production flag on environments; see section 4, Security).
- No new conformance rules. What counts as a violation stays whatever `openapi.ValidateResponse` and `conformance.Runner` decide today.
- No CI or Action changes, and no new dependencies.
- No in-app editing of baselines in phase 1.

*Devin: Goals are clear. Consider adding a goal that scenario-with-spec and the Conformance page share one violation display component, to avoid two renderings of the same data. Non-goal "no format change" conflicts with the production warning in section 4 (see my comment there).*

*Claude: Agreed on the shared violation component, added as a goal. The non-goal and section 7 are now consistent because the production flag is dropped (see Security).*

## 3. Requirement review

- **Architecture:** consistent. `conformance` is a core package with no Wails dependency; the desktop calls it through an application-level service, like the CLI. Nothing is implemented in the frontend (rule 1).
- **Locked stack / out-of-scope:** no new dependency; nothing from the out-of-scope list (no load testing or mock servers).
- **File format contract:** unchanged. Baseline files are an existing versioned format, and phase 2 only reads and writes them through existing code.
- **Size:** phase 1 is small (frontend plus a few fields on an existing result type). Phase 2 is a new UI area (a Conformance page), which is why this document exists.
- **Necessity:** the capability exists. This only makes it visible and reachable. The smaller alternative is phase 1 alone, which delivers the "yes, this is conformance testing" signal for scenarios. **Recommendation: build phase 1 now; build phase 2 once this proposal is accepted.**

*Devin: I checked the claims. `reporting.ConformHTML/Markdown/JSON/JUnit` and `conformance.Runner/NegativeRunner/Baseline` exist as described. Two corrections to the review: (1) the `conform` orchestration (runner, negative probes, filters, baseline) is inline in `apps/cli/main.go` (and again in `mcp.go`), not in a shared package. The "same Go code, identical results" goal and the parity test need that logic extracted into an application-level function first, so this is larger than "a thin wrapper". (2) There is no `production` marker on environments (no hits in `domain`), so the "environment marked production" warning needs a new serialized environment field. That is a file-format change: it needs `format-spec.md`, a schema decision and migration notes, and contradicts "format-spec.md: no change" in section 7. Recommendation: either drop the flag and warn on non-localhost base URL plus writes only, or add the flag as its own small proposal.*

*Claude: Agreed on both, verified in code (`apps/cli/main.go:509`, `mcp.go:251`; `packages/core/application` is empty; no production marker in `domain`). Phase 1.5 added for the extraction, so this is not a thin wrapper. Production flag dropped, taking Devin's first option; no extra proposal needed.*

## 4. Design

### Phase 1: conformance signal on scenario runs

When the scenario has a `specFile` (and at least one step validates against it), the Result tab of the Setup row and the scenario header show:

- A **Spec conformance** line: `7 of 7 responses match openapi.yaml`, or `5 of 7 match, 2 violations`.
- Under it, the violations grouped by step: operation, kind (status, header, body schema) and detail, using the messages the validator already produces.
- Steps with `validateSpec: false` show as "not checked" and are not counted as conforming. A step that failed on a network error shows as "not checked", not as a violation.

Without a spec the line is absent. The Result tab keeps showing the spec-file warning it shows today.

Data: today `scenario.Runner` turns each violation into a failed assertion named `spec conformance` and keeps no record of whether a step was checked at all, so a match and an operation missing from the spec look identical. Phase 1 therefore starts in the core: `StepResult` gains a `SpecCheck` (state `matched`, `violations` or `not-checked`; a reason for `not-checked`: validation turned off, no matching operation in the spec, network error; the violations as `{kind, detail}`). The existing failed-assertion behaviour is unchanged. `StepResultIO` carries the same additive fields. No file format change.

Counting: the summary counts steps, one per step, using the result of the last attempt when a step is retried. A step whose operation is not in the spec is "not checked (no matching operation)", never a match or a violation.

*Devin: Phase 1 is solid and low risk. Define "matches" precisely: a step with `validateSpec` true but no matching operation in the spec should be "not checked" with a reason, not a violation or a match. Also decide whether the summary counts steps or responses when a step is retried.*

*Claude: Agreed. "Not checked (no matching operation)" is now its own state, and the summary counts steps using the last attempt of a retried step. Note phase 1 is no longer frontend-only: it needs the small core change to the runner.*

### Phase 1.5: extract the `conform` orchestration

Move runner, negative probes, filters and baseline handling from `apps/cli/main.go` (and the duplicate in `mcp.go`) into an application-level function under `packages/core/application`, taking an options struct and returning the result, coverage and negative result. The CLI and MCP become thin callers; behaviour and defaults do not change. Existing CLI tests are the regression guard. This is the first task of phase 2 and the reason the parity test is meaningful.

### Phase 1.6: Dev Servers menu and the Conformance Server

The Help menu's "Demo Server" submenu becomes **Dev Servers**, with an entry pair per server (Start/Stop, API Spec), so more servers can be added later. A second server, the **Conformance Server**, is a deliberately imperfect API with its own OpenAPI spec (`help/conformance-openapi.yaml`): four operations conform and three break the spec in different ways (`getProduct`: price is a string; `listCategories`: required `name` missing; `listOrders`: `id` is a string). A Go test pins the exact set of violations, so later conformance tests can assert "these three, no others". It opens no workspace and has no state shared with the Demo Server; it runs on its own port (from 8120) with a status bar chip. Fixed defects per operation are enough, since `conform` calls each operation independently; per-defect switches were dropped.

Finding: `openapi.ValidateResponse` does not flag a wrong response `Content-Type` or a missing declared response header (it only reads the content type to choose the schema). The server therefore has no such defect; changing the validator is a non-goal here and would be its own proposal.

### Phase 2: "Check conformance" page

Delivered in two steps. **2a:** spec, base URL, environment, run, results table, four totals. **2b:** options (tags, prefixes, negative probes, fail fast, timeout), export, Get CLI command.

A singleton page tab (like the CLI Command Builder), opened from the Activity Rail or from the Contracts page. The user picks:

1. A spec (from the workspace or imported specs).
2. A base URL (default: the spec's `servers[0]`) and an environment.
3. Options that mirror the CLI: tags, path prefixes, "include negative probes", fail fast, request timeout.

Run executes the shared conformance function (phase 1.5) via a new `ConformanceService` (Wails-bound, a thin wrapper over that function). The result is a table per operation (passed, failed, skipped; expandable violations), a summary of the same four totals used on the Contracts page, and an **Export report** menu (html, junit, json, markdown) using `reporting.Conform*`. A **Get CLI command** button reuses the CLI Command Builder (`conform` is already in the catalog) so the user can repeat the check in CI.

Accumulated coverage feeds the existing coverage store only for unfiltered runs (no tag or prefix filter), and the page says so; a filtered run is reported but not recorded, so it cannot skew the Contracts totals.

*Devin: Scope concern: this page adds several controls (tags, prefixes, negative probes, fail fast, timeout, env, base URL, export, CLI command, coverage). Suggest shipping 2a with spec, base URL, environment, run and results only, and adding options and export afterwards. The coverage feed should not be silent: a run limited by tag filters would skew accumulated coverage, so say what is recorded or only record unfiltered runs.*

*Claude: Agreed. Split into 2a and 2b as suggested. Coverage is recorded only for unfiltered runs and the page says so.*

### Security
- Credentials come from the selected environment and are redacted in results exactly as in scenario runs (secret environment values never shown).
- "Warn before destructive requests against production" applies. There is no production marker on environments and adding one is a file-format change, so this proposal does not add one. Instead:
  - The page runs **read-only operations (GET/HEAD) by default**; writes (POST/PUT/PATCH/DELETE) are an explicit opt-in checkbox showing "will send N write requests" before Run.
  - A non-localhost base URL is always shown and must be confirmed before Run; a base URL taken from the spec's `servers` is never used silently when it is non-local.
  - Redaction also covers negative-probe payloads in the page and in exports.
  - The CLI keeps its current behaviour; only the desktop default is read-only, set by the caller in the options struct, not in the shared function.
- No new network listener; IPC stays Wails-only.

*Devin: Good that this is explicit. Add: sending writes needs a pre-run count (stated in Risks), and the base URL should never be taken from the spec silently when it points to a non-local host. Show it and require it to be confirmed. Redaction should also cover the negative-probe payloads in exports.*

*Claude: Agreed on all three: write count before Run, non-local base URL shown and confirmed, redaction of negative-probe payloads in exports.*

### Dependency direction
Frontend → Wails `ConformanceService` → `packages/core/application` conformance function → `conformance` and `openapi` core packages. The CLI and MCP call the same application function.

*Devin: Matches the architecture rules only once the `conform` orchestration is extracted from `apps/cli/main.go` into an application service under `packages/core/application`, called by both the CLI and the Wails `ConformanceService`.*

*Claude: Agreed; the application function is now phase 1.5 and the diagram names it.*

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| 1 | Conformance line and violation list on scenario Result; step-level "not checked" states | A scenario with a spec against the demo server shows "N of N match"; breaking a response (changed field type) shows the violation under its step; a scenario without a spec shows no line |
| 1.6 | Dev Servers menu and the Conformance Server | Start from the menu; `clrsig conform` against it reports exactly the three known violations |
| 1.5 | `conform` orchestration extracted to `packages/core/application`; CLI and MCP call it | All existing CLI conform tests pass unchanged; no flag or output change |
| 2 (2a, then 2b) | Conformance page and `ConformanceService`, read-only default with write opt-in, base-URL confirmation, export, unfiltered-run coverage accumulation | Running the page against the demo server gives the same per-operation outcome and exit-equivalent summary as `clrsig conform` with the same options |
| 3 (deferred) | Baseline support in the page, only on demand | Matches CLI `--write-baseline` / `--baseline` output |

*Devin: Phase 2 acceptance is good (parity with the CLI). Phase 2 depends on the extraction work, so list it as an explicit phase 1.5 or a first task of phase 2. Phase 3 may be dropped given the recommendation to wait.*

*Claude: Agreed; phase 1.5 is an explicit row and phase 3 is marked deferred.*

## 6. Testing

- Go: runner `SpecCheck` states (matched, violations, validation off, operation not in spec, network error, retried step); cancellation mid-run; `ConformanceService` tests against `examples/server` over `httptest`, including a deliberately wrong response; parity test comparing service output with the CLI path for the same options; read-only default and write opt-in count.
- Vitest and React Testing Library: conformance line states (all match, violations, not checked, no spec); the page's options form and result table.
- Playwright: scenario Result shows the line (mock bridge gains the new fields); page flow with the mock bridge.

*Devin: Add a test for the not-checked cases and for cancellation mid-run. The parity test is only meaningful if both paths call the same function, so it doubles as a guard against drift.*

*Claude: Agreed; both added to section 6.*

## 7. Documentation impact

- User guide: a "Spec Conformance Testing in the desktop app" section and a pointer from the Scenarios section.
- `format-spec.md`: no change (no serialized field changes; the production flag was dropped).
- `README.md`: one line plus a screenshot in the gallery; `README-DEVELOPER.md`: service and package map.
- `CLAUDE.md` changelog entry when each phase ships; screenshots via the Playwright screenshot specs.

*Devin: `format-spec.md` is only unchanged if the production flag is dropped. The ancillary docs (AI prompt, which derives from the spec) are unaffected.*

*Claude: Agreed; the flag is dropped, so the line stays true.*

## 8. Risks

- **Destructive calls from the desktop.** `conform` really sends writes. Mitigation: read-only by default, opt-in for writes with a visible "will send N write requests" count, and confirmation of a non-local base URL.
- **Two ways to do the same thing** (scenario with spec versus conform page) could confuse users. Mitigation: wording. A scenario is hand-written flows with the spec as a check; the page is "every operation in the spec".
- **Long runs block the UI.** Mitigation: run asynchronously with cancellation and progress events, as scenario runs do.
- **Defaults diverging between CLI and desktop.** Mitigation: all defaults live in the shared function; the desktop only passes explicit overrides (read-only), and the parity test uses the same options on both paths.
- **Result type growth.** Phase 1 adds fields to `StepResultIO`; keep them additive so existing frontend and tests keep working.

*Devin: Add risks: (a) the CLI and desktop could diverge on defaults (timeout, probe set) unless defaults live in the shared function; (b) a base URL pointing at a real server with writes enabled is the dominant risk, so default to read-only operations (GET/HEAD) with an explicit opt-in for writes, which is safer than only warning.*

*Claude: Agreed on both, with one objection of detail: read-only is the desktop default, not the shared function's default. Changing the CLI's `conform` default to skip writes would be a breaking behaviour change for existing CI users, so the desktop sets it explicitly in the options it passes.*

## 9. Open decisions

1. **Scope now?** After Devin's review: approve phase 1 now; phase 1.5 and phase 2 once the owner accepts this revised document.
2. **Where does the page live?** Recommendation: Activity Rail entry next to Contracts, since both are about the spec.
3. **Negative probes in the desktop by default?** Recommendation: off by default, as in the CLI.
4. **Baselines (phase 3):** build now or wait for demand? Recommendation: wait.
5. **Read-only by default on the desktop, writes opt-in?** Recommendation: yes (added after Devin's review).

*Devin: Answers: (1) approve phase 1 now; hold phase 2 until the production-flag question and the extraction are settled. (2) Activity Rail next to Contracts is fine. (3) Agree, off by default. (4) Agree, wait. Suggested new decision 5: read-only by default with opt-in for writes.*

*Claude: Accepted. Decision 5 added with my recommendation (yes). One note: phase 1 can proceed now, but it includes the small runner change described above, not just frontend work.*

## 10. Decision log

| Date | Decision | Who |
|---|---|---|
| 2026-11-19 | Proposal written after the owner asked how conformance testing works on the desktop | Claude Code |
| 2026-11-19 | Devin's review accepted: verified in code that `conform` is inline in `apps/cli/main.go` and `mcp.go`, the runner keeps no checked flag, and environments have no production marker. Added phase 1.5 (extraction), dropped the production flag in favour of a read-only default, split phase 2 into 2a and 2b | Claude Code |
