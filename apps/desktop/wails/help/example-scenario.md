# Clear Signal — Example Scenario

A complete, runnable scenario that uses every feature of the format spec (Help → Format Spec). Use it as a reference when writing your own, or copy any block into a workspace and run it.

It exercises a small user-management API end to end: health check, login, profile, create, read, update, list, other auth styles, a negative upload test, delete, and a confirmation that the user is gone.

It runs against the built-in **Demo Server** (Help menu → Demo Server → Start Demo Server). The demo server implements exactly the endpoints used here.

---

## What you need

Three files in one workspace:

| File | Location | Purpose |
|---|---|---|
| Environment | `environments/example.yaml` | Base URL, credentials and secrets |
| Request | `requests/check-health.yaml` | A reusable request, referenced by the scenario with `requestRef` |
| Scenario | `scenarios/user-onboarding.yaml` | The multi-step workflow |

Select the **example** environment before running.

---

## 1. Environment

Variables are plain values or **secrets**. Secret values are masked in the editor and redacted in history and reports.

```yaml
# file: environments/example.yaml
schemaVersion: "1"
name: example

variables:
  - name: baseUrl
    value: "http://localhost:8090"     # change to the port shown in the status bar

  - name: username
    value: admin

  - name: password
    value: secret
    secret: true                        # redacted in history and reports

  - name: apiKey
    value: demo-key-12345
    secret: true

  - name: basicUser
    value: demo

  - name: basicPass
    value: demo
    secret: true
```

---

## 2. Reusable request

A request file can be run on its own or referenced from any scenario step with `requestRef`. It shows the request-level `timeout` (a duration string, **not** `timeoutMs`).

```yaml
# file: requests/check-health.yaml
schemaVersion: "1"
title: Check health

method: GET
url: "{{baseUrl}}/health"

headers:
  - name: Accept
    value: application/json

timeout: 5s            # 500ms, 5s, 1m30s … omit to use the global default
```

---

## 3. Scenario

