# Future enhancements backlog

- **Status:** Items 1–10 approved and implemented (see [CI and parity batch](2026-11-17-ci-and-parity-batch.md)); items 11–17 remain ideas for the project owner to think about
- **Date:** 2026-11-17
- **Purpose:** a place to keep ideas that came out of building the CLI Command Builder, HTML reports and folder runs, so they are not lost. Nothing here is a commitment.

**How an idea graduates:** when you want one built, it gets its own dated proposal in this folder (see [README](README.md)), including the requirement review, and is agreed before any code is written. Sizes: **S** ≈ a day or less, **M** ≈ a few days, **L** ≈ a week or more. "Evidence" says where the idea came from, so you can judge how real the need is.

Ideas already written up elsewhere are only linked here: the remaining phases of the [CLI Command Builder](2026-11-17-cli-command-builder.md) ("Test it here", shipping the CLI with the desktop app, more commands, Docker and GitHub Action targets, multi-step pipeline snippets), and the older items in the project backlog (screenshots in the user guide, Wails v3, Analytics tab, mock server, step `retry` / `timeoutMs` in the runner, `bundle run --workspace`).

## Summary

| # | Idea | Theme | Size | Suggested |
|---|---|---|---|---|
| 1 ✅ | Redact secrets and extracted values in every report | Consistency | S | **First** |
| 2 ✅ | Contracts reports from the Go core, not the frontend | Consistency | M | Soon |
| 3 ✅ | Several report files from one run | CI | S–M | Soon |
| 4 ✅ | `--fail-on` severity for `diff` | CI | S | Soon |
| 5 ✅ | Folders for `validate` and `impact` | Parity | S | Soon |
| 6 ✅ | Run a folder from the desktop, and export the run report | Parity | M | Soon |
| 7 ✅ | `clrsig import` (Postman / Insomnia) | Parity | S–M | Next |
| 8 ✅ | `--var name=value` overrides | CI | S | Next |
| 9 ✅ | `markdown` report format (PR comments, job summaries) | CI | M | Next |
| 10 ✅ | `diff` against a git ref | CI | M | Next |
| 11 | Baseline mode for `conform` | Adoption | M–L | Later |
| 12 | `validate --strict` (scenario quality lint) | Quality | S–M | Later |
| 13 | `clrsig init` (workspace and CI scaffolding) | Adoption | M | Later |
| 14 | Select scenarios by tag or name | Scale | M | Later |
| 15 | Parallel scenario execution | Scale | M | Later |
| 16 | Project config file (`.clrsig.yaml`) | Ergonomics | L | Think twice |
| 17 | Data-driven scenarios | Power | L | Think twice |

## Consistency fixes (not really "features")

### 1. Redact secrets and extracted values in every report — S, first

- **Problem.** `CLAUDE.md` requires credentials to be redacted in request history, logs and reports. The text report prints extracted values (`→ token = "…"`), and the JSON report carries them. Only the HTML report leaves extracted values out, on purpose.
- **Idea.** One redaction rule in the reporting package, applied by every format: values whose names look like secrets, values of environment variables marked `secret: true`, and extracted values from `Authorization`/token-like sources are replaced with `***`. A flag (`--show-extracted`) could opt back in for local debugging.
- **Notes.** This changes the existing text and JSON output, so it needs a clear changelog note. It is the only item here that fixes a stated rule.

### 2. Contracts reports from the Go core — M

- **Problem.** The desktop Contracts page builds its HTML, Markdown and CSV exports in the frontend (`CoverageTab.tsx`, `coverage-report`), which goes against "never implement logic in the frontend" (rule 1). The CLI's `coverage quality` has only text and JSON, so the two can disagree.
- **Idea.** Move the report builders to `packages/core/reporting` (reusing the HTML document model) and call them from both. `clrsig coverage list|quality` then gains `--format html|markdown|csv`, identical to the desktop export.
- **Notes.** No new dependency; removes duplicated logic; the existing frontend tests move to Go.

