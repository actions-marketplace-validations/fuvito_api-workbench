# clrsig GitHub Action

Run API scenarios, validate spec conformance, or detect breaking API changes in CI.

```yaml
- uses: fuvito/api-workbench@v1   # pinned release (recommended)
  with:
    command: run
    scenario: tests/my-scenario.yaml
    env: environments/ci.yaml
    format: junit
    output: test-results/results.xml
```

> **Tip:** `fuvito/api-workbench@v1` resolves from the repo root `action.yml` and is the form that appears in GitHub Actions Marketplace search. The subdirectory form `fuvito/api-workbench/clrsig-action@main` also works and is kept for backward compatibility.

## Commands

| Command | What it does |
|---|---|
| `run` | Execute a multi-step scenario and validate assertions |
| `conform` | Execute every operation in an OpenAPI spec against a live server and validate responses |
| `diff` | Compare two OpenAPI specs and report breaking changes |
| `validate` | Check scenario files for errors without executing them |
| `request` | Execute a single saved request file |

## Inputs

| Input | Required | Default | Description |
|---|---|---|---|
| `command` | yes | `run` | Command to run |
| `scenario` | for `run`/`validate`/`request` | — | Path to scenario or request file |
| `spec` | for `conform` | — | Path or URL to OpenAPI spec |
| `base-spec` | for `diff` | — | Path or URL to the base (older) spec |
| `revision-spec` | for `diff` | — | Path or URL to the revised (newer) spec |
| `env` | no | — | Path to environment YAML/JSON file |
| `base-url` | no | — | Override base URL from spec (`conform`) |
| `format` | no | `text` | Output format: `text`, `json`, or `junit` |
| `output` | no | — | Write report to this file |
| `fail-fast` | no | `false` | Stop after the first failure (`conform`) |
| `coverage` | no | `false` | Append operation coverage summary (`conform`) |
| `breaking-only` | no | `false` | Report only breaking changes (`diff`) |
| `version` | no | `latest` | clrsig release tag, e.g. `v1.2.0` |

## Outputs

| Output | Description |
|---|---|
| `result` | Exit code of the clrsig command (`0` = all checks passed) |

## Examples

### Run a scenario and produce a JUnit report

```yaml
- uses: fuvito/api-workbench@v1
  with:
    command: run
    scenario: tests/api-tests.yaml
    env: environments/ci.yaml
    format: junit
    output: test-results/clrsig.xml

- uses: actions/upload-artifact@v4
  if: always()
  with:
    name: api-test-results
    path: test-results/clrsig.xml
```

### Validate spec conformance

```yaml
- uses: fuvito/api-workbench@v1
  with:
    command: conform
    spec: openapi.yaml
    base-url: http://localhost:8080
    coverage: 'true'
    format: junit
    output: test-results/conform.xml
```

### Detect breaking API changes on pull requests

```yaml
- name: Check for breaking changes
  uses: fuvito/api-workbench@v1
  with:
    command: diff
    base-spec: https://raw.githubusercontent.com/${{ github.repository }}/main/openapi.yaml
    revision-spec: openapi.yaml
    breaking-only: 'true'
```

## Exit codes

| Code | Meaning |
|---|---|
| `0` | All checks passed / no breaking changes |
| `1` | One or more assertion failures, violations, or breaking changes |
| `2` | Parse error, missing argument, or configuration error |
