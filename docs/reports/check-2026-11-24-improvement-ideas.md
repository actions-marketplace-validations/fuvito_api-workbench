# Improvement ideas: features, UI/UX, quality — 2026-11-24

## Abstract

A read-through of the project at **v0.8.9 plus 5 commits** (head `d1e7197`), looking for what to build, fix or simplify next. It collects **38 ideas from the first pass (Devin) and 12 more from a second pass (Claude, section 10)** in six groups: conformance accuracy and coverage, CLI and CI, desktop UI/UX, onboarding and documentation, engineering health, and distribution and growth. Section 11 turns them into a phased plan with the decisions to take first; it supersedes the order in section 7.

The product's core (API Contract Validation and Spec Conformance Testing) is in good shape; the three themes that come out of the review are:

1. **Close the remaining "silent pass" and "false fail" gaps in conformance.** The catalog's open gaps (CS-GAP-001 to 011) and the sandbox findings are where users will lose trust first. Formats, warnings (a severity level below "failure"), and stable check IDs in reports matter more than new check types.
2. **Make the desktop app tell the user what to do next.** The Workspace page started this; the same idea (explain, then offer the fix button) can reach Contracts, the scenario run log and the empty states.
3. **Reduce the cost of adoption.** `clrsig init`, scenario selection by tag, a project config (idea 43) and `validate --strict` are all small, and each removes a first-hour obstacle. The biggest single obstacle found in the second pass is that `conform` against a secured API needs a token the user must fetch by hand (idea 39).

Every idea below is a **suggestion**, not a decision. Per `CLAUDE.md` section 6, anything marked *Proposal* needs a dated file in `docs/enhancements/` before code; anything marked *Decision* needs an owner decision first. Items that would conflict with the locked scope are listed separately in section 8 and are **not** proposed for the open-source core.

## Method and limits

- Read: `CLAUDE.md`, `docs/planning/backlog.md`, `docs/enhancements/` (including the future-enhancements and Postman-gap documents), `docs/enterprise/`, the reports `check-2026-11-20`, `check-2026-11-21` and `check-2026-11-22-sandbox-guide`, the catalog (`what-we-check.md`, section "known gaps"), the user guide's table of contents and shortcut table, and the layout of `packages/core`, `apps/cli` and `apps/desktop/frontend/src`.
- Ran: `clrsig --help` and `git log`. Counted file sizes and a few patterns in the frontend.
- **Not run:** the desktop app (no GUI session), Playwright, the test suites. UI/UX comments therefore come from the code structure, the user guide and the earlier audit reports, not from looking at the screen. Treat them as hypotheses to confirm with a quick look at the running app.
- Ideas that are already an open backlog item are tagged with the `B-###` ID so they are not duplicated; new ideas are tagged **NEW**.
- Size: **S** about a day or less, **M** a few days, **L** a week or more.

## Summary table

