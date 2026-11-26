# Task: track what your tests have not reached

[← Guide index](README.md)

**Goal:** see which operations and error answers no scenario has ever produced.

## 1. Record coverage while you run scenarios

Give `run` the spec and a **workspace** folder. Coverage collects there, run after run.

<!-- run: exit=0 first="0" contains="of 8 operations" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage list --workspace my-ws
```

## 2. See the gaps

<!-- run: exit=0 first="0" contains="NEVER EXERCISED" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage quality examples/server/openapi.yaml --workspace my-ws
```

Gaps are only reported: the exit code is 0. Write scenarios for the ones that matter most, usually the error answers of operations that change data.

## 3. Use it as a gate, if you want one

<!-- run: exit=1 first="0" contains="NEVER EXERCISED" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml --workspace my-ws
clrsig coverage quality examples/server/openapi.yaml --workspace my-ws --fail-on-gaps
```

With `--fail-on-gaps`, the step also fails when **nothing** was recorded.

| Remember | |
|---|---|
| Write the spec the same way each time | Coverage is stored under the spec path or address you give. |
| A fresh CI machine starts empty | Run the scenarios and the gate in the same job, or keep the workspace folder between jobs. |
| Reached is not tested well | An operation counts as covered even when its step asserts little. |

Releases up to 0.13.5 let `--fail-on-gaps` pass with nothing recorded. Details: [`coverage`](coverage.md).
