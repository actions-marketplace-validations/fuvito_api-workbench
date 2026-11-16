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
- provides copy-pasteable scenario templates a developer can drop directly into their own workspace
- links each failure pattern to the specific clrsig assertion type that catches it

---

## 2. Goals

| Goal | Description |
|---|---|
| **Test clrsig in production** | Each sub-project is a live regression test against the latest released clrsig. If a failure mode stops being caught, CI turns red. |
| **Guard the development branch** | A second nightly CI leg builds clrsig from `api-workbench@main` and runs the catalog against it, catching engine regressions before they ship. |
| **Prove value to evaluators** | Clone one sub-project, run two commands, see `clrsig` catch the problem in under two minutes. |
| **Feed the product website** | Each sub-project maps to a page at `clrsig.com/failures/fm-NNN/` and is linked from the full catalog at [clrsig.com/failures](https://clrsig.com/failures/). The catalog CI badge appears on [clrsig.com](https://clrsig.com) as a live signal the detection engine is working. |
| **Show "what good looks like"** | The `fixed/` variant in each sub-project demonstrates the corrected spec or server, not just the broken one. |

## 3. Non-goals

- Not a test suite for the clrsig source code (that lives in `fuvito/api-workbench`)
- Not a general-purpose API testing tutorial
- Not covering GraphQL, gRPC, or non-OpenAPI APIs
- **Not the in-product failure-mode catalog** — a future clrsig feature will ship detection templates that auto-apply to a user's spec; this repo is a separate, earlier thing

**Relationship between the three things that share the name:**
```
This repo (executable proofs)
  → clrsig.com/failures/ (website pages, human-readable)
    → future in-product catalog (detection templates auto-applied to your spec)
```

---

## 4. Success Metrics

- 12 sub-projects live, all running in CI, within the first release
- Every sub-project produces `clrsig` exit 1 (failure caught) on the broken server and exit 0 on the fix
- CI badge green on the catalog repo homepage
- Each sub-project linked from its corresponding `clrsig.com/failures/fm-NNN/` page
- README reproduction steps are verified in CI — the commands printed in each sub-project README are literally executed in a dedicated CI step so they cannot silently drift
- Catalog always tests the latest released clrsig version; a Dependabot-style check or manual update task keeps the pinned version current

---

## 5. User Stories

**As a developer evaluating clrsig**, I want to run a real failure example locally in under two minutes so I can see exactly what the tool catches and how the output looks.

**As a developer debugging a spec violation**, I want to find a named failure pattern that matches my situation so I can understand the root cause and the fix.

**As a Clear Signal maintainer**, I want a CI-green catalog that breaks immediately if a detection regression is introduced to the engine.

**As a clrsig user**, I want to copy a `detect.yaml` from this repo into my own workspace, swap in my base URL, and immediately have a working scenario — the scenarios are designed to be lifted, not just run here.

---

## 6. Technical Design

### 6.1 Repository Structure

```
clrsig-failure-catalog/
├── README.md                        # Index — links to every sub-project and clrsig.com/failures/
├── go.mod                           # Single module: github.com/fuvito/clrsig-failure-catalog
├── go.sum
├── .github/
│   └── workflows/
│       ├── catalog.yml              # Runs all sub-projects; nightly (released) + nightly (main branch)
│       └── update-outputs.yml       # Regenerates expected-output blocks in sub-project READMEs
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
└── internal/
    └── testutil/
        └── wait.go                  # HTTP readiness poll helper (used by CI scripts, not servers)
```

**Go module:** `go.mod` lives at the repo root so `internal/testutil` is importable by CI tooling and future test harnesses. The single-file servers under `failures/` import nothing outside stdlib — `go.mod` is there for future use, not a current dependency.

**Port convention:** each sub-project uses a fixed port derived from its FM number: `19000 + NNN`. FM-001 = port 19001, FM-003 = 19003, FM-010 = 19010, etc. This keeps every `environments/local.yaml` static and collision-free, and removes the need for a port-picker that stdlib-only servers cannot use.

Directory names use the FM number as prefix so they sort in catalog order and link unambiguously to `clrsig.com/failures/fm-NNN/`.

### 6.2 Sub-project Structure

Two layouts depending on the failure mode type.

**Server-based** (most failure modes — broken server + fix server):
```
failures/fm-NNN-slug/
├── README.md           # Content for the product page (see template §6.3)
├── broken/
│   └── main.go         # Server that exhibits the failure
├── fixed/
│   └── main.go         # Corrected server — same spec, all assertions pass
├── openapi.yaml        # The declared spec (what the API promises)
├── scenarios/
│   └── detect.yaml     # clrsig scenario that catches the failure
└── environments/
    └── local.yaml      # baseUrl: http://localhost:19NNN
```

**Spec-only** (diff-based failure modes — FM-004, FM-006, FM-007, FM-015):
```
failures/fm-NNN-slug/
├── README.md
├── v1/
│   └── openapi.yaml    # Original spec
├── v2/
│   └── openapi.yaml    # Changed spec that introduces the breaking change
└── scenarios/
    └── detect.yaml     # clrsig diff command and expected violations
```

**Key constraint:** every server is a single Go file with no external dependencies beyond the standard library. No frameworks, no databases. The spec is OpenAPI 3.0 YAML.

### 6.3 Sub-project README Template

```markdown
# FM-NNN — Failure name

> Full catalog entry: https://clrsig.com/failures/fm-NNN/

**Difficulty:** Easy / Medium / Hard
**Detected by:** `clrsig run` / `clrsig conform` / `clrsig diff`
**Port:** 19NNN

## What this failure is

One paragraph. What breaks, why it happens in practice, what a consumer experiences.

## Reproduce it

\```bash
go run broken/main.go &
clrsig run scenarios/detect.yaml --env environments/local.yaml
\```

Expected output *(illustrative — run the command to see current output)*:
\```
(regenerated by CI — see update-outputs.yml)
\```

## Why clrsig catches it

Which assertion fires. Where in the OpenAPI spec the violation is declared.
Link to the relevant section of the [Clear Signal format spec](https://clrsig.com/docs/format-spec/).

## Fix

\```bash
go run fixed/main.go &
clrsig run scenarios/detect.yaml --env environments/local.yaml
# exits 0
\```

What changed: one sentence.

## Adapt this to your API

Replace `baseUrl` in `environments/local.yaml` with your API's address.
The `{{placeholders}}` in `detect.yaml` match your environment variable names.

## In the wild

When and why this pattern appears in real projects.
```

*Devin: the pasted "Expected output" block will drift from real output within a few releases. Either regenerate it in CI (a job that runs the scenario and refreshes the block) or label it explicitly as illustrative. Stale expected output in a repo whose entire purpose is demonstrating correctness is a credibility problem — the same one the user-guide audit just fixed.*

*Claude: labelled as illustrative and added an `update-outputs.yml` workflow to the structure (§6.1) that regenerates the blocks on demand. The task is in §7.2. This is better than marking it static — it stays accurate with no manual work.*

### 6.4 GitHub Actions Workflow

```yaml
# .github/workflows/catalog.yml
name: Failure Catalog

on:
  push:
  schedule:
    - cron: '0 6 * * *'   # nightly — released clrsig
    - cron: '0 7 * * *'   # nightly — clrsig built from api-workbench@main

jobs:
  catalog:
    runs-on: ubuntu-latest
    strategy:
      fail-fast: false   # one failure does not cancel the rest
      matrix:
        fm:
          - { id: "001", slug: "number-precision",        kind: "server" }
          - { id: "003", slug: "pagination-contract",     kind: "server" }
          - { id: "004", slug: "renamed-removed-field",   kind: "diff"   }
          - { id: "005", slug: "nullable-required",       kind: "server" }
          - { id: "010", slug: "spec-drift",              kind: "server" }
          - { id: "016", slug: "rate-limit-schema",       kind: "server" }
          - { id: "018", slug: "idempotency",             kind: "server" }
          - { id: "022", slug: "semantically-invalid",    kind: "server" }
          - { id: "025", slug: "incorrect-status",        kind: "server" }
          - { id: "027", slug: "lost-update",             kind: "server" }
          - { id: "034", slug: "method-semantics",        kind: "server" }
          - { id: "044", slug: "sensitive-data",          kind: "server" }

    steps:
      - uses: actions/checkout@v4

      - uses: actions/setup-go@v5
        with:
          go-version: "1.26"

      - name: Install clrsig
        run: |
          curl -sSfL https://github.com/fuvito/api-workbench/releases/latest/download/clrsig-linux-amd64 \
            -o /usr/local/bin/clrsig && chmod +x /usr/local/bin/clrsig

      # Server-based: build binary first to avoid cold-compile race, then poll for readiness
      - name: Build and start broken server
        if: matrix.fm.kind == 'server'
        run: |
          go build -o /tmp/server-broken \
            ./failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/broken/main.go
          /tmp/server-broken &
          until curl -sf http://localhost:19${{ matrix.fm.id }}/health; do sleep 0.2; done

      - name: "Detect (must exit 1)"
        if: matrix.fm.kind == 'server'
        run: |
          set +e
          clrsig run failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/scenarios/detect.yaml \
            --env failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/environments/local.yaml
          test $? -eq 1

      - name: Stop broken server
        if: matrix.fm.kind == 'server'
        run: pkill -f server-broken || true

      - name: Build and start fixed server
        if: matrix.fm.kind == 'server'
        run: |
          go build -o /tmp/server-fixed \
            ./failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/fixed/main.go
          /tmp/server-fixed &
          until curl -sf http://localhost:19${{ matrix.fm.id }}/health; do sleep 0.2; done

      - name: "Fix (must exit 0)"
        if: matrix.fm.kind == 'server'
        run: |
          clrsig run failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/scenarios/detect.yaml \
            --env failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/environments/local.yaml

      # Diff-based: no server needed
      - name: "Diff (must exit 1)"
        if: matrix.fm.kind == 'diff'
        run: |
          set +e
          clrsig diff \
            failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/v1/openapi.yaml \
            failures/fm-${{ matrix.fm.id }}-${{ matrix.fm.slug }}/v2/openapi.yaml \
            --breaking-only
          test $? -eq 1
```

**Exit code convention:** `set +e` disables bash's exit-on-error so `clrsig` returning 1 does not abort the step. `test $? -eq 1` then asserts that is exactly what happened — if the broken server unexpectedly passes (`exit 0`), CI fails (detection regressed). The fixed server step runs with default `bash -e` and must exit 0.

### 6.5 V1 Sub-project Catalog

12 sub-projects for the first release. All map to pages in the [clrsig.com/failures](https://clrsig.com/failures/) catalog.

**Recommended build order:** ship the four Easy modes first (FM-001, FM-005, FM-010, FM-025) to validate the full pipeline end-to-end before tackling the Hard scenarios.

| Directory | Failure mode | Difficulty | Detection | Port | clrsig.com page |
|---|---|---|---|---|---|
| `fm-001-number-precision` | Currency, Number, and Precision Formatting Failures | Easy | `clrsig run` schema assertion | 19001 | [FM-001](https://clrsig.com/failures/fm-001/) |
| `fm-003-pagination-contract` | Pagination Contract Failures | Hard | `clrsig run` multi-step scenario | 19003 | [FM-003](https://clrsig.com/failures/fm-003/) |
| `fm-004-renamed-removed-field` | Renamed or Removed Response Fields | Medium | `clrsig diff` v1 vs v2 | — | [FM-004](https://clrsig.com/failures/fm-004/) |
| `fm-005-nullable-required` | Incorrect Optional, Nullable, and Required Field Behavior | Easy | `clrsig run` schema assertion | 19005 | [FM-005](https://clrsig.com/failures/fm-005/) |
| `fm-010-spec-drift` | OpenAPI Specification and Implementation Drift | Easy | `clrsig conform` | 19010 | [FM-010](https://clrsig.com/failures/fm-010/) |
| `fm-016-rate-limit-schema` | Rate-Limit and Throttling Failures | Medium | `clrsig run` undeclared 429 | 19016 | [FM-016](https://clrsig.com/failures/fm-016/) |
| `fm-018-idempotency` | Idempotency and Duplicate-Operation Failures | Hard | `clrsig run` multi-step scenario | 19018 | [FM-018](https://clrsig.com/failures/fm-018/) |
| `fm-022-semantically-invalid` | Syntactically Valid but Semantically Invalid Payloads | Medium | `clrsig run` jsonpath assertions | 19022 | [FM-022](https://clrsig.com/failures/fm-022/) |
| `fm-025-incorrect-status` | Incorrect HTTP Status and Business-Error Semantics | Easy | `clrsig run` status assertion | 19025 | [FM-025](https://clrsig.com/failures/fm-025/) |
| `fm-027-lost-update` | Concurrent Update and Lost-Update Failures | Hard | `clrsig run` multi-step scenario | 19027 | [FM-027](https://clrsig.com/failures/fm-027/) |
| `fm-034-method-semantics` | HTTP Method and Operation Semantics Failures | Medium | `clrsig conform` + schema assertion | 19034 | [FM-034](https://clrsig.com/failures/fm-034/) |
| `fm-044-sensitive-data` | Sensitive Data Exposure in API Responses | Medium | `clrsig run` schema (`additionalProperties: false`) | 19044 | [FM-044](https://clrsig.com/failures/fm-044/) |

The remaining 38 failure modes in the catalog ([clrsig.com/failures](https://clrsig.com/failures/)) are candidates for future sub-projects. Any contributor can propose one by opening a pull request using the template in §6.3.

---

## 7. Task List

### 7.1 Repository Setup

- [ ] Create `github.com/fuvito/clrsig-failure-catalog` (public)
- [ ] Add root `README.md` with project purpose, CI badge, and sub-project index table with FM links
- [ ] Initialise root `go.mod` (`module github.com/fuvito/clrsig-failure-catalog`)
- [ ] Add `internal/testutil/wait.go` HTTP readiness poll helper
- [ ] Add `.gitignore` (Go binaries, IDE files)
- [ ] Add `LICENSE` (MIT)
- [ ] Add `CONTRIBUTING.md` — how to add a new failure mode (uses template from §6.3)
- [ ] Document the port convention (`19000 + FM number`) in `CONTRIBUTING.md`

### 7.2 CI Scaffolding

- [ ] Create `.github/workflows/catalog.yml` with matrix strategy and exit-code inversion (§6.4)
- [ ] Add a second nightly schedule leg that installs clrsig built from `api-workbench@main`
- [ ] Wire FM-001 as the first matrix entry to verify the full pipeline end-to-end
- [ ] Create `.github/workflows/update-outputs.yml` — runs all scenarios and refreshes expected-output blocks in sub-project READMEs (triggered manually or on release)
- [ ] Add CI status badge to root `README.md`
- [ ] Add `verify.sh` (bash) and `verify.ps1` (PowerShell) at repo root — wrappers that run the reproduction steps from §6.3 so Windows developers can verify locally without reading bash syntax

### 7.3 Pre-build Decisions (resolve before first sub-project)

- [ ] Confirm `clrsig diff` exit-code behaviour for breaking changes (must be exit 1 for the matrix inversion to work)
- [ ] Decide spec-only layout for diff-based modes — adopt `v1/openapi.yaml` + `v2/openapi.yaml` structure as defined in §6.2
- [ ] Confirm all 12 sub-projects ship with a `fixed/` variant (exit-0 leg is required — without it, exit-1 on the broken server is unverifiable)

### 7.4 Sub-projects

Build order: Easy modes first, then Medium, then Hard.

- [ ] [FM-001](https://clrsig.com/failures/fm-001/) — Currency, Number, and Precision Formatting Failures *(Easy — build first to validate pipeline)*
- [ ] [FM-005](https://clrsig.com/failures/fm-005/) — Incorrect Optional, Nullable, and Required Field Behavior *(Easy)*
- [ ] [FM-010](https://clrsig.com/failures/fm-010/) — OpenAPI Specification and Implementation Drift *(Easy)*
- [ ] [FM-025](https://clrsig.com/failures/fm-025/) — Incorrect HTTP Status and Business-Error Semantics *(Easy)*
- [ ] [FM-004](https://clrsig.com/failures/fm-004/) — Renamed or Removed Response Fields *(Medium — first diff-based mode)*
- [ ] [FM-016](https://clrsig.com/failures/fm-016/) — Rate-Limit and Throttling Failures *(Medium)*
- [ ] [FM-022](https://clrsig.com/failures/fm-022/) — Syntactically Valid but Semantically Invalid Payloads *(Medium)*
- [ ] [FM-034](https://clrsig.com/failures/fm-034/) — HTTP Method and Operation Semantics Failures *(Medium)*
- [ ] [FM-044](https://clrsig.com/failures/fm-044/) — Sensitive Data Exposure in API Responses *(Medium)*
- [ ] [FM-003](https://clrsig.com/failures/fm-003/) — Pagination Contract Failures *(Hard)*
- [ ] [FM-018](https://clrsig.com/failures/fm-018/) — Idempotency and Duplicate-Operation Failures *(Hard)*
- [ ] [FM-027](https://clrsig.com/failures/fm-027/) — Concurrent Update and Lost-Update Failures *(Hard)*

### 7.5 Website Integration

- [ ] Confirm `clrsig.com/failures/fm-NNN/` page structure can link to the GitHub directory
- [ ] Verify each sub-project README renders cleanly as source material for the product page
- [ ] Link each live sub-project from its corresponding `clrsig.com/failures/fm-NNN/` page
- [ ] Add catalog CI badge to [clrsig.com](https://clrsig.com) landing page

### 7.6 Release Gate

- [ ] All 12 sub-projects passing CI (broken exits 1, fixed exits 0)
- [ ] Root README index complete with correct FM links
- [ ] Repository announced on product site and in release notes

---

## 8. Backlog — Future Failure Mode Additions

The 38 failure modes not included in v1, with an assessment of build effort and clrsig feasibility. The "clrsig work needed" column notes engine improvements that would unlock the full scenario — this backlog also serves as roadmap input for the clrsig engine.

Three feasibility ratings:
- **Ready** — clrsig can detect this today with existing scenario/conform/diff commands
- **Partial** — detectable with a workaround or only partially; a targeted improvement makes it clean
- **Needs feature** — requires a new clrsig capability before a useful sub-project can be built

### Category: Data Contracts (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-002](https://clrsig.com/failures/fm-002/) | Silent Timeout and Incomplete Response Failures | Medium | Partial | A `durationLt` step setting would make this clean; `Response.Duration` is already measured in the executor — small addition |
| [FM-006](https://clrsig.com/failures/fm-006/) | Mobile and Older-Client Compatibility Failures | Medium | Ready | None — `clrsig diff` with a "legacy" spec file captures this cleanly |
| [FM-007](https://clrsig.com/failures/fm-007/) | API Versioning and Compatibility Failures | Medium | Ready | None — `clrsig diff v1.yaml v2.yaml` is the primary use case; good showcase |
| [FM-013](https://clrsig.com/failures/fm-013/) | Undocumented Dependencies and Accumulated Integration Failures | Hard | Ready | None — `additionalProperties: false` catches undocumented fields |
| [FM-014](https://clrsig.com/failures/fm-014/) | Timezone and Daylight-Saving-Time Failures | Medium | Partial | `format: date-time` validates format but not timezone offset; the `matches` regex operator fills the gap today |
| [FM-015](https://clrsig.com/failures/fm-015/) | Deprecated Endpoint and Lifecycle Failures | Medium | Ready | None — `clrsig conform` catches 404/410 on removed endpoints; `clrsig diff` flags removals |
| [FM-021](https://clrsig.com/failures/fm-021/) | File Upload and Downstream Processing Failures | Medium | Ready | None — `multipart/form-data` is already implemented in the executor |
| [FM-024](https://clrsig.com/failures/fm-024/) | Localization and Machine-Readable Data Failures | Medium | Ready | None — header assertion on `Accept-Language`, then assert stable machine-readable status fields |
| [FM-047](https://clrsig.com/failures/fm-047/) | Character Encoding and Unicode Handling Failures | Easy | Ready | None — scenario sends Unicode input, asserts round-trip fidelity |
| [FM-049](https://clrsig.com/failures/fm-049/) | API Documentation and Example Drift | Easy | Ready | None — this is precisely what `clrsig conform` was built for; highest-value quick win after v1 |

### Category: Authentication and Security (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-008](https://clrsig.com/failures/fm-008/) | OAuth Implementation and Token-Validation Failures | Hard | Partial | Multi-step scenario covers expired/invalid token assertions today; full OAuth flow (PKCE, refresh) needs an auth-flow step type |
| [FM-009](https://clrsig.com/failures/fm-009/) | Healthcare Interoperability and HL7 Conformance Failures | Hard | Needs feature | Requires FHIR/HL7 schema validator; outside clrsig's current OpenAPI scope |
| [FM-011](https://clrsig.com/failures/fm-011/) | Payment Gateway Certification and Protocol Conformance Failures | Hard | Partial | Schema assertions cover field/type validation; payment-protocol-specific rules (e.g. Luhn check) need a custom assertion type |
| [FM-023](https://clrsig.com/failures/fm-023/) | Broken Object-Level Authorization and Tenant Isolation | Medium | Ready | None — multi-step scenario with two auth tokens; User A cannot read User B's resource |
| [FM-030](https://clrsig.com/failures/fm-030/) | API Key and Client-Credential Rotation Failures | Hard | Partial | Scenario simulates expired key returning 401; full rotation lifecycle needs step-retry + time-based support |
| [FM-037](https://clrsig.com/failures/fm-037/) | Input Injection and Unsafe Interpretation Failures | Medium | Partial | Scenario verifies injection inputs return declared error codes (not 500/200); active exploit detection is out of scope by design |
| [FM-043](https://clrsig.com/failures/fm-043/) | Mass Assignment and Unauthorized Field Modification | Easy | Ready | None — scenario sends extra fields (e.g. `role: admin`), asserts they are absent on read-back |
| [FM-045](https://clrsig.com/failures/fm-045/) | Authentication and Authorization Confusion Across Services | Hard | Partial | Needs two cooperating servers in one sub-project; feasible but complex to set up cleanly |
| [FM-046](https://clrsig.com/failures/fm-046/) | Security Policy and Secret Leakage Failures | Medium | Ready | None — jsonpath assertions confirm no credential fields in error responses; pairs well with FM-044 |

### Category: HTTP and Request Handling (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-012](https://clrsig.com/failures/fm-012/) | Accessibility Conformance Failures in API-Driven Applications | Medium | Partial | clrsig can assert error response structure (machine-readable `code`, `message`); ARIA/WCAG validation is out of scope |
| [FM-026](https://clrsig.com/failures/fm-026/) | HTTP Content-Negotiation Failures | Easy | Ready | None — scenario sends `Accept: application/xml`, asserts 406 or checks `Content-Type` response header |
| [FM-031](https://clrsig.com/failures/fm-031/) | CORS and Browser-Based API Access Failures | Easy | Ready | None — header assertion on `Access-Control-Allow-Origin`; quick sub-project, high visibility |
| [FM-032](https://clrsig.com/failures/fm-032/) | TLS/mTLS and Certificate Lifecycle Failures | Medium | Partial | Standard TLS works; mTLS (client certificates) not yet in the executor |
| [FM-033](https://clrsig.com/failures/fm-033/) | API Gateway Routing and Policy Drift Failures | Medium | Ready | None — multi-environment scenario (local vs staging env files); highlights clrsig's environment model |
| [FM-035](https://clrsig.com/failures/fm-035/) | Request Header Handling and Validation Failures | Easy | Ready | None — scenario sends correlation headers, asserts they are echoed back or handled correctly |
| [FM-036](https://clrsig.com/failures/fm-036/) | Request Size and Resource Exhaustion Failures | Medium | Ready | None — scenario sends oversized body, asserts 413 is declared in spec and returned correctly |

### Category: Time, State, and Async (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-017](https://clrsig.com/failures/fm-017/) | Stale Cache and Cache-Invalidation Failures | Hard | Partial | Multi-step write-then-read scenario works; `Cache-Control` header assertions work today; detecting stale data cleanly needs `durationLt` |
| [FM-019](https://clrsig.com/failures/fm-019/) | Webhook Delivery and Event-Processing Failures | Hard | Needs feature | clrsig is a client tool; receiving and asserting on inbound webhooks requires a listener mode |
| [FM-020](https://clrsig.com/failures/fm-020/) | Distributed Transaction and Partial-Workflow Failures | Hard | Partial | Multi-step scenario can simulate (create partial state, assert inconsistency); multiple cooperating servers complicate setup |
| [FM-028](https://clrsig.com/failures/fm-028/) | Long-Running Operation and Asynchronous API Failures | Medium | Partial | Scenario can poll a job-status endpoint; a `retry` step with `until` condition makes this clean — see engine note below |
| [FM-029](https://clrsig.com/failures/fm-029/) | Bulk API Partial-Success Failures | Medium | Ready | None — jsonpath assertions on per-item error fields in batch response |
| [FM-038](https://clrsig.com/failures/fm-038/) | Eventual Consistency and Read-After-Write Failures | Medium | Partial | `retry.maxAttempts`/`retry.delay` parse and serialize correctly but the runner never reads them — retries are dead config today; fix the runner loop first |
| [FM-039](https://clrsig.com/failures/fm-039/) | Out-of-Order Event and State Transition Failures | Hard | Needs feature | Requires event ordering and async delivery; outside clrsig's synchronous model |
| [FM-050](https://clrsig.com/failures/fm-050/) | Environment Configuration and Feature-Flag Drift | Medium | Ready | None — multi-environment scenario; different env files expose behaviour variance; natural clrsig showcase |

### Category: Distributed Systems and Infrastructure (remaining)

| FM | Title | Build effort | Feasibility | clrsig work needed |
|---|---|---|---|---|
| [FM-040](https://clrsig.com/failures/fm-040/) | Database Transaction and Atomicity Failures | Hard | Partial | Scenario verifies post-error state via read-back assertions; actual DB atomicity requires a stateful server |
| [FM-041](https://clrsig.com/failures/fm-041/) | Retry Storm and Cascading Failure | Hard | Needs feature | Requires load/concurrency simulation; clrsig is single-threaded by design |
| [FM-042](https://clrsig.com/failures/fm-042/) | Dependency Failure and Fault Isolation | Hard | Partial | Scenario can simulate a degraded dependency (server returns 503); two cooperating servers needed |
| [FM-048](https://clrsig.com/failures/fm-048/) | Streaming and Large-Response Handling Failures | Hard | Needs feature | clrsig reads the full response body; streaming/chunked transfer requires a streaming response mode |

### Backlog Summary

| Feasibility | Count | Failure modes |
|---|---|---|
| **Ready** — build now, no engine work | 17 | FM-006, FM-007, FM-013, FM-015, FM-021, FM-023, FM-024, FM-026, FM-029, FM-031, FM-033, FM-035, FM-036, FM-043, FM-046, FM-047, FM-049, FM-050 |
| **Partial** — small improvement unlocks it | 16 | FM-002, FM-008, FM-011, FM-012, FM-014, FM-017, FM-020, FM-028, FM-030, FM-032, FM-037, FM-038, FM-040, FM-042, FM-045 |
| **Needs feature** — significant clrsig work first | 5 | FM-009, FM-019, FM-039, FM-041, FM-048 |

**Highest-leverage engine improvements** (each one unlocks multiple "Partial" modes):
1. **Implement step retry in the runner** — `step.Retry` config already parses and serializes; a small loop in `runStep` makes FM-038 Ready and cleans up FM-028, FM-017, FM-030
2. **Add `durationLt` step assertion** — `Response.Duration` is already measured; one new assertion type closes FM-002 and improves FM-017
3. **Add mTLS support to executor** — closes FM-032

The five "Needs feature" modes (FM-009, FM-019, FM-039, FM-041, FM-048) correctly map to capabilities outside clrsig's intended scope (HL7, inbound webhooks, async ordering, load testing, streaming). Keep them in the catalog as documentation; do not build sub-projects for them until the engine deliberately expands scope.

---

## 9. Decisions

Previously open questions — resolved.

| Question | Decision | Rationale |
|---|---|---|
| Pin to a specific clrsig release or track `@latest`? | **Pin for push builds; track `@latest` for one of the nightly legs** | Pinning keeps PR builds deterministic; a nightly `@latest` leg serves the regression-guard goal |
| Ship `fix/` for all 12 in v1? | **Yes, required for all** | The exit-0 leg is what makes exit-1 meaningful; without it a scenario failing for the wrong reason is indistinguishable from a caught failure |
| Add `clrsig conform` alongside scenario runs? | **Yes, in v1 where a spec is present** | One extra line per step, exercises the flagship command on every commit |
