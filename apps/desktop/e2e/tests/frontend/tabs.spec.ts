import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('Tabs', () => {
  test('starts with exactly one tab', async ({ page }) => {
    await gotoApp(page)
    await expect(page.getByRole('tab')).toHaveCount(1)
  })

  test('initial tab is active and shows "New Request"', async ({ page }) => {
    await gotoApp(page)
    const tab = page.getByRole('tab')
    await expect(tab).toHaveAttribute('aria-selected', 'true')
    await expect(tab).toContainText('New Request')
  })

  test('can open a new blank tab via toolbar button', async ({ page }) => {
    await gotoApp(page)
    await page.getByTitle('New request tab (Ctrl+N)').click()
    await expect(page.getByRole('tab')).toHaveCount(2)
  })

  test('can open a new blank tab via tab bar + button', async ({ page }) => {
    await gotoApp(page)
    await page.getByTitle('New request tab', { exact: true }).click()
    await expect(page.getByRole('tab')).toHaveCount(2)
  })

  test('new tab is active after opening', async ({ page }) => {
    await gotoApp(page)
    await page.getByTitle('New request tab', { exact: true }).click()
    // The second tab should be active (aria-selected = true)
    const tabs = page.getByRole('tab')
    await expect(tabs.last()).toHaveAttribute('aria-selected', 'true')
  })

  test('typing URL marks tab as dirty (amber dot appears)', async ({ page }) => {
    await gotoApp(page)
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com')
    // Dirty tab shows an amber dot inside the tab bar
    await expect(page.getByRole('tab').getByTitle('Unsaved changes')).toBeVisible()
  })

  test('switching between tabs preserves state', async ({ page }) => {
    await gotoApp(page)
    // Fill URL in first tab
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://api.example.com/users')
    // Open new tab
    await page.getByTitle('New request tab', { exact: true }).click()
    // New tab URL should be empty
    await expect(page.getByPlaceholder('https://api.example.com/endpoint')).toHaveValue('')
    // Switch back to first tab
    await page.getByRole('tab').first().click()
    // URL should be restored
    await expect(page.getByPlaceholder('https://api.example.com/endpoint')).toHaveValue('https://api.example.com/users')
  })

  test('can close a clean tab', async ({ page }) => {
    await gotoApp(page)
    // Open a second tab so we always have one left
    await page.getByTitle('New request tab', { exact: true }).click()
    await expect(page.getByRole('tab')).toHaveCount(2)
    // Close the second (last) tab — force click because close button is opacity-0 until hover
    await page.getByTitle('Close tab').last().click({ force: true })
    await expect(page.getByRole('tab')).toHaveCount(1)
  })

  test('dirty tab shows close confirmation when closed', async ({ page }) => {
    await gotoApp(page)
    // Make first tab dirty
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com/dirty')
    // Open a second tab
    await page.getByTitle('New request tab', { exact: true }).click()
    await expect(page.getByRole('tab')).toHaveCount(2)
    // Switch to first (dirty) tab and close it
    await page.getByRole('tab').first().click()
    await page.getByTitle('Close tab').first().click({ force: true })
    // Confirmation dialog heading should appear
    await expect(page.getByRole('heading', { name: /unsaved changes/i })).toBeVisible({ timeout: 3_000 })
  })
})
