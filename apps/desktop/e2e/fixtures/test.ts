import { test as base, expect, type Page } from '@playwright/test'
import { setupMockBridge } from './mockBridge.js'

export { expect }

/** Extended test that injects the mock bridge before every navigation. */
export const test = base.extend<{ page: Page }>({
  page: async ({ page }, use) => {
    await page.addInitScript(setupMockBridge)
    await use(page)
  },
})

/** Navigate to the app root and wait for the URL input to appear. */
export async function gotoApp(page: Page) {
  await page.goto('/')
  await page.getByPlaceholder('https://api.example.com/endpoint').waitFor({ timeout: 10_000 })
}
