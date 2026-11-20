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

const op = (operationId: string, method: string, path: string, status: string, statusCode: number, durationMs: number, violations: string[] = []) =>
  ({ operationId, method, path, status, statusCode, durationMs, violations })

const RESULT = {
  specTitle: 'Clear Signal Conformance Server',
  baseUrl: 'http://localhost:8120',
  writes: true,
  durationMs: 38,
  total: 7,
  covered: 7,
  notCovered: 0,
  passing: 4,
  failing: 3,
  operations: [
    op('listCategories', 'GET', '/categories', 'failed', 200, 3, ['Error at "/0/name": property "name" is missing']),
    op('getHealth', 'GET', '/health', 'passed', 200, 2),
    op('listOrders', 'GET', '/orders', 'failed', 200, 3, ['Error at "/0/id": value must be an integer']),
    op('listProducts', 'GET', '/products', 'passed', 200, 2),
    op('createProduct', 'POST', '/products', 'passed', 201, 4),
    op('getProduct', 'GET', '/products/{{id}}', 'failed', 200, 3, ['Error at "/price": value must be a number']),
    op('deleteProduct', 'DELETE', '/products/{{id}}', 'passed', 204, 2),
  ],
  probes: [],
  warnings: [],
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
      svc['InspectConformSpec'] = () => Promise.resolve({ title: 'Clear Signal Conformance Server', version: '1.0.0', serverUrl: 'http://localhost:8120', reads: 5, writes: 2 })
      svc['RunConformance'] = () => Promise.resolve(result)
    },
    { theme, result: RESULT },
  )
  await gotoApp(page)
  await page.getByRole('button', { name: 'Check spec conformance' }).click()
  await page.getByLabel(/^Spec/).selectOption('/workspace/specs/conformance-openapi.yaml')
  await page.getByLabel(/^Environment/).selectOption('/workspace/environments/local.yaml')
  await page.getByLabel(/Also send write requests/).check()
  await page.getByRole('button', { name: 'Run conformance check' }).click()
  const results = page.getByRole('region', { name: 'Conformance results' })
  await results.waitFor({ timeout: 5_000 })
  // Put the totals and the whole operation table in view, not the form above them.
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
          done('Get a product', 'violations', { specViolations: ['Error at "/price": value must be a number'] }),
          done('List categories', 'violations', { specViolations: ['Error at "/0/name": property "name" is missing'] }),
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
