# What Clear Signal checks, and what it does not

This is the list of every failure, warning and "not checked" outcome Clear Signal can report, and of the failures it cannot catch. Read it instead of guessing: if a failure is not here, we either do not catch it or we have forgotten to write it down, and in both cases we want to hear about it.

In the app this list is the **What We Check** tab of the Help panel (menu Help, What We Check); in the repository it is `apps/desktop/wails/help/what-we-check.md`.

Two kinds of checking, with the names the product uses:

- **API Contract Validation** (section 1): a response is compared with what the OpenAPI spec declares for it. It runs on every scenario step that has a spec, in `clrsig conform`, and for the `schema` assertion.
- **Spec Conformance Testing** (section 2): every operation of the spec is exercised, with requests built from the spec, and the run itself is checked (requests that could not be sent, negative probes, coverage, changes between two versions of a spec).

## How to read the tables

- **ID.** `CS-<AREA>-<number>`. These are provisional: backlog item B-049 will make them the IDs shown in every report, and will keep this file and the code in step with a test. Quote an ID when you report a problem.
- **State.** **caught**: we detect it. **partly**: we detect some cases (the note says which). **not caught**: we do not detect it. **not checked**: the outcome looks like a pass but nothing was compared (section 1.2).
- **Severity today.** Every failure below fails the run (exit code 1). The only exceptions are marked *warning* (shown, exit code unchanged) or *not a failure*. Separate warning levels are backlog item B-050.
- **Evidence.** The test (or the Conformance Server operation) that proves the row. A row without evidence is at most *partly*. Rows marked **KnownGap** are pinned by a test of the same name, so when the behaviour changes the test fails and this file must change with it.
- **Two checkers.** The response checks exist twice. **A** is the step spec check (`validateSpec`) and `conform`, which call `ValidateResponseByPath`. **B** is the `schema` assertion (`type: schema`), which calls `ValidateResponse` by operation id. B is narrower: it only compares the body.

Every failure message follows the same shape: **what** is wrong, **where** it is (the part of the response and the place in the spec), **why** it fails and **how** to fix it. Use `clrsig conform --explain` to see it on screen; the JSON, HTML, Markdown and JUnit reports and the desktop page always show it.

## 1. API Contract Validation: is this response what the spec says?

### 1.1 Failures

