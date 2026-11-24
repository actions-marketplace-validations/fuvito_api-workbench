# Clear Signal 0.9.1 — release notes (draft, not tagged)

**Nothing in this release changes a result or an exit code unless you add a `clrsig.yaml` or use a new flag.** (One wording change: running `clrsig conform`, `run` or `validate` with no argument now says what is missing and where to set it, instead of an argument-count message.)

## The first hour: a project file, `init` and fewer flags

- **Project file.** A `clrsig.yaml` at the project root holds the defaults of the flags that repeat: `spec`, `environment`, `baseUrl`, the `conform` options, `run.scenarios` and `reports`. `clrsig conform`, `run` and `validate` then work with no arguments. A flag or argument always wins; paths are relative to the file, so a command works from a subfolder and in CI; the file never holds secrets (`${NAME}` is rejected; use an environment file); and a command says on stderr when it used it. `--config <file>` chooses another file, `--no-config` ignores it, and `clrsig config show` lists every setting a command takes from it and whether each value comes from the file or is the default.
- **`clrsig init`** creates `clrsig.yaml` (read-only by default), `environments/local.yaml`, `environments/ci.yaml` (token and base URL as `${NAME}` references) and empty `scenarios/`, `requests/` and `specs/` folders. It never overwrites or deletes: run it again and nothing changes. `--dry-run` writes nothing, and a URL given as `--spec` is not fetched. **`clrsig init github-actions`** prints a ready workflow; `--write` saves it to `.github/workflows/clrsig.yml` and refuses to replace a file that exists.
- **`clrsig conform --setup <scenario>`** runs a login (or any scenario) first. What its steps extract, such as a token, is available to the generated requests as `{{name}}`, so secured operations are sent instead of being reported as not sent. Extracted values are never printed and credential-like names are masked in every report. If the setup fails, the check does not start (exit 2).
- **`clrsig validate --strict`** reports valid but weak scenarios as warnings that fail the check: a step that checks nothing, an extraction no later step uses and, with `--env`, a `{{name}}` nothing defines.

## Desktop

- The **Workspace page** has a Project file card (what the file gives, or the `clrsig init` command when there is none) and flags a project file that cannot be used.
- **Check Conformance** starts from the project file's spec, environment, base URL and options, and has **Run a scenario first** in Options. It stays read-only until you tick Allow writes, whatever the file says.

## GitHub Action

New inputs `setup` and `strict`. An empty `spec` or `scenario` input is no longer passed as an empty argument, so a `clrsig.yaml` in the checkout can supply it (an input still wins).

## Before tagging

- [ ] CI green on the exact commit (`docs/process/release-process.md`).
- [ ] Set the date of the 0.9.1 entry in `AboutDialog.tsx` (now `unreleased`).
- [ ] Not verified yet: the Docker image run, the Homebrew install and the Action at the new tag.
