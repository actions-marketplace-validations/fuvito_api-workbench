# Improvement ideas, third round: what is left and what the last release batch got wrong — 2026-11-25

## Abstract

A new round after the work that followed [check-2026-11-24-improvement-ideas.md](check-2026-11-24-improvement-ideas.md). The project is at **v0.11.0 plus 4 commits** (head `5806ac0`). The 2026-11-24 report ended with 50 ideas and a four-step plan; most of steps 1 to 3 shipped in four releases (0.9.0 to 0.11.0) in one day.

This round differs from the earlier two: it **runs the product** instead of only reading it. It built the CLI from the head commit, started the example server, followed the first-hour path (`init`, `conform`, reports, `init github-actions`) and checked what the documentation tells users to type. That found problems that no test covers, because they sit between the pieces: a documented Action tag that does not exist, a generated workflow that disagrees with its own proposal, and reports that disagree about a path.

The three themes:

1. **The newest adoption path has loose ends.** The Action reference users are told to copy (`@v1`) does not exist, `init github-actions` pins to `@main`, and the Action installs its binary without checking the checksum the release already publishes. These are small, and they sit exactly where trust is decided.
2. **Reports still disagree with each other in small ways.** The same operation is `/users/{id}` in the text and GitHub output and `/users/{{id}}` in Markdown, HTML, JUnit and JSON.
3. **Bookkeeping has fallen behind the code.** Four backlog rows are done but not ticked, two are duplicates of finished rows, and idea 30 (terminology) was never given an ID, so it was never done.

Every idea is a **suggestion**. Per `CLAUDE.md` section 6, anything that changes a serialized format, a report's shape or a command needs a proposal first; those are marked.

> **Claude: review notes (2026-11-25).** Paragraphs marked **Claude:** were added in a second pass. The original text is unchanged so the two can be compared; where they disagree, the note is the newer reading. Section 7 adds findings and section 8 is the proposed plan, which replaces section 4. Nothing is implemented yet.
>
> **What this pass did:** built the CLI from head twice (a dev build and one stamped `v0.11.0`), ran both `init github-actions`, ran the example server with `conform` (text with `--negative`, Markdown, JSON paths, JUnit names), listed the remote tags and the checksum assets of four releases, and read the code behind V2 to V9. The server was stopped afterwards; binaries are in the session scratch folder, outside the repository. **Not done:** the test suites in "Method and limits" were not run again, and nothing was run on macOS, in Docker or on a GitHub runner.
>
> **Main corrections:** V2 is mostly already built; `init --spec` already exists (V5); V8 is larger than described; the post-release job in V10 would not work as written; and `clrsig-action/action.yml`, a stale second copy of the Action, was missed (V11).

## Method and limits

- **Ran, and passed:** `go vet` and `go test ./...` in `packages/core`, `go test ./...` in `apps/cli` and in the desktop module root, `tsc --noEmit`, Vitest (85 files, 1,680 tests). No `TODO` or `FIXME` remains in Go, TypeScript or workflow files.
- **Ran by hand:** a CLI built from the head commit, against the shipped example server on `localhost:8080`, in an empty folder: `init`, `init github-actions`, `config show`, `validate --spec --strict`, `conform` (text, JSON, Markdown, JUnit, HTML, GitHub), `conform --negative`, `--base-url` to a dead port, `--changed-against` outside git, an invalid `--fail-on`. Exit codes and output were read, not assumed. The temporary folder and binaries were deleted afterwards.
- **Also checked:** `git ls-remote --tags origin` (which tags exist), `action.yml`, `release.yml`, `ci.yml`, `docs/planning/backlog.md` against the engineering changelog, and a search for the term "contract testing".
- **Not run:** the desktop app (no GUI session), Playwright, the release workflow, Docker, Homebrew. UI observations in this round come from the changelog, not from the screen.
- Size: **S** about a day or less, **M** a few days, **L** a week or more.

## 1. Status of the earlier ideas

Checked against the backlog, the changelog and the code, not against memory.

