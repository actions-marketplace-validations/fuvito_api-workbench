# `clrsig request`

[← Guide index](README.md)

## Purpose

`request` sends **one** saved request file and prints the answer. Use it to look at what your API really returns while you write an environment or a scenario.

It **checks nothing**: it does not compare the answer with a spec, and a `401` or `500` is not a failure to it. For checks use [`run`](run.md) or [`conform`](conform.md).

> **It prints the response as it is,** with tokens, passwords and personal data in it. Nothing is masked: that is its purpose. Think before you paste the output anywhere, or use it in a CI log.

## Synopsis

```text
clrsig request <request-file> [flags]
```

Request files are described in the [format spec](../../../apps/desktop/wails/help/format-spec.md). [`generate`](generate.md) writes them from a spec.

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--env`, `-e` | Environment file that fills the `{{name}}` places in the request. | |
| `--format`, `-f` | `text` or `json`. | `text` |
| `--timeout` | How long to wait, in milliseconds; replaces the file's own. | `0`: the file's value. |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. Send a request

<!-- run: exit=0 contains="200 OK" -->
```bash
clrsig request examples/requests/health-check.yaml --env examples/environments/local.yaml
```

**What you will see** (exit code 0):

<!-- output -->
```text
GET http://localhost:8080/health
…
Content-Type: application/json
…
  "status": "ok",
```

You see the method and address sent, the status with the time and size, the response headers, and the body (JSON is indented).

### 2. An error answer still exits 0

**What it does:** asks for the user list with no token.

<!-- run: exit=0 contains="401 Unauthorized" -->
```bash
clrsig request examples/requests/list-users.yaml --env examples/environments/local.yaml
```

### 3. For a script

<!-- run: exit=0 contains="statusCode" -->
```bash
clrsig request examples/requests/health-check.yaml --env examples/environments/local.yaml --format json
```

The JSON has `method`, `url`, `statusCode`, `status`, `durationMs`, `sizeBytes`, `headers`, `body` (as text) and `error`. To fail a script on a status, test `statusCode` yourself, or write a [scenario](run.md) with a `status` assertion.

### 4. A value is missing

**What it does:** leaves out `--env`, so nothing fills `{{baseUrl}}`.

<!-- run: exit=1 contains="not sent: no value for {{baseUrl}}" -->
```bash
clrsig request examples/requests/health-check.yaml
```

### 5. No server

<!-- run: exit=1 contains="ERROR: executing request" -->
```bash
clrsig request docs/guides/cli/files/unreachable-request.yaml
```

## Reading the result

| Exit code | Meaning |
|---|---|
| 0 | The server answered, **with any status**. |
| 1 | Not sent (a `{{name}}` has no value), or no answer: it could not connect, timed out, or the address is not usable. |
| 2 | The file could not be read, or is not a request file. |

## What to do next

| Result | Next step |
|---|---|
| `not sent: no value for {{…}}` | Give `--env`, with every name the message lists. |
| `401` or `403` | Check the token in the environment and the request's `auth`. |
| `context deadline exceeded` | The server is slow or stuck; try a larger `--timeout`. |
| The answer is what you expect | Turn it into a [scenario](run.md) step with assertions. |

## Common mistakes and limits

- **A request with a `{{name}}` that nothing defines is not sent**, so a request that must carry such braces as text cannot be sent this way. Releases up to 0.13.5 sent it as written, and could exit 0.
- Do not use `request` as a health check that fails on errors: it exits 0 on a `500`.

## Related commands and the desktop app

[`generate`](generate.md), [`run`](run.md). The desktop app's request editor does the same, with history.
