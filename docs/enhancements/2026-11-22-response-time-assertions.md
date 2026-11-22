# Response-time assertions

**Status:** Accepted (2026-11-22); phase 1 built in 0.8.0. **Source:** external finding F-018 (the sandbox's `docs/clrsig-findings.md`), catalog gap CS-GAP-005, backlog B-082.

## 1. Summary

A response that takes four seconds, or that gets slower with every call, passes today as long as it arrives before the request timeout. The JSON report already carries `durationMs`, but nothing can fail on it. The only workaround is a short `--request-timeout`, which also aborts the request (no response is left to compare) and applies to every operation alike.

## 2. Goals / non-goals

**Goals**
- A scenario assertion on the response time of a step.
- A `conform` limit on response time that keeps the response, so it is still compared with the spec.
- The duration recorded where runs are already recorded, so a run that "gets slower" can be seen.

**Non-goals**
- Load testing, percentiles over many requests, or a performance dashboard (out of scope in `CLAUDE.md` §9).
- Network-level timing breakdowns (the executor already measures DNS, connect, TLS, first byte and transfer; this proposal uses only the total).

## 3. Requirement review

- **Consistent with `CLAUDE.md`:** the executor already measures duration; the assertion is evaluated in the Go core (`assertions` package), like the others. No new dependency.
- **File format:** a new assertion `type` and value set is a change to the scenario format, so `help/format-spec.md` changes in the same commit; the field is additive (`schemaVersion` stays `"1"`), and an older clrsig rejects a file that uses it.
- **Larger than it looks:** the `conform` flag touches `application.ConformInput`, the CLI, the Action, the CLI Command Builder catalog and the Check Conformance page. The assertion alone is small.
- **Not unnecessary:** nothing else in the product can turn slowness into a failure without also aborting the request. **Smaller alternative:** ship only the scenario assertion first (phase 1).

## 4. Design

**Scenario assertion** (phase 1)

```yaml
assertions:
  - type: duration
    operator: lt        # eq, ne, gt, lt (the existing operator set); lt is the useful one
    expected: 500       # milliseconds; {{variables}} work as in other assertions
```

- Evaluated against `Response.Duration` of the attempt that produced the final response (with `retry`, the last attempt).
- The failure message says the limit and the measured time (`took 812 ms, limit 500 ms`).

**`conform` limit** (phase 2)

- `--max-response-time <ms>` (a limit for every operation) and an optional per-operation override by tag or path prefix, reusing the existing filters (`--max-response-time 500 --max-response-time-for /reports=2000` is one shape to decide).
- A slow response is a violation of its own kind (`slow-response`, a new catalog ID), reported next to, not instead of, the spec comparison: the response is kept and compared.
- Probes are not timed.

**Recording** (phase 3)

- Baselines record the duration class of a violation only (never a timestamp, so diffs stay stable).
- The Contracts store keeps, per operation, the last and the slowest observed duration, shown in the operation table and exported in the existing reports. Needs a decision on the store format (additive field).

## 5. Phasing and acceptance

1. **Assertion:** `type: duration` in the format, validation (numeric, non-negative), evaluator, reports, desktop step editor, format spec, AI prompt, catalog row.
2. **`conform --max-response-time`:** application input, CLI, catalog of the builder, Action input, the page, catalog row.
3. **Recording:** Contracts store and baselines.

Each phase is releasable on its own.

## 6. Testing

Real local servers with a deliberate delay (`httptest`); an assertion that passes and fails; a limit applied to `conform` with the spec comparison still present; stable output in baselines.

## 7. Documentation impact

User guide (assertions, `conform`), `format-spec.md`, AI prompt (guard test), `what-we-check.md` (CS-GAP-005 becomes a check), `README-DEVELOPER.md`, the Action's README and `CLAUDE.md` changelog.

## 8. Risks

- **Flaky by nature:** wall-clock limits fail on a busy CI machine. Mitigation: documentation says to use limits with margin, and the failure message shows the measurement.
- **Format change:** an older clrsig cannot read a scenario with a `duration` assertion.

## 9. Open decisions

1. **Phase order.** Recommendation: build phase 1 first and judge the rest from use.
2. **Shape of the per-operation override** for `conform`. Recommendation: reuse `--paths` and `--tags` selectors rather than a new syntax.
3. **Contracts store.** Recommendation: record last and slowest duration per operation, additive.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-22 | Proposal written from finding F-018; nothing built. | Claude |
| 2026-11-22 | Accepted; build phase 1 first (owner: "yes"). | Owner |
| 2026-11-22 | Phase 1 built in 0.8.0. The operator set stays the existing `eq ne gt lt` (no `lte`/`gte`, which would be new operators for every assertion type); `lt` covers the limit use. | Claude |
