# Documentation map

Documents live in two places, depending on who reads them and where.

## Served by the app: `apps/desktop/wails/help/`

What a user reads **inside the app** (the Help panel). These files are embedded in the program, so they have to live in that folder; edit them there.

| File | Shown as |
|---|---|
| [`user-guide.md`](../apps/desktop/wails/help/user-guide.md) | Help, User Guide |
| [`format-spec.md`](../apps/desktop/wails/help/format-spec.md) | Help, Format Spec |
| [`example-scenario.md`](../apps/desktop/wails/help/example-scenario.md) | Help, Example Scenario |
| [`ai-prompt.md`](../apps/desktop/wails/help/ai-prompt.md) | Help, AI Prompt |
| [`what-we-check.md`](../apps/desktop/wails/help/what-we-check.md) | Help, What We Check (every failure Clear Signal reports, and what it does not catch) |
| `*.yaml` | the sample specs and environment the Demo and Conformance servers hand out |

## Developer documents: `docs/`

| Folder | What goes there |
|---|---|
| [`enhancements/`](enhancements/README.md) | One dated design proposal per major feature, agreed before it is built (`YYYY-MM-DD-short-name.md`) |
| [`planning/`](planning/) | What is open and what comes next: the [backlog](planning/backlog.md) (edited in place), dated [next-step snapshots](planning/next_steps/README.md), product requirement documents such as the failure-catalog repo and the [sandbox project](planning/api-workbench-sandbox-project.md) (the separate, in-memory API fixtures Clear Signal is tested against, local first and then Cloud Run) |
| [`reports/`](reports/) | Audits and comparisons, written once: `check-YYYY-MM-DD.md` project checks, competitor comparisons |
| [`guides/`](guides/) | How-to guides for people who run Clear Signal outside the app: [CI/CD integration](guides/ci-integration.md), [authenticated conformance runs](guides/authed-conform.md) |
| [`process/`](process/) | How the project is run: the [release process](process/release-process.md), the [engineering changelog](process/engineering-changelog.md), [what separates 0.x from 1.0](process/path-to-1.0.md), manual steps that need an account or a browser |
| [`product/`](product/) | Why the product exists and where it is going: the founding planning notes, the requirements and roadmap, the presentation |
| [`enterprise/`](enterprise/README.md) | Notes for the commercial enterprise edition (teams that own part of an API, org-wide rollup, identity, audit). Kept apart from the open-source project; nothing there is built |
| [`archive/`](archive/) | Retired documents kept only so old links still resolve |
| `images/`, `demo.gif`, `demo.tape` | Assets the README, the demo workflow and the screenshot tests use; the paths are fixed |

## Where does a new document go?

- Something a **user** needs while using the app, or a reference the app should show: `apps/desktop/wails/help/`, and add a Help tab (see `HelpPanel.tsx` and `help_service.go`).
- A **design** for a new feature: `docs/enhancements/`.
- A **plan, a to-do or a backlog item**: `docs/planning/backlog.md`.
- An **audit or comparison**: `docs/reports/`.
- A note about the **enterprise edition** (not the open-source product): `docs/enterprise/`.
- A **how-to** for CI or scripting users: `docs/guides/`.
- A failure kind or a gap we learn about: a row in `what-we-check.md` (and its evidence test).
