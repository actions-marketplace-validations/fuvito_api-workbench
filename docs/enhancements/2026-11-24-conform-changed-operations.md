# `conform --changed-against`: check only the operations a change touched

- **Status:** Accepted
- **Date:** 2026-11-24
- **Backlog:** B-085 (also B-087 benefits: fewer, more relevant findings per pull request)

## 1. Summary

A pull-request gate that runs `clrsig conform` against a large API sends every operation and reports every old violation, even when the pull request touched one endpoint. That makes the gate slow, and its failures are mostly not the author's.

`clrsig diff` already knows which operations changed between two versions of a spec, including operations whose response schema changed through a shared component (it attributes component changes to every operation that uses them). The idea: `clrsig conform spec.yaml --changed-against origin/main` compares the spec with its version at that git ref, and checks **only the operations that were added or changed**.

It is a speed and relevance option for pull requests, not a replacement for a full run; section 4.6 says what it cannot see.

## 2. Goals and non-goals

**Goals**
- `conform --changed-against <git-ref>` checks only operations that are new or whose contract changed against that ref, with the same checks, reports and exit codes as a full run.
- One definition of "changed": the one `diff` uses (oasdiff), exposed through the shared core, not re-implemented in the CLI.
- A run that finds nothing changed says so plainly and passes.
- Works with `--baseline`, `--negative`, `--tags`/`--paths` (as an intersection) and every report format.

