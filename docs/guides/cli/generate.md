# `clrsig generate`

[← Guide index](README.md)

## Purpose

`generate` reads an OpenAPI 3.0 or 3.1 spec and writes **one request file per operation**, as a starting point for exploring the API or writing scenarios. It sends nothing to your API.

You do not need it to check an API against its spec: [`conform`](conform.md) builds its requests itself.

> **⚠ It overwrites without asking.** A file in the output folder with the same name as a generated one is **replaced**, your edits included. Other files are left alone. Generate into a **new folder**, or commit your work first (backlog B-136).

## Synopsis

```text
clrsig generate <spec-file-or-url> [flags]
```

## Options

| Option | What it does | Default and limits |
|---|---|---|
| `--output`, `-o` | The folder for the files. | A folder named after the spec's title, in the current folder. |
| `--base-url`, `-b` | Use this server address in the files. | The spec's first `servers` entry. |
| `--dry-run` | Lists what would be generated and writes nothing. It does **not** say which files exist already. | |
| `--format`, `-f` | `text` or `json`: how the list is printed. The request files are always YAML. | `text` |

The [options every command has](conform.md#options) apply too.

## Examples

### 1. See what would be generated

<!-- run: exit=0 contains="dry run" contains="8 request(s) generated" -->
```bash
clrsig generate examples/server/openapi.yaml --dry-run
```

**What you will see** (exit code 0):

<!-- output -->
```text
  POST    http://localhost:8080/auth/login
  GET     http://localhost:8080/health
  …
  GET     http://localhost:8080/users/{{id}}
  …
8 request(s) generated
```

### 2. Write them into a new folder and use one

<!-- run: exit=0 first="0" contains="200 OK" -->
```bash
clrsig generate examples/server/openapi.yaml --output my-requests --base-url http://localhost:8080
clrsig request my-requests/getHealth.yaml
```

A file is named after its operation's `operationId`. This is `createUser.yaml`:

```yaml
schemaVersion: "1"
title: Create a new user
method: POST
url: http://localhost:8080/users
body:
    type: json
    content: |-
        {
          "name": "{{name}}",
          "email": "{{email}}"
        }
auth:
    type: bearer
    token: '{{token}}'
```

Give `name`, `email` and `token` in an environment file when you send it.

## Reading the result

| What is in a file | Where it comes from |
|---|---|
| `{{id}}` in the address | A path parameter. |
| `auth` with `{{token}}` (or the key or basic form) | The operation's security scheme. |
| A JSON body | The **required** fields, with the spec's examples or defaults where there are any, and `{{name}}` places otherwise. A body with no required field comes out as `{}`. |

Exit code **0** when the files were written; **2** when the spec could not be read or is not valid.

## What to do next

| Result | Next step |
|---|---|
| Files written | Fill an environment with the `{{…}}` names, then [`clrsig request`](request.md). |
| A body is `{}` but the operation takes fields | The spec marks none as required; add them yourself (backlog B-047). |
| The addresses point at the wrong server | Generate again into a new folder with `--base-url`. |
| The spec changed | Generate into a **new** folder and compare, so your edits are not lost. |

## Common mistakes and limits

- **Generating again over files you edited** replaces them.
- Swagger 2.0 files are refused.

## Related commands and the desktop app

[`request`](request.md), [`conform`](conform.md), `clrsig import` (from Postman or Insomnia). The desktop app imports a spec into a collection.