## CI and automation

### 3. Several report files from one run — S–M

- **Problem.** To get both a JUnit file for the dashboard and an HTML file for humans, the user runs the scenarios twice (our docs say "run the command once per format"). That doubles the requests against the API.
- **Idea.** A repeatable `--report <format>=<file>` flag (`--report junit=reports/a.xml --report html=reports/a.html`), with the screen output still controlled by `--format`. The builder would offer it as "Also save as…".
- **Notes.** Backward compatible; `--format`/`--output` stay.

### 4. `--fail-on` for `diff` — S

- **Problem.** `diff` fails only on breaking changes. Teams that treat "potentially breaking" (`WARN`) as a stop sign cannot say so.
- **Idea.** `--fail-on breaking|warning|info` (default `breaking`, today's behaviour), with the exit code and the JUnit failures following it.

### 8. `--var name=value` overrides — S

- **Problem.** Changing just the base URL for one CI job means writing another environment file or exporting variables the file refers to as `${NAME}`.
- **Idea.** A repeatable `--var baseUrl=https://staging.example.com` that overrides or adds an environment variable for one run. The user guide would say secrets belong in environment variables, not on the command line (process lists are visible).
- **Notes.** The builder lists them under More options and never offers to type a secret.

### 9. `markdown` report format — M

- **Problem.** Pull-request comments and the GitHub job summary (`$GITHUB_STEP_SUMMARY`) render Markdown, not JUnit or HTML.
- **Idea.** `--format markdown` for `run`, `conform`, `diff` (and `bundle run`), built on the same document model as the HTML report. The GitHub Action could append it to the job summary automatically.

### 10. `diff` against a git ref — M

- **Problem.** The natural pull-request check is "did this branch break the spec relative to `main`?", which today needs a manual `git show` into a temp file.
- **Idea.** `clrsig diff openapi.yaml --against origin/main` reads the base version of the same path from git.
- **Requirement review.** Needs the `git` executable on the machine; no Go dependency. Worth a short design note on failure modes (shallow clones in CI, missing ref).

## Parity between desktop and CLI

### 5. Folders for `validate` and `impact` — S

- **Problem.** `run` now accepts folders, but `validate` and `impact` still take only files (the shell can expand a glob, but not recursively on every platform).
- **Idea.** Both use `scenario.Discover`, with the same skipping rules and `--no-recursive`. The builder's catalog and preview extend to them.

### 6. Run a folder from the desktop, and export the run report — M

- **Problem.** The desktop **Run All** runs every scenario in the workspace; there is no way to run just one folder, and no way to save the result as the HTML or JUnit report the CLI produces.
- **Idea.** A "Run this folder" action on scenario folders (using the same discovery as the CLI), and **Export report** (HTML / JUnit / JSON) on the Run Results tab, produced by the Go reporting package.

### 7. `clrsig import` (Postman / Insomnia) — S–M

- **Problem.** The desktop can import Postman and Insomnia collections (`importer` package); the CLI cannot, so a team cannot convert their collections inside a pipeline or a script.
- **Idea.** `clrsig import postman|insomnia <file> --output <folder>` writing request files, with `--dry-run` like `generate`. Pure reuse of existing code.

## Adoption and quality

### 11. Baseline mode for `conform` — M–L

- **Problem.** Pointing `conform` at a long-lived API often reports hundreds of violations at once, so the check is switched off instead of fixed gradually.
- **Idea.** `--write-baseline file` records today's violations; `--baseline file` fails only on violations that are not in it, and reports fixed ones, so quality can only ratchet up.
- **Notes.** Needs a small versioned baseline format (schemaVersion, documented in `format-spec.md`) and a stable violation identity. Worth a proper design.

### 12. `validate --strict` — S–M

- **Idea.** Optional lint rules beyond "does it parse": steps with no assertions, extractions never used, `requestRef`s that do not resolve, variables used but defined nowhere (the builder already warns about this one). Findings as warnings, `--strict` turns them into exit code 1.

### 13. `clrsig init` — M

- **Idea.** `clrsig init` creates the workspace layout (`scenarios/`, `environments/`, `requests/`) and `clrsig init github-actions` prints a ready workflow (or writes it with an explicit flag). It shares its snippets with the builder's future Docker and Action targets.
- **Notes.** Writing into the repository is something the user must ask for explicitly; printing to the screen stays the default.

## Scale and power

### 14. Select scenarios by tag or name — M

- **Idea.** `--tag smoke` and `--name pattern` for folder runs. Scenarios need an optional `tags` field.
- **Requirement review.** A serialized field change: schemaVersion handling, `format-spec.md`, and the AI prompt must be updated in the same commit.

### 15. Parallel scenario execution — M

- **Idea.** `--parallel N` for folder runs (`run`, `bundle run`), opt-in. A worker count, not an on/off flag: "all at once" would send hundreds of simultaneous flows at the API.
- **Design discussed 2026-11-17 (not decided).**
  - `0` and `1` both mean sequential, which stays the default; a hard cap of 16 (an error above it) and never more workers than scenarios.
  - The combined report stays in path order and the exit code is unchanged, so the same files give the same report at any worker count.
  - The desktop Run All stays sequential at first (its progress display assumes one scenario at a time).
- **Concerns and how to handle them.**
  1. *Shared data between scenarios* (two scenarios creating the same user, one deleting what another reads) is the main risk: opt-in, documented as unsafe for scenarios that share state; a later "run alone / run together" marker would change the scenario format, so it is out of the first version.
  2. *Server load* (429s, timeouts, inflated per-scenario timings): opt-in default and the cap.
  3. *Coverage recording with `--workspace`* is a read-modify-write on a shared file: record after all scenarios finish, in path order (also deterministic).
  4. *Shared CLI state:* lock the spec cache and collect per-scenario warnings to print in order. Each scenario already has its own HTTP client, cookie jar and resolver; secrets are read-only.
  5. *`--fail-fast`* can only mean "stop starting new scenarios"; scenarios already running finish, so which ones ran varies between runs. Document it.
  6. *Proof:* run the tests under the race detector and assert an identical report for 1, 4 and 16 workers.
- **Requirement review.** Scenarios often create and delete data and may depend on each other's side effects. It must be opt-in, with the reports and exit codes unchanged, and documented as unsafe for scenarios that share state. Needs evidence of demand before it is built.

## Think twice before building

### 16. Project config file (`.clrsig.yaml`) — L

- **Idea.** Defaults for environment, spec, report folder and format so commands get short.
- **Why hesitate.** A new serialized format with versioning and migration rules, and a second way to say the same thing as the command line. The builder already makes long commands easy to produce. The command builder's "saved presets" were rejected for the same reason.

### 17. Data-driven scenarios — L

- **Idea.** Run a scenario once per row of a CSV or JSON file.
- **Why hesitate.** A large change to the scenario format and the runner, and it moves the product toward load and data testing, which the project has kept out of scope. Better evidence of demand first.

## Suggested order

1. **#1 redaction** (a stated rule is currently broken), then **#4**, **#5**, **#3** (small, each makes CI use nicer).
2. **#2** and **#6** (remove the last logic that lives in the frontend, and give the desktop the same reports as the CLI).
3. **#7**, **#8**, **#9**, **#10** as CI usage grows.
4. **#11–#15** when real users ask for them. **#16–#17** only with evidence.

Alongside these, the CLI Command Builder's phase 2 ("Test it here") is the larger step that makes the builder complete.

## Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Backlog written; nothing approved. The project owner will consider the ideas as future enhancements | assistant |
| 2026-11-17 | Items 1–10 approved ("go with all above other than later items"); built as one batch | project owner |
| 2026-11-17 | Items 11–17 ("Later" and "Think twice") stay on the backlog until the project owner decides | project owner |