| ID | Title | What it means | Example | A: step / `conform` | B: `schema` assertion | Evidence |
|---|---|---|---|---|---|---|
| CS-RESP-001 | Undeclared response status | The server answered a status the spec does not declare for the operation. A `default` response, or a range such as `2XX`, counts as declared. | Spec declares `200` and `404`; the server answers `500`. | caught | not caught (status is not looked at) | `TestResponseChecks`, `TestCheck_ResponseStatus`, Conformance Server `GET /status` |
| CS-RESP-002 | Wrong response Content-Type | The `Content-Type` is not one the spec declares for that response. The body is then **not compared**, so nothing else about it is checked. | Spec says `application/json`, server sends `text/plain`. | caught | not caught (reads the body as JSON whatever the type) | `TestResponseChecks`, `TestCheck_ResponseMediaTypes`, Conformance Server `GET /widgets` |
| CS-RESP-003 | Required response header not sent | A header the spec marks `required` is missing. The body is not compared. | `X-Total-Count` is required and absent. | caught | not caught (headers ignored) | `TestResponseChecks`, `TestFindings_TellWhatWhyWhereAndHowToFix`, Conformance Server `GET /inventory` |
| CS-RESP-004 | Response header of the wrong type | A declared header (required or optional) is sent with a value that does not fit its schema. | `X-Total-Count: many` where an integer is declared. | caught | not caught | `TestResponseChecks`, `TestCheck_ResponseHeaders` |
| CS-RESP-005 | Required body field missing | The body lacks a property the schema lists as `required`. | `{"x":1}` where `id` is required. | caught | caught | `TestCheck_ResponseBody`, Conformance Server `GET /categories` |
| CS-RESP-006 | Body field of the wrong type | A property, or the whole body, has a different type from the schema. | `"price": "9.99"` where a number is declared; an array where an object is declared. | caught | caught | `TestCheck_ResponseBody`, Conformance Server `GET /products/{id}` and `GET /orders` |
| CS-RESP-007 | Value outside the declared constraints | A value breaks `enum`, `minimum`, `maximum`, `minLength`, `maxLength`, `pattern`, `minItems`, `uniqueItems`, the `date-time` format, or is `null` where `nullable` is not set. | `"age": -1` with `minimum: 0`. | caught | caught | `TestCheck_ResponseBody` |
| CS-RESP-008 | Body is not valid JSON, or is empty | The body cannot be parsed, or is empty where the response declares a schema. | `{not json`; an empty body on a `200` with a schema. | caught | **partly**: invalid JSON is caught, an empty body is not checked | `TestCheck_ResponseBodyFormat` |
| CS-RESP-009 | Body matches no `oneOf` / `anyOf` branch | A polymorphic body fits none of the allowed shapes. | `{"c":1}` with branches that need `a` or `b`. | caught | caught | `TestCheck_ResponseBody` |
| CS-RESP-010 | Undeclared extra property | The body has a property the schema does not declare. | `{"id":1,"extra":true}`. | **partly**: caught only when the schema sets `additionalProperties: false`. OpenAPI allows extra properties by default, so on an open schema this is not a violation. | same | `TestCheck_ResponseBody` (both rows, one is **KnownGap**) |
| CS-RESP-011 | `writeOnly` property in a response | A property marked `writeOnly` (for example a password) appears in a response. | `{"secret":"x"}`. | caught | not caught | `TestCheck_ResponseOnlyRules` |
| CS-RESP-012 | Body media type that cannot be read | The declared media type has no decoder (for example `application/xml`). It is **reported as a violation whatever the body is**, so a valid XML response fails. | A valid `<r><id>1</id></r>` against `application/xml`. | **partly**: a false failure | **partly**: reads it as JSON and reports invalid JSON | `TestCheck_ResponseBodyFormat` (**KnownGap**) |

Media types that are read: `application/json`, any `+json` type such as `application/problem+json`, and `text/plain` (checked against its schema). Parameters such as `charset` are ignored when matching. (`TestCheck_ResponseMediaTypes`.)

### 1.2 Not checked: looks like a pass, nothing was compared

These never fail the run and never count as passing a check. Scenario results show the ones marked * as "not checked" with the reason.

| ID | Title | When it happens | Evidence |
|---|---|---|---|
| CS-NC-001 | Path not in the spec * | A scenario step calls a URL the spec does not describe: nothing to compare with. | `TestCheck_NotChecked`, `TestRunner_SpecCheckStates` |
| CS-NC-002 | No response content declared | The spec declares the status but no body (`204`, or a description only): any body is accepted. | `TestCheck_NotChecked` |
| CS-NC-003 | Binary or schema-less media type | `application/octet-stream` and similar: there is no structure to compare. | `TestCheck_NotChecked`, `TestCheck_ResponseMediaTypes` |
| CS-NC-004 | Spec validation turned off for the step * | The step has `validateSpec: false`. | `TestRunner_SpecCheckStates` |
| CS-NC-005 | Scenario has no spec | No `specFile`, so no response check runs at all. | `TestRunner_NoSpecLeavesSpecCheckEmpty` |
| CS-NC-006 | The step did not complete * | The request failed before a response arrived (network error). | `TestRunner_SpecCheckStates` |
| CS-NC-007 | Operation skipped by a filter or read-only | `--tags`, `--paths` or `--read-only` left it out: not sent, shown as skipped, counted as not covered. | `TestRunner_Run_TagFilter`, `TestRunConformance_ReadOnlySendsNoWrites`, `TestConformReadOnlySendsNoWrites` |
| CS-NC-008 | Response without a Content-Type | A body with no `Content-Type` header is read as JSON (lenient), not reported. | `TestCheck_ResponseMediaTypes` |

## 2. Spec Conformance Testing: does the API do what the whole spec says?

### 2.1 The request Clear Signal builds, and the run