| # | Idea | Group | Size | Status |
|---|---|---|---|---|
| 1 | Register the missing format validators (`email`, `uuid`, `uri`, …) | Conformance | S–M | B-053 |
| 2 | Warning and note levels beside "failure" | Conformance | M | B-050 |
| 3 | Stable check IDs in every report | Conformance | M | B-049 |
| 4 | Generate bodies for XML request operations (or report "not sent: no body can be built") | Conformance | M | CS-GAP-006, sandbox finding |
| 5 | More negative probes (enum, boundaries, unknown fields, header and path types) | Conformance | M–L | CS-GAP-009 |
| 6 | Credential probes: wrong and expired tokens | Conformance | M | CS-GAP-008 |
| 7 | Report all problems of a response, not only the first | Conformance | M | CS-GAP-011 |
| 8 | Spec lint before conformance (the spec itself is wrong) | Conformance | M | NEW |
| 9 | Per-operation response-time budget declared in the spec (`x-clrsig-max-response-time`) | Conformance | S–M | NEW |
| 10 | Response-time trend over runs | Conformance | M | CS-GAP-005 |
| 11 | Webhooks and callbacks | Conformance | L | CS-GAP-010 |
| 12 | `clrsig init` | CLI and CI | M | B-021 |
| 13 | Select scenarios by tag or name | CLI and CI | M | B-022 |
| 14 | `validate --strict` and workspace attention items in `validate` | CLI and CI | S–M | B-020, B-071 |
| 15 | Parallel scenario execution | CLI and CI | M | B-023 |
| 16 | GitHub Action: inputs for the newer `conform` flags; job summary by default | CLI and CI | S | B-058 |
| 17 | `clrsig doctor` (environment self-check) | CLI and CI | S | NEW |
| 18 | Stable machine-readable `--list` / `--explain` for scenarios and operations | CLI and CI | S | NEW |
| 19 | Baselines in the desktop page, and one decision on baseline keys | UI/UX | M | B-006, B-007, B-049 |
| 20 | "Fix it for me" actions on findings | UI/UX | M | NEW |
| 21 | Run history view with compare-two-runs | UI/UX | M | NEW |
| 22 | Contracts: fix the duplicate card for URL specs, show a trend | UI/UX | S–M | B-010, B-017 |
| 23 | Accessibility pass (focus order, labels, contrast, reduced motion) | UI/UX | M | NEW |
| 24 | Keyboard shortcuts for the newer pages and a shortcut sheet | UI/UX | S | NEW |
| 25 | Command palette (extend Ctrl+P beyond opening files) | UI/UX | M | NEW |
| 26 | Better empty states and first-run path | UI/UX | S–M | NEW |
| 27 | Run log: collapse passed steps, jump to first failure, copy as cURL | UI/UX | S | NEW |
| 28 | Wording and naming consistency (U2 to U11) | UI/UX | S | B-067 to B-070 |
| 29 | Screenshots in the user guide; task-based recipes | Docs | M | B-013 |
| 30 | Retire "contract testing" wording in the user guide; terminology check | Docs | S | NEW |
| 31 | Split `App.tsx` (1,619 lines) | Engineering | M | NEW |
| 32 | Generated bindings and types stay in sync (CI check) | Engineering | S | NEW |
| 33 | Close the Low findings of the 0.6.0 check (hardening batch) | Engineering | S–M | B-061 to B-064, R3 to R9 |
| 34 | Contracts store key collisions | Engineering | S | B-080 |
| 35 | Define what "1.0" means (migration policy, frozen flags) | Project | S | B-037 |
| 36 | Failure-catalog repository wired to CI | Growth | L | B-036 |
| 37 | Community scaffolding (`CONTRIBUTING.md`, templates) | Growth | S | B-032 |
| 38 | Hosted demo server and code signing (decisions) | Distribution | M–L | B-031, B-035 |

---

## 1. Conformance accuracy and coverage

The catalog says what is checked; the gaps below are what a user can still be surprised by. A surprise here is worse than a missing feature: a green run that should be red, or the reverse.

### 1. Format validators (B-053) — S–M

`format: email`, `uuid` and others are not validated; `"mail": "nope"` passes (CS-GAP-001). The sandbox guide shows several problems of this shape.
- **Idea.** Register the formats the OpenAPI specification lists plus the common ones (`email`, `uuid`, `uri`, `ipv4`, `ipv6`, `date`, `hostname`), and print the supported list in the catalog.
- **Watch for.** This turns some existing green CI red, like B-002 did. Say so in the release notes and point to `--write-baseline`.

### 2. Warning and note levels (B-050) — M

Today every finding fails the run. That forces a choice between "report undeclared properties" and "keep CI green".
- **Idea.** Three levels: *failure* (changes the exit code), *warning* (shows, counts, no exit code unless `--fail-on warning`) and *note* (information, such as "media type not checked"). `diff` already has `--fail-on`; reuse the vocabulary.
- **Unlocks.** CS-GAP-002 (extra properties on open schemas), CS-GAP-003 (undeclared response headers), CS-NC-009 as visible notes, `deprecated` operations still in use.

### 3. Stable check IDs in every report (B-049) — M

The catalog has IDs (`CS-RESP-003`); the reports do not print them. A reader cannot jump from a failing line to the explanation.
- **Idea.** One Go catalog the reports and the page both read; print the ID next to every finding in text, JSON, HTML, Markdown and JUnit; make the desktop ID a link to Help, What We Check.
- **Decide together.** B-007 and B-049's baseline key: both change how a violation is identified, so decide once.

