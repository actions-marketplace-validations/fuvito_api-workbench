# Schema versioning and migration policy

Status: **Draft for owner review** (the machine-written files clause in section 5 was added while building baseline v2) (decision D3 of `docs/reports/check-2026-11-24-improvement-ideas.md`, 2026-11-24). Required by `CLAUDE.md` section 8 ("documented migration rules for future schema versions"). It applies to every persisted file: request, scenario, environment, bundle, conformance baseline, and any new one (project config, scenario tags, data files).

## 1. What is versioned

Every persisted file carries `schemaVersion` as a quoted string. Today every file format is at `"1"`. Each format has its own version; they do not have to move together.

## 2. Two kinds of change

| Kind | Examples | `schemaVersion` | Rule |
|---|---|---|---|
| **Additive** | a new optional field, a new assertion type, a new enum value that old files never contain | stays | Old files stay valid and run unchanged. Document in `format-spec.md` in the same commit. Note in the release notes that an older clrsig rejects files that use the new field (unknown fields are errors for bundles, strict decode). |
| **Breaking** | renaming or removing a field, changing a type, changing the meaning of a value, making an optional field required | **increases** (`"2"`, quoted) | Needs a proposal in `docs/enhancements/`, a migration path (section 4), and a new release. |

When unsure, treat the change as breaking. A renamed field is breaking even if a migration is trivial (the 2026-11-14 `title` to `label` rename shipped without a version bump and needed manual file edits; this policy exists so that does not happen again).

## 3. Reading rules

- A file whose `schemaVersion` equals the version the program writes: read as is.
- A file with an **older** supported version: read, upgrade in memory, never rewrite the file silently. Writing it back (a save in the desktop, `clrsig migrate --write`) is an explicit user action.
- A file with a **newer** version than the program knows: refuse with what / why / fix, naming the file, both versions and "upgrade clrsig". Never guess.
- A file with no `schemaVersion`: an error that names the field and the current value to add.
- The program supports reading the **current and the previous** schema version of each format for at least two releases after a bump; older versions need `clrsig migrate` from an earlier release or a documented manual step.

## 4. Migration rules for a breaking change

A breaking change ships with all of:
1. the proposal, which lists every field affected and the mapping old to new;
2. a Go function that upgrades an old document in memory (one function per version step, tested with a golden file per old version kept under `testdata/`);
3. `clrsig migrate <file-or-folder> [--write]`, which prints what would change by default and writes atomically only with `--write`;
4. a section in `format-spec.md` ("Changes between versions") and the release notes with the manual steps;
5. a guard test that the old golden files still load.

## 5. Derived and recorded data

- **Machine-written files that can be regenerated** (a conformance baseline): the migration is "write it again with the same command". No `clrsig migrate` is needed. The program still reads the previous version in memory, says that it is old, never rewrites it on its own and never writes the old version. Hand-written files (scenarios, requests, environments, the project file `clrsig.yaml`, schemaVersion "1" since 0.9.1) still need `clrsig migrate`.

- **Baselines** (`conform --write-baseline`): the violation key is part of the format. Changing it (D1, check ID plus operation) is a breaking change of the baseline schema; the first read of an old baseline upgrades it in memory and says so, and `--write-baseline` writes the new version.
- **Contracts store, history database, run logs, config in `~/.clrsig/`**: not user-authored. They may change with a short migration in code (rename or rewrite on first read) and a line in the release notes; they must never fail to start the app. If data cannot be upgraded, keep the old file, start empty and say so.
- **Reports** (JSON, JUnit, HTML, Markdown): not persisted project files, but the JSON report is an interface for scripts. Additions are allowed at any time; renaming or removing a field is breaking and follows section 2's release-note rule.

## 6. Unknown extension fields

Fields starting with `x-` are preserved where the format allows it (`CLAUDE.md` section 8) and ignored by validation. Anything else unknown is an error in strict formats (bundles, baselines) and a warning elsewhere, as today.

## 7. Checklist for a change to a persisted format

- [ ] Additive or breaking (section 2)?
- [ ] `format-spec.md` updated in the same commit.
- [ ] AI prompt guard test still passes.
- [ ] Validation message says what / why / fix.
- [ ] If breaking: proposal, migration function, golden files, `clrsig migrate`, release notes.
