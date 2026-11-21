# api-workbench-sandbox project notes
# Test-target APIs — a separate project that Clear Signal is tested against


**Status:** Draft v0.2. This is a working document; we will keep editing it. Open questions are in section 9.
**Written:** 2026-11-20. **v0.2:** runs locally first, then on Google Cloud Run (section 10); adds latency, timeout and light business-rule cases (sections 3.4, 3.5, 5.4, 5.9).
**Lives in:** `docs/planning/`. The new project lives in its **own repository**, managed from here.

---

## 1. Purpose (read this first)

Build one small, separate project that serves **several OpenAPI-described APIs from one running program**, so that every part of Clear Signal's **API Contract Validation** and **Spec Conformance Testing** can be exercised against a real server, repeatedly, by hand and in CI.

It serves three families of APIs:

| Family | Question it answers | Expected Clear Signal result |
|---|---|---|
| **A. `ok`** (the success API) | Does Clear Signal stay quiet, and handle a big, realistic API, when everything is right? | Everything passes. Exit code 0. No false failures. |
| **B. `contract-fail`** (the response-failure API) | Does API Contract Validation catch every wrong **response**? | Each broken operation fails with the expected check ID (`CS-RESP-*`). |
| **C. `conform-fail`** (the conformance-failure API) | Does Spec Conformance Testing catch everything about the **whole API and the run** (requests, probes, missing operations, spec changes)? | Each broken behaviour fails with the expected check ID (`CS-REQ-*`, `CS-RUN-*`, `CS-PROBE-*`, `CS-DIFF-*`). |

The catalog of what Clear Signal reports is `apps/desktop/wails/help/what-we-check.md` in the api-workbench repository (check IDs such as `CS-RESP-001`). This project is the **executable proof** of that catalog: every row that says *caught* should have an operation here that triggers it, and every "known gap" row should have one that shows the gap.

### Non-goals

- No real business logic. Controllers do create, read, update, delete and list on in-memory maps, plus a **small set of simple rules** (stock, order states, uniqueness, expiry; section 3.4) so that realistic 409 and 422 answers and multi-step scenarios exist. Rules stay a few lines each.
- No database, no authentication provider, no external services, no migrations.
- Not a mock server product. It is a test fixture.
- Not a replacement for `docs/planning/failure-showcase-repo.md` (see section 11).

---

## 2. Shape of the project

One repository, **one program, no monorepo** (no workspaces, no multiple modules or packages to version separately).

```
clrsig-test-targets/
├── README.md
├── specs/
│   ├── ok.yaml                  # OpenAPI 3.0/3.1 for family A
│   ├── contract-fail.yaml       # family B
│   ├── conform-fail.yaml        # family C
│   └── diff/                    # pairs of specs for `clrsig diff` (section 8)
│       ├── <case>/v1.yaml
│       └── <case>/v2.yaml
├── src/                         # the server (one program)
├── clrsig/                      # scenarios + environments that drive Clear Signal at it
│   ├── environments/
│   └── scenarios/
└── .github/workflows/ci.yml
```

- **Language:** not decided (section 9). Recommendation: Go with the standard library only, one binary, matches the rest of the family. Anything with a single-command start works.
- **One process, one port, three APIs under path prefixes**, each serving its own spec at `<prefix>/openapi.yaml`. One port is required because Cloud Run exposes a single port per service (section 10); using it locally too keeps both identical. The port comes from the `PORT` environment variable (default `8080`).

| API | Prefix | Spec |
|---|---|---|
| `ok` | `/ok` | `specs/ok.yaml` |
| `contract-fail` | `/contract-fail` | `specs/contract-fail.yaml` |
| `conform-fail` | `/conform-fail` | `specs/conform-fail.yaml` |

- Each spec's `servers:` entry is a relative path (`/ok`) or is rewritten at startup from `PUBLIC_BASE_URL`, so the served spec always matches where it runs. Paths elsewhere in this document (for example `GET /cases/...`) are relative to the prefix.

- Every API exposes `GET /health` (200, any body) for readiness polling.
- **Specs are hand-written and are the source of truth.** The server is written to match `ok.yaml` exactly, and to deviate from the other two on purpose, in documented ways.

