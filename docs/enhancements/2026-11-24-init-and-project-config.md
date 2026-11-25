# `clrsig init` and the project config file (0.9.1)

- **Status:** Done (phases 1 to 4 built 2026-11-24; ships in the next release)
- **Date:** 2026-11-24
- **Backlog:** B-021 (`clrsig init`), B-089 (project config file); related: B-015 (CLI Command Builder snippets), B-084 (`conform --setup`) and B-020 (`validate --strict`), which are separate and need no proposal
- **Depends on:** the schema migration policy (`docs/process/schema-migration-policy.md`), accepted for new persisted files

## 1. Summary

The first hour with the CLI has three avoidable costs:

1. **The same flags every time.** A real `conform` command is `clrsig conform specs/openapi.yaml --env environments/ci.yaml --base-url http://localhost:8080 --read-only --baseline conform-baseline.yaml --report junit=reports/conform.xml ...`. It is copied into scripts, workflows and the CLI Command Builder, and the copies drift.
2. **A blank folder.** There is no command that creates the layout the desktop app and the guides assume (`environments/`, `scenarios/`, `requests/`), and the GitHub workflow is copied by hand from `docs/guides/ci-integration.md`.
3. **No place to say what a project is.** The desktop remembers the active environment per workspace in `~/.clrsig/config.json`, outside the repository, so a teammate or a CI runner never sees it.

The idea: a small, versioned **`clrsig.yaml`** at the project root that holds the defaults for the flags that repeat, plus **`clrsig init`**, which creates it (and the folders) and prints the next command, and **`clrsig init github-actions`**, which prints a ready workflow. After `init`, `clrsig conform` with no arguments works.

## 2. Goals and non-goals

**Goals**
- `clrsig conform` (and `run`, `validate`) work from a project folder with no flags, using `clrsig.yaml`.
- A flag always wins over the file; the file always wins over the built-in default; the run says when it used a file.
- `clrsig init` is safe to run in an existing project: it never overwrites or deletes, and running it twice changes nothing the second time.
- One core package (`packages/core/projectconfig`) reads, validates and resolves the file; the CLI and later the desktop app call it (architecture rule 1).
- The file is a normal persisted format: `schemaVersion`, validated before use, YAML or JSON, documented in `format-spec.md`, migration rules per the policy.

**Non-goals**
- Secrets in the file. There is no `vars` key; secrets stay `${NAME}` references in environment files (`CLAUDE.md` section 7).
- A starter scenario written from guesses. `init` does not invent endpoints.
- Per-user settings (theme, ports, proxy): those stay in `~/.clrsig/settings.json`.
- Replacing environment files, bundles or scenarios, and any plugin or scripting mechanism.
- Writing a CI workflow without being asked; `init` never touches `.github/` by default.

## 3. Requirement review (`CLAUDE.md` section 6)

| Question | Finding |
|---|---|
| Conflicts with a requirement? | No. Local files only, no server (rule 5), no new dependency (`yaml.v3` is in the locked list), atomic writes for everything `init` writes (rule 7). The file is not out of scope (no cloud, team sync or marketplace). |
| Changes a serialized format? | **Yes: a new persisted file.** `schemaVersion: "1"`, strict validation, `format-spec.md` in the same commit, and the policy's rules for hand-written files. |
| Larger than it looks? | Yes in reach, not in logic: every command that takes `--spec`, `--env`, `--base-url`, `--report`... gets a resolution step. Contained by one function in the CLI (`applyProjectConfig`) that fills flags that were not set (`cmd.Flags().Changed`), and by a core package with its own tests. The CLI Command Builder and the Action are not changed in the first phase (section 4.6). |
| Unnecessary? | Partly overlapping with shell scripts and `package.json` scripts, which is why the file stays small and declarative. The smaller alternative (only `init`, no config) would still leave the repeated flags; the other (only config, no `init`) leaves the blank folder. Both parts are small and share one package, so they ship together. |
| New command family? | `init` (with one subcommand) and, in phase 3, `config show`. Both are named here, so this document is the proposal for them. |

## 4. Design

### 4.1 The file

Name: `clrsig.yaml`. `clrsig.yml` and `clrsig.json` are equivalent (one serialization model, two paths; rule 4). If more than one exists in a folder, that is an error that names them.

