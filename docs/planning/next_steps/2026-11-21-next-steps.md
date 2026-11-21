# Next Steps — 2026-11-21 (after v0.6.0)

v0.6.0 is out (GitHub Release with all 13 files, Docker image `v0.6.0`, Homebrew and Scoop manifests bumped). This snapshot replaces the order in [2026-11-19-next-steps.md](2026-11-19-next-steps.md), which stays as the record of what was planned then. The living list of open work is `docs/planning/backlog.md`; this file only says what to do first and why.

## What happened since the 2026-11-19 snapshot

| Then | Now |
|---|---|
| 1 Desktop conformance entry point + conform docs | **Done in 0.6.0**: the Check Conformance page, the Conformance Server, read-only runs, plain-language failures (what / where / why / fix), the What We Check catalog. |
| 2 Hosted demo/test server + everything workspace | Not started (B-031, needs a hosting decision). The local Demo and Conformance Servers now work with the Check Conformance page and Contracts. |
| 3 Community scaffolding | Not started (B-032). |
| 4 Video user guides | Not started (B-033). |
| 5 User traction | Not started (B-034). 0.6.0 is the first release worth posting about. |
| 6 Code signing + notarization | Still **postponed** by the owner until real feedback (B-035). |
| 7 Remaining backlog | Partly done: the review items of 2026-11-20 and 2026-11-21 (B-051 to B-077) shipped in 0.6.0. |

New in 0.6.0 that was not on the list: the Workspace page (B-055), secret redaction in every report, dynamic values and scenario variables, the CLI Command Builder, running scenarios from folders.

## Priority order

| # | Item | Why this position |
|---|---|---|
| 1 | **Use 0.6.0 and collect feedback** (post it, watch issues) | Every later choice (signing, hosted demo, UI polish) is cheaper to decide with a few real reports. Community scaffolding (B-032) is the one-hour prerequisite so feedback has somewhere to land. |
| 2 | **A sidebar and a last tab on every rail icon (B-081)** | Owner request; the rail is the one place the UI is still inconsistent. Proposal written (`docs/enhancements/2026-11-21-rail-sidebars-and-last-tab.md`); four decisions open. Absorbs most of B-068. |
| 3 | **GitHub Action inputs for the newer conform flags (B-058)** | The only open Low item that users of the Action hit directly; the CI guide documents the workaround today. |
| 4 | **Scenario result explains failures like the conformance page (B-067)** | The same failure reads two ways in two places; the finding data already exists. |
| 5 | **Request screen layout (B-069), UI polish (B-070)** | The URL field is narrow at 1280 px; the rest is wording and layout. |
| 6 | **Hardening (B-061, B-063, B-064, B-078)** | All Low: temporary workspace cleanup across app instances, CORS on the dev servers, path checks on the conformance page, symlinks. |
| 7 | **Hosted demo + everything workspace (B-031), video guides (B-033)** | Needs a hosting decision; videos should be recorded after item 2 so they show the final layout. |
| 8 | **Check IDs and severity levels (B-049, B-050)** | Large and touches every report; wait for feedback on what users actually want to be able to quote or soften. |
| 9 | **Code signing + notarization (B-035)** | Postponed by the owner until feedback says unsigned downloads are a real barrier. |

## Notes for the next release (0.7.0)

- Release notes in `AboutDialog.tsx` for 0.6.0 were written before the last UI fixes (rail follows tabs, Workspace tab opens first, Run All and Contracts fixes). The GitHub Release for v0.6.0 has the full notes; start the 0.7.0 entry from the final state of `main`.
- The rail rules shipped in 0.6.0 (page tabs close the sidebar, returning from a page reopens it) are expected to be replaced by B-081; do not build more on them.
- Decide B-080 (Contracts store key collisions) before the Contracts store gets any new fields: a key change needs a migration.