| Earlier idea | State on 2026-11-25 |
|---|---|
| 1 Formats | Done for `email`, `uuid`, `ipv4`, `ipv6`; `uri` and `hostname` still open (CS-GAP-001). |
| 2, 3 Severity levels, check IDs | Done (0.9.0). |
| 4 XML request bodies | **Open.** Only a test mentions it. |
| 5, 6, 7 More probes, credential probes, all problems per response | Open. |
| 8, 48 Spec lint and examples | Done (`validate --spec`, CS-SPEC-001 to 003). |
| 9, 10, 11 Spec-declared time budget, time trends, webhooks | Open; 10 and 11 were meant to wait for demand. |
| 12, 14, 39, 43 `init`, `validate --strict`, `--setup`, project file | Done. |
| 13, 15 Scenario selection, parallel runs | Open (B-022, B-023). |
| 16, 41 Action inputs, annotations | Done. |
| 17, 18 `doctor`, machine-readable listings | Open. |
| 19 Baselines in the page | Done (B-006). |
| 20, 49 Fix actions, shared `Action` type | Proposal written, deliberately deferred. |
| 21, 22 Run history, Contracts duplicates | Open (B-010 still open). |
| 23 to 27 Accessibility, shortcuts, command palette, first run, run log | Open. A search finds no command palette and no shortcut sheet. |
| 28 Wording pass | Done (B-067 to B-070). |
| 29 Screenshots in the guide | Open (B-013). |
| 30 Terminology | **Open and untracked**, see finding V7. |
| 31 Split `App.tsx` | Open: still 1,619 lines. |
| 32 Bindings kept in sync by CI | Open: no `wails generate` step in any workflow. |
| 33 Hardening batch | Done (B-061 to B-064, B-074 to B-079). |
| 34 Contracts store keys | Open (B-080, decided in D6, not built). |
| 35 Path to 1.0 | Drafted; waits for the owner. |
| 36, 38 Failure-catalog repo, hosted demo, signing | Waiting for feedback, as planned. |
| 37 Community scaffolding | Done except GitHub Discussions (a repository setting). |
| 40, 42, 44, 45, 46, 47, 50 | 40, 44, 46, 47 done; 42 declined with reasons; 45 and 50 done in part. |

## 2. Findings from running the product

Severity is a suggestion: **High** misleads or breaks a user following the docs, **Medium** is a trust or consistency gap, **Low** is polish.

### V1. The documented Action tag `@v1` does not exist — High

`docs/guides/ci-integration.md` (lines 58, 166, 182), `docs/guides/authed-conform.md`, `clrsig-action/README.md` (five places, including "pinned release (recommended)") and `apps/desktop/wails/help/user-guide.md` line 944 tell users to write `uses: fuvito/api-workbench@v1`. `git ls-remote --tags origin` lists only `v0.x` tags (the newest is `v0.11.0`). A workflow that copies the documented line fails at the "set up job" step with "unable to find version v1". The Action self-test in CI uses `uses: ./`, so it cannot catch this. `ci-integration.md` line 101 uses an exact tag, which works.
- **Fix, either way:** (a) publish and move a floating major tag and say what it means while the product is 0.x, or (b) document exact tags (`@v0.11.0`) everywhere and print the current release in the examples. Option (b) needs no process change and matches the "pin a release" advice; option (a) needs a release step to move the tag.
- **Guard:** a doc check that every `fuvito/api-workbench@<ref>` in the Markdown names a tag or branch that exists (read from `git tag` in a local test, so it needs no network).
- **Decision:** (a) or (b). Recommendation: (b) now, (a) when 1.0 is defined.

> **Claude:** Confirmed: the remote has no `v1` tag. Agree with (b). `docs/process/release-process.md` already says a floating `v1` starts at 1.0.0, so (b) matches the written process, while its channel table (`@vN`) needs the same fix.
> - **Missed places:** `README.md` line 231 and `README-DEVELOPER.md` line 63. The `@v1` in `AboutDialog.tsx` line 256 is an old release note, so it is history, but it was wrong when written.
> - **Guard, changed:** a test based on `git tag` would find no tags in CI's shallow checkout. Instead, require every current-usage `@vX.Y.Z` to equal `v` + `APP_VERSION`, next to `TestVersionsAgree`, with an allow-list for history. The release prep commit then updates the docs. Between that commit and the tag the documented ref does not exist yet, so the release checklist should say to tag straight away.

### V2. `clrsig init github-actions` pins to `@main` — Medium

The generated workflow uses `fuvito/api-workbench@main  # pin a release tag such as v0.9.0 ...`. The init proposal (`2026-11-24-init-and-project-config.md`, line 137) says `uses: fuvito/api-workbench@<current version>`; the code does not do that. Also: the comment names `v0.9.0` (two releases old), the workflow uses `actions/checkout@v4` while this repository's own CI uses `@v7`, and there is no `permissions:` block, so the job gets the repository default token rights.
- **Fix:** write the CLI's own release version when it is a release build (a dev build keeps a placeholder and says so), add `permissions: contents: read`, and use one checkout version defined once for CI, `init` and the guides.
- **Guard:** a test that the generated file parses and names the running version.