```yaml
schemaVersion: "1"

spec: specs/openapi.yaml            # the conform spec argument (not applied to run: a scenario's specFile decides)
environment: environments/ci.yaml   # --env
baseUrl: http://localhost:8080      # --base-url

conform:
  readOnly: true                    # --read-only
  negative: false                   # --negative
  failOn: failure                   # --fail-on (failure | warning)
  requestTimeoutMs: 10000           # --request-timeout
  maxResponseTimeMs: 0              # --max-response-time
  maxResponseTimeFor:               # --max-response-time-for
    - /reports=2000
  tags: []                          # --tags
  paths: []                         # --paths
  baseline: conform-baseline.yaml   # --baseline

run:
  scenarios: [scenarios]            # default arguments of run, validate and impact
  failFast: false                   # --fail-fast

reports:                            # --report (repeatable)
  - junit=reports/conform.xml
```

Rules:
- Every key is optional; an empty file with only `schemaVersion` is valid.
- Unknown keys are errors with a suggestion, except keys that start with `x-`, which are preserved and ignored (`CLAUDE.md` section 8).
- **Paths are relative to the folder of the config file**, not to the current directory, so the same command works from a subfolder and in CI. Absolute paths are allowed.
- Values are checked with the same functions the flags use (`checks.ParseFailOn`, `conformance.ParsePathLimits`, report `format=file` parsing), so there is one definition of "valid".
- `${NAME}` is **not** expanded in `clrsig.yaml` (it is for environment files); a value containing it is rejected with a pointer to environment files. This keeps secrets out by construction.

### 4.2 Finding it

1. `--config <file>` if given (an error if it does not exist).
2. Otherwise start in the current directory and walk up. At each folder look for `clrsig.yaml`, `.yml`, `.json`. Stop at the first one found; stop without one at a folder that contains `.git` or after the user's home folder.
3. `--no-config` ignores any file (for reproducing what a teammate without it sees).

When a file is used, commands print one line to **stderr** (so machine-readable output on stdout stays pure): `Using clrsig.yaml (spec, environment, baseUrl, conform.readOnly from the file)`. Nothing is printed when no file is found.

### 4.3 Precedence

`flag` > `CLRSIG_*` environment variable (where one exists today, for example `CLRSIG_SEED`, `CLRSIG_CLIENT_CERT`) > `clrsig.yaml` > built-in default. A flag counts as given when `cmd.Flags().Changed(name)` is true, so `--read-only=false` overrides `readOnly: true`.

Positional arguments win over `run.scenarios`; `conform <spec>` wins over `spec`. A command that needs a value and has neither (a flag nor the file) fails with what / why / fix: *"conform needs a spec: pass one, or set `spec:` in clrsig.yaml."*

### 4.4 The core package

`packages/core/projectconfig`:

```go
type Config struct { /* the fields above, plus unexported extension fields */ }
func Discover(startDir string) (path string, err error)      // 4.2, no I/O beyond directory checks
func Load(path string) (*Config, error)                       // parse, strict, validate, schemaVersion
func (c *Config) Dir() string                                 // base for relative paths
func (c *Config) Resolve(rel string) string
```

It depends only on `serialization` helpers and the packages that own the value rules; the domain and engine packages do not depend on it. The CLI maps `Config` onto flags; the desktop will map it onto its page defaults (phase 4).

### 4.5 `clrsig init`

```
clrsig init [folder] [--spec <file-or-url>] [--base-url <url>] [--dry-run]
clrsig init github-actions [--write [--output <file>]]
```

`clrsig init` (default folder: the current one) creates only what is missing:

| Created | Content |
|---|---|
| `clrsig.yaml` | `schemaVersion`, `spec` and `baseUrl` from the flags (omitted when not given), `environment: environments/local.yaml`, `conform.readOnly: true`, a short comment per section. |
| `environments/local.yaml` | `schemaVersion`, `baseUrl` and one example variable, as the desktop's new workspace does. |
| `environments/ci.yaml` | The same shape with an `${API_TOKEN}` reference, so the CI path never needs a secret in the repository. |
| `scenarios/`, `requests/`, `specs/` | Empty folders with a `.gitkeep`. |

Then it prints what it created, what already existed and was left alone, and the next commands: `clrsig conform` when a spec was given, `clrsig generate <spec>` otherwise.

