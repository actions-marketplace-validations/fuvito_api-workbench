# CLI Command Builder

- **Status:** Accepted (phase 1 in progress)
- **Date:** 2026-11-17
- **Supersedes:** the backlog item "Copy CLI command button" (a subset of this)
- **Touches:** `packages/core` (new), `apps/desktop/wails`, `apps/desktop/frontend`, `apps/cli` (tests), installer and release workflow (phase 2)

## 1. Summary

Running Clear Signal in CI means assembling a `clrsig` command by hand: which command, which flags, which paths, relative to what, with which secrets, producing which file, and what the exit code means. People get this wrong in quiet ways (the `--spec` path bug fixed in v0.5.0 silently skipped schema validation).

The **CLI Command Builder** is a guided screen in the desktop app that:

1. asks for what the user wants to do, in plain words;
2. builds the exact command, with correct relative paths and quoting for their shell, Docker or GitHub Actions;
3. explains every part: what input is expected, what output and files to expect, what each exit code means;
4. **runs the command locally from the desktop** so the user sees it work (or fail, with an explanation) before pasting it into a pipeline.

## 2. Goals and non-goals

**Goals**
- A user who has never read the CLI docs can produce a working, CI-ready command.
- The command shown is the command that runs: one source of truth, no second implementation.
- Paths are right the first time, on the user's machine and in CI.
- Every input and output is explained in place.
- The builder cannot drift from the real CLI (enforced by a test).

**Non-goals**
- Re-implementing run/conform/diff logic in the GUI (architecture rule 1).
- Writing workflow files into the repository (outside the selected project; copy only).
- A shell, terminal emulator or arbitrary command runner. Only `clrsig` is ever executed.
- Saved "command presets" or a new file format (see open decision 7).

## 3. Requirement review

| Question | Finding |
|---|---|
| Consistent with `CLAUDE.md`? | Yes. Core stays Wails-free (builder and catalog are pure Go in `packages/core`); the frontend only calls bindings; no HTTP server is opened (rule 5); no new dependencies (stdlib `os/exec`, `path/filepath`). Not on the out-of-scope list. |
| New surface area | Executing a subprocess from the desktop is new (today only "reveal in Explorer"). Mitigated by an argv array (no shell), only the resolved `clrsig` binary, a fixed working directory, a timeout and cancel. See section 6. |
| Size | **Large.** Roughly: a new core package pair, one Wails service, 3–4 React components, and (phase 2) installer and release-workflow changes to ship the CLI with the desktop app. Hence the phasing in section 7. |
| Necessary? | Yes, moderately high. The MVP definition is "run the same scenario through the CLI and GitHub Actions"; today that step is unguided. The v0.5.0 `--spec` bug is direct evidence that path semantics confuse even us. Phase 1 alone (build + explain + copy) is the smaller alternative and already covers the old "Copy CLI command" backlog item. |
| Duplicates something? | Overlaps the Help panel's CLI docs, but as interactive guidance, not a second copy: both read the same catalog (section 5.1). |
| Main risk | Catalog drift from the real CLI. Mitigated by a parity test that fails CI when a flag is added, renamed or removed without updating the catalog. |

## 4. User experience

**Entry points**
- A singleton **CLI Builder** page tab (like the Runner tab), opened from the Activity Rail.
- **Contextual buttons** that open it pre-filled: "Get CLI command" on a scenario's header bar, on a spec's Contracts view, and in the bundle import dialog.

**Layout** (form left, live result right)

