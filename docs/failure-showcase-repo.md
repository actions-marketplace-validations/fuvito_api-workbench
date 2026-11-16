# Failure Modes Showcase Repository

**Purpose:** A standalone GitHub repository containing self-contained sub-projects, each simulating one real-world API contract failure. Every sub-project runs `clrsig` via GitHub Actions and is linked from [clrsig.com/failures](https://clrsig.com/failures/).

---

## Goals

1. **Test clrsig in the wild** — each sub-project is a live regression test for the detection engine. If a failure mode stops being caught, CI turns red.
2. **Prove value to evaluators** — a developer who lands on a failure mode page can clone one sub-project, run it in sixty seconds, and see `clrsig` catch the problem.
3. **Build a searchable failure catalog** — each sub-project becomes a URL on the product site, discoverable by developers debugging the same issue in their own APIs.

---

## Suggested Repository Name

`clrsig-failure-catalog` (public, under the `fuvito` GitHub org)

---

## Repository Structure

```
clrsig-failure-catalog/
├── README.md                       # Index of all failure modes
├── .github/
│   └── workflows/
│       └── catalog.yml             # Runs all sub-projects on push + nightly
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
    └── server-template/            # Reusable Go HTTP server base
```

---

## Sub-project Structure

Each sub-project is self-contained and follows this layout:

```
failures/NN-failure-name/
├── README.md           # What the failure is, how to reproduce it, how to fix it
├── server/
│   └── main.go         # Minimal Go HTTP server that exhibits the failure
├── openapi.yaml        # The declared spec (what the API promises)
├── scenarios/
│   └── detect.yaml     # clrsig scenario that catches the failure
├── environments/
│   └── local.yaml      # baseUrl pointing to the local server
└── fix/
    └── main.go         # Corrected server (optional — shows the passing state)
```

The `fix/` directory is intentional: seeing the broken and fixed versions side-by-side makes the failure concrete and the correction obvious.

---

## Sub-project README Template

Each `README.md` follows this structure:

```markdown
# [NN] Failure name

**Difficulty:** Easy / Medium / Hard
**Detects with:** clrsig conform / clrsig run / clrsig diff

## What this failure is

One paragraph. What breaks, why it happens in practice, what a consumer
experiences when it does.

## How to reproduce it

1. Start the server: `go run server/main.go`
2. Run the scenario: `clrsig run scenarios/detect.yaml --env environments/local.yaml`
3. Expected output: (paste the failing clrsig output)

## Why clrsig catches it

What assertion or schema check fires. Where in the OpenAPI spec the
violation is declared.

## How to fix it

What to change in the server (or the spec, if the spec is wrong).
Point to `fix/main.go` for the working implementation.

## In the wild

One or two lines on when this failure appears in real projects.
```

---

## Planned Sub-projects

Ordered roughly easy → hard.

| # | Name | Difficulty | Detection method |
|---|---|---|---|
| 01 | Missing required field | Easy | Schema assertion on response body |
| 02 | Wrong field type | Easy | Schema assertion (`price` returned as `"12.99"` string, spec says `number`) |
| 03 | Undeclared status code | Easy | `clrsig conform` — status not in spec's responses map |
| 04 | Nullable violation | Easy | Field returns `null` but spec does not declare `nullable: true` |
| 05 | Enum out of range | Easy | Field value outside the spec's declared enum list |
| 06 | Breaking field rename | Medium | `clrsig diff` between v1 and v2 spec files |
| 07 | Breaking field removal | Medium | `clrsig diff` — required field dropped from response schema |
| 08 | Array item type mismatch | Medium | Schema assertion on array response — items are strings, spec says objects |
| 09 | Date format violation | Medium | `format: date-time` field returns a plain date string |
| 10 | Additional properties leak | Medium | Server returns undocumented fields when `additionalProperties: false` |
| 11 | Auth schema drift | Hard | 401 returned on a protected endpoint not declared in the spec's responses |
| 12 | Pagination contract break | Hard | Multi-step scenario — `nextPage` token extracted from page 1 fails on page 2 |

Additional failure modes from [clrsig.com/failures](https://clrsig.com/failures/) are added as new numbered directories as they are built.

---

## GitHub Actions Integration

A single workflow runs all sub-projects. Each step is independent so one failure does not block the others.

```yaml
# .github/workflows/catalog.yml
name: Failure Catalog

on:
  push:
  schedule:
    - cron: '0 6 * * *'   # nightly run to catch regressions

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

      # One step per failure mode
      - name: "01 — Missing required field"
        run: |
          go run failures/01-missing-required-field/server/main.go &
          sleep 1
          clrsig run failures/01-missing-required-field/scenarios/detect.yaml \
            --env failures/01-missing-required-field/environments/local.yaml
        continue-on-error: true   # document the result, don't block the matrix

      - name: "02 — Wrong field type"
        run: |
          go run failures/02-wrong-type/server/main.go &
          sleep 1
          clrsig run failures/02-wrong-type/scenarios/detect.yaml \
            --env failures/02-wrong-type/environments/local.yaml
        continue-on-error: true

      # ... repeated for each sub-project
```

`continue-on-error: true` is intentional for most steps: the whole point of these scenarios is that `clrsig` exits `1` (assertions failed). The nightly run is a regression guard — if a step that _should_ fail unexpectedly _passes_, it means detection broke.

> A future refinement is an inversion wrapper: treat exit code `1` as success (failure detected correctly) and exit code `0` as the failure (detection regressed).

---

## Website Integration

Each sub-project maps to a page at `clrsig.com/failures/<slug>`. The README content (what it is, how to reproduce, how to fix) is the source material for that page.

Workflow:
1. Build the sub-project in this repository.
2. Verify it catches the failure in CI.
3. Add the slug to the failures index on the product site, linking to the GitHub directory for the full reproduction steps.

The catalog CI badge appears on the product site landing page as a live signal that the detection engine is working.

---

## Getting Started (for contributors)

```bash
git clone https://github.com/fuvito/clrsig-failure-catalog
cd clrsig-failure-catalog

# Run one failure mode locally
go run failures/01-missing-required-field/server/main.go &
clrsig run failures/01-missing-required-field/scenarios/detect.yaml \
  --env failures/01-missing-required-field/environments/local.yaml
```

No dependencies beyond Go and the `clrsig` binary. Each sub-project's server is a single-file Go program with no external packages.

---

## Open Questions Before Building

- **Repository visibility:** public from day one (showcase value) or private until at least five sub-projects are solid?
- **Versioning:** should the catalog pin to a specific `clrsig` release tag, or always track `@latest`? Tracking latest exercises the action continuously but could break on a bad release.
- **Inversion CI:** implement the exit-code inversion wrapper in the first pass, or add it later once the basic catalog is running?
