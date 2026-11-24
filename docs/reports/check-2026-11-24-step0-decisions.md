# Step 0 decisions — 2026-11-24

Source: section 11 of [`check-2026-11-24-improvement-ideas.md`](check-2026-11-24-improvement-ideas.md). All six were decided by the owner on 2026-11-24, each taking the recommended option. This file is a point-in-time record; the backlog rows carry the follow-up.

| # | Decision | Outcome | What it unlocks | Backlog |
|---|---|---|---|---|
| D1 | Baseline key | **Check ID + operation (method, path template) + short detail.** Old baselines are upgraded in memory on first read; `--write-baseline` writes the new version. This is a breaking change of the baseline schema (`schemaVersion` `"2"`), handled by the migration policy. | B-049, B-006, B-007, idea 19 | B-049, B-006, B-007 |
| D2 | Severity levels | **Three levels: failure, warning, note.** Only failures change the exit code; `--fail-on warning` promotes. Reuse `diff`'s vocabulary. A level may only be given to a finding that has a catalog ID. Ship together with the format validators in one release note, with the adoption path (`--write-baseline`). | B-050, B-053 | B-050, B-053 |
| D3 | Migration policy | **Write it before any new persisted file.** Draft: [`docs/process/schema-migration-policy.md`](../process/schema-migration-policy.md), awaiting review. | B-089 (project config), scenario tags (B-022), data files (B-088), D1 | B-037 |
| D4 | Spec lint | **Part of `clrsig validate` plus a card on the Check Conformance page**, not a new command. | B-086, idea 8 | B-086 |
| D5 | Failure-catalog repository | **Reuse the sandbox repository.** No third repository; remaining work is CI wiring and generating the guide's counts. | B-036 | B-036 |
| D6 | Contracts store key | **Key by the resolved file when it exists; hash only for URLs.** Keeps one card for one spec spelled two ways (the demo workspace and the Workspace page rely on it) and separates different specs. Needs a migration note: existing entries are renamed on first read, and an entry that cannot be matched is kept. | B-080, B-010, idea 22 | B-080 |

## Consequences to plan for

- **D1 and D2 together touch exit codes and baselines.** The proposal for step 1 (`docs/enhancements/2026-11-xx-severity-and-check-ids.md`) must cover both and state the upgrade path of existing baselines.
- **D1 is the first breaking schema change under the new policy**, so it is also the test of the policy's migration steps (function, golden files, `clrsig migrate`, release notes).
- **D6 changes stored keys**, so it falls under the "recorded data" rule (never fail to start; keep what cannot be upgraded).
- **D4** removes the "own command" option from the step 1 proposal; the lint is an extra section of `validate` output and the page.

## Next

1. Review the draft migration policy and adjust.
2. Write the step 1 proposal (severity levels, check IDs, baseline key).
3. Then start step 1 (0.9.0) in the order given in section 11.
