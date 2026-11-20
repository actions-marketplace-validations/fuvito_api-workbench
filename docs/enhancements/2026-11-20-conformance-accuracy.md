# Conformance accuracy: real response checks and typed request bodies

**Status:** Done (2026-11-20): part A and part B accepted (all six recommendations) and built
**Date:** 2026-11-20
**Author:** Claude Code, for the project owner
**Backlog:** B-002 (response checks), B-003 (generated request bodies)

## 1. Summary

Two things make `clrsig conform`, scenario spec validation and the Check Conformance page say "passing" when they should not, or fail in ways the user cannot fix:

- **A. Response checks silently pass (B-002).** Measured on the current code, a response is only really checked when the spec declares **no response headers** and the server answers with a **declared status and Content-Type**. Otherwise the check returns "no violations" without looking at the body. This is one bug, not three separate gaps (section 3).
- **B. The generated request body cannot be valid (B-003).** Every property becomes a *quoted string* placeholder, optional properties included, and `allOf` bodies become `{}`. A required number or boolean can therefore never be sent correctly, whatever the user puts in the environment (section 6).

Part A is a correctness fix with a behaviour change (more violations will appear). Part B changes what generated and imported requests contain.

## 2. Goals / non-goals

**Goals**
- A response is never reported as conforming when it was not actually compared with the spec.
- Say precisely what is wrong: undeclared status, wrong `Content-Type`, missing or malformed declared header, or a body that does not match.
- A generated request can be made valid by supplying values in the environment, for every field type.
- Keep one implementation in the Go core, used by the CLI, the scenario runner and the desktop.

**Non-goals**
- No change to scenario, request, environment or bundle file formats (`format-spec.md` is unaffected; only the *content* of generated bodies changes).
- No new dependency (kin-openapi already provides the checks).
- No generation of invented sample data in this proposal (an option for later, section 6, backlog B-046).
- No change to how negative probes build their bodies.

## 3. Part A: what is wrong today

`openapi.ValidateResponseByPath` forwards **only `Content-Type`** into kin-openapi's `ValidateResponse` and then does `parseSchemaErrors(me.Err)`. kin-openapi reports several problems as `ResponseError{Reason: "…", Err: nil}`; with `Err == nil`, `parseSchemaErrors` returns nothing.

Measured (a spec whose `200` declares a required `X-Total-Count` header and a JSON body schema):

| Response | kin-openapi says | `clrsig` reports today |
|---|---|---|
| Body violates the schema, header present | `Err`: property "id" is missing | **1 violation** (correct) |
| Same body violation, but the spec declares a response header | header check fails first (our call never forwards the real header) | **0 violations**, body is never checked |
| Required header really missing | `Reason`: response header "X-Total-Count" missing | **0 violations** |
| Header has the wrong type | `Reason` + `Err` (header value) | partial, the header name is dropped |
| Wrong `Content-Type` (`text/plain`) | `Reason`: response header Content-Type has unexpected value | **0 violations**, body is not checked |
| Undeclared status (500) | `Reason`: status is not supported | **0 violations** |