| ID | Title | What it means | Example | State | Severity today | Evidence |
|---|---|---|---|---|---|---|
| CS-REQ-001 | Request parameter does not fit | A path, query or header value built from the environment does not match the parameter's declared type, so the server was not tested with a valid request. | `id=abc` for an integer path parameter. | caught (the first problem found is reported) | failure | `TestRequestChecks_NameTheParameterOrTheBody`, `TestFindings_RequestProblemsSayWhatWhereWhyAndHowToFix` |
| CS-REQ-002 | Request body does not fit | The generated body, with the environment's values filled in, does not match the request schema. | `price="cheap"` where a number is declared. | caught | failure | same two tests |
| CS-RUN-001 | Not sent: no value for a placeholder | A `{{name}}` in the URL, query, headers, auth or body has no value in the environment or `--var`. The request is **not sent**, the operation counts as failed and not covered, and one line lists every missing name once. A defined but unused variable is ignored. | `GET /items/{{id}}` with no `id`. | caught | failure | `TestRunner_DoesNotSendARequestWithAMissingValue`, `TestConformDoesNotSendWhenAValueIsMissing` |
| CS-RUN-002 | Not sent: a dynamic value failed | A `{{$function}}` is unknown or has bad arguments, so the request is not sent. | `{{$noSuchFunction}}`. | caught | failure | `TestDiagnose_Blocked` |
| CS-RUN-003 | No response | The server did not answer: connection refused, timeout, TLS error. | Wrong base URL. | caught | failure | `TestDiagnose_OtherAndRequestAndNoResponse` |
| CS-RUN-004 | Spec cannot be read | The spec file or address is missing or does not parse. | A typo in the path. | caught | error (exit code 2) | `TestConformCmd_MissingSpec` |
| CS-RUN-005 | Environment refers to an unset `${NAME}` | A value in the environment file refers to a process variable that is not set; it stays as written. | `token: ${API_TOKEN}` with no `API_TOKEN`. | caught | **warning** (exit code unchanged) | `TestRun_WarnsOnceAboutAnUnsetEnvironmentVariable`, `TestConformanceService_WarnsAboutUnsetEnvironmentReferences` |
| CS-RUN-006 | New violation against a baseline | With `--baseline`, a violation that is not in the recorded file. Recorded ones are accepted, fixed ones are reported. | A new undeclared status after the baseline was written. | caught | failure (only for new ones) | `TestConformBaseline_RatchetsQualityUp` |

### 2.2 Negative probes (`clrsig conform --negative`)

Each probe sends a deliberately wrong request and **passes when the server answers any 4xx**. A probe fails when the server accepts the bad request.

| ID | Title | What is sent | State | Evidence |
|---|---|---|---|---|
| CS-PROBE-001 | Missing required field accepted | A body without one required field (expected `400` or `422`). | caught | `TestGenerateNegativeProbes_ExpectedKinds`, `TestNegativeRunner_LenientServer_ProbesFail` |
| CS-PROBE-002 | Wrong field type accepted | A body with one field of the wrong type (expected `400` or `422`). | caught | same |
| CS-PROBE-003 | Missing required query parameter accepted | A request without a required query parameter (expected `400`). | caught | same |
| CS-PROBE-004 | Missing credentials accepted | A request to a secured operation without credentials (expected `401` or `403`). | caught | same, `TestNegativeRunner_StrictServer_AllPass` |

Only these four kinds are probed. Wrong header or path-parameter types, boundary values, enum violations, unknown fields and bad credentials (as opposed to none) are **not** probed (CS-GAP-009).

### 2.3 Coverage signals (Contracts page, `clrsig coverage quality`)

These tell you what has **not** been proven yet. They are signals, not failures: they never change an exit code.

| ID | Title | What it means | Evidence |
|---|---|---|---|
| CS-COV-001 | Never exercised | No recorded run reached the operation. | `TestComputeQuality_NeverExercised` |
| CS-COV-002 | Happy path only | Only `2xx` responses were ever observed for it. | `TestComputeQuality_HappyPathOnly` |
| CS-COV-003 | Error responses never triggered | The spec declares `4xx`/`5xx` responses that no run has produced; each missing code is listed. | `TestComputeQuality_ErrorsNeverTriggered` |
| CS-COV-004 | Secured but never tested with credentials | The spec requires security, yet every call was made without any. | `TestComputeQuality_SecuredNeverAuthed` |

