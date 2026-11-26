# Clear Signal 0.13.3 (draft; delete after publishing)

## Please upgrade from 0.13.2: its baselines can hide new problems

**Correction.** The 0.13.2 notes said that no verdict or exit code changed and that existing baselines kept working. **That was wrong.** Our checks compared baselines only with answers that had not changed.

With **OpenAPI 3.1**, one response can have several problems, and 0.13.2 compared them as one message, only up to its first quoted value, in an order that changes from run to run. In a test with one recorded problem and a second, new one:

| | New problem hidden (build passed) |
|---|---|
| 0.13.1 | 0 of 60 runs |
| **0.13.2** | **51 of 60 runs** |
| 0.13.3 | 0 of 60 runs |

With nothing changed, 0.13.2 also reported a new violation in 4 of 40 runs. This affects `conform --baseline` (and the Check Conformance page with a baseline) for 3.1 specs.

**Fixed:** every problem is now a violation of its own, listed in the same order each run, and a baseline matches problem by problem. The errors inside one `anyOf` or `oneOf` stay together.

### Your existing baselines (OpenAPI 3.1)

| Recorded with | With 0.13.3 |
|---|---|
| 0.13.0 or 0.13.1 | Keeps matching every problem it recorded (tested with files those releases wrote) |
| 0.13.2 | It saved only the start of a message with several problems. **The others show as new, and the build may fail.** |

**Do not simply record a new baseline when that happens.** Some of those "new" problems may be real ones that 0.13.2 hid while your builds passed. Read each one first, fix what should be fixed, and then record with `--write-baseline`. Also review what changed in your API while you ran 0.13.2 with a baseline.

A baseline written by 0.13.3 may not match in 0.13.2 or earlier, so use one version on all machines that share the file. Specs in OpenAPI 3.0 report one problem per body, as before.

## Security: passwords written inside longer text are now masked

0.13.2 stopped quoting whole responses, but a password written **inside** a text could still reach `conform` and `run` reports, GitHub annotations, the Check Conformance page, desktop results, and some OpenAPI 3.1 **baseline files**. Earlier releases have the same problem, and those up to 0.13.1 quote whole responses too.

Now only the password is replaced by `***`:
- in an address, `postgres://app:***@db.internal:5432/orders` (scheme, user and host stay);
- after `password=`, `Pwd=`, `token=`, `secret=`, `api_key=`, `AccountKey=` and similar names, in connection strings and query strings (the full list is in What We Check, CS-RUN-011).

Checks, extracted values and later requests still use the real values.

**This does not find every secret.** For example, these are **not** masked: `user:password@host` without `://`; `password: x` or JSON written inside a text; other names such as `pass=`; `--password x`; percent-encoded text; what follows a space in an unquoted password; and a fragment of a password that your scenario expects (`contains app:secret@`). See CS-GAP-017. Shown as they are, on purpose: the desktop response viewer and extracted values, `clrsig request`, and `run --show-extracted`. The desktop request history keeps requests as you wrote them, so it can hold a password.

**If a saved report, CI log, artifact or committed baseline holds a password, change that password** and delete the copies you can. Upgrading does not rewrite old files, and removing a value from a file does not remove it from Git history.

## If scripts read your reports

- An OpenAPI 3.1 response with several problems now gives **one line, finding, annotation and report entry for each**, instead of one long `validation failed due to: … | …` line. A single problem loses that prefix too.
- Without a baseline, which operations pass or fail does not change, and neither do exit codes.

## Known limits

- Some 3.1 messages contain the failing value. If it changes on every run (an id, a time), the problem keeps showing as new: the build fails, nothing is hidden (CS-GAP-019).
- Some messages begin with a path whose parts are repeated (`/user/user/age`). The path after `at` is the right one.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.13.2...v0.13.3