```
┌ CLI Command Builder ───────────────────────────────────────────────────┐
│ I want to…  (•) Run a scenario  ( ) Check an API against its spec      │
│             ( ) Detect breaking changes   ( ) …                         │
├───────────────────────────────┬────────────────────────────────────────┤
│ Scenario   [scenarios/user-lifecycle.yaml ▾]  ✓ valid, 4 steps         │
│ Environment[environments/ci.yaml ▾]  needs: API_BASE_URL, API_TOKEN     │
│ Spec       [openapi.yaml ▾]  ✓ 8 operations                             │
│ Report     (•) html ( ) junit ( ) json ( ) text                         │
│ Save to    [reports/user-lifecycle.html]                                │
│ Run from   [repo root ▾]   ⓘ paths are written relative to this folder  │
│ ☑ coverage   ☐ fail fast …                                              │
├───────────────────────────────┴────────────────────────────────────────┤
│ [ Terminal ▾ bash | PowerShell | cmd ] [ Docker ] [ GitHub Action ]     │
│ clrsig run scenarios/user-lifecycle.yaml \                              │
│   --env environments/ci.yaml --spec openapi.yaml --coverage \           │
│   --format html --output reports/user-lifecycle.html        [Copy]      │
│                                                                         │
│ ▸ Explain   what it does · input expected · output produced · exit codes│
│ ▸ Warnings  ⚠ environment needs API_TOKEN set in CI (secret)            │
│                                         [ ▶ Test it here ]  [Cancel]   │
│ Result: exit 0 · 1.2 s · wrote reports/user-lifecycle.html [Open report]│
└─────────────────────────────────────────────────────────────────────────┘
```

**Flow:** pick a goal → fill the fields (file pickers list only valid files from the workspace) → the command and explanation update live → copy, or **Test it here**.

## 5. Design

### 5.1 Command catalog (single source of truth)

New package `packages/core/clispec`: a typed description of each command — summary, when to use it, arguments and flags, exit codes, produced outputs, examples.

Each input has a **kind** so the UI and builder can be smart:

| Kind | Meaning | Examples |
|---|---|---|
| `file` (with accepted types) | existing local file, validated | scenario, environment, spec, bundle |
| `fileOrURL` | local file or http(s) URL | `--spec`, `conform <spec>` |
| `dir` | existing folder | `--workspace` |
| `outputFile` | file that will be created | `--output` |
| `url` | `--base-url` | |
| `enum` | fixed values, per command | `--format` (html only where supported) |
| `int`, `bool`, `string` | | `--timeout`, `--coverage` |

Each input also records **what its relative path is relative to** (`cwd` or `containing file`), its default, a plain-language explanation, and rules: `requires` (`--coverage` needs a spec), `conflictsWith`, `onlyFor` (format `html` is valid for `run`, `conform`, `diff`, `bundle run`).

**Anti-drift test** (in `apps/cli`, where the cobra tree lives): for every catalogued command, each catalog flag exists in cobra with the same name, shorthand and default, and every non-hidden cobra flag is in the catalog. Adding a flag without updating the catalog fails CI.

The Help panel and docs can later be generated from the same catalog.

### 5.2 Builder (pure Go, `packages/core/clibuild`)

Input: command, values, and a context (workspace root, run-from directory, target platform). Output:
- `Argv []string` — the canonical argument vector (this is what actually runs);
- rendered text per target (bash, PowerShell, cmd, Docker, GitHub Action);
- `Warnings` and `Errors` (located, with a suggested fix);
- an **explanation**: per-token meaning, expected input, produced output, exit codes;
- `Needs`: environment variables the command requires (see 5.4).

### 5.3 Smart paths and names

- **Run-from directory.** Default: the repository root (nearest ancestor with `.git`), else the workspace folder. All paths are shown relative to it, with forward slashes (backslashes for `cmd`).
- **Right semantics per input.** `--spec` and other flags are relative to the current directory; a `specFile:` inside a scenario is relative to that file. The builder knows which is which (the exact trap behind the v0.5.0 fix) and never rewrites a file's own contents.
- **Outside the repo?** A path outside the run-from directory is emitted absolute with a warning: "this will not exist on a CI runner".
- **Docker:** paths are rewritten under the mount (`/workspace/...`) and the mount flag is generated. **GitHub Action:** paths are relative to `$GITHUB_WORKSPACE`, mapped onto the Action's inputs where the Action supports them, otherwise falls back to a `run:` step.
- **Names.** Output file suggestions follow `reports/<command>-<slug>.<ext>` with the extension taken from the format (`html`, `xml` for junit, `json`, `txt`); slugified; warns if the file exists; notes that parent folders are created automatically.
- **Quoting.** Correct per target (spaces, quotes, `$`, backticks, `%`, Unicode), covered by golden tests. This is where hand-written commands usually break.

