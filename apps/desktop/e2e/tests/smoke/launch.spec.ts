/**
 * Smoke tests: run against `wails dev` (real Go backend + Wails IPC).
 * Start: E2E_SMOKE=1 pnpm test:smoke
 *
 * These do NOT inject mock bridges — they test against the real binary.
 */
import { test, expect } from '@playwright/test'

test.describe('App launch (smoke)', () => {
  test('renders at the wails dev URL', async ({ page }) => {
    await page.goto('/')
    // Wait up to 15s for the Vite bundle to load — wails dev is slower to start
    await page.getByPlaceholder('https://api.example.com/endpoint').waitFor({ timeout: 15_000 })
  })

  test('URL input is interactive', async ({ page }) => {
    await page.goto('/')
    const urlInput = page.getByPlaceholder('https://api.example.com/endpoint')
    await urlInput.waitFor({ timeout: 15_000 })
    await urlInput.fill('https://httpbin.org/get')
    await expect(urlInput).toHaveValue('https://httpbin.org/get')
  })

  test('Send button exists and is disabled with empty URL', async ({ page }) => {
    await page.goto('/')
    await page.getByPlaceholder('https://api.example.com/endpoint').waitFor({ timeout: 15_000 })
    await expect(page.getByRole('button', { name: 'Send' })).toBeDisabled()
  })

  test('sidebar shows workspace and history tabs', async ({ page }) => {
    await page.goto('/')
    await page.getByPlaceholder('https://api.example.com/endpoint').waitFor({ timeout: 15_000 })
    await expect(page.getByRole('button', { name: /workspace/i })).toBeVisible()
    await expect(page.getByRole('button', { name: /history/i })).toBeVisible()
  })

  test('settings panel opens from gear button', async ({ page }) => {
    await page.goto('/')
    await page.getByPlaceholder('https://api.example.com/endpoint').waitFor({ timeout: 15_000 })
    await page.getByTitle(/settings/i).click()
    await expect(page.getByText(/font size|theme|timeout/i)).toBeVisible({ timeout: 5_000 })
  })
})
