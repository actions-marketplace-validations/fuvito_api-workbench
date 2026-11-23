# Review of the sandbox guide — 2026-11-22

Document reviewed: `docs/contract-and-conformance-guide.md` in `fuvito/api-workbench-sandbox` (main, 857 lines: Part 0 basics, Part 1 problems P01 to P60, Part 2 how clrsig finds them, Part 3 notes for the NotebookLM videos).

Method: read all of it, then checked its claims against this repository at the current head (**clrsig 0.8.8**): the catalog `apps/desktop/wails/help/what-we-check.md`, `--help` of `conform`, `run`, `diff`, `coverage quality`, `generate`, and the commit log since v0.8.5. I did not run the sandbox server or re-run its examples.

Overall: it is a strong guide. The "what goes wrong / why it hurts / real fixture" shape is easy to follow, the limits are stated honestly, and the P-numbers tie Parts 1 and 2 together. The comments below are mostly **out-of-date facts** (the guide is pinned to 0.8.5 and we are at 0.8.8) and a few **wording and structure** points.

## 1. Facts that are out of date (fix before recording)

| # | Where | Comment |
|---|---|---|
| 1 | P15, P35, P52, P60, Part 2 "Group L", Episode 11 "Limits", Episode 12 demo, checklist 3.8 | **F-033, F-034 and F-035 are fixed in 0.8.6** (commit "NDJSON is not read as one JSON document, XML examples no longer invalidate a spec, the no-credentials probe drops a cookie API key"). The guide says all three are "still open in v0.8.5" in at least eight places, and Episode 12 builds a story on the open NDJSON case. The best fix is the stronger story: found by a fixture, reported, fixed in the next release. Re-run those three fixtures on 0.8.8 and flip their expectations. |
| 2 | P37 to P42 (Part 2), P56, Episode 4 | **"One global time limit, not one per operation (CS-GAP-005)" is no longer true.** 0.8.8 added `conform --max-response-time-for /prefix=ms` (the longest prefix wins, `0` turns the limit off there), and the catalog's CS-GAP-005 now reads "response-time trends". The remaining limit is trends and a per-operation budget in the spec itself. Update P56's list, the Part 2 "Limits" line and the command table (`--max-response-time-for` is missing). Add a fixture: a slow report endpoint under a generous prefix next to a fast API under a tight global limit. |
| 3 | Version pin everywhere (`v0.8.5`, Docker tag `ghcr.io/fuvito/api-workbench:v0.8.5`, the NotebookLM instruction "Say that results are for clrsig v0.8.5") | Move to 0.8.8 (or whatever is current at recording time) and keep the version in **one** place (a variable in the guide, or one line the episode cards refer to). It appears in a dozen places, so the next release will leave some stale. |
| 4 | Check IDs the guide never mentions | The catalog now has IDs for several things the guide describes without one. Use them, so a viewer can look each one up: `CS-RUN-008` (an extraction that finds nothing fails the run: your P59 "failed extraction exited 0"), `CS-RUN-009` (repeated headers, integers above 2^53 keep their digits: P60), `CS-PROBE-006` (a probe with no answer is a failure, "the summary said PASSED": P59), `CS-PROBE-005` (probes follow `--tags` and `--paths` and reach JSON/JUnit), `CS-REQ-003` (requests carry credentials, cookies, array styles, form and multipart bodies: P25 and P26 talk about it), `CS-RESP-012`. Run `clrsig conform --explain` against each and quote the real text. |
| 5 | Part 2 "Limits" for P35/P36 | CS-GAP-008 and CS-GAP-009 are still open in the catalog and the guide is right to say so. Worth saying where the guide's own workaround (a scenario with one step per bad token) is the **recommended** answer, not a stopgap, so that the day a probe exists the advice still stands. |

## 2. Wording that conflicts with our own terminology

| # | Where | Comment |
|---|---|---|
| 6 | Title, Part 0 glossary, Episode 0, the NotebookLM series instructions | **"Contract testing" is used alone**, as a name for what clrsig does. Our own rule (`CLAUDE.md` section 1) is to say **API Contract Validation** and **Spec Conformance Testing**, because "contract testing" on its own makes most developers think of **Pact-style consumer-driven** testing, which this is not. In a public video series this is the most likely source of confusion (people who search "contract testing" expect Pact). Suggest: define the two canonical terms in Part 0, add one sentence "this is not consumer-driven contract testing (Pact)", and keep "contract testing" only inside that sentence. Do the same in the NotebookLM instructions, or the generated narration will use the short form. |
| 7 | Part 1 as a whole | It never says **which problems belong to which kind of check**. A reader cannot tell "response contract" from "whole-API conformance" from "scenario behavior". Add a tag to each problem heading, for example `[Contract]`, `[Conformance]`, `[Scenario]`, `[Spec file]`, `[Tool limit]`. The sandbox already splits the fixtures this way (`contract-fail` against `conform-fail`); the guide should show the same split. |

## 3. Things a reader needs that are missing

