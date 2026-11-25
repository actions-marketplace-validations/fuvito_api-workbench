# `clrsig conform`

[← Guide index](README.md)

## Purpose

`conform` is **Spec Conformance Testing** with no test files to write. It reads an OpenAPI 3.0 or 3.1 spec (not Swagger 2.0, which it refuses with `invalid OpenAPI spec`), builds one request for every operation, sends it to your running API, and compares each answer with what the spec declares for it: the status, the Content-Type, the declared headers and the body.

Use something else when:

- you need a **sequence** of requests (log in, create, read, delete) or checks of your own: [scenarios with `clrsig run`](../../../apps/desktop/wails/help/user-guide.md#8-scenarios-and-api-contract-validation);
- you want to know whether a **spec change** breaks clients, without a server: `clrsig diff`;
- you only want to check the **spec document** itself: `clrsig validate --spec`.

> **Before you point it at a real system**
> 1. `conform` sends **every** operation, including `POST`, `PUT`, `PATCH` and `DELETE`. Use `--read-only` anywhere data matters. (A project created by `clrsig init` is read-only by default.)
> 2. **Reports quote little of a response, but not nothing.** No report holds a whole body. The value that broke the spec may be quoted, and is masked when its field is named like a credential or it looks like a token; a secret that nothing marks as one can still appear ([What We Check](../../../apps/desktop/wails/help/what-we-check.md), CS-RUN-011 and CS-GAP-017). Releases up to 0.13.1 quoted a failing response **in full**: upgrade, and treat older reports of authentication endpoints as sensitive.

## Synopsis

```text
clrsig conform <spec-file-or-url> [flags]
```

The spec can be left out when a `clrsig.yaml` project file names it.

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--base-url`, `-b` | The server to send requests to. | The first `servers` entry of the spec. |
| `--env`, `-e` | Environment file with the values the requests need: ids, credentials, body fields. | |
| `--var` | `name=value`: set or override one value for this run. Repeatable. | Wins over the environment file. Visible in your shell history: use the file with `${NAME}` for secrets. |
| `--list` | **Sends nothing.** Lists each operation with the values it needs and the ones missing. | `text` or `json` only. Refused with the options that only matter when requests are sent. |
| `--read-only` | Sends only `GET` and `HEAD`. Everything else is shown as `SKIP`. | Off, unless the project file turns it on. |
| `--tags` | Only operations with one of these OpenAPI tags (comma separated). | A filter that matches nothing is an error. |
| `--paths` | Only operations whose path starts with one of these prefixes (comma separated), written as in the spec, `/users/{id}`. | `{{id}}` is accepted too. A filter that matches nothing is an error. |
| `--changed-against` | Only operations that are new or changed compared with the spec at a git ref, such as `origin/main`. | Needs git and a local spec file. |
| `--setup` | A scenario to run first, usually a login. What it extracts fills `{{name}}` in the requests. | If it fails, the check does not start (exit 2). |
| `--negative` | Also sends **probes**: requests that are deliberately wrong, which the server should refuse. | Do not use against production. |
| `--explain` | Under each failure in the text output, says what was expected, what came back, why it fails and how to fix it. | JSON, JUnit, HTML and Markdown always include it. |
| `--coverage` | Adds a list of the operations that were and were not reached. | |
| `--fail-fast` | Stops at the first failure. | |
| `--fail-on` | `failure`, or `warning` to fail on warnings too. | `failure`. Today every finding of `conform` is a failure. |
| `--request-timeout` | How long to wait for one answer, in milliseconds. | `10000` |
| `--max-response-time` | Fails an operation that takes longer than this many milliseconds. | No limit. Leave a margin: it measures wall-clock time. |
| `--max-response-time-for` | `/prefix=ms`: another limit for the operations under a path. Repeatable. | The longest matching path wins; `0` turns the limit off there. |
| `--format`, `-f` | What is printed: `text`, `json`, `junit`, `html`, `markdown`, or `github` (annotations on a pull request). | `text` |
| `--output`, `-o` | Writes that output to a file instead; folders are created. | |
| `--report` | `format=file`: **also** write another report from the same run. Repeatable. | Not `github`. |
| `--write-baseline` | Records today's violations in a file and exits 0. | Needs a full run: refused with `--tags`, `--paths` or `--changed-against`. |
| `--baseline` | Accepts the violations in that file: only **new** ones fail the run. | Not together with `--write-baseline`. |

The options every command has (`--ca-cert`, `--client-cert`, `--client-key`, `--config`, `--no-config`) apply too: certificates for a private CA or mutual TLS, and which project file to use.

## Where values come from

`conform` writes each value the spec needs as `{{name}}` in the request it builds: path parameters (`{{id}}`), credentials (`{{token}}`, `{{apiKey}}`, `{{basicUser}}`, `{{basicPass}}`) and required body fields (`{{email}}`). You give them in an environment file:

```yaml
schemaVersion: "1"
name: local
variables:
  - name: baseUrl
    value: http://localhost:8080
  - name: token
    value: ${API_TOKEN}      # read from the process environment when the run starts
    secret: true
```

An operation with a value **nothing provides** is **not sent**, because the request would not test it. An **empty** value counts as missing in the path and in credentials; elsewhere (query, header, cookie and body values, and a basic password) it is sent empty ([CS-GAP-013](../../../apps/desktop/wails/help/what-we-check.md)). In scripts, a flag wins over the project file, and `--var` wins over the environment file.

## Examples

### 1. See what a run will need, before sending anything

**What it does:** reads the spec and the environment and lists every operation. No request reaches your API.

<!-- run: exit=0 contains="Nothing was sent." -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --list --read-only
```

**What you will see** (exit code 0):

<!-- output -->
```text
Nothing was sent. The values each operation needs:
  SKIP   POST    /auth/login  needs {{username}}, {{password}}
  READY  GET     /health  needs nothing
  …
  READY  GET     /users/{id}  needs {{id}}
  …
8 operations  ·  4 ready  ·  0 would not be sent  ·  4 skipped
Values used: token, id
Nothing is missing.
```

`READY` would be sent, `MISS` would not, `SKIP` is left out by a filter or by `--read-only`. It exits 0 even when values are missing: it is a question, not a check.

### 2. Check the whole API

**What it does:** sends all eight operations, writes included, to a server on your machine.

<!-- run: exit=0 contains="8 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --base-url http://localhost:8080 --env examples/environments/demo-conform.yaml
```

**What you will see** (exit code 0):

<!-- output -->
```text
Base URL: http://localhost:8080
  PASS  POST    /auth/login  200  1ms
  PASS  GET     /health  200  0ms
  …
  PASS  DELETE  /users/{id}  204  0ms
PASSED  8 operations  ·  8 passed  ·  0 failed  ·  0 skipped  6ms
```

This run created, changed and deleted users. That is why the other examples use `--read-only` where they can.

### 3. When values are missing

**What it does:** the same check with an environment that has a user name and password, an **empty** token, and no ids or body fields.

<!-- run: exit=1 contains="6 not sent" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/local.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
  PASS  GET     /health  200  0ms
  FAIL  GET     /protected
         ✗ not sent: no value for {{token}} (defined but empty: {{token}}; a path value or a credential cannot be empty)
  …
  FAIL  GET     /users/{id}
         ✗ not sent: no value for {{id}}
  …
FAILED  8 operations  ·  2 passed  ·  0 failed  ·  6 not sent  ·  0 skipped  2ms
6 operations were not sent because nothing provides: token, name, email, id. Add them to the environment, or pass them as overrides (name=value).
```

`0 failed · 6 not sent`: nothing is known to be wrong with the API. Six operations were not tested, and the run fails so that this cannot be missed.

### 4. Log in first, with a setup scenario

**What it does:** runs [`login.yaml`](files/login.yaml), which logs in and extracts `token`, then checks the read operations with it. `--var` gives the id.

<!-- run: exit=0 contains="values available to the check: token" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/local.yaml --read-only \
  --setup docs/guides/cli/files/login.yaml --var id=1
```

**What you will see** (exit code 0):

<!-- output -->
```text
Setup scenario docs/guides/cli/files/login.yaml passed; values available to the check: token
…
PASSED  8 operations  ·  4 passed  ·  0 failed  ·  4 skipped  1ms
```

### 5. What a failure looks like

**What it does:** checks the server against [a spec it does not follow](files/strict-openapi.yaml), and explains each failure.

<!-- run: exit=1 contains="CS-RESP-005" contains="2 failed" -->
```bash
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --explain
```

**What you will see** (exit code 1):

<!-- output -->
```text
  FAIL  GET     /health  200  1ms
         ✗ Error at "/version": property "version" is missing
  …
  FAIL  GET     /users/{id}  200  0ms
         ✗ Error at "/nickname": property "nickname" is missing
  …
FAILED  2 operations  ·  0 passed  ·  2 failed  ·  0 skipped  1ms
```

A line under a violation may quote the one value at fault. With `--explain`, each failure ends with one line like this one:

<!-- output -->
```text
         → [CS-RESP-005] "version" is missing from the response. Why: The spec marks "version" as required, but the server's response did not include it. Where: response body, field version (spec: GET /health → response 200 → body schema). Fix: Return "version" from the server, or make it optional in the spec if it is not always present.
```

`CS-RESP-005` is the check's ID. Look it up in [What We Check](../../../apps/desktop/wails/help/what-we-check.md). Only the **first** problem of a response is reported: fix it and run again.

### 6. Adopt it on an API that fails today

**What it does:** records today's violations, then runs again accepting them. Commit the baseline file; from then on only **new** violations fail.

<!-- run: exit=0 contains="Baseline written: 2 violations" contains="PASSED with 2 known violations" -->
```bash
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --write-baseline baseline.yaml
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --baseline baseline.yaml
```

**What you will see** (both exit 0):

<!-- output -->
```text
Baseline written: 2 violations recorded in baseline.yaml
PASSED with 2 known violations  2 operations  ·  0 passed  ·  2 failed  ·  0 skipped  2ms
Baseline: 2 known, 0 new, 0 fixed
```

The **recording** run still prints `FAILED` above its last line, and exits 0: recording accepts what it found. The baseline summary goes to stderr, so `--format json` on stdout stays readable by programs. A violation is identified by its operation, check ID and field, not by the wording, so upgrading `clrsig` does not turn known violations into new ones. When some are reported as fixed, write the baseline again so they cannot come back unnoticed.

### 7. Check one part of the API, and see what was left out

<!-- run: exit=0 contains="Coverage: 2 / 8 operations (25%)" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --read-only --paths /users --coverage
```

**What you will see** (exit code 0):

<!-- output -->
```text
  SKIP  POST    /auth/login
  …
  PASS  GET     /users  200  1ms
  …
PASSED  8 operations  ·  2 passed  ·  0 failed  ·  6 skipped  2ms
Coverage: 2 / 8 operations (25%)
```

End a filter at a parameter or write the parameter as the spec does: `--paths /users/{id}`. **In Git Bash on Windows** set `MSYS_NO_PATHCONV=1` first, or the shell turns `/users` into a Windows path.

### 8. A filter that matches nothing is not a pass

<!-- run: exit=2 contains="NOTHING CHECKED" contains="no operation matches --paths /nowhere" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --paths /nowhere
```

Exit code **2**: the command was wrong, not the API.

### 9. Check that bad requests are refused

**What it does:** after the normal check, sends wrong requests: a required field left out, a wrong type, no credentials. A probe **passes** when the server refuses it.

<!-- run: exit=0 contains="11 probes" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --negative
```

**What you will see** (exit code 0):

<!-- output -->
```text
Negative conformance probes — Clear Signal Example API 1.0.0
  PASS  POST /auth/login — missing required field 'username' (expect 400/422)  422  0ms
  …
  PASS  GET /users — no credentials (expect 401/403)  401  0ms
  …
PASSED  11 probes  ·  11 passed  ·  0 failed  ·  0 skipped  ·  0 errors  1ms
```

**Read the status printed after each pass.** Any `4xx` passes, so a `401` or a rate limit's `429` can pass a probe whose body was never examined (CS-GAP-012). If no operation has anything to probe, the summary says `PASSED  0 probes`: that means nothing was tested.

### 10. Keep reports for CI

**What it does:** prints the usual output and also writes a JUnit file for the CI's test view and an HTML page to attach to the run.

<!-- run: exit=0 contains="4 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --read-only \
  --report junit=reports/conform.xml --report html=reports/conform.html
```

Use `--format github` in GitHub Actions to put each finding on the pull request:

<!-- run: exit=1 contains="::error file=docs/guides/cli/files/strict-openapi.yaml" -->
```bash
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --format github
```

A complete workflow: `clrsig init github-actions`, or the [CI guide](../ci-integration.md).

### 11. The server is not there

<!-- run: exit=1 contains="No operation got an answer from http://localhost:1: is the server running?" -->
```bash
clrsig conform examples/server/openapi.yaml --base-url http://localhost:1 --env examples/environments/demo-conform.yaml --read-only
```

Exit code 1. The first error under that line says why the connection failed.

### 12. Pull requests: only what changed

*Not run by a test: it needs git history.*

<!-- not run: needs a git repository with the base branch -->
```bash
clrsig conform specs/openapi.yaml --env environments/ci.yaml --base-url "$API_URL" --changed-against origin/main --read-only
```

Checks only operations that are new or changed compared with the spec on `origin/main`. A pull request that does not touch the spec says `NOTHING CHECKED` and passes. It follows the **spec** only: a server fault with no spec change is not seen, so also run the full check on your main branch. In CI the checkout needs the history (`fetch-depth: 0`).

## Reading the result

**Each operation line:** status, method, path as the spec writes it, the HTTP status the server answered, and the time.

| Status | Meaning |
|---|---|
| `PASS` | The answer matches what the spec declares **for the status the server gave**. |
| `WARN` | Passed, with a warning. |
| `FAIL` | A violation, no answer, or the request was not sent. The lines under it say which. |
| `SKIP` | Left out by `--tags`, `--paths`, `--changed-against` or `--read-only`. Nothing was sent. |

> **A pass is not "it worked".** If the spec declares a `404` for `GET /users/{id}` and the id does not exist, the answer conforms and the line says `PASS … 404`. The operation was never seen succeeding (CS-GAP-016). **Look at the status codes**, and give values that exist.

For example, an id that does not exist:

<!-- run: exit=0 contains="404" contains="1 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --read-only --paths /users/{id} --var id=999999
```

**The verdict** is the first word of the summary line.

| Verdict | Exit | Meaning |
|---|---|---|
| `PASSED` | 0 | Every operation that was sent conforms, and none was left unsent. |
| `PASSED with N warnings` | 0 | The same, with warnings. |
| `PASSED with N known violations` | 0 | Every failure is already in the baseline. |
| `FAILED` | 1 | At least one operation broke the spec, got no answer or was not sent. A failed probe also gives exit 1. |
| `NOTHING CHECKED` | 0 or 2 | Nothing was sent. Exit 0 when the spec declares nothing, `--read-only` skipped everything, or nothing changed; **2** when a filter matched nothing. |

The counts are `operations · passed · failed · not sent · skipped`; `not sent` appears only when there is one.

**Exit codes:** `0` passed · `1` something failed · `2` the command could not do its job (a spec or file that cannot be read, options that cannot be combined, a filter that matches nothing, a failed `--setup`).

## What to do next

| You see | It means | Do this |
|---|---|---|
| `not sent: no value for {{name}}` | The request needs a value you did not give. | Run with `--list`; add the value to the environment or pass `--var name=value`. For a token, use `--setup`. |
| `(defined but empty: {{name}}; …)` | The value exists but is empty, often an unset secret in CI. | Set it. Check the CI secret's name. |
| A violation with a check ID | The answer and the spec disagree. | Run with `--explain`. Decide which is right: fix the server, or fix the spec. |
| `the spec declares …Content-Type…; the body was not checked` | Wrong or undeclared media type, so the body was not compared. | Send the declared `Content-Type`, or declare the one you send. |
| `status … is not declared` | The server answered with a status the spec does not list. | If it is a real answer, declare it. A `404` here often means a wrong base URL or path: check the route first. |
| `401` or `403` on secured operations | Missing, wrong or expired credentials. | Check the token; use `--setup` to get a fresh one. |
| `No operation got an answer from …` | Nothing is listening, or it cannot be reached. | Check the server, `--base-url`, proxies and certificates (`--ca-cert`). |
| `no operation matches --paths …` | The filter selects nothing. | Copy the path from the spec or from `--list`. In Git Bash set `MSYS_NO_PATHCONV=1`. |
| Many failures on a first run | An API that was never checked. | `--write-baseline` once, commit the file, run with `--baseline`, and reduce it over time. |
| A probe `FAIL`: the server accepted an invalid request | Missing validation or authentication. | Fix the server. For authentication, treat it as urgent. |
| It passes and you do not believe it | A pass may be an error answer, or little may be checked. | Look at the status codes and the `skipped` count; add `--coverage`; check that the spec has schemas. |

## Common mistakes and limits

- **Running writes against shared data.** Add `--read-only`.
- **Sharing a report without reading it.** It can still hold a value from a response; see the warning at the top.
- **A green run with most operations skipped.** `PASSED` speaks only for what was sent.
- **An outdated base URL.** Without `--base-url`, the spec's first server is used.
- **Not everything is checked:** only the first branch of `oneOf` is sent, some formats and non-JSON bodies are not validated, and more. The honest list is [What We Check](../../../apps/desktop/wails/help/what-we-check.md), with its known gaps.

## Related

- `clrsig init`: create the project file so that `clrsig conform` needs no options.
- `clrsig config show conform`: what the project file will supply.
- `clrsig validate --spec`: problems in the spec itself.
- `clrsig run`: multi-step scenarios, also checked against the spec.
- `clrsig diff`: breaking changes between two versions of the spec.

## In the desktop app

**Check Conformance** (the shield in the left rail) runs the same check with the same code: choose a spec, base URL and environment. It sends read requests only until you allow writes. Options hold the filters, probes, time limits, variable overrides, the setup scenario and the baseline, and **Get CLI command** gives you the command for CI. It has no `--list`.
