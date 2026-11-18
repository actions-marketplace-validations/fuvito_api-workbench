# Dynamic values and scenario variables

- **Status:** Approved — phase 1 built (2026-11-17); phases 2 and 3 pending
- **Date:** 2026-11-17
- **Touches:** `packages/core/variables`, `packages/core/scenario`, `packages/core/domain`, `serialization` (scenario and bundle formats), `importer`, `clispec`, desktop editors, `format-spec.md`, the AI prompt

## 1. Summary

Today every value in a scenario or environment is static text, so a test that creates a user, an order or a booking can only use the same data every time. That limits how thoroughly an API can be tested (unique constraints, many different inputs, dates relative to "today") and it makes tests brittle.

This proposal adds **dynamic values** (random text, numbers, booleans, ids, dates and times with offsets, simple arithmetic) and **scenario-scoped variables**, written in the YAML files in a familiar `{{$name args}}` style, and **evaluated by the Go core**, never by the frontend. There is no scripting language.

## 2. Goals and non-goals

**Goals**
- A user writes `{{$randomInt 18 65}}` or `{{$date +7d}}` and it works: no code, no setup.
- Follow conventions testers already know (Postman, Bruno and the VS Code REST Client all use `{{$guid}}`, `{{$timestamp}}`, `{{$randomInt}}`), so imported Postman collections keep working.
- Two scopes with obvious rules: **global** (environment) and **scenario**.
- A failing run can be **reproduced** (a seed).
- Mistakes are caught early (`clrsig validate`), not by sending `{{$randomIntt}}` to the API.

**Non-goals**
- Pre-request or test scripts (JavaScript). They are powerful but hard to review, need an engine dependency and widen the security surface. Everything stays declarative.
- A large fake-data library in the first version (a small embedded set only; see open decision 6).
- Changing how extraction (`extractions:`) works.

## 3. Requirement review

| Question | Finding |
|---|---|
| Consistent with `CLAUDE.md`? | Yes. Evaluation lives in the Go core (rule 1); YAML and JSON stay serialization formats of one model (rule 4); no new dependency (stdlib only: `math/rand`, `crypto/rand`, `time`). |
| File format contract | **Adds an optional `variables` field to scenarios** (and bundle scenarios). It is additive, so `schemaVersion` stays `"1"`, but an older `clrsig` reading a newer file will reject the unknown field (bundles decode strictly). `format-spec.md`, the AI prompt and the release notes must say so in the same commit. |
| Behaviour change | `{{$something}}` used to be sent as literal text; an unknown `$name` will become an error. Imported Postman names are mapped (section 5). |
| Size | Medium–large: resolver, scenario model, validation, importer mapping, docs, desktop editing. Phased in section 7. |
| Necessary? | Yes. It is the main gap between "replays requests" and "tests an API thoroughly", and every comparable tool has it. |
| Smaller alternative | Only built-ins with no scenario scope (cannot reuse one generated value across steps) — rejected: create-then-fetch flows need it. |

## 4. Design

### 4.1 Syntax: `{{$function arg arg key=value}}`

Same braces users already know. A name starting with `$` is a built-in function; arguments are space-separated, quoted when they contain spaces, and `key=value` options are allowed. Variables keep their current form `{{name}}`.

```yaml
variables:                      # scenario scope (new)
  - name: email
    value: "user-{{$randomString 8}}@example.com"
  - name: dueDate
    value: "{{$date +7d}}"
  - name: total
    value: "{{$calc price * quantity}}"

steps:
  - label: Create order
    request:
      method: POST
      url: "{{baseUrl}}/orders"
      body:
        type: json
        content: '{"customer":"{{email}}","due":"{{dueDate}}","qty":{{$randomInt 1 5}},"gift":{{$randomBool}}}'
```

### 4.2 Built-in functions (first set)

| Group | Functions |
|---|---|
| Ids | `$guid` (alias `$uuid`), `$randomHex n` |
| Text | `$randomString n` (letters and digits), `$randomAlpha n`, `$randomChoice a b c` |
| Numbers | `$randomInt [min max]` (default 0–1000, like Postman), `$randomFloat [min max] [decimals]`, `$randomBool` |
| Date and time | `$timestamp` (Unix seconds), `$timestampMs`, `$isoTimestamp` (RFC 3339, UTC), `$date [offset] [format=…]`, `$datetime [offset] [format=…]`, `$time [offset]` |
| Calculation | `$calc <expression>` (section 4.5) |
| Small fake data (phase 2) | `$randomEmail`, `$randomFirstName`, `$randomLastName`, `$randomFullName`, `$randomPhone`, `$randomWord` |

