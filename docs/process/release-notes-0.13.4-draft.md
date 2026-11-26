# Clear Signal 0.13.4 (draft; delete after publishing)

## Please upgrade: a `schema` check could pass without checking anything

A scenario's `schema` assertion **passed without reading the body** when it had no schema for the response:
- a **status it did not know**: with an `operationId`, one the operation does not declare; with a schema written in the scenario, anything but `200`;
- a status the spec declares only as a **range** (`2XX`, `4XX`): this check never used ranges;
- a response with **no schema**, or with no JSON type (XML, CSV).

With a `specFile`, the step's own spec check still failed an undeclared status. **Without one, a run that got a `500` could be reported as `PASSED`** (shown with released 0.13.3; the unexpected-status cases were also tested with 0.13.1 and 0.13.2). Part of this was documented in What We Check as not caught.

**Now the assertion passes only when the body was compared with a schema.** It uses the response declared for the status, its range or `default`, and a declared JSON schema even if the server sent another type. A schema written in the scenario is checked **for every status**. When there is nothing to compare with, the assertion fails and says why.

## ⚠️ Runs that passed may now fail

Look for `schema` assertions on:
- error or other statuses your spec does not declare;
- a schema written in the scenario on a step that answers something other than `200` (the body is now checked against it);
- `204` and other responses without a body or without a schema;
- XML, CSV or other non-JSON responses (use `header` or other assertions for these).

To see whether past runs hid a problem, run your scenarios again and read each new failure. `conform` and the automatic spec check of a step are not changed by this.

## `clrsig generate` no longer replaces your files

Up to 0.13.3, `generate` **overwrote files without asking**, your edits included. Now, if a file with other content is in the way, **nothing is written**, the files are listed and the exit code is 2. `--force` (new) replaces them. Unchanged files and other files are left alone, two requests with the same file name are both kept, and `--dry-run` marks the files in the way. A script that generates over old output needs `--force`. The desktop import never replaced files.

## Other fixes

- A negative-probe run that sends nothing says `NOTHING CHECKED` and why, not `PASSED  0 probes`. Exit codes are unchanged.
- OpenAPI 3.1 messages no longer repeat parts of a path (`/user/user/age`).
- **Correction to the 0.13.3 notes:** they said a single 3.1 problem no longer starts with `validation failed due to:`. In 0.13.3 it still did; from this release it does not. Baselines written by 0.13.3 still match.
- A JSON `run` report shows an empty `expected`, not `<nil>`, for a `schema` assertion.
- A YAML schema written in a scenario with nested keys can now be read; it used to fail as "invalid schema".
- The format guide describes both forms of a `schema` assertion.

## The detailed command-line guide

New pages: [your first check](https://github.com/fuvito/api-workbench/blob/main/docs/guides/cli/getting-started.md), `init`, `config show`, `run`, `validate`, `request` and `generate`, with examples that the tests run. More commands will follow.

## Known problems (not fixed yet)

- `validate` accepts an HTTP method that does not exist.
- `clrsig request` sends a `{{name}}` nothing defines as text, and can exit 0.
- `validate` exits 1, not 2 as its help says, for a file it cannot read.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.13.3...v0.13.4
