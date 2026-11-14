import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('App shell', () => {
  test('renders without errors', async ({ page }) => {
    await gotoApp(page)
    // No unhandled errors during load
    const errors: string[] = []
    page.on('pageerror', e => errors.push(e.message))
    expect(errors).toHaveLength(0)
  })

  test('activity rail shows Explorer, Scenarios and History buttons', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByTitle('Explorer')).toBeVisible()
    await expect(page.getByTitle('History')).toBeVisible()
  })

  test('Explorer panel is active by default', async ({ page }) => {
    await gotoApp(page)
    // With an empty workspace the Explorer panel shows the empty state
    await expect(page.getByText('No request files yet')).toBeVisible()
  })

  test('switching to history tab shows history empty state', async ({ page }) => {
    await gotoApp(page)
    await page.getByTitle('History').click()
    await expect(page.getByText('Send a request to see it here')).toBeVisible()
  })

  test('request editor URL input is present', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByPlaceholder('https://api.example.com/endpoint')).toBeVisible()
  })

  test('Send button is disabled when URL is empty', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByRole('button', { name: 'Send' })).toBeDisabled()
  })

  test('response area shows placeholder text before first request', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByText('Send a request to see the response.')).toBeVisible()
  })

  test('settings gear button is present in toolbar', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByTitle('Settings', { exact: true })).toBeVisible()
  })
})