**Dates.** Default is UTC, ISO 8601 (`2026-11-17`, `2026-11-17T09:30:00Z`). Offsets: `+7d`, `-1M`, `+2h`, `+30m`, `-1y` (units `s m h d w M y`). `format=` uses the familiar `yyyy-MM-dd HH:mm:ss` tokens (not Go layouts, which only Go developers know) plus the names `iso`, `date`, `rfc1123`, `unix`. `tz=Europe/Istanbul` is an optional override.

### 4.3 Scopes and when a value is generated

| Where it is written | Generated | Example use |
|---|---|---|
| Inline in a request (`{{$randomInt 1 5}}`) | **every time it appears** | a different quantity in each request |
| Scenario `variables:` | **once per scenario run**, then stable | an email created in step 1 and fetched in step 3 |
| Environment variables | **once per run of the whole command** (all scenarios share it) | a `runId` that tags every record of one CI run |

Lookup order, most specific first: values extracted by steps → scenario variables → environment variables. Scenario variables are evaluated in order, so a later one may use an earlier one or an environment variable; a cycle or a reference to something undefined is a validation error. Scenario variables cannot use values extracted by steps (those do not exist yet at the start); steps use extraction as today.

"One run" means one `clrsig run` invocation, one desktop Run All, or one single request send.

**Does the same function give the same value?** It depends only on where it is written:

```yaml
variables:
  - name: orderNo
    value: "{{$randomInt 1000 9999}}"     # generated ONCE, when the scenario starts
  - name: otherNo
    value: "{{$randomInt 1000 9999}}"     # a separate variable: generated separately, so a different value
steps:
  - request:
      url: "{{baseUrl}}/orders/{{orderNo}}?copy={{orderNo}}"     # the same value everywhere orderNo is used
      body: {content: '{"a":{{$randomInt 1 9}},"b":{{$randomInt 1 9}}}'}   # inline: two calls, two independent values
```

- `{{orderNo}}` is the same in every step, header, URL, body and assertion of that scenario run.
- Writing the same function twice inline, or defining it in two variables, gives two independent values (they can still match by chance).
- Two scenarios that define the same variable each get their **own** value, because a scenario variable is generated once per scenario run.
- An environment variable written with a function is generated once for the whole command, so every scenario in a folder run sees the same value.
- With the same `--seed`, every one of these values replays identically.

### 4.4 Reproducibility: a seed

All randomness comes from one seeded generator. The seed is chosen at random unless given (`--seed 12345`, or the environment variable `CLRSIG_SEED`), and it is **printed in every report** (text header, JSON `seed`, JUnit property, HTML and Markdown header). Each scenario derives its own generator from the run seed and the scenario's path, so the values do not depend on execution order and stay the same if scenarios later run in parallel. A failing CI run can then be replayed exactly with `--seed`.

`$guid` and `$randomHex` use the same generator, so they are reproducible too. Time functions read the system clock (an injectable clock in the core makes tests deterministic).

### 4.5 `$calc`: basic arithmetic, no code execution

A small built-in evaluator in the core: numbers, variables (their text parsed as numbers), `+ - * / %`, parentheses, and the functions `round(x, n)`, `floor`, `ceil`, `abs`, `min`, `max`. Integers stay integers; floats are trimmed (`0.1*3` gives `0.3`, always with `.`). Division by zero or a non-numeric operand is a clear error. It cannot call anything else, so it is safe to run on shared CI.

### 4.6 Validation and errors

- `clrsig validate` (and the builder, and desktop save) check every `{{$…}}`: unknown function, wrong number or type of arguments, bad offset or format, and variable cycles are reported with their location.
- **Assertions can use variables and functions.** Today assertion values are never interpolated (only requests are), so a test cannot check that a response echoes a generated value. Phase 1 interpolates the assertion `expected` value and its `path` (header name or JSONPath) with the same rules, so `expected: "{{email}}"` works. This is a behaviour change for any assertion that currently contains literal `{{...}}` text.
- At run time an unknown `$` function or a failed evaluation stops that scenario with a clear error; it never sends the literal text.
- An unknown plain `{{name}}` keeps today's behaviour (left as written, with the existing missing-variable hint).

### 4.7 Importers and the AI prompt

- The Postman and Insomnia importers map the common dynamic names to ours (`$guid`, `$timestamp`, `$isoTimestamp`, `$randomInt`, `$randomEmail`, …) and report names they cannot map.
- `format-spec.md` documents the function list, scope rules and the seed, and the AI prompt (which embeds the spec) therefore teaches assistants to generate unique, relative data.

