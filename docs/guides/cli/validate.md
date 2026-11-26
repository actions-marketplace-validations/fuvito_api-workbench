# `clrsig validate`

[← Guide index](README.md)

## Purpose

`validate` reads scenario files and reports mistakes **without sending anything**. With `--strict` it also reports scenarios that are valid but weak. With `--spec` it checks the **OpenAPI document itself**, which is worth doing before you trust any result read against it.

It does not talk to your API. To check the API, use [`conform`](conform.md) or [`run`](run.md).

## Synopsis

```text
clrsig validate <scenario-file-or-folder> [more...] [flags]
clrsig validate --spec <openapi-file>
```

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--strict` | Also warns about a step that checks nothing and extracts nothing, an extraction no later step uses, and (with `--env`) a `{{name}}` that nothing defines. **Warnings then fail the check.** | |
| `--env`, `-e` | With `--strict`: the environment the scenarios will run against. | |
| `--spec` | Also checks this OpenAPI file: a missing `operationId`, a `required` property that is not declared, an example that breaks its format. Errors fail; warnings are notes and fail only with `--strict`. | The scenario files are then optional. |
| `--no-recursive` | For folders: only the top level. | |
| `--format`, `-f` | `text` or `json`. | `text` |
| `--output`, `-o` | Writes the result to a file. | |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. Check a scenario

<!-- run: exit=0 contains="OK" -->
```bash
clrsig validate examples/scenarios/user-lifecycle.yaml
```

**What you will see** (exit code 0):

<!-- output -->
```text
examples/scenarios/user-lifecycle.yaml  OK
```

A folder works too, and ends with a count.

<!-- run: exit=0 contains="0 invalid" -->
```bash
clrsig validate examples/scenarios
```

### 2. A file with a mistake

**What it does:** checks [a scenario](files/broken-scenario.yaml) with an assertion type that does not exist.

<!-- run: exit=1 contains="unknown assertion type" -->
```bash
clrsig validate docs/guides/cli/files/broken-scenario.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
docs/guides/cli/files/broken-scenario.yaml
  [step 1] Health: unknown assertion type "statuz"
```

### 3. Valid, but weak

**What it does:** [this scenario](files/weak-scenario.yaml) is valid, yet one step checks nothing.

<!-- run: exit=0 contains="OK" -->
```bash
clrsig validate docs/guides/cli/files/weak-scenario.yaml
```

<!-- run: exit=1 contains="has no assertions" contains="nothing defines" -->
```bash
clrsig validate docs/guides/cli/files/weak-scenario.yaml --strict --env examples/environments/local.yaml
```

**What you will see** with `--strict` (exit code 1):

<!-- output -->
```text
  warning: [step 1] checks nothing: has no assertions and extracts nothing: it passes whenever the request is answered, so a wrong status or body would go unnoticed
  warning: [step 2] unused extraction: uses {{nowhere}}, which nothing defines: not a scenario variable, not extracted by an earlier step and not in the environment
  warning: [step 2] unused extraction: extracts "unused", but no later step uses {{unused}}
```

### 4. Check the spec itself

<!-- run: exit=0 contains="OK" -->
```bash
clrsig validate --spec examples/server/openapi.yaml
```

## Reading the result

| Line | Meaning |
|---|---|
| `OK` | Nothing was found. It does **not** mean the scenario will pass: nothing was sent. |
| `[step N] label: …` | A mistake in that step. |
| `warning: …` | Only with `--strict` (or from `--spec`). |
| `PARSE ERROR: …` | The file could not be read, or is not valid YAML or JSON. |

| Exit code | Meaning |
|---|---|
| 0 | No errors (and, with `--strict`, no warnings). |
| 1 | A file has errors or warnings, **or could not be read** (the command's help says 2 for that; see below). |
| 2 | The command was wrong. |

## What to do next

| Result | Next step |
|---|---|
| An error in a step | Fix it; the keys and values allowed are in the [format spec](../../../apps/desktop/wails/help/format-spec.md). |
| `has no assertions` | Add at least a `status` assertion. |
| `which nothing defines` | Add the value to the environment, or extract it in an earlier step; check the spelling. |
| `extracts "x", but no later step uses` | Use it, or remove the extraction. |
| All `OK` | [`clrsig run`](run.md) it. |

## Common mistakes and limits

- **An HTTP method that does not exist is not reported:** `method: GETT` is `OK` here and fails in `run` (CS-GAP-021).
- A key that is misspelled or unknown is ignored without a word, here and in `run` (backlog B-121).
- A file that cannot be read exits **1**, although `clrsig validate --help` says 2 (backlog B-139). In a script, treat any code but 0 as a failure.

## Related commands and the desktop app

[`run`](run.md). The desktop app's scenario editor validates a file as you save it.