### In-memory data and reset (all APIs)

- Data lives in process memory. It survives between requests until reset or restart.
- Every API has `POST /__admin/reset`. It restores the seed data and returns `204`. Deliberately **not** in the specs, so Clear Signal never calls it as an operation. Scenarios call it explicitly as their first step.
- Seed data is small and fixed (a few records per resource) so assertions can name real ids.
- Reset is the only admin endpoint. Optionally `GET /__admin/state` (dump of counts) for debugging.
- Binding: `127.0.0.1` when run locally by default; `0.0.0.0` when `HOST=0.0.0.0` is set (the container image sets it). When deployed, reset needs a shared secret (section 10).
- Each API has its own store, so resetting one does not touch the others.

---

## 3. Family A: `ok` (everything correct)

Goal: a large, realistic-looking API where **every operation conforms to the spec**, so a full `clrsig conform` run and a full scenario run give zero violations. It is also the stress test for scale: **many endpoints (target 80 to 120 operations)**, many resources, many schema features.

### 3.1 Resources (suggested, each with full CRUD and list)

Customers, Addresses, Products, Categories, Inventory, Orders, Order Items, Payments, Invoices, Shipments, Reviews, Tags, Users, Teams, Projects, Tasks, Comments, Attachments (metadata only), Notifications, Webhooks (registrations only, nothing is delivered), Settings, Audit Events (read-only), Feature Flags.

Names are examples; the only requirement is variety, not domain sense.

### 3.2 What the spec must exercise (so the checker is exercised)

| Area | Include |
|---|---|
| Methods | GET, POST, PUT, PATCH, DELETE, HEAD, OPTIONS |
| Parameters | path (int, string, uuid), query (int, enum, array, boolean, date-time), header, cookie, optional and required, defaults |
| Request bodies | JSON objects, nested objects, arrays, `required`, `enum`, `minimum`/`maximum`, `minLength`/`maxLength`, `pattern`, `format` (date-time, email, uuid, date) |
| Responses | `200`, `201`, `202`, `204`, `400`, `401`, `403`, `404`, `409`, `422`, `429`, `default`, status ranges (`2XX`) |
| Response headers | required and optional (`X-Total-Count`, `Location`, `ETag`, `X-Request-Id`, rate-limit headers) |
| Schemas | `$ref` and shared components, `allOf`, `oneOf` with discriminator, `anyOf`, `nullable` (3.0) and type arrays (3.1), `additionalProperties`, `readOnly`, `writeOnly`, arrays with `minItems`/`uniqueItems`, maps |
| Media types | `application/json`, `application/problem+json`, `text/plain` |
| Security | `bearer`, `basic`, `apiKey` (header and query); some operations public, some secured. Valid credentials are fixed, documented values (for example bearer token `test-token`) |
| Structure | tags, operation ids, deprecated operations, pagination (offset and cursor), filtering, sorting, bulk operations, sub-resources (`/orders/{id}/items/{itemId}`), actions (`POST /orders/{id}/cancel`), idempotency key header, optimistic concurrency (`ETag` / `If-Match`) |

### 3.3 Behaviour rules

- A valid request always gets a response that matches the spec (status, headers, body, content type).
- Errors use the spec's declared error shape and status (`404` for unknown id, `409` for duplicate unique value, `422` for a business-shaped validation failure, `400` for malformed input, `401` without credentials on secured operations).
- It **must reject** bad input (this is what the negative probes test): missing required field, wrong field type, missing required query parameter and missing credentials all return 4xx. `ok` is the strict server.
- `ok` is the baseline for every claim: if a Clear Signal run against `ok` reports anything, that is a Clear Signal bug (a false failure) or a bug in `ok`.

### 3.4 Light business rules (only these; a few lines each)

They exist so that realistic error answers and multi-step scenarios work, and so Clear Signal's variable extraction and status checks are used on state, not just shapes.

