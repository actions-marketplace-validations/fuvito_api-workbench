# Next Steps — 2026-11-19

*Claude: edited after checking the code. `clrsig conform` and JUnit output already exist, so the old §3 and the backlog were rewritten, the item order changed, and the signing, hosting and traction details corrected.*
*2026-11-19 — reordered after owner decision: code signing is postponed until the application gets real user feedback. Signing moved to the bottom (still planned, not dropped); community scaffolding promoted because feedback needs a landing zone before traction drives visitors.*

The app is in a good state: Phase 5 (public MVP) is complete, releases ship to five channels, and the remaining work is about making the product trustworthy to install, easy to evaluate, and visible to users. The current goal is feedback: ship the missing headline UI piece, give evaluators something real to try, put it in front of users, and let what they say decide when signing and heavier promotion are worth the spend.

## Priority order

| # | Item | Why this position |
|---|---|---|
| 1 | Desktop conformance entry point + conform docs | `clrsig conform` already exists in the CLI; the missing pieces are a desktop entry point and current docs. Small, and it is the headline demo for everything below |
| 2 | Hosted demo/test server + everything workspace | Shared infrastructure: needed for pre-release verification, video demos, and lets evaluators try the app without standing up a local server |
| 3 | Community scaffolding (feedback channels) | Cheap (~1h). Feedback needs somewhere to land — Discussions, issue/PR templates — before traction drives outside visitors |
| 4 | Video user guides | Depends on 1–2 (videos should demo the real conform flow and a live server); cheap to redo if scripts/screenshots are kept |
| 5 | User traction: posts, shorts, forums | Doubles as the feedback-gathering step that un-postpones signing. Unsigned-download friction is accepted during this phase |
| 6 | Code signing + notarization | **Postponed** (owner decision) until feedback arrives — still planned, not dropped. See §6 |
| 7 | Remaining backlog | Failure-catalog repo, demo GIF refresh, v1.0 criteria — valuable but not blocking |

---

## 1. Desktop conformance entry point + conform docs

**Status today:** `clrsig conform <spec-file-or-url>` is implemented, with `--baseline` / `--write-baseline`, `--report`, and text/json/junit/html/markdown output, alongside `coverage`, `diff`, `impact` and the MCP server. `docs/contract-testing-roadmap.md` (dated 2026-11-06) predates all of this and still describes conformance as unassembled. The desktop app has no conform entry point (to be confirmed).

**What it involves:**
- A desktop entry point that runs generated requests against a base URL and reports per-operation conformance. This is a new UI area, so it needs a `docs/enhancements/` proposal first per `CLAUDE.md`.
- Update or retire `docs/contract-testing-roadmap.md`.
- A short "conform in CI" guide using the GitHub Action.

**Why first:** it is the product's headline capability (API Contract Validation and Spec Conformance Testing are the canonical terms), the CLI half is done, and it gives the videos and posts something concrete to show.

## 2. Hosted demo/test server + everything workspace

**Status today:** `examples/server/main.go` already implements a local demo server (CRUD users, Bearer auth, failed-login rate limiting, `openapi.yaml`). It runs locally on `:8080` only.

**What it involves:**
- Deploy `examples/server` to a live host (free tier on Fly.io/Render, or a small VPS — keep `go run ./examples/server` working for local use). The hosted build must disable `/admin/reset`, cap body sizes, and reset its data on a schedule, since anyone can reach it.
- Extend the existing `demo_workspace/` into a **kitchen-sink demo workspace** that exercises every feature: all auth types, all assertion types (status/header/jsonpath/schema), variable extraction and dynamic values, fail policies, environments — the "Demo workspace" idea, but deliberately exhaustive rather than minimal.
- Add a step to `docs/release-process.md` §2: run the everything workspace against the live server before tagging. Keep the local-server run as the blocking gate and treat the live run as advisory, so a network outage cannot block a release.
- This workspace doubles as the video demo target (§4) and seeds the failure-catalog repo (§7).

**Effort:** small. The server exists; the work is hosting config, hardening, extending one workspace directory, and a release-checklist line.

## 3. Community scaffolding (feedback channels)