> **Claude: severity Medium → Low; the main claim is wrong.** `@main` appears only in a dev build, which is what the report built. `scaffold/workflow.go` writes the CLI's own tag for a release version, and `release.yml` stamps that version. A build stamped `v0.11.0` printed `uses: fuvito/api-workbench@v0.11.0`. `scaffold_test.go` already tests both cases.
> - **Still valid:** the dev-build comment names `v0.9.0`; there is no `permissions:` block; the file uses `checkout@v4` and `upload-artifact@v4` while this repository's workflows use `@v7`. The user guide and `ci-integration.md` also use `@v4`.

### V3. The Action installs a binary without verifying it — Medium

`action.yml` downloads `clrsig-<os>-<arch>` with `curl -fsSL` and puts it on the `PATH`. The release workflow already publishes a `.sha256` file for each CLI binary (`release.yml` lines 237 to 264), so the data to verify exists and is not used. `version: latest` is resolved through the unauthenticated GitHub API, which is also rate-limited on shared runners.
- **Fix (S):** download the `.sha256` beside the binary, compare with `sha256sum` (or `shasum -a 256` on macOS), and fail on a mismatch. Keep the default, but make the log line say which tag `latest` resolved to.
- **Why it matters:** the Action runs with the user's secrets in its environment; a verified download is the cheapest supply-chain guard available.

> **Claude:** Agree. v0.7.0, v0.8.0, v0.9.0 and v0.11.0 each have four `.sha256` assets; older releases were not checked. A missing checksum should fail with a clear message, not be skipped.
> - The `.sha256` files hold only the hash (`release.yml` pipes through `awk`), so compare the strings; `sha256sum -c` will not read them.
> - **Limit to state:** the checksum comes from the same release as the binary. It catches a damaged or partly replaced download, not a compromised release. That would need signing or build attestations (B-035 territory).
> - The log already names the resolved tag ("Downloading … from vX"), so that part needs no work.

### V4. The same operation has two different paths in the reports — Medium

With `id=1` supplied, the text output prints `GET /users/{id}` (the spec path, since B-060), and so does the GitHub format. Markdown, HTML, JUnit and JSON print the **sent** path: `GET /users/{{id}}`. Observed in the HTML, the JUnit `name="DELETE /users/{{id}}"` attributes, the Markdown list and the JSON `"path"` value. Besides being inconsistent, a JUnit test name that differs from the spec path and from the check ID's operation makes test history in CI tools harder to follow.
- **Fix:** print the spec path in every human report (Markdown, HTML, JUnit name). For JSON keep `path` as it is (a consumer may rely on it) and add `specPath`, which is an additive field.
- **Needs:** a short note in `format-spec.md` or the report documentation; JSON gains a field, so list it in the release notes.

> **Claude:** Reproduced in Markdown, JSON and JUnit. HTML was confirmed by reading the code (`html.go` line 279). Agree with the fix, with these conditions:
> - **Do not change `OperationResult.Path`.** Baselines identify an operation as method + `Path` (`conformance/baseline.go` line 138, probes line 163), so changing the value would break recorded baselines. `--paths` matches against it too. Change only what reports display.
> - **The text report is affected as well:** its coverage lists print `op.Path` (`report.go` lines 519 and 526). So are the coverage lists in HTML, the coverage entries in JSON, and probe rows. A probe result has only `Path`, so check what it holds and add the spec path if needed. Use one display helper and one test over all formats.
> - **JUnit names change once**, so CI tools that keep test history will see those tests as new. Say so in the release note.
> - This changes report shape, so section 6 asks for agreement first. Your answer to decision 2 can be that agreement; a separate proposal is more than this needs.

### V5. A first `conform` after `init` is red and the starter files do not say why — Medium

In an empty folder: `clrsig init`, put the spec in `specs/`, set `spec:`, run `clrsig conform`. The result is `FAILED 8 operations · 1 passed · 3 failed · 4 skipped`, because the spec needs `token` and `id` and `environments/local.yaml` lists only `baseUrl`. The message after the run is correct and actionable ("nothing provides: token, id"), and exit 1 for a not-sent operation was an owner decision (B-009). The gap is the step before: `init` knows the spec path only after the user edits the file, and never offers the names.
- **Idea:** `clrsig init --spec <file>` (or `config` plus a new `conform --needs`) reads the spec once and writes the environment with the names it will need as empty, commented variables (`token` marked `secret: true` when the scheme is a credential). Output: "this spec needs: token, id, username, password. Fill them in `environments/local.yaml`."
- **Related:** idea 18 (`conform --list`) is the same data; build the listing first and let `init` reuse it.
- **Size:** S–M. No format change: the environment file already supports it.

