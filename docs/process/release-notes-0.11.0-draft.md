# Clear Signal 0.11.0 — release notes (draft, not tagged)

**No existing result or exit code changes in this release.** Everything below is new.

## Desktop: baselines and spec health in Check Conformance

- **Baseline.** Choose a baseline file (Options → Baseline) and the violations recorded in it are accepted: only new ones fail the run, and the result shows how many are known, new and fixed against it. **Record as baseline** saves the violations of a run as a baseline file (it asks before replacing one, and is not available while tags or path prefixes limit the run, as with `--write-baseline`). A baseline written by clrsig 0.8.9 or earlier still works and is flagged as old. This is the page's version of `clrsig conform --baseline` and `--write-baseline`.
- **Spec health.** As soon as you choose a spec, the page lists problems in the spec itself, with what, why and how to fix: an operation without an `operationId` (CS-SPEC-001), a schema whose `required` lists a property it does not declare (CS-SPEC-002), an example that breaks its format (CS-SPEC-003). Response violations blame the server for what the spec got wrong, so fix these first.

## CLI and GitHub Actions

- **`clrsig validate --spec openapi.yaml`** checks the spec document itself (scenarios are then optional). Errors fail the check; warnings are shown as notes and fail it only with `--strict`. The GitHub Action takes `spec` for `validate`.
- **`clrsig run --format github`** prints GitHub workflow commands: each failed assertion, request that could not be sent and response that breaks the spec appears on the pull request, on the line of the step in the scenario file (the check ID of a spec problem is in the title). `clrsig conform --format github` already did this for the operation in the spec. Use `format: github` in the Action and keep other reports with `report:`.

## For contributors

`CONTRIBUTING.md`, issue forms (a wrong or missing finding asks for the check ID), a pull request template, and **Report a problem** in the About dialog. CI now also runs the GitHub Action itself against the example server.

## Before tagging

- [ ] CI green on the exact commit (`docs/process/release-process.md`).
- [ ] Not verified yet: the Docker image run, the Homebrew install and the Action at the new tag.