```yaml
# file: scenarios/user-onboarding.yaml
schemaVersion: "1"
title: User onboarding and account lifecycle
description: >
  Logs in, creates a user, reads and updates it, exercises the other auth
  styles, runs one negative test, then cleans up and proves the user is gone.

# Binds the scenario to an OpenAPI spec. Every step's response is validated
# against it, and `type: schema` assertions can use operationIds from it.
# Use the demo server's spec (open http://localhost:8090/openapi.yaml, save it).
specFile: "/absolute/path/to/demo-openapi.yaml"

steps:
  # ── 1. Reusable request, referenced instead of inlined ──────────────────
  - label: Health check
    notes: >
      Uses requestRef — the request lives in requests/check-health.yaml.
      Use requestRef OR request on a step, never both.
    requestRef: ../requests/check-health.yaml
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath               # a field in the JSON body
        path: "$.status"
        operator: eq
        expected: ok
      - type: header                 # a response header (name is case-insensitive)
        path: Content-Type
        operator: contains
        expected: json
      - type: schema                 # response matches the spec's schema for this operationId
        schema: getHealth
    extractions:
      - name: healthStatus           # source: status — capture the HTTP status code
        source: status
    failPolicy: stop                 # if this fails, skip every remaining step (the default)

  # ── 2. Login: capture a token from the body ─────────────────────────────
  - label: Login
    notes: POST credentials as JSON. Extracts the bearer token used by later steps.
    request:
      method: POST
      url: "{{baseUrl}}/auth/login"
      headers:
        - name: Accept
          value: application/json
      body:
        type: json
        content: '{"username":"{{username}}","password":"{{password}}"}'
      timeout: 10s
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.token_type"
        operator: eq
        expected: Bearer
      - type: jsonpath
        path: "$.expires_in"
        operator: gt                 # numeric comparison
        expected: 0
      - type: schema
        schema: login
        name: Login response matches the spec   # optional label (shown in the JSON report)
    extractions:
      - name: token
        source: jsonpath             # from the JSON response body
        path: "$.token"
    failPolicy: stop

  # ── 3. Bearer auth block, instead of a manual Authorization header ─────
  - label: Read own profile
    request:
      method: GET
      url: "{{baseUrl}}/auth/profile"
      auth:
        type: bearer
        token: "{{token}}"           # extracted by the previous step
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.role"
        operator: eq
        expected: admin
      - type: schema
        schema: getProfile

  # ── 4. Create a user and capture its id and a header ───────────────────
  - label: Create user
    notes: >
      POST with a JSON body. Extracts the new id for the steps that follow, and
      the Content-Type header to show source: header.
    request:
      method: POST
      url: "{{baseUrl}}/users"
      headers:
        - name: Authorization
          value: "Bearer {{token}}"   # manual header — equivalent to the auth block above
      body:
        type: json
        content: '{"name":"Grace Hopper","email":"grace@example.com","role":"developer"}'
    assertions:
      - type: status
        operator: eq
        expected: 201
      - type: jsonpath
        path: "$.email"
        operator: eq
        expected: grace@example.com
      - type: jsonpath
        path: "$.role"
        operator: matches            # Go regular expression
        expected: "^(developer|viewer|admin)$"
      - type: schema
        schema: createUser
    extractions:
      - name: userId
        source: jsonpath
        path: "$.id"
      - name: createContentType
        source: header               # a response header, matched case-insensitively
        path: Content-Type
    failPolicy: stop

  # ── 5. Read it back using the extracted id ─────────────────────────────
  - label: Read user
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.name"
        operator: contains
        expected: Hopper
      - type: header
        path: Content-Type
        operator: ne
        expected: ""
      - type: schema
        schema: getUser

  # ── 6. Update; keep going even if it fails so cleanup still runs ───────
  - label: Update user
    request:
      method: PUT
      url: "{{baseUrl}}/users/{{userId}}"
      body:
        type: json
        content: '{"role":"manager"}'
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.role"
        operator: eq
        expected: manager
    failPolicy: continue             # a failure here does not stop the scenario

  # ── 7. Only meaningful if the update worked ────────────────────────────
  - label: Verify update
    notes: failPolicy skip — this step is skipped when the previous step failed.
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: jsonpath
        path: "$.role"
        operator: eq
        expected: manager
    failPolicy: skip

  # ── 8. Query parameters and numeric bounds ─────────────────────────────
  - label: List users (page 1)
    request:
      method: GET
      url: "{{baseUrl}}/users"
      queryParams:
        - name: page
          value: "1"
        - name: limit
          value: "2"
    assertions:
      - type: status
        operator: eq
        expected: 200
      - type: jsonpath
        path: "$.limit"
        operator: eq
        expected: 2
      - type: jsonpath
        path: "$.total"
        operator: gt
        expected: 4
      - type: jsonpath
        path: "$.total"
        operator: lt
        expected: 1000
      - type: header
        path: Content-Length
        operator: gt                 # gt / lt also work on numeric headers
        expected: 10

  # ── 9. The other auth styles ───────────────────────────────────────────
  - label: Basic auth
    request:
      method: GET
      url: "{{baseUrl}}/auth/basic"
      auth:
        type: basic
        username: "{{basicUser}}"
        password: "{{basicPass}}"
    assertions:
      - type: status
        operator: eq
        expected: 200

  - label: API key in a header
    request:
      method: GET
      url: "{{baseUrl}}/auth/apikey"
      auth:
        type: apikey
        key: X-Api-Key
        value: "{{apiKey}}"
        location: header
    assertions:
      - type: status
        operator: eq
        expected: 200

  - label: API key in the query string
    request:
      method: GET
      url: "{{baseUrl}}/auth/apikey"
      auth:
        type: apikey
        key: api_key
        value: "{{apiKey}}"
        location: query
    assertions:
      - type: status
        operator: eq
        expected: 200

  # ── 10. Variables carry across steps ───────────────────────────────────
  - label: Echo extracted values
    notes: >
      healthStatus (extracted in Health check) and userId (extracted in Create user) are sent back by the
      echo endpoint. Raw bodies use type: raw.
    request:
      method: POST
      url: "{{baseUrl}}/echo"
      queryParams:
        - name: previousStatus
          value: "{{healthStatus}}"
        - name: userId
          value: "{{userId}}"
      headers:
        - name: Content-Type
          value: text/plain
      body:
        type: raw
        content: "created {{userId}}"
    assertions:
      - type: jsonpath
        path: "$.queryParams.previousStatus"
        operator: eq
        expected: "200"
      - type: jsonpath
        path: "$.body"
        operator: contains
        expected: created

  # ── 11. A negative test ────────────────────────────────────────────────
  - label: Upload without a file is rejected
    notes: >
      Sends multipart/form-data with no "file" field, so the server must answer 400.
      validateSpec: false stops the automatic spec check from flagging an
      intentionally invalid request.
    request:
      method: POST
      url: "{{baseUrl}}/files/upload"
      body:
        type: multipart
        # Fields as a JSON array: n = name, t = text | file, v = value (absolute path for files).
        content: '[{"n":"note","t":"text","v":"no file attached"}]'
    assertions:
      - type: status
        operator: eq
        expected: 400
      - type: jsonpath
        path: "$.error"
        operator: contains
        expected: file
    validateSpec: false

  # ── 12. A form-encoded body ────────────────────────────────────────────
  - label: Form-encoded echo
    request:
      method: POST
      url: "{{baseUrl}}/echo"
      body:
        type: form-urlencoded
        content: "name=Grace&role=manager"
    assertions:
      - type: jsonpath
        path: "$.body"
        operator: contains
        expected: name=Grace

  # ── 13. Disabled: kept in the file, skipped without counting as a failure
  - label: Slow endpoint
    notes: Disabled so normal runs stay fast. Enable it to test timeouts.
    disabled: true
    request:
      method: GET
      url: "{{baseUrl}}/slow"
      queryParams:
        - name: delay
          value: "3000"
      timeout: 1s
    assertions:
      - type: status
        operator: eq
        expected: 200

  # ── 14. Cleanup, then prove it worked ──────────────────────────────────
  - label: Delete user
    request:
      method: DELETE
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: status
        operator: eq
        expected: 204
    failPolicy: continue

  - label: User is gone
    request:
      method: GET
      url: "{{baseUrl}}/users/{{userId}}"
    assertions:
      - type: status
        operator: eq
        expected: 404
      - type: jsonpath
        path: "$.error"
        operator: contains
        expected: not found
    validateSpec: false              # the error body is not part of the spec's schema
```