> **Claude:** The gap is real, but **`init --spec` already exists**. It checks that a local file parses and writes the path into `clrsig.yaml`; it does not read which values the spec needs. The work is to extend it, with no new flag:
> - Take the names from the same generate-and-resolve code that decides "not sent", so the two cannot disagree.
> - Do it only when `init` creates the environment file; it never overwrites one. A URL spec is still not fetched.
> - An empty variable must still count as missing. Otherwise a pre-filled `id: ""` would send a broken request. Test this first.
> - A new `conform --needs` flag is unnecessary if `--list` (idea 61) shows the names.

### V6. The text summary counts "not sent" as "failed" — Low

`FAILED 8 operations · 1 passed · 3 failed · 4 skipped` mixes two different things: three operations that were never sent and operations that were sent and broke the spec. The desktop page already separates them (NOT SENT cards). The exit code is right as it is; only the line of words is blunt.
- **Idea:** `1 passed · 0 failed · 3 not sent · 4 skipped`, in the text and Markdown summaries, with no change to exit codes or JSON counts.
- **Needs:** release note; tests that parse the summary line (the sandbox repository) will need the new wording.

> **Claude:** Agree. The same count also feeds the HTML summary (`html.go` line 261) and the GitHub notice (`github.go` line 89), so change all four together. "Not sent" means blocked only; an operation that was sent and got no answer stays "failed". The verdict word, exit code and `Counts()` stay as they are.

### V7. "Contract testing" is still in the user guide, the demo server and the product document — Low

`CLAUDE.md` section 1 forbids the bare phrase. Idea 30 of the earlier report asked to fix it and to add a test; it was never given a backlog ID (it is not among B-084 to B-095), so nothing happened. Still present: `user-guide.md` line 16 and the heading at line 294 ("Scenarios and contract testing"), `apps/desktop/wails/demo_server.go` line 686 (text the Demo Server serves: "API contract testing platform"), `docs/product/Project.md` line 15. The README, `CONTRIBUTING.md` and `CLAUDE.md` use it only to contrast with Pact, which is allowed.
- **Fix:** rename the guide heading and its anchor (a link test already exists, so a broken anchor is caught), change the demo text, and add a `doccheck` test that rejects the bare phrase outside a sentence that also says "Pact" or "consumer-driven". Allow-list the dated reports, which are history.

> **Claude:** Confirmed, with one correction: **`CLAUDE.md` does not only use the phrase for contrast.** Line 10 calls the product an "API contract testing platform", which its own rule nine lines later forbids. I suggest "API Contract Validation platform". It is your operating file, so I will change it only if you say so (decision 5).
> - `docs/process/manual-steps.md` line 114 has "Contract test your API in CI", a title aimed at search traffic. It can be reworded or allow-listed.
> - The test should also match "contract-testing" and "contract test", and skip history: reports, archive, dated plans, the engineering changelog and the backlog.

### V8. The shipped example server does not follow its own spec — Low

`examples/server/openapi.yaml` declares `bearerAuth` on `GET /users` and `POST /users`, but `main.go` enforces the token only on `/protected`. `clrsig conform --negative` against it reports two failing probes ("server accepted invalid request with 200 / 201"). In the same run, a 404 answered with `application/problem+json` where the spec declares `application/json` (that 404 appeared because an earlier run of mine had deleted user 1 with `--read-only=false`; the content-type mismatch itself is independent of that). Nothing runs `--negative` against the example, so CI stays green.
- **Why it matters:** the example is what README and guides send new users to, and it is the target of the Action self-test. A user who adds `--negative` sees red on a project's own sample.
- **Fix:** either enforce the declared auth in the example (and answer 404 with the declared type), or label the example as "read-only quick start" and keep the failing flags out of the docs. Prefer the first: it turns the example into a regression test for `--negative` and credential probes.
- **Also:** the example has no reset, so a run that writes (`--read-only=false`) leaves it changed until restart; say so in the guide or add `POST /reset` to the example only (not to the product).

