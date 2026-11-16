# clrsig-failure-catalog — PRD & Technical Design

**Repository:** `github.com/fuvito/clrsig-failure-catalog` (new, public)
**Owner:** Fuat Yazar
**Status:** Pre-build — this document is the source of truth for implementation

---

## 1. Problem

API contract failures are abstract until a developer sees one caught in their own CI pipeline. The clrsig documentation explains *what* the tool detects; this catalog proves it by running against real, broken servers on every commit.

There is currently no public, runnable collection of API failure mode reproductions that:
- works out of the box with a single binary
- shows both the failing state and the fix side by side
- is kept green by CI so it cannot quietly rot

---

## 2. Goals

| Goal | Description |
|---|---|
| **Test clrsig in production** | Each sub-project is a live regression test. If a failure mode stops being caught, CI turns red. |
| **Prove value to evaluators** | Clone one sub-project, run two commands, see `clrsig` catch the problem in under two minutes. |
| **Feed the product website** | Each sub-project maps to a page at `clrsig.com/failures/<slug>` — a searchable catalog of failure patterns with reproduction steps. |
| **Show "what good looks like"** | The `fix/` variant in each sub-project demonstrates the corrected spec or server, not just the broken one. |

## 3. Non-goals

- Not a test suite for the clrsig source code (that lives in `fuvito/api-workbench`)
- Not a general-purpose API testing tutorial
- Not covering GraphQL, gRPC, or non-OpenAPI APIs

---

## 4. Success Metrics

- 12 sub-projects live, all running in CI, within the first release
- Every sub-project produces a `clrsig` exit code 1 (failure caught) on the broken server and exit code 0 on the fix
- CI badge green on the catalog repo homepage
- Each sub-project linked from `clrsig.com/failures/`

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
├── README.md                        # Index — links to every sub-project
├── .github/
│   └── workflows/
│       └── catalog.yml              # Runs all sub-projects; nightly + on push
│
├── failures/
│   ├── 01-missing-required-field/
│   ├── 02-wrong-type/
│   ├── 03-undeclared-status-code/
│   ├── 04-nullable-violation/
│   ├── 05-enum-out-of-range/
│   ├── 06-breaking-field-rename/
│   ├── 07-breaking-field-removal/
│   ├── 08-array-item-type/
│   ├── 09-format-violation/
│   ├── 10-additional-properties-leak/
│   ├── 11-auth-schema-drift/
│   └── 12-pagination-contract-break/
│
└── shared/
    ├── go.mod                        # Single Go module for all servers
    └── testutil/
        └── server.go                 # Shared helpers (port picker, graceful shutdown)
```

### 6.2 Sub-project Structure

Every sub-project is self-contained:

```
failures/NN-failure-name/
├── README.md           # PRD content for the product page (see template §6.3)
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
# [NN] Failure name

**Difficulty:** Easy / Medium / Hard
**Detected by:** `clrsig run` / `clrsig conform` / `clrsig diff`
**Slug:** `failure-name` (maps to clrsig.com/failures/failure-name)

## What this failure is

One paragraph. What breaks, why it happens, what a consumer experiences.

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

When this pattern appears in real projects.
```

### 6.4 GitHub Actions Workflow

```yaml
# .github/workflows/catalog.yml
name: Failure Catalog

on:
  push:
  schedule:
    - cron: '0 6 * * *'

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

      - name: "01 — Missing required field (must fail)"
        run: |
          go run failures/01-missing-required-field/server/main.go &
          sleep 1
          clrsig run failures/01-missing-required-field/scenarios/detect.yaml \
            --env failures/01-missing-required-field/environments/local.yaml
          test $? -eq 1    # exit 1 = clrsig caught the violation — this is correct
