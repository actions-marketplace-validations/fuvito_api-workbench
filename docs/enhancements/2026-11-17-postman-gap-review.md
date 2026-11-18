# Postman gap review and candidate enhancements

- **Status:** Proposed — nothing here is approved. The project owner decides which items graduate to their own proposal.
- **Date:** 2026-11-17
- **Purpose:** compare Clear Signal with Postman against the current code, and list the few gaps worth closing while keeping the product lean. Supersedes the feature table in [`../clrsig-vs-postman-2026-11-15.md`](../clrsig-vs-postman-2026-11-15.md), which predates `impact`, `conform --negative`, `clrsig mcp`, `clrsig import`, dynamic values and the CLI Command Builder.
- **Method:** commit history up to `d3a2428`, plus code and documentation searches for each capability. The desktop app was **not** run, so "No" means "nothing found in the code", not "verified absent in the UI".

**How an item graduates:** it gets its own dated proposal in this folder (see [README](README.md)) with the full requirement review, and is agreed before code is written. Sizes: **S** ≈ a day or less, **M** ≈ a few days, **L** ≈ a week or more.

## 1. Comparison

### Where Clear Signal is ahead or equal on its core purpose

| Capability | Clear Signal | Postman |
|---|---|---|
| Response validation against OpenAPI | Yes, automatic | Via scripts or paid features |
| Request validation and negative-path probes | Yes (`conform --negative`) | No |
| Breaking-change diff, including `--against <git ref>` | Yes | Governance features (paid) |
| Impact of a spec change on your scenarios | Yes (`impact`) | No |
| Coverage, quality gaps, Contracts page and reports | Yes | No |
| Local-first, git-diffable files, no account | Yes | Cloud-first |
| Desktop and CLI share one engine | Yes | Newman is a separate runner |
| CI output: JUnit, JSON, HTML, Markdown, several reports per run | Yes | Via Newman |
| Secret redaction in reports | Yes | Partial |
| AI and agent support: MCP server, AI prompt, bundles | Yes | Postbot, Postman MCP |
| Postman and Insomnia import | Yes | Not applicable |

### Everyday client features

| Capability | Clear Signal | Postman |
|---|---|---|
| Methods, params, headers; JSON, raw, form, multipart bodies | Yes | Yes |
| Auth: bearer, basic, API key | Yes | Yes |
| Auth: OAuth2, Digest, AWS Signature, NTLM, Hawk | No | Yes |
| Environments and secret variables | Yes | Yes |
| Global and collection variable scopes | No (environment and scenario variables) | Yes |
| Dynamic values (`$guid`, `$calc`, fake data, `--seed`) | Yes | Yes |
| Cookie jar | Yes (desktop) | Yes |
| Proxy and TLS skip-verify | Yes | Yes |
| Client certificates (mTLS) | No | Yes |
| Redirect control | No | Yes |
| Response viewer, search, diff, timing | Yes | Yes |
| cURL import and code snippets | Yes | Yes |
| Pre-request and test scripts (JavaScript) | No, by design | Yes |
| Step `retry` and `timeoutMs` | In the file format, **not applied** | Via scripts |
| Data-driven runs (CSV/JSON iterations) | No (backlog 17) | Yes |
| Parallel runs, tags, scheduled monitors | No (backlog 14, 15) | Partial or paid |

### Deliberately out of scope (keep it that way)

Collaboration and cloud workspaces, mock servers, monitors, documentation publishing, API catalog and governance, GraphQL, WebSocket, gRPC, and JavaScript scripting. See `CLAUDE.md` §9. Scripting in particular would cost the declarative, reviewable, diffable property that sets the product apart: assertions plus dynamic values cover the same ground without a script engine.

## 2. Candidate enhancements

| # | Idea | Theme | Size | Suggested |
|---|---|---|---|---|
| A | Make step `retry` and `timeoutMs` real (or remove them) | Completeness | S–M | **First** |
| B | Redirect control (`followRedirects`) | Conformance accuracy | S | **First** — confirm the problem with a test |
| C | Baseline mode for `conform` | Adoption | M–L | Soon (already backlog item 11) |
| D | Document the OAuth2 token pattern; client-credentials auth only on demand | Auth | S (docs) / M | Docs now, code later |
| E | Client certificates (mTLS) | Enterprise | M | Only when someone asks |

### A. Make step `retry` and `timeoutMs` real — S–M

