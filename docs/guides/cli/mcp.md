# `clrsig mcp`

[← Guide index](README.md)

## Purpose

`mcp` lets an **AI agent** (such as Claude Desktop or Cursor) use Clear Signal. It speaks the Model Context Protocol on standard input and output. It opens no network port. You do not run it by hand: the agent's configuration starts it.

> ## ⚠ What an agent can change
> - **`conform` is read-only:** it sends only `GET` and `HEAD`. Writes need **both** your `--allow-writes` and the agent's `allowWrites: true`, and are refused when a project file says `readOnly: true`.
> - **`run_scenario` is not limited:** it sends whatever the scenario says, `DELETE` included, and an agent that can write files can write a scenario. Give an agent only servers and credentials whose misuse you can afford.
> - **Releases up to 0.13.4:** `conform` sent **every** operation, writes included, ignored a read-only project, and could not be limited. Upgrade, or use them only against data you can lose.

## Synopsis

```text
clrsig mcp [flags]
```

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--allow-writes` | Lets the `conform` tool send `POST`, `PUT`, `PATCH` and `DELETE` **when a call also asks for it**. | Off. A project file with `conform.readOnly: true` still forbids writes. |

The [options every command has](conform.md#options) apply too: certificates for a private CA or mutual TLS, and `--config` / `--no-config`.

### When `conform` sends writes

| Server started with `--allow-writes` | Call has `allowWrites: true` | Project says `readOnly: true` | What happens |
|---|---|---|---|
| no | no | any | Read-only: `GET` and `HEAD`. |
| no | yes | any | **Refused**, nothing is sent. |
| yes | no | any | Read-only. |
| yes | yes | yes | **Refused**, nothing is sent. |
| yes | yes | no, or no project file | Every operation is sent. |

The project files looked at are the one `clrsig` finds from the folder the server started in (or the one named by `--config`), **and** the one found from the folder of the spec. Either can forbid writes; one that cannot be read refuses them too. `--no-config` ignores both. A spec given as a URL has no folder.

A read-only reply starts with `READ-ONLY`, and the operations it did not send are shown as `SKIP`: **they were not checked.**

## Setting it up

<!-- not run: this is the agent's configuration, not a command -->
```json
{
  "mcpServers": {
    "clrsig": { "command": "clrsig", "args": ["mcp"] }
  }
}
```

Where that file is depends on the agent: see its documentation. Use the full path of `clrsig` if the agent does not find it. Relative file names the agent passes are read from the folder `mcp` was started in, so full paths are safest.

## The tools

| Tool | What it does | Arguments (**required**) |
|---|---|---|
| `conform` | Checks a running API against a spec. Read-only unless writes are allowed twice (above). | **`spec`**, `baseUrl`, `envFile`, `allowWrites`, `failOn`, `maxResponseTime`, `maxResponseTimeFor` |
| `run_scenario` | Runs one scenario, **writes included**. | **`scenario`**, `envFile`, `specFile` |
| `diff` | Compares two specs. Sends nothing. | **`baseSpec`**, **`revisionSpec`**, `breakingOnly` |
| `coverage` | Lists a workspace's coverage. Sends nothing. | **`workspace`** |

## Reading the result

The tools return text, the same kind of report the command line prints, which the agent reads. What the agent says about it is its own summary: for a decision, look at the report itself.

## What to do next

| Result | Next step |
|---|---|
| The agent does not list the tools | Check the configuration and the path to `clrsig`; restart the agent. |
| A tool says a file does not exist | Give full paths. |
| `writes are not allowed` | Decide whether the agent should change that server's data. If so, add `--allow-writes` to the server's `args`, and check no project file says `readOnly: true`. |
| Operations show `SKIP` and the reply says `READ-ONLY` | They were not checked. Allow writes only against data you can lose. |

## Common mistakes and limits

- The agent can read files you name to it, and its provider receives what the tools return. Reports quote little of a response, but not nothing (see the [`conform` warning](conform.md#purpose)).
- A project file can **forbid** writes of `conform`; none of its other settings apply to `mcp`, and it does not limit `run_scenario`.

## Related commands

[`conform`](conform.md), [`run`](run.md), [`diff`](diff.md), [`coverage`](coverage.md).
