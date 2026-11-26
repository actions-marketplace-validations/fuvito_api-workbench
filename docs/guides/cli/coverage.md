# `clrsig coverage`

[← Guide index](README.md)

## Purpose

**Coverage** is which operations of your spec your tests have actually reached, collected over many runs in a **workspace** folder. It answers "what have we **not** tested?". Nothing is sent by these commands.

Runs add to it when you give them a workspace: `clrsig run … --spec <spec> --workspace <folder>`. (`conform --coverage` records a separate history, shown by `conform` itself.)

> **A gate needs something to look at.** `coverage quality --fail-on-gaps` also fails when **nothing was recorded** for the spec, and every `coverage` command stops with an error when the workspace folder does not exist. Releases up to 0.13.5 passed in both cases.

| Command | What it does |
|---|---|
| `coverage list` | One summary per spec in the workspace. |
| `coverage quality` | The gaps for one spec. |
| `coverage reset` | Deletes what was collected for one spec. |

## `clrsig coverage list`

```text
clrsig coverage list --workspace <folder> [flags]
```

## Options: `clrsig coverage list`

| Option | What it does | Default and limits |
|---|---|---|
| `--workspace`, `-w` | The workspace folder. | Required. |
| `--format`, `-f` | `text`, `json`, `csv`, `markdown` or `html`. | `text` |
| `--output`, `-o` | Writes the report to a file. | |

<!-- run: exit=0 first="0" contains="of 8 operations" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage list --workspace my-ws
```

**What you will see** (both exit 0):

<!-- output -->
```text
  Covered 4 (50%) · Not covered 4 (50%) · Passing 4 (50%) · Failing 0 (0%) · of 8 operations
```

`Covered` means a run reached the operation. `Passing` and `Failing` say how those went.

## `clrsig coverage quality`

```text
clrsig coverage quality <spec-file-or-url> --workspace <folder> [flags]
```

## Options: `clrsig coverage quality`

| Option | What it does | Default and limits |
|---|---|---|
| `--workspace`, `-w` | The workspace folder. | Required. |
| `--fail-on-gaps` | Exit 1 when there is a gap, or nothing was recorded. | Off: gaps are only reported. |
| `--format`, `-f` | `text`, `json`, `csv`, `markdown` or `html`. | `text`. With no data, `text` prints a sentence; the others give a report with nothing covered. |
| `--output`, `-o` | Writes the report to a file. | |

<!-- run: exit=1 first="0" contains="NEVER EXERCISED" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage quality examples/server/openapi.yaml --workspace my-ws --fail-on-gaps
```

| Signal | Meaning |
|---|---|
| `NEVER EXERCISED` | No recorded run reached the operation. |
| `HAPPY PATH ONLY` | Only successful answers were ever seen. |
| `SECURED NEVER AUTHED` | It needs credentials, and was never tested with them. |
| `UNTESTED ERROR PATHS` | Declared error answers that never happened. |

Exit code **0**: the report was made (with `--fail-on-gaps`: data and no gap); **1**: with `--fail-on-gaps`, a gap or no data; **2**: the spec or the command was wrong, or the workspace is not a folder. This is the same for every format.

## `clrsig coverage reset`

```text
clrsig coverage reset <spec-file-or-url> --workspace <folder>
```

## Options: `clrsig coverage reset`

| Option | What it does | Default and limits |
|---|---|---|
| `--workspace`, `-w` | The workspace folder. | Required. No undo. |

<!-- run: exit=0 first="0" contains="Coverage reset for" contains="No coverage data found" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage reset examples/server/openapi.yaml --workspace my-ws
clrsig coverage list --workspace my-ws
```

When nothing is stored for that spec, it says `nothing was reset` and exits 0. Check the spelling of the spec: it must be written as it was in the runs.

## What to do next

| Result | Next step |
|---|---|
| `No coverage data found` | Check the spec is written as in the runs, and that they used both `--workspace` and `--spec`. |
| `NEVER EXERCISED` | Add scenarios, or run [`conform`](conform.md) to reach them. |
| `HAPPY PATH ONLY`, `UNTESTED ERROR PATHS` | Add scenarios that cause the declared errors. |
| Old results mislead after a big change | `coverage reset`, then run again. |

## Common mistakes and limits

- Coverage says an operation was **reached**, not that it was tested well.
- Specs whose names differ only in punctuation or case can share one stored entry (backlog B-080).

## Related commands and the desktop app

[`run`](run.md). The desktop app shows the same data on its Contracts page.