- **Problem.** The scenario format accepts `retry: {maxAttempts, delayMs}` and `timeoutMs` per step, the app edits them and the files preserve them, but the runner ignores both (`format-spec.md` line 246, user guide, bundle import warns). A format that lets users write something the product silently ignores is the most visible unfinished edge. Already listed in the older project backlog.
- **Evidence.** `domain.ScenarioStep` carries `Retry` and `TimeoutMs`; `packages/core/scenario/runner.go` never reads them. `bundle.go:284` warns about them at import.
- **Idea.** Implement in the runner only (the single place that executes steps): `timeoutMs` overrides the request timeout for that step; `retry` re-runs the step up to `maxAttempts` with `delayMs` between attempts. The report shows the attempt count.
- **Smaller alternative.** Remove both fields from the format, editor and docs. Cheaper, but loses a feature users of flaky environments expect.
- **Notes.** Decide what triggers a retry (see open decisions). Format change: update `format-spec.md` in the same commit and remove the "not applied" warnings. Retried steps must not double-count in coverage or run totals.

### B. Redirect control — S

- **Problem.** The executor's `http.Client` has no `CheckRedirect`, so Go follows up to 10 redirects silently. A spec that declares `301`/`302`/`307` for an operation can never be validated, because the client only sees the final response. For a conformance tool that is a blind spot.
- **Evidence.** `packages/core/executor/executor.go` builds `&http.Client{Transport: transport}`; no redirect handling exists anywhere in the code. This is inferred from reading, not yet shown by a test.
- **Idea.** A request-level option `followRedirects` (default `true`, so nothing changes for existing files). When `false`, the redirect response itself is returned and validated. Same field in the request file, the scenario step's request, the desktop editor and the bundle format.
- **Notes.** Step 1 is a failing test against an `httptest` server that returns a `302`, to confirm the problem. Serialized field: update `format-spec.md`.

### C. Baseline mode for `conform` — M–L

- Already described as **backlog item 11** in [future-enhancements](2026-11-17-future-enhancements.md). Repeated here because it is the largest adoption lever left: pointing `conform` at an existing API produces a wall of failures on day one and teams switch the check off instead of fixing it gradually.
- `--write-baseline file` records today's violations; `--baseline file` fails only on new ones and reports fixed ones, so quality can only ratchet up. Needs a small versioned baseline format and a stable violation identity.

### D. OAuth2 — docs now, code on demand

- **Problem.** After bearer tokens, OAuth2 is the most common thing users ask about.
- **Finding.** A scenario already handles the common machine-to-machine case: a first step POSTs to the token endpoint, extracts `access_token` with a `jsonpath` extraction, and later steps use `auth: {type: bearer, token: "{{token}}"}`. Extracted values whose name looks secret are already masked in reports.
- **Idea.** Document that pattern in the user guide and the example scenario (S). Build a dedicated `oauth2` auth type (client-credentials only, with token caching) only if users find the pattern too heavy (M). Authorization-code and PKCE flows need a browser and a redirect listener: not recommended.

### E. Client certificates (mTLS) — M

- **Problem.** Some internal and enterprise APIs require a client certificate. Postman supports it; Clear Signal does not (`TLS skip verify` and proxy exist, client certificates do not).
- **Idea.** Settings-level certificate, key and CA bundle paths (never stored in project files; paths or env-var references only, in line with the secrets rule). Applies to the desktop app and the CLI through the same executor options.
- **Notes.** Do not build before a real user asks.

## 3. Suggested order

1. **B** (small, a correctness gap in the core purpose), then **A** (finish or remove what the format already promises).
2. **D** documentation (a day, closes a common question without code).
3. **C** when real users start adopting against existing APIs.
4. **E** and a real OAuth2 type only with evidence.

## 4. Open decisions

| # | Question | Recommendation |
|---|---|---|
| 1 | Item A: implement `retry`/`timeoutMs`, or remove them from the format? | Implement. The fields are already saved, edited and documented; removing them is a breaking format change. |
| 2 | Item A: what triggers a retry: transport errors and `5xx` only, or also failed assertions? | Transport errors and `5xx` only. Retrying on assertion failures would hide real contract violations. |
| 3 | Item B: should `followRedirects: false` be a request field or a global setting? | Request field, default `true`; a global setting would make behaviour differ between files. |
| 4 | Item D: build a dedicated OAuth2 auth type? | Not yet. Document the scenario pattern first and see if anyone needs more. |

## 5. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-17 | Gap review written from a comparison with Postman; nothing approved | assistant |