**Non-goals**
- Detecting that the **server** changed without the spec changing. This only follows the spec (section 4.6).
- Comparing against anything but a git ref of the same spec file (two files remain `diff`'s job).
- The desktop app: it has no git integration; the Check Conformance page is unchanged.
- A saved project-file key: the ref is different on every run.

## 3. Requirement review (`CLAUDE.md` section 6)

| Question | Finding |
|---|---|
| Conflicts with a requirement? | No. Local files and `git` only (as `diff --against` already does), no server, no new dependency. Core logic stays in Go (rule 1): the change set is computed in `packages/core/openapi` and applied in `conformance`; the CLI only supplies the two spec versions. |
| Changes a serialized format? | No persisted file changes. The JSON conform report gains an additive `scope` field (section 4.4). |
| Larger than it looks? | A little: `openapi` (the change set), `conformance` (an operation filter for requests and probes), `application`, the CLI, the builder catalog, the Action. Phased in section 5. It also exposed an existing defect (section 4.5) that this feature would make common. |
| Unnecessary? | `--tags`/`--paths` can already narrow a run by hand, but they need the author to know which operations changed. A wrapper script around `clrsig diff` could compute them, but would parse text output and drift. A smaller alternative, printing the changed operations from `diff` as `--paths` values, does not handle methods, so it does not fit. |

## 4. Design

### 4.1 The command

```
clrsig conform <spec-file> --changed-against <git-ref> [other conform flags]
```

- The spec must be a local file inside a git repository (a URL is refused, exactly as for `diff --against`); the ref is checked the same way (`gitSpecAt`), including the "must not start with `-`" rule.
- `--changed-against` combines with `--tags` and `--paths`: an operation must be changed **and** match them.
- It needs the ref's history. In a shallow clone the command stops with: "the ref origin/main is not available: fetch more history (in GitHub Actions, check out with `fetch-depth: 0`)".

### 4.2 What counts as changed

`openapi.ChangedOperations(base, revision *Spec) (ChangeSet, error)`:

1. Run `Diff(base, revision)`.
2. Keep every change that names a method and a path (any level: breaking, warning and info; an added endpoint is an info change).
3. Keep only operations that exist in the **revision** (a removed operation cannot be sent; it is reported as "removed, not checked").
4. Return the set keyed by `METHOD /path-as-the-spec-writes-it`, plus counts: added, modified, removed, and spec-level changes that no operation could be attributed to (a changed `servers` list, `info`, global `security`).

A modified operation is one whose parameters, request body, responses, headers or the schemas they reference changed. The file's formatting, descriptions and examples may or may not count, as oasdiff decides.

### 4.3 Applying it

`conformance.Options.OnlyOperations map[string]bool` (nil means no restriction). An operation outside the set is left out like one excluded by `--tags`: not sent, listed as skipped with the reason, never counted as passed. Negative probes follow the same set (they already follow the tag and path filters). `application.ConformInput` carries it; the CLI builds it.

### 4.4 Reporting

- **stderr, always:** `Changed against origin/main: 3 operations (1 added, 2 modified), 1 removed and not checked, 24 operations unchanged and not checked.` plus, when relevant, `2 spec-level changes (servers, security) were not attributed to an operation and do not widen the check.`
- **Reports:** the text report prints the same line under the base URL; JSON gains `scope: { "changedAgainst": "origin/main", "checked": [...], "unchanged": 24, "removed": 1 }`; HTML and Markdown show the line. JUnit is unchanged (skipped cases already carry the reason).
- **Nothing changed:** the run prints `NOTHING CHECKED: no operation changed against origin/main` and exits **0**. This differs on purpose from a mistyped `--tags`/`--paths` filter, which exits 2: there, nothing matched by mistake; here, nothing changed is the correct answer for a pull request that does not touch the API.

### 4.5 Baselines (and an existing defect)

`BaselineDiff.Fixed` lists every baseline entry that the current run did not find. On a run that checked only some operations, that is wrong: an entry for an operation that was never sent is reported as "fixed". It already happens with `--tags` and `--paths`; with `--changed-against` it would happen on nearly every pull request.

**Fix, for all filtered runs:** an entry is "fixed" only if its operation was actually checked in this run. Entries of operations that were not checked are left alone and not listed, so the baseline stays accurate and `--write-baseline` on a filtered run must refuse to drop them (it writes only with a full run, or merges: see open decision 5).

### 4.6 What it cannot see

- A **server** change with no spec change (a fixed bug, a regression) is invisible to it. Keep a full `conform` on `main` and nightly.
- A spec change that oasdiff attributes to no operation (for example a new `servers` entry or a changed global security requirement) does not widen the check; it is reported in the note (4.4).
- The comparison is spec against spec, so a hand-edited ref that does not contain the file means every operation is "added".

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| **1. Core and CLI** | `openapi.ChangedOperations`, `conformance.Options.OnlyOperations`, the baseline "fixed" fix for filtered runs, `conform --changed-against` | With a spec changed in one response field and one new endpoint, only those two operations are sent; a pull request that changes nothing exits 0 with NOTHING CHECKED; a baseline entry of an unchecked operation is not reported fixed (with `--tags` as well) |
| **2. Surroundings** | Action input `changed-against`, builder catalog entry, user guide and CI guide (with `fetch-depth: 0`), catalog row | The generated workflow (`init github-actions`) does not change by default; the CI guide shows a pull-request job using it |

## 6. Testing

- **Core:** `ChangedOperations` on the specs from the investigation (added endpoint, component schema change attributed to two operations, added response status, removed endpoint, version-only change); the set excludes removed operations and spec-level changes; `OnlyOperations` skips requests and probes; baseline "fixed" only for checked operations (also for `--tags`).
- **CLI end to end:** a temporary git repository with two commits of one spec and a real local server: only changed operations are requested (server hit counts), nothing changed exits 0, a shallow or unknown ref fails with the message, a URL spec is refused, combination with `--paths`, `--baseline` and `--format json`. Skipped when `git` is not installed.
- **Parity:** `TestCatalogMatchesCobra` for the new flag.

## 7. Documentation impact

User guide (`conform` flags and a "Pull requests" paragraph), `docs/guides/ci-integration.md` (job example, `fetch-depth: 0`, the limits of 4.6), `what-we-check.md` (CS-NC-007 notes the new skip reason; the limit of 4.6 as a known gap), `CLAUDE.md` changelog, backlog B-085.

## 8. Risks

| Risk | Mitigation |
|---|---|
| A green pull request hides a server regression | Stated in 4.6, in the guide and in the report line; recommend a full run on `main` and nightly. |
| oasdiff misses a change that affects an operation | We reuse its rules as `diff` does; the note counts unattributed spec-level changes. |
| Shallow clones fail confusingly | A specific message and the `fetch-depth: 0` instruction. |
| The baseline fix changes the output of `--tags` runs | It only removes wrong "fixed" lines; it is listed in the release notes. |

## 9. Open decisions (owner), with recommendations

1. **Which changes count.** Any level, including informational ones (an added endpoint is informational). **Recommend yes**: otherwise a new operation would never be checked.
2. **Nothing changed.** Exit 0 with NOTHING CHECKED. **Recommend yes**, with the message naming the ref.
3. **Unattributed spec-level changes.** Do not widen the check; report them. **Recommend that**; widening to everything on any global change makes the option rarely useful.
4. **Flag name.** `--changed-against <ref>` (parallel to `diff --against`) or `--since <ref>`. **Recommend `--changed-against`:** it says what is compared.
5. **`--write-baseline` on a filtered run.** Refuse it (a baseline of a subset would drop the other entries), or merge into the existing file. **Recommend refusing with a message** now; merging can come later.

## 10. Decision log

| Date | Decision | By |
|---|---|---|
| 2026-11-24 | Proposal written; open decisions 1 to 5 above | Claude, awaiting owner |
| 2026-11-24 | Open decisions 1 to 5 accepted as recommended: any level counts; nothing changed exits 0 with NOTHING CHECKED; unattributed spec-level changes are reported, not widened; `--changed-against`; `--write-baseline` is refused on a filtered run | owner |