> **Claude: severity Low → Medium; the problem is larger.** Both probe failures (`GET` and `POST /users`, 200 and 201 with no credentials) and the 404 content-type finding reproduced.
> - **Every error body breaks the spec, not only the label.** The spec's `ErrorResponse` requires `error`, but `problemJSON` sends `type`, `title`, `status` and `detail`. This was read from the code. It stays hidden because the content-type mismatch stops the body from being compared, so fixing only the media type would reveal a second failure. A blocked-token run showed the same mismatch on the 401 from `/protected`.
> - **Recommended fix:** the server's RFC 7807 responses are sensible, so change the spec to declare `application/problem+json` with a matching schema. Then enforce bearer auth on `GET` and `POST /users` and declare their 401.
> - **Check while building:** every example scenario, the demo recording and the README quick start that calls `/users` must send the token.
> - **Related:** the login probes get 401 or 400, and wrong logins hit the example's rate limit. A 429 would also count as a pass (V12), so a green `--negative` job proves less than it appears to.
> - There is already a `POST /admin/reset`, but it resets only the rate limit. Extending it to restore the users avoids a new route.

### V9. Backlog bookkeeping has drifted — Low

The engineering changelog and the backlog's own notes say these are done, but the box is still `☐`: **B-049** (check IDs; "baseline key done, data catalog and severities are phases A and B", which shipped in 0.9.0), **B-050** (severity levels, shipped with B-053), **B-007** (resolved by baseline v2), **B-032** (done except GitHub Discussions, a setting). **B-024** (project config) and **B-025** (data-driven scenarios) duplicate **B-089** (done) and **B-088** (declined). An open-items view cannot be trusted while it has these. Idea 30 and ideas 31, 32 of the earlier report have no ID at all.
- **Fix (S):** tick, close with "see B-089/B-088", split B-032 into the done part and a one-line "turn on Discussions" item, and add IDs for the untracked ideas below.
- **Guard (optional):** a test that a row whose notes contain "Done" is ticked, or the reverse.

> **Claude:** Confirmed: B-007, B-024, B-025, B-032, B-049 and B-050 are unticked. Before ticking B-049, check each report format for the ID; I did not check them all.
> - **The reverse drift also exists.** Ticked rows hide open leftovers that need their own rows: annotations for `diff` and `bundle run` (B-087), the vulnerability scan (B-090), the HTML anchor per check (B-092) and the rendered changelog (B-094).
> - **B-082 is out of date:** it says only a per-operation limit remains. The per-path limit shipped in 0.8.8, so the row should say exactly what is left or be closed.
> - **Skip the guard.** Many open rows rightly contain "Done" for one part, so the test would raise false alarms.

### V10. Three release checks stay "not verified" every time — Medium

Each release entry in the engineering changelog ends with "Not verified: the Docker image run (no daemon on the machine), the Homebrew install (no Mac), and the Action at the new tag". That is five releases in a row, and the Docker image and the Action are exactly what CI users run.
- **Fix:** a **post-release smoke job** in `release.yml` (or a separate workflow run on `release: published`): on `ubuntu-latest` (which has Docker) run `docker run <image> --version` and compare with the tag; run the Action by its real tag against the example server (`uses: fuvito/api-workbench@${{ github.ref_name }}`) and check the `result` output. On `macos-latest`, install the tap and print the version. The same job would have caught V1 if the documented ref were exercised.
- **Size:** M (one workflow file). It runs after the release, so a failure is a warning to fix forward, not a gate; say that in the checklist.

> **Claude:** Agree with the goal, but two parts would not work as written:
> 1. **`uses: …@${{ github.ref_name }}` is not allowed.** `uses:` takes no expressions. Instead, check out the tag, run `uses: ./` and pass `version: ${{ github.ref_name }}`. This runs the tagged `action.yml` with the tagged binary. It does not test the remote reference, which the V1 test covers in part.
> 2. **`release: published` would not fire.** `release.yml` creates the release with the default `GITHUB_TOKEN`, and events from that token do not start other workflows. Make it a job in `release.yml` with `needs: [publish, update-tap]`.
>
> "Five releases in a row" holds for Docker and Homebrew (0.8.9 to 0.11.0). Since 0.11.0 the Action script runs in CI, but on Linux only and with the previous binary.

## 3. New ideas

Numbering continues from the earlier report (it ended at 50).

