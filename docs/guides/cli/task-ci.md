# Task: run it all in CI

[← Guide index](README.md)

**Goal:** every pull request runs the checks, fails when it should, and keeps a report people can read.

## 1. Get it working on your machine first

[Check the API](task-check-an-api.md) by hand until the result is what you expect. A pipeline is a slow place to find a missing value.

## 2. Print a workflow for GitHub Actions

<!-- run: exit=0 contains="command: conform" contains="upload-artifact" -->
```bash
clrsig init github-actions
```

It only prints. `--write` saves it to `.github/workflows/clrsig.yml` and never replaces a file. **Edit it:** the pipeline needs a step that starts your API, or the address of one that is running. Other ways, including Docker and other CI systems, are in [CI integration](../ci-integration.md); a login is in [authenticated checks](../authed-conform.md).

## 3. Keep reports

One run can write several reports:

<!-- run: exit=0 contains="4 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --read-only \
  --report junit=reports/conform.xml --report html=reports/conform.html
```

`junit` is for the CI's test view, `html` for people, `markdown` for a pull request comment. In GitHub Actions, `--format github` marks the lines of the spec in the pull request.

## 4. What should fail the build

| Check | Fails the step when | Make it stricter |
|---|---|---|
| `conform` | A violation or an operation that could not be checked | `--fail-on warning` |
| `run` | A scenario fails | |
| `diff` | A breaking change | `--fail-on warning` or `info` |
| `validate` | An error in a file | `--strict` |
| `coverage quality` | Only with `--fail-on-gaps`: a gap, or nothing recorded | |

Give each command its own step, so each keeps its own log and exit code. Treat **any** exit code but 0 as a failure: 1 and 2 mean different things ([the table](reference.md#exit-codes)), and neither is a pass.

## Keep it safe

- Put credentials in the CI's secret store, and refer to them as `${NAME}` in the environment file. Do not pass them with `--var` in a script that is logged.
- Use `--read-only` against any server whose data matters.
- Reports mask what they recognize as a secret. They can miss one, so limit who can read the artifacts. See [what we check](../../../apps/desktop/wails/help/what-we-check.md) (CS-RUN-011).
- Pin the version of `clrsig`, and read the release notes before you raise it: a newer release can find problems an older one missed.