---

## Options the format accepts but the runner does not apply yet

These are valid in a scenario file and editable in the app, but a step currently runs once with the request's own `timeout`. They are shown here only so you recognise them:

```yaml
- label: Flaky dependency
  request:
    method: GET
    url: "{{baseUrl}}/health"
  timeoutMs: 5000        # per-step timeout; overrides the request's own timeout
  retry:                 # re-send after a network error or a 5xx response
    maxAttempts: 3
    delayMs: 500
```

---

## Feature index

| Spec feature | Where it appears |
|---|---|
| `schemaVersion`, `title`, `description` | All three files |
| Environment variables, `secret: true` | Environment: `password`, `apiKey`, `basicPass` |
| `{{variable}}` from the environment | `baseUrl`, `username`, … in many steps |
| `{{variable}}` extracted by an earlier step | `token`, `userId`, `healthStatus` |
| `requestRef` | Health check |
| Request `timeout` | `check-health.yaml`, Login, Slow endpoint |
| `auth: bearer / basic / apikey` (header and query) | Read own profile, Basic auth, API key in a header / in the query string |
| Manual `Authorization` header | Create user |
| Body types `json / raw / form-urlencoded / multipart` | Login, Echo, Form-encoded echo, Upload |
| `queryParams` | List users, Echo, Slow endpoint |
| Assertions `status / jsonpath / header / schema` | Throughout |
| Operators `eq / ne / gt / lt / contains / matches` | Throughout |
| Assertion `name` | Login |
| Extractions `jsonpath / header / status` | Login, Create user, Health check |
| `failPolicy` `stop / continue / skip` | Health check, Update user, Verify update |
| `disabled` | Slow endpoint |
| `notes` | Most steps |
| `specFile` and `type: schema` | Scenario header, steps with `schema:` |
| `validateSpec: false` | Upload, User is gone |
