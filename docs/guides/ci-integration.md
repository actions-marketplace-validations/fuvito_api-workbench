# CI/CD integration guide

`clrsig` runs in any CI/CD pipeline with no plugin or adapter: it reads files, calls your API, writes a report and returns an exit code every CI system understands. This guide covers the commands, the exit codes, the reports, the GitHub Action, other CI systems, secrets, and how to run **Spec Conformance Testing** (`clrsig conform`) safely and adopt it on an API that already has problems.

What each check reports, with an ID and an example, is in [What We Check](../../apps/desktop/wails/help/what-we-check.md) (in the app: Help, What We Check).

---

## Which command

| Command | Use it to | Typical CI trigger |
|---|---|---|
| `clrsig run <scenario-or-folder>...` | Run multi-step scenarios and assert the results (add a spec to also validate every response against it: API Contract Validation) | every push |
| `clrsig conform <spec> --base-url <url>` | Call every operation in an OpenAPI spec and check each response against it (Spec Conformance Testing) | pull requests, nightly, after deploy |
| `clrsig diff <old-spec> <new-spec>` | Find breaking changes between two versions of a spec | pull requests that touch the spec |
| `clrsig impact <old> <new> <scenarios>...` | Show which scenarios a breaking change would affect | pull requests that touch the spec |
| `clrsig validate <scenario-or-folder>...` | Check scenario files for mistakes without sending anything | every push (fast) |

## Exit codes

Every command uses the same three signals, so a CI step fails on a non-zero code without extra configuration.

| Code | `run` | `conform` | `diff` | `impact` |
|---|---|---|---|---|
| `0` | every assertion passed | every operation conformed | nothing at or above the `--fail-on` level (default: breaking) | no scenario is affected |
| `1` | an assertion failed, or a response broke the spec | a violation, a request that could not be sent (missing value), an accepted negative probe, or a new violation against `--baseline` | something at or above the `--fail-on` level | a step matches a breaking change |
| `2` | a file could not be read or parsed, or the run could not start | the spec could not be read, or a configuration error | a spec could not be read | a spec could not be read |

## Reports

`--format text|json|junit|html|markdown` chooses the output and `--output <file>` writes it to a file (missing folders are created). To produce several reports from **one run**, repeat `--report format=file`:

```sh
clrsig conform openapi.yaml --base-url "$API_URL" --env environments/ci.yaml \
  --report junit=reports/conform.xml \
  --report html=reports/conform.html
```

| Format | Use it for |
|---|---|
| `text` (default) | build logs: plain text, no colour codes |
| `junit` | native test results in Jenkins, GitHub, GitLab, CircleCI, with trends |
| `html` | a single self-contained page to keep as a build artifact |
| `markdown` | a pull-request comment or a job summary |
| `json` | custom dashboards, webhooks, `jq` |

Secret values are replaced by `***` in every report. For `conform`, JSON, HTML, Markdown and JUnit always include an explanation of each failure (what, where, why, how to fix it); add `--explain` to see it in the text output as well.

---

## GitHub Actions

New project? `clrsig init github-actions` prints a ready workflow for this action (and `--write` saves it to `.github/workflows/clrsig.yml`). If the repository has a `clrsig.yaml`, the action finds it from the checkout and uses it for any option the step does not set; an input always wins.

The action installs the right `clrsig` for the runner, runs one command and passes its exit code on.

```yaml
- uses: fuvito/api-workbench@v0.12.0
  with:
    command: conform
    spec: openapi.yaml
    base-url: ${{ vars.API_BASE_URL }}
    env: environments/ci.yaml
    format: junit
    output: reports/conform.xml
```