| # | Idea | Group | Size | Needs |
|---|---|---|---|---|
| 51 | Fix the documented Action reference and guard it (V1) | CI | S | Decision (a or b) |
| 52 | `init github-actions` writes the current release, `permissions`, one checkout version (V2) | CI | S | None |
| 53 | The Action verifies the published checksum (V3) | CI | S | None |
| 54 | One path in every report; `specPath` in JSON (V4) | Reports | S | Release note |
| 55 | `init --spec` pre-fills the environment with the names the spec needs (V5) | CLI | S–M | None |
| 56 | Summary says "not sent" instead of "failed" for unsent operations (V6) | Reports | S | Release note |
| 57 | Terminology test and the remaining "contract testing" strings (V7, was idea 30) | Docs | S | None |
| 58 | The example server enforces its spec; `--negative` joins the example job (V8) | Quality | S | None |
| 59 | Backlog tidy and IDs for untracked ideas (V9) | Process | S | None |
| 60 | Post-release smoke job: Docker, the Action by tag, Homebrew (V10) | Release | M | None |
| 61 | `conform --list` first, then `doctor` (earlier ideas 17 and 18, combined) | CLI | S–M | None |
| 62 | Bindings sync check in CI (earlier idea 32), on its own because it is a day | Engineering | S | None |
| 63 | `uri` and `hostname` format checks (finishes CS-GAP-001) | Conformance | S | Release note (can turn CI red) |
| 64 | Ship the `clrsig` CLI with a `clrsig help topics` or `--examples` view pointing at the guide sections | Docs | S | None |

Ideas 61 and 62 restate open earlier ideas on purpose: they are small, nothing depends on a decision, and they were lost in the long list. Idea 64 is the only one here that is a pure suggestion without a finding behind it; drop it first if the list needs shortening.

> **Claude, on the table:**
> - **51, 53, 57, 59, 62:** keep. I did not try the bindings check for 62.
> - **52:** keep, smaller (see V2).
> - **54, 56:** keep, with the conditions under V4 and V6.
> - **55:** reword to "`init --spec` also fills in the names the spec needs".
> - **58:** keep; it now includes the error schema, so S–M.
> - **60:** keep, redesigned as under V10.
> - **61:** split it. `conform --list` is one flag. `doctor` is a new command, which section 6 sends to a proposal first, so "Needs: None" is wrong for it. Build `--list` and let `doctor` wait.
> - **63:** keep waiting.
> - **64:** drop. No finding supports it, and its description does not say clearly what would be built.
>
> New ideas 65 to 68 are in section 7.

## 4. Suggested order

Principles are the same as before: trust before features, one behaviour change per release, and a release note for anything that can turn a green run red.

**Batch A, a documentation and CI hygiene day, no behaviour change (ideas 51, 52, 53, 57, 59, 62).** Every item is either a text fix with a guard test or a script fix, and V1 is the one a user can hit today. Ship as a patch release (0.11.1) so the corrected guide and the generated workflow reach people.

**Batch B, report consistency (ideas 54, 56).** Both change how an existing report reads, so they share one release note and one adoption sentence ("JSON gains `specPath`; text summary says *not sent*"). Update the sandbox repository's expectations at the same time, since it parses these lines.

**Batch C, first hour (ideas 55, 61, 58).** The listing of needed names is the data for `init --spec` and for `doctor`; the example server becomes the test that proves `--negative` and credential probes work on a clean project.

**Batch D, release trust (idea 60).** Needs the owner's attention once, since it touches the release workflow. After it, the three "not verified" lines can go.

**Wait:** idea 63 until the next conformance release (it can turn CI red); everything under "Open" in section 1 stays as the earlier plan ordered it (probes, history, accessibility, shortcuts, `App.tsx`), driven by feedback.

> **Claude:** Section 8 replaces this order. The Action fixes now go first and together, because the second copy and the macOS risk are in the same files. The example server moves ahead of the report changes so they have a clean target to run against.

## 5. Decisions for the owner

1. **V1:** document exact release tags now (recommended), or start publishing a moving `v1`/`v0` tag?
2. **V4:** is it acceptable to change the sent-path in Markdown, HTML and JUnit names to the spec path, keeping JSON `path` and adding `specPath`?
3. **V8:** fix the example server to follow its spec (recommended), or keep it deliberately loose and say so?
4. **V10:** accept a post-release workflow that can only warn, not block?

## 6. What this report did not do

It did not change code, tests or documentation; the only files created are this report and short-lived temporary files outside the repository, since removed. It did not add the ideas to `docs/planning/backlog.md`; if you want that, ideas 51 to 63 can be added as the next free IDs, and V9's ticking can be done in the same commit. No engineering-changelog entry was written, because no behaviour changed.

## 7. Claude: additional findings

### V11. A second, stale copy of the Action is still published — High

