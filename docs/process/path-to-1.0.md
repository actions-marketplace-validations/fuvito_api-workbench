# What separates 0.x from 1.0

Status: **Proposed criteria, for the owner to accept or change** (backlog B-037, written 2026-11-24). Nothing here is built because of this document; it names what "1.0" would promise and where each promise stands today.

## What 1.0 would promise

1.0 is a promise to people who put Clear Signal in a pipeline: **what passes today passes tomorrow unless you upgrade on purpose, and an upgrade never silently changes what a file means.** That is a promise about four surfaces:

1. **The files you write** (requests, scenarios, environments, bundles, the project file, baselines): readable by every 1.x release, with migrations for breaking changes.
2. **The command line:** command and flag names, exit codes and the shape of the JSON report.
3. **The checks:** a check ID keeps its meaning; a new check never makes an existing green run red without a documented way to adopt it.
4. **The release:** signed, reproducible binaries and a stable GitHub Action.

## Criteria and where they stand

| # | Criterion | Today (0.10.0) | To do before 1.0 |
|---|---|---|---|
| 1 | **A schema migration policy** is accepted and followed | Written ([schema-migration-policy.md](schema-migration-policy.md)); the machine-written clause is accepted; the rest awaits review. It has been followed for the baseline (v1 to v2) | Owner accepts the whole policy; `clrsig migrate` exists the first time a hand-written format needs a breaking change (not before) |
| 2 | **A frozen command-line surface**: documented commands, flags and exit codes change only in a minor release with a deprecation first, and are removed only in a major | `TestCatalogMatchesCobra` pins the flags the builder knows; behaviour changes so far are listed in release notes | Write the compatibility rule (below); keep a golden list of commands and flags checked by a test, so a removal fails CI |
| 3 | **The JSON report is an interface**: fields are added, never renamed or removed in 1.x | Fields have only been added since 0.8 | Document the report schemas in `format-spec.md`; golden-file tests per command |
| 4 | **Check IDs are stable** and every check is in the catalog | Done: IDs in `packages/core/checks`, guard tests keep the catalog and the code equal | Decide how a retired check is handled (the ID stays, marked retired) |
| 5 | **A new check never turns a green run red without a way in** | Practised (0.9.0 formats): release notes lead with it, `--baseline` adopts it | Make it a rule in the release process: such a check ships with a warning level first (`--fail-on warning` exists) |
| 6 | **Signed builds** (Windows code signing, macOS notarization) | Not done (B-035; postponed until there is real feedback) | Decide and do; without it the desktop installers trigger OS warnings |
| 7 | **The GitHub Action is tested** and pinned by a major tag | Self-tested in CI against the example server; users can pin `@v0.x.y` | Publish a moving `v1` tag; the self-test pins the latest release |
| 8 | **Documentation is complete and checked**: user guide, format spec, catalog, CI guide | Guard tests for the format spec, the catalog, the versions and the Markdown links | Screenshots in the guide (B-013); a docs pass by someone who has not built it |
| 9 | **Cross-platform confidence**: CI runs on the three desktop platforms | CI runs the Go and frontend tests on Linux only; release builds run on all three | Run the CLI tests on Windows and macOS runners (cheap), and a smoke test of the installed desktop app |
| 10 | **Evidence of use**: people other than the author run it in a pipeline and report findings | The sandbox repository and the failure catalog exist; no outside reports tracked yet | A few real users; issues with check IDs from outside (the new templates ask for them) |
| 11 | **Known gaps are listed, not hidden** | `what-we-check.md` section 3, kept up to date by rule | Review it for anything a 1.0 user would be surprised by |

## The compatibility rule proposed for 1.x

- **Minor release (1.x.0):** may add commands, flags, report fields, check IDs, file fields (additive), and may change the default of nothing. A flag or command is deprecated by printing a warning for at least one minor release before a major removes it.
- **Patch release (1.x.y):** fixes only; a fix that makes a check stricter counts as a new check and ships in a minor release with the adoption path.
- **Major release (2.0):** may remove deprecated flags and bump a file `schemaVersion` with a migration.

## What is deliberately not required for 1.0

Hosted services, accounts, team features, an OS credential store, GraphQL, WebSockets, gRPC, load testing and mock servers are out of scope (`CLAUDE.md` section 9); they are not part of the promise.

## Decision for the owner

Accept this list, or change it. Recommendation: **accept as it stands, aim for 1.0 when criteria 1, 2, 3, 6 and 10 are met** (they are the ones that cost a user real trouble if missed); the others can land before or just after. Until then 0.x releases keep the same discipline, so the move to 1.0 is a decision about promises, not a rewrite.