| Input | Used by | Meaning |
|---|---|---|
| `command` | all | `run`, `conform`, `diff`, `request` or `validate` |
| `scenario` | run, validate, request | a scenario file, or for `run` a folder |
| `spec` | conform | OpenAPI spec path or URL |
| `base-spec`, `revision-spec` | diff | the two spec versions |
| `env` | run, conform, request | environment file |
| `base-url` | conform | overrides the spec's `servers[0]` |
| `format`, `output` | all | report format and file |
| `fail-fast` | conform | stop at the first failure |
| `coverage` | conform | append a coverage summary |
| `breaking-only` | diff | show only breaking changes |
| `fail-on` | diff | `breaking`, `warning` or `info`: how serious a change must be to fail |
| `against` | diff | compare `base-spec` with the spec at this git ref (checkout with `fetch-depth: 0`) |
| `read-only` | conform | send only GET and HEAD; other operations are skipped and never sent |
| `changed-against` | conform | check only the operations that are new or changed compared with the spec at this git ref (for pull requests; checkout with `fetch-depth: 0`) |
| `strict` | validate | also report valid but weak scenarios (steps that check nothing, extractions nobody uses; names nothing defines when `env` is set) as warnings that fail the step |
| `setup` | conform | a scenario to run first, usually a login: what it extracts (a token) is available to the generated requests as `{{name}}`; if it fails the check does not start |
| `conform-fail-on` | conform | `failure` (default) or `warning`: with `warning`, findings that are only warnings also fail the run |
| `negative` | conform | also run negative probes |
| `explain` | conform | explain each failure in the text report |
| `baseline`, `write-baseline` | conform | fail only on violations not in the baseline file / record the current ones |
| `tags`, `paths` | conform | comma-separated OpenAPI tags and path prefixes to include; a filter that matches no operation fails the step (exit 2) instead of passing. On Windows Git Bash set `MSYS_NO_PATHCONV=1` so a leading `/` is not rewritten |
| `request-timeout` | conform | per-request timeout in milliseconds |
| `max-response-time` | conform | fail an operation slower than this many milliseconds (the response is still compared with the spec) |
| `max-response-time-for` | conform | other limits for path prefixes, one `/prefix=ms` per line, for example `/reports=2000` (the longest prefix wins; `0` turns the limit off there) |
| `var` | run, conform | environment overrides, one `name=value` per line (not for secrets) |
| `report` | run, conform, diff | extra reports, one `format=file` per line |
| `version` | all | `clrsig` release to use (default: latest) |

The step's `result` output is the exit code. Inputs reach the script as environment variables, so a value with quotes or shell characters cannot change what runs. Secrets belong in the step's `env:` and in the environment file as `${NAME}`, not in `var`. For anything the action does not expose (`--seed`, `--show-extracted`, client certificates, `bundle`, `coverage`, `impact`), install the CLI yourself (below) and call `clrsig` directly.

```yaml
- uses: fuvito/api-workbench@v0.12.0
  with:
    command: conform
    spec: specs/openapi.yaml
    base-url: ${{ vars.API_BASE_URL }}
    read-only: 'true'
    baseline: conform-baseline.json
    report: |
      junit=reports/conform.xml
      html=reports/conform.html
```

### Calling the CLI directly (all options)

```yaml
name: API conformance
on:
  pull_request:
  schedule:
    - cron: '0 3 * * *'

jobs:
  conform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v7

      - name: Install clrsig
        run: |
          curl -fsSL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 \
            -o /usr/local/bin/clrsig && chmod +x /usr/local/bin/clrsig

      - name: Check the API against its spec
        env:
          API_TOKEN: ${{ secrets.API_TOKEN }}   # used as ${API_TOKEN} in environments/ci.yaml
        run: |
          clrsig conform openapi.yaml \
            --base-url "${{ vars.API_BASE_URL }}" \
            --env environments/ci.yaml \
            --read-only \
            --baseline conform-baseline.yaml \
            --report junit=reports/conform.xml \
            --report html=reports/conform.html

      - name: Publish results
        if: always()
        uses: dorny/test-reporter@v1
        with:
          name: Conformance
          path: reports/conform.xml
          reporter: java-junit

      - name: Keep the report
        if: always()
        uses: actions/upload-artifact@v7
        with:
          name: conformance-report
          path: reports/conform.html
```

### Findings on the pull request

`format: github` (`clrsig conform --format github`) prints each finding as a GitHub workflow command, so it shows up in the pull request's **Checks** view and as an annotation, with the check ID in the title (`CS-RESP-005 GET /users/{id}`), what is wrong, why it fails and how to fix it, and, when the spec is a file in the repository, on the line of the operation. It also works for `clrsig run` (`command: run`, `format: github`): each failed assertion, unsent request or response that breaks the spec is annotated on the line of its step in the scenario file. It prints only those commands plus one summary notice, so keep the usual reports with `report:` (for example `junit=reports/conform.xml`). Failures are errors, warnings are warnings, and the exit code is the same as for any other format. GitHub shows only the first ten annotations of each kind per step in the pull request view; the log has all of them.

```yaml
- uses: fuvito/api-workbench@v0.12.0
  with:
    command: conform
    spec: openapi.yaml
    base-url: ${{ vars.API_BASE_URL }}
    format: github
    report: |
      junit=reports/conform.xml
      html=reports/conform.html
```