> **Claude:** `clrsig-action/action.yml` is a separate copy of the root `action.yml`. It lacks **18 inputs**: `read-only`, `negative`, `baseline`, `write-baseline`, `var`, `report`, `fail-on`, `against`, `changed-against`, `strict`, `setup`, `conform-fail-on`, `explain`, `tags`, `paths`, `request-timeout`, `max-response-time` and `max-response-time-for`. It last changed on 2026-11-17. Its README says the subdirectory form "also works and is kept for backward compatibility", and `action_test.go` does not mention it.
> - **Why High:** GitHub only warns about an unknown input, then carries on. A user of this path who sets `read-only: 'true'` would have **every operation sent, including `DELETE`**. This follows from the file contents; I did not run it.
> - **Fix:** make it an exact copy of the root file, with a test that the two are identical. This keeps old workflows working. Deleting it would break them. A composite wrapper that calls the root Action would have to pin a ref, so it can drift too. V3 then applies to both copies through the one test.
> - **Behaviour change:** an old workflow on `@main` that passes one of these inputs will start honouring it. Put that in the release note.

### V12. Negative probes pass on any 4xx — Medium

> **Claude:** `NegativeProbeResult.Passed()` accepts any status from 400 to 499. The probe labels and the catalog say "expected `400` or `422`" (CS-PROBE-001 and 002) or "`401` or `403`" (CS-PROBE-004), both marked **caught**. So a server that answers 401, 404 or 429 to a malformed body passes, although its validation may never have run. In this pass, `POST /auth/login — missing required field 'username' (expect 400/422)` passed with **401**.
> - **Now, no behaviour change:** add a known-gap row to `what-we-check.md` with a test that pins today's rule, and make the CS-PROBE rows say what is really accepted.
> - **Later:** decide whether an unexpected 4xx becomes a warning ("rejected, but not for the tested reason"). Warnings do not fail a run by default. 429 should probably mean "not checked". Needs your decision.

### V13. The Action's install step may fail on macOS runners — unverified

> **Claude:** Both Action files use `${RUNNER_OS,,}`, which needs bash 4. The system bash on macOS is 3.2, where this is a "bad substitution" error. I do **not** know which bash a GitHub macOS runner picks for `shell: bash`, and nothing has run the Action there. Lowercasing with `tr` is safe either way. A macOS leg in the CI self-test would settle it, and the leg must be able to run the next step with its empty-argument handling.

### New ideas

| # | Idea | Group | Size | Needs |
|---|---|---|---|---|
| 65 | Make `clrsig-action/action.yml` identical to the root file, with a test (V11) | CI | S | Release note |
| 66 | Record the any-4xx probe rule as a gap and correct the CS-PROBE rows (V12) | Docs, conformance | S | None |
| 67 | Make probe results stricter: warning or "not checked" for an unexpected 4xx (V12) | Conformance | S–M | Decision, release note |
| 68 | Portable lowercasing in the Action and a macOS leg in the self-test (V13) | CI | S | None |

## 8. Claude: proposed plan and task list

**Approved by the owner on 2026-11-25** ("go with your recommendations"), which settles the seven decisions below. Batch A is built. Each batch ends with the tests green, an engineering-changelog entry and backlog ticks. Commits, pushes and tags happen only when you ask.

### Batch A: the Action and the documented path (release 0.11.1)

- ☑ **A1** (65) Sync the two Action files and test that they are identical.
- ☑ **A2** (68) Replace the bash 4 syntax; add macOS to the Action self-test.
- ☑ **A3** (53) Verify the SHA-256 in the Action; fail on a missing or wrong checksum.
- ☑ **A4** (51) Exact version tags in all guides and READMEs, the `release-process.md` table, and the version test.
- ☑ **A5** (52) Generated workflow: `permissions: contents: read`, current checkout and artifact versions, no old version in the dev comment; align the guides.
- ☑ **A6** (66) Catalog gap row and corrected CS-PROBE wording, with a test.
- ☑ **A7** (57) Terminology fixes and the test (`CLAUDE.md` line 10 only if you agree).
- ☑ **A8** (59) Backlog tidy; add IDs for the leftovers and for ideas 31, 32 and 51 to 68 that are not done in this batch.
- ☑ **A9** Release 0.11.1: versions, notes, tag on your word. *Released 2026-11-25 as `v0.11.1` on `496dd58`.*

(62, the bindings check, is not needed for the release. It moves to batch E so it cannot delay the Action fixes.)

> **Claude, after batch A (2026-11-25):**
> - A correction to V7: the existing link test skips `#` anchors, so it would **not** have caught a broken guide anchor. The three uses of the renamed anchor were changed by hand and searched for afterwards.
> - V13: after the push, the macOS leg ran the **new** script with `/bin/bash` and passed (download, checksum, install). The old script was never run on macOS, so that it failed there remains an inference.
> - Passed locally: `go vet` and `go test` in all four Go modules, `tsc --noEmit`, Vitest (1,680 tests), `gofmt`. The browser tests were not run.

