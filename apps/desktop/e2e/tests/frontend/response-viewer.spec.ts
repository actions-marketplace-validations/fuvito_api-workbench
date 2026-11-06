import { test, expect, gotoApp } from '../../fixtures/test.js'

async function sendRequest(page: Parameters<typeof gotoApp>[0]) {
  await gotoApp(page)
  await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com')
  await page.getByRole('button', { name: 'Send' }).click()
  // Wait for response to appear
  await page.locator('[data-testid="response-status"]').waitFor({ timeout: 5_000 })
}

test.describe('Response viewer', () => {
  test('shows 200 status badge after successful request', async ({ page }) => {
    await sendRequest(page)
    await expect(page.locator('[data-testid="response-status"]')).toHaveText('200')
  })

  test('shows response body', async ({ page }) => {
    await sendRequest(page)
    const body = page.locator('[data-testid="response-body"]')
    await expect(body).toContainText('ok')
    await expect(body).toContainText('true')
  })

  test('raw/pretty toggle changes body display', async ({ page }) => {
    await sendRequest(page)
    // Default is pretty — click raw toggle
    await page.getByTitle(/switch to raw view/i).click()
    // Raw body should be the compact JSON string
    await expect(page.locator('[data-testid="response-body"]')).toContainText('{"ok":true}')
    // Click back to pretty
    await page.getByTitle(/switch to pretty view/i).click()
  })

  test('copy button shows checkmark after clicking', async ({ page }) => {
    await sendRequest(page)
    // Grant clipboard permissions so writeText succeeds
    await page.context().grantPermissions(['clipboard-read', 'clipboard-write'])
    const copyBtn = page.getByTitle('Copy response body')
    await copyBtn.click()
    // After copy, the button title changes to "Copied!" — old title disappears
    await expect(page.getByTitle('Copy response body')).toBeHidden({ timeout: 2_600 })
    await expect(page.getByTitle('Copied!')).toBeVisible()
  })

  test('shows "Network Error" in red on error response', async ({ page }) => {
    await gotoApp(page)
    // Override Execute to return an error
    await page.evaluate(() => {
      ;(window as unknown as Record<string, Record<string, Record<string, Record<string, unknown>>>>)
        ['go']['main']['RequestService']['Execute'] = () =>
          Promise.resolve({
            statusCode: 0,
            statusText: '',
            body: '',
            headers: [],
            duration: 0,
            size: 0,
            error: 'connection refused',
            timing: null,
          })
    })
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://unreachable.example.com')
    await page.getByRole('button', { name: 'Send' }).click()
    await expect(page.getByText('Network Error')).toBeVisible({ timeout: 5_000 })
  })

  test('shows "Sending request…" while loading', async ({ page }) => {
    await gotoApp(page)
    await page.evaluate(() => {
      ;(window as unknown as Record<string, Record<string, Record<string, Record<string, unknown>>>>)
        ['go']['main']['RequestService']['Execute'] = () => new Promise(() => { /* never resolves */ })
    })
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com')
    await page.getByRole('button', { name: 'Send' }).click()
    await expect(page.getByText('Sending request…')).toBeVisible({ timeout: 3_000 })
  })

  test('response headers tab shows headers', async ({ page }) => {
    await sendRequest(page)
    await page.getByRole('button', { name: 'Headers' }).nth(1).click()
    // The mock returns Content-Type: application/json
    await expect(page.getByText('Content-Type')).toBeVisible()
    await expect(page.getByText('application/json')).toBeVisible()
  })

  test('timing bar is visible after successful request', async ({ page }) => {
    await sendRequest(page)
    // Timing bar contains the total duration
    await expect(page.getByText('123 ms')).toBeVisible()
  })
})