### 4. XML request bodies — M

Operations with an XML body are sent with **no body**, and the report says "the request body does not fit the spec: value is required but missing" (sandbox review, item 13). That is a tool limit presented as a server fault.
- **Minimum (S).** Report it as "not sent: no XML request body can be built" in the same outcome family as `not sent: no value for {{id}}`.
- **Fuller (M).** Generate XML from the schema (`xml.name`, attributes, wrapped arrays) for the simple cases and mark the rest as not sent.

### 5. More negative probes (CS-GAP-009) — M–L

Only four kinds exist. Candidates in rough order of value: enum violation, number boundary (`minimum`/`maximum`), string length, unknown field when `additionalProperties: false`, wrong type in a path or header parameter.
- **Note.** The sandbox guide recommends a scenario with one step per bad token for security cases; keep that advice even when probes exist.

### 6. Credential probes (CS-GAP-008) — M

Only "no credentials" is probed. Add an option for "wrong credentials" (a deliberately bad token for the declared scheme), expecting `401`/`403` and no data. Expired and under-privileged tokens need the user to supply them, so offer `--probe-token name=${ENV}` rather than guessing.

### 7. All problems per response (CS-GAP-011) — M

The validation library stops at the first problem, so the loop is "fix, re-run, repeat". Walking the schema in the core to collect all body problems would shorten that loop, and the Check Conformance page already has a failure-first layout that can show several.

### 8. Spec lint before conformance — M (NEW)

Several guide problems (P20 to P24) are caused by the **spec** being wrong, not the server (for example an `integer` id that is really a UUID). `conform` currently blames the response.
- **Idea.** A cheap pre-pass, `clrsig lint <spec>` and a "Spec health" card on the page: missing `operationId`, duplicate `operationId`, examples that do not validate against their own schema, `required` listing properties that do not exist, a path parameter not declared. Run it first so the user fixes the spec before reading 40 response violations.
- **Requirement review.** New command, so it needs a short proposal. It fits the product's identity (spec conformance) and needs no new dependency.

### 9. Response-time budget declared in the spec — S–M (NEW)

`--max-response-time-for /prefix=ms` fixes the global-limit problem but keeps the budget on the command line, away from the contract.
- **Idea.** Read an optional `x-clrsig-max-response-time` on an operation or path item. The command-line value wins when both exist.
- **Notes.** Extension fields are already in the "preserve unknown extension fields" rule of section 8 of `CLAUDE.md`; document it in the catalog.

### 10. Response-time trends (CS-GAP-005) — M

The Contracts store already keeps the last and slowest time per operation. A trend needs a short history per operation (last N, median). Render it as a sparkline on the Contracts page. The proposal `2026-11-22-response-time-assertions.md` lists trends as a non-goal, so this needs its own small proposal and should wait for demand.

### 11. Webhooks and callbacks (CS-GAP-010) — L

Operations outside `paths` are not exercised. Receiving callbacks needs a listener, which touches the "no local control server" rule (`CLAUDE.md` section 3, item 5). **Recommendation: skip for now**; revisit only with a design that opens no unauthenticated port.

---

## 2. CLI and CI

### 12. `clrsig init` (B-021) — M

Creates `scenarios/`, `environments/`, `requests/` and prints a workflow file by default; writes it only with an explicit flag. It should share its snippets with the CLI Command Builder (B-015). First-hour value for every new user.

### 13. Select scenarios by tag or name (B-022) — M

`--tag smoke`, `--name pattern`. Needs an optional `tags` field in the scenario file: a serialized-format change, so `format-spec.md` and the AI prompt update in the same commit. `conform` already has `--tags` for spec tags; keep the two names clearly separate in help text (spec tag versus scenario tag).

### 14. Strict validation (B-020, B-071) — S–M

Warn on steps with no assertions, extractions never used, unresolved `requestRef`, variables defined nowhere. B-071 asks for the same attention items the Workspace page shows. Build both on `packages/core/workspaceinfo` so the page and the CLI cannot disagree (rule 1: the Go core is the product).