| # | Comment |
|---|---|
| 8 | **No safety paragraph.** `conform` sends `POST`, `PUT` and `DELETE` requests built from the spec. Part 0 and Episode 13 should say: run against a test server, never production; use `--read-only` first; secrets come from the environment (`${NAME}`), not from files; the desktop app warns before a remote base URL. A developer who pastes Episode 13's recipe into CI against a shared staging API will otherwise write data. |
| 9 | **The desktop app is not mentioned.** Everything is CLI. Since 0.6.0 the same checks run in the app (Check Conformance page, Contracts, Workspace page) and "Help, What We Check" shows the catalog. One short box in Part 0 ("the same checks are in the desktop app; this guide uses the command line because it is easy to show and to script") would stop viewers asking. Same for the **MCP server** (conform tool, 0.8.7) and the **GitHub Action**. |
| 10 | **Ep13 "Pin the version"** points to `CLRSIG_VERSION`, but a reader of the guide uses the GitHub Action, Homebrew or Docker. Give the three one-liners (the Action's `version:` input, whose default is *latest*; the Docker tag; `brew`/`scoop` version). The Action had a bug in 0.8.8's notes ("installing into the workspace"), so tell readers to move to 0.8.8 or later. |
| 11 | **Exit codes** are explained well in the command table, but the three-way meaning (`0`, `1`, `2`) should be in Part 0 too, because Episode 0 uses it in a recap question before the table appears. |
| 12 | **Baselines** (P50) explain adoption but not the file's life: commit it, review diffs of it in pull requests, and know that recorded baselines from before a clrsig upgrade can change meaning (catalog backlog B-007). One sentence. |

## 4. Comments on the one wording that may mislead

| # | Where | Comment |
|---|---|---|
| 13 | P25, Part 2 (XML request bodies) | The report says "the request body does not fit the spec: value is required but missing" for an XML-body operation, because clrsig sent **no body**. The guide reports this honestly as a known gap, but the message itself points at the server or the test, and is really a **tool limit**. Treat it as a finding for clrsig (it should say "not sent: no XML request body can be built", like a `CS-NC` outcome), and mention the wording in the guide until fixed. I can open a backlog item. |
| 14 | P23 | The example shows a correct, strict server (UUID only) failing because the spec says integer. The cause is the *spec* being wrong, not the server. Say that plainly in "What to do" (P20 to P24 currently say "find the source of truth", which is right, but this one deserves "the spec is the likely culprit"). |
| 15 | P12 and P56 | "Only the first problem per response" is shown as a limit. It also affects how to read **any** failing run: fix, re-run, repeat. A single sentence in "How to read a result" would help every episode. |

## 5. Structure and upkeep

| # | Comment |
|---|---|
| 16 | **Part 3 is production notes for the video maker, in the same file as the developer guide.** It is 240 of 857 lines. A developer reading the guide on GitHub meets "paste into NotebookLM" instructions. Move Part 3 to its own file (for example `docs/video-production.md`) and link it. The guide then reads as documentation and the video file can change without touching the facts. |
| 17 | **The episode sources (3.7) are built by copying headings by hand into `ep00.md` to `ep13.md`.** That is exactly the kind of manual step that drifts after the first fix. A short script in `scripts/` that splits the guide by problem number (the headings are regular) and writes the episode files and the glossary would remove the step, and `check_expectations.py` could fail if a P-number named in an episode card does not exist. |
| 18 | **Parts 1 and 2 repeat each problem.** Reading one problem means jumping between two places, and NotebookLM gets the same problem twice in different sources. Consider one card per problem (what goes wrong, why it hurts, example, check ID, what you see, what to do, limits) and generate the two "views" if you still want them. The P-numbers already allow it. |
| 19 | **Counts that will drift:** "60 problems", "14 episodes, about 100 minutes", "65 operations" (Episode 0), "35 wrong answers (F-001 to F-035)", the 500-operation timing claims. Generate them or put them in one table. |
| 20 | **Timing claims** ("well under a second", "2,000 operations in 0.3 s") depend on the machine and on the clrsig version. Say "on a laptop, v0.8.8" next to them, as the guide already does for the latency fixtures. |

## 6. What I would keep exactly as it is

- The three-part shape of each problem and the real fixture names; they are the strongest thing in the guide.
- The honesty rules in 3.3 ("never say clrsig catches everything", "a test tool can lie, test the tool").
- Part 2's "which command for which question" table; it is the best single page.
- P59 and P60: the story that the sandbox found 35 wrong answers in clrsig itself is a better advertisement than any feature list. With the 0.8.6 fixes it is stronger, because it ends with "fixed".
- The checklist in 3.8, in particular "read `docs/clrsig-findings.md` again before you publish"; this review is that check failing, so keep it and make the version pin the first item.

## Suggested order

1. Items 1 to 4 (facts): without them the series will say things that are no longer true.
2. Item 6 (terminology) before anything is recorded, because narration repeats whatever the instructions say.
3. Items 8 and 9 (safety, desktop app, MCP, Action): a short box each.
4. Items 16 and 17 (move Part 3, generate the episode files), then 18 if you want the cards.
