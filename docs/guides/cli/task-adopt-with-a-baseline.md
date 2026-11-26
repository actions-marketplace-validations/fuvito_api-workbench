# Task: adopt the check on an API that fails today

[← Guide index](README.md)

**Goal:** stop **new** violations from today, without fixing every old one first.

A **baseline** is a file that records the violations you have now. With it, a run fails only for violations that are not in the file.

## 1. Read the violations

<!-- run: exit=1 contains="CS-RESP-005" -->
```bash
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --explain
```

Read them before you record them. **A baseline accepts a violation; it does not make it safe.** Fix first anything that leaks data or breaks a client now.

## 2. Record them, then run with the file

<!-- run: exit=0 contains="Baseline written: 2 violations" contains="PASSED with 2 known violations" -->
```bash
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --write-baseline baseline.yaml
clrsig conform docs/guides/cli/files/strict-openapi.yaml --env examples/environments/demo-conform.yaml --baseline baseline.yaml
```

The recording run prints `FAILED` above its last line and exits 0: recording accepts what it found.

## 3. Commit the file and use it in CI

Add `--baseline baseline.yaml` to the command in your pipeline. See [run it in CI](task-ci.md).

## 4. Shrink it over time

When a run reports violations as **fixed**, record the baseline again and commit it, so they cannot come back unnoticed.

| Rule | Why |
|---|---|
| Record with a full run | `--write-baseline` is refused with `--tags`, `--paths` or `--changed-against`. |
| Record again only after reading the new results | Re-recording to make a build green accepts the new problem for good. |
| Review the file's changes like code | A new line in it is a violation someone chose to accept. |

With an OpenAPI 3.0 spec, one problem is reported per response body, so fixing it can show the next one as new. Details: the [`conform` page](conform.md).
