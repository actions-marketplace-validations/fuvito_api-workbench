# Task: test a workflow with scenarios

[← Guide index](README.md)

**Goal:** check things one request cannot show: log in, create something, read it back, delete it, with your own checks.

A **scenario** is a file of steps. A step sends a request, **asserts** things about the answer, and can **extract** a value for later steps. The file format is in the [format spec](../../../apps/desktop/wails/help/format-spec.md).

> A scenario sends whatever it says, `DELETE` included. Read a scenario you did not write before you run it.

## 1. Check the files without sending anything

`--strict` also reports a step that checks nothing and an extraction no step uses.

<!-- run: exit=0 contains="OK" -->
```bash
clrsig validate examples/scenarios/user-lifecycle.yaml --strict --env examples/environments/local.yaml
```

## 2. Run it

<!-- run: exit=0 contains="PASSED" -->
```bash
clrsig run examples/scenarios/user-lifecycle.yaml --env examples/environments/local.yaml
```

## 3. Also check the answers against the spec

With `--spec`, a step that names an operation is checked against what the spec declares, as well as against your assertions.

<!-- run: exit=0 contains="PASSED" -->
```bash
clrsig run examples/scenarios/contract-inline-schema.yaml --env examples/environments/local.yaml --spec examples/server/openapi.yaml
```

## Writing your own

| Advice | Why |
|---|---|
| Give every step a `status` assertion | A step with no assertion and no extraction passes whenever the request is answered. |
| Extract ids and tokens; do not type them | A typed id works until the data changes. |
| Keep secrets in the environment file, as `${NAME}` | The scenario can then be committed. |
| Choose `failPolicy` on purpose | With `stop`, later steps are skipped after a failure, and the report says so. |
| Make each scenario clean up | A scenario that leaves data behind can fail the next run. |

## Then

[Run it in CI](task-ci.md), or [see what your runs have not reached](task-track-coverage.md). Every option is on the [`run` page](run.md) and the [`validate` page](validate.md).
