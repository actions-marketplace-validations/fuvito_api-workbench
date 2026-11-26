# `clrsig run`

[← Guide index](README.md)

## Purpose

`run` runs **scenarios**: files you write, with a sequence of requests, the checks you want after each one (`assertions`), and values taken from one response and used in the next (`extractions`). Use it to test a workflow, such as log in, create, read, delete.

Use something else when:

- you only want every operation of the spec checked, with no files to write: [`conform`](conform.md);
- you want to send one request and look at the answer: [`request`](request.md);
- you want to find mistakes in scenario files without sending anything: [`validate`](validate.md).

How to write a scenario is in the [format spec](../../../apps/desktop/wails/help/format-spec.md).

> **Before you point it at a real system:** a scenario sends exactly what it says, including `DELETE`. There is no read-only switch. Reports mask values that are marked or named as secrets and quote little of a response, but not nothing ([What We Check](../../../apps/desktop/wails/help/what-we-check.md), CS-RUN-011 and CS-GAP-017).

## Synopsis

```text
clrsig run <scenario-file-or-folder>... [flags]
```

Several files and folders may be given. A folder runs every scenario in it and its sub-folders; other files in it (environments, requests, specs) are skipped and listed. **All files are read before anything is sent:** if one cannot be read, nothing runs. With a project file that lists `run.scenarios`, the argument can be left out.

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--env`, `-e` | Environment file with the values the scenarios use. | |
| `--var` | `name=value`: set or override one value for this run. Repeatable. | Stays in your shell history. |
| `--spec`, `-s` | OpenAPI file or URL: replaces the scenario's `specFile`. | With a spec, every response is also compared with it. |
| `--coverage` | Lists which operations of the spec the run reached. | Needs `--spec` or `specFile`. |
| `--workspace`, `-w` | Adds this run's coverage to that workspace's stored coverage. | |
| `--fail-fast` | Stops after the first **scenario** that fails. Inside a scenario, each step's `failPolicy` decides. | |
| `--no-recursive` | For folders: only the top level. | |
| `--timeout` | One limit for every step, in milliseconds; replaces the steps' own. | `0`: each step's own. |
| `--seed` | Seed for random and dynamic values (`{{$randomInt}}`, `{{$guid}}`). | Random; printed in the report. `CLRSIG_SEED` works too. |
| `--show-extracted` | Shows extracted values whose names look secret, and extracted values as they are. | Off. Secret environment values stay hidden. |
| `--format`, `-f` | What is printed: `text`, `json`, `junit`, `html`, `markdown`, or `github`. | `text` |
| `--output`, `-o` | Writes that output to a file instead; folders are created. | |
| `--report` | `format=file`: **also** write another report from the same run. Repeatable. | Not `github`. |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. A workflow: log in, create, read, delete

<!-- run: exit=0 contains="4 passed" -->
```bash
clrsig run examples/scenarios/user-lifecycle.yaml --env examples/environments/local.yaml
```

**What you will see** (exit code 0):

<!-- output -->
```text
Scenario: User lifecycle
  …
  PASS  Login  200  1ms
         ✓ status eq 200
  …
         → token = "***"
  PASS  Create user  201  0ms
  …
PASSED  4 passed, 0 failed, 0 skipped  (9/9 assertions passed)  2ms
```

`→ token = "***"` is an extraction: later steps use the real token, the report does not show it.

### 2. What failures look like

**What it does:** runs a scenario written to fail: its schemas ask for things the example server does not send.

<!-- run: exit=1 contains="CS-RESP-005" contains="0 passed, 3 failed" -->
```bash
clrsig run examples/scenarios/contract-violation.yaml --env examples/environments/local.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
         ✓ status eq 200
         ✗ schema: Error at "/version": property "version" is missing
  …