### Only what a pull request changed

```yaml
      - uses: actions/checkout@v7
        with: {fetch-depth: 0}           # the comparison needs the history of the target branch
      - uses: fuvito/api-workbench@v0.12.0
        with:
          command: conform
          spec: openapi.yaml
          base-url: ${{ vars.API_BASE_URL }}
          changed-against: origin/main
          read-only: 'true'
```

`--changed-against <ref>` checks only the operations that are **new or changed** compared with the spec at that ref (changes to a schema that several operations share count for each of them), with the same checks and exit codes as a full run. A pull request that does not touch the API passes with `NOTHING CHECKED`. It follows the **spec only**: a server regression with no spec change is invisible to it, and a change that belongs to no operation (a different `servers` list, global `security`) does not widen the check; the report says how many such changes there were. Keep a full `conform` on your main branch and nightly. With `--baseline`, entries of operations that were not checked are neither known nor fixed, and `--write-baseline` refuses to run on a limited run (record it with a full run).

### Breaking changes on pull requests

```yaml
      - uses: fuvito/api-workbench@v0.12.0
        with:
          command: diff
          base-spec: openapi-main.yaml      # for example the spec from the target branch
          revision-spec: openapi.yaml
          breaking-only: 'true'
```

Or without checking out the old file: `clrsig diff openapi.yaml --against origin/main` compares a spec with the version at a git ref. `--fail-on warning` or `--fail-on info` make less serious changes fail the run too.

---

## Other CI systems

Each of these installs the binary, runs a command and publishes JUnit. Replace `conform ...` with whichever command you need.

### Jenkins (declarative)

```groovy
pipeline {
  agent any
  environment { API_TOKEN = credentials('api-token') }   // used as ${API_TOKEN} in the environment file
  stages {
    stage('Install') {
      steps {
        sh 'curl -fsSL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 -o clrsig && chmod +x clrsig'
      }
    }
    stage('Conformance') {
      steps {
        sh './clrsig conform openapi.yaml --base-url "$API_BASE_URL" --env environments/ci.yaml --report junit=reports/conform.xml'
      }
    }
  }
  post { always { junit 'reports/*.xml' } }
}
```

To record results without failing the build, end the command with `|| true` and publish the JUnit file.

### GitLab CI

```yaml
conformance:
  image: alpine:3
  before_script:
    - apk add --no-cache curl
    - curl -fsSL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 -o /usr/local/bin/clrsig && chmod +x /usr/local/bin/clrsig
  script:
    - clrsig conform openapi.yaml --base-url "$API_BASE_URL" --env environments/ci.yaml --report junit=reports/conform.xml
  artifacts:
    when: always
    reports:
      junit: reports/conform.xml
```

### CircleCI

```yaml
version: 2.1
jobs:
  conformance:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install clrsig
          command: |
            curl -fsSL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 -o clrsig
            chmod +x clrsig
      - run:
          name: Conformance
          command: ./clrsig conform openapi.yaml --base-url "$API_BASE_URL" --env environments/ci.yaml --report junit=reports/conform.xml
      - store_test_results:
          path: reports
workflows:
  main:
    jobs: [conformance]
```

### Buildkite

```yaml
steps:
  - label: "API conformance"
    command: |
      curl -fsSL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 -o clrsig && chmod +x clrsig
      ./clrsig conform openapi.yaml --base-url "$API_BASE_URL" --env environments/ci.yaml --report junit=reports/conform.xml
    artifact_paths: "reports/*"
```

---

## Secrets and environment values

An environment file can refer to a variable of the CI job with `${NAME}`. It is expanded **when the run starts**, from the process environment, and the expanded value is never written to disk:

```yaml
schemaVersion: "1"
name: CI
variables:
  - name: token
    value: "${API_TOKEN}"     # filled from the CI job's API_TOKEN
    secret: true              # shown as *** in every report
  - name: baseUrl
    value: "${API_BASE_URL}"
```

- Use the CI system's own secret store (GitHub secrets, Jenkins credentials, GitLab CI/CD variables, CircleCI environment variables) and map the secret to the job's environment.
- `$${NAME}` is a literal `${NAME}`. If a `${NAME}` has no value in the job, `clrsig` prints one warning per variable and leaves the text as written; the run does not fail by itself.
- `--var name=value` (repeatable) sets or overrides a value for one run, for example `--var id=1`. Command lines are visible to other processes, so do not use it for secrets.
- Do not commit environment files that contain real values.

### Values a request needs