### 15. Parallel scenario execution (B-023) — M

Useful once folder runs exist (they do). Default stays sequential; `--parallel N` for independent scenarios. Report order must stay deterministic. Warn when scenarios share a workspace environment that they mutate.

### 16. GitHub Action polish (B-058) — S

The Action has no inputs for the newer `conform` flags (`--max-response-time-for`, `--read-only`, probes). Add them, and append the Markdown report to `$GITHUB_STEP_SUMMARY` by default. Pin the examples in the guide to a version rather than `latest`.

### 17. `clrsig doctor` — S (NEW)

One command that checks what usually goes wrong on a new machine or CI runner: spec readable, environment file parses, `${NAME}` variables set (names only, never values), the server answers on the base URL, TLS/proxy settings, git present for `diff --against`. Output uses the same what / why / fix wording as findings. This mirrors the Workspace page for people who never open the desktop app.

### 18. Machine-readable listings — S (NEW)

`conform --list` (operations that would be sent and which would be skipped, with the reason) and a JSON form of `--explain`. Helps CI authors and the MCP server (`clrsig mcp`), and gives the builder a data source instead of text parsing.

---

## 3. Desktop UI/UX

The app already has a rail, sidebars, tabs, quick open, a Workspace page and a CLI builder. The ideas below add guidance and polish rather than new areas. All of them should be confirmed against the running app first (see Method).

### 19. Baselines in the page (B-006, B-007, B-049) — M

The CLI has `--write-baseline`/`--baseline`; the page has nothing. A user who adopts through the desktop cannot reproduce the CI behaviour. Add "Record baseline" and "Compare with baseline" (new / fixed / unchanged). This is the single largest CLI-versus-desktop gap in conformance. Decide the key (B-049) first.

### 20. "Fix it for me" actions on findings — M (NEW)

The page already has one such button (add missing names to Variable overrides). Generalise:
- *Not sent: no value for `{{id}}`* → add to overrides (exists).
- *Environment variable unset* → open the environment editor on that variable.
- *Spec file missing* → file picker.
- *Violation is a wrong spec example* → open the spec at the line.
Each finding already has "where" and "fix" text; map the common ones to a button.

### 21. Run history and compare two runs — M (NEW)

`packages/core/history` stores runs, and the response viewer has a diff. A "History" view for scenarios and conformance runs: list, filter by status, and compare two runs side by side (new failures, fixed, slower). It reuses `ResponseDiffView` ideas and gives the baseline concept a visual home.

### 22. Contracts page (B-010, B-017) — S–M

- Fix the duplicate card for URL specs (B-010): two cards with the same label look identical; at minimum show the source (file or URL) under the name.
- A small trend (pass rate and covered percentage over the last runs) is a cheaper first step than the full Analytics tab (B-017, L).

### 23. Accessibility pass — M (NEW)

26 component files use `aria-label` or `role`, out of about 70. A focused pass: visible focus on the rail and tree, every icon-only button labelled, status not conveyed by colour alone (the pass/fail/not-sent cards need a text or icon cue), contrast in the light theme (the stylesheet already remaps Tailwind 400 colours to 700 by hand, which suggests contrast was found lacking once), respect `prefers-reduced-motion`. Add `eslint-plugin-jsx-a11y` only after asking, as it is a new dependency outside the locked list.

### 24. Shortcuts for the newer pages — S (NEW)

The shortcut table covers requests and tabs. Add shortcuts for: switching rail pages, running the current scenario, running conformance, opening Contracts, and a visible "?" sheet. Surface the shortcut next to menu items.

### 25. Command palette — M (NEW)

Ctrl+P opens requests and scenarios. Make it a palette that also runs commands ("Run all scenarios", "Check conformance", "Open environment…", "Copy CLI command", "Open workspace"). It reuses the existing `QuickOpen` and `MenuBar` actions.

### 26. Empty states and first run — S–M (NEW)

A first launch should end in a first success within a minute. Suggested path on the empty state: *Open the demo workspace* → *Run the demo scenario* → *Check conformance against the demo server* → *Open the Contracts page*. Each step is a button, and the Workspace page shows which steps are done. (The demo server and workspace exist; this is wiring and wording.)

