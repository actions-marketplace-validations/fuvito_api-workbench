# Clear Signal CI/CD Integration Guide

Clear Signal CLI (`clrsig`) is designed to drop into any CI/CD pipeline without plugins or adapters. Its three commands produce machine-readable output and return meaningful exit codes that most CI systems understand natively.

---

## Commands and What They Do

| Command | Purpose |
|---|---|
| `clrsig run <scenario.yaml>` | Execute a multi-step API scenario and assert results |
| `clrsig conform <spec.yaml> --base-url <url>` | Run every operation in an OpenAPI spec against a live server |
| `clrsig diff <base.yaml> <revision.yaml>` | Detect breaking changes between two OpenAPI spec versions |

---

## Output Formats

All three commands accept `--format` and `--output` flags.

```
--format   text | json | junit   (default: text)
--output   path/to/file          (default: stdout)
```

| Format | Use case |
|---|---|
| `text` | Human-readable terminal output; suitable for build logs |
| `json` | Machine-readable; use for custom dashboards or downstream tooling |
| `junit` | JUnit XML; understood natively by Jenkins, GitHub Actions, GitLab CI, CircleCI |

### Exit Codes

#### `clrsig run`
| Code | Meaning |
|---|---|
| 0 | All steps passed |
| 1 | One or more assertions failed |
| 2 | Parse error, file not found, or execution error |

#### `clrsig conform`
| Code | Meaning |
|---|---|
| 0 | All operations conform to the spec |
| 1 | One or more operations returned violations |
| 2 | Error loading spec, connecting to server, or invalid arguments |

#### `clrsig diff`
| Code | Meaning |
|---|---|
| 0 | No breaking changes detected |
| 1 | Breaking changes found |
| 2 | Error loading or parsing specs |

---

## GitHub Actions

### Scenario run with JUnit report

```yaml
# .github/workflows/api-tests.yml
name: API Tests

on:
  push:
    branches: [main]
  pull_request:

jobs:
  api-tests:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Set up Go
        uses: actions/setup-go@v5
        with:
          go-version: '1.23'

      - name: Build clrsig CLI
        run: go build -o clrsig ./apps/cli

      - name: Run API scenarios
        env:
          API_BASE_URL: ${{ secrets.API_BASE_URL }}
          API_TOKEN: ${{ secrets.API_TOKEN }}
        run: |
          ./clrsig run examples/scenarios/user-lifecycle.yaml \
            --env environments/ci.yaml \
            --format junit \
            --output results/scenario-results.xml

      - name: Publish test results
        uses: dorny/test-reporter@v1
        if: always()
        with:
          name: API Scenario Results
          path: results/scenario-results.xml
          reporter: java-junit
```

### Spec conformance check

```yaml
      - name: Run spec conformance
        env:
          API_BASE_URL: ${{ secrets.API_BASE_URL }}
        run: |
          ./clrsig conform openapi.yaml \
            --base-url $API_BASE_URL \
            --format junit \
            --output results/conform-results.xml

      - name: Publish conformance results
        uses: dorny/test-reporter@v1
        if: always()
        with:
          name: Spec Conformance
          path: results/conform-results.xml
          reporter: java-junit
```

### Breaking change detection on PRs

```yaml
      - name: Check for breaking API changes
        run: |
          ./clrsig diff openapi-base.yaml openapi.yaml --format text
        # Exit code 1 = breaking changes found → PR is blocked
```

---

## Jenkins

### Declarative Jenkinsfile

```groovy
pipeline {
    agent any

    environment {
        API_BASE_URL = credentials('api-base-url')
        API_TOKEN    = credentials('api-token')
    }

    stages {
        stage('Build CLI') {
            steps {
                sh 'go build -o clrsig ./apps/cli'
            }
        }

        stage('Run API Scenarios') {
            steps {
                sh '''
                    ./clrsig run examples/scenarios/user-lifecycle.yaml \
                        --env environments/ci.yaml \
                        --format junit \
                        --output results/scenario-results.xml
                '''
            }
            post {
                always {
                    junit 'results/scenario-results.xml'
                }
            }
        }

        stage('Spec Conformance') {
            steps {
                sh '''
                    ./clrsig conform openapi.yaml \
                        --base-url ${API_BASE_URL} \
                        --format junit \
                        --output results/conform-results.xml
                '''
            }
            post {
                always {
                    junit 'results/conform-results.xml'
                }
            }
        }

        stage('Breaking Change Check') {
            when {
                changeRequest()
            }
            steps {
                sh './clrsig diff openapi-base.yaml openapi.yaml'
                // Non-zero exit code fails the stage and blocks the PR
            }
        }
    }

    post {
        always {
            archiveArtifacts artifacts: 'results/*.xml', allowEmptyArchive: true
        }
    }
}
```

