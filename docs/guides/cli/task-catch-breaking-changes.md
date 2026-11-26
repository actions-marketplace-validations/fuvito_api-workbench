# Task: catch breaking spec changes before they ship

[← Guide index](README.md)

**Goal:** when someone edits the spec, know whether existing clients would break. Nothing is sent, and no server is needed.

## 1. Compare the old spec with the new one

<!-- run: exit=1 contains="2 breaking" -->
```bash
clrsig diff docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml
```

Exit code **1** means there is a change at or above `--fail-on` (`breaking` by default). `--fail-on warning` also fails on potentially breaking changes.

## 2. Find which of your scenarios it affects

<!-- run: exit=1 contains="1 scenario(s) affected" -->
```bash
clrsig impact docs/guides/cli/files/shop-v1.yaml docs/guides/cli/files/shop-v2.yaml docs/guides/cli/files/shop-scenarios
```

## 3. In a pull request

Compare with the spec on the main branch. `--against` takes a git ref, so you need no second file:

<!-- not run: needs a git repository with an origin/main branch -->
```bash
clrsig diff --against origin/main openapi.yaml --format markdown --output reports/diff.md
```

The pipeline's checkout must include that branch: with GitHub Actions, set `fetch-depth: 0` on the checkout step.

## Reading a breaking change

| Question | If yes |
|---|---|
| Is it a mistake? | Put the old shape back. |
| Is it meant? | Use a new version, or keep both shapes for a while, and tell the clients first. |
| Does no released client use it? | Say so in the review; the tool cannot know it. |

`diff` compares **specs**. It does not find a server that fails to follow its own spec: [check the API](task-check-an-api.md) for that. Details: [`diff`](diff.md), [`impact`](impact.md).