| Rule | Behaviour |
|---|---|
| Unique values | Duplicate customer email, product SKU or tag name: `409` |
| Stock | Creating an order line that exceeds inventory: `422` with the error shape; paying or shipping decrements; cancelling restores |
| Order states | `pending -> paid -> shipped -> delivered`, and `pending/paid -> cancelled`. An illegal move (cancel a shipped order, pay twice): `409` |
| Computed fields | Order total = sum of lines, tax at a fixed rate; invoice number sequential; `createdAt` and `updatedAt` set by the server |
| Ownership | A task belongs to a project; deleting a project with open tasks: `409`; with `?cascade=true`: `204` |
| Expiry | A discount code or reset token has `expiresAt`; use after it: `410` |
| Concurrency | `ETag` / `If-Match` on orders and tasks; stale tag: `412`; missing header where required: `428` |
| Idempotency | Same `Idempotency-Key` and body returns the first result (`200`, not a second record); same key with a different body: `422` |
| Async job | `POST /reports` returns `202` with a `Location`; `GET /reports/{id}` is `202` (running) for N seconds or N polls, then `200` (done) |
| Rate limit | Per client, K calls per window: then `429` with `Retry-After`; declared in the spec |
| Soft delete | `DELETE` then `GET` is `404`; `?includeDeleted=true` still shows it |

### 3.5 Latency and timeouts on `ok`

Everything above must also work when the server is **slow but within bounds**. `ok` has a few operations that are correct and deliberately take time, so Clear Signal's timeout handling is exercised without any failure:

| Operation | Behaviour |
|---|---|
| `GET /ok-slow/{ms}` | Waits `ms` (capped at 10000), then answers `200` per spec |
| `GET /reports/{id}` | As in 3.4, polling takes seconds |
| any list operation with `?delay=<ms>` | Optional, capped; absent means no delay (the spec declares it as an optional query parameter) |

They must pass with the default Clear Signal timeout, and give CS-RUN-003 only when a scenario sets a shorter timeout than the delay.

### 3.6 Acceptance

- `clrsig conform specs/ok.yaml --negative` exits 0 and covers every operation.
- A shipped scenario chains create, read, update, list and delete across at least five resources, with variable extraction, and passes.
- Running the same scenarios twice without reset works (create steps use unique values or the scenario resets first).
- Running after reset gives identical results to a fresh start.

---

## 4. Family B: `contract-fail` (response failures)

Goal: **API Contract Validation** failures. The server's answer differs from what the spec declares for it. One operation per failure, named after the check it must trigger, so a report line maps straight to an operation.

Convention: `GET /cases/<check-id-lowercase>-<slug>` (for example `GET /cases/resp-001-undeclared-status`). Each operation has a spec entry that **declares the correct behaviour** and a handler that **breaks exactly one thing**. Where a case needs a request body or an id, it uses a plain `POST`/`GET {id}` with the id fixed to a seed record.

### 4.1 Cases (one per row; each must fail on exactly the stated check)

| Case operation | Spec declares | Server does | Expected check |
|---|---|---|---|
| `resp-001-undeclared-status` | `200`, `404` | returns `500` | CS-RESP-001 |
| `resp-001b-default-response-ok` | `200` and `default` | returns `503` | **passes** (a `default` response counts as declared) |
| `resp-002-wrong-content-type` | `application/json` | `text/plain` with a JSON body | CS-RESP-002 |
| `resp-003-missing-required-header` | required `X-Total-Count` | omits it | CS-RESP-003 |
| `resp-004-wrong-header-type` | `X-Total-Count` integer | `X-Total-Count: many` | CS-RESP-004 |
| `resp-005-missing-required-field` | `id` required | omits `id` | CS-RESP-005 |
| `resp-006-wrong-field-type` | `price` number | `"price": "9.99"` | CS-RESP-006 |
| `resp-006b-array-for-object` | object | returns an array | CS-RESP-006 |
| `resp-007-enum` | enum `[a,b]` | `"c"` | CS-RESP-007 |
| `resp-007b-minimum` | `minimum: 0` | `-1` | CS-RESP-007 |
| `resp-007c-pattern`, `-minlength`, `-maxlength`, `-minitems`, `-uniqueitems` | each constraint | violates it | CS-RESP-007 |
| `resp-007d-null-not-nullable` | not nullable | `null` | CS-RESP-007 |
| `resp-007e-bad-datetime` | `format: date-time` | `"yesterday"` | CS-RESP-007 |
| `resp-008-invalid-json` | JSON body | `{not json` | CS-RESP-008 |
| `resp-008b-empty-body` | `200` with schema | empty body | CS-RESP-008 |
| `resp-009-oneof-no-match` | `oneOf` A or B | matches neither | CS-RESP-009 |
| `resp-010-extra-property-closed` | `additionalProperties: false` | extra property | CS-RESP-010 |
| `resp-010b-extra-property-open` | open schema | extra property | **passes** (OpenAPI default) |
| `resp-011-writeonly-leak` | `password` is `writeOnly` | returns `password` | CS-RESP-011 |
| `resp-012-xml-valid` | `application/xml` | valid XML | CS-RESP-012 (a false failure today; pins the known gap) |
| `resp-nested-wrong-type` | nested object and array items | wrong type deep inside | CS-RESP-006 (checks the location is reported) |
| `resp-poly-discriminator-wrong` | discriminator `type` | body shape does not match `type` | CS-RESP-009 |
| `resp-multi-problem` | header and body both wrong | both | CS-RESP-003 or -004 first (only the first problem is reported: CS-GAP-011) |

