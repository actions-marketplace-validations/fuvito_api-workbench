# CI/CD Integration Guide

ApiDesk's CLI (`apidesk`) is designed to run in any CI/CD pipeline without configuration beyond the scenario file and an optional environment file.

---

## How it works

```
apidesk run <scenario.yaml> [--env <env.yaml>] [--format text|json|junit]
```

**Exit codes** — the universal CI signal:

| Code | Meaning |
|------|---------|
| `0` | All steps passed all assertions |
| `1` | One or more assertions failed (request succeeded but response was wrong) |
| `2` | Execution error — file not found, YAML parse error, network failure |

Every CI system (Jenkins, GitHub Actions, GitLab, CircleCI, Buildkite, …) treats a non-zero exit code as a build failure automatically.

---

## Output formats

| Flag | Use case |
|------|---------|
| `--format text` (default) | Human-readable, CI-safe (no ANSI escape codes), appears cleanly in build logs |
| `--format json` | Machine-readable; pipe to `jq` or ingest into dashboards / Slack bots |
| `--format junit` | JUnit XML; enables native test result display in Jenkins, GitHub Actions, GitLab CI, CircleCI |

**JUnit XML is the recommended format for any CI system that can display test results** — it gives you pass/fail counts, per-step details, and historical trend graphs without any extra tooling.

---

## GitHub Actions

The release workflow (`.github/workflows/release.yml`) already ships pre-built binaries. For testing against a live API in a workflow:

```yaml
name: API conformance

on:
  push:
    branches: [main]
  pull_request:

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Download apidesk CLI
        run: |
          curl -sSL https://github.com/fuvito/api-workbench/releases/latest/download/apidesk-linux-amd64 \
            -o /usr/local/bin/apidesk && chmod +x /usr/local/bin/apidesk

      - name: Run scenario
        env:
          API_BASE_URL: ${{ vars.API_BASE_URL }}
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: |
          apidesk run examples/scenarios/user-lifecycle.yaml \
            --env examples/environments/ci.yaml \
            --format junit > test-results.xml

      - name: Publish test results
        uses: dorny/test-reporter@v1
        if: always()
        with:
          name: Scenario Results
          path: test-results.xml
          reporter: java-junit
```

> **Secrets** — use `${{ secrets.NAME }}` in GitHub Actions; map them to environment variable names and reference them as `{{NAME}}` inside your environment file or scenario URL/headers.

---

## Jenkins

### Declarative Pipeline

```groovy
pipeline {
    agent any

    environment {
        API_TOKEN = credentials('api-token-credential-id')
    }

    stages {
        stage('API Conformance') {
            steps {
                sh '''
                    apidesk run scenarios/user-lifecycle.yaml \
                        --env environments/staging.yaml \
                        --format junit > junit-results.xml
                '''
            }
        }
    }

    post {
        always {
            // Publish to Jenkins test result UI
            junit 'junit-results.xml'
        }
        failure {
            // Optional: send Slack notification
            slackSend(color: 'danger', message: "API conformance failed: ${env.JOB_NAME} #${env.BUILD_NUMBER}")
        }
    }
}
```

### Freestyle project

1. Add a **Execute shell** build step:
   ```sh
   apidesk run scenarios/smoke.yaml --format junit > results.xml
   ```
2. Add a **Post-build action → Publish JUnit test result report**, pattern: `results.xml`
3. Jenkins will show pass/fail trends on the job page automatically.

### Reporting failure without failing the build

If you want warnings rather than hard failures (e.g. for non-critical checks):

```sh
apidesk run scenarios/optional-checks.yaml --format junit > results.xml || true
```

The `|| true` prevents the shell step from failing the build; Jenkins still records the test failures in the results report.

---

## GitLab CI

```yaml
api-conformance:
  image: ubuntu:22.04
  stage: test
  before_script:
    - curl -sSL $APIDESK_DOWNLOAD_URL -o /usr/local/bin/apidesk && chmod +x /usr/local/bin/apidesk
  script:
    - apidesk run scenarios/smoke.yaml --env environments/ci.yaml --format junit > results.xml
  artifacts:
    when: always
    reports:
      junit: results.xml   # GitLab parses this natively — results appear in MR pipelines
    paths:
      - results.xml
    expire_in: 7 days
  variables:
    API_TOKEN: $API_TOKEN   # set in Settings → CI/CD → Variables
```

GitLab renders per-step pass/fail counts directly on merge requests when the `junit` artifact key is set.

---

## CircleCI

```yaml
version: 2.1

jobs:
  api-conformance:
    docker:
      - image: cimg/base:stable
    steps:
      - checkout
      - run:
          name: Install apidesk
          command: |
            curl -sSL $APIDESK_DOWNLOAD_URL -o /usr/local/bin/apidesk
            chmod +x /usr/local/bin/apidesk
      - run:
          name: Run scenarios
          command: |
            mkdir -p test-results
            apidesk run scenarios/smoke.yaml \
              --env environments/ci.yaml \
              --format junit > test-results/results.xml
      - store_test_results:
          path: test-results   # CircleCI reads JUnit XML from this directory
      - store_artifacts:
          path: test-results

workflows:
  conformance:
    jobs:
      - api-conformance
```

---

## Buildkite

```yaml
steps:
  - label: "API conformance"
    command: |
      apidesk run scenarios/smoke.yaml \
        --env environments/ci.yaml \
        --format junit > junit.xml
    artifact_paths: "junit.xml"
    plugins:
      - junit-annotate#v2.4.1:
          artifacts: junit.xml
```

---

## Running multiple scenarios

Use a shell glob or loop — each invocation reports independently and the last non-zero exit code determines overall pass/fail:

```sh
#!/bin/bash
set -e
mkdir -p results

for scenario in scenarios/*.yaml; do
  name=$(basename "$scenario" .yaml)
  apidesk run "$scenario" \
    --env environments/ci.yaml \
    --format junit > "results/${name}.xml"
done
```

GitHub Actions `dorny/test-reporter` and Jenkins' `junit` post-build action both accept glob patterns (`results/*.xml`), so all results appear as a single report.

---

## Injecting secrets and environment variables

ApiDesk resolves `{{VARIABLE_NAME}}` placeholders at run time from the active environment file.

**Environment file** (`environments/ci.yaml`):

```yaml
schemaVersion: "1"
name: CI
variables:
  - name: BASE_URL
    value: "{{API_BASE_URL}}"   # resolved from OS environment variable
  - name: TOKEN
    value: "{{API_TOKEN}}"
    secret: true                # redacted in all logs and reports
```

The double-brace syntax passes through unresolved if no environment variable with that name exists — the run will likely fail with a clear error. This surfaces missing secret configuration early.

**Never commit secret values to your scenario or environment files.** Use your CI system's secret store and reference them as OS-level environment variables.

---

## JSON format for custom reporting

```sh
apidesk run scenarios/smoke.yaml --format json | jq '
  {
    scenario: .scenario,
    passed: .passed,
    duration_s: (.durationMs / 1000),
    failed_steps: [.steps[] | select(.passed == false) | .name]
  }
'
```

Pipe to `curl` to send results to a webhook, Slack, or any observability platform that accepts JSON.

---

## Recommended CI workflow

1. **Smoke** — 2–3 critical paths, runs on every push, `--format junit`, hard failure.
2. **Conformance** — full scenario suite against staging, runs nightly or on release tags, `--format junit`, results published.
3. **Contract validation** — OpenAPI schema assertions against production (read-only), runs post-deploy, failures open an incident ticket via JSON webhook.
