# Clear Signal 0.13.2 (draft; delete after publishing)

## Security: reports could contain secrets sent by your API. Please upgrade

**What happened.** When a response did not match its schema, Clear Signal quoted it **in full** in its output. It masked the values your environment already held, but it could not know a secret that only the response contained. So **a token the server had just issued, a password or personal data could be written into reports and CI logs.**

**Where it could appear:**
- `clrsig conform` output and reports (text, JSON, JUnit, HTML, Markdown; GitHub annotations with OpenAPI 3.1);
- `clrsig run` reports. A scenario `schema` assertion also kept the whole response as its `actual` value **even when it passed**, which JSON reports print; `jsonpath` and `header` assertions kept the values they looked at;
- the short excerpt of an error response (an undeclared status);
- **baseline files** (older ones, and violations Clear Signal does not recognise), which are usually committed;
- the desktop app's Check Conformance findings, scenario results and run history.

The workflow written by `clrsig init github-actions` uploads the HTML and JUnit reports as a build artifact.

**Affected:** tested and confirmed on 0.11.1, 0.12.0 and 0.13.1; earlier releases use the same code and should be treated as affected.

**What to do if you ran an earlier version against a system with real credentials or personal data.** Upgrading does not remove anything already written.
1. Look through saved reports, CI logs and uploaded artifacts, especially for login, token and user endpoints, and delete the affected copies where you can.
2. Replace (rotate) passwords, tokens and keys that may appear in them.
3. Check committed baseline files. Re-record affected ones. Removing a value from the current file does not remove it from Git history.

## What reports show now

- **No whole response body, object, list or schema.** The single value that broke the spec may still be quoted, shortened.
- It is masked (`***`) when its field or header is named like a credential (token, password, key, session, cookie, signature, PIN and similar) or it looks like a token. Error excerpts also mask e-mail addresses.
- Checks still compare the real values.

**This is a set of rules, not a guarantee.** A secret in a field with an ordinary name, too plain to look like a token, is still quoted when it is the value that broke the spec (What We Check, CS-GAP-017). Treat reports of sensitive endpoints with care. The desktop response viewer, `clrsig request` and `run --show-extracted` show values on purpose and are unchanged.

## If scripts read your reports

- Messages are shorter: the schema and the body no longer follow a violation.
- In JSON scenario reports, `actual` is now **empty** for `schema` assertions, `***` for credential-named `jsonpath` and `header` assertions and token-like values, and `{…}` or `[…]` for objects and lists.
- **Existing baselines keep working** (checked with baselines recorded by 0.13.1, including OpenAPI 3.1, and with an 0.8.9 file). No check, verdict or exit code changes.

## Also new: a detailed command-line guide (first page)

[`clrsig conform`](https://github.com/fuvito/api-workbench/blob/main/docs/guides/cli/conform.md): every option, examples with what you will see, how to read the result and what to do next. Its examples are run by the tests. More commands will follow.

**Full Changelog**: https://github.com/fuvito/api-workbench/compare/v0.13.1...v0.13.2