Consequences today: an API that answers `500`, answers with `text/html`, drops a declared header, or sends a broken body can show as **Passing** (CLI exit 0, Contracts "passing", the page's green rows) whenever the operation declares any response header, which is common (`Location`, `X-Total-Count`, rate-limit headers, `ETag`).

## 4. Part A: design

1. **Forward every real response header** into the check, so declared headers are compared with what the server actually sent (`ValidateResponseByPath` already receives `resp.Headers`).
2. **Never drop a `ResponseError`**: when `Err` is nil, report `Reason`; when both exist, report both (so a header problem names the header). Keep `ValidationError` as the carrier; add a small `Kind` so callers can classify without parsing text.
3. **New violation kinds**, each with a plain-language explanation in `conformance.Diagnose` (what the spec expected, what the server sent, why it fails, how to fix):
   - `undeclared-status`: "The server answered 500, which the spec does not declare for GET /a." (A `default` response counts as declared.)
   - `wrong-content-type`: the spec declares `application/json`, the server sent `text/plain`; the body was **not** compared.
   - `missing-header` and `bad-header`.
4. **Same function everywhere.** The scenario runner (`validateSpec`), `conform`, the desktop page and the exports all call it, so they change together. A step that deliberately tests an error path the spec does not declare keeps its existing escape hatch, `validateSpec: false`, or the spec can declare the status.
5. **Order.** kin-openapi stops at the first problem and checks headers before the body, so a response with both a header problem and a body problem reports the header problem first. Reporting both would need a second pass with the headers removed from a copy of the operation; not proposed now (open decision 2).

### Rollout and compatibility

This is a correctness fix, so the old silent behaviour is not kept behind a flag. Existing users will see new violations (correctly) the first time. The baseline feature already exists for adopting it gradually: `clrsig conform --write-baseline`, then `--baseline`. The release notes and the user guide say so. Reports gain the new kinds additively (new `findings` kinds in JSON; new lines elsewhere).

## 5. Part A: testing

- Table-driven core tests built from the measurements above: every row of the table, each with headers forwarded and with the `Reason`-only paths.
- A test that **fails on the old code** for the "declared header disables body checking" case.
- Runner and `conform` tests: an undeclared status and a wrong Content-Type fail the operation; `default` responses are accepted; `validateSpec: false` still skips.
- The Conformance Server gets three more deliberate defects so the known-violation set is end to end (open decision 3): a wrong Content-Type, a missing declared header, an undeclared status.
- Existing tests, the parity test and the Playwright specs are updated for the new known set.

## 6. Part B: what is wrong today and the design

Measured with `GenerateRequests` on a typed request schema:

```
"name": "{{name}}",  "price": "{{price}}",  "inStock": "{{inStock}}",
"kind": "{{kind}}",  "note": "{{note}}",    "size": "{{size}}",  "tags": "{{tags}}"
```
- `price` (number), `inStock` (boolean), `size` (object) and `tags` (array) are quoted strings, so after substitution the request still violates the spec: nothing the user can put in the environment fixes it.
- `example`, `default` and `enum` are ignored; optional properties are included, so every one needs a value (since B-009 an operation with an unfilled placeholder is **not sent**).
- An `allOf` body becomes `{}` and its required property is missing (1 request violation).
- The Swagger 2.0 importer has the same generator.
- The negative probes are **not** affected: they build their own typed bodies (`bodyWithoutField`, `placeholderForType`), they do not parse the stub.

### Design

1. **Typed placeholders.** `string` stays `"{{name}}"`; `integer`, `number`, `boolean`, `array` and `object` become the bare `{{name}}`, so a value of `9.99`, `true`, `[1,2]` or `{"w":3}` produces valid JSON. Substitution is textual, so this needs no resolver change and no new syntax.
2. **Required properties only**, plus any optional property that has an `example` or `default`. Fewer placeholders means fewer values the user must supply and fewer blocked operations.
3. **`example` / `default` / first `enum` value** are used as the *literal* value when present, so such fields need no environment entry. (They cannot be overridden from the environment; editing the request file does.)
4. **Composition.** `allOf` members are merged (properties and required lists); for `oneOf` / `anyOf` the first branch is used.
5. **Nested objects and arrays stay one level:** the property is a single `{{size}}` / `{{tags}}` placeholder holding a JSON value. Expanding nested properties needs a naming scheme for the flat variable namespace and is left for later.
6. **One shared implementation** for OpenAPI 3.x and Swagger 2.0.

### Trade-off to be aware of

A generated request file that contains `"price": {{price}}` is **not valid JSON until the variable is filled in**. It is still a valid YAML file and runs correctly; the cost is that the request editor's *Format JSON* button cannot format it before substitution. Alternatives (a typed marker such as `"{{price|number}}"`) need new resolver syntax, which is a file-format decision, so they are not proposed.

### Larger alternative (not proposed here)

For `conform`, generate valid *sample values* from the schema (`format`, `minimum`, `enum`, …) so no body values are needed in the environment at all, keeping `{{placeholders}}` only for what cannot be invented (path ids, credentials). That is a product-direction change, tracked as backlog **B-046**.

### Compatibility

Importing a spec produces different request files than before (typed, fewer fields). Existing saved requests are untouched. `conform` runs that supplied values for optional properties simply no longer use them. No file format changes.

## 7. Part B: testing

- Generator tests: each type, required-only, `example`/`default`/`enum`, `allOf`, `oneOf`, nested object and array properties, the Swagger 2.0 path.
- A test that every generated stub, after filling plain typed values, **passes `ValidateRequest`** (this is the check that is impossible today).
- The negative-probe tests and the Conformance Server tests still pass unchanged.
- The Conformance Server spec gets one operation with number, boolean and array fields so the end-to-end `conform` run exercises it.

## 8. Requirement review

- **Architecture:** all changes are in `packages/core/openapi` and `conformance` plus the existing reporting and desktop display of `Diagnose` findings. No Wails, React or Cobra dependency is added. Consistent with rules 1 and 2.
- **Locked stack / out of scope:** no new dependency; nothing from the out-of-scope list.
- **File format contract:** unchanged (`schemaVersion` stays `1`, no new fields). Generated *content* changes (Part B).
- **Size:** A is small to medium (one function, new finding kinds, tests, the Conformance Server). B is medium (generator rewrite, importer, tests).
- **Necessity:** A fixes a silent false pass and is not optional in my view. B removes failures a user cannot fix; without it B-009's "not sent" rule makes typed bodies strictly worse.
- **Smaller alternative:** A alone (items 1 and 2 of section 4 are about twenty lines) closes the dangerous gap; B can follow.
- **Recommendation: build A now; build B after the open decisions below.**

## 9. Documentation impact

User guide (what conform checks and the new violation kinds; adopting with a baseline), the README line on conformance if it overstates, `CLAUDE.md` changelog, the AI prompt only if it describes generated bodies. `format-spec.md`: no change.

## 10. Risks

- **Existing CI turns red** when the new violations appear. Mitigation: they are correct findings, the baseline feature adopts them gradually, release notes call it out.
- **Over-strict undeclared status** on specs that rely on implicit error responses. Mitigation: `default` responses count, `validateSpec: false` per step, declare the status in the spec.
- **Typed placeholders make request files non-JSON before substitution** (section 6). Mitigation: documented; no behaviour impact at run time.
- **kin-openapi message wording changes** could degrade a finding to the generic "other difference". Mitigation: kinds come from the `Reason` strings under test, with a generic fallback that still shows the raw text.

## 11. Open decisions

1. **Undeclared status is a violation, always?** Recommendation: yes, with `default` accepted and `validateSpec: false` as the escape hatch.
2. **Report header and body problems together?** Recommendation: not now (first problem only); revisit if users ask.
3. **Add three deliberate defects (wrong Content-Type, missing header, undeclared status) to the Conformance Server?** Recommendation: yes, it makes Part A demonstrable and testable end to end; the "three known violations" in the docs become six.
4. **Part B scope:** typed placeholders, required-only, `allOf` merge and `example`/`default`/`enum` literals (section 6) now, sample-value generation (B-046) later? Recommendation: yes.
5. **Required-only bodies:** drop optional properties from generated requests unless they have an example or default? Recommendation: yes (it is what makes B-009 usable on real specs).
6. **Order of work:** Part A first as its own release step, then Part B. Recommendation: yes.

## 12. Decision log

| Date | Decision | Who |
|---|---|---|
| 2026-11-20 | Proposal written after measuring the validator (all blind spots trace to dropped `ResponseError.Reason` and unforwarded headers) and the body generator | Claude Code |
| 2026-11-20 | Part B built as designed. Differences: the request-side validator got the same fix as the response side (parameter name and body context were dropped); `readOnly` properties are also left out; Format JSON understands placeholders; the typed end-to-end test is a CLI test instead of a new Conformance Server operation (it would have changed every count again for no new coverage) | Claude Code |
| 2026-11-20 | Owner accepted decisions 1 to 6 ("go") and added: every failure must say what, why and where so the user can locate and fix it. Part A built with a `Part`, `Where` and `Spec` location on every finding | Owner, Claude Code |
