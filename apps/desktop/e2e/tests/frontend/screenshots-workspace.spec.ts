/**
 * Workspace page screenshot suite — not a pass/fail test suite.
 * Captures the Workspace page (state of the open workspace and what needs attention) in dark and
 * light themes and saves PNG files to docs/images/ for the README.
 *
 * Run with: pnpm test -- --grep @screenshot-workspace
 */
import { test, gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const IMAGES_DIR = path.join(__dirname, '..', '..', '..', '..', '..', 'docs', 'images')
const VIEWPORT = { width: 1440, height: 900 }

const OVERVIEW = {
  open: true, path: 'C:/work/shop-api', name: 'shop-api', temporary: false,
  requests: 21, scenarios: 8, environments: 3,
  environmentName: 'Staging', environmentPath: 'C:/work/shop-api/environments/staging.yaml', environmentSource: 'remembered',
  envVars: [
    { name: 'baseUrl', secret: false, set: true },
    { name: 'userId', secret: false, set: true },
    { name: 'apiKey', secret: true, set: false, unsetRef: 'SHOP_API_KEY' },
  ],
  specs: [
    {
      ref: '../specs/shop.yaml', refs: ['../specs/shop.yaml'], path: 'C:/work/shop-api/specs/shop.yaml', state: 'found',
      scenarios: ['a', 'b', 'c', 'd', 'e'],
      coverage: [{ ref: '../specs/shop.yaml', total: 14, covered: 10, notCovered: 4, passing: 9, failing: 1 }],
    },
    {
      ref: '../specs/legacy.yaml', refs: ['../specs/legacy.yaml'], path: 'C:/work/shop-api/specs/legacy.yaml', state: 'missing',
      scenarios: ['f'], coverage: [],
    },
  ],
  lastRun: { at: '2026-11-21T09:42:00Z', scenario: 'Checkout flow', passed: false, scenarios: 8, failingNow: 1 },
  attention: [
    {
      id: 'unset-env-var', severity: 'warning',
      what: 'The process variable SHOP_API_KEY is not set.',
      where: 'environment "Staging", variable "apiKey" refers to it as ${SHOP_API_KEY}',
      why: 'The value stays as written in every request, so requests fail for the wrong reason.',
      fix: 'Set SHOP_API_KEY in the environment that starts clrsig, or give "apiKey" a value in the environment file.',
      target: { kind: 'environment', path: 'C:/work/shop-api/environments/staging.yaml' },
    },
    {
      id: 'spec-missing', severity: 'warning',
      what: 'The spec file "../specs/legacy.yaml" does not exist.',
      where: 'used by scenarios/legacy-orders.yaml',
      why: 'Responses are not checked against a spec, and conformance and coverage are not recorded.',
      fix: "Choose the spec file in the scenario's Setup row, or put the file back at C:/work/shop-api/specs/legacy.yaml.",
      target: { kind: 'scenario', path: 'C:/work/shop-api/scenarios/legacy-orders.yaml' },
    },
  ],
  snapshotAt: '2026-11-21T09:45:00Z',
}

async function openWorkspacePage(page: Page, theme: 'dark' | 'light') {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(
    ({ theme, overview }) => {
      const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
      ;(b['settings'] as Record<string, unknown>)['theme'] = theme
      b['workspaceOverview'] = overview
      // The environment the overview names, so the sidebar's selector agrees with it.
      const staging = { name: 'Staging', path: 'C:/work/shop-api/environments/staging.yaml' }
      b['environments'] = [{ name: 'Local', path: 'C:/work/shop-api/environments/local.yaml' }, staging, { name: 'Production', path: 'C:/work/shop-api/environments/production.yaml' }]
      b['activeEnv'] = staging
    },
    { theme, overview: OVERVIEW },
  )
  await gotoApp(page)
  await page.getByRole('navigation', { name: 'Activity' }).getByRole('button', { name: 'Workspace' }).click()
  await page.getByTestId('workspace-page').waitFor({ timeout: 5_000 })
  await page.getByRole('article', { name: /SHOP_API_KEY/ }).waitFor({ timeout: 5_000 })
}

for (const theme of ['dark', 'light'] as const) {
  const tag = theme === 'dark' ? '@screenshot-workspace' : '@screenshot-workspace-light'
  const suffix = theme === 'dark' ? '' : '-light'

  test(`${tag} workspace page`, async ({ page }) => {
    await openWorkspacePage(page, theme)
    await page.screenshot({ path: path.join(IMAGES_DIR, `17-workspace-page${suffix}.png`), animations: 'disabled' })
  })
}

// The Check Conformance and CLI Builder pages with their sidebars: every rail icon opens a sidebar.
for (const theme of ['dark', 'light'] as const) {
  const tag = theme === 'dark' ? '@screenshot-workspace' : '@screenshot-workspace-light'
  const suffix = theme === 'dark' ? '' : '-light'

  test(`${tag} conformance sidebar`, async ({ page }) => {
    await openWorkspacePage(page, theme)
    await page.locator('button[aria-label="Check Conformance"]').click()
    await page.getByTestId('sidebar-conformance').getByRole('button', { name: 'specs/shop.yaml' }).click()
    await page.getByLabel(/Base URL/).waitFor({ timeout: 5_000 })
    await page.getByRole('button', { name: /Run conformance check/ }).click()
    await page.getByTestId('sidebar-last-result').waitFor({ timeout: 5_000 })
    await page.screenshot({ path: path.join(IMAGES_DIR, `18-conformance-sidebar${suffix}.png`), animations: 'disabled' })
  })

  test(`${tag} cli builder sidebar`, async ({ page }) => {
    await openWorkspacePage(page, theme)
    await page.locator('button[aria-label="CLI Command Builder"]').click()
    await page.getByTestId('sidebar-cli').getByRole('button', { name: /clrsig conform/ }).click()
    await page.getByTestId('cli-command').waitFor({ timeout: 5_000 })
    await page.screenshot({ path: path.join(IMAGES_DIR, `19-cli-builder-sidebar${suffix}.png`), animations: 'disabled' })
  })
}