Known gaps (the catalog says we do not catch them; these show it so the day we do, a test flips):

| Case operation | Server does | Catalog row |
|---|---|---|
| `gap-format-email-uuid` | `"mail": "nope"`, bad uuid | CS-GAP-001 |
| `gap-undeclared-header` | sends a header the spec never mentions | CS-GAP-003 |
| `gap-slow-response` | answers after 5 seconds | CS-GAP-005 |

"Not checked" outcomes (must **not** fail, must show as not checked):

| Case operation | Why |
|---|---|
| `nc-204-with-body` | spec declares `204` without content, server sends a body (CS-NC-002) |
| `nc-octet-stream` | binary media type (CS-NC-003) |
| `nc-no-content-type` | body with no `Content-Type` is read as JSON (CS-NC-008) |
| a path **not in the spec** | CS-NC-001 (scenario step only) |

### 4.2 Also needed

- A scenario file per case (or one scenario with a step per case) that calls it with `specFile: ../specs/contract-fail.yaml` and expects the failure. Use the **step spec check** and, for a few cases, the **`schema` assertion** as well. The two checkers behave differently (see the catalog, columns A and B), and each difference should have a case.
- A one-line expected-result table in the README: operation, check id, expected exit code.

### 4.3 Acceptance

- `clrsig conform specs/contract-fail.yaml` exits 1.
- Every case above produces exactly its expected check id (the `passes` rows produce none).
- A CI step compares actual to expected and fails the build if a case stops failing the way it should.

---

## 5. Family C: `conform-fail` (conformance failures)

Goal: **Spec Conformance Testing** failures: everything that is about the whole API, the requests Clear Signal builds, the run itself, and spec-against-spec changes. Unlike family B, here the **response** is often fine; what is wrong is the server's behaviour as a whole or the spec itself.

### 5.1 Server is too lenient (negative probes, `clrsig conform --negative`)

The server accepts bad requests when it should answer 4xx.

| Operation | Spec requires | Server does | Expected check |
|---|---|---|---|
| `POST /lenient/missing-field` | a required body field | accepts a body without it (`201`) | CS-PROBE-001 |
| `POST /lenient/wrong-type` | `quantity` integer | accepts `"lots"` | CS-PROBE-002 |
| `GET /lenient/missing-query` | required query `q` | answers `200` without it | CS-PROBE-003 |
| `GET /lenient/no-auth` | bearer security | answers `200` with no credentials | CS-PROBE-004 |
| `GET /lenient/wrong-auth` | bearer security | accepts any token | not probed today (CS-GAP-008) |
| `GET /lenient/enum-and-boundary` | enum and `maximum` | accepts out-of-range values | not probed today (CS-GAP-009) |
| `POST /lenient/unknown-fields` | `additionalProperties: false` | accepts unknown fields | not probed today (CS-GAP-009) |
| `GET /strict/...` | same shapes as above | correctly rejects all | **passes** (control group) |

