# clrsig GitHub Action

Run API scenarios, validate spec conformance, or detect breaking API changes in CI.

```yaml
- uses: fuvito/api-workbench@v0.13.3   # an exact release tag
  with:
    command: run
    scenario: tests/my-scenario.yaml
    env: environments/ci.yaml
    format: junit
    output: test-results/results.xml
```

> **Which reference to use:** write an exact release tag, as above; there is no moving `v1` tag yet (it starts with 1.0.0). The root form is the one GitHub Actions Marketplace shows. The subdirectory form (`fuvito/api-workbench/clrsig-action`) is kept for older workflows and is the same file as the root `action.yml`.
>
> The Action checks the downloaded `clrsig` against the SHA-256 published with the release and fails if it is missing or different. This catches a damaged or replaced download; it is not a signature, so it does not protect against a compromised release.

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
| `spec` | for `conform` | — | Path or URL to OpenAPI spec (with `validate`, a spec to check itself) |
| `base-spec` | for `diff` | — | Path or URL to the base (older) spec |
| `revision-spec` | for `diff` | — | Path or URL to the revised (newer) spec |
| `env` | no | — | Path to environment YAML/JSON file |
| `base-url` | no | — | Override base URL from spec (`conform`) |
| `format` | no | `text` | Output format: `text`, `json`, `junit`, `html` or `markdown` |
| `output` | no | — | Write report to this file |
| `fail-fast` | no | `false` | Stop after the first failure (`conform`) |
| `coverage` | no | `false` | Append operation coverage summary (`conform`) |
| `breaking-only` | no | `false` | Report only breaking changes (`diff`) |
| `fail-on` | no | — | `breaking`, `warning` or `info`: how serious a change must be to fail (`diff`) |
| `against` | no | — | Compare `base-spec` with the spec at this git ref (`diff`; checkout with `fetch-depth: 0`) |
| `read-only` | no | `false` | Send only GET and HEAD requests (`conform`) |
| `negative` | no | `false` | Also run negative probes (`conform`) |
| `explain` | no | `false` | Explain each failure in the text report (`conform`) |
| `baseline` | no | — | Accept violations recorded in this file; only new ones fail (`conform`) |
| `write-baseline` | no | — | Record the current violations in this file and succeed (`conform`) |
| `tags`, `paths` | no | — | Comma-separated OpenAPI tags and path prefixes to include (`conform`) |
| `request-timeout` | no | — | Per-request timeout in milliseconds (`conform`) |
| `max-response-time` | no | — | Fail an operation whose response takes longer than this many milliseconds; the response is still compared with the spec (`conform`) |
| `max-response-time-for` | no | — | Other limits for path prefixes, one `/prefix=ms` per line (`conform`) |
| `changed-against` | no | — | Check only operations new or changed since this git ref (`conform`; checkout with `fetch-depth: 0`) |
| `setup` | no | — | A scenario to run first, usually a login (`conform`) |
| `conform-fail-on` | no | — | `failure` (default) or `warning` (`conform`) |
| `strict` | no | `false` | Also warn about weak scenarios (`validate`) |
| `var` | no | — | Environment overrides, one `name=value` per line (`run`, `conform`); not for secrets |
| `report` | no | — | Extra reports, one `format=file` per line (`run`, `conform`, `diff`) |
| `version` | no | `latest` | clrsig release tag, e.g. `v1.2.0` |

## Outputs

| Output | Description |
|---|---|
| `result` | Exit code of the clrsig command (`0` = all checks passed) |

## Examples

### Run a scenario and produce a JUnit report

```yaml
- uses: fuvito/api-workbench@v0.13.3
  with:
    command: run
    scenario: tests/api-tests.yaml
    env: environments/ci.yaml
    format: junit
    output: test-results/clrsig.xml

- uses: actions/upload-artifact@v7
  if: always()
  with:
    name: api-test-results
    path: test-results/clrsig.xml
```

### Validate spec conformance

```yaml
- uses: fuvito/api-workbench@v0.13.3
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
  uses: fuvito/api-workbench@v0.13.3
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
