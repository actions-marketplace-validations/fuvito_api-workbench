# `clrsig bundle`

[← Guide index](README.md)

## Purpose

A **bundle** is one file that holds environments, requests and scenarios. Use it to share a set of tests, to take a copy of a workspace, or to run what an AI assistant wrote for you without saving it first.

| Command | What it does | Sends requests? |
|---|---|---|
| `bundle validate` | Checks a bundle for errors. | No |
| `bundle run` | Runs its scenarios. | **Yes** |
| `bundle import` | Saves its files into a workspace folder. | No |
| `bundle export` | Writes a workspace as one bundle. | No |

> **Treat a bundle you did not write like a script you did not write.** `bundle run` sends whatever it says, to whatever address it names, including `DELETE`. Read it first.

> **After an export, give the secrets.** `export` replaces secret values with the text `CHANGE_ME`. `bundle run` treats a secret that still holds it as **having no value**: it names it in a warning, and a step that uses it is not sent. Give the real value with `--env-file` or `--var`. This protects `bundle run` only: after `bundle import`, the environment files still hold `CHANGE_ME`, and `run`, `conform` and the app **send it** (backlog B-149). Releases up to 0.13.5 sent it from `bundle run` too.

The bundle format is in the [format spec](../../../apps/desktop/wails/help/format-spec.md).

## `clrsig bundle validate`

```text
clrsig bundle validate <bundle-file> [flags]
```

## Options: `clrsig bundle validate`

| Option | What it does | Default and limits |
|---|---|---|
| `--spec`, `-s` | The OpenAPI file or URL you would give to `run` or `import`. | |

<!-- run: exit=0 contains="Bundle is valid" -->
```bash
clrsig bundle validate docs/guides/cli/files/demo-bundle.yaml
```

Exit code **0**: valid (warnings are printed and do not fail); **1**: errors; **2**: the file could not be read or parsed.

## `clrsig bundle run`

```text
clrsig bundle run <bundle-file> [flags]
```

The bundle is validated first; any error stops the run before anything is sent.

## Options: `clrsig bundle run`

| Option | What it does | Default and limits |
|---|---|---|
| `--env`, `-e` | The **name** of one of the bundle's environments. | Needed when it has more than one. |
| `--env-file` | An environment file to use instead of the bundle's own. | |
| `--var` | `name=value` for this run. Repeatable. | Stays in your shell history. |
| `--scenario` | Run only the scenario with this title or file name. Repeatable. | A name that matches nothing is an error. |
| `--spec`, `-s` | OpenAPI file or URL; replaces every scenario's spec. | |
| `--seed` | Seed for random and dynamic values. | Random, printed in the report. |
| `--show-extracted` | Shows extracted values, secret-looking names included. | Off. |
| `--format`, `-f` | `text`, `json`, `junit`, `html` or `markdown`. | `text` |
| `--output`, `-o` | Writes that output to a file instead. | |
| `--report` | `format=file`: also write another report. Repeatable. | |

<!-- run: exit=0 contains="Bundle PASSED" -->
```bash
clrsig bundle run docs/guides/cli/files/demo-bundle.yaml
```

**What you will see** (exit code 0):

<!-- output -->
```text
  PASS  Health  200  1ms
         ✓ status eq 200
…
Bundle PASSED  1/1 scenarios passed  (1 steps, 1/1 assertions passed)  1ms
```

Exit code **0**: every scenario passed; **1**: at least one failed; **2**: the bundle is invalid or unreadable, or `--scenario` matched nothing. Results read as in [`run`](run.md#reading-the-result), with the same warning about `schema` assertions and status.

## `clrsig bundle import`

```text
clrsig bundle import <bundle-file> --workspace <folder> [flags]
```

Writes files under `environments/`, `requests/` and `scenarios/`. If the bundle has any error, nothing is written.

## Options: `clrsig bundle import`

| Option | What it does | Default and limits |
|---|---|---|
| `--workspace`, `-w` | The workspace folder to write into. | Required. |
| `--on-conflict` | When a file exists: `rename`, `skip` or `overwrite`. | `rename` |
| `--dry-run` | Shows what would be written. | |
| `--spec-file`, `-s` | OpenAPI file or URL; replaces the spec of every imported scenario. | |

<!-- run: exit=0 first="0" contains="2 renamed" -->
```bash
clrsig bundle import docs/guides/cli/files/demo-bundle.yaml --workspace my-ws
clrsig bundle import docs/guides/cli/files/demo-bundle.yaml --workspace my-ws
```

The second import does not replace anything: it writes `health-2.yaml` and `local-2.yaml`. **`--on-conflict overwrite` replaces files without keeping a copy:** run `--dry-run` first.

## `clrsig bundle export`

```text
clrsig bundle export --workspace <folder> [flags]
```

The workspace is not changed.

## Options: `clrsig bundle export`

| Option | What it does | Default and limits |
|---|---|---|
| `--workspace`, `-w` | The workspace folder to export. | Required. |
| `--output`, `-o` | The file to write. | Printed to the screen. |
| `--keep-secrets` | Writes secret values as they are, instead of `CHANGE_ME`. | Off. Do not share the result. |

<!-- run: exit=0 first="0" contains="title: my-ws" -->
```bash
clrsig bundle import docs/guides/cli/files/demo-bundle.yaml --workspace my-ws
clrsig bundle export --workspace my-ws
```

Only values **marked** `secret: true` in an environment are replaced. A password written directly in a request, a header or a scenario is exported as it is: read the file before you share it.

## What to do next

| Result | Next step |
|---|---|
| `Bundle is valid` | `bundle run` it, or `bundle import --dry-run`. |
| Errors | Each names the file and field; fix and validate again. |
| `no scenario matches` | Use a scenario's title or file name, as in the bundle. |
| `still holds the CHANGE_ME` | Give the value: `--var name=value`, or `--env-file`. |

## Related commands and the desktop app

[`run`](run.md), [`validate`](validate.md). The desktop app imports and exports bundles.