FAILED  0 passed, 3 failed, 0 skipped  (3/6 assertions passed)  2ms
```

The line starting `→ [CS-RESP-005]` under it says why it fails and how to fix it.

### 3. A step fails, and the rest are skipped

<!-- run: exit=1 contains="3 skipped" -->
```bash
clrsig run examples/scenarios/user-lifecycle.yaml --env examples/environments/local.yaml --var password=wrong
```

**What you will see** (exit code 1):

<!-- output -->
```text
  FAIL  Login  401  1ms
  …
  SKIP  Create user
         skipped: step 1 (Login) failed and its failPolicy is stop; set failPolicy: continue to run the steps after a failure
  …
FAILED  0 passed, 1 failed, 3 skipped  (0/2 assertions passed)  1ms
```

### 4. Values are missing

**What it does:** the same scenario with no environment file.

<!-- run: exit=1 contains="not sent: no value for" -->
```bash
clrsig run examples/scenarios/user-lifecycle.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
  FAIL  Login
         error: not sent: no value for {{baseUrl}}, {{username}}, {{password}} (set it in the environment, with --var, in the scenario's variables, or extract it in an earlier step)
```

A step with a value nothing provides is **not sent**: sending `{{baseUrl}}` as text would test nothing.

### 5. A whole folder, stopping at the first failing scenario

<!-- run: exit=1 contains="stopped after the first failure" -->
```bash
clrsig run examples/scenarios --env examples/environments/local.yaml --fail-fast
```

You get one combined report, ending in a line that counts scenarios, and one exit code.

### 6. Keep reports, and see how much of the spec was reached

<!-- run: exit=0 contains="Spec coverage: 4 / 8 operations (50%)" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml \
  --spec examples/server/openapi.yaml --coverage --report junit=reports/run.xml
```

**What you will see** (exit code 0):

<!-- output -->
```text
PASSED  4 passed, 0 failed, 0 skipped  (8/8 assertions passed)  3ms
Spec coverage: 4 / 8 operations (50%)
…
Not covered:
  GET     /protected
```

## Reading the result

| Line | Meaning |
|---|---|
| `PASS` step | Every assertion held (and, with a spec, the response fitted it). |
| `FAIL` step | Something did not hold, or there was no answer. The lines under it say what. |
| `✓` / `✗` | One assertion. |
| `→ name = …` | A value extracted for later steps. |
| `error:` | The step could not be run or judged: a missing value, no connection, an extraction that found nothing. |
| `SKIP` | Not run, because an earlier step's `failPolicy` stopped the scenario. |
| `N of M responses were not compared with the spec` | With a spec: these steps were not checked against it, and the line says why. |

| Exit code | Meaning |
|---|---|
| 0 | Every scenario passed. |
| 1 | At least one scenario failed. |
| 2 | A file could not be read or parsed, nothing was found, or the command was wrong. |

## What to do next

| Result | Next step |
|---|---|
| `not sent: no value for {{x}}` | Give `x` in the environment or with `--var`, or extract it in an earlier step. |
| A `status` assertion fails with `401` or `403` | The credential is wrong or missing. Check the login step and the environment. |
| `extraction "x": … unknown key` | The response did not hold what you extract; usually the request before it failed. Read its status. |
| `✗ schema: …` | The response does not fit the schema. Fix the server or the schema; `CS-…` IDs are explained in [What We Check](../../../apps/desktop/wails/help/what-we-check.md). |
| `executing request: …` (connection refused, timeout) | Is the server running? Is `baseUrl` right? See `--timeout`. |
| Everything passes but you are not sure it checks enough | [`clrsig validate --strict`](validate.md) finds steps that check nothing. |

## Common mistakes and limits

- **A `schema` assertion alone does not check the status.** It passes, without reading the body, for a status it does not know: with an `operationId`, one the operation does not declare; with a schema written in place, anything but `200`. Without a spec for the scenario, a `500` can then be reported as `PASSED`. **Add a `status` assertion to such steps** (CS-GAP-020).
- A response-time or schema check is only as good as what you assert: a step with no assertions passes whenever the request is answered.
- `--fail-fast` stops between scenarios, not between steps.

## Related commands and the desktop app

[`validate`](validate.md), [`request`](request.md), [`conform`](conform.md). In the desktop app, open a scenario and use **Run**; each step shows the request as sent and the response.
