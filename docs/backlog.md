# Project backlog

The one living list of work that is not done yet. It is edited in place: tick an item when it ships, add new ones as they come up. It does not replace the detailed documents; each row links to where the idea is described.

| Document | What it is for |
|---|---|
| **This file** | What is open, at a glance. Edited in place. |
| [`docs/next_steps/`](next_steps/README.md) | Point-in-time plans: what to do next and in what order. Never edited to stay current. |
| [`docs/enhancements/`](enhancements/README.md) | One design proposal per major feature, agreed before it is built. |

## How to use it

- **Done:** `☐` open, `☑` shipped. Move nothing; tick the box and add the date or commit in *Notes*.
- **Size:** **S** about a day or less, **M** a few days, **L** a week or more. These are rough estimates, not commitments.
- **Needs:** what must happen before building. *Proposal* means a dated file in `docs/enhancements/` per `CLAUDE.md`; *Decision* means an owner decision first; *None* means it can be built as is.
- Priority is the owner's call and is deliberately not a column. Ordering inside a section is not a ranking.
- A new item gets the next free ID (`B-###`). IDs are never reused.

---

## Conformance (Check Conformance page, `clrsig conform`)

| Done | ID | Item | Size | Needs | Source / notes |
|---|---|---|---|---|---|
| ☑ | B-001 | Use the plain-language explanation (expected, got, why, fix) in the exported reports (HTML, JUnit, JSON, Markdown) and in the `clrsig conform` text output. | M | None (core `conformance.Diagnose` already exists) | Done 2026-11-19. JSON gains an additive `findings` array; HTML, Markdown and JUnit add one `→ … Why: … Fix: …` line per failure (JUnit's failure message is unchanged); the screen output only with the new opt-in `--explain`, so existing CI logs do not change. Values are redacted like the raw lines. Tests in `reporting/conform_explain_test.go` and `apps/cli/readonly_test.go` |
| ☑ | B-002 | Response checks silently pass: with a declared response header, a wrong `Content-Type` or an undeclared status, the response body is never compared and the operation shows as passing. Root cause (measured 2026-11-20): only `Content-Type` is forwarded to the validator and kin-openapi's `ResponseError.Reason` is dropped when `Err` is nil. | M | Proposal | Done 2026-11-20 ([proposal](enhancements/2026-11-20-conformance-accuracy.md) part A, decisions accepted). Every real response header is now forwarded and `ResponseError.Reason` is reported, giving the violation kinds undeclared status, wrong Content-Type, missing header and bad header, each with what / expected / got / where (response part and spec location) / why / fix. Applies to `conform`, scenario spec checks and the page. The Conformance Server has three more deliberate defects (six in all). Changes what counts as a violation, so CI that was silently green can turn red; `--write-baseline` adopts it gradually. The `type: schema` assertion by operationId is unchanged |
| ☑ | B-003 | Generated request bodies cannot be valid: every property is a quoted string placeholder (numbers, booleans, objects, arrays included), optional properties are included, `example`/`default`/`enum` are ignored and an `allOf` body becomes `{}`. The Swagger 2.0 importer has the same generator. | M | Proposal | Done 2026-11-20 ([proposal](enhancements/2026-11-20-conformance-accuracy.md) part B, decisions accepted). One shared generator (`openapi/bodystub.go`) for OpenAPI 3.x and Swagger 2.0: typed placeholders, required properties only (plus any with an example or default), `readOnly` left out, literal example / default / first enum value, `allOf` merged, first `oneOf`/`anyOf` branch, array bodies. Tests assert a filled-in generated request passes `ValidateRequest`, and an end-to-end CLI test sends real numbers, booleans and arrays. The request-side validator no longer drops the parameter name or the "request body" context, and request problems are explained like response problems. Format JSON understands `{{placeholders}}`. Imported request files now contain fewer fields (see B-047). Not done: a Conformance Server operation with typed fields (the CLI end-to-end test covers it) |
| ☑ | B-004 | Progress events during long conformance runs (the page only shows "Running…" and Cancel). | S | None | Done 2026-11-19. `conformance.Options.OnOperation` → `application.ConformInput.Progress` → Wails event `conformance:progress` → a progress bar and "3 of 7 · GET /products" on the page. Negative probes do not report progress yet. Phase 2 of the [desktop conformance proposal](enhancements/2026-11-19-desktop-conformance.md) |
| ☑ | B-005 | `--var` overrides on the Check Conformance page (the CLI has them). | S | None | Done 2026-11-19. A "Variable overrides" box in Options (one `name=value` per line, this run only, never saved); the override logic moved to `variables.WithOverrides`, shared with the CLI. Not carried by Get CLI command because `var` is still a deferred builder input |
| ☐ | B-006 | Baselines in the page (record, compare, show new and fixed). | M | Decision | Phase 3 of the proposal, deferred until there is demand |
| ☐ | B-007 | Conformance baselines recorded before the path-level parameter fix list those operations under the old path (`{id}` rather than `{{id}}`); decide whether to migrate them or just document it. | S | Decision | `CLAUDE.md` changelog 2026-11-19 |
| ☑ | B-009 | Operations whose request still has an unresolved `{{placeholder}}` (for example `{{id}}` with no value in the environment) were sent anyway, and the server's 404 counted as a match when the spec declares it, so the operation showed as **Passing** though nothing was compared. | S–M | Decision | Done 2026-11-20 (owner: tell the user, do not proceed). Such an operation is now **not sent**: `OperationResult.Blocked`/`Missing`, counted as failed (run exits 1) and as not covered, with `not sent: no value for {{id}}` and one aggregated "missing values" line; the page shows NOT SENT cards and a button that adds the names to Variable overrides. Applies to the URL, query, headers, auth and body, and to failed dynamic values. A defined but unused variable is ignored. Affects any CI that ran `conform` without supplying every value: the repo's own `demo-conform.yaml` supplies them all. Follow-up for B-003: the generated body stub still needs a value for every property |
| ☐ | B-008 | "Add this run to Contracts coverage" has no CLI counterpart (`conform` always records to history, never to the Contracts store). | S | Decision | Why the page's Get CLI command cannot reproduce that option |
| ☐ | B-047 | Imported request files only contain the required fields (plus examples and defaults), so a spec whose schemas mark nothing as required imports as `{}` and the optional fields are no longer visible as a scaffold. Offer a full scaffold (every property, typed) when importing, while `conform` keeps required-only. | S–M | Decision | Consequence of B-003 (required-only was decided for `conform`; imports share the generator). Raised 2026-11-20 |
| ☐ | B-046 | For `conform`, generate valid sample values for request bodies from the schema (format, minimum, enum, …) so no body values are needed in the environment; keep `{{placeholders}}` only for what cannot be invented (path ids, credentials). | L | Decision, then proposal | Larger alternative named in the conformance accuracy proposal; a product-direction change, so deliberately not part of B-003 |

## Desktop app

| Done | ID | Item | Size | Needs | Source / notes |
|---|---|---|---|---|---|
| ☑ | B-019 | "Run all requests in a folder" cannot be cancelled: the desktop called `ScenarioService.RunFolder` as one blocking call that runs every request in turn (up to the default 30 s timeout each) with no stop button. | S–M | None | Done 2026-11-20. `ScenarioService.CancelFolderRun` stops it: the request in flight is abandoned (not reported), the rest never start, and the results so far come back; the folder results show a **Stop** button while running and ", stopped" afterwards. Test: `TestRunFolder_CancelStopsTheRun`; component tests for the button and label. No Playwright test (the mock `RunFolder` answers at once) |
| ☐ | B-010 | Contracts shows a second card for URL specs (such as the demo's) because the page picks files, not URLs; the page cannot join the card a URL-based scenario uses. | S–M | Decision | Conformance coverage recording, 2026-11-19 |
| ☑ | B-011 | A sample scenario and requests in the Conformance workspace, so the Scenarios and Requests panels are not empty and the scenario conformance line can be tried. | S | None | Done 2026-11-19. `scenarios/product-lifecycle.yaml` (names the spec; step 1 conforms, steps 2 and 3 break it) and four requests; `TestConformanceWorkspaceSampleScenario` runs it and expects matched / violations / violations |
| ☐ | B-012 | Regenerate the older screenshots so they show the new shield button in the rail (running the whole screenshot suite rewrites every image). | S | None | Deliberately left out of the conformance commits |
| ☐ | B-013 | Screenshots inside the user guide (the guide is text only). | M | None | Memory note; status not re-checked |
| ☐ | B-014 | "Copy CLI command" for the bundle import dialog, and `clrsig bundle run --workspace` coverage accumulation. | M | None | Written up in the `CLAUDE.md` changelog 2026-11-16 |
| ☐ | B-015 | CLI Command Builder, later phases: "Test it here", shipping the CLI with the app, the other commands, Docker and GitHub Action targets, multi-step pipeline snippets. | L | Proposal per phase | [CLI Command Builder](enhancements/2026-11-17-cli-command-builder.md) |
| ☐ | B-016 | Wails v3 migration (the app is held at v2.17.1; module path and service structs change). | L | Proposal | Memory note: about 2–3 days; see the official migration guide |
| ☐ | B-017 | Analytics page tab: stat cards, pass-rate trend, slowest endpoints, conformance donut, violations feed. | L | Proposal | Memory note |
| ☐ | B-018 | Simulations panel (spec-derived mocks, fault injection). **Mock servers are out of scope for the MVP** in `CLAUDE.md`, so this needs an explicit owner decision. | L | Decision, then proposal | Memory note |

## CLI and CI

| Done | ID | Item | Size | Needs | Source / notes |
|---|---|---|---|---|---|
| ☐ | B-020 | `validate --strict` (scenario quality lint). | S–M | None | [Future enhancements](enhancements/2026-11-17-future-enhancements.md) item 12 |
| ☐ | B-021 | `clrsig init` (workspace and CI scaffolding). | M | Proposal | Same file, item 13 |
| ☐ | B-022 | Select scenarios by tag or name. | M | Proposal | Same file, item 14 |
| ☐ | B-023 | Parallel scenario execution. | M | Proposal | Same file, item 15 |
| ☐ | B-024 | Project config file (`.clrsig.yaml`). | L | Decision | Same file, item 16 ("think twice") |
| ☐ | B-025 | Data-driven scenarios. | L | Decision | Same file, item 17 ("think twice") |
| ☐ | B-026 | A "conform in CI" guide using the GitHub Action (the CI docs show conformance jobs; whether they use the Action was not checked). | S | None | [Next steps 2026-11-19](next_steps/2026-11-19-next-steps.md) §1 |

## Docs and project

| Done | ID | Item | Size | Needs | Source / notes |
|---|---|---|---|---|---|
| ☐ | B-030 | Update or retire `docs/contract-testing-roadmap.md` (dated 2026-11-06; it still says conformance is "not yet assembled"). | S | None | Next steps §1 |
| ☐ | B-031 | Hosted demo/test server (hardened: no reset endpoint, body limits, scheduled reset) and a kitchen-sink demo workspace; add a run of it to the release checklist as an advisory step. | M | Decision (hosting) | Next steps §2 |
| ☐ | B-032 | Community scaffolding: `CONTRIBUTING.md`, issue and PR templates, GitHub Discussions, a feedback link in the README. | S | None | Next steps §3 |
| ☐ | B-033 | Video user guides, kept as regenerable scripts and screenshots under `docs/video/`. | M | None | Next steps §4 |
| ☐ | B-034 | User traction: posts, shorts, forums. Also the feedback loop that decides when signing is worth it. | L | None | Next steps §5 |
| ☐ | B-035 | Code signing and notarization (Windows Azure Trusted Signing, macOS Developer ID and notarytool). **Postponed by the owner** until real user feedback. | L | Decision | Next steps §6 |
| ☐ | B-036 | Failure-catalog repo: deliberately broken APIs with green CI against released clrsig and nightly against `main`. | L | None | [PRD](failure-showcase-repo.md) |
| ☐ | B-037 | Decide what separates 0.x from 1.0 (signed builds, desktop conform, a `schemaVersion` migration policy, a frozen CLI flag surface, some traction evidence). | S | Decision | Next steps §7 |

## Done

| Done | ID | Item | Notes |
|---|---|---|---|
| ☑ | B-040 | Demo GIF refresh | Automated: the `demo.yml` workflow regenerates `docs/demo.gif` on push |
| ☑ | B-041 | Desktop entry point for spec conformance | Check Conformance page, 2026-11-19; [proposal](enhancements/2026-11-19-desktop-conformance.md) |
| ☑ | B-042 | `clrsig conform --read-only`, and the page's Get CLI command carrying the page's choices | 2026-11-19 |
| ☑ | B-043 | Step `retry` / `timeoutMs` honoured by the runner | [Postman gap review](enhancements/2026-11-17-postman-gap-review.md) (A) |
| ☑ | B-044 | Future-enhancements items 1–11 | See the status line of the [future enhancements](enhancements/2026-11-17-future-enhancements.md) file |
| ☑ | B-045 | Conformance Server starts like the Demo Server: one menu item that starts it and opens its workspace (the separate "Open Conformance Workspace" item was removed) | Done 2026-11-20, raised by the owner |
