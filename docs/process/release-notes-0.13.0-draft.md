# Clear Signal 0.13.0 (draft; delete after publishing)

## Read this first: `conform` no longer sends empty path values or credentials

**A run that passed before can now fail.** A variable that is defined but empty (`value: ""`, or a `${NAME}` whose process variable is set to nothing, as a secret that is not configured often is) used to fill its placeholder with nothing, and the request was sent: `GET /users/{id}` went out as `GET /users/`. If the spec declared the `400`, `401` or `404` that came back, the operation showed as **passing** although it was never exercised.

`conform` and the Check Conformance page now treat an empty value as **missing** in:

- the URL path,
- a bearer token,
- an API key (in a header, the query or a cookie),
- a basic-auth user name.

The operation is reported as **not sent**, the message says which value is empty, the run exits `1`, and the operation's negative probes are skipped. Give the value, or leave the operation out with `--paths`, `--tags` or `--read-only`.

**Unchanged:** an empty basic-auth **password** is still sent (some APIs take a key as the user name and no password), as are empty query, header, cookie and body values. `clrsig run` is unchanged: a scenario may send an empty value on purpose. An empty value that is still sent can still lead to a misleading pass; What We Check records this as CS-GAP-013.

## New: see what a run will need

`clrsig conform openapi.yaml --list` shows every operation as `READY`, `MISS` or `SKIP`, with the values its request needs and which are missing. It sends nothing to the API (a spec given as a URL is still downloaded), prints no values, writes text or JSON, and exits `0` once the spec has been read. It uses the same rules as a run.

## `clrsig init --spec` tells you what to fill in

With a local spec file, the new environment files also list the values the spec's requests take, **as comments**, with the operations that use each. Remove the `#` from the ones you need. Credentials are marked `secret: true`, and in `environments/ci.yaml` suggest a process variable such as `${API_KEY}`. Environment files that already exist are not changed.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.12.0...v0.13.0
