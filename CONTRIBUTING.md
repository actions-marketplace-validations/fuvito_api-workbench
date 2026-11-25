# Contributing to Clear Signal

Thank you for helping. Clear Signal is a local-first tool for **API Contract Validation** and **Spec Conformance Testing**; a workbench (request editor, scenarios, collections) supports that purpose. This page says how to report a problem, how to propose a change and what a change has to satisfy. The build and test commands are in [README-DEVELOPER.md](README-DEVELOPER.md).

## Reporting a problem

Use **Issues → New issue** and pick the form that fits:

- **A wrong or missing finding.** Clear Signal said an API conforms and it does not (a false pass), or said it breaks the spec and it does not (a false failure), or a message is hard to follow. Please give the **check ID** from the report (for example `CS-RESP-005`; the full list is *What We Check* in the app's Help menu, or [the catalog](apps/desktop/wails/help/what-we-check.md)), the operation, and the smallest spec and response that show it. In the desktop app, **Copy as issue** on a finding writes most of this for you.
- **A bug** in the CLI, the desktop app or the GitHub Action.
- **A feature idea.** Say what you are trying to do, not only what you want added.

Never paste a real token, password or private spec. Reports redact secret values from environment files, but a response body you copy by hand is yours to check.

## Proposing a change

- A small, clearly consistent fix or improvement: open a pull request.
- A larger change (a new command, a new file format or field, a new screen, a change that touches several packages): first write a short design proposal in `docs/enhancements/` as `YYYY-MM-DD-short-name.md`, following [its README](docs/enhancements/README.md), and open an issue or a draft pull request so it can be agreed before it is built. Proposals are kept after they are decided; they are the decision history of the project.
- Things outside the scope of the project (accounts, team sync, hosted features, GraphQL, WebSockets, gRPC, load testing, mock servers, AI-generated tests) are not accepted into the open-source core; see section 9 of [CLAUDE.md](CLAUDE.md).

## What a change must satisfy

[CLAUDE.md](CLAUDE.md) is the contract; read sections 1 to 9. In short:

- **The Go core is the product.** Execution, validation and reporting live in `packages/core`; the CLI and the desktop app call it and contain no such logic. The core never depends on Wails, React or Cobra.
- **Use the project's terms:** "API Contract Validation" and "Spec Conformance Testing", never "contract testing" alone (it suggests Pact-style consumer-driven testing, which this is not).
- **Every failure says what, where, why and how to fix it**, and every new check or gap is added to the catalog (`apps/desktop/wails/help/what-we-check.md`) in the same change; the guard tests fail if the code and the catalog drift apart.
- **A change to a persisted file format** (request, scenario, environment, bundle, project file, baseline) updates `apps/desktop/wails/help/format-spec.md` in the same commit and follows the [schema migration policy](docs/process/schema-migration-policy.md): additive changes keep `schemaVersion`, breaking changes need a new version and a migration.
- **No new dependency** outside the locked stack (section 2 of `CLAUDE.md`) without asking first.
- **Secrets are never written** to history, logs or reports; project files and baselines are meant to be committed.
- **Tests use real local servers** (`httptest`), not mocks of the HTTP client. A bug fix comes with a test that fails without it.
- **Comments say why, not what**, in English. No `TODO` left in committed code; open an issue instead.
- **Record the change** with an entry in the [engineering changelog](docs/process/engineering-changelog.md) (what changed and why, what guards it, what is not done), and update the user guide when behaviour a user sees changes.

## Before you open a pull request

```bash
gofmt -l $(git ls-files '*.go')                      # must print nothing
(cd packages/core && go vet ./... && go test ./...)
(cd apps/cli && go vet ./... && go test ./...)
(cd apps/desktop/wails && go vet ./... && go test ./...)
(cd apps/desktop/frontend && npx tsc --noEmit -p tsconfig.app.json && pnpm test)
(cd apps/desktop/e2e && pnpm exec playwright test --project=frontend --grep-invert "smoke|@screenshot")
```

CI runs the same on every pull request. A pull request that turns a green run red for existing users (a new check, a stricter rule) must say so, and say how to adopt it gradually (`--write-baseline`, `--baseline`).

## License

By contributing you agree that your contribution is licensed under the [MIT License](LICENSE).
