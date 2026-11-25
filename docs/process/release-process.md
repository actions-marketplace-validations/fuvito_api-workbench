# Release Process

Clear Signal ships from five places simultaneously. This doc covers what to do before, during, and after a release so nothing slips through.

**Distribution channels:**
| Channel | What ships | Automated? |
|---|---|---|
| GitHub Releases | Desktop app (Win/macOS/Linux) + CLI binaries + `.sha256` sidecars | ✓ CI |
| GHCR Docker | `ghcr.io/fuvito/api-workbench` CLI image | ✓ CI |
| Homebrew tap | `brew install fuvito/api-workbench/clrsig` | ✓ CI (post-publish) |
| Scoop bucket | `scoop install clrsig` | ✓ CI (post-publish) |
| GitHub Actions Marketplace | `uses: fuvito/api-workbench@<release tag>` | ✓ auto after initial listing |

---

## 1. Decide the version

| Change type | Version bump |
|---|---|
| Bug fixes only | Patch — `0.1.x` |
| New features, new CLI flags, new commands | Minor — `0.x.0` |
| Breaking CLI/API/file-format changes | Major — `x.0.0` |

---

## 2. Before — prep work (local)

### 2a. Bump the version in two files

```
apps/desktop/frontend/src/components/AboutDialog.tsx   ← version string + RELEASE_NOTES entry
apps/desktop/wails/help/user-guide.md                  ← same
```

Then update every `uses:` reference to this Action in the current documents to the new version (`TestDocumentedActionRefsNameThisVersion` fails until they agree). Until the tag is pushed those references name a tag that does not exist, so tag soon after the release commit passes CI.

`RELEASE_NOTES` in `AboutDialog.tsx` is an array — prepend a new entry for this release; don't edit old ones.

- [ ] Version bumped in both files (also the two version tests: `AboutDialog.test.tsx`, `App.test.tsx`)
- [ ] Release notes entry added in `AboutDialog.tsx`

### 2b. Run the pre-release checks

All commands from the repo root:

```bash
# Formatting (CI fails on any difference)
test -z "$(gofmt -l $(git ls-files '*.go'))"

# Go tests, all four modules
go test ./packages/core/...
(cd apps/cli && go test ./...)
(cd apps/desktop/wails && go test ./...)
(cd examples/server && go test ./...)

# CLI builds clean
go build -o /tmp/clrsig-check ./apps/cli/...

# Frontend type check
cd apps/desktop/frontend && npx tsc --noEmit -p tsconfig.app.json

# Frontend unit tests
pnpm test

# Browser tests against the mock bridge (everything except the live-server smoke tests)
cd apps/desktop/e2e && npx playwright test --grep-invert "smoke|@screenshot"

# Screenshot tests (also regenerates docs/images/ -- look at the diff before committing)
cd apps/desktop/e2e && npx playwright test --grep "@screenshot"
```

- [ ] All commands exit 0

### 2c. Visual checks

- [ ] Screenshots in `docs/images/` look correct in the README (re-run E2E above if stale)
- [ ] Demo GIF (`docs/demo.gif`) plays without errors — if it looks broken, run the demo tape manually: `vhs docs/demo.tape`
- [ ] engineering changelog (`docs/process/engineering-changelog.md`) has an entry for any significant changes in this release

### 2d. Commit everything

```bash
git add -A
git commit -m "chore: prepare v0.x.0 release"
git push origin main
```

Wait for CI to go green on `main` before tagging, and tag **the commit CI tested**: after a push, bots commit to `main` (the demo GIF workflow and, after a release, the tap bump), so run `git fetch` and tag the SHA whose CI run is green (`git tag v0.x.0 <sha>`), not whatever is at the top. If a push is rejected because a bot got there first, `git pull --rebase` and push again.

- [ ] CI is green on `main`

---

## 3. During — tag and push

```bash
git tag v0.2.0        # replace with actual version
git push origin v0.2.0
```

This triggers `release.yml`, which runs these jobs in order:

