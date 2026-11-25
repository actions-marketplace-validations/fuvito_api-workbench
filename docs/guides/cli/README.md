# Clear Signal: the detailed command-line guide

For every `clrsig` command: what it is for, every option, examples with what you will see, and what to do next. The desktop app is mentioned where it does the same thing. For a shorter tour of the app and the CLI together, see the [user guide](../../../apps/desktop/wails/help/user-guide.md).

**This guide is being written.** One page exists so far; the plan is in [the proposal](../../enhancements/2026-11-25-detailed-user-guide.md).

## Command pages

| Command | What it is for |
|---|---|
| [`clrsig conform`](conform.md) | Spec Conformance Testing: check a running API against its OpenAPI spec |

## Following the examples

The examples use the example server and files that come with the repository. From the repository root:

```bash
go run ./examples/server
```

It listens on `http://localhost:8080`. Leave it running in its own terminal and run the examples in another, from the repository root. A run that sends write requests changes the sample users; this puts them back:

```bash
curl -X POST http://localhost:8080/admin/reset
```

Examples on these pages are run by the repository's tests against that server, and the lines shown under **What you will see** are checked against the real output (times differ; `…` marks lines left out). An example marked *not run by a test* needs something the tests do not have.
