# Severity levels, stable check IDs and the baseline key (0.9.0)

- **Status:** In progress (phases A to D built; phase E prepared 2026-11-24: version, in-app notes and draft release notes; not tagged)
- **Date:** 2026-11-24
- **Backlog:** B-049 (check IDs), B-050 (severity), B-053 (format validators), B-006 and B-007 (baselines)
- **Decisions already taken (step 0, `docs/reports/check-2026-11-24-step0-decisions.md`):** D1 baseline key = check ID + operation + short detail; D2 three levels (failure, warning, note) with `--fail-on warning`; D3 migration policy first (`docs/process/schema-migration-policy.md`, draft).

## 1. Summary

Three problems share one root: a finding has no stable identity.

1. **Findings are not addressable.** The catalog (`help/what-we-check.md`) has IDs such as `CS-RESP-005`, and `conformance.checkIDFor` maps a finding to one, but the mapping is a `switch` in code, the catalog is Markdown, and nothing forces the two to agree beyond a test that cites test names. A new check can ship without an ID.
2. **Every finding fails the run.** To report "the response has a property the spec does not declare" a team must also accept a red build, so those checks are not built (CS-GAP-002, CS-GAP-003).
3. **Baselines are keyed by validator wording.** `conformance.Violation` is `operation + kind + detail`, where `detail` is the library's message text (`v.Error()`). A wording change, a new kind, or an upgrade of `kin-openapi` silently turns every known violation into a "new" one and fails CI (B-007, B-049).

The idea: make the catalog **data in Go** (ID, title, default severity); give every finding a **severity**; key baselines on **check ID + operation + where**, with the wording only as a readable summary. Ship the format validators (B-053) in the same release, because they are the first new check that benefits from all three and the most likely to turn existing CI red.

## 2. Goals and non-goals

**Goals**
- One list of checks in Go; the code cannot emit an ID that is not in it, and the Markdown catalog cannot list one that is not.
- `failure`, `warning`, `note` on every finding; only failures change the exit code; `--fail-on warning` promotes warnings.
- Baselines that survive wording changes; a v1 baseline keeps working (no silent mass-failure on upgrade).
- Format validation for the formats the OpenAPI specification names and the common ones.
- No change in what today's checks decide: every existing check stays a failure in 0.9.0.

**Non-goals**
- Searchable "What We Check" table in the Help panel (follow-up of B-049; it becomes possible once the catalog is data).
- Changing `diff`, `impact` or the `schema` assertion (they keep their own vocabularies; see section 8, question 3).
- A per-check configuration file to change severities (a later option; the project config, B-089, is the natural home).
- New negative probes, report-all-problems, spec lint (separate backlog items).

## 3. Requirement review (`CLAUDE.md` section 6)

| Question | Finding |
|---|---|
| Conflicts with a requirement? | No. Core logic stays in Go (rule 1); the desktop page and CLI both read the same data. A new package `packages/core/checks` depends on nothing (rule 2). No new dependency: formats use the validation library already in `go.mod`. |
| Changes a serialized format? | **Yes: the baseline file** (breaking: the key changes), additive fields in the JSON report. `format-spec.md` and the migration policy apply (section 5.3). |
| Larger than it looks? | Yes: it touches `conformance`, `openapi`, `reporting` (five formats), `application`, the CLI, the MCP tool, the Action, the builder catalog and the desktop page. It is phased so each phase ships green (section 6). |
| Unnecessary? | A smaller option is to do only the baseline key. It fixes the silent baseline breakage but leaves the catalog and severity gaps, and the format validators would then ship as 100% failures with no softer level. Rejected because the three parts reuse one structure. |
| Behaviour change risk | Formats will turn some green CI red (like B-002 did). Handled by release notes and the baseline path, and by keeping everything else unchanged. |

## 4. Design

### 4.1 The check catalog as data (`packages/core/checks`)

