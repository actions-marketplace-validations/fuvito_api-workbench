# Your first check

[← Guide index](README.md)

This page takes you from nothing to a finished **API Contract Validation** run, using the example server that comes with the repository. Nothing is sent anywhere but to your own machine.

## What Clear Signal checks

You have a **spec**: an OpenAPI 3.0 or 3.1 file that says what your API accepts and answers. Clear Signal sends requests to the **running** API and compares each answer with the spec: the status, the Content-Type, the declared headers and the body. That is **Spec Conformance Testing**. You write no tests for it.

Four things come up on every page:

| Word | What it is |
|---|---|
| spec | Your OpenAPI file. It is the contract. |
| environment | A file with the values the requests need: the server address, ids, credentials. |
| scenario | A file you write when one request is not enough: log in, create, read, delete, with checks of your own. |
| project file | `clrsig.yaml`. It remembers your spec, environment and server, so commands need fewer flags. |

## 1. Install

- **Windows:** `scoop bucket add clrsig https://github.com/fuvito/api-workbench`, then `scoop install clrsig`.
- **macOS and Linux:** `brew install fuvito/api-workbench/clrsig`.
- **Without a package manager:** download the binary for your system from the [releases page](https://github.com/fuvito/api-workbench/releases/latest). The binaries are not signed: on macOS run `xattr -d com.apple.quarantine ./clrsig` once; on Windows, SmartScreen may ask you to confirm.
- **Docker:** `docker run --rm -v "$PWD:/work" -w /work ghcr.io/fuvito/api-workbench:latest --version`.

Check that it works. Use version 0.13.4 or later: earlier releases can write secrets from your API into reports, or pass a `schema` check that checked nothing.

<!-- run: exit=0 contains="clrsig version" -->
```bash
clrsig --version
```

## 2. Start the example server

In a terminal of its own, from the repository root, and leave it running:

```bash
go run ./examples/server
```

It listens on `http://localhost:8080`. Run the rest from the repository root, in a second terminal.

## 3. Set up a project and see what a check will need

**What it does:** `init` creates a folder with a project file and an environment file. `--list` then shows what a check would send, without sending anything.

<!-- run: exit=0 first="0" contains="Nothing was sent." contains="would not be sent" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig conform --config my-check/clrsig.yaml --list
```

**What you will see** (both exit 0):

<!-- output -->
```text
Created in my-check:
  clrsig.yaml
  environments/local.yaml
  …
The spec needs: username, password, token, name, email, id.
They are listed in environments/local.yaml as comments: remove the # and fill in the ones you need.
```

<!-- output -->
```text
  SKIP   POST    /auth/login  needs {{username}}, {{password}}
  READY  GET     /health  needs nothing
  MISS   GET     /protected  needs {{token}}  (not sent: no value for {{token}})
  …
  MISS   GET     /users/{id}  needs {{id}}  (not sent: no value for {{id}})
  …
8 operations  ·  1 ready  ·  3 would not be sent  ·  4 skipped
```

`SKIP` here is on purpose: a new project is **read-only**, so it sends only `GET` and `HEAD`. `MISS` means a value is missing: a token and a user id.

> You can `cd my-check` instead and leave `--config` out: `clrsig` finds `clrsig.yaml` in the current folder or above it.

## 4. Run the check before the values are there

Seeing this once saves confusion later.

<!-- run: exit=1 first="0" contains="3 not sent" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig conform --config my-check/clrsig.yaml
```

**What you will see** (exit code 1):

<!-- output -->
```text
  PASS  GET     /health  200  1ms
  FAIL  GET     /protected
         ✗ not sent: no value for {{token}}
  …
FAILED  8 operations  ·  1 passed  ·  0 failed  ·  3 not sent  ·  4 skipped  1ms
```

`0 failed · 3 not sent`: nothing is known to be wrong with the API. Three operations were **not tested**, and the run fails so that you cannot miss it.

## 5. Give the values and run again

Open `my-check/environments/local.yaml`, remove the `#` in front of `token` and `id`, and give them values. For the example server the token is `demo-token-12345`, a made-up value that only it accepts, and user `1` exists.

For one run you can also give a value on the command line, which is what this example does:

<!-- run: exit=0 first="0" contains="4 passed" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig conform --config my-check/clrsig.yaml --var token=demo-token-12345 --var id=1
```

**What you will see** (exit code 0):

<!-- output -->
```text
  SKIP  POST    /auth/login
  PASS  GET     /health  200  1ms
  PASS  GET     /protected  200  0ms
  PASS  GET     /users  200  0ms
  …
PASSED  8 operations  ·  4 passed  ·  0 failed  ·  4 skipped  2ms
```

For a real secret, do not use `--var` (your shell keeps a history). Write `value: ${API_TOKEN}` in the environment file and set the variable in your shell or CI.

## 6. Keep a report

<!-- run: exit=0 first="0" contains="4 passed" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig conform --config my-check/clrsig.yaml --var token=demo-token-12345 --var id=1 --report html=my-check/reports/conform.html
```

Open the HTML file in a browser. Before sharing any report, read it: see the warning on the [`conform` page](conform.md#purpose).

## Where to go next

| You want to | Read |
|---|---|
| Understand every option of the check, and what each result means | [`clrsig conform`](conform.md) |
| Check writes too (create, change, delete) | [`conform`, example 2](conform.md#2-check-the-whole-api): only against data you can lose |
| Test a sequence of requests | [`clrsig run`](run.md) |
| Run it on every pull request | [`clrsig init github-actions`](init.md) |
| See which settings a command will use | [`clrsig config show`](config-show.md) |