### 2.4 Spec against spec (`clrsig diff`, `clrsig impact`)

| ID | Title | What it means | State | Evidence |
|---|---|---|---|---|
| CS-DIFF-001 | Breaking change | A change that can break existing clients (oasdiff level **ERR**). | caught | `TestDiff_FailOnThresholds`; example from the shipped specs: `api-removed-without-deprecation` |
| CS-DIFF-002 | Warning-level change | A change that may matter (oasdiff level **WARN**). | caught (passed through) | `TestDiff_FailOnThresholds` |
| CS-DIFF-003 | Informational change | A change that is safe to ship (oasdiff level **INFO**). | caught (passed through) | `TestDiff_FailOnThresholds`; examples: `endpoint-added`, `api-version-changed` |
| CS-IMP-001 | Scenario step hits a breaking change | A step's method and path match an operation that has a breaking change. Only changes that name a method and a path are matched. | caught | `TestImpactPathMatches_PathParam` |

`diff` uses [oasdiff](https://github.com/oasdiff/oasdiff)'s rules and levels as they are (rule IDs such as `endpoint-added`); this file does not copy its list. `--fail-on breaking|warning|info` sets which level fails the run.

## 3. Known gaps: what we do not catch

| ID | Title | What is missed | Why | Evidence |
|---|---|---|---|---|
| CS-GAP-001 | Formats other than `date-time` | `format: email`, `uuid` and other formats are not validated: `"mail": "nope"` passes. | The validation library only checks the formats it knows by default. Registering more is possible. | `TestCheck_ResponseBody` (**KnownGap** rows) |
| CS-GAP-002 | Extra properties on open schemas | See CS-RESP-010: not a violation under the OpenAPI default. A strict mode could warn about them. | By design of OpenAPI; a warning level (B-050) would be the place. | `TestCheck_ResponseBody` (**KnownGap**) |
| CS-GAP-003 | Undeclared response headers | A header the spec does not mention is never reported. | Not implemented; low value until warnings exist (B-050). | `TestCheck_ResponseHeaders` (**KnownGap**) |
| CS-GAP-004 | Non-JSON response bodies | XML and other media types are not validated; they are reported as a violation (CS-RESP-012). | No decoder for them; should be "not checked" instead. | `TestCheck_ResponseBodyFormat` (**KnownGap**) |
| CS-GAP-005 | Response time | Nothing is measured against a time limit. | Not implemented; scenario assertions are `status`, `header`, `jsonpath` and `schema` only. | no check exists |
| CS-GAP-006 | Non-JSON request bodies | Operations that take a form, multipart or XML body are generated **without** a body. | Only `application/json` request bodies are generated. | `TestCheck_OnlyJSONRequestBodiesAreGenerated_KnownGap` |
| CS-GAP-007 | `impact` and placeholders in the path | A step whose URL path contains a placeholder (`{{baseUrl}}/users/{{userId}}`) is never matched, and `.../{{userId}}/orders` is read as `/orders`, so a breaking change can be missed or matched to the wrong operation. | Bug: the path is read after the last `}}`. Backlog B-051. | `TestImpact_PlaceholderInThePathIsNotMatched_KnownGap` |
| CS-GAP-008 | Security beyond "no credentials" | Wrong, expired or under-privileged credentials are not probed; only their absence is (CS-PROBE-004). | Not implemented. | no check exists |
| CS-GAP-009 | More negative probes | Wrong header or path-parameter types, boundary values, enum violations, unknown fields. | Not implemented; only four kinds are generated. | `TestGenerateNegativeProbes_ExpectedKinds` |
| CS-GAP-010 | Callbacks, webhooks and links | Operations that are not under `paths` are not exercised. | Only `paths` are generated. | no check exists |
| CS-GAP-011 | Only the first problem per response | A response with a header problem and a body problem reports the header problem first. | The validation library stops at the first problem. | `TestResponseChecks` |

## Adding to this list

Whenever we learn of a failure kind, a case we do not catch, a false pass or a false failure, add a row here in the same change: ID, title, what it means, example, state and the test that proves it. When a gap is fixed, move its row to the checks and update its test. A test (`TestCatalogCitesRealTests`) fails if a row cites a test that does not exist.