### 5.2 Spec and server disagree about what exists

| Case | Server does | Expected |
|---|---|---|
| `GET /ghost` | spec declares it; server answers `404` (not declared for it) | CS-RESP-001 and a conformance failure |
| `DELETE /orphan` | spec declares the operation; server has no route (`404` / `405`) | CS-RESP-001 |
| `GET /undocumented` | server has it, spec does not | not exercised by `conform`; a scenario step on it is CS-NC-001 (not checked) |
| `PATCH /wrong-method` | spec says `PATCH`; server only accepts `PUT` (`405`) | CS-RESP-001 |
| `GET /path-param/{id}` | spec says `id` integer; server accepts only uuid and answers `400` for the generated value | CS-RESP-001 |
| `POST /requires-extra-header` | server needs a header the spec does not declare and answers `400` without it | CS-RESP-001 |
| `POST /form` and `POST /multipart` | spec declares a form / multipart body | pins CS-GAP-006 (generated **without** a body) |
| webhook / callback in the spec | not under `paths` | pins CS-GAP-010 |

### 5.3 Requests Clear Signal builds do not fit

These are failures the **spec plus environment** cause before any response matters. Driven through the environment file, so ship a deliberately wrong environment next to the right one.

| Case | How | Expected check |
|---|---|---|
| Wrong parameter type | environment gives `id=abc` for an integer path parameter | CS-REQ-001 |
| Wrong body value | environment gives `price="cheap"` for a number | CS-REQ-002 |
| Missing placeholder | environment omits a value used by the spec's `{{name}}` | CS-RUN-001 (not sent) |
| Bad dynamic function | scenario uses `{{$noSuchFunction}}` | CS-RUN-002 |
| Server down | wrong base URL, or the server stopped | CS-RUN-003 |
| Unreadable spec | wrong path or invalid YAML | CS-RUN-004 (exit code 2) |
| Unset `${NAME}` | environment refers to a process variable that is not set | CS-RUN-005 (warning, exit code unchanged) |

### 5.4 Timing and instability

| Operation | Server does | Why |
|---|---|---|
| `GET /slow` | waits longer than the client timeout | CS-RUN-003 (timeout) |
| `GET /flaky` | fails every Nth call with `500`, otherwise correct | shows a run that passes sometimes (baseline and retry behaviour) |
| `GET /rate-limited` | `429` after K calls, undeclared in spec | CS-RESP-001 (undeclared `429`) |
| `GET /latency/{ms}` | answers correctly after exactly `ms` (cap 120000) | nothing fails unless the timeout is shorter; response time is not measured today (CS-GAP-005) |
| `GET /latency/jitter?min=&max=` | random delay in the range | shows timing noise across repeated runs |
| `GET /latency/degrading` | each call is slower than the last, reset restores it | shows a run that gets slower |
| `GET /timeout/hang` | accepts the request and never answers | CS-RUN-003 (timeout) |
| `GET /timeout/slow-headers` | waits before sending headers | CS-RUN-003 when shorter than the timeout |
| `GET /timeout/slow-body` | sends headers at once, then trickles the body byte by byte | timeout while reading the body |
| `GET /timeout/truncated-body` | declares `Content-Length: 1000`, sends 100 bytes, closes | CS-RUN-003 or CS-RESP-008 (record which) |
| `GET /timeout/connection-reset` | closes the connection with no answer | CS-RUN-003 |
| `GET /timeout/after-write` | does the write, then waits past the timeout | a timed-out `POST` that **did** take effect; a retry creates a duplicate |
| `GET /unavailable` | `503` with `Retry-After`, undeclared in spec | CS-RESP-001 |
| `GET /unavailable-then-ok` | `503` the first N calls after reset, then correct | retry behaviour |
| `GET /redirect-loop` | redirects to itself | CS-RUN-003 (too many redirects) |
| `GET /large` | returns a body close to, and a body over, the client's size cap (50 MB today) | cap behaviour |
| `POST /idempotent` | duplicate `Idempotency-Key` creates a second record | only visible in a scenario with two steps |
| `PUT /lost-update` | ignores `If-Match` | only visible in a scenario |