```go
type Severity string // "failure" | "warning" | "note"

type Check struct {
    ID       string   // "CS-RESP-005"
    Title    string   // "A required field is missing"
    Severity Severity // default severity
    Area     string   // "response", "request", "run", "probe", "coverage", "not-checked"
}

func All() []Check
func ByID(id string) (Check, bool)
```

- Holds the checks that produce **findings** (CS-RESP, CS-REQ, CS-RUN, CS-PROBE) and the **not checked** notes (CS-NC). CS-COV, CS-DIFF, CS-IMP and CS-GAP rows stay documentation only; they have no per-run finding.
- `conformance.checkIDFor` keeps its job (finding kind to ID) but returns an ID that must exist in the catalog; `Finding` gains `Severity`, read from the catalog (never set by the caller).
- **Guard tests**, in the style of `TestCatalogMatchesCobra`:
  1. every ID the code can return exists in `checks.All()`;
  2. every ID in `checks.All()` has a row in `help/what-we-check.md`, and every `CS-` ID in that file's tables with a finding area exists in the data;
  3. IDs are unique and follow the scheme.
- The Markdown stays hand-written in 0.9.0 (it carries examples and prose). Generating it, or rendering it as a table by ID, is the follow-up.

### 4.2 Severity

- **Default severity is per check**, from the catalog. In 0.9.0 **every existing check is `failure`**, so nothing that passes today fails and nothing that fails today passes.
- **New checks introduced as `warning`** (they are what the levels are for): response properties the spec does not declare on schemas that allow them (CS-GAP-002), response headers the spec does not declare (CS-GAP-003), a `deprecated` operation exercised. Whether these ship in 0.9.0 or a later release is open question 4; the mechanism ships either way.
- **`note`** is for information that is neither a pass nor a problem: today's "not checked" outcomes (CS-NC), shown in reports but never counted as a failure.
- **Exit code.** Unchanged for `0` (pass), `1` (failures), `2` (could not run). New: `clrsig conform --fail-on failure|warning` (default `failure`). With `warning`, a run with only warnings exits `1`. `note` never fails.
- **Counts.** An operation with only warnings is **Passing** on the four-tile totals (Passing + Failing = Covered stays true) and carries a warning marker; the totals gain a separate "Warnings" count in the page, text and JSON, not a fifth tile that changes the invariants.
- **Where levels show.**
  - Text: `WARNING` and `NOTE` lines under the operation; the headline says `PASSED with 3 warnings`.
  - JSON: additive `severity` on each finding and a `warnings` count; `passed` stays true for warning-only runs unless `--fail-on warning`.
  - HTML and Markdown: a level chip; JUnit: warnings are passing test cases with the finding in `system-out` (JUnit has no warning status), failures as today.
  - Desktop: a warning chip, and the Check Conformance option "Fail on warnings" carried into Get CLI command.
- **Other entry points** carry the flag: MCP `conform` (`failOn`), the GitHub Action (`fail-on` already exists for `diff`; add `conform-fail-on`, see question 3), the CLI Command Builder catalog (`TestCatalogMatchesCobra` guards it).

### 4.3 Baseline key (breaking change of the baseline format, `schemaVersion: "2"`)

