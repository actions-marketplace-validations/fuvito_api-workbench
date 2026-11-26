# A detailed, CLI-first user guide

- **Status:** Phases 0 to 3 done (2026-11-25); phase 4 (inside the app) not decided
- **Date:** 2026-11-25
- **Backlog:** B-123 (owner request, 2026-11-25)
- **Release:** none needed for phases 0 to 3 (documents and tests only)

## 1. Summary

The existing user guide is fine and stays as it is. It is one document that covers the desktop app and the CLI, so each CLI command gets a short entry: what it is and its flags. It does not say, command by command, **what you will see and what to do about it**.

The owner asked for a second, detailed guide, based on the CLI (the desktop second). For every command: its purpose, every option, example commands and what they do, what to expect, and what to do next depending on the result.

**The size, measured from the CLI built at `d7e9057` on 2026-11-25:**

- **22 commands** that can be run (`bundle` has four, `coverage` and `telemetry` three each, `init` two with its subcommand).
- **91 command flags** and **5 global flags** (besides `--help` and `--version`). `conform` alone has 22.
- No command has an Examples section in its `--help`.
- The command catalog the desktop builder uses (`packages/core/clispec`) describes **3** commands: `run`, `conform` and `diff`.
- CLI material in the existing guide: section 10, about 380 of its 1,238 lines.

## 2. Goals and non-goals

**Goals**
- A reader who has never used Clear Signal can finish a first check using only this guide.
- Every command and every flag is documented, and a test fails when one is added, renamed or removed without the guide.
- Every example that can run against the example server **is run by a test**, so the exit code and the quoted output stay true.
- From any result, the reader can find what it means and the next step.

**Non-goals**
- Replacing, shrinking or restructuring the existing user guide.
- Repeating the file formats (`format-spec.md`) or the check catalog (`what-we-check.md`). The guide links to them.
- A step-by-step desktop manual. The desktop gets a short "in the app" note where it has the same action.
- Translations, video, or a documentation website.
- Changing how any command behaves.

## 3. Requirement review (`CLAUDE.md` section 6)

| Question | Finding |
|---|---|
| Conflicts with a requirement? | Not with `CLAUDE.md`. It does change an earlier arrangement of **one** user guide (the old `docs/user-guide.md` was deleted on 2026-11-16 because two copies drifted). The owner has now asked for a second one; the answer to the old problem is a different purpose and tests (section 6). |
| Changes a serialized format, adds a dependency? | No. |
| Larger than it looks? | **Yes.** 22 commands and 96 flags, each with examples whose output has to be produced and checked. The test harness is a piece of work of its own. |
| Necessary? Smaller alternative? | A smaller step is to add `Examples` to each command's `--help` and a result table to section 10. It would help at the terminal but not give the walkthrough or the "what to do next" the owner asked for. |
| Risk of two documents disagreeing? | Real. Limited by: tests for commands, flags and examples; the existing guide staying the short reference; and no format or check descriptions copied. |

## 4. Design

### 4.1 Where it lives

`docs/guides/cli/`: an index and one file per command or command group, read on GitHub. Not embedded in the app at first: showing it there needs a new Help entry and navigation between many files, which is app work (phase 4, optional).

The existing documentation tests already cover this folder: links, the bare-terminology rule, Action references and the Homebrew command.

### 4.2 Outline

**Part 1: Start here**
1. What Clear Signal checks, in five minutes: API Contract Validation and Spec Conformance Testing; specs, environments, scenarios, project file.
2. Install and verify (Scoop, Homebrew, release binary, Docker), including the macOS and Windows notes.
3. First check, end to end, against the example server: `init`, `conform --list`, fill the environment, `conform`, read the result, save a report.

**Part 2: By goal** (short pages that send the reader to commands): check an API against its spec; adopt it on an API that fails today (baselines); test a workflow with scenarios; catch breaking spec changes in pull requests; run it all in CI; track what is covered; move from Postman or Insomnia.

**Part 3: Command reference**, one page each:
`init` (and `github-actions`), `config show`, `conform`, `run`, `validate`, `request`, `generate`, `diff`, `impact`, `bundle` (four), `coverage` (three), `import`, `mcp`, `telemetry` (three).

**Part 4: Reference**
- Global flags; project file and precedence; environments, variables and secrets.
- Exit codes of every command in one table.
- Reports and which command writes which format.
- **Troubleshooting by message:** the text you see, why, what to do (starting with "not sent", "no operation matches", "NOTHING CHECKED", TLS errors, "is the server running?").

### 4.3 The page of a command

Every command page has the same sections, in this order:

1. **Purpose**: one paragraph, and when to use something else instead.
2. **Synopsis**.
3. **Options**: a table of every flag with what it does, its default, and what it cannot be combined with.
4. **Examples**: for each, the goal, the command, **what it does**, **what you will see** (real output, shortened where marked) and the exit code.
5. **Reading the result**: the verdicts and statuses, line by line.
6. **What to do next**: a table: result → what it means → next step.
7. **Common mistakes and limits**, linking the check catalog's known gaps.
8. **Related commands**, and **In the desktop app** (where it exists).

Commands are given in `bash`. Where Windows differs (quoting, `MSYS_NO_PATHCONV`), the page says so next to the example.

### 4.4 What we find while writing

Writing "what you will see" means running every command, which is likely to turn up surprises (that is how `--paths` with a parameter was found). A surprise that is a bug or a gap is **recorded** in the backlog and the catalog and told to the owner; the guide describes what the product really does and does not word around it. Fixes are separate work.

## 5. Phasing and acceptance criteria

| Phase | Delivers | Done when |
|---|---|---|
| 0 | This proposal agreed | The decisions in section 9 are answered. |
| 1 | The skeleton, both test guards, and **one complete page: `conform`**, the largest command | The owner has read the `conform` page and accepts the format and depth. Nothing else is written before that. |
| 2 | The first-hour walkthrough, and the pages of `init`, `config show`, `run`, `validate`, `request`, `generate` | A new reader can follow the walkthrough against the example server; the tests pass for these pages. |
| 3 | The remaining commands, the goal pages, exit codes and troubleshooting | All 22 commands and 96 flags are covered, and the "missing page" allowance in the test is empty. |
| 4 (optional) | The guide inside the app | A separate decision, after phase 3. |

Size: **L**. Phase 1 shows how long one page really takes, and the rest can be estimated from it.

## 6. Testing

1. **Coverage test (CLI module).** Walks the real command tree. Every runnable command has a page; every flag of it appears in that page's options table; a documented flag that does not exist fails. During phases 1 and 2, the pages not yet written are listed in the test, and the list may only shrink.
2. **Example test.** An example block marked with a short comment before it, giving the expected exit code and lines the output must contain, is run with the built CLI against the example server, in a temporary folder with its own home directory. Timings and other changing values are not compared.
3. **What the tests do not prove:** that the prose is correct or clear, and examples that cannot run offline (a remote URL, an MCP client, a real CI service). Those carry a visible "not run by a test" mark.

Checking flags against the command tree, rather than extending `clispec` to the other 19 commands, is deliberate: the catalog also drives the desktop CLI Command Builder, so extending it changes the app.

## 7. Documentation impact

- New: `docs/guides/cli/`.
- `docs/README.md` and `README.md`: one link each.
- The existing user guide: one link at the start of section 10, nothing else (decision 4).
- The engineering changelog, at the end of each phase.

## 8. Risks

| Risk | Handling |
|---|---|
| The guide goes stale | The two tests in section 6; a release cannot pass with an undocumented flag. |
| A large body of text nobody reads | Same page shape everywhere, goal pages as the way in, one command per file. |
| Quoted output drifts between releases | Examples assert the lines that matter, not whole outputs; shortened output is marked. |
| The example server cannot show a case (OAuth, mTLS) | Say so on the page and show the command without invented output. |
| The two guides disagree | The detailed guide links to the format spec and the catalog instead of restating them. |

## 9. Open decisions

| # | Question | Recommendation |
|---|---|---|
| 1 | CLI first with short desktop notes, or desktop first? | **CLI first.** Every action can be shown, copied and tested; the desktop is mentioned where it does the same. |
| 2 | In the repository (`docs/guides/cli/`) or inside the app? | **Repository first**, in many files. Decide about the app after phase 3. |
| 3 | Run the examples in tests? | **Yes.** It is the part that keeps "what you will see" true. It makes phase 1 larger. |
| 4 | May the existing guide get one link to the new one? | **Yes, one line**; otherwise a reader of the app's guide never learns it exists. |
| 5 | Stop after the `conform` page for review? | **Yes.** A format change is cheap after one page and expensive after twenty. |
| 6 | Your request ended with "etc." Is anything missing from the page in 4.3 or the outline in 4.2? | Candidates left out on purpose: a desktop manual, a FAQ, a glossary page. Say which you want. |

## 10. Decision log

| Date | Decision | Who |
|---|---|---|
| 2026-11-25 | A second, detailed guide, CLI-based (desktop secondary, "or vice versa"); the existing guide stays. | Owner |
| 2026-11-25 | Proposal written. | Claude |
| 2026-11-25 | Accepted with every recommendation in section 9: CLI first with short desktop notes; in the repository first; examples run by tests; one link from the existing guide; stop after the `conform` page for review; no desktop manual, FAQ or glossary. | Owner |