### 5.4 Validation with the real parsers

The builder reuses the same Go code the CLI uses to check inputs, before anything runs: parse the scenario (same checks as `clrsig validate`), load the spec, parse the environment, resolve `requestRef`s. Findings appear next to the field ("✓ valid, 4 steps" or the located error).

- **Secrets and variables.** Values like `${API_TOKEN}` in an environment file are listed as "set this in CI", with the exact `env:` / `secrets.` snippet for the chosen target. Secret values are never shown or stored.
- **Cross-field rules** from the catalog are enforced live ("`--coverage` needs a spec").
- **Optional reachability probe** of the base URL, off by default.

### 5.5 "Test it here"

Runs the same `Argv` the user is about to copy, locally, and explains the outcome.

- **How:** execute the real `clrsig` binary (see open decision 1). Rejected alternative: call the core packages in-process. It would not test the real command line (quoting, flags, paths), and the CLI's logic currently lives in the cobra handlers, so it would need a large refactor first.
- **Safety:** `os/exec` with an argv array (never a shell); only the resolved `clrsig` binary; working directory = the run-from directory; timeout (default 5 minutes) and a Cancel button; output size cap; known secret values masked in streamed output.
- **Environment variables:** inherited from the desktop process, plus per-run overrides the user types for the variables listed in `Needs`. Overrides live in memory only.
- **Production guard:** before running, if the selected environment looks like production or the scenario sends mutating requests, ask for confirmation (reusing the app's existing "warn before destructive requests against production" behaviour).
- **Result panel:** exit code with its meaning from the catalog ("1 = assertion failures — expected if a check is meant to fail"), duration, streamed stdout/stderr, files created with **Open report**, and "what this means for CI".
- **Binary discovery order:** (1) a path set in Settings; (2) next to the desktop executable (`clrsig.exe` beside the app, `Contents/MacOS` on macOS); (3) `PATH`; otherwise a guided "CLI not found" state (install via Homebrew/Scoop/download, with a Re-check button).
- **Version lock:** `clrsig --version` must match the desktop version; a mismatch warns, because the catalog describes that version's flags.

### 5.6 Architecture

```
Frontend (CLI Builder page) → Wails bindings → CliBuilderService (Wails)
                                                  ├─ clibuild + clispec  (packages/core, pure Go)
                                                  ├─ existing parsers (scenario, spec, environment)
                                                  └─ process runner (os/exec, Wails events for streaming)
```

The domain/core packages never depend on Wails; the runner lives in the Wails layer because it streams through Wails events. The CLI binary is the only thing executed.

## 6. Security

- No new network listener.
- No shell: argv exec, so values cannot inject commands.
- Only the resolved `clrsig` binary is executable from this feature; the path is checked and shown.
- File access limited to the selected workspace and explicitly picked files (Security Requirements, section 7 of `CLAUDE.md`).
- Secrets are never persisted, never written into the generated text (placeholders are used), masked in streamed output.

## 7. Phasing and acceptance criteria

| Phase | Delivers | Acceptance |
|---|---|---|
| **1 — Build and explain** (order: catalog and builder with the default text output → page and entry points → verify end to end → **last: report formats `junit`, `html`, `json` and output-file naming**) | `clispec` catalog (+ parity test), `clibuild`, builder page, commands `run`, `conform`, `diff`; Terminal tabs (bash, PowerShell, cmd); path/name logic, validation, explanations, copy; entry points from the scenario header and the Rail | A user can produce a correct `clrsig run` command for a workspace scenario in under a minute; paths are right from the repo root; quoting golden tests pass; the parity test fails when a flag is added without a catalog entry |
| **2 — Test it here** | process runner, binary discovery, streaming result panel, exit-code explanations, Open report, prod guard; CLI shipped with the desktop app (installer, macOS bundle, Linux) and version check | Pressing Test it runs the exact copied command; cancel and timeout work; a failing scenario is explained, not just shown; the installed app finds its CLI with no PATH setup |
| **3 — More targets and commands** | `bundle run`, `impact`, `coverage quality`, `generate`, `request`, `validate`; Docker and GitHub Action targets; env-var checklist for CI | Docker and Action snippets run unmodified in a sample repo (checked in CI) |

Phase 1 alone replaces the backlog "Copy CLI command" item.

## 8. Testing

- **Unit (Go):** builder tables for paths across Windows/Unix forms and each target; quoting golden tests; rule enforcement (`requires`, `onlyFor`); validation messages.
- **Parity test** (`apps/cli`): catalog ↔ cobra tree.
- **Integration (Wails module):** build `clrsig` into a temp dir, run a catalog-built command against an `httptest` server, assert exit code, output file and explanation; cancel and timeout cases.
- **Frontend:** Vitest for the form logic; Playwright (mocked bridge) for the flow and a screenshot for the user guide.
- **Docs example check (phase 3):** run the generated Docker and Action snippets in CI against the demo server.

## 9. Documentation impact

- User guide: new section "CLI Command Builder" (and a pointer from CLI usage).
- `README-DEVELOPER.md`: the catalog, the parity test, and "when you add a flag, update `clispec`".
- `CLAUDE.md`: changelog entry; add the catalog rule to the CLI notes.
- `format-spec.md`: unchanged (no serialized format changes).
- README: one line under the CI/CD bullet.

## 10. Risks

| Risk | Mitigation |
|---|---|
| Catalog drift | Parity test in CI; one source for docs and UI |
| Shipping the CLI inside the desktop app (two binaries, version skew) | Version check at runtime; same release job builds both; discovery order with a clear "not found" state |
| Shell quoting bugs | Golden tests per shell; argv exec for Test it, so quoting affects only the copied text |
| Scope creep (presets, workflow-file writing, more targets) | Non-goals above; phase gates |
| A test run hits a production API | Environment/method guard with confirmation; clear "this sends real requests" banner |

## 11. Open decisions

1. **How "Test it here" runs.** *Recommend:* execute the real `clrsig` binary. Alternative: in-process via a large refactor of the CLI into shared services.
2. **Ship the CLI inside the desktop installer/app?** *Recommend:* yes (Windows installer, macOS bundle, Linux alongside) so Test it works with no setup; fall back to `PATH` and a Settings override.
3. **Phasing.** *Recommend:* phase 1 then phase 2 as consecutive releases. Alternative: phases 1 and 2 together in one release.
4. **Entry points.** *Recommend:* a Rail page tab plus contextual "Get CLI command" buttons. Alternative: a dialog only.
5. **Commands in phase 1.** *Recommend:* `run`, `conform`, `diff` (the CI core); the rest in phase 3.
6. **Targets in phase 1.** *Recommend:* terminal only; Docker and GitHub Action in phase 3.
7. **Saved presets.** *Recommend:* no. It needs a new serialized format (schemaVersion, format-spec, migration) for little gain over copy-paste.
8. **Name.** *Recommend:* "CLI Command Builder".

## 12. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Proposal written; awaiting decisions on section 11 | assistant |
| 2026-11-17 | All recommendations in section 11 accepted (real binary, ship CLI with the app, phase 1 then 2, Rail tab plus contextual buttons, `run`/`conform`/`diff`, no presets, name "CLI Command Builder") | project owner |
| 2026-11-17 | Addition: the report formats (`text` first; then `junit`, `html`, `json` with `--output` naming) are added as the **last step of phase 1**, after the basic flow has been verified working | project owner |