Safety rules:
- **Never overwrites, never deletes.** An existing file is listed as `kept`. There is no `--force`.
- Writes only inside the folder given (links resolved, as the desktop already does); every file is written atomically.
- `--spec` that is a local file is checked to parse (so a typo is found now); a URL is **not fetched**.
- `--dry-run` prints the plan and writes nothing.
- It does not read or write `.gitignore`; it prints a one-line suggestion (`reports/`).

`clrsig init github-actions` prints a workflow file for stdout by default: checkout, `uses: fuvito/api-workbench@<current version>` with `conform` and the values from the config (spec, environment, read-only, report), and a comment where to set `API_BASE_URL`. `--write` writes `.github/workflows/clrsig.yml` (or `--output <file>`) and refuses an existing file. The snippets live in `packages/core/scaffold` as templates with tests, so the CLI Command Builder's future Action and Docker targets (B-015) can share them.

### 4.6 What does not change in the first phases

- **CLI Command Builder:** it keeps producing explicit commands (a pasted command must not depend on a file the reader may not have). It gains a note when a config file exists in the run-from folder, in phase 3.
- **GitHub Action:** no new input. The action runs `clrsig` in the checkout, so discovery works; inputs are flags and win. The CI guide gains a section.
- **Desktop app:** unchanged until phase 4.
- **Bundles, scenarios, requests, environments:** unchanged.

### 4.7 Security

- The file cannot carry secrets by construction (no `vars`, no `${NAME}` expansion) and is meant to be committed.
- It cannot widen what a command may send: `readOnly: false` is what the CLI already does by default; the file cannot disable a safety check that exists today. `--read-only` stays one flag away and `--no-config` ignores the file.
- Running `clrsig` in a cloned repository already runs whatever scenarios it contains; a config file adds a base URL and defaults but no code and no network access of its own.
- The desktop app (phase 4) only reads a config file inside the open workspace and resolves its paths through the same workspace-bounds check as everything else (B-064).

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| **1. Config** | `packages/core/projectconfig`; `applyProjectConfig` for `conform`, `run`, `validate`; `--config`, `--no-config`; stderr note; `format-spec.md` section | In a folder with `clrsig.yaml`, `clrsig conform` with no arguments runs; the same command with `--base-url` uses the flag; an unknown key, a bad `failOn` and a `${NAME}` value are rejected with what / why / fix; from a subfolder the relative paths still resolve |
| **2. `init`** | `clrsig init`, `clrsig init github-actions`, `packages/core/scaffold` | In an empty folder, `clrsig init --spec openapi.yaml --base-url http://localhost:8080` then `clrsig conform` works; a second `init` creates nothing; an existing `clrsig.yaml` is never changed; `init github-actions` output is valid YAML and passes the Action file test's checks |
| **3. Visibility** | `clrsig config show [command]` (effective value and its source for each setting); the builder's note | `config show` agrees with what the command does, in a test that runs both |
| **4. Desktop (may slip)** | The Check Conformance page and the Workspace page read the config for defaults; a Workspace page card suggests `init` when none exists | Opening a workspace with `clrsig.yaml` preselects its spec, environment and base URL; nothing outside the workspace is read |

Each phase ships green on its own. Phases 1 and 2 are the 0.9.1 scope; 3 and 4 follow if they fit.

## 6. Testing

- **Core:** table tests for parsing (YAML and JSON identical, unknown keys, `x-` kept, bad values, `${NAME}`, two files in one folder, newer `schemaVersion` refused, missing `schemaVersion`), discovery (parent folders, the `.git` stop, `--config`), path resolution.
- **CLI:** each flag overridden by the command line, each key respected, positional arguments over `run.scenarios`, the stderr note (and its absence under `--format json` on stdout), exit code 2 for configuration errors. Real local servers, as in the existing conform tests.
- **`init`:** idempotence, no overwrite, `--dry-run` writes nothing, path escape attempts rejected, atomic write leaves no `.tmp`, output files parse with the real parsers and pass `clrsig validate` / the config loader. The workflow template is parsed as YAML and its inputs are checked against `action.yml` (the existing Action test pattern), so it cannot reference a missing input.
- **Parity:** `TestCatalogMatchesCobra` for the new commands and flags if they are added to the builder catalog (they are not in phase 1).
- **Golden files:** the files `init` writes are snapshots, so a template change is a visible diff.

## 7. Documentation impact