```

**Exit code convention:** `test $? -eq 1` treats a clrsig failure (assertions failed) as CI success. If the broken server unexpectedly passes (`exit 0`), CI fails — detection regressed. The `fix/` variant is tested separately and must exit 0.

### 6.5 Failure Mode Catalog

| # | Slug | Difficulty | Detection command | What breaks |
|---|---|---|---|---|
| 01 | `missing-required-field` | Easy | `clrsig run` | Response omits a field declared `required` in the schema |
| 02 | `wrong-type` | Easy | `clrsig run` | `price` returned as `"12.99"` (string); spec declares `number` |
| 03 | `undeclared-status-code` | Easy | `clrsig conform` | Endpoint returns 422; spec only declares 200 and 400 |
| 04 | `nullable-violation` | Easy | `clrsig run` | Field returns `null`; spec does not declare `nullable: true` |
| 05 | `enum-out-of-range` | Easy | `clrsig run` | `status` field returns `"archived"`; enum is `["active","inactive"]` |
| 06 | `breaking-field-rename` | Medium | `clrsig diff` | `full_name` renamed to `name` between v1 and v2 |
| 07 | `breaking-field-removal` | Medium | `clrsig diff` | Required field `email` removed from response schema in v2 |
| 08 | `array-item-type` | Medium | `clrsig run` | Endpoint returns `["a","b"]`; spec declares array of objects |
| 09 | `format-violation` | Medium | `clrsig run` | `created_at` returns `"2024-11-15"` (date); spec declares `date-time` |
| 10 | `additional-properties-leak` | Medium | `clrsig run` | Server returns undocumented fields; spec has `additionalProperties: false` |
| 11 | `auth-schema-drift` | Hard | `clrsig run` | Protected endpoint returns 401; spec does not declare a 401 response |
| 12 | `pagination-contract-break` | Hard | `clrsig run` | Multi-step scenario: `nextPage` token from page 1 is invalid on page 2 |

---

## 7. Task List

Tasks are ordered for sequential execution. Each sub-project (§7.3) can be built in any order once the scaffolding is in place.

### 7.1 Repository Setup

- [ ] Create `github.com/fuvito/clrsig-failure-catalog` (public)
- [ ] Add root `README.md` with project purpose and sub-project index table
- [ ] Initialise `shared/go.mod` (`module github.com/fuvito/clrsig-failure-catalog`)
- [ ] Add `.gitignore` (Go binaries, IDE files)
- [ ] Add `LICENSE` (MIT)
- [ ] Add `CONTRIBUTING.md` — one-page guide: how to add a new failure mode

### 7.2 CI Scaffolding

- [ ] Create `.github/workflows/catalog.yml` with the exit-code inversion pattern (§6.4)
- [ ] Wire one placeholder step (01) to verify the inversion logic works end-to-end
- [ ] Add CI status badge to root `README.md`
- [ ] Confirm the nightly schedule trigger fires correctly

### 7.3 Sub-projects (one task each)

Each task = broken server + spec + scenario + fix + README + CI step.

- [ ] 01 — Missing required field
- [ ] 02 — Wrong type
- [ ] 03 — Undeclared status code
- [ ] 04 — Nullable violation
- [ ] 05 — Enum out of range
- [ ] 06 — Breaking field rename
- [ ] 07 — Breaking field removal
- [ ] 08 — Array item type mismatch
- [ ] 09 — Date format violation
- [ ] 10 — Additional properties leak
- [ ] 11 — Auth schema drift
- [ ] 12 — Pagination contract break

### 7.4 Website Integration

- [ ] Confirm slug naming convention with product site (`clrsig.com/failures/<slug>`)
- [ ] Add `slug:` front-matter field to each sub-project README so the site can consume it
- [ ] Link each live sub-project from `clrsig.com/failures/`
- [ ] Add CI badge to the product site landing page

### 7.5 Release

- [ ] All 12 sub-projects passing CI (broken server exits 1, fix exits 0)
- [ ] Root README index complete and accurate
- [ ] Repository announced on product site and in release notes

---

## 8. Open Questions

| Question | Options | Decision needed by |
|---|---|---|
| Pin to a specific clrsig release or track `@latest`? | Pin = stable catalog; latest = exercises the action continuously | Before CI scaffolding |
| Ship `fix/` variant for all 12 sub-projects in v1, or optional? | Required for all = consistent; optional = faster first release | Before sub-project work starts |
| Add a `clrsig conform` conformance run alongside scenario runs? | More coverage; slightly more CI time | Can be added in v1.1 |