`conform` builds each request from the spec and fills `{{placeholders}}` (path parameters, `{{token}}` for secured operations, body fields) from the environment. A request that still has a placeholder nothing provides is **not sent**: it is marked `FAIL` with `not sent: no value for {{id}}` (the counts line shows it as `not sent`, apart from `failed`), the run exits `1`, and one line lists every missing name once. Add those names to the environment file or pass them with `--var`. A variable that is defined but never used is ignored.

---

## Conformance in CI

Spec Conformance Testing sends real requests, so decide what it may do, and how strict a first run should be.

**1. Choose what it may send.** By default `conform` sends every operation, including `POST`, `PUT`, `PATCH` and `DELETE`. Against anything shared or production-like use `--read-only`: only `GET` and `HEAD` are sent, and every other operation is reported as skipped and never sent. Limit the scope with `--tags users,orders` or `--paths /users`.

**2. Adopt it on an API that already has violations.** Record today's state once, commit the file, and let CI fail only on new problems:

```sh
clrsig conform openapi.yaml --base-url "$API_URL" --env environments/ci.yaml --write-baseline conform-baseline.yaml   # exits 0
clrsig conform openapi.yaml --base-url "$API_URL" --env environments/ci.yaml --baseline conform-baseline.yaml         # fails only on new violations
```

Violations that were fixed are reported, so the file can be regenerated and committed to ratchet quality up. Do this too after upgrading `clrsig`: releases before the 2026-11-20 change did not check declared response headers, so a response whose operation declared any header could pass with a broken body, and a run that used to be green may now report real problems.

**3. Know what fails the run.** Exit code `1` means at least one of: a response that does not match the spec (wrong status, Content-Type, header or body), a request that could not be built or sent, an operation that was not sent for lack of a value, a negative probe that the server accepted (`--negative`), or, with `--baseline`, a violation that is not in the file. Skipped operations and coverage signals do not fail the run. Every message says what is wrong, where, why, and how to fix it; the full list with IDs, including what is **not** checked, is in [What We Check](../../apps/desktop/wails/help/what-we-check.md).

**4. Prove the CI step itself.** The repository's own pipeline runs `clrsig conform` against the example server (`examples/server`) with `examples/environments/demo-conform.yaml` and expects it to pass, with and without `--negative`, and runs `clrsig diff` on two example specs and expects it to find breaking changes: a useful template for your own smoke step.

---

## Running many scenarios

`clrsig run` takes files and folders. A folder is searched recursively (`--no-recursive` to stop that), and only files that look like scenarios are run:

```sh
clrsig run scenarios/ --env environments/ci.yaml --report junit=reports/run.xml
clrsig run scenarios/ --fail-fast        # stop after the first failing scenario
```

All scenarios are parsed before any request is sent, and you get one combined report and one exit code. Add `--spec openapi.yaml` (or `specFile:` in a scenario) to validate every response against the spec, and `--coverage` to list the spec operations the scenarios exercised.

## JSON for custom reporting

```sh
clrsig run scenarios/smoke.yaml --format json | jq '{
  scenario: .scenario,
  passed: .passed,
  duration_s: (.durationMs / 1000),
  failed_steps: [.steps[] | select(.passed == false) | .name]
}'
```

Pipe the result to `curl` to post it to a webhook, a chat channel or any observability tool that accepts JSON.

## Starting the API in CI

Start your application as a service container (GitHub Actions `services:`, GitLab CI `services:`, a CircleCI secondary Docker image) or as a background process, wait for its health endpoint, then run `clrsig` against `http://localhost:<port>`. To try the pattern locally, the repository ships a demo server:

```sh
cd examples/server && go run .        # terminal 1
clrsig run examples/scenarios/user-lifecycle.yaml --env examples/environments/local.yaml   # terminal 2
```

## A recommended pipeline

1. **Smoke**, on every push: `clrsig validate` on the scenario folder, then 2 or 3 critical scenarios with `--report junit=...`. Hard failure.
2. **Conformance**, on pull requests and nightly against staging: `clrsig conform --read-only --baseline conform-baseline.yaml` with JUnit published and the HTML report kept as an artifact.
3. **Spec changes**, on pull requests that touch the spec: `clrsig diff --against origin/main --breaking-only`, plus `clrsig impact` to see which scenarios a change affects.
4. **After deploy**, against production, read-only: `clrsig conform --read-only --negative` is not recommended there (probes send deliberately wrong requests); use `--read-only` alone and a `--tags` or `--paths` scope, and open a ticket from the JSON report on failure.