### 5.5 Baseline ratchet (`--baseline`)

Keep the fixture simple: no mode-switching endpoint. The drift level is set at startup by the `--drift=<n>` flag or the `DRIFT` environment variable (the only way to change it on Cloud Run is a new revision; section 10):

- `--drift=0` behaves as `conform-fail` above.
- `--drift=1` additionally breaks **one new** operation.

Procedure: record a baseline at drift 0, run at drift 1, expect CS-RUN-006 for the new violation only.

### 5.6 Spec against spec (`clrsig diff`, `clrsig impact`)

No server needed. Pairs in `specs/diff/<case>/v1.yaml` and `v2.yaml`:

| Case | Change | Expected level |
|---|---|---|
| `removed-endpoint` | operation removed with no deprecation | ERR (CS-DIFF-001) |
| `renamed-field` | response field renamed | ERR |
| `new-required-request-field` | a request field becomes required | ERR |
| `type-narrowed` | response type changed | ERR |
| `enum-value-removed` | enum value removed | ERR |
| `optional-to-required-param` | parameter becomes required | ERR |
| `deprecated-operation` | `deprecated: true` added | WARN (CS-DIFF-002) |
| `endpoint-added` | new operation | INFO (CS-DIFF-003) |
| `description-changed` | text only | none or INFO |
| `impact-hit` | breaking change on an operation a scenario step calls | CS-IMP-001 |
| `impact-placeholder-path` | the same, but the step URL has `{{id}}` in the path | CS-IMP-001 (matched since 0.6.0) |

Each case has `--fail-on breaking`, `warning` and `info` expectations.

### 5.7 Coverage signals

Not failures; they must show on the Contracts page:

- one operation never exercised (CS-COV-001);
- one exercised only with `2xx` (CS-COV-002);
- one whose declared `4xx`/`5xx` are never triggered (CS-COV-003);
- one secured operation only ever called without credentials (CS-COV-004).

A scenario set that intentionally leaves these gaps ships with the project.

### 5.8 Acceptance

- Each row above has an operation, a spec entry, an environment or a pair of specs, and a stated expectation.
- A script runs every expectation and compares it to the actual Clear Signal result.

---

### 5.9 Business-rule failures (family C)

The rules of section 3.4, broken in a way the spec does not describe, mostly visible in scenarios:

| Case | Server does | Visible as |
|---|---|---|
| `POST /orders` over stock | accepts and goes negative | scenario assertion on stock fails; spec check passes |
| illegal state move | cancel after shipped returns `200` | scenario assertion on `409` fails |
| duplicate email | accepts the second | scenario assertion on `409` fails |
| wrong error status | `422` rule failure answered `200` with an error body | CS-RESP-001 if `200` is declared with another shape, else a scenario assertion |
| stale `If-Match` | ignored (`PUT /lost-update`, 5.4) | scenario assertion |
| unstable pagination | a record is added between pages so one repeats | scenario assertion |
| expired token | still accepted | scenario assertion |

These show what contract and conformance checks alone do **not** prove, and that scenario assertions are needed.

## 6. How Clear Signal drives it

The project ships ready-to-run commands (the Clear Signal CLI is `clrsig`):

```
# start everything
<run the server>                    # prints the three base URLs

# A: must be clean
clrsig conform specs/ok.yaml --env clrsig/environments/ok.yaml --negative
clrsig run clrsig/scenarios/ok-crud.yaml --env clrsig/environments/ok.yaml

# B: must fail, per case
clrsig conform specs/contract-fail.yaml --env clrsig/environments/contract-fail.yaml --explain

# C: must fail, per case
clrsig conform specs/conform-fail.yaml --env clrsig/environments/conform-fail.yaml --negative --explain
clrsig diff specs/diff/removed-endpoint/v1.yaml specs/diff/removed-endpoint/v2.yaml --fail-on breaking
```

Check exact flags with `clrsig --help`; the flag names above come from the catalog and may differ in detail. Scenarios start with a step `POST {{baseUrl}}/__admin/reset` so every run begins from the seed data.

Exit codes: 0 all pass, 1 violations, 2 execution or parse error.

---

