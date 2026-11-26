# `clrsig impact`

[← Guide index](README.md)

## Purpose

`impact` answers: **which of my scenarios call something this spec change breaks?** It compares two specs as [`diff`](diff.md) does, then looks through your scenario files for steps that call an affected path. Nothing is sent.

## Synopsis

```text
clrsig impact <base-spec> <revision-spec> <scenario-file-or-folder>... [flags]
```

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--no-recursive` | For folders: only the top level. | |
| `--format`, `-f` | `text` or `json`. | `text` |
| `--output`, `-o` | Writes the result to a file. | |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. Which scenarios are affected?

<!-- run: exit=1 contains="1 scenario(s) affected" -->
```bash
clrsig impact docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml docs/guides/cli/files/shop-scenarios
```

**What you will see** (exit code 1):

<!-- output -->
```text
Breaking changes (2):
  [ERR] GET /users/{id} — API path removed without deprecation
Affected scenarios (1):
    ✗ Fetch user 1                    GET {{baseUrl}}/users/1
      └─ API path removed without deprecation
1 scenario(s) affected — review before shipping the spec revision
```

### 2. A scenario the change does not touch

<!-- run: exit=0 contains="No scenario steps match" -->
```bash
clrsig impact docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml docs/guides/cli/files/shop-scenarios/health.yaml
```

## Reading the result

| Line | Meaning |
|---|---|
| `Breaking changes (N)` | What `diff` calls breaking. |
| `Affected scenarios` | Files with a step that calls a broken path, with the step and the reason. |
| `No scenario steps match the breaking change paths.` | None of the scenarios you gave calls them. |

Exit code **0**: no scenario is affected; **1**: at least one step is; **2**: a spec or a file could not be read.

## What to do next

| Result | Next step |
|---|---|
| A scenario is affected | Update it together with the spec change, or reconsider the change. |
| None affected, but there are breaking changes | **Your clients may still break.** `impact` knows only the scenarios you gave it. |

## Common mistakes and limits

- `0 affected` is not "safe to ship": it speaks only for these files.
- It matches the **method and path** a step calls. A scenario that depends on a changed **field** is listed only if it calls the changed operation.

## Related commands and the desktop app

[`diff`](diff.md), [`run`](run.md).