1. `release-windows` — builds desktop app for Windows
2. `release-macos` — builds desktop app for macOS (arm64 + amd64)
3. `release-linux` — builds desktop app for Linux
4. `cli-binaries` — cross-compiles CLI for all platforms
5. `docker` — builds and pushes to GHCR
6. `publish` — downloads all artifacts, generates SHA256 sidecars, creates GitHub Release
7. `update-tap` — runs `scripts/update-tap.py`, auto-commits updated `Formula/clrsig.rb` and `bucket/clrsig.json`
8. `smoke` (`.github/workflows/smoke.yml`) — after publishing, checks as a user would: the Docker image reports the version and validates a mounted spec (no registry login); the Action from the tag installs and runs that release on Linux, macOS and Windows; `brew tap` + `brew install fuvito/api-workbench/clrsig` gives that version on macOS. **It cannot stop a release** (it is already out): a red `smoke` job means fix forward. To run it again for the newest release: Actions → *Post-release smoke* → Run workflow, with the tag. Not covered by it: the remote reference `uses: fuvito/api-workbench@<tag>` (`uses:` takes no variable), the Scoop install, and the desktop apps.

Watch the run at `github.com/fuvito/api-workbench/actions` (or `gh run watch <id>`). The release job installs the pinned Wails CLI (`wails@v2.17.1`, matching `go.mod`); raise the pin together with the app, never to `@latest`.

- [ ] All CI jobs green

---

## 4. After — verify each channel

### GitHub Releases

Go to `github.com/fuvito/api-workbench/releases` and confirm:

- [ ] Release exists and is not marked as draft
- [ ] Pre-release flag is correct (should be set for tags containing `-`, e.g. `v0.2.0-beta.1`)
- [ ] All 13 expected files are attached:
  - `clrsig-windows-amd64-setup.exe` (desktop installer, Windows)
  - `clrsig.exe` (desktop portable, Windows)
  - `clrsig-darwin-arm64.zip` (desktop, macOS Apple Silicon)
  - `clrsig-darwin-amd64.zip` (desktop, macOS Intel)
  - `clrsig` (desktop, Linux)
  - `clrsig-windows-amd64.exe` (CLI, Windows)
  - `clrsig-darwin-arm64` (CLI, macOS Apple Silicon)
  - `clrsig-darwin-amd64` (CLI, macOS Intel)
  - `clrsig-linux-amd64` (CLI, Linux)
  - `clrsig-darwin-arm64.sha256`
  - `clrsig-darwin-amd64.sha256`
  - `clrsig-linux-amd64.sha256`
  - `clrsig-windows-amd64.exe.sha256`

### Docker (GHCR)

```bash
docker run --rm ghcr.io/fuvito/api-workbench:v0.2.0 --version
docker run --rm ghcr.io/fuvito/api-workbench:latest --version
```

- [ ] Both tags pull and print the correct version (the image gets it from the `VERSION` build argument in `release.yml`; `dev` means the argument was lost)

### Homebrew

The `update-tap` job auto-commits updated SHA256s to `Formula/clrsig.rb`. Verify the formula commit landed on `main`, then test:

```bash
brew tap fuvito/api-workbench https://github.com/fuvito/api-workbench
brew upgrade fuvito/api-workbench/clrsig   # first time: brew install fuvito/api-workbench/clrsig
clrsig --version
```

If the formula SHA256s are wrong (race between publish and update-tap), re-run the `update-tap` job manually from the Actions tab.

- [ ] `clrsig --version` shows the new version via Homebrew

### Scoop

```powershell
scoop bucket add clrsig https://github.com/fuvito/api-workbench
scoop update clrsig   # or install if first time
clrsig --version
```

- [ ] `clrsig --version` shows the new version via Scoop

### GitHub Actions Marketplace

After the initial listing (see `docs/process/manual-steps.md`), subsequent releases are picked up automatically. Verify the new version tag is usable:

```yaml
# In any test workflow
- uses: fuvito/api-workbench@v0.13.1
  with:
    command: conform
    spec: openapi.yaml
    base-url: http://localhost:8080
```

> **Note on major-version tags:** Once the project reaches v1.0.0, maintain a floating `v1` tag so users can pin to `@v1` without tracking every patch. Update it after each minor/patch release:
> ```bash
> git tag -f v1
> git push origin v1 --force
> ```
> Until then, the recommended pin is the full version tag, which is what the documents show.

- [ ] Action works with the new version tag

---

## 5. After — announce (optional but high leverage)

- [ ] Post a short release note in any community channels (GitHub Discussions, Discord, etc.)
- [ ] If this is a significant release, it's a good trigger for one of the two content wedge posts in `docs/process/manual-steps.md`