**Jenkins setup notes:**

- Install the **JUnit** plugin (usually pre-installed). The `junit` step in `post { always { ... } }` publishes results to the build's Test Results page regardless of whether the stage passed or failed.
- Store `api-base-url` and `api-token` as **Secret text** credentials in *Manage Jenkins → Credentials*. Reference them with `credentials('id')`.
- For multi-environment pipelines, parameterise the environment file: `parameters { choice(name: 'ENV', choices: ['ci', 'staging']) }` and pass `--env environments/${ENV}.yaml`.

---

## GitLab CI

```yaml
# .gitlab-ci.yml
stages:
  - build
  - test
  - quality

build-cli:
  stage: build
  script:
    - go build -o clrsig ./apps/cli
  artifacts:
    paths:
      - clrsig

api-scenarios:
  stage: test
  needs: [build-cli]
  script:
    - |
      ./clrsig run examples/scenarios/user-lifecycle.yaml \
        --env environments/ci.yaml \
        --format junit \
        --output results/scenario-results.xml
  artifacts:
    when: always
    reports:
      junit: results/scenario-results.xml
  variables:
    API_TOKEN: $API_TOKEN   # set in Settings → CI/CD → Variables

spec-conformance:
  stage: test
  needs: [build-cli]
  script:
    - |
      ./clrsig conform openapi.yaml \
        --base-url $API_BASE_URL \
        --format junit \
        --output results/conform-results.xml
  artifacts:
    when: always
    reports:
      junit: results/conform-results.xml

breaking-changes:
  stage: quality
  needs: [build-cli]
  only:
    - merge_requests
  script:
    - ./clrsig diff openapi-base.yaml openapi.yaml
  allow_failure: false
```

GitLab parses JUnit artifacts automatically and displays pass/fail counts on the pipeline page and merge request widget.

---

## CircleCI

```yaml
# .circleci/config.yml
version: 2.1

jobs:
  api-tests:
    docker:
      - image: cimg/go:1.23
    steps:
      - checkout

      - run:
          name: Build clrsig CLI
          command: go build -o clrsig ./apps/cli

      - run:
          name: Run API scenarios
          command: |
            mkdir -p results
            ./clrsig run examples/scenarios/user-lifecycle.yaml \
              --env environments/ci.yaml \
              --format junit \
              --output results/scenario-results.xml
          environment:
            API_TOKEN: $API_TOKEN

      - run:
          name: Spec conformance
          command: |
            ./clrsig conform openapi.yaml \
              --base-url $API_BASE_URL \
              --format junit \
              --output results/conform-results.xml

      - store_test_results:
          path: results

      - store_artifacts:
          path: results

workflows:
  api:
    jobs:
      - api-tests
```

The `store_test_results` step tells CircleCI to parse JUnit XML and display results in the **Tests** tab of each build.

---

## Secrets and Environment Variables

Clear Signal environments use `{{variableName}}` placeholders in request files. In CI, sensitive values are injected via environment files rather than hardcoded.

### CI environment file pattern

Create a `environments/ci.yaml` committed to the repo (without secrets):

```yaml
schemaVersion: "1"
name: CI
variables:
  - name: baseUrl
    value: ""          # overridden at runtime
  - name: token
    value: ""          # overridden at runtime
    secret: true
```

Then generate a populated file at runtime (never commit this):

```bash
# In your CI script, before running clrsig:
cat > environments/ci-runtime.yaml << EOF
schemaVersion: "1"
name: CI
variables:
  - name: baseUrl
    value: "${API_BASE_URL}"
  - name: token
    value: "${API_TOKEN}"
    secret: true
EOF

./clrsig run scenario.yaml --env environments/ci-runtime.yaml
```

**Security notes:**
- Clear Signal never writes `secret: true` variable values to request history or JUnit output — they are redacted at the executor layer.
- Do not commit environment files containing real values. The workspace `.gitignore` created by `clrsig new-workspace` already excludes `environments/`.
- Use your CI platform's native secret store (GitHub Secrets, Jenkins Credentials, GitLab CI/CD Variables, CircleCI Environment Variables) rather than `.env` files in the repo.

---

## Running the Example Server Locally

The repo ships with a demo server and scenario for end-to-end testing:

```bash
# Terminal 1: start the example server
cd examples/server && go run .

# Terminal 2: run the scenario against it
clrsig run examples/scenarios/user-lifecycle.yaml \
  --env examples/environments/local.yaml
```

This same pattern works in CI: start your application server as a service container (GitHub Actions `services:`, GitLab CI `services:`, CircleCI secondary Docker image), then run `clrsig conform` or `clrsig run` against `http://localhost:<port>`.
