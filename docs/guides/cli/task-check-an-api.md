# Task: check an API against its spec

[← Guide index](README.md)

**Goal:** find out whether a running API does what its OpenAPI spec says (**API Contract Validation**), without changing its data until you decide to.

Start the [example server](README.md#following-the-examples) to follow along.

## 1. Check the spec first

A wrong spec gives wrong results, so look at it before you read results against it.

<!-- run: exit=0 contains="OK" -->
```bash
clrsig validate --spec examples/server/openapi.yaml
```

## 2. See what would be sent

`--list` sends nothing. It shows each operation, the values its request needs and the ones nothing provides.

<!-- run: exit=0 contains="Nothing was sent." -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --list --read-only
```

## 3. Run it read-only

`--read-only` sends only `GET` and `HEAD`. Use it for the first run against anything whose data matters.

<!-- run: exit=0 contains="4 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --read-only
```

The other operations show as `SKIP`: **they were not checked.**

## 4. Run everything

**This sends `POST`, `PUT`, `PATCH` and `DELETE`.** Run it only against a server whose data you can afford to change.

<!-- run: exit=0 contains="8 passed" -->
```bash
clrsig conform examples/server/openapi.yaml --base-url http://localhost:8080 --env examples/environments/demo-conform.yaml
```

## Then

| What you see | Next |
|---|---|
| `PASSED`, nothing skipped | Add [invalid-request probes](conform.md) with `--negative`, then [run it in CI](task-ci.md). |
| `FAIL` lines | Run again with `--explain`; each violation says what and where. |
| `not sent` | A value or a credential is missing: see [troubleshooting](troubleshooting.md). |
| Too many failures to fix now | [Adopt it with a baseline](task-adopt-with-a-baseline.md). |

A pass covers what was sent: one request per operation, built from the spec. Sequences, such as create then read, need [scenarios](task-test-a-workflow.md). Every option is on the [`conform` page](conform.md).
