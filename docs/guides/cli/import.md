# `clrsig import`

[← Guide index](README.md)

## Purpose

`import` turns a **Postman** collection (v2.1) or an **Insomnia** export (v4) into request files, one per request. It is a start for moving your work over. It sends nothing.

> **⚠ Test scripts are not converted, and nothing says so.** Postman `pm.test(...)` and other scripts are left out without a warning (backlog B-143). The imported files **check nothing** until you write assertions in [scenarios](run.md).

## Synopsis

```text
clrsig import <postman|insomnia> <file> [flags]
```

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--output`, `-o` | The folder for the request files. | `requests` |
| `--dry-run` | Lists the files that would be written. | |
| `--format`, `-f` | `text` or `json`: how the list is printed. | `text` |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. See what would be written

<!-- run: exit=0 contains="Would write 2 request files" -->
```bash
clrsig import postman docs/guides/cli/files/shop-postman.json --dry-run
```

### 2. Import

<!-- run: exit=0 contains="Wrote 2 request files" -->
```bash
clrsig import postman docs/guides/cli/files/shop-postman.json --output imported
```

The layout is `<output>/<collection>/<folders>/<request>.yaml`. In this collection, a bearer token setting is kept as `auth` with `{{token}}`; headers and raw bodies are kept. The **Create user** request had a test script: it is not in the file.

## Reading the result

| Exit code | Meaning |
|---|---|
| 0 | The requests were converted (or listed). |
| 1 | Some could not be written. |
| 2 | The file could not be read, or is not a supported export (also: the wrong kind named, such as a Postman file given as `insomnia`). |

Nothing existing is overwritten: a taken name gets `-2`, `-3` and so on. **Running the same import twice gives you two copies of every file.**

## What to do next

| Result | Next step |
|---|---|
| Imported | Put the `{{…}}` values in an environment file and try one with [`request`](request.md). |
| You had tests in Postman | Open the collection, and write each check as an assertion in a scenario. |
| A warning about a value that could not be converted | Look at that request and fix it by hand. |

## Common mistakes and limits

- Scripts (pre-request and test) are not run or converted (B-143).
- Secrets written **literally** in a collection are copied into the files. Replace them with `{{name}}` before you commit.

## Related commands and the desktop app

[`generate`](generate.md) (from a spec), [`request`](request.md). The desktop app has the same import in its **Import** menu.
