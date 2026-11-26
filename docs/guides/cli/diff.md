# `clrsig diff`

[← Guide index](README.md)

## Purpose

`diff` compares two versions of an OpenAPI spec and says which changes **break clients**. Use it on a pull request that changes the spec, before the change ships. It reads two files: no server, no requests.

It says what changed in the **contract**. It does not check that your server follows either version: that is [`conform`](conform.md).

## Synopsis

```text
clrsig diff <base-spec> <revision-spec> [flags]
clrsig diff <spec> --against <git-ref> [flags]
```

The **base** is the old version, the **revision** the new one. The order matters.

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--against` | Compare one spec file with its version at a git ref, such as `origin/main`. | Needs git. Give one file. |
| `--fail-on` | The least serious change that fails the command: `breaking`, `warning` or `info`. | `breaking` |
| `--breaking-only` | Show only breaking changes. | Refused together with `--fail-on info` or `warning`, which it would hide. |
| `--format`, `-f` | `text`, `json`, `junit`, `html` or `markdown`. | `text` |
| `--output`, `-o` | Writes that output to a file instead; folders are created. | |
| `--report` | `format=file`: also write another report from the same run. Repeatable. | |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. Compare two versions

<!-- run: exit=1 contains="2 breaking" -->
```bash
clrsig diff docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
  [ERR] !  the response property `items/id` became optional for the status `200`  GET /users
  [ERR] !  API path removed without deprecation  GET /users/{id}
  [WARN]    deleted the `query` request parameter `limit`  GET /users
  [INFO]    the API version changed from `1` to `2`
  [INFO]    endpoint added  GET /health
5 change(s)  ·  2 breaking
```

### 2. Fail on warnings too

<!-- run: exit=1 contains="at or above" -->
```bash
clrsig diff docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml --fail-on warning
```

### 3. Nothing changed

<!-- run: exit=0 contains="No changes." -->
```bash
clrsig diff docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v1.yaml
```

### 4. In a pull request

<!-- not run: needs a git repository with the base branch fetched -->
```bash
clrsig diff openapi.yaml --against origin/main --report junit=reports/diff.xml
```

In CI, fetch the base branch first (with `actions/checkout`, use `fetch-depth: 0`).

## Reading the result

| Mark | Meaning |
|---|---|
| `[ERR] !` | **Breaking:** a client that worked with the base can stop working. |
| `[WARN]` | May break some clients; read it. |
| `[INFO]` | A change that should not break clients. |
| `No changes.` | No change that matters to the contract (a changed description alone is not one). |

| Exit code | Meaning |
|---|---|
| 0 | No change at or above `--fail-on`. |
| 1 | At least one. |
| 2 | A spec could not be read or parsed, or the command was wrong. |

## What to do next

| Result | Next step |
|---|---|
| A breaking change you did not mean | Undo it, or add instead of changing: a new field, a new path. |
| A breaking change you meant | Version the API, tell your clients, and see which of your scenarios it affects: [`impact`](impact.md). |
| Only warnings | Decide for each. If your team wants them to fail, use `--fail-on warning`. |

## Common mistakes and limits

- Swapping the two files turns removals into additions.
- It judges the contract by rules; a change in **meaning** with the same shape is not seen.
- Swagger 2.0 files are not supported.

## Related commands and the desktop app

[`impact`](impact.md), [`conform --changed-against`](conform.md#options). The desktop app compares specs on its Contracts page.
