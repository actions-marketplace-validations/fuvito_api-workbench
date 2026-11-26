# `clrsig config show`

[← Guide index](README.md)

## Purpose

`config show` answers one question: **if I run this command with no flags, what will it use, and where does each value come from?** It covers `conform`, `run` and `validate`. Use it when a command checks a different spec, server or environment than you expected. It sends nothing and changes nothing.

## Synopsis

```text
clrsig config show [conform|run|validate]
```

Without a command, all three are shown.

## Options

`config show` has no options of its own. Two of the [options every command has](conform.md#options) matter here: `--config` shows another project file, and `--no-config` shows what happens without one.

## Examples

### 1. What will `conform` use here?

<!-- run: exit=0 first="0" contains="conform.readOnly" -->
```bash
clrsig init my-check --spec examples/server/openapi.yaml --base-url http://localhost:8080
clrsig config show conform --config my-check/clrsig.yaml
```

**What you will see** (exit code 0). Paths are shown from the folder you are in:

<!-- output -->
```text
clrsig conform (with no flags or arguments):
  spec                        examples/server/openapi.yaml      file
  environment                 my-check/environments/local.yaml  file
  …
  conform.readOnly            true                              file
  conform.negative            false                             default
  …
A flag or argument on the command line replaces any value above.
```

### 2. Without the project file

<!-- run: exit=0 contains="The project file is ignored" -->
```bash
clrsig config show --no-config
```

## Reading the result

| Column | Meaning |
|---|---|
| setting | The name it has in `clrsig.yaml`. |
| value | What would be used. `(not set)` means nothing is. |
| `file` or `default` | `file`: your project file says so. `default`: built into `clrsig`. |

The order, from strongest: **a flag or argument** you type, then **the project file**, then **the built-in default**. `config show` shows the last two; it does not know what you will type.

Exit code **0**; **2** for a command it does not cover (`config show diff`) or a project file that cannot be read.

## What to do next

| Result | Next step |
|---|---|
| A value you did not expect, marked `file` | Edit `clrsig.yaml`, or give the flag for this run. |
| Your value is there but the command behaves differently | A flag in your command or script replaces it. |
| It reads a project file you did not mean | `clrsig` searches the current folder and those above it. Use `--config <file>` or `--no-config`. |

## Common mistakes and limits

- It shows settings, not the **values inside** the environment file. For those a check needs, use [`clrsig conform --list`](conform.md#1-see-what-a-run-will-need-before-sending-anything).
- It does not check that the spec or environment file named exists.

## Related commands and the desktop app

[`init`](init.md) creates the project file; its keys are listed in the [format spec](../../../apps/desktop/wails/help/format-spec.md). The desktop app's Check Conformance page fills its fields from the same file.
