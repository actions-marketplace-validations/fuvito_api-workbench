# clrsig-failure-catalog — PRD & Technical Design

**Repository:** `github.com/fuvito/clrsig-failure-catalog` (new, public)
**Owner:** Fuat Yazar
**Status:** Pre-build — this document is the source of truth for implementation
**Product site:** [clrsig.com](https://clrsig.com)
**Failure mode catalog:** [clrsig.com/failures](https://clrsig.com/failures/)

---

## 1. Problem

API contract failures are abstract until a developer sees one caught in their own CI pipeline. The clrsig documentation explains *what* the tool detects; this catalog proves it by running against real, broken servers on every commit.

There is currently no public, runnable collection of API failure mode reproductions that:
- works out of the box with a single binary
- shows the broken state and the fix side by side
- is kept green by CI so it cannot quietly rot

---

## 2. Goals

| Goal | Description |
|---|---|
| **Test clrsig in production** | Each sub-project is a live regression test. If a failure mode stops being caught, CI turns red. |
| **Prove value to evaluators** | Clone one sub-project, run two commands, see `clrsig` catch the problem in under two minutes. |
| **Feed the product website** | Each sub-project maps to a page at `clrsig.com/failures/fm-NNN/` and is linked from the full catalog at [clrsig.com/failures](https://clrsig.com/failures/). The catalog CI badge appears on [clrsig.com](https://clrsig.com) as a live signal the detection engine is working. |
| **Show "what good looks like"** | The `fix/` variant in each sub-project demonstrates the corrected spec or server, not just the broken one. |

## 3. Non-goals

- Not a test suite for the clrsig source code (that lives in `fuvito/api-workbench`)
- Not a general-purpose API testing tutorial
- Not covering GraphQL, gRPC, or non-OpenAPI APIs

---

## 4. Success Metrics

- 12 sub-projects live, all running in CI, within the first release
- Every sub-project produces `clrsig` exit 1 (failure caught) on the broken server and exit 0 on the fix
- CI badge green on the catalog repo homepage
- Each sub-project linked from its corresponding `clrsig.com/failures/fm-NNN/` page

---

## 5. User Stories

**As a developer evaluating clrsig**, I want to run a real failure example locally in under two minutes so I can see exactly what the tool catches and how the output looks.

**As a developer debugging a spec violation**, I want to find a named failure pattern that matches my situation so I can understand the root cause and the fix.

**As a Clear Signal maintainer**, I want a CI-green catalog that breaks immediately if a detection regression is introduced to the engine.

---

## 6. Technical Design

### 6.1 Repository Structure

```
clrsig-failure-catalog/
├── README.md                        # Index — links to every sub-project and clrsig.com/failures/
├── .github/
│   └── workflows/
│       └── catalog.yml              # Runs all sub-projects; nightly + on push
│
├── failures/
│   ├── fm-001-number-precision/
│   ├── fm-003-pagination-contract/
│   ├── fm-004-renamed-removed-field/
│   ├── fm-005-nullable-required/
│   ├── fm-010-spec-implementation-drift/
│   ├── fm-016-rate-limit-schema/
│   ├── fm-018-idempotency/
│   ├── fm-022-semantically-invalid/
│   ├── fm-025-incorrect-http-status/
│   ├── fm-027-lost-update/
│   ├── fm-034-http-method-semantics/
│   └── fm-044-sensitive-data-exposure/
│
└── shared/
    ├── go.mod                        # Single Go module for all servers
    └── testutil/
        └── server.go                 # Port picker, graceful shutdown helpers
```

Directory names use the FM number as prefix so they sort in catalog order and link unambiguously to `clrsig.com/failures/fm-NNN/`.

### 6.2 Sub-project Structure

Every sub-project is self-contained:

```
failures/fm-NNN-slug/
├── README.md           # Content for the product page (see template §6.3)
├── server/
│   └── main.go         # Broken server — exhibits the failure
├── openapi.yaml        # The declared spec (what the API promises)
├── scenarios/
│   └── detect.yaml     # clrsig scenario that catches the failure
├── environments/
│   └── local.yaml      # baseUrl + any required variables
└── fix/
    └── main.go         # Corrected server — same spec, all assertions pass
```

**Key constraint:** every server is a single Go file with no external dependencies beyond the standard library. No frameworks, no databases. The spec is OpenAPI 3.0 YAML.

### 6.3 Sub-project README Template

```markdown
# FM-NNN — Failure name

> Full catalog entry: https://clrsig.com/failures/fm-NNN/

**Difficulty:** Easy / Medium / Hard
**Detected by:** `clrsig run` / `clrsig conform` / `clrsig diff`

## What this failure is

One paragraph. What breaks, why it happens in practice, what a consumer experiences.

## Reproduce it

\```bash
go run server/main.go &
clrsig run scenarios/detect.yaml --env environments/local.yaml
\```

Expected output:
\```
(paste actual clrsig failure output here)
\```

## Why clrsig catches it

Which assertion fires. Where in the OpenAPI spec the violation is declared.

## Fix

\```bash
go run fix/main.go &
clrsig run scenarios/detect.yaml --env environments/local.yaml
# exits 0
\```

What changed: one sentence.

## In the wild

When and why this pattern appears in real projects.
```

### 6.4 GitHub Actions Workflow

```yaml
# .github/workflows/catalog.yml
name: Failure Catalog

on:
  push:
  schedule:
    - cron: '0 6 * * *'   # nightly regression guard

jobs:
  catalog:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Install clrsig
        uses: fuvito/api-workbench@v1
        with:
          command: install

      - name: "FM-001 — Number precision (must fail)"
        run: |
          go run failures/fm-001-number-precision/server/main.go &
          sleep 1
          clrsig run failures/fm-001-number-precision/scenarios/detect.yaml \
            --env failures/fm-001-number-precision/environments/local.yaml
          test $? -eq 1    # exit 1 means clrsig caught it — this is correct

      # ... one step per failure mode (same pattern)

      - name: "FM-001 — Number precision fix (must pass)"
        run: |
          go run failures/fm-001-number-precision/fix/main.go &
          sleep 1
          clrsig run failures/fm-001-number-precision/scenarios/detect.yaml \
            --env failures/fm-001-number-precision/environments/local.yaml
```

**Exit code convention:** `test $? -eq 1` treats a clrsig assertion failure as CI success. If the broken server unexpectedly passes (`exit 0`), CI fails — detection regressed. The `fix/` variant is tested in a separate step and must exit 0.

### 6.5 V1 Sub-project Catalog

12 sub-projects for the first release, selected to cover a range of detection methods and difficulty levels. All map to pages in the [clrsig.com/failures](https://clrsig.com/failures/) catalog.

| Directory | Failure mode | Difficulty | Detection | clrsig.com page |
|---|---|---|---|---|
| `fm-001-number-precision` | Currency, Number, and Precision Formatting Failures | Easy | `clrsig run` schema assertion | [FM-001](https://clrsig.com/failures/fm-001/) |
| `fm-003-pagination-contract` | Pagination Contract Failures | Hard | `clrsig run` multi-step scenario | [FM-003](https://clrsig.com/failures/fm-003/) |
| `fm-004-renamed-removed-field` | Renamed or Removed Response Fields | Medium | `clrsig diff` v1 vs v2 | [FM-004](https://clrsig.com/failures/fm-004/) |
| `fm-005-nullable-required` | Incorrect Optional, Nullable, and Required Field Behavior | Easy | `clrsig run` schema assertion | [FM-005](https://clrsig.com/failures/fm-005/) |
| `fm-010-spec-drift` | OpenAPI Specification and Implementation Drift | Easy | `clrsig conform` | [FM-010](https://clrsig.com/failures/fm-010/) |
| `fm-016-rate-limit-schema` | Rate-Limit and Throttling Failures | Medium | `clrsig run` undeclared 429 | [FM-016](https://clrsig.com/failures/fm-016/) |
| `fm-018-idempotency` | Idempotency and Duplicate-Operation Failures | Hard | `clrsig run` multi-step scenario | [FM-018](https://clrsig.com/failures/fm-018/) |
| `fm-022-semantically-invalid` | Syntactically Valid but Semantically Invalid Payloads | Medium | `clrsig run` jsonpath assertions | [FM-022](https://clrsig.com/failures/fm-022/) |
| `fm-025-incorrect-status` | Incorrect HTTP Status and Business-Error Semantics | Easy | `clrsig run` status assertion | [FM-025](https://clrsig.com/failures/fm-025/) |
| `fm-027-lost-update` | Concurrent Update and Lost-Update Failures | Hard | `clrsig run` multi-step scenario | [FM-027](https://clrsig.com/failures/fm-027/) |
| `fm-034-method-semantics` | HTTP Method and Operation Semantics Failures | Medium | `clrsig conform` + schema assertion | [FM-034](https://clrsig.com/failures/fm-034/) |
| `fm-044-sensitive-data` | Sensitive Data Exposure in API Responses | Medium | `clrsig run` schema (`additionalProperties: false`) | [FM-044](https://clrsig.com/failures/fm-044/) |

The remaining 38 failure modes in the catalog ([clrsig.com/failures](https://clrsig.com/failures/)) are candidates for future sub-projects. Any contributor can propose one by opening a pull request using the template in §6.3.

---

## 7. Task List

### 7.1 Repository Setup

- [ ] Create `github.com/fuvito/clrsig-failure-catalog` (public)
- [ ] Add root `README.md` with project purpose, CI badge, and sub-project index table with links to `clrsig.com/failures/fm-NNN/`
- [ ] Initialise `shared/go.mod` (`module github.com/fuvito/clrsig-failure-catalog`)
- [ ] Add `.gitignore` (Go binaries, IDE files)
- [ ] Add `LICENSE` (MIT)
- [ ] Add `CONTRIBUTING.md` — how to add a new failure mode (uses template from §6.3)

### 7.2 CI Scaffolding

- [ ] Create `.github/workflows/catalog.yml` with the exit-code inversion pattern (§6.4)
- [ ] Wire FM-001 as the first step to verify the full pipeline end-to-end
- [ ] Add separate "fix" step for each sub-project (must exit 0)
- [ ] Add CI status badge to root `README.md`
- [ ] Confirm nightly schedule trigger fires correctly

### 7.3 Sub-projects (one task each)

Each task = broken server + spec + scenario + fix + README + CI step.

- [ ] [FM-001](https://clrsig.com/failures/fm-001/) — Currency, Number, and Precision Formatting Failures
- [ ] [FM-003](https://clrsig.com/failures/fm-003/) — Pagination Contract Failures
- [ ] [FM-004](https://clrsig.com/failures/fm-004/) — Renamed or Removed Response Fields
- [ ] [FM-005](https://clrsig.com/failures/fm-005/) — Incorrect Optional, Nullable, and Required Field Behavior
- [ ] [FM-010](https://clrsig.com/failures/fm-010/) — OpenAPI Specification and Implementation Drift
- [ ] [FM-016](https://clrsig.com/failures/fm-016/) — Rate-Limit and Throttling Failures
- [ ] [FM-018](https://clrsig.com/failures/fm-018/) — Idempotency and Duplicate-Operation Failures
- [ ] [FM-022](https://clrsig.com/failures/fm-022/) — Syntactically Valid but Semantically Invalid Payloads
- [ ] [FM-025](https://clrsig.com/failures/fm-025/) — Incorrect HTTP Status and Business-Error Semantics
- [ ] [FM-027](https://clrsig.com/failures/fm-027/) — Concurrent Update and Lost-Update Failures
- [ ] [FM-034](https://clrsig.com/failures/fm-034/) — HTTP Method and Operation Semantics Failures
- [ ] [FM-044](https://clrsig.com/failures/fm-044/) — Sensitive Data Exposure in API Responses

### 7.4 Website Integration

- [ ] Confirm `clrsig.com/failures/fm-NNN/` page structure can link to the GitHub directory
- [ ] Verify each sub-project README renders cleanly as source material for the product page
- [ ] Link each live sub-project from its corresponding `clrsig.com/failures/fm-NNN/` page
- [ ] Add CI badge to the product site landing page

### 7.5 Release Gate

- [ ] All 12 sub-projects passing CI (broken server exits 1, fix exits 0)
- [ ] Root README index complete with correct FM links
- [ ] Repository announced on product site and in release notes

---

## 8. Backlog — Future Failure Mode Additions

The 38 failure modes not included in v1, with an assessment of build effort and clrsig feasibility. Feasibility reflects what clrsig can do today; the "clrsig work needed" column notes engine improvements that would unlock the full scenario.

Three feasibility ratings:
- **Ready** — clrsig can detect this today with existing scenario/conform/diff commands
- **Partial** — detectable with a workaround or only partially; a targeted improvement makes it clean
- **Needs feature** — requires a new clrsig capability before a useful sub-project can be built

### Category: Data Contracts (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-002](https://clrsig.com/failures/fm-002/) | Silent Timeout and Incomplete Response Failures | Medium | Partial | Timeout assertion type in scenarios; currently only network-level timeout is enforced |
| [FM-006](https://clrsig.com/failures/fm-006/) | Mobile and Older-Client Compatibility Failures | Medium | Ready | None — `clrsig diff` with a "legacy" spec file captures this cleanly |
| [FM-007](https://clrsig.com/failures/fm-007/) | API Versioning and Compatibility Failures | Medium | Ready | None — `clrsig diff v1.yaml v2.yaml` is the primary use case; good showcase |
| [FM-013](https://clrsig.com/failures/fm-013/) | Undocumented Dependencies and Accumulated Integration Failures | Hard | Ready | None — `additionalProperties: false` in schema catches undocumented fields; scenario tests undocumented behaviour |
| [FM-014](https://clrsig.com/failures/fm-014/) | Timezone and Daylight-Saving-Time Failures | Medium | Partial | `format: date-time` validates format but not timezone offset; a `matches` regex assertion fills the gap today |
| [FM-015](https://clrsig.com/failures/fm-015/) | Deprecated Endpoint and Lifecycle Failures | Medium | Ready | None — `clrsig conform` catches 404/410 on removed endpoints; `clrsig diff` flags removals |
| [FM-021](https://clrsig.com/failures/fm-021/) | File Upload and Downstream Processing Failures | Medium | Partial | `multipart/form-data` body type not yet in executor; needs adding before a file-upload scenario works |
| [FM-024](https://clrsig.com/failures/fm-024/) | Localization and Machine-Readable Data Failures | Medium | Ready | None — header assertion on `Accept-Language`, then assert stable machine-readable status fields |
| [FM-047](https://clrsig.com/failures/fm-047/) | Character Encoding and Unicode Handling Failures | Easy | Ready | None — scenario sends Unicode input, asserts round-trip fidelity; straightforward to build |
| [FM-049](https://clrsig.com/failures/fm-049/) | API Documentation and Example Drift | Easy | Ready | None — this is precisely what `clrsig conform` was built for; highest-value quick win after v1 |

### Category: Authentication and Security (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-008](https://clrsig.com/failures/fm-008/) | OAuth Implementation and Token-Validation Failures | Hard | Partial | Multi-step scenario covers the happy path; expired/invalid token assertions work today; full OAuth flow (PKCE, refresh) needs an auth-flow step type |
| [FM-009](https://clrsig.com/failures/fm-009/) | Healthcare Interoperability and HL7 Conformance Failures | Hard | Needs feature | Requires FHIR/HL7 schema validator; outside clrsig's current OpenAPI scope |
| [FM-011](https://clrsig.com/failures/fm-011/) | Payment Gateway Certification and Protocol Conformance Failures | Hard | Partial | Schema assertions cover field/type validation; payment-protocol-specific rules (e.g. Luhn check) need custom assertion type |
| [FM-023](https://clrsig.com/failures/fm-023/) | Broken Object-Level Authorization and Tenant Isolation | Medium | Ready | None — multi-step scenario with two auth tokens; User A cannot read User B's resource; clean demo |
| [FM-030](https://clrsig.com/failures/fm-030/) | API Key and Client-Credential Rotation Failures | Hard | Partial | Scenario simulates expired key returning 401; full rotation lifecycle (revoke + reissue) needs time-based step support |
| [FM-037](https://clrsig.com/failures/fm-037/) | Input Injection and Unsafe Interpretation Failures | Medium | Partial | Scenario verifies injection inputs return declared error codes (not 500/200); clrsig cannot do active exploit detection — that is out of scope by design |
| [FM-043](https://clrsig.com/failures/fm-043/) | Mass Assignment and Unauthorized Field Modification | Easy | Ready | None — scenario sends extra fields (e.g. `role: admin`), asserts they are absent on read-back; clean and impactful |
| [FM-045](https://clrsig.com/failures/fm-045/) | Authentication and Authorization Confusion Across Services | Hard | Partial | Needs two cooperating servers in one sub-project; feasible but complex to set up cleanly |
| [FM-046](https://clrsig.com/failures/fm-046/) | Security Policy and Secret Leakage Failures | Medium | Ready | None — jsonpath assertions confirm no credential fields in error responses; pairs well with FM-044 |

### Category: HTTP and Request Handling (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-012](https://clrsig.com/failures/fm-012/) | Accessibility Conformance Failures in API-Driven Applications | Medium | Partial | clrsig can assert error response structure (machine-readable `code`, `message`); actual ARIA/WCAG validation is out of scope |
| [FM-026](https://clrsig.com/failures/fm-026/) | HTTP Content-Negotiation Failures | Easy | Ready | None — scenario sends `Accept: application/xml`, asserts 406 or checks `Content-Type` response header; easy sub-project |
| [FM-031](https://clrsig.com/failures/fm-031/) | CORS and Browser-Based API Access Failures | Easy | Ready | None — header assertion on `Access-Control-Allow-Origin`; quick sub-project, high visibility |
| [FM-032](https://clrsig.com/failures/fm-032/) | TLS/mTLS and Certificate Lifecycle Failures | Medium | Partial | clrsig supports TLS skip-verify and standard TLS; mTLS (client certificates) not yet supported — needs adding to executor |
| [FM-033](https://clrsig.com/failures/fm-033/) | API Gateway Routing and Policy Drift Failures | Medium | Ready | None — multi-environment scenario (local vs staging env files); highlights clrsig's environment model |
| [FM-035](https://clrsig.com/failures/fm-035/) | Request Header Handling and Validation Failures | Easy | Ready | None — scenario sends correlation headers, asserts they are echoed back or handled correctly |
| [FM-036](https://clrsig.com/failures/fm-036/) | Request Size and Resource Exhaustion Failures | Medium | Ready | None — scenario sends oversized body, asserts 413 is declared in spec and returned correctly |

### Category: Time, State, and Async (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-017](https://clrsig.com/failures/fm-017/) | Stale Cache and Cache-Invalidation Failures | Hard | Partial | Multi-step write-then-read scenario works; asserting `Cache-Control` headers is ready today; detecting stale data requires a timing step |
| [FM-019](https://clrsig.com/failures/fm-019/) | Webhook Delivery and Event-Processing Failures | Hard | Needs feature | clrsig is a client tool; receiving and asserting on inbound webhooks requires a listener mode not yet implemented |
| [FM-020](https://clrsig.com/failures/fm-020/) | Distributed Transaction and Partial-Workflow Failures | Hard | Partial | Multi-step scenario can simulate (create partial state, assert inconsistency); multiple cooperating servers complicate setup |
| [FM-028](https://clrsig.com/failures/fm-028/) | Long-Running Operation and Asynchronous API Failures | Medium | Partial | Scenario can poll a job-status endpoint; a dedicated `retry` step with `until` condition would make this much cleaner |
| [FM-029](https://clrsig.com/failures/fm-029/) | Bulk API Partial-Success Failures | Medium | Ready | None — jsonpath assertions on per-item error fields in batch response; straightforward sub-project |
| [FM-038](https://clrsig.com/failures/fm-038/) | Eventual Consistency and Read-After-Write Failures | Medium | Partial | Multi-step write-then-read; `retry.maxAttempts` + `retry.delay` on the read step covers this cleanly today |
| [FM-039](https://clrsig.com/failures/fm-039/) | Out-of-Order Event and State Transition Failures | Hard | Needs feature | Requires event ordering and async delivery; outside clrsig's current synchronous request model |
| [FM-050](https://clrsig.com/failures/fm-050/) | Environment Configuration and Feature-Flag Drift | Medium | Ready | None — multi-environment scenario; different env files expose different behaviour; natural clrsig showcase |

### Category: Distributed Systems and Infrastructure (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-040](https://clrsig.com/failures/fm-040/) | Database Transaction and Atomicity Failures | Hard | Partial | Scenario verifies post-error state via read-back assertions; actual DB atomicity requires a stateful server and careful setup |
| [FM-041](https://clrsig.com/failures/fm-041/) | Retry Storm and Cascading Failure | Hard | Needs feature | Requires load/concurrency simulation; clrsig is single-threaded by design; out of scope without a load-testing mode |
| [FM-042](https://clrsig.com/failures/fm-042/) | Dependency Failure and Fault Isolation | Hard | Partial | Scenario can simulate a degraded dependency (server returns 503); two cooperating servers complicate setup |
| [FM-048](https://clrsig.com/failures/fm-048/) | Streaming and Large-Response Handling Failures | Hard | Needs feature | clrsig reads the full response body; streaming/chunked transfer assertion requires a streaming response mode |

### Backlog Summary

| Feasibility | Count | Failure modes |
|---|---|---|
| **Ready** — build now | 16 | FM-006, FM-007, FM-013, FM-015, FM-023, FM-024, FM-026, FM-029, FM-031, FM-033, FM-035, FM-036, FM-043, FM-046, FM-047, FM-049, FM-050 |
| **Partial** — small improvement unlocks it | 17 | FM-002, FM-008, FM-011, FM-012, FM-014, FM-017, FM-020, FM-021, FM-028, FM-030, FM-032, FM-037, FM-038, FM-040, FM-042, FM-045 |
| **Needs feature** — significant clrsig work first | 5 | FM-009, FM-019, FM-039, FM-041, FM-048 |

The 16 "Ready" modes are the highest-value additions after v1 ships — no engine work required, just sub-project authoring. The "Partial" group doubles as a roadmap input for the clrsig engine: each one names a concrete, motivated improvement.

---

## 9. Open Questions

| Question | Options | Decision needed by |
|---|---|---|
| Pin to a specific clrsig release or track `@latest`? | Pin = stable catalog; latest = exercises the action on every release | Before CI scaffolding |
| Ship `fix/` for all 12 in v1, or optional? | Required for all = consistent; optional = faster first release | Before sub-project work starts |
| Add a `clrsig conform` run alongside scenario runs where applicable? | More coverage; slightly more CI time | Can be added in v1.1 |