### 4.8 Desktop

- The existing `{{` autocomplete also lists the `$` functions with a one-line description and an example, so they are discoverable without reading the docs.
- The scenario editor gets a **Variables** section (a name/value list like the environment editor). Until then, the YAML view works.
- Resolved scenario variables and the seed are shown in the run result, so a user can see what was generated.

### 4.9 Reports and redaction

- Reports list the seed and the resolved scenario variables (generated data is ordinary, not secret). A variable marked `secret: true` is redacted exactly as environment secrets are.
- Dynamic environment values (for example a generated `runId`) are generated at the start of the run, so they appear in the report like any other variable.

## 5. Postman-compatible names

`$guid`, `$randomUUID`→`$guid`, `$timestamp`, `$isoTimestamp`, `$randomInt`, `$randomAlphaNumeric`→`$randomString 1`, `$randomBoolean`→`$randomBool`, `$randomEmail`, `$randomFirstName`, `$randomLastName`, `$randomFullName`, `$randomPhoneNumber`→`$randomPhone`, `$randomWord`. Others are reported as "not supported" at import, never silently dropped.

## 6. Risks

| Risk | Mitigation |
|---|---|
| Hard-to-reproduce failures from random data | Seed printed everywhere; `--seed` replays |
| "Is it the same value or a new one?" confusion (a well-known Postman pain point) | The three-row table above is the whole rule; documented in one place and shown in the editor hint |
| Random values colliding with unique constraints | `$guid`, `$timestampMs` and longer `$randomString` are documented as the unique options |
| Time zones and clock skew between the runner and the server | UTC default; `tz=`; documented |
| Older `clrsig` rejects a newer file | Called out in `format-spec.md` and the release notes |
| Scope creep toward a scripting language | Non-goal; `$calc` is a fixed evaluator |

## 7. Phasing and acceptance criteria

| Phase | Delivers |
|---|---|
| **1 — Variables and core functions** | interpolation of assertion values (`expected`, `path`); scenario `variables:` (ordered list) and scope rules; `$guid`, `$randomString/Alpha/Hex/Choice`, `$randomInt/Float/Bool`, `$timestamp*`, `$date/$datetime/$time` with offsets and formats; seed (`--seed`, report header); validation; importer name mapping; `format-spec.md`, AI prompt, user guide; CLI Command Builder `--seed`. Acceptance: one scenario with a generated email used by create and fetch passes; the same `--seed` gives identical requests; a typo is a validation error; the Postman fixture with `{{$randomInt}}` imports and runs. |
| **2 — Calculations and fake data** | `$calc` (with its error cases) and the small fake-data set |
| **3 — Desktop** | autocomplete with function help, Variables section in the scenario editor, run result shows seed and variables |

## 8. Testing

- Unit tests per function (ranges, formats, offsets across month ends and DST, leap years) with an injected clock and seed.
- Determinism: the same seed gives byte-identical requests regardless of scenario order.
- Scope tests for every row of the table in 4.3, including the folder-run case (environment value shared, scenario value per scenario).
- Validation tests for every error in 4.6.
- Importer tests with a Postman collection using the mapped names.
- CLI tests through the real command tree; format round-trip (YAML and JSON) with and without `variables`.

## 9. Open decisions (recommendation first)

1. **Syntax.** `{{$name args}}` (Postman, Bruno, REST Client style; recommended) or Go-template style `{{ randomInt 1 10 }}`, which clashes with the current `{{var}}` form.
2. **Generation rule.** Inline = every time, scenario variables = once per scenario run, environment = once per command run (recommended).
3. **Scenario variables as an ordered list** like environment variables (recommended) rather than a YAML map (order is not reliable in a map).
4. **Seed.** Random by default, `--seed` to replay, printed in every report, derived per scenario (recommended).
5. **Calculations.** Our own small evaluator (recommended; no dependency) or approve a library such as an expression engine.
6. **Fake data.** A small embedded set in phase 2 (recommended), or approve a fake-data library later.
7. **Unknown `$function`** is an error (recommended), not literal text.
8. **Date formats.** `yyyy-MM-dd`-style tokens (recommended) or Go layouts.
9. **Phasing.** Phase 1 first; calculations right after (recommended), desktop UI last.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Proposal written from the project owner's request: dynamic values, scenario scope, easy and conventional, evaluated in the core | assistant |
| 2026-11-17 | Project owner replied "yes" to the nine recommendations in section 9 and asked how the same function behaves in scenario variables (answered in 4.3) | project owner |
| 2026-11-17 | Found while answering: assertion values are not interpolated today; added to phase 1 | assistant |