### 27. Run log ergonomics — S (NEW)

Collapse passed steps by default, auto-scroll to and expand the first failure, keep a sticky summary (passed / failed / not sent / skipped) while scrolling, and offer "Copy as cURL" on a step with secrets masked like reports. Matches the 2026-11-18 scenario declutter direction.

### 28. Wording and naming consistency — S (carried over)

The 0.6.0 check lists UI items U2 to U11 (wording, naming, layout, B-067 to B-070) as still open. Do them in one pass together with the accessibility pass, since both touch labels.

---

## 4. Documentation and onboarding

### 29. Screenshots and recipes (B-013) — M

The user guide is text only, though the screenshot gallery is regenerated automatically. Embed the generated images in the guide and add five task-based recipes: *import a spec and run conformance*, *adopt on a messy API with a baseline*, *gate a pull request on `diff`*, *test a login-then-fetch scenario*, *run in GitHub Actions*. The CI guide covers the last one already.

### 30. Terminology in the user guide — S (NEW)

Section 8 of the user guide is titled "Scenarios and contract testing". `CLAUDE.md` section 1 says not to use "contract testing" alone (it suggests Pact-style consumer-driven testing). Rename to "Scenarios and API Contract Validation", grep the help documents and the AI prompt for the bare phrase, and add a test that fails if it reappears outside an explicit "not consumer-driven (Pact)" sentence. The sandbox guide review found the same slip in another repository.

---

## 5. Engineering health

### 31. Split `App.tsx` — M (NEW)

`App.tsx` is 1,619 lines; the next largest component is about 600. Rule 1 and the "state lives in the appropriate layer" rule are respected in spirit, but one file that owns tabs, workspace, menus, dialogs and shortcuts is hard to change safely. Extract by responsibility (tab host, dialog host, global shortcuts, workspace session) into hooks next to the existing `useTabsState`/`useWorkspace`. No behaviour change; the 1,600+ vitest tests are the safety net.

### 32. Generated bindings and types stay in sync — S (NEW)

The 0.6.0 check notes that `wailsjs/go/main/*` is tracked so CI can build the frontend before running `wails`. Add a CI step that regenerates the bindings and fails on a diff, so a changed Go struct cannot leave stale TypeScript types. Also pin Wails in the release workflow (R2 of the 0.6.0 check, `@v2.17.1`) if not already done.

### 33. Close the Low findings — S–M

From the 0.6.0 check and the backlog: B-061 (temporary workspace cleanup), B-063 (demo servers accept any web origin), B-064 and R7 (path guards), R3/R4/R5/R6 (Workspace page accuracy), B-080 (Contracts store key collisions). None blocks anything, but B-063/B-064/R7 are security-flavoured (sections 3 and 7 of `CLAUDE.md`: restrict file operations to the selected project) and are best done together as one hardening batch.

### 34. Contracts store keys — S

`coverage.Key` replaces characters with `_` and keeps only the last 80, so different specs can share an entry (B-080). Hash the full resolved path and keep a readable prefix for display. This changes stored keys, so it needs a short migration note.

### 35. Define 1.0 (B-037) — S

Write down what separates 0.x from 1.0: a `schemaVersion` migration policy (required by section 8 of `CLAUDE.md`, "documented migration rules"), a frozen CLI flag surface, signed builds, evidence of use. Naming the criteria turns many open items into "before 1.0" or "after 1.0" and shortens the backlog.

---

## 6. Distribution and growth

### 36. Failure-catalog repository (B-036) — L

One deliberately broken operation per *caught* catalog row and one per *known gap* showing it passes today, green in CI against the released version and nightly against `main`. The sandbox repository and its guide already cover much of this; the remaining work is the CI wiring and keeping the guide's counts generated, not typed (sandbox review, item 19).

### 37. Community scaffolding (B-032) — S

`CONTRIBUTING.md`, issue and pull-request templates (with a "which check ID?" field), a feedback link in the README and the About dialog.

### 38. Hosted demo server (B-031) and code signing (B-035)

