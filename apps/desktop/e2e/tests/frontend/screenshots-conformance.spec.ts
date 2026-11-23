/**
 * Check Conformance screenshot suite — not a pass/fail test suite.
 * Captures the Check Conformance page after a run and a scenario's spec conformance result,
 * in dark and light themes, and saves PNG files to docs/images/ for the README.
 * The data mirrors what the built-in Conformance Server returns.
 *
 * Run with: pnpm test -- --grep @screenshot-conformance
 */
import { test, gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const IMAGES_DIR = path.join(__dirname, '..', '..', '..', '..', '..', 'docs', 'images')

const VIEWPORT = { width: 1440, height: 900 }

function shot(name: string) {
  return path.join(IMAGES_DIR, name)
}

// Captured from a real run of the built-in Conformance Server through ConformanceService (all ten
// operations, writes on), so the screenshot shows what the product really reports.
const RESULT = {
  "specTitle": "Clear Signal Conformance Server",
  "baseUrl": "http://localhost:8120",
  "writes": true,
  "durationMs": 41,
  "operations": [
    {
      "operationId": "listCategories",
      "method": "GET",
      "path": "/categories",
      "status": "failed",
      "statusCode": 200,
      "durationMs": 2,
      "violations": [
        "Error at \"/0/name\": property \"name\" is missing\nSchema:\n  {\n    \"properties\": {\n      \"id\": {\n        \"type\": \"integer\"\n      },\n      \"name\": {\n        \"type\": \"string\"\n      }\n    },\n    \"required\": [\n      \"id\",\n      \"name\"\n    ],\n    \"type\": \"object\"\n  }\n\nValue:\n  {\n    \"id\": 1\n  }\n"
      ],
      "findings": [
        {
          "kind": "missing-field",
          "part": "response body",
          "where": "[0].name",
          "spec": "GET /categories → response 200 → body schema",
          "summary": "\"name\" is missing from the response",
          "expected": "the field \"name\"",
          "got": "no such field",
          "why": "The spec marks \"name\" as required, but the server's response did not include it.",
          "fix": "Return \"name\" from the server, or make it optional in the spec if it is not always present.",
          "detail": "Error at \"/0/name\": property \"name\" is missing\nSchema:\n  {\n    \"properties\": {\n      \"id\": {\n        \"type\": \"integer\"\n      },\n      \"name\": {\n        \"type\": \"string\"\n      }\n    },\n    \"required\": [\n      \"id\",\n      \"name\"\n    ],\n    \"type\": \"object\"\n  }\n\nValue:\n  {\n    \"id\": 1\n  }\n"
        }
      ]
    },
    {
      "operationId": "getHealth",
      "method": "GET",
      "path": "/health",
      "status": "passed",
      "statusCode": 200,
      "durationMs": 3,
      "violations": [],
      "findings": []
    },
    {
      "operationId": "getInventory",
      "method": "GET",
      "path": "/inventory",
      "status": "failed",
      "statusCode": 200,
      "durationMs": 4,
      "violations": [
        "GET /inventory 200: the response header \"X-Total-Count\" is required by the spec but was not sent"
      ],
      "findings": [
        {
          "kind": "missing-header",
          "part": "response header",
          "where": "X-Total-Count",
          "spec": "GET /inventory → response 200 → headers → X-Total-Count",
          "summary": "The required header \"X-Total-Count\" was not sent",
          "expected": "the header X-Total-Count",
          "got": "not sent",
          "why": "The spec marks the \"X-Total-Count\" header of response 200 as required, but the server's response did not include it. The body was not compared with the schema.",
          "fix": "Send \"X-Total-Count\" from the server, or make it optional (or remove it) in the spec.",
          "detail": "GET /inventory 200: the response header \"X-Total-Count\" is required by the spec but was not sent"
        }
      ]
    },
    {
      "operationId": "listOrders",
      "method": "GET",
      "path": "/orders",
      "status": "failed",
      "statusCode": 200,
      "durationMs": 2,
      "violations": [
        "Error at \"/0/id\": value must be an integer\nSchema:\n  {\n    \"type\": \"integer\"\n  }\n\nValue:\n  \"1\"\n"
      ],
      "findings": [
        {
          "kind": "wrong-type",
          "part": "response body",
          "where": "[0].id",
          "spec": "GET /orders → response 200 → body schema",
          "summary": "\"[0].id\" is text, the spec says integer",
          "expected": "integer",
          "got": "\"1\"",
          "why": "The server sent text (\"1\") where the spec declares an integer.",
          "fix": "Send an integer from the server (for example without quotes for a number), or change the type in the spec if the server is right.",
          "detail": "Error at \"/0/id\": value must be an integer\nSchema:\n  {\n    \"type\": \"integer\"\n  }\n\nValue:\n  \"1\"\n"
        }
      ]
    },
    {
      "operationId": "listProducts",
      "method": "GET",
      "path": "/products",
      "status": "passed",
      "statusCode": 200,
      "durationMs": 3,
      "violations": [],
      "findings": []
    },
    {
      "operationId": "createProduct",
      "method": "POST",
      "path": "/products",
      "status": "passed",
      "statusCode": 201,
      "durationMs": 4,
      "violations": [],
      "findings": []
    },
    {
      "operationId": "getProduct",
      "method": "GET",
      "path": "/products/{{id}}",
      "status": "failed",
      "statusCode": 200,
      "durationMs": 2,
      "violations": [
        "Error at \"/price\": value must be a number\nSchema:\n  {\n    \"type\": \"number\"\n  }\n\nValue:\n  \"49.90\"\n"
      ],
      "findings": [
        {
          "kind": "wrong-type",
          "part": "response body",
          "where": "price",
          "spec": "GET /products/{id} → response 200 → body schema",
          "summary": "\"price\" is text, the spec says number",
          "expected": "number",
          "got": "\"49.90\"",
          "why": "The server sent text (\"49.90\") where the spec declares a number.",
          "fix": "Send a number from the server (for example without quotes for a number), or change the type in the spec if the server is right.",
          "detail": "Error at \"/price\": value must be a number\nSchema:\n  {\n    \"type\": \"number\"\n  }\n\nValue:\n  \"49.90\"\n"
        }
      ]
    },
    {
      "operationId": "deleteProduct",
      "method": "DELETE",
      "path": "/products/{{id}}",
      "status": "passed",
      "statusCode": 204,
      "durationMs": 3,
      "violations": [],
      "findings": []
    },
    {
      "operationId": "getStatus",
      "method": "GET",
      "path": "/status",
      "status": "failed",
      "statusCode": 503,
      "durationMs": 4,
      "violations": [
        "GET /status answered 503, but the spec declares only 200"
      ],
      "findings": [
        {
          "kind": "undeclared-status",
          "part": "response status",
          "where": "503",
          "spec": "GET /status → responses",
          "summary": "Status 503 is not declared for GET /status",
          "expected": "one of: 200",
          "got": "503",
          "why": "The spec lists the responses GET /status may return (200) and 503 is not one of them: the server returned something the spec does not describe, or the spec is incomplete.",
          "fix": "Make the server answer a declared status, or add 503 (or a default response) to GET /status → responses in the spec.",
          "detail": "GET /status answered 503, but the spec declares only 200"
        }
      ]
    },
    {
      "operationId": "listWidgets",
      "method": "GET",
      "path": "/widgets",
      "status": "failed",
      "statusCode": 200,
      "durationMs": 2,
      "violations": [
        "GET /widgets 200 answered with Content-Type \"text/plain\", but the spec declares application/json for this response; the body was not checked"
      ],
      "findings": [
        {
          "kind": "wrong-content-type",
          "part": "response header",
          "where": "Content-Type",
          "spec": "GET /widgets → response 200 → content",
          "summary": "Content-Type is \"text/plain\", the spec says application/json",
          "expected": "application/json",
          "got": "text/plain",
          "why": "The spec declares the 200 response body as application/json, but the server labelled it \"text/plain\", so the body was not compared with the schema: nothing about its fields was checked.",
          "fix": "Make the server send Content-Type application/json, or declare \"text/plain\" for response 200 in the spec.",
          "detail": "GET /widgets 200 answered with Content-Type \"text/plain\", but the spec declares application/json for this response; the body was not checked"
        }
      ]
    }
  ],
  "total": 10,
  "covered": 10,
  "notCovered": 0,
  "passing": 4,
  "failing": 6,
  "probes": [],
  "missingValues": [],
  "blockedOperations": 0,
  "warnings": []
}

async function openConformance(page: Page, theme: 'dark' | 'light') {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(
    ({ theme, result }) => {
      const w = window as unknown as Record<string, Record<string, Record<string, Record<string, unknown>>>>
      const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
      ;(b['settings'] as Record<string, unknown>)['theme'] = theme
      const svc = w['go']['main']['ConformanceService']
      svc['GetConformContext'] = () => Promise.resolve({
        workspace: '/workspace',
        specs: [{ path: '/workspace/specs/conformance-openapi.yaml', label: 'specs/conformance-openapi.yaml' }],
        environments: [{ path: '/workspace/environments/local.yaml', label: 'environments/local.yaml' }],
      })
      svc['InspectConformSpec'] = () => Promise.resolve({ title: 'Clear Signal Conformance Server', version: '1.0.0', serverUrl: 'http://localhost:8120', reads: 8, writes: 2 })
      svc['RunConformance'] = () => Promise.resolve(result)
    },
    { theme, result: RESULT },
  )
  await gotoApp(page)
  await page.getByRole('button', { name: 'Check Conformance' }).click()
  await page.getByLabel(/^Spec/).selectOption('/workspace/specs/conformance-openapi.yaml')
  await page.getByLabel(/^Environment/).selectOption('/workspace/environments/local.yaml')
  await page.getByLabel(/Also send write requests/).check()
  await page.getByRole('button', { name: 'Run conformance check' }).click()
  const results = page.getByRole('region', { name: 'Conformance results' })
  await results.waitFor({ timeout: 5_000 })
  // Put the report in view, not the form above it; the summary panel stays at the right.
  await results.evaluate(el => el.scrollIntoView({ block: 'start' }))
}

async function openScenarioResult(page: Page, theme: 'dark' | 'light') {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(
    ({ theme }) => {
      const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
      ;(b['settings'] as Record<string, unknown>)['theme'] = theme
      b['workspacePath'] = '/workspace'
      b['scenarios'] = [{ title: 'Product lifecycle', path: '/ws/scenarios/product-lifecycle.yaml', stepCount: 3 }]
      const step = (label: string, method: string, url: string) => ({
        label, method, url, requestRef: '', failPolicy: '', assertions: [{ type: 'status', path: '', operator: 'eq', expected: 200, schema: '' }], extractions: [],
      })
      b['scenarioDetail'] = {
        path: '/ws/scenarios/product-lifecycle.yaml', title: 'Product lifecycle', description: '', error: '',
        specFile: 'specs/conformance-openapi.yaml',
        steps: [
          step('List products', 'GET', 'http://localhost:8120/products'),
          step('Get a product', 'GET', 'http://localhost:8120/products/1'),
          step('List categories', 'GET', 'http://localhost:8120/categories'),
        ],
      }
      const done = (label: string, specState: string, extra: object = {}) => ({
        label, error: '', passed: specState === 'matched', skipped: false, statusCode: 200, durationMs: 4, extracted: {},
        assertions: [{ type: 'status', message: 'status = 200', passed: true }], specState, ...extra,
      })
      b['scenarioResult'] = {
        title: 'Product lifecycle', error: '', passed: false, durationMs: 31,
        steps: [
          done('List products', 'matched'),
          done('Get a product', 'violations', {
            specViolations: ['Error at "/price": value must be a number'],
            specFindings: [{
              kind: 'wrong-type', part: 'response body', where: 'price', spec: 'GET /products/{id} → response 200 → body schema',
              summary: '"price" is text, the spec says number', expected: 'number', got: '"9.99"',
              why: 'The server sent text ("9.99") where the spec declares a number.', fix: 'Send a number from the server, or change the spec.',
              detail: 'Error at "/price": value must be a number',
            }],
          }),
          done('List categories', 'violations', {
            specViolations: ['Error at "/0/name": property "name" is missing'],
            specFindings: [{
              kind: 'missing-field', part: 'response body', where: '[0].name', spec: 'GET /categories → response 200 → body schema',
              summary: 'The field "name" is missing', expected: 'a "name" field', got: 'no "name" field',
              why: 'The spec lists "name" as required in every category, but the server left it out.', fix: 'Send "name" with every category, or make it optional in the spec.',
              detail: 'Error at "/0/name": property "name" is missing',
            }],
          }),
        ],
      }
    },
    { theme },
  )
  await gotoApp(page)
  await page.getByTitle('Scenarios').click()
  await page.locator('aside').getByRole('button', { name: 'Product lifecycle' }).click()
  await page.getByRole('button', { name: /^Run$/ }).click()
  await page.getByRole('button', { name: 'Setup' }).click()
  await page.getByRole('tab', { name: 'result' }).click()
  await page.getByRole('region', { name: 'Spec conformance' }).waitFor({ timeout: 5_000 })
}

for (const theme of ['dark', 'light'] as const) {
  const tag = theme === 'dark' ? '@screenshot-conformance' : '@screenshot-conformance-light'
  const suffix = theme === 'dark' ? '' : '-light'

  test(`${tag} check conformance page`, async ({ page }) => {
    await openConformance(page, theme)
    await page.screenshot({ path: shot(`15-conformance-results${suffix}.png`), animations: 'disabled' })
  })

  test(`${tag} scenario spec conformance`, async ({ page }) => {
    await openScenarioResult(page, theme)
    await page.screenshot({ path: shot(`16-scenario-spec-conformance${suffix}.png`), animations: 'disabled' })
  })
}
