# Enterprise track

Notes for the **commercial, enterprise edition** of Clear Signal. They live apart from the open-source project on purpose.

## Boundary

- The open-source product (everything else in this repository) stays a **local-first API Contract Validation and Spec Conformance Testing tool**. Per `CLAUDE.md` section 9, cloud accounts, team sync and hosted collaboration are out of scope for it, and nothing in these notes changes that.
- Anything described here is **not planned for the open-source repository** and is not built or stubbed there. Whether it becomes a separate repository, a separate module, or a hosted service is open (see [`2026-11-21-enterprise-notes.md`](2026-11-21-enterprise-notes.md), section 8).
- What the open-source core must keep doing is stay **embeddable and scriptable**: the engine is a Go library, every result is machine-readable, files are plain YAML/JSON. That is what lets an enterprise layer sit on top without forking it.

## Documents

| File | What it is |
|---|---|
| [`2026-11-21-enterprise-notes.md`](2026-11-21-enterprise-notes.md) | First notes: how teams that own only some endpoints would use Clear Signal, what an enterprise layer would add, and what the open-source core already offers |

## Rules for this folder

- Name documents `YYYY-MM-DD-short-name.md`; keep old ones, add new dated ones.
- Say plainly what **exists today**, what is a **gap**, and what is an **assumption** to validate with customers.
- Use the canonical terms: **API Contract Validation** and **Spec Conformance Testing**.
- Open items go to the open-questions section of the notes, not to `docs/planning/backlog.md`, until the owner moves one into the open-source backlog.
