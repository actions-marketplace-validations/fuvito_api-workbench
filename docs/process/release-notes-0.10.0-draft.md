# Clear Signal 0.10.0 — release notes (draft, not tagged)

**Read this first: three behaviour changes, all about baselines and `--strict`.**

1. **`--write-baseline` is refused with `--tags`, `--paths` and `--changed-against`.** A baseline written from a limited run would drop the entries of every operation that was left out. Record the baseline with a full run (`clrsig conform spec.yaml --write-baseline conform-baseline.yaml`).
2. **A baseline entry is "fixed" only when its operation was actually sent.** Before, a `--tags`, `--paths` or `--read-only` run reported every entry of the operations it left out as fixed. This removes wrong `FIXED` lines; it does not change what passes or fails.
3. **`clrsig validate --strict` is stricter:** it warns when a scenario's `specFile` does not exist and, with `--env`, about a `${NAME}` the environment needs that no process variable sets. It also reads the request files a step references, which removes a false "extraction nobody uses" that 0.9.1 reported for a value used only inside a referenced request.

## New

- **`clrsig conform --changed-against <git-ref>`** checks only the operations that are new or changed compared with the spec at that ref, for pull requests. "Changed" is what `clrsig diff` reports (a change to a schema several operations share counts for each of them); removed operations are listed, not sent. A pull request that does not touch the API passes with `NOTHING CHECKED` (exit 0). It needs git and a local spec file, and in CI a checkout with full history. It follows the spec only: a server fault with no spec change is invisible to it, so keep a full run on your main branch. GitHub Action input: `changed-against`.
- **`clrsig conform --format github`** prints GitHub workflow commands, so every finding appears on the pull request with the check ID in the title (`CS-RESP-005 GET /users/{id}`), what is wrong, why and how to fix it, and, when the spec is a file in the repository, on the line of the operation. Keep your other reports with `report:` (for example `junit=…`). It is a screen format of `conform`.
- **Desktop:** the Workspace page lists weak scenarios as information items; every finding has **Copy as issue**; and findings shown in the app now hide a secret that came back in a response, as the reports always did.

## Under the hood

CI now runs the CLI and desktop Go tests, `go vet`, the frontend tests and the browser tests (before, only builds ran for the CLI and desktop), and guard tests keep the format spec, the version numbers and the Markdown links in step with the code. The engineering changelog moved from `CLAUDE.md` to `docs/process/engineering-changelog.md`.

## Before tagging

- [ ] CI green on the exact commit (`docs/process/release-process.md`).
- [ ] Date of the 0.10.0 entry in `AboutDialog.tsx` is right.
- [ ] Not verified yet: the Docker image run, the Homebrew install and the Action at the new tag.