## 7. CI

- Build the server, start it, wait for `/health`, run every command in section 6, and **compare to an expectations file** (`expected.yaml`: command, expected exit code, expected check ids).
- Family A expects exit 0. Families B and C expect exit 1 with the listed ids. A case that stops failing is a regression in Clear Signal and must turn CI red.
- Two legs, as in the failure-showcase design: against the released `clrsig`, and nightly against `clrsig` built from `api-workbench@main`.

---

## 8. Build order

1. Skeleton: one program, one port (`PORT`), three path prefixes, `/health`, `/openapi.yaml`, `POST /__admin/reset`, in-memory store helper (a generic map with create, get, update, delete, list).
2. `ok` with 3 resources and its spec. Get a clean `conform` run.
3. Grow `ok` to the full breadth (section 3) and the scenarios.
4. `contract-fail`: first the CS-RESP-001 to 006 cases, then the rest.
5. `conform-fail`: probes (5.1), then 5.2, 5.3, then diff pairs (5.6).
6. Expectations file and CI.
7. Latency, timeout and business-rule cases (3.4, 3.5, 5.4, 5.9); can start earlier for the cases that need no new resources.
8. Container image and Cloud Run deployment (section 10). Everything before this runs locally.

Slice rule: after step 2 there must already be a working end-to-end run; do not write all the specs first.

---

## 9. Open questions (to settle in the next session)

