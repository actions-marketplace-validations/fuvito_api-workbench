/**
 * Contracts page screenshot suite — not a pass/fail test suite.
 * Captures the Contracts sidebar, operation table, quality signals and export
 * menu in dark and light themes and saves PNG files to docs/images/ for the
 * README and the web site.
 *
 * Run with: pnpm test -- --grep @screenshot-contracts
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

const ago = (minutes: number) => new Date(Date.now() - minutes * 60_000).toISOString()

const USERS_SPEC = 'specs/users-api.yaml'

const USERS_RESULTS: Record<string, { lastPassed: boolean; lastRunAt: string }> = {
  'GET /health': { lastPassed: true, lastRunAt: ago(4) },
  'POST /auth/login': { lastPassed: true, lastRunAt: ago(4) },
  'GET /auth/profile': { lastPassed: true, lastRunAt: ago(4) },
  'GET /users': { lastPassed: true, lastRunAt: ago(4) },
  'POST /users': { lastPassed: true, lastRunAt: ago(4) },
  'GET /users/{id}': { lastPassed: true, lastRunAt: ago(4) },
  'PUT /users/{id}': { lastPassed: false, lastRunAt: ago(4) },
  'DELETE /users/{id}': { lastPassed: true, lastRunAt: ago(4) },
  'GET /users/{id}/orders': { lastPassed: true, lastRunAt: ago(37) },
  'POST /orders': { lastPassed: true, lastRunAt: ago(37) },
}

const COVERAGE = [
  {
    specFile: USERS_SPEC,
    covered: 10,
    total: 14,
    percent: (10 / 14) * 100,
    totalRuns: 23,
    passedRuns: 19,
    failedRuns: 4,
    lastRunAt: ago(4),
    exercised: Object.keys(USERS_RESULTS),
    missed: ['PATCH /users/{id}', 'POST /users/{id}/avatar', 'GET /orders/{id}', 'DELETE /orders/{id}'],
    operationResults: USERS_RESULTS,
    quality: {
      neverExercised: ['PATCH /users/{id}', 'POST /users/{id}/avatar', 'GET /orders/{id}', 'DELETE /orders/{id}'],
      happyPathOnly: ['GET /health', 'GET /users', 'POST /users', 'GET /users/{id}/orders', 'POST /orders'],
      securedNeverAuthed: ['GET /users/{id}/orders', 'POST /orders'],
      errorsNeverTriggered: [
        { operation: 'POST /auth/login', missing: [401, 429] },
        { operation: 'POST /users', missing: [400, 409] },
        { operation: 'GET /users/{id}', missing: [404] },
        { operation: 'DELETE /users/{id}', missing: [403, 404] },
      ],
    },
  },
  {
    specFile: 'specs/payments-api.yaml',
    covered: 11,
    total: 12,
    percent: (11 / 12) * 100,
    totalRuns: 41,
    passedRuns: 41,
    failedRuns: 0,
    lastRunAt: ago(55),
    exercised: [],
    missed: [],
    quality: { neverExercised: [], happyPathOnly: [], securedNeverAuthed: [], errorsNeverTriggered: [] },
  },
  {
    specFile: 'specs/legacy-orders.yaml',
    covered: 3,
    total: 9,
    percent: (3 / 9) * 100,
    totalRuns: 8,
    passedRuns: 5,
    failedRuns: 3,
    lastRunAt: ago(60 * 26),
    exercised: [],
    missed: [],
    quality: { neverExercised: [], happyPathOnly: [], securedNeverAuthed: [], errorsNeverTriggered: [] },
  },
]

async function openContracts(page: Page, theme: 'dark' | 'light') {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(
    ({ coverage, theme }) => {
      const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
      b['coverage'] = coverage
      ;(b['settings'] as Record<string, unknown>)['theme'] = theme
    },
    { coverage: COVERAGE, theme },
  )
  await gotoApp(page)
  await page.getByTitle('Contracts', { exact: true }).click()
  await page.getByRole('group', { name: 'Covered', exact: true }).waitFor({ timeout: 5_000 })
}

for (const theme of ['dark', 'light'] as const) {
  const tag = theme === 'dark' ? '@screenshot-contracts' : '@screenshot-contracts-light'
  const suffix = theme === 'dark' ? '' : '-light'

  test(`${tag} contracts overview`, async ({ page }) => {
    await openContracts(page, theme)
    await page.screenshot({ path: shot(`10-contracts-overview${suffix}.png`), animations: 'disabled' })
  })

  test(`${tag} contracts quality signals`, async ({ page }) => {
    await openContracts(page, theme)
    await page.getByText('Quality Signals').scrollIntoViewIfNeeded()
    await page.getByText('Quality Signals').evaluate(el => el.scrollIntoView({ block: 'start' }))
    await page.screenshot({ path: shot(`11-contracts-quality-signals${suffix}.png`), animations: 'disabled' })
  })

  test(`${tag} contracts export menu`, async ({ page }) => {
    await openContracts(page, theme)
    await page.getByRole('button', { name: 'Export' }).click()
    await page.getByText('Export Markdown').waitFor()
    await page.screenshot({ path: shot(`12-contracts-export-menu${suffix}.png`), animations: 'disabled' })
  })
}
