# Clear Signal 0.11.1 (draft; delete after publishing)

A fix release for the GitHub Action and the documented way to use it. No check, result or exit code of `clrsig` changes.

## Read this if you use `fuvito/api-workbench/clrsig-action`

That path was an old copy of the Action. It **ignored 18 inputs**, including `read-only`, `baseline`, `negative`, `paths` and `var`. GitHub only warns about an input an action does not declare, so a step with `read-only: 'true'` ran anyway and **sent every operation, `DELETE` included**. If you used it that way, check the server you pointed it at.

It is now the same file as the root Action. **Behaviour change:** inputs that were ignored now take effect, so a run can check fewer operations, apply a baseline or add probes. A workflow that follows `@main` gets this as soon as it is pushed.

## GitHub Action

- **The download is verified.** The Action compares `clrsig` with the SHA-256 published with the release and fails, without installing, when the checksum is missing or different. This catches a damaged or replaced download. It is not a signature and does not protect against a compromised release.
- **macOS:** the install script no longer uses syntax that needs bash 4. CI now runs the Action on macOS as well as Linux.

## Fixes

- The guides told you to write `uses: fuvito/api-workbench@v1`. **That tag does not exist.** Use an exact release, `uses: fuvito/api-workbench@v0.11.1`. A moving `v1` tag starts with 1.0.0.
- `clrsig init github-actions` now writes `permissions: contents: read`, and uses `actions/checkout@v7` and `actions/upload-artifact@v7`.
- **What We Check:** a negative probe passes on **any** `4xx`, not only the codes its label names. A `401`, `404` or `429` can pass a missing-field probe although the body was never validated (CS-GAP-012).

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.11.0...v0.11.1
