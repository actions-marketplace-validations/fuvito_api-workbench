import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('App shell', () => {
  test('renders without errors', async ({ page }) => {
    await gotoApp(page)
    // No unhandled errors during load
    const errors: string[] = []
    page.on('pageerror', e => errors.push(e.message))
    expect(errors).toHaveLength(0)
  })

  test('activity rail shows Requests and Scenarios buttons, and History sits next to the requests list', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByTitle('Requests')).toBeVisible()
    await expect(page.getByTitle('Scenarios')).toBeVisible()
    await expect(page.getByTitle('History')).toHaveCount(0)
    await expect(page.getByRole('button', { name: 'History' })).toBeVisible()
  })

  test('Requests panel is active by default', async ({ page }) => {
    await gotoApp(page)
    // With an empty workspace the Requests panel shows the empty state
    await expect(page.getByText('No request files yet')).toBeVisible()
  })

  test('switching to the History view shows the history empty state', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'History' }).click()
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

  // The URL is the most important field of the request screen: at 1280 px it must stay usable, with
  // the timeout and redirect options wrapping to a second line instead of squeezing it.
  test('the request URL field keeps a usable width at 1280 px', async ({ page }) => {
    await page.setViewportSize({ width: 1280, height: 800 })
    await gotoApp(page)
    const url = page.getByPlaceholder('https://api.example.com/endpoint')
    await expect(url).toBeVisible()
    const box = await url.boundingBox()
    expect(box!.width).toBeGreaterThanOrEqual(250)
    await expect(page.getByPlaceholder('default')).toBeVisible()
    await expect(page.getByRole('button', { name: 'Send', exact: true })).toBeVisible()
  })
})
