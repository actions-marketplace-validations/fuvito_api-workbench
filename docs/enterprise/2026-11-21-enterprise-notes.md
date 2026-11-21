# Enterprise notes: teams that own part of an API

**Date:** 2026-11-21
**Status:** Notes for discussion, not a proposal. Nothing here is built, and none of it is agreed.
**Origin:** The project owner asked how a large organisation, where each team owns only some endpoints of a shared API, would use Clear Signal. Decision: this is an enterprise-edition concern, not an open-source one, so it is written down here.

## 1. The problem

In a large organisation one API (one OpenAPI document, or a gateway in front of many) is owned by many teams. A team is accountable for its own endpoints, not for the other 380. Today's tools measure and gate the whole document, so a team sees noise it cannot act on, and nobody sees the whole picture.

Questions each role asks:

| Role | Question |
|---|---|
| Team lead | Do *my* endpoints match the spec? Did a change break *my* consumers? |
| Platform / API governance | Which endpoints have no owner, no tests, or are failing? Is every team gating in CI? |
| Security / compliance | Who can run checks against which environment? Is there an audit trail? Where do secrets live? |
| Architecture | Are teams drifting from the standards (naming, error shape, auth)? |

## 2. What the open-source core already gives a team

- **A workspace per team** is just a folder, usually in the team's repository, pointing `specFile` at the shared spec (relative path or URL).
- **Scoped runs:** `clrsig conform --tags ... --path-prefix ...`, also on the desktop Check Conformance page.
- **Adopt gradually:** `--baseline` / `--write-baseline` (fail only on new violations).
- **Breaking-change gate:** `diff --against <git-ref> --fail-on breaking`.
- **CI-ready output:** text, JSON, JUnit, HTML, Markdown reports; stable exit codes; a GitHub Action; Docker image.
- **Secret handling:** values masked in every report, `${NAME}` taken from the process environment.

## 3. Gaps for the "I own part of the API" case

| Gap | Effect today |
|---|---|
| Coverage and the Workspace page measure against the **whole** spec | A team owning 20 of 400 operations sees about 5% covered and 380 "not covered" |
| A filtered conformance run is **not recorded** to Contracts | Scoped runs leave no history |
| Scope is **retyped** in every command and CI job | Drift between jobs; no single definition of "our endpoints" |
| `diff` and `impact` do not know who owns an operation | A change cannot be reported as "this breaks operations you own" |
| No view **across** teams | Nobody can list operations that have no owner or no coverage |

## 4. Enterprise capabilities worth validating

Ordered by how directly they answer section 1. Each is an assumption to test with design partners.

1. **Ownership scopes.** A named, saved set of operations (by tag, path prefix, operationId, or an `x-owner` extension in the spec). Coverage, conformance, diff and impact all count only in-scope operations; the rest are shown as "not ours", not "not covered". Needs a small, optional definition file and a `--scope` flag. The engine change is in the core and is generally useful, so this one may belong in the open-source core after all (open question, section 8).
2. **Org-wide rollup.** Merge each team's report into one view: coverage and conformance per team and per operation, an **orphan list** (operations nobody owns or covers), trends over time. Delivered first as a CLI report and a static HTML page; a hosted dashboard only if customers need it.
3. **Policy and standards.** Organisation rules applied to every team's spec and responses (required error shape, naming, auth on every write, no `additionalProperties` drift), versioned centrally, reported as findings with the same what / where / why / fix wording as today.
4. **Central configuration.** One place for shared environments, base URLs, certificates (mTLS) and report destinations, so teams do not each copy them. Includes policy for which environments may be hit with write requests.
5. **Identity, access and audit.** Single sign-on, roles (team member, lead, governance, auditor), and an audit log of who ran what against which environment. Required by most enterprise buyers.
6. **Secrets integration.** Read credentials from the company vault (HashiCorp Vault, cloud secret managers) instead of environment variables; never write them to reports or the run log.
7. **Change impact across teams.** When a shared spec changes, tell each team which of its operations and consumer scenarios are affected, as a pull-request comment or a ticket.
8. **Scheduling and runners.** Run conformance on a schedule from inside the network (private runners), not only from a developer machine or a CI job; results pushed to the rollup.
9. **Integrations.** GitHub, GitLab and Azure DevOps checks and comments; Jira or ServiceNow tickets for failures; Slack or Teams notices; SIEM export of the audit log.
10. **Deployment options.** Self-hosted (air-gapped networks, data residency) and a managed service; the local desktop and CLI keep working offline in both.

## 5. Possible shapes of the product

| Shape | What it means | Trade-off |
|---|---|---|
| **A. Rollup only** | Open-source CLI emits reports; the enterprise product ingests and aggregates them | Smallest build, keeps all execution local (good for security reviews); no live control of runs |
| **B. Control plane + runners** | A server holds scopes, policy, schedules and identity; runners (the same engine) execute inside customer networks and report back | Matches how enterprises buy; largest build; needs the security model first |
| **C. Managed service** | Hosted multi-tenant version of B | Fastest for customers to adopt; hardest for regulated buyers; most operations work for us |

Recommendation to validate: start with **A**, add **ownership scopes** (item 1), grow into **B** when a design partner needs scheduling and identity.

## 6. Constraints the enterprise layer must respect

- **The engine stays one.** The desktop, the CLI and any runner use the same Go packages (`CLAUDE.md` architecture rule 1). The enterprise layer calls them; it never reimplements checking.
- **Local-first stays true** for the open-source product. Enterprise features are opt-in and reachable only when a customer configures them.
- **No unauthenticated local control server** (`CLAUDE.md` rule 5), also for runners: they connect out to the control plane; nothing listens.
- **Credentials are redacted everywhere**, including in anything sent to a server.
- **File formats stay compatible:** an additive optional field (such as a scope) with `schemaVersion` unchanged, documented in `format-spec.md` in the same commit.
- **Every failure still says what, where, why and how to fix.**

## 7. What to do in the open-source core, whatever is decided

Small, general changes that make an enterprise layer possible and help ordinary users too:

- A stable, versioned **JSON report schema** for `run` and `conform` (the rollup depends on it).
- Report fields that identify the **run**: spec path and version, environment name, tool version, time, the operations in scope.
- Keep the engine **importable as a library** (`packages/core`) with its public surface documented.
- A way to **select operations** consistently across commands (tags and path prefixes exist; operationId and `x-owner` selection would complete it).

These are candidates for the normal backlog once the owner agrees; none is added yet.

## 8. Open questions for the owner

1. **Licence and packaging:** is the enterprise edition a separate repository and licence, or a module in this one? Is ownership scoping (item 1) open source, or enterprise-only?
2. **First customer profile:** platform/governance team, or product teams? It decides whether the rollup (A) or scopes come first.
3. **Hosting stance:** self-hosted only at first, or a managed service too?
4. **Identity:** which providers must be supported first (Entra ID, Okta, Google)?
5. **Compliance:** SOC 2 or ISO 27001 expected early? It shapes the audit log and data handling from the start.
6. **Pricing unit:** per seat, per team, per operation under management? Scopes make "operations under management" measurable.
7. **Design partners:** two or three teams with a real multi-team API to test sections 3 and 4 against before anything is built.

## 9. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-21 | Multi-team ownership is an enterprise-edition concern, not an open-source responsibility; notes live in `docs/enterprise/`. Nothing is built | Project owner |
