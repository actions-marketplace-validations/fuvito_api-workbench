# `clrsig telemetry`

[← Guide index](README.md)

## Purpose

Telemetry is **off unless you turn it on**. When on, some commands send a small record of each run to the project, to learn how the tool is used. These three commands show and change that setting.

## What is sent when it is on

After `run`, `conform` and `bundle run`, a record is sent to `https://telemetry.clrsig.com/v1/event` with:

- the command name, whether it passed or failed, and how long it took;
- how many steps and assertions it had;
- **a random identifier for this installation** (created on first use and kept in your Clear Signal settings folder, so that runs from one installation can be told apart from another's);
- the `clrsig` version, your operating system and processor type, and the time.

It does **not** contain request addresses, headers, bodies, file names or values. As with any request, the server sees your IP address.

`telemetry status` (when on) and `telemetry enable` print this list.

A failure to send is ignored: it never changes a result or an exit code.

## Synopsis

```text
clrsig telemetry status
clrsig telemetry enable
clrsig telemetry disable
```

## Options: `clrsig telemetry status`

`status` has no options of its own.

## Options: `clrsig telemetry enable`

`enable` has no options of its own.

## Options: `clrsig telemetry disable`

`disable` has no options of its own.

## Examples

<!-- run: exit=0 first="0" contains="Telemetry: disabled" -->
```bash
clrsig telemetry status
```

<!-- run: exit=0 first="0" contains="Telemetry enabled." contains="Telemetry disabled." -->
```bash
clrsig telemetry enable
clrsig telemetry status
clrsig telemetry disable
```

## What to do next

| Result | Next step |
|---|---|
| `disabled` | Nothing: this is the default. |
| `enabled`, and you did not mean to | `clrsig telemetry disable`. |

The setting is stored for your user on this computer, so it applies to every project. The desktop app has the same switch in **Settings**.