Both are decisions, not tasks. The hosted demo needs a hosting and abuse decision; signing was postponed by the owner until real feedback exists. A cheap way to get that feedback is an opt-in, local-only usage summary the user can read before sharing; the `telemetry` package exists, so review what it records and whether the wording makes the choice clear.

---

## 7. Suggested order (first pass; superseded by section 11)

If only five things happen next, in this order:

1. **B-053 formats, B-050 warning levels, B-049 check IDs.** They are one theme (trustworthy findings) and unlock gaps 2 and 3 of the catalog.
2. **Idea 4 (XML bodies as "not sent")**, because it is a wording fix that removes a misleading message.
3. **Idea 19 (baselines in the page)**, the biggest CLI-versus-desktop gap in conformance.
4. **Ideas 12, 13 and 14 (`init`, tags, `validate --strict`)**, which together remove the first-hour friction in the CLI.
5. **Ideas 33 and 30**, a one-day hardening and wording batch.

Everything else can wait for user feedback.

## 8. Considered and deliberately not proposed

These came up while reading and conflict with the locked scope in `CLAUDE.md` section 9 or the architecture rules. They are noted so they are not re-raised without a decision, and **none is added to the backlog**.

- **JavaScript scripting in requests.** Already ruled out in the Postman gap review: it would cost the declarative, diffable property that distinguishes the product.
- **Mock servers and simulations (B-018).** Out of scope for the MVP; needs an owner decision first. Spec-derived mocks are the nearest neighbour to the Conformance Server and Demo Server, which are test fixtures, not product features.
- **GraphQL, WebSocket, gRPC, load testing, AI-generated tests.** Out of scope.
- **Team features: ownership scopes across teams, org-wide rollups, policy packs, SSO and audit, private runners.** These belong in `docs/enterprise/` (see `2026-11-21-enterprise-notes.md`). One open question there is whether "scopes" (a saved set of operations that coverage and conformance count) is a general core feature after all; idea 5 of that document should be decided before anyone builds either.
- **OS credential store.** Post-MVP per `CLAUDE.md`; secrets stay as `${NAME}` references.

## 9. Open questions for the owner

1. Is warning level (idea 2) acceptable as a change to what a finding is, given that B-002 already turned some CI red once? If yes, ship it with idea 1, in one release note.
2. Should a spec lint (idea 8) be its own command or a section of `validate`/`conform`? A separate command is simpler to explain; a section reuses the page.
3. Is a saved scenario-tag field (idea 13) worth a schema change before there is a migration policy (idea 35)? Doing the policy first is safer.
4. Should the sandbox repository become the failure-catalog repository (idea 36), instead of a third one?

---

## 10. Second pass (Claude, 2026-11-24)

### 10.1 Checks of the first pass

Verified against the repository on the same day:

| Claim in the first pass | Result |
|---|---|
| `App.tsx` has 1,619 lines | Correct. |
| 26 component files use `aria-label` or `role`, "out of about 70" | 26 is correct (non-test files). The component folder holds about 130 entries including tests, so the share is lower than stated; count again before quoting a percentage. |
| "1,600+ vitest tests" | About **1,400** test cases (`it`/`test` calls). |
| User guide section 8 says "contract testing" | Correct (`user-guide.md` lines 16 and 294). The README uses the phrase only to contrast with Pact, which is fine. |
| The abstract mentions a "project config" | No numbered idea covered it; added as idea 43. |
| CI | A search of `ci.yml` found no `go vet`, vulnerability or lint step. Confirm by reading the file before adding (idea 47). |

Nothing in the first pass looked wrong in substance. Ideas 2, 3 and 19 all depend on the same open decision (the baseline key, B-049/B-007); section 7 did not make that the first step, section 11 does.

### 10.2 New ideas

#### 39. Run a login scenario before `conform` — M (NEW)

