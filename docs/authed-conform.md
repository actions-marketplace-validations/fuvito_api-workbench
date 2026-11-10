# Authenticated Conform Runs

`clrsig conform` runs every operation in a spec against a live server. When the spec declares `security:` requirements, the generated requests include an `Authorization: Bearer {{token}}` header placeholder. This guide shows how to supply the token so conform doesn't report 401s as violations.

---

## How it works

`clrsig generate` (and the internal request generator used by `conform`) reads the `securitySchemes` block and, for each operation that declares a security requirement, adds:

```
Authorization: Bearer {{token}}
```

The `{{token}}` placeholder is resolved from the active environment file at run time.

---

## Step 1 — Create an environment file

Create a YAML file with your credentials. Use `secret: true` on any value you don't want stored in plain text.

**`environments/ci.yaml`**
```yaml
schemaVersion: "1"
name: ci
variables:
  - name: token
    value: "your-bearer-token-here"
    secret: true
  - name: baseUrl
    value: "https://api.example.com"
```

For CI, inject the token from a secret rather than hard-coding it:

```yaml
# GitHub Actions example
- name: Write environment file
  run: |
    cat > environments/ci.yaml <<EOF
    schemaVersion: "1"
    name: ci
    variables:
      - name: token
        value: "${{ secrets.API_TOKEN }}"
        secret: true
    EOF
```

---

## Step 2 — Run conform with the environment

```bash
clrsig conform openapi.yaml \
  --base-url https://api.example.com \
  --env environments/ci.yaml \
  --coverage
```

The `--env` flag resolves `{{token}}` in every generated `Authorization` header before the request is sent.

---

## Step 3 — Run in CI (GitHub Actions)

```yaml
- uses: fuvito/api-workbench@v1
  with:
    command: conform
    spec: openapi.yaml
    base-url: https://api.example.com
    env: environments/ci.yaml
    coverage: 'true'
    format: junit
    output: test-results/conform.xml
  env:
    API_TOKEN: ${{ secrets.API_TOKEN }}
```

Write the env file in a prior step (see Step 1 above), then reference it via `env:`.

---

## Other auth schemes

| Scheme | What to put in the env file |
|---|---|
| Bearer token | `name: token` |
| API key (header) | `name: x-api-key` (match whatever the spec uses as `in: header, name:`) |
| API key (query) | `name: api_key` (match the query param name in the spec) |
| Basic auth | `name: username` + `name: password` — the generator uses `{{username}}` and `{{password}}` |

The placeholder name always matches the spec's parameter name or the security scheme name. If the generated request shows `Authorization: Bearer {{myToken}}`, your env variable name should be `myToken`.

---

## Example: the included example server

The example server at `examples/server/` requires a Bearer token for the `GET /protected` endpoint. The pre-built environment `examples/environments/demo-conform.yaml` includes the demo token:

```bash
clrsig conform examples/server/openapi.yaml \
  --base-url http://localhost:8080 \
  --env examples/environments/demo-conform.yaml \
  --coverage
```

The demo token is `demo-token-12345`, returned by `POST /auth/login` with `admin` / `secret`.
