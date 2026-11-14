/**
 * Light-mode screenshot suite — same scenes as screenshots.spec.ts but with
 * theme: 'light' injected into the mock bridge settings.
 *
 * Run with: pnpm test -- --grep @screenshot-light
 */
import { test, gotoApp } from '../../fixtures/test.js'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const IMAGES_DIR = path.join(__dirname, '..', '..', '..', '..', '..', 'docs', 'images')

const VIEWPORT = { width: 1280, height: 800 }

function shot(name: string) {
  return path.join(IMAGES_DIR, name)
}

/** Override bridge settings to light theme before the app boots. */
function useLightTheme(page: import('@playwright/test').Page) {
  return page.addInitScript(() => {
    // Runs after setupMockBridge — override the theme key.
    const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
    if (b?.settings) (b.settings as Record<string, unknown>)['theme'] = 'light'
  })
}

const SCENARIO = { title: 'User flow', path: '/ws/scenarios/user-flow.yaml', stepCount: 2 }

const SCENARIO_DETAIL = {
  path: '/ws/scenarios/user-flow.yaml',
  title: 'User flow',
  description: 'Login, create a user, fetch it back, then delete it.',
  error: '',
  steps: [
    {
      title: 'Login',
      method: 'POST',
      url: 'https://api.example.com/auth/login',
      requestRef: '',
      failPolicy: 'stop',
      assertions: [
        { type: 'status', path: '', operator: 'eq', expected: 200, schema: '' },
        { type: 'jsonpath', path: '$.token', operator: 'ne', expected: '', schema: '' },
      ],
      extractions: [{ name: 'token', source: 'jsonpath', path: '$.token' }],
    },
    {
      title: 'Create user',
      method: 'POST',
      url: 'https://api.example.com/users',
      requestRef: '',
      failPolicy: '',
      bodyType: 'json',
      bodyContent: '{"name":"Diana Prince","email":"diana@example.com"}',
      headers: [{ name: 'Authorization', value: 'Bearer {{token}}' }],
      assertions: [
        { type: 'status', path: '', operator: 'eq', expected: 201, schema: '' },
        { type: 'jsonpath', path: '$.id', operator: 'gt', expected: 0, schema: '' },
      ],
      extractions: [{ name: 'userId', source: 'jsonpath', path: '$.id' }],
    },
  ],
}

const SCENARIO_RESULT = {
  title: 'User flow',
  error: '',
  passed: true,
  durationMs: 148,
  steps: [
    {
      title: 'Login',
      error: '',
      passed: true,
      skipped: false,
      statusCode: 200,
      durationMs: 62,
      extracted: { token: 'eyJhbGciOiJIUzI1NiJ9...' },
      assertions: [
        { type: 'status', message: 'status eq 200', passed: true },
        { type: 'jsonpath', message: '$.token ne ""', passed: true },
      ],
    },
    {
      title: 'Create user',
      error: '',
      passed: true,
      skipped: false,
      statusCode: 201,
      durationMs: 86,
      extracted: { userId: '7' },
      assertions: [
        { type: 'status', message: 'status eq 201', passed: true },
        { type: 'jsonpath', message: '$.id gt 0', passed: true },
      ],
    },
  ],
}

// ── 1. App shell ──────────────────────────────────────────────────────────────

test('@screenshot-light app shell — initial state', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await useLightTheme(page)
  await gotoApp(page)
  await page.screenshot({ path: shot('01-app-shell-light.png'), animations: 'disabled' })
})

// ── 2. Request editor — params tab ───────────────────────────────────────────

test('@screenshot-light request editor — params tab', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await useLightTheme(page)
  await gotoApp(page)

  await page.getByRole('combobox').first().selectOption('POST')
  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users')

  await page.getByRole('button', { name: 'Params' }).click()
  await page.getByText('+ Add row').first().click()
  await page.getByPlaceholder('Parameter').last().fill('include')
  await page.getByPlaceholder('Value').last().fill('roles')
  await page.getByText('+ Add row').first().click()
  await page.getByPlaceholder('Parameter').last().fill('page')
  await page.getByPlaceholder('Value').last().fill('1')

  await page.screenshot({ path: shot('02-request-params-light.png'), animations: 'disabled' })
})

// ── 3. Response viewer ────────────────────────────────────────────────────────

test('@screenshot-light response viewer — JSON response', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await useLightTheme(page)

  await page.addInitScript(() => {
    const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
    b['executeResult'] = {
      statusCode: 200,
      status: 'OK',
      body: JSON.stringify({
        id: 7,
        name: 'Diana Prince',
        email: 'diana@example.com',
        role: 'user',
        createdAt: '2026-11-06T10:34:00Z',
      }, null, 2),
      headers: [
        { name: 'Content-Type', value: 'application/json' },
        { name: 'X-Request-Id', value: 'req_8f2a9c1d' },
      ],
      durationMs: 86,
      size: 148,
      error: '',
      timing: { dnsMs: 2, connectMs: 8, tlsMs: 22, ttfbMs: 48, transferMs: 6 },
    }
  })

  await gotoApp(page)
  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users/7')
  await page.getByRole('button', { name: 'Send' }).click()
  await page.locator('[data-testid="response-status"]').waitFor({ timeout: 5_000 })

  await page.screenshot({ path: shot('04-response-viewer-light.png'), animations: 'disabled' })
})

// ── 4. Scenario — run result (PASSED) ─────────────────────────────────────────

test('@screenshot-light scenario — run result PASSED', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await useLightTheme(page)
  await page.addInitScript(`
    (function() {
      window.__bridge.workspacePath = '/workspace';
      window.__bridge.scenarios = [${JSON.stringify(SCENARIO)}];
      window.__bridge.scenarioDetail = ${JSON.stringify(SCENARIO_DETAIL)};
      window.__bridge.scenarioResult = ${JSON.stringify(SCENARIO_RESULT)};
    })();
  `)
  await gotoApp(page)
  await page.getByTitle('Scenarios').click()
  await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
  await page.getByText('Login').first().waitFor()
  await page.getByRole('button', { name: 'Run', exact: true }).click()
  await page.getByText('PASSED').first().waitFor({ timeout: 5_000 })

  await page.screenshot({ path: shot('07-scenario-result-light.png'), animations: 'disabled' })
})

// ── 5. Settings panel ─────────────────────────────────────────────────────────

test('@screenshot-light settings panel', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await useLightTheme(page)
  await gotoApp(page)
  await page.getByTitle('Settings', { exact: true }).click()
  await page.getByText('Appearance').waitFor({ timeout: 3_000 })

  await page.screenshot({ path: shot('09-settings-light.png'), animations: 'disabled' })
})