**Why first.** Since B-009 an operation that needs `{{token}}` is *not sent* unless the user supplies it. For a secured API the first `conform` run reports most operations as "could not be checked", and the user must fetch a token by hand and pass `--var token=...`. The OAuth2 pattern is documented for scenarios but cannot feed `conform`.
- **Idea.** `clrsig conform --setup login.yaml` runs that scenario first and uses its extracted variables as overrides (same redaction as `--var`, never in reports). The desktop page gets a "Run this scenario first" picker.
- **Review.** A new flag plus a small orchestration in `packages/core/application`; no format change. It reuses the scenario runner, so rule 1 holds.

#### 40. Conform only what changed — M (NEW)

`diff --against <git-ref>` knows which operations changed. `conform --changed-against main` would check only added or modified operations, so a pull-request gate takes seconds on a large API. Needs the operation identity from `diff` exposed through the shared core, not the CLI.

#### 41. Annotations in pull requests — S–M (NEW)

A `--format github` output (workflow commands such as `::error ...::`) turns findings into inline annotations with the check ID in the title. The job summary (idea 16) gives the overview; annotations give the location. Plain text, no dependency. Locating the spec line may need a small lookup, since `kin-openapi` does not keep line numbers.

#### 42. Data-driven scenarios — M (NEW, check first)

Run one scenario over the rows of a CSV or JSON file (`--data users.csv`), reported per row. Read `2026-11-17-postman-gap-review.md` first: if it was considered and declined, drop this idea. Otherwise it is the most common Postman-runner feature still missing and fits the declarative format. It is format-adjacent, so it needs a proposal.

#### 43. Project config file — S–M (NEW)

The same flags repeat in every command and workflow (`--spec`, `--env`, `--base-url`, limits, report list). A `clrsig.yaml` at the workspace root, overridden by flags, would shorten commands, CI files and the builder's output, and is the natural thing for `init` (idea 12) to write. It is a new persisted file: `schemaVersion`, an entry in `format-spec.md` and validation (`CLAUDE.md` section 8). Decide the migration policy (idea 35) first.

#### 44. Keep `format-spec.md` honest by test — S (NEW)

Section 8 of `CLAUDE.md` requires the spec to change in the same commit, but only people enforce it. The catalog already has guard tests; add one that walks the YAML/JSON tags of the serialization structs and fails if a field is missing from the spec. The AI prompt is built from that spec, so drift also reaches AI-generated files.

#### 45. Version and release notes from one source — S (NEW)

The version is typed in `AboutDialog.tsx`, its tests, the user guide and the release notes, and the 0.6.0 in-app notes already drifted from GitHub's. Keep one user-facing `docs/CHANGELOG.md` that the About dialog renders and a test compares with the version constants.

#### 46. Move the engineering changelog out of `CLAUDE.md` — S (NEW)

`CLAUDE.md` is loaded into every working session and its changelog is now several times larger than the rules. Move section 10 to `docs/process/engineering-changelog.md`, leave a pointer and keep sections 1 to 9. Do it in its own commit so history stays readable. (This changes the project's operating file, so it needs the owner's approval.)

#### 47. CI gaps — S (confirm first)