1. **Language and framework** for the server (recommendation: Go, standard library only).
2. **Repository name and owner.** (Candidates: `clrsig-test-targets`, `clrsig-fixtures`.)
3. ~~One port or three~~ **Settled in v0.2:** one port, path prefixes (Cloud Run allows one port per service).
4. **Authentication values.** Fixed bearer token, basic user, and API key for family A and the secured operations in C.
5. **Specs: OpenAPI 3.0, 3.1, or both?** Suggest `ok` in both versions (`ok-3.0.yaml`, `ok-3.1.yaml`), failures in 3.0 first.
6. **Size target for `ok`.** 80 to 120 operations proposed; confirm.
7. **Should `contract-fail` and `conform-fail` share one spec file?** This draft keeps them separate, so a report from one never mixes the two kinds of failure.
8. **Where to keep the expected results** so that the catalog and this project stay in step (a test in api-workbench that reads this project's `expected.yaml` is a possible later step, tied to backlog item B-049).
9. **Public or private on Cloud Run?** Public is simplest for anyone to try Clear Signal against, but then it is an open, deliberately broken service (see section 10 for what that costs). Private needs identity tokens, which Clear Signal can send as a bearer header.
10. **Cloud project, region and a custom domain.** Not decided.
11. **Tail cases under Cloud Run limits.** Cases that hang or take minutes (`/timeout/hang`, `/latency/{ms}` above the request timeout) are cut off by Cloud Run's own request timeout. Accept that, or keep those cases local-only (section 10).

---

## 10. Running locally now, Google Cloud Run later

Local first, but write the program so the move to Cloud Run needs a container and a deploy command, nothing else.

### 10.1 Design rules that make it portable (apply from day one)

- **Configuration only through environment variables:** `PORT` (Cloud Run sets it; default 8080), `HOST`, `PUBLIC_BASE_URL`, `DRIFT`, `ADMIN_TOKEN`, optional `LATENCY_SCALE` (multiplies every deliberate delay, so a local run can be fast and a deployed one realistic).
- **One port, path prefixes** (section 2).
- **Log to stdout** as one line per request (method, path, status, duration, API prefix). Cloud Run collects it.
- **Graceful shutdown** on `SIGTERM` (finish in-flight requests within the grace period).
- **No files written at runtime.** The container filesystem is in memory and disposable. Specs are embedded in the binary or read-only files in the image.
- **Fast start.** A static binary in a small image (distroless or scratch) so cold starts are short.
- Container image and a `Dockerfile` live in the repository from the first slice that runs, even before deploying.

### 10.2 In-memory state on Cloud Run: what changes

In-memory data is **per container instance**, and Cloud Run may start, stop or add instances at any time. So:

| Concern | Rule |
|---|---|
| Scale to zero | The instance disappears when idle and data is lost. Acceptable by design: scenarios begin with reset. Set `min-instances=1` if data must survive between runs. |
| More than one instance | Two instances would hold different data and break any multi-step scenario. Deploy with **`max-instances=1`**. |
| Concurrency | One instance serves many requests at once. Stores must be safe for concurrent use (a mutex per store). |
| CPU | Use "CPU always allocated" if background work is needed (async jobs of section 3.4 are computed from timestamps on request instead, so no background CPU is needed). Prefer this. |
| Timers and counters | Rate-limit windows, flaky counters, drift and "degrading" latency live in memory too and reset with the store. |
| Several users at once | Two people running scenarios at the same time share one store and disturb each other. Say so in the README; optional later: a per-client namespace taken from a header (`X-Test-Tenant`), defaulting to a shared one. Not in the first version. |

### 10.3 Timeouts and deliberately slow cases in the cloud

- Cloud Run has a **request timeout** (default 300 seconds, maximum 3600). Set it above the longest deliberate delay, or let those cases hit it.
- The platform's front end may close idle or very slow connections before the program does. Results of `/timeout/*` cases can differ from local runs; record the cloud result separately in `expected.yaml` (a `cloud:` override per case).
- The truncated body, reset and slow-header cases (5.4) depend on connection behaviour that the Google front end can change (it may buffer, retry or replace the answer with its own `502`/`504`). Treat these as **local-only** unless a cloud run proves they behave the same, and mark them `local-only: true` in `expected.yaml`.
- A cold start adds seconds to the first request. Warm the service first (`GET /health`), or the first scenario step looks like a latency failure.
- Keep deliberate delays bounded and cheap: a hanging request keeps an instance busy and is billed.

### 10.4 Security once it is public

A deployed instance is reachable from the internet and is intentionally misbehaving. Therefore:

- **No real data, no real secrets.** The only credentials are documented test values.
- `POST /__admin/reset` and `GET /__admin/state` require `Authorization: Bearer <ADMIN_TOKEN>` when `ADMIN_TOKEN` is set (always set it when deployed; locally optional). Scenarios send it from the environment file, as a secret variable.
- Bound input sizes (body, delay, `ms`, `/large`) so the service cannot be used to burn money or bandwidth. Cap concurrent slow requests.
- The `ok` API must **not** reflect arbitrary input into places that make it an open relay (no outbound calls, no redirects to caller-supplied URLs; `redirect-loop` redirects only to itself).
- Decide public versus private (open question 9). If public, set a low `max-instances` and a budget alert.

### 10.5 Deployment (when the time comes)

- Build with the `Dockerfile`; push to Artifact Registry; `gcloud run deploy` with `--max-instances=1`, `--concurrency` sensible (for example 80), `--timeout` as in 10.3, `--set-env-vars` for the variables above, and `ADMIN_TOKEN` from Secret Manager.
- `PUBLIC_BASE_URL` set to the service URL so the served specs name the right server.
- CI deploys on tag, then runs the section 7 checks against the deployed URL as an extra job (same expectations file, `cloud:` overrides applied).
- Clear Signal users can then try the product with no setup: `clrsig conform https://<service>/ok/openapi.yaml`.

### 10.6 Acceptance for the cloud step

- The same expectations file passes locally and against the deployed service, apart from the cases marked `local-only` or overridden under `cloud:`.
- Reset through the deployed URL returns the seed data; a request without the token is refused.
- Two scenarios run in sequence give the same results as locally.

---

## 11. Relationship to other documents

- `docs/planning/failure-showcase-repo.md`: the earlier design for `clrsig-failure-catalog`: one **tiny server per failure mode**, each with a broken and a fixed variant, linked to pages on the product site. This project is different in purpose: **one large, long-lived fixture** whose main job is to test Clear Signal itself and its coverage of the catalog, not to teach. They can share ideas (readiness poll, CI inversion of exit codes, two CI legs); decide later whether to merge them.
- `apps/desktop/wails/help/what-we-check.md`: the catalog of check IDs this project must cover.
- `docs/planning/backlog.md`: gaps found while building this project become backlog items, and rows in the catalog.
