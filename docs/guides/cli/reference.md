# Reference: exit codes, reports and shared options

[← Guide index](README.md)

## Exit codes

**0** is the only success. A script should treat every other code as a failure.

| Command | 0 | 1 | 2 |
|---|---|---|---|
| `conform` | No failures (also with only warnings, or only known violations) | A violation, or an operation that could not be checked | The spec or the command was wrong, or the filters matched nothing |
| `conform --list` | The spec was read | | The spec or the command was wrong |
| `run` | Every scenario passed | A scenario failed | A file could not be read or parsed, nothing was found, or the command was wrong |
| `validate` | No errors (with `--strict`, no warnings) | A file has errors or warnings, or does not parse | A file could not be read, or the command was wrong |
| `request` | A response arrived, **with any status** | Not sent, or no response | A file could not be read, or the command was wrong |
| `generate` | The files are as the spec says | | The spec was wrong, or a file is in the way |
| `diff` | No change at or above `--fail-on` | At least one | A spec could not be read, or the command was wrong |
| `impact` | No scenario step is affected | At least one is | A spec or a file could not be read |
| `import` | Converted (or listed) | Some files could not be written | The file could not be read, or is not a supported export |
| `coverage list`, `reset` | Done | | The workspace is not a folder, or the command was wrong |
| `coverage quality` | The report was made | With `--fail-on-gaps`: a gap, or nothing recorded | The spec or the command was wrong, or the workspace is not a folder |
| `bundle validate` | Valid | It has errors | It could not be read or parsed |
| `bundle run` | Every scenario passed | A scenario failed | The bundle is invalid or unreadable |
| `bundle import` | Done | | The bundle is invalid or unreadable |
| `config show`, `init` | Done | | It could not be done |

## Reports

| Command | `--format` values | `--report` (extra files) | `--output` |
|---|---|---|---|
| `conform`, `run` | `text`, `json`, `junit`, `html`, `markdown`, `github` | Yes, every one but `github` | Yes |
| `diff`, `bundle run` | `text`, `json`, `junit`, `html`, `markdown` | Yes | Yes |
| `coverage list`, `quality` | `text`, `json`, `csv`, `markdown`, `html` | No | Yes |
| `validate` | `text`, `json` | No | Yes |
| `impact` | `text`, `json` | No | Yes |
| `import`, `generate` | `text`, `json` | No | There `--output` is the **folder** for the files written |
| `request` | `text`, `json` | No | No |

| Format | Use it for |
|---|---|
| `text` | Reading in a terminal. |
| `json` | Your own scripts. |
| `junit` | A CI system's test view. |
| `html` | One file to send to a person. |
| `markdown` | A pull request comment or a job summary. |
| `github` | Annotations on a GitHub pull request. |
| `csv` | A spreadsheet. |

**Reports can hold sensitive data.** They mask what is recognized as a credential, and they can miss some. `request` prints the response **as it is**. See [what we check](../../../apps/desktop/wails/help/what-we-check.md) (CS-RUN-011).

## Options every command has

| Option | What it does |
|---|---|
| `--config` | The project file to use, instead of the `clrsig.yaml` found in this folder or above it. |
| `--no-config` | Ignores any project file. |
| `--client-cert`, `--client-key` | A certificate and key for mutual TLS (PEM; the key without a password). |
| `--ca-cert` | Extra certificate authorities to trust. |

## Project file, environments and secrets

- `clrsig.yaml` gives defaults for options you would repeat. **An option on the command line wins.** [`config show`](config-show.md) says where each value comes from; [`init`](init.md) creates the file.
- An **environment** file fills the `{{name}}` places. `--var name=value` replaces one for a single run, where the command has `--var`.
- Write a secret as `${NAME}` in the environment file: the value is read from the process environment when the command runs. If `NAME` is not set, a warning says so. **Read that warning:** the text `${NAME}` is then used as the value.

The keys of every file are in the [format spec](../../../apps/desktop/wails/help/format-spec.md).
