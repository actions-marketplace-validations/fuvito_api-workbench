import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('App shell', () => {
  test('renders without errors', async ({ page }) => {
    await gotoApp(page)
    // No unhandled errors during load
    const errors: string[] = []
    page.on('pageerror', e => errors.push(e.message))
    expect(errors).toHaveLength(0)
  })

  test('sidebar is visible with workspace and history tabs', async ({ page }) => {
    await gotoApp(page)
    const sidebar = page.locator('aside')
    await expect(sidebar.getByRole('button', { name: /workspace/i }).first()).toBeVisible()
    await expect(sidebar.getByRole('button', { name: /history/i }).first()).toBeVisible()
  })

  test('workspace tab is active by default', async ({ page }) => {
    await gotoApp(page)
    // "New Workspace" button only appears on the workspace tab
    await expect(page.getByRole('button', { name: 'New Workspace' })).toBeVisible()
  })

  test('switching to history tab shows history panel', async ({ page }) => {
    await gotoApp(page)
    const sidebar = page.locator('aside')
    await sidebar.getByRole('button', { name: /history/i }).first().click()
    await expect(page.getByText('No requests yet.')).toBeVisible()
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
