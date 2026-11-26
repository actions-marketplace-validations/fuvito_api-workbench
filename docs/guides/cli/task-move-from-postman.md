# Task: move from Postman or Insomnia

[← Guide index](README.md)

**Goal:** bring your requests over, then rebuild the checks. **Only the requests are converted; test scripts are not.**

## 1. Export

- **Postman:** export the collection as *Collection v2.1*.
- **Insomnia:** export as *Insomnia v4* (JSON).

## 2. Look before you write

<!-- run: exit=0 contains="Would write 2 request files" contains="test script was not imported" -->
```bash
clrsig import postman docs/guides/cli/files/shop-postman.json --dry-run
```

**Keep the warnings.** Each names a request whose script was left out. That list is your list of checks to rebuild.

## 3. Import

<!-- run: exit=0 contains="Wrote 2 request files" -->
```bash
clrsig import postman docs/guides/cli/files/shop-postman.json --output imported
```

For Insomnia, use `clrsig import insomnia <file>`.

## 4. Before you commit the files

Look for passwords, tokens and keys written **literally** in them: they are copied as they were. Replace each with a `{{name}}`, and put the value in an environment file as `${NAME}`.

## 5. Try one, then add checks

Send one request with [`clrsig request`](request.md) and your environment file. Then write [scenarios](task-test-a-workflow.md): turn each `pm.test(...)` into an assertion.

| Postman test | Scenario assertion |
|---|---|
| Status is 200 | `{type: status, operator: eq, expected: 200}` |
| A header contains a value | `type: header`, with `path` the header name and `operator: contains` |
| A field has a value | `type: jsonpath`, with `path` such as `$.status` |
| The body matches a schema | `type: schema` |
| An answer is quick enough | `type: duration`, `operator: lt`, `expected` in milliseconds |

If you have an OpenAPI spec, [check the API against it](task-check-an-api.md) too: that needs no hand-written tests. The desktop app imports the same files but does not show the warnings, so use the command line to see them.