**What it involves:**
- `CONTRIBUTING.md`, issue templates, PR template.
- Enable GitHub Discussions as the primary feedback channel.
- A "feedback" link in the README (and optionally in the app's Help menu) pointing at Discussions.

**Why this position:** it is about an hour of work, and item 5 is now explicitly a feedback-gathering exercise. Feedback with nowhere to land is feedback lost. Promoted from the old §6 backlog for that reason.

## 4. Video user guides (NotebookLM)

**Plan:** three videos — general overview, Scenarios in the desktop UI, CLI functionality — generated with Google NotebookLM (video overviews) or a screen-recording tool for anything NotebookLM can't narrate accurately.

**Obsolescence concern (raised by owner):** the app is still changing, so videos will age. Mitigation — treat each video as a *regenerable artifact*:
- Keep a `docs/video/` folder with one script + shot list per video (exact clicks, files, commands used).
- Scripts should reference the `@screenshot-*` test names rather than copy UI text, so they break loudly when the UI changes.
- Keep a screenshots manifest; `apps/desktop/e2e` already has `@screenshot` Playwright tests that regenerate `docs/images/` — reuse that to refresh visuals quickly.
- When the UI changes enough, re-run the script and regenerate rather than editing the old video.
- Prefer shorter videos/shorts — cheaper to remake, better for §5 anyway.

**Order note:** make videos *after* the conform entry point and the live demo server, so the video shows the real feature hitting a real URL. Signing no longer gates videos — the audiences for these are evaluators, not installers.

## 5. User traction

**Channels worth the effort:**
- Written posts: dev.to / Medium / personal site (clrsig.com) — "validate your API against its OpenAPI spec in CI" is the hook; contract-validation and breaking-change detection angles.
- Shorts/slices of the §4 videos on YouTube.
- Forums: r/softwaretesting, r/qualityassurance, r/api, OpenAPI community spaces. Show HN is deferred until signing lands — HN's audience punishes unsigned downloads, but Reddit/OpenAPI communities are more forgiving and better for early feedback.
- Back every claim with evidence before posting (for example `docs/clrsig-vs-postman-2026-11-15.md`).
- The failure-catalog repo (§7) is itself a traction asset — "watch clrsig catch a real contract violation in CI" is a better demo than a feature list.

**Feedback loop:** every post should point at GitHub Discussions (§3). The specific things to listen for: whether the install warnings scare people off (that's the trigger to un-postpone signing), whether the conform story lands, and what evaluators trip over first.

## 6. Code signing and notarization — POSTPONED

**Owner decision (2026-11-19):** deferred until the application has real user feedback. Still planned — this section keeps the full plan so it can be picked up without re-research.

**Status today:** `release.yml` produces unsigned binaries for all platforms (Windows, macOS, Linux, CLI binaries, Docker image, plus the Homebrew/Scoop update job). Windows shows SmartScreen warnings; macOS Gatekeeper blocks unsigned/unnotarized apps outright for most users.

**What it involves:**
- **Windows:** Azure Trusted Signing (Microsoft's cloud signing service, ~$10/mo). Requires an Azure account + identity validation, then a signing step in `release.yml` via the `azure/trusted-signing-action`.
- **macOS:** Apple Developer Program membership ($99/yr), a Developer ID Application certificate, and notarization (`codesign` + `xcrun notarytool` + `stapler`) run on the `.app` before the `ditto` zip step in `release.yml`.
- Sign the Windows CLI exe as well as the desktop installer; the Docker image is not covered by this item.
- Secrets stored in GitHub repo secrets; no certificate files committed.

**When to resume:** as soon as feedback says the product is worth installing — or if feedback specifically reports install warnings as the blocker. Identity validation for Azure Trusted Signing has human-in-the-loop delays, so starting the *account/paperwork* side early is cheap optionality even while the CI work waits.

## 7. Remaining backlog

Items that matter but don't gate the above:
- **Failure-catalog repo** — full PRD exists at `docs/failure-showcase-repo.md`: public repo of deliberately broken APIs, green CI against released clrsig + nightly against `main`. Doubles as regression testing and marketing.
- **Demo GIF refresh** — `docs/demo.tape` + `vhs` regenerates `docs/demo.gif`; keep it in sync with UI changes.
- **v1.0 criteria** — decide what separates 0.x from 1.0 (likely: signed builds + desktop conform + a documented `schemaVersion` migration policy and a frozen CLI flag surface + some traction evidence).

---

## Notes for the next snapshot

- Signing is postponed, not cancelled — it re-enters the list at the top the moment feedback justifies it, and Show HN waits for it.
- Items 2 and 4 share artifacts: the everything workspace is both the release gate and the video demo set.
- Watch for install-warning complaints in feedback; that signal moves §6 back up.
