# Manual Steps — Things That Need You

Code is done; these require a browser, an account, or typing something outside the repo.

---

## GitHub

### Tag and cut the first public release

1. Make sure CI is green on `main`.
2. Run in your terminal:
   ```bash
   git tag v0.1.0
   git push origin v0.1.0
   ```
3. GitHub Actions will build all binaries, push the Docker image, create the GitHub Release, and update the Homebrew formula and Scoop manifest automatically.
4. Visit `github.com/fuvito/api-workbench/releases` and confirm the release looks correct.

- [x] First release tagged and published (`v0.1.0`)

---

### List the action on GitHub Actions Marketplace

Requires a tagged release to exist first (see above).

1. Go to `https://github.com/fuvito/api-workbench`.
2. In the right sidebar, click **"Use this GitHub Action with your project"** (GitHub shows this automatically when an `action.yml` is detected at the repo root).
3. Accept the GitHub Marketplace terms.
4. Fill in the category (suggested: **Testing**) and submit.
5. After listing, users can find it by searching "Clear Signal" or "clrsig" in the Marketplace.

- [ ] Action listed on GitHub Actions Marketplace

---

### Make the GitHub repo public (if not already)

1. Go to `https://github.com/fuvito/api-workbench/settings`.
2. Scroll to **Danger Zone** → **Change repository visibility** → **Make public**.

Marketplace listing and Homebrew tap both require the repo to be public.

- [ ] Repository is public

---

### Enable GitHub Packages (GHCR) write permissions

The Docker job in `release.yml` pushes to `ghcr.io` using `GITHUB_TOKEN`. This works automatically for public repos, but if you see a permission error on first push:

1. Go to `https://github.com/settings/packages`.
2. Find `api-workbench` → **Package settings** → set visibility to **Public**.

- [x] GHCR package visibility set to Public (verified — docker job succeeded in v0.2.0)

---

## Homebrew

### Verify the tap works after first release

After the `update-tap` CI job runs (automatically after a tag push), confirm:

```bash
brew tap fuvito/api-workbench
brew install clrsig
clrsig --version
```

If the formula SHA256s are wrong (tap job ran before binaries were fully uploaded), re-run the `update-tap` job manually from the Actions tab.

- [ ] `brew install clrsig` works from the tap

---

## Scoop

### Verify the Scoop manifest works after first release

```powershell
scoop bucket add clrsig https://github.com/fuvito/api-workbench
scoop install clrsig
clrsig --version
```

- [ ] `scoop install clrsig` works

---

## macOS Signing (longer term)

The macOS binaries are unsigned — Gatekeeper blocks them. The README has a workaround note (`xattr -d com.apple.quarantine ...`), but signed binaries remove the friction entirely.

What's needed:
1. Enroll in the **Apple Developer Program** ($99/year) at `developer.apple.com`.
2. Create a **Developer ID Application** certificate.
3. Add the certificate as a GitHub Actions secret (`APPLE_CERTIFICATE`, `APPLE_CERTIFICATE_PASSWORD`).
4. Add a notarization step in `release.yml` using `codesign` + `xcrun notarytool`.

This is a bigger lift — do it when macOS adoption friction becomes a reported blocker.

- [ ] macOS binaries signed and notarized

---

## Content

### Write the first two blog posts

Highest-leverage untapped channel once the packaging above is done. Suggested sequence:

1. **"Contract test your API in CI in 10 minutes"** — walks through `clrsig conform` from zero, ends with a GitHub Actions snippet. Targets "openapi contract testing ci" searches.
2. **"Newman alternative: local-first API testing with Clear Signal"** — comparison-driven post for teams already using Postman/Newman. Targets "newman alternative" and "postman alternative open source" searches.

Publish on dev.to, Hacker News (Show HN), and cross-post to any relevant subreddits.

- [ ] Post 1 written and published
- [ ] Post 2 written and published