### Batch B: an example that follows its own spec (no product change)

- ☑ **B1** (58) Problem-details schema and media type in the spec.
- ☑ **B2** (58) Enforce the declared auth; declare 401; update scenarios, README and the demo recording as needed.
- ☑ **B3** (58) Extend `/admin/reset` to restore the users.
- ☑ **B4** (58) CI runs `conform --negative` against the example and it must pass.

> **Claude, after batch B (2026-11-25):** built.
> - Beyond the plan: the spec also lacked `text/plain` for `/health` (found by running the scenarios with spec checking) and the 400, 422 and 429 answers the server can give. `/auth/login` now answers a missing field with 422 instead of counting it as a wrong password, so each probe is rejected for what it tests; `server_test.go` asserts the status of each, which a passing probe alone does not show (CS-GAP-012).
> - The desktop scenario templates target the app's Demo Server, not the example server, and were not changed.
> - `docs/demo.tape` needed no change; pushing changes under `examples/` makes the demo workflow record the GIF again.
>
> ### V14. Ten of the eleven example scenarios do not pass `clrsig validate` (new, Medium)
>
> Their steps use `title:`, but a step's name is `label:`. The unknown key is ignored, so the steps run unnamed, and `validate` reports "step label is required". The committed files before batch B fail the same way. The two login steps added in batch B use `label:`.
>
> **Fixed 2026-11-25** (B-120, on the owner's word): 29 `title:` keys became `label:`, and three step `description:` keys in `contract-violation.yaml`, also ignored, became `notes:`. A test now rejects unknown keys and validation problems in every example. Whether the product should report unknown keys is a separate decision (B-121).

### Batch C: report consistency (0.12.0, one release note)

- ☑ **C1** (54) Spec path in all human reports, coverage lists and probe rows; add JSON `specPath`; a test that baselines are unchanged.
- ☑ **C2** (56) "not sent" in the text, Markdown, HTML and GitHub summaries.
- ☑ **C3** *(sandbox `a0918ad`; its CI passed with the released clrsig and, on a manual run, with clrsig built from main)* Update the sandbox repository's expectations (a separate repository; I will ask before pushing there).

> **Claude, after batch C (2026-11-25):** C1 and C2 built.
> - "not sent" is printed only when above zero, so every other summary line is unchanged.
> - Also covered, beyond the list: the `NEW`/`FIXED` lines of the baseline comparison and the cumulative "Never covered" line, which printed stored paths. Only the display changed.
> - A baseline recorded by the released v0.11.1 was read by the new build (2 known, 0 new), and the new build wrote the same entries.
> - **C3:** the sandbox's nightly job builds clrsig from `main`, so its change has to be in place when this is pushed, not at the release. All six suites pass with v0.11.1; with the new build exactly one pattern failed (500 unsent operations, formerly "500 failed"). It now accepts each version's exact line and passes with both. Timing and transport were run on this Windows host, not in the container.
> - The Check Conformance page was not in the list and still showed generated paths (B-122); fixed the same day, before the push.

### Batch D: first hour

- ☐ **D1** (61, first half) `conform --list`: operations and the values each needs, sending nothing. Update the CLI command catalog (`TestCatalogMatchesCobra`).
- ☐ **D2** (55) `init --spec` pre-fills the new environment files from the same data.

### Batch E: release and engineering checks

- ☐ **E1** (60) Smoke job in `release.yml`: Docker version, tagged Action, Homebrew install.
- ☐ **E2** (62) Bindings sync check in CI.

### Held

67 (after your decision) and 63, together in the next conformance release, since both change results. `doctor` waits for a proposal. 64 is dropped. The other open ideas in section 1 keep their earlier order.

### Decisions needed

| # | Question | Claude recommends |
|---|---|---|
| 1 | V1: exact tags, or a moving `v1`? | Exact tags now; moving tag at 1.0 |
| 2 | V4: spec path in reports (JUnit names change once), plus JSON `specPath`? | Yes |
| 3 | V8: make the example follow its spec? | Yes, with the spec adopting problem details |
| 4 | V10: a smoke job that can only warn? | Yes, as a job in `release.yml` |
| 5 | Change `CLAUDE.md` line 10 to the canonical term? | Yes |
| 6 | V11: keep the subdirectory Action as an identical copy? | Yes; deleting it breaks workflows |
| 7 | V12: document the gap now and decide strictness later? | Yes; later, warning for another 4xx and "not checked" for 429 |
