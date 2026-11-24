# Clear Signal 0.9.0 — release notes (draft, not tagged)

**Read this first: two changes can turn a green CI job red.**

1. **Format checks.** `format: email`, `uuid`, `ipv4` and `ipv6` values are now checked in responses, generated requests and the `schema` assertion (`date`, `date-time` and `byte` already were). A server that returns `"mail": "nope"` now fails `clrsig conform` with `CS-RESP-007`. Other formats (`uri`, `hostname`, ...) are accepted as before. A spec whose own `example` breaks one of these formats still loads; its examples are then not validated.
   *Adopt gradually:* `clrsig conform ... --write-baseline conform-baseline.yaml` once, then `--baseline conform-baseline.yaml` in CI; only new violations fail.
2. **Baseline format 2.** A baseline now records the check (`CS-RESP-005`), operation and field instead of the validator's message, so upgrading clrsig no longer turns known violations into new ones. A baseline written by 0.8.9 or earlier still works (the summary says it is old); run `--write-baseline` once to upgrade it. Older clrsig cannot read the new file.

## New

- **Severity levels.** Every finding is a failure, a warning or a note. Warnings are shown (`WARN`, `PASSED with N warnings`, `severity` in JSON, a warning chip in the app) and fail the run only with `clrsig conform --fail-on warning` (GitHub Action input `conform-fail-on`, MCP `failOn`, *Fail on warnings* on the Check Conformance page). No existing check is a warning in this release.
- **Check catalog as data.** `packages/core/checks` lists every check with its ID and default severity; tests keep the code, the list and *What We Check* in agreement.

## Before tagging

- [ ] CI green on the exact commit (`docs/process/release-process.md`).
- [ ] Set the date of the 0.9.0 entry in `AboutDialog.tsx` (now `unreleased`).
- [ ] Owner has confirmed the open decisions in `docs/enhancements/2026-11-24-severity-and-check-ids.md` section 9.
