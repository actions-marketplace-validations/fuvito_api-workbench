# `clrsig init`

[← Guide index](README.md)

## Purpose

`init` creates what a project needs so that `clrsig conform`, `run` and `validate` work with few or no flags: a project file (`clrsig.yaml`), a local and a CI environment file, and empty `scenarios/`, `requests/` and `specs/` folders. `init github-actions` prints a workflow that runs the check on pull requests.

It **only creates what is missing**. It never overwrites or deletes, so it is safe to run again. It sends nothing and fetches nothing from the network.

You do not need it: every command works with flags alone.

## Synopsis

```text
clrsig init [folder] [flags]
clrsig init github-actions [flags]
```

Without a folder, `init` works in the current one.

## Options: `clrsig init`

| Option | What it does | Default and limits |
|---|---|---|
| `--spec` | The OpenAPI file or URL to write into `clrsig.yaml`. A file is read: it must parse, and the values its requests need are listed, commented out, in the environment files. | A URL is written but **not** fetched, so no values are listed. |
| `--base-url` | The server to check, written into `clrsig.yaml` and `environments/local.yaml`. | `http://localhost:8080` in the environment file. |
| `--dry-run` | Prints what would be created and writes nothing. | |

## Options: `clrsig init github-actions`

| Option | What it does | Default and limits |
|---|---|---|
| `--write` | Saves the workflow to `.github/workflows/clrsig.yml`. Without it the workflow is only printed. | Refuses to replace a file that exists. Run it from the repository root. |
| `--output` | With `--write`: the file to create instead. | |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. See what it would create

<!-- run: exit=0 contains="nothing was written" contains="would be listed" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --dry-run
```

**What you will see** (exit code 0):

<!-- output -->
```text
Dry run: nothing was written. In my-check, clrsig init would:
Would create:
  clrsig.yaml
  environments/local.yaml
  environments/ci.yaml
  …
The spec needs: username, password, token, name, email, id.
```

### 2. Create a project, then run it again

**What it does:** the second run changes nothing and says so.

<!-- run: exit=0 first="0" contains="Nothing to create" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig init my-check
```

**What you will see** (both exit 0):

<!-- output -->
```text
Created in my-check:
  clrsig.yaml
  environments/local.yaml
  environments/ci.yaml
  scenarios
  requests
  specs
…
Kept (already there, not changed):
  clrsig.yaml
  …
Nothing to create: this folder is already set up.
```

### 3. What the files hold

`clrsig.yaml` after example 2:

```yaml
schemaVersion: "1"

spec: ../examples/server/openapi.yaml     # relative to the folder of clrsig.yaml
environment: environments/local.yaml
baseUrl: http://localhost:8080

conform:
  readOnly: true        # send only GET and HEAD; set to false to also send writes
```

- `environments/local.yaml` is for your machine. The values the spec needs are there as comments: remove the `#` and fill in the ones you want.
- `environments/ci.yaml` reads `${API_BASE_URL}` and `${API_TOKEN}` from the process environment, so no secret is written to a file.

### 4. A workflow for GitHub Actions

<!-- run: exit=0 contains="command: conform" contains="upload-artifact" -->
```bash
clrsig init github-actions
```

It prints the workflow. Read the comments in it: you have to set the spec's path, the repository variable `API_BASE_URL` and the secret `API_TOKEN`. It uploads the JUnit and HTML reports as a build artifact, which anyone who can read the repository's runs can download.

<!-- not run: writes into .github/ of the repository you are in -->
```bash
clrsig init github-actions --write
```

## Reading the result

| Line | Meaning |
|---|---|
| `Created …` | These files and folders are new. |
| `Kept (already there, not changed)` | They existed and were left as they are, whatever they contain. |
| `Nothing to create` | The folder is already set up. |

Exit code **0** when it did what it said; **2** when it could not (for example a `--spec` file that does not parse, or `--write` on a workflow that exists).

## What to do next

| Result | Next step |
|---|---|
| Created, with "The spec needs: …" | Fill those values in `environments/local.yaml`, then [`clrsig conform --list`](conform.md#1-see-what-a-run-will-need-before-sending-anything). |
| Kept, but you wanted new content | `init` does not update files. Edit them, or delete the one you want recreated and run `init` again. |
| You are not sure what a command will use now | [`clrsig config show`](config-show.md) |
| A released `clrsig` printed the workflow | The workflow pins that release. A development build writes `@main`: pin a release tag yourself. |

## Common mistakes and limits

- **Expecting writes to be tested.** A new project has `readOnly: true`. Set it to `false` only against data you can lose.
- **Putting a secret into `clrsig.yaml` or an environment file you commit.** Write `${NAME}` instead.
- With a spec **URL**, the environment files do not list the needed values: use `clrsig conform --list` to see them.

## Related commands and the desktop app

[`config show`](config-show.md), [`conform`](conform.md). The desktop app opens a folder as a workspace and reads the same `clrsig.yaml`.
