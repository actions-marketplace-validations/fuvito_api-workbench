/**
 * Screenshot capture suite — not a pass/fail test suite.
 * Navigates to key UI states and saves PNG files to docs/images/
 * for use in the product presentation.
 *
 * Run with: pnpm test -- --grep @screenshot
 */
import { test, gotoApp } from '../../fixtures/test.js'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const IMAGES_DIR = path.join(__dirname, '..', '..', '..', '..', '..', 'docs', 'images')

const VIEWPORT = { width: 1920, height: 1080 }

function shot(name: string) {
  return path.join(IMAGES_DIR, name)
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

test('@screenshot app shell — initial state', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await gotoApp(page)
  await page.screenshot({ path: shot('01-app-shell.png'), animations: 'disabled' })
})

// ── 2. Request editor — params tab ───────────────────────────────────────────

test('@screenshot request editor — params tab', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
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

  await page.screenshot({ path: shot('02-request-params.png'), animations: 'disabled' })
})

// ── 3. Request editor — auth tab ─────────────────────────────────────────────

test('@screenshot request editor — bearer auth tab', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await gotoApp(page)

  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users')
  await page.getByRole('button', { name: 'Auth' }).click()
  await page.locator('select').nth(1).selectOption('bearer')
  await page.getByPlaceholder('eyJhbGci…').fill('eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjMifQ.abc')

  await page.screenshot({ path: shot('03-request-auth.png'), animations: 'disabled' })
})

// ── 4. Response viewer ────────────────────────────────────────────────────────

test('@screenshot response viewer — JSON response', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)

  // Override response to richer JSON
  await page.addInitScript(`
    (function() {
      window.__bridge = window.__bridge || {};
    })();
  `)
  await page.addInitScript(`
    (function() {
      var orig = window.__bridge;
      var patched = Object.assign({}, orig, {
        executeResult: {
          statusCode: 200,
          status: 'OK',
          body: JSON.stringify({
            id: 7,
            name: "Diana Prince",
            email: "diana@example.com",
            role: "user",
            createdAt: "2026-11-06T10:34:00Z"
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
      });
      window.__bridge = patched;
    })();
  `)

  await gotoApp(page)
  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users/7')
  await page.getByRole('button', { name: 'Send' }).click()
  await page.locator('[data-testid="response-status"]').waitFor({ timeout: 5_000 })

  await page.screenshot({ path: shot('04-response-viewer.png'), animations: 'disabled' })
})

// ── 5. Response viewer — headers tab ─────────────────────────────────────────

test('@screenshot response viewer — headers tab', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await gotoApp(page)

  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users')
  await page.getByRole('button', { name: 'Send' }).click()
  await page.locator('[data-testid="response-status"]').waitFor({ timeout: 5_000 })
  await page.getByRole('button', { name: 'Headers' }).nth(1).click()

  await page.screenshot({ path: shot('05-response-headers.png'), animations: 'disabled' })
})

// ── 6. Scenario — view mode ───────────────────────────────────────────────────

test('@screenshot scenario — view mode with steps', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(`
    (function() {
      window.__bridge = window.__bridge || {};
      window.__bridge.workspacePath = '/workspace';
      window.__bridge.scenarios = [${JSON.stringify(SCENARIO)}];
      window.__bridge.scenarioDetail = ${JSON.stringify(SCENARIO_DETAIL)};
    })();
  `)
  await gotoApp(page)
  await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
  await page.getByText('Login').first().waitFor()

  await page.screenshot({ path: shot('06-scenario-view.png'), animations: 'disabled' })
})

// ── 7. Scenario — run result (PASSED) ─────────────────────────────────────────

test('@screenshot scenario — run result PASSED', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(`
    (function() {
      window.__bridge.workspacePath = '/workspace';
      window.__bridge.scenarios = [${JSON.stringify(SCENARIO)}];
      window.__bridge.scenarioDetail = ${JSON.stringify(SCENARIO_DETAIL)};
      window.__bridge.scenarioResult = ${JSON.stringify(SCENARIO_RESULT)};
    })();
  `)
  await gotoApp(page)
  await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
  await page.getByText('Login').first().waitFor()
  await page.getByRole('button', { name: 'Run', exact: true }).click()
  await page.getByText('PASSED').first().waitFor({ timeout: 5_000 })

  await page.screenshot({ path: shot('07-scenario-result.png'), animations: 'disabled' })
})

// ── 8. Scenario — visual editor ───────────────────────────────────────────────

test('@screenshot scenario — visual editor', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(`
    (function() {
      window.__bridge.workspacePath = '/workspace';
      window.__bridge.scenarios = [${JSON.stringify(SCENARIO)}];
      window.__bridge.scenarioDetail = ${JSON.stringify(SCENARIO_DETAIL)};
    })();
  `)
  await gotoApp(page)
  await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
  await page.getByText('Login').first().waitFor()
  await page.getByRole('button', { name: 'Edit', exact: true }).click()
  await page.getByPlaceholder('Step title').first().waitFor()

  await page.screenshot({ path: shot('08-scenario-editor.png'), animations: 'disabled' })
})

// ── 9. Settings panel ─────────────────────────────────────────────────────────

test('@screenshot settings panel', async ({ page }) => {
  await page.setViewportSize(VIEWPORT)
  await gotoApp(page)
  await page.getByTitle('Settings', { exact: true }).click()
  // Wait for the settings panel to appear
  await page.getByText('Appearance').waitFor({ timeout: 3_000 })

  await page.screenshot({ path: shot('09-settings.png'), animations: 'disabled' })
})
