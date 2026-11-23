# Next steps after 0.7.0

**Date:** 2026-11-21 (written when 0.7.0 was prepared)
**Replaces:** [`2026-11-21-next-steps.md`](2026-11-21-next-steps.md), which stands as the record of what was planned after 0.6.0.

## What happened since the last snapshot

- **Done in 0.7.0:** B-081 (a sidebar and a last tab for every rail icon, three phases), B-058 (Action inputs), B-061, B-063, B-064, B-078 (hardening), B-065 (`.gitattributes`), B-067 (scenario results explain violations), B-068, B-069, B-070 (UI consistency and polish, mostly), B-079, B-012.
- **Decided:** commercial and enterprise ideas go to [`docs/enterprise/`](../../enterprise/README.md), not the backlog; the first note covers teams that own part of an API.
- **Carried over deliberately:** B-080 (Contracts store keys), see below.

## Priority order

| # | Item | Why |
|---|---|---|
| 1 | **Feedback on 0.7.0** (the rail sidebars, the workspace-only rule on the Check Conformance page) | New interaction model and one behaviour change; real use will show what is missing |
| 2 | **Code signing and notarization (B-035)** | Every release still warns on first run; needs accounts and money |
| 3 | **B-080: key the Contracts store by the resolved spec file** | Needs a decision: a hash of the spelling would split spellings of one file (the Demo Server relies on them sharing a card); keying by the resolved file needs the workspace in `coverage.Key` and a migration |
| 4 | **B-070 leftovers:** one entry point per place for Settings and Help, shorter Settings placeholders | Small polish |
| 5 | **B-006, B-007, B-008 (conformance baselines in the page, baseline paths, a CLI counterpart of "add to Contracts")** | Finish the desktop conformance story |
| 6 | **B-053 (formats such as `email`, `uuid`), B-046, B-047 (sample values and import scaffolds)** | Accuracy of what we check; each needs an entry in *What We Check* |
| 7 | **Distribution checks:** Homebrew install, the Action at the new tag | Only verifiable after a release exists |
| 8 | **Traction and community (B-032, B-034)** | After the product is steady |

## Not planned here

Enterprise features (multi-team ownership scopes, org rollup, identity, audit) are written up in `docs/enterprise/` and are not part of this project's plan unless the owner moves one across.
