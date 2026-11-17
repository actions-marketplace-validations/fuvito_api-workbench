/**
 * CLI Command Builder screenshot suite — not a pass/fail test suite.
 * Captures the builder (a complete command, and the validation-problems state) in dark and light themes
 * and saves PNG files to docs/images/ for the README and the user guide.
 *
 * Run with: pnpm test -- --grep @screenshot-cli
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

async function openBuilder(page: Page, theme: 'dark' | 'light') {
  await page.setViewportSize(VIEWPORT)
  await page.addInitScript(
    ({ theme }) => {
      const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
      ;(b['settings'] as Record<string, unknown>)['theme'] = theme
    },
    { theme },
  )
  await gotoApp(page)
  await page.getByRole('button', { name: 'CLI Command Builder' }).click()
  await page.getByLabel('Choose Scenario from the workspace').selectOption('/workspace/scenarios/user-lifecycle.yaml')
  await page.getByLabel('Choose Environment from the workspace').selectOption('/workspace/environments/ci.yaml')
  await page.getByLabel('Coverage report').check()
  await page.getByLabel('Report format').selectOption('html')
  await page.getByRole('button', { name: 'Use reports/run-user-lifecycle.html' }).click()
  await page.getByText('Set these where the command runs').waitFor({ timeout: 5_000 })
  await page.getByTestId('cli-command').filter({ hasText: '--output reports/run-user-lifecycle.html' }).waitFor({ timeout: 5_000 })
  await page.getByText(/printed to the screen/).waitFor({ state: 'detached', timeout: 5_000 })
}

for (const theme of ['dark', 'light'] as const) {
  const tag = theme === 'dark' ? '@screenshot-cli' : '@screenshot-cli-light'
  const suffix = theme === 'dark' ? '' : '-light'

  test(`${tag} cli builder command`, async ({ page }) => {
    await openBuilder(page, theme)
    await page.screenshot({ path: shot(`13-cli-builder-command${suffix}.png`), animations: 'disabled' })
  })

  test(`${tag} cli builder problems`, async ({ page }) => {
    await page.setViewportSize(VIEWPORT)
    await page.addInitScript(
      ({ theme }) => {
        const b = (window as unknown as Record<string, Record<string, unknown>>)['__bridge']
        ;(b['settings'] as Record<string, unknown>)['theme'] = theme
      },
      { theme },
    )
    await gotoApp(page)
    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await page.getByLabel('Coverage report').check()
    await page.getByText('Scenario is required').waitFor({ timeout: 5_000 })
    await page.screenshot({ path: shot(`14-cli-builder-problems${suffix}.png`), animations: 'disabled' })
  })
}
