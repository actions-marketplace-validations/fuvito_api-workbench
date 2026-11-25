# Clear Signal 0.12.0 (draft; delete after publishing)

`clrsig conform` reports now agree with each other. No check, verdict or exit code changes.

## Two changes to how reports read

- **One name per operation.** Markdown, HTML, JUnit and the coverage lists now write the path as your spec does, `GET /users/{id}`, as the text report already did. Before, they showed the generated request's `GET /users/{{id}}`. So do the `NEW` and `FIXED` lines of a baseline comparison and the cumulative "Never covered" line.
  - **JUnit test names change once** for operations with path parameters. A CI tool that keeps test history will treat them as new tests.
  - **JSON:** `path` is unchanged. A new `specPath` field holds the spec's path, on each operation and each coverage entry.
  - **Baselines keep working.** Files are identified and written as before.
- **"Not sent" is not "failed".** An operation that was never sent for lack of a value used to be counted as failed in the summary line. It is now counted apart: `4 passed · 0 failed · 4 not sent · 0 skipped`. It still fails the run. The part appears only when at least one operation was not sent, so other summary lines are unchanged. This applies to the text, HTML, Markdown and GitHub output; JUnit totals and JSON are unchanged.

**If you parse the summary line** of runs that have unsent operations, expect the new part.

## Desktop

The Check Conformance page names operations the same way: in its lists, problem cards, copied issues, baseline list and progress line.

## Also since 0.11.1

- The example server follows its own spec, so `conform --negative` passes against it. It now requires its demo token for `GET` and `POST /users`.
- The example scenarios named their steps with keys the format does not have; their steps now show names and all of them pass `clrsig validate`.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.11.1...v0.12.0
