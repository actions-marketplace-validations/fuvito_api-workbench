# Run scenarios from folders (`clrsig run <file-or-folder>...`)

- **Status:** Done — implemented and verified (CLI, core, builder); awaiting release
- **Date:** 2026-11-17
- **Related:** [CLI Command Builder](2026-11-17-cli-command-builder.md) (the builder follows this change)
- **Touches:** `packages/core/scenario` (discovery), `apps/cli` (`run`), `packages/core/clispec` and `clibuild`, desktop builder UI, user guide

## 1. Summary

`clrsig run` takes exactly one scenario file. A folder fails with an unhelpful error and several files are rejected, although the desktop app's **Run All** and `clrsig bundle run` already run many scenarios. CI users want "run every scenario under `scenarios/`" as one step with one report.

`clrsig run` will accept one or more files **or folders**. A folder runs every scenario inside it and its sub-folders, and the results are combined into one report with one exit code. The CLI Command Builder gains a matching "Scenario or folder" input.

## 2. Goals and non-goals

**Goals**
- One command runs many scenarios; a single file behaves exactly as today.
- One combined report in every format (text, json, junit, html) and one exit code.
- The same scenario discovery in the CLI and the desktop Run All (one implementation in the core).
- Failures are loud: nothing is silently skipped that the user asked for.

**Non-goals**
- Parallel execution (scenarios run one after another, in a stable order).
- Glob patterns inside `clrsig` (the shell expands `scenarios/*.yaml` into several file arguments, which this change allows).
- Selecting scenarios by tag or name (a later enhancement if needed).

## 3. Requirement review

| Question | Finding |
|---|---|
| Consistent with `CLAUDE.md`? | Yes. Discovery lives in the Go core, the CLI stays an interface (rule 1). "CLI parity with the desktop" is a stated goal. No new dependency (`yaml.v3` is already used). |
| Backward compatible? | Yes. One file gives the same output and exit codes as before. Only inputs that failed before (folders, several files) gain meaning. Minor release. |
| Size | Medium: one core function, a refactor of `run`, catalog and builder changes, tests, docs. |
| Necessary? | Yes. Running a whole suite in CI is the main use; today it needs a shell loop (many processes, many reports, hand-written exit-code handling) or an exported bundle that goes stale. |
| Smaller alternative | A shell-loop snippet in the builder. Rejected: separate reports, per-shell syntax, and fragile exit handling. |

## 4. Design

### 4.1 Discovery (`scenario.Discover`)

`Discover(paths []string, recursive bool)` expands arguments into scenario file paths:

- A **file** is taken as given (it must be a scenario; an invalid one is an error).
- A **folder** contributes files ending in `.yaml`, `.yml` or `.json`, searched recursively by default. Hidden folders (`.git`, …) and `node_modules` are skipped.
- A file found by scanning a folder counts as a scenario only if it has a top-level `steps` key. Environment, request and spec files that share the folder are **skipped and reported as skipped**, not run and not failed. A scenario file that has `steps` but is otherwise invalid is **an error**.
- Results are de-duplicated and sorted by path, so the order is stable.
- No scenarios found is an error.

### 4.2 `clrsig run`

- `Use: run <scenario-file-or-folder>...`; at least one argument.
- `--no-recursive` searches only the top level of each folder.
- `--fail-fast` stops after the first scenario that fails.
- `--env`, `--spec`, `--coverage`, `--workspace`, `--timeout` apply to every scenario. A scenario's own `specFile:` is still relative to that scenario file; `--spec` is relative to the current directory.
- **One scenario:** unchanged output. **Several:** each scenario's result, then an overall summary; coverage (if requested) is aggregated per spec; with `--workspace` it accumulates exactly as separate runs would.
- **Reports:** `text` prints each scenario then `N/M scenarios passed`; `junit` is one document with a test suite per scenario; `html` is one page with a section per scenario; `json` is `{passed, durationMs, scenarios: [...]}`, the shape `bundle run` already uses. Existing renderers (`JUnitMulti`, `HTMLMulti`) are reused.
- **Exit codes:** `0` every scenario passed; `1` at least one failed; `2` an input could not be read or parsed, nothing was found, or a scenario file is invalid.
- Skipped non-scenario files are listed on stderr so a stray file is visible, never silent.

### 4.3 CLI Command Builder

- The `run` input becomes "Scenario or folder" (kind: file or folder, still required). The workspace list offers each scenario file, each folder that contains scenarios, and "All scenarios in this workspace".
- A live preview: "12 scenarios found in `scenarios/` (2 sub-folders), 1 skipped file", the invalid files by name, and an explanation that the report and exit code cover all of them.
- `--no-recursive` and `--fail-fast` appear under More options. The suggested report name uses the folder name (`reports/run-scenarios.html`).

## 5. Phasing and acceptance criteria

1. **Core and CLI:** discovery, `run` with many inputs, combined reports, new flags; the parity test updated; builder commands still run through the real command tree.
2. **Builder:** folder input, preview, new flags, suggestions.
3. **Docs:** user guide (CLI usage and builder), `README-DEVELOPER.md`, `CLAUDE.md` changelog.

Acceptance: a folder with nested scenarios runs in order and reports once in every format; a passing single file is byte-for-byte unchanged; invalid scenarios exit 2; non-scenario files are skipped with a note; the builder produces a folder command that runs.

## 6. Testing

- Discovery table tests: files, folders, nesting, `--no-recursive`, hidden and `node_modules` skipped, non-scenario skipped, invalid scenario error, empty folder error, de-duplication, ordering.
- CLI tests with an `httptest` server: single file unchanged; folder with passing and failing scenarios; exit codes; `--fail-fast`; each format; coverage aggregation; `--workspace` accumulation.
- Parity test (catalog ↔ cobra) covers the new flags; builder integration test runs a folder command.
- Frontend and browser tests for the new input and preview.

## 7. Documentation impact

User guide (CLI usage, builder), `README-DEVELOPER.md` (run reference), `CLAUDE.md` changelog. `format-spec.md` is unchanged (no serialized format changes).

## 8. Risks

| Risk | Mitigation |
|---|---|
| A folder holds files that are not scenarios | Detect by the `steps` key; skipped files are listed, not run, not failures |
| A broken scenario is mistaken for a non-scenario | A file with `steps` that fails to parse or validate is an error (exit 2) |
| Large folders take long | Sequential by design; `--fail-fast`; per-step `--timeout` |
| Combined reports differ from single-run reports | Reuse the existing multi-scenario renderers; a single file keeps the old renderers |

## 9. Open decisions

None. The four decisions below were taken by the project owner on 2026-11-17.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Folders are searched recursively by default (`--no-recursive` to opt out) | project owner |
| 2026-11-17 | One combined report and one exit code | project owner |
| 2026-11-17 | An invalid scenario file makes the run fail (exit 2) | project owner |
| 2026-11-17 | `--fail-fast` is added | project owner |
| 2026-11-17 | Project owner approved building it without further questions | project owner |
| 2026-11-17 | Implemented: `scenario.Discover`, `clrsig run` with files/folders/`--fail-fast`/`--no-recursive`, combined reports, builder folder input with live preview | assistant |
