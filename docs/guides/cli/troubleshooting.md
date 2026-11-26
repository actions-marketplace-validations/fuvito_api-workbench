# Troubleshooting, by what you see

[← Guide index](README.md)

Find the message, or part of it, in the first column.

## Nothing was sent, or nothing was checked

| Message | Why | What to do |
|---|---|---|
| `not sent: no value for {{name}}` | Nothing gives `name` a value. | Add it to the environment file, or give `--env`. In a scenario, check that an earlier step extracts it, and the spelling. |
| `NOTHING CHECKED` … `Every operation was skipped` | The filters or `--read-only` left nothing. | Loosen `--tags` or `--paths`. This is **not** a pass. |
| `no operation matches --paths …` | `--paths` matches the **start** of a path as the spec writes it. | Compare with `conform <spec> --list`. |
| The same message, and the path shows `C:/Program Files/Git/…` | Git Bash on Windows rewrote your `/path`. | Set `MSYS_NO_PATHCONV=1`, or use PowerShell. |
| `NOTHING CHECKED  0 probes` | `--negative` had nothing it could probe. | The spec declares nothing to break for these operations. |
| `No coverage data found` | Nothing was recorded for this spec in this folder. | Run scenarios with **both** `--spec` and `--workspace`, and write the spec the same way. |

<!-- run: exit=2 contains="no operation matches --paths /nowhere" -->
```bash
clrsig conform examples/server/openapi.yaml --env examples/environments/demo-conform.yaml --paths /nowhere
```

## The server could not be reached

| Message | Why | What to do |
|---|---|---|
| `is the server running?` | No operation got an answer. | Read the `First error` line below it, then check the address and the port. |
| `connection refused` or `actively refused it` | Nothing listens at that address. | Start the server. Inside Docker, `localhost` is the container itself: use the host's address. |
| `no such host` | The name does not resolve. | Check the spelling, and that this machine can resolve it (VPN, CI network). |
| `x509` or `certificate` | The certificate is not trusted, has expired, or is for another name. | For a private authority, use `--ca-cert`. |
| `context deadline exceeded` | The answer took longer than the time limit. | Find out why it is slow; then raise the limit (see the command's page). |
| `unsupported protocol scheme ""` | The address has no `http://` or `https://`. Often a `${NAME}` that is not set. | Read the warnings above it. |

<!-- run: exit=1 contains="is the server running?" -->
```bash
clrsig conform examples/server/openapi.yaml --base-url http://localhost:1 --env examples/environments/demo-conform.yaml --read-only
```

## The server answered, and the check failed

| Message | Why | What to do |
|---|---|---|
| `401` or `403` on many operations | No credentials, or wrong ones. | Put the token in the environment, or log in with [`--setup`](../authed-conform.md). |
| `404` on operations with an id | The generated id names nothing. | Give a real one in the environment, or with `--var`. |
| A violation with an ID such as `CS-RESP-005` | The answer differs from the spec. | Run with `--explain`. Decide whether the **server** or the **spec** is wrong. Look the ID up in [what we check](../../../apps/desktop/wails/help/what-we-check.md). |
| A run that passed before fails after an upgrade | A newer release found something an older one missed. | Read the release notes before you change a baseline. |

## The command or a file was refused

| Message | Why | What to do |
|---|---|---|
| `unknown flag: …` | The option does not exist for this command. | `clrsig <command> --help`. Check your version: `clrsig --version`. |
| `needs a scenario file or folder` | No file was given, and the project file names none. | Give one, or set `run.scenarios` in `clrsig.yaml`. |
| `reading scenario: open …` | The path is wrong from the folder you are in. | Paths on the command line start at the current folder. |
| `schemaVersion "…" is not supported` | A newer or mistaken file version. | Upgrade `clrsig`, or correct the file. (Scenario, request and environment files are **not** checked for this yet: backlog B-150.) |
| `the bundle has several environments` | `bundle run` cannot choose. | Add `--env <name>`. |
| `still holds the CHANGE_ME` | An exported bundle has no value for that secret. | Give it with `--var` or `--env-file`. |
| `nothing was written: … already there` | `generate` will not replace edited files. | Use `--output` with a new folder; `--force` replaces them. |
| `already exists and was not changed` | `init github-actions --write` will not replace a workflow. | Choose another `--output`, or merge by hand. |
| `set both the certificate file and the key file` | Mutual TLS needs both. | Give `--client-cert` **and** `--client-key`. |
| `is not a folder: check the name given to --workspace` | The workspace does not exist. | Check the spelling; run scenarios with that `--workspace` first. |
| `REFUSED: writes are not allowed` (from an AI agent) | The [MCP server](mcp.md) is read-only unless you allow writes. | Read that page before you allow them. |

<!-- run: exit=2 contains="unknown flag: --scenarios" -->
```bash
clrsig run examples/scenarios/user-lifecycle.yaml --scenarios x
```

## Still stuck

Run the command again with `--explain` where it has it, and check `clrsig config show conform` (or `run`, `validate`) for a setting that comes from a project file. When you report a problem, give the `clrsig` version, the command and the message, **with tokens, passwords and personal data removed**.
