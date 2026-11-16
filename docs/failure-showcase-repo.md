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

## 8. Open Questions

| Question | Options | Decision needed by |
|---|---|---|
| Pin to a specific clrsig release or track `@latest`? | Pin = stable catalog; latest = exercises the action on every release | Before CI scaffolding |
| Ship `fix/` for all 12 in v1, or optional? | Required for all = consistent; optional = faster first release | Before sub-project work starts |
| Add a `clrsig conform` run alongside scenario runs where applicable? | More coverage; slightly more CI time | Can be added in v1.1 |
