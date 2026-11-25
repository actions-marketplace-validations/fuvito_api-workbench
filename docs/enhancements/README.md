# Enhancement proposals

Every **major** enhancement (new UI area, new command family, format change, anything that touches several packages or needs a release decision) starts as a document here, is agreed, and only then is built. Small, clearly consistent changes do not need one.

## Naming

`YYYY-MM-DD-short-name.md` — the date is the day the proposal was written. Keep finished and rejected proposals; they are the project's decision history.

## Status

`Proposed` → `Accepted` → `In progress` → `Done` (or `Rejected` / `Superseded by <file>`). Update the status line and the decision log as things change.

## Required sections

1. **Summary** — the problem and the idea in a few lines.
2. **Goals / non-goals.**
3. **Requirement review** — consistency with `CLAUDE.md` (architecture rules, locked stack, out-of-scope list, file-format contract), size, necessity, and a smaller alternative if one exists.
4. **Design** — user experience, architecture (which packages, dependency direction), security, data and file formats.
5. **Phasing and acceptance criteria** — what each phase delivers and how we know it works.
6. **Testing.**
7. **Documentation impact** — user guide, `format-spec.md`, READMEs, engineering changelog (`docs/process/engineering-changelog.md`).
8. **Risks.**
9. **Open decisions** — questions for the project owner, each with a recommendation.
10. **Decision log** — date, decision, who.

## When the work ships

Add an entry to the engineering changelog (`docs/process/engineering-changelog.md`), update the user guide (and `format-spec.md` if a serialized format changed), set the status to `Done`, and link the release.