**Today** (`schemaVersion: "1"`): `operation`, `kind` (`request`, `response`, `error`, `probe`), `detail` (the library's message). Key = the three joined.

**v2:**

```yaml
schemaVersion: "2"
violations:
  - operation: GET /products/{id}      # method + path template, as today
    check: CS-RESP-006                 # the check ID (stable)
    where: price                       # field path, header name or status (stable, may be empty)
    rule: ""                           # only for checks with sub-rules, e.g. the constraint that broke
    summary: '"price" is text, the spec says number'   # readable, NOT part of the key
```

- **Key** = `operation | check | where | rule`. Wording, the library's message and the order of problems no longer matter.
- Findings the package does not recognise (`other`, no ID) use `check: other` and the raw detail as `rule`, which is today's behaviour for those: they are the only wording-sensitive entries left, and they are marked as such.
- Probes: `check` is the probe's check ID (CS-PROBE-00x); `where` is the probe target; `rule` the probe's description as today.
- Secrets are masked in `summary` and `where` exactly as in v1 (the same redaction step).
- **No timestamps**; sorting and de-duplication as today, so an unchanged run writes an identical file.

**Reading a v1 baseline (in-memory upgrade, per the policy):** each violation of the current run carries both its v2 key and its **legacy v1 key** (computed the way v1 computes it, from the same raw message). A v1 entry matches a current violation by legacy key. Matched entries count as known; unmatched v1 entries are "fixed". The run then notes once on stderr: `baseline uses the old format (schemaVersion 1); it still works, run with --write-baseline to upgrade it`. `--write-baseline` writes v2. Nothing is rewritten without that flag (policy section 3).

This is the only moment where wording can still matter: if the validator wording changed between the run that wrote the v1 file and the current run, the v1 entries do not match. That risk exists today and ends with v2. The release notes say to regenerate baselines once after upgrading.

**Reading v2 with an older clrsig:** refused with the existing "schemaVersion not supported" message (what / why / fix: upgrade clrsig).

### 4.4 Format validators (B-053)

- Register validators for: `email`, `uuid`, `uri`, `uri-reference`, `hostname`, `ipv4`, `ipv6`, `date`, `time`, `byte`, `binary` where the library does not already do it (`date-time` already works), using the library's own registration, once, in `openapi`. Unknown formats stay ignored, as the specification allows, and are not findings.
- A mismatch is a **constraint** finding (CS-RESP-007 for responses, the request equivalent for generated requests) with `where` = field and `rule` = `format: email`. The finding names the format and the value, never anything outside the response.
- Requests: values come from the environment and `{{placeholders}}`; a placeholder that is not filled is already "not sent", so the format check only sees filled values.
- CS-GAP-001 moves to "caught" in the catalog; the supported list is printed in the catalog.
- Severity: `failure` (the spec says the value must be an email). This is the release's main behaviour change, hence the adoption path below.

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| **A. Catalog as data** | `packages/core/checks`; `Finding.Severity`; the three guard tests; `checkIDFor` returns catalog entries | Removing an ID from the data or the Markdown fails a test; JSON findings gain `severity` (all `failure`); no report text changes |
| **B. Severity** | `--fail-on`, counts, headlines, five report formats, page chip and option, MCP, Action, builder catalog | A fixture with one failure and one warning exits 0 by default and 1 with `--fail-on warning`; every report shows the level; the four tiles' invariants hold in a test |
| **C. Baseline v2** | v2 format, v1 read path, `--write-baseline` upgrade, desktop page uses the same code | A v1 baseline written by 0.8.9 against the Conformance Server still passes with 0.9.0; the same run with a changed validator message (test hook) still matches v2; an unchanged run writes an identical v2 file |
| **D. Formats** | validators, catalog rows, release note | `"mail": "nope"` fails with `CS-RESP-007`, `where: mail`, `rule: format: email`; valid values pass; unknown formats pass |
| **E. Docs and release** | docs below, 0.9.0 notes | See section 7 |

Phases A to C can ship as separate commits on `main` with CI green after each; D may ship in the same release or the next (question 4). Do not mix C and D in one release note section: the first is "your baselines were upgraded", the second is "new checks can turn CI red".

## 6. Testing

- **Unit:** catalog guard tests; severity per finding; `Violations` v2 keys; legacy key computation matches v1 exactly (golden: the 0.8.9 output for the Conformance Server's six defects, stored under `testdata/`); format validators table test (valid and invalid per format).
- **Integration (real local servers, no mocks):** the Conformance Server run with a v1 and a v2 baseline (known, new, fixed); `--fail-on warning` using a server that sends an undeclared header (if CS-GAP-003 ships) or a test-only check marked `warning`.
- **Parity:** `TestConformanceParity_ServiceMatchesTheCLI` extended with severity and check IDs; `TestCatalogMatchesCobra` for `--fail-on`.
- **Frontend:** Vitest for the warning chip and the option; Playwright for the page with one warning and one failure.
- **Guards that exist today and must stay green:** `what_we_check_doc_test.go`, the format-spec and AI-prompt tests.

## 7. Documentation impact

- `help/format-spec.md`: the baseline file (v2), a "Changes between versions" section for the baseline, the JSON report fields `severity` and `warnings`.
- `help/what-we-check.md`: severity column, the format list, CS-GAP-001 moved to caught, new warning rows if shipped.
- `help/user-guide.md`: severity levels, `--fail-on` for `conform`, adopting with a baseline after upgrading.
- `docs/guides/ci-integration.md`: the Action input; a "upgrade to 0.9.0" paragraph (regenerate baselines once; formats may turn a green job red).
- `docs/process/schema-migration-policy.md`: add the **machine-written files** clause (question 2) after review.
- `CLAUDE.md` changelog entry; `AboutDialog.tsx` 0.9.0 notes; backlog ticks for B-049 (partly: the searchable page remains), B-050, B-053, B-006 and B-007.

## 8. Risks

| Risk | Mitigation |
|---|---|
| Formats turn green CI red | Release note names it first; `--write-baseline` / `--baseline` adoption path; the Action's `baseline` input exists. |
| Two entries of one check at one place collapse into one baseline entry (key has no message) | `rule` carries the sub-rule where one exists; documented limit: a baseline marks "this check at this place" as known. |
| v1 entries stop matching if wording differed between writing and reading | Same as today; ends after `--write-baseline`; the stderr note tells the user. |
| `Severity` becomes a second place to forget | Severity is read from the catalog only; a guard test fails if `Finding.Severity` is empty for an ID. |
| Warnings nobody reads | Headline counts them (`PASSED with 3 warnings`); `--fail-on warning` is one flag; the Action input exposes it. |
| Scope creep into `diff`/`impact` | Explicit non-goal; `diff` keeps its own `--fail-on` values (breaking, warning, info). |

## 9. Open decisions (owner), with recommendations

1. **Warning semantics for existing checks.** Keep all of them `failure` in 0.9.0? **Recommend yes**: no existing outcome changes except through formats.
2. **Machine-written files and `clrsig migrate`.** The policy asks for a `migrate` command for a breaking change. A baseline is regenerated by `--write-baseline` and is not hand-edited. **Recommend** adding a clause to the policy: for machine-written, regenerable files the documented migration is "regenerate with the same command", no `migrate` command; the in-memory v1 read path stays. Hand-written files (project config, scenarios) still need `clrsig migrate`.
3. **Flag naming.** `diff --fail-on` takes `breaking|warning|info`. `conform --fail-on` would take `failure|warning`. **Recommend** these names (the levels are the ones in reports) and a sentence in the guide; do not rename `diff`'s. For the Action, a separate `conform-fail-on` input avoids reinterpreting the existing `fail-on` (used by `diff`).
4. **Which new warnings in 0.9.0.** Options: (a) none, mechanism only plus formats; (b) also undeclared response headers and undeclared properties on open schemas. **Recommend (a)**: one behaviour change per release; (b) ships in 0.9.1 with the levels already known to users.
5. **Format list and strictness.** Include `uri` and `hostname` (the library's own checks differ from some servers' expectations)? **Recommend** shipping `email`, `uuid`, `date`, `ipv4`, `ipv6` first and the URL-like ones after a sample against the demo and conformance specs, as these have the most false-fail risk.
6. **Contracts store and baselines.** D6 (Contracts key) is independent and stays in step 3; baselines do not use the Contracts key.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-24 | Baseline key = check ID + operation + short detail (D1) | owner |
| 2026-11-24 | Three severity levels, `--fail-on warning` (D2) | owner |
| 2026-11-24 | Migration policy written before new persisted files (D3) | owner |
| 2026-11-24 | Proposal written; open decisions 1 to 6 above | Claude, awaiting owner |
