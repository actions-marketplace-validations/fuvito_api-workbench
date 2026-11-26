# Clear Signal 0.13.5 (draft; delete after publishing)

## Security: `clrsig mcp` could change data when asked only to check an API. Please upgrade

**What happened.** Through `clrsig mcp`, which lets AI agents use Clear Signal, the `conform` tool sent **every operation of the spec: `POST`, `PUT`, `PATCH` and `DELETE` included.** It had no read-only argument, and it **ignored `readOnly: true`** in your project file, which the command line obeys. An agent asked to check an API could create, change or delete data on that server. Confirmed with released 0.13.4.

**If an agent used it against data you care about,** look at what that server received from `clrsig` during those checks.

## ⚠️ Changed: the `conform` tool is now read-only

By default it sends only `GET` and `HEAD`. The reply starts with `READ-ONLY`, and the other operations show as `SKIP`: **they were not checked.**

To let it send writes, **all** of these are needed:
1. you start the server with the new flag: `clrsig mcp --allow-writes` (in your agent's configuration: `"args": ["mcp", "--allow-writes"]`);
2. the call sets the new argument `allowWrites: true`;
3. no project file says `conform.readOnly: true`: neither the one `clrsig` finds from the folder the server runs in (or `--config`), nor the one beside the spec. `--no-config` ignores both.

A call that asks for writes without this is **refused with an explanation, and nothing is sent**. A project file that cannot be read also refuses writes. One switch was not enough: the agent writes the call, and only you write the server's configuration.

**If your agent set-up relied on writes,** add the flag, and have the agent ask for them.

**This does not limit the `run_scenario` tool.** It sends whatever the scenario says, `DELETE` included, and an agent that can write files can write a scenario. Give an agent only servers and credentials whose misuse you can afford.

The command line, the desktop app and the GitHub Action are not changed by this release.

## The detailed command-line guide now covers every command

[Start here](https://github.com/fuvito/api-workbench/blob/main/docs/guides/cli/getting-started.md), or see the [list of commands](https://github.com/fuvito/api-workbench/blob/main/docs/guides/cli/README.md). Its examples are run by the tests.

## Known problems found while writing it (not fixed yet)

**Check these if you rely on them:**
- **`coverage quality --fail-on-gaps` exits 0 when there is no data,** also for a misspelled workspace, so a CI gate can pass with nothing checked. Look for `No coverage data found` in the output.
- **`bundle run` sends the `CHANGE_ME` that `bundle export` puts in place of a secret,** and the run can pass. Give the real value with `--env-file` or `--var`.
- `bundle export` replaces only values **marked** as secrets: a token written directly in a request or header is exported as it is.
- **`import` leaves out Postman scripts without a warning:** the imported requests check nothing.
- **The telemetry notice (off by default) names less than is sent:** when on, a random identifier of the installation, the version, operating system, processor type and time are sent too.
- `coverage reset` reports success when there was nothing to reset.
- `validate` accepts an HTTP method that does not exist; `request` sends a `{{name}}` nothing defines as text and can exit 0; `validate` exits 1, not 2 as its help says, for a file it cannot read.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.13.4...v0.13.5