- `help/format-spec.md`: "Project Config File" with every key, the rules in 4.1 and the precedence in 4.3.
- `help/user-guide.md`: a "Project file and `clrsig init`" section; CLI usage mentions `--config` and `--no-config`.
- `docs/guides/ci-integration.md`: the config in CI, and the generated workflow.
- `docs/process/schema-migration-policy.md`: list the project config among the hand-written formats.
- `README.md`: one line in the CLI quick start (product pitch only; developer detail stays in the guide).
- `CLAUDE.md` changelog; backlog B-021 and B-089 ticked when shipped; the "first hour" idea 12, 14 and 43 of the improvement report.
- Not changed: the AI prompt (it generates scenarios and bundles, not the project file).

## 8. Risks

| Risk | Mitigation |
|---|---|
| Hidden defaults make a local run differ from CI | The stderr note names the file and the keys it supplied; `--no-config`; phase 3 `config show`; the file is meant to be committed. |
| A walk-up finds an unrelated `clrsig.yaml` | It stops at the first file, at a `.git` folder and at the home folder; the note names the path used. |
| The file grows into a second scenario format | The key list is closed and mirrors flags; a new key needs a proposal. No lists of requests, no scripting, no variables. |
| A config key and a flag drift apart | One resolution table in the CLI and a test that every key maps to a real flag (in the style of `TestCatalogMatchesCobra`). |
| `init` clobbers work | It never overwrites; existing files are reported as kept; `--dry-run`; tests cover re-running. |
| Breaking the format later | `schemaVersion` from the start and the migration policy: hand-written, so a breaking change ships with `clrsig migrate`. |

## 9. Open decisions (owner), with recommendations

1. **File name.** `clrsig.yaml` (with `.yml` and `.json` equivalents), or `.clrsig.yaml` (hidden). **Recommend `clrsig.yaml`:** it is meant to be seen and reviewed.
2. **Discovery.** Walk up to the first file (stopping at `.git`), or only the current directory. **Recommend walk up**, because commands are often run from a subfolder; `--no-config` is the escape.
3. **`vars` in the file.** **Recommend no**: it invites secrets and duplicates the environment files.
4. **Does `init` write a workflow?** **Recommend only `init github-actions --write`**, never by default; printing stays the default.
5. **Starter scenario in `init`.** **Recommend none**: it would be invented content. The next-step message points to `conform` and `generate`.
6. **`config show` in this release.** **Recommend yes, phase 3**, as small as possible, because precedence questions are the first support question a config file creates.
7. **Desktop reads the file (phase 4).** **Recommend yes, read-only**, but allowed to slip past 0.9.1 so the CLI work is not held back.
8. **`environment` as a path or a name.** A path matches `--env` and works today. A name (`ci` for `environments/ci.yaml`) is friendlier. **Recommend the path now**, and a name lookup as a later additive change.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-24 | Proposal written; open decisions 1 to 8 above | Claude, awaiting owner |
| 2026-11-24 | Open decisions 1 to 8 accepted as recommended: `clrsig.yaml` (with `.yml` and `.json`); walk up to the first file, stopping at `.git`; no `vars`; `init` writes a workflow only with `init github-actions --write`; no starter scenario; `config show` in phase 3; desktop reads the file read-only in phase 4 (may slip); `environment` is a path | owner |
| 2026-11-24 | Phase 1 built. Clarified while building: `spec` applies to `conform` only (not `run --spec`, which would override every scenario's own `specFile`); `impact` does not read the file yet; a key must be spelled exactly (the JSON decoder alone would accept another letter case) | Claude |
| 2026-11-24 | Phase 2 built. Details settled while building: `init` keeps an existing `clrsig.yml`/`.json` as the project file; an existing folder gets no `.gitkeep`; `--spec` that is a local file must parse (checked through the same parser conform uses); the workflow pins the running release tag (or `main` with a note for a dev build) and is read-only unless `clrsig.yaml` says `readOnly: false` | Claude |
| 2026-11-24 | Phase 3 built: `clrsig config show [conform\|run\|validate]` and the builder note. `config show` applies the file to a fresh copy of the command through the same code the command uses, so it cannot disagree with it; a test keeps the key table equal to the file format's key list and checks every named flag exists | Claude |
| 2026-11-24 | Phase 4 built, read-only: the Workspace page card and attention items, and Check Conformance preselecting from `clrsig.yaml` in the workspace folder. Settled while building: only the workspace folder is searched (no walking up); `conform.readOnly: false` is deliberately not applied (the page stays read-only until Allow writes is ticked); the page offers the `clrsig init` command to copy rather than writing files itself | Claude |