Check `ci.yml` and add what is missing: `go vet ./...`, `tsc --noEmit`, Vitest, the Playwright `frontend` project (the 5 `smoke` tests need a live server; keep them apart), a link check over `docs/`, and a vulnerability scan. A scanner is a tool outside the locked list: ask first (`govulncheck` is the Go team's own and adds no module dependency).

#### 48. Spec examples validated in `conform` — S (NEW)

The spec's own `examples` are only checked at parse time for JSON media types. Report an example that does not match its schema as a spec problem (part of the lint, idea 8), because an invalid example is a common reason a user "fixes" the server and then finds the spec was wrong.

#### 49. One "what next" mechanism — M (NEW)

Ideas 20, 26 and the Workspace page's attention list are the same thing: a finding with a fix and a button. Define one `Action` type in the Go core (`kind`, `label`, `target`) next to `conformance.Finding` and the `workspaceinfo` items, so the desktop renders buttons and the CLI prints the same advice. It avoids three separate "fix" vocabularies and keeps the logic in the core.

#### 50. Copy a finding as an issue — S (NEW)

An HTML report anchor per check ID and a "copy as issue" action (title, ID, operation, status, expected and got, no secrets) on the desktop and in the HTML report. It shortens the feedback loop that produced the sandbox findings.

---

## 11. Plan for the next steps

Principles: **trust before features** (a wrong green or red costs more than a missing check); **one behaviour change per release**; every behaviour change gets a release note and an adoption path (`--write-baseline`); every format change follows `CLAUDE.md` section 8. Release numbers are suggestions.

### Step 0: decisions (owner, no code, about an hour)

| # | Decision | Recommendation |
|---|---|---|
| D1 | Baseline key (B-049, B-007) | Key on check ID plus operation (method, path template) plus a short detail; migrate old baselines on first read. |
| D2 | Warning level (B-050, idea 2) | Yes: `failure` / `warning` / `note`, `--fail-on warning` promotes. Reuse `diff`'s vocabulary. |
| D3 | Migration policy (idea 35) | Write it before any new persisted file (ideas 13, 42, 43): one page in `docs/process/`. |
| D4 | Spec lint: command or part of `validate` | Part of `validate` plus a card on the page; one command fewer to teach. |
| D5 | Sandbox repo as the failure-catalog repo (idea 36) | Yes; avoids a third repository. |
| D6 | Contracts store key (B-080) | Key by the resolved file when it exists; hash only for URLs. Settles the earlier back-out. |

### Step 1: 0.9.0, "trustworthy findings"

1. Format validators (idea 1) and the XML "not sent" wording (idea 4).
2. Severity levels and the check-ID catalog as Go data (ideas 2 and 3), using D1 and D2; the existing guard test becomes data-driven.
3. Ideas 44 (format-spec guard test) and 30 (terminology): cheap, and they protect everything after.
- **Done when:** a `format: email` mismatch shows its ID in all five report formats; old baselines still work after migration; an uncatalogued kind fails a test; the release note lists which green CI runs may turn red and how to adopt.
- **Needs:** one proposal (`docs/enhancements/2026-11-xx-severity-and-check-ids.md`).

### Step 2: 0.9.1, "first hour"

1. Idea 39 (`conform --setup`).
2. Ideas 12, 14 and 43 (`init`, `validate --strict`, project config) together, so `init` writes the config.
3. Ideas 16 and 41 (Action inputs, job summary, annotations).
4. Idea 26 (first-run path in the app), the desktop twin of `init`.
- **Done when:** on an empty folder, `clrsig init` plus one command gives a meaningful report against a spec, in CI and in the app.
- **Needs:** one proposal for `init` and the config file (format change).

### Step 3: 0.10.0, "desktop parity"

1. Baselines in the page (idea 19, needs D1), then run history and compare (idea 21).
2. The shared `Action` type (idea 49), then buttons on findings (idea 20).
3. Contracts duplicates and key (ideas 22 and 34, D6) with a migration note.
4. Ideas 23, 24, 27 and 28 as one UI pass (accessibility, shortcuts, run log, wording touch the same labels).
- **Done when:** a baseline recorded by the CLI compares on the page and the reverse; every icon-only control has a name; Playwright covers the new flows.

### Step 4: ongoing

- **Engineering health, one small change each:** 46, 45, 47, 32, 33; split `App.tsx` (31) after step 3 so the split follows the new flows.
- **Conformance depth, on demand:** ideas 5, 6, 7, 8, 9, 40; each adds catalog rows and tests.
- **Wait for feedback:** ideas 10, 11, 42, 36, 38. Idea 37 (community scaffolding) is cheap and can go in any release.

### Risks

- Two behaviour changes in one release hide which one turned CI red; steps 1 and 2 are split for that reason.
- Every new persisted file (config, scenario tags, data files) multiplies the migration burden; D3 first.
- UI comments here come from code and documents, not a running app; confirm each in a short session before building.

### Tracking

Ideas 39 to 50 were added to `docs/planning/backlog.md` on 2026-11-24 as B-084 to B-095 (39 B-084, 40 B-085, 48 B-086, 41 B-087, 42 B-088, 43 B-089, 47 B-090, 49 B-091, 50 B-092, 44 B-093, 45 B-094, 46 B-095). Tick items there as they ship. This file stays a point-in-time snapshot; later reviews go in a new `check-YYYY-MM-DD-*.md`.
