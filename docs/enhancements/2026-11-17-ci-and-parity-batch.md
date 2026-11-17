# CI and parity batch (backlog ideas 1–10, plus one prerequisite fix)

- **Status:** Done — items 0–10 implemented and verified; awaiting release
- **Date:** 2026-11-17
- **Source:** items 1–10 of [Future enhancements backlog](2026-11-17-future-enhancements.md); items 11–17 stay on the backlog for the project owner to think about
- **Approval:** the project owner approved items 1–10 on 2026-11-17 and asked not to be asked again

One document, one section per item, because each is small. Each section is its own acceptance checklist.

## 0. Prerequisite fix: `${NAME}` in environment files is never expanded

**Found while starting item 1.** The user guide, `CLAUDE.md` ("protect secrets via env-var references for CI/CD") and the CLI Command Builder's "Set these where the command runs" panel all say that an environment value written as `${API_TOKEN}` is read from the machine's environment variables. The code never does it: a probe with `baseUrl: "${DEMO_BASE}"` sent the literal text and failed with `unsupported protocol scheme ""`. The documented CI setup therefore does not work.

**Fix.** Expand at **resolve time** (`variables.NewResolver`), not at parse time, so saving an environment in the desktop app can never write a real secret to disk.
- `${NAME}` is replaced by the process environment variable; `$${NAME}` is a literal `${NAME}`.
- A reference to an unset variable is left as written, and `clrsig run|conform|bundle run` print one warning per missing variable naming the environment variable it feeds. It is a warning, not an error, because an unused variable should not fail a run.
- Applies to the desktop too (same resolver).
- `variables.MissingEnv(env)` and `variables.SecretValues(env)` support the warning and item 1.

Acceptance: a test proves expansion, the escape, the unset case, and that parsing and saving an environment keeps `${NAME}` untouched.

## 1. Redact secrets and extracted values in every report

`CLAUDE.md`: credentials must be redacted in reports. Today the text report prints `→ token = "…"` and the JSON report carries `extracted`.

- Reporting redacts with `***`: (a) the extracted value of any variable whose name looks secret (`token`, `secret`, `password`, `passwd`, `key`, `auth`, `session`, `credential`, `bearer`, `jwt`, `cookie`), and (b) anywhere in assertion actual values, messages and errors, any value of an environment variable marked `secret: true` (resolved value; only values of 4 characters or more, so a short secret cannot corrupt ordinary text).
- Other extracted values (for example an `id`) stay visible.
- `--show-extracted` on `run` and `bundle run` disables the extracted-value redaction for local debugging. Secret environment values are always redacted.
- Applies to text, JSON, JUnit, HTML and Markdown. This changes existing output, so the changelog says so.

Acceptance: tests for name heuristic, secret-value replacement in actual/message/error, short-secret guard, the flag, and every format.

## 2. Contracts reports from the Go core

The desktop builds the Contracts HTML, Markdown and CSV in the frontend, against rule 1; the CLI has no equivalents.
- `packages/core/reporting`: `CoverageCSV`, `CoverageMarkdown`, `CoverageHTML` for a spec's accumulated coverage and quality signals.
- `clrsig coverage list|quality` gain `--format html|markdown|csv` and `--output`.
- The desktop Export menu calls the Go service; the frontend report builders are removed.

Acceptance: Go tests replace the deleted frontend report tests; desktop export output equals the CLI output for the same data.

## 3. Several report files from one run

Repeatable `--report <format>=<file>` on `run`, `conform`, `diff` and `bundle run` writes extra reports from the same run (the screen output is still `--format`/`--output`). An unknown format or a malformed value is an error before anything runs. Not offered in the builder yet (the builder has one value per input).

## 4. `diff --fail-on`

`--fail-on breaking|warning|info` (default `breaking`). The exit code, the JUnit failures and the HTML/Markdown verdict follow it. Offered in the builder.

## 5. Folders for `validate` and `impact`

Both expand their scenario arguments with `scenario.Discover`, and gain `--no-recursive`. Same skipping rules and the same errors as `run`.

## 6. Desktop: export the run report (run-a-folder already existed)

- **Correction found while building:** the Scenarios panel already has **Run folder** on every scenario folder (`handleRunFolder`), and its results already open in the Run Results tab. Only the export half was missing.
- The Run Results tab gets **Export report** (HTML, JUnit, JSON, Markdown, text), produced by the Go reporting package (`ScenarioService.ExportRunReport`) from the latest result of each listed scenario, which the service keeps in memory. Secrets are redacted by the same code as the CLI.

## 7. `clrsig import`

`clrsig import postman|insomnia <file> --output <folder> [--dry-run]` writes request files using the same parsing and naming as the desktop import (shared in the core, not duplicated).

## 8. `--var name=value`

Repeatable on `run`, `conform` and `bundle run`; overrides or adds a plain environment variable for one run. The user guide warns that secrets belong in environment variables, not on the command line.

## 9. `markdown` report format

`--format markdown` for `run`, `conform`, `diff`, `bundle run`, built on the same document model as the HTML report, suitable for pull-request comments and `$GITHUB_STEP_SUMMARY`. Added to the catalog and the builder.

## 10. `diff --against <git-ref>`

`clrsig diff openapi.yaml --against origin/main` compares the spec at that git ref with the working copy. Needs the `git` executable (not a Go dependency); clear errors for: git missing, not a repository, unknown ref, file absent at that ref (a shallow clone hint). With `--against`, exactly one positional argument.

## Testing, documentation and risks

- Every item: unit tests, CLI tests through the real command tree, and the catalog parity test where the catalog changes.
- User guide, `README-DEVELOPER.md`, `CLAUDE.md` changelog; the format spec is unaffected (no serialized format changes).
- **Risk — output changes (item 1):** scripts that parse extracted values from JSON reports will see `***` for secret-looking names; `--show-extracted` is the escape hatch.
- **Risk — scope:** items are independent and committed one at a time, each with passing tests.

## Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Items 1–10 approved; no further questions needed | project owner |
| 2026-11-17 | Prerequisite fix (item 0) added: `${NAME}` expansion was documented but missing | assistant |
| 2026-11-17 | Item 6 reduced to the export: Run folder already existed in the desktop | assistant |
| 2026-11-17 | Implemented. Notes: `--report` and `--var` are CLI-only (not in the builder, which has one value per input); `coverage list` file formats are per-spec summaries, `coverage quality` is the full per-spec report | assistant |
