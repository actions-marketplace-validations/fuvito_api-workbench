import { test, expect, gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'

async function openHelp(page: Page, item: string) {
  await page.getByRole('menuitem', { name: 'Help' }).click()
  await page.getByRole('menuitem', { name: item }).click()
  await expect(page.getByTestId('help-panel')).toBeVisible()
}

async function panelWidth(page: Page): Promise<number> {
  const box = await page.getByTestId('help-panel').boundingBox()
  if (!box) throw new Error('the Help panel has no box')
  return box.width
}

test.describe('Help panel width', () => {
  test('takes about 90% of the window and grows and shrinks with it', async ({ page }) => {
    await page.setViewportSize({ width: 1000, height: 800 })
    await gotoApp(page)
    await openHelp(page, 'User Guide')
    const narrow = await panelWidth(page)
    expect(narrow).toBeGreaterThan(880)
    expect(narrow).toBeLessThanOrEqual(920)

    await page.setViewportSize({ width: 1600, height: 800 })
    const wide = await panelWidth(page)
    expect(wide).toBeGreaterThan(1420)
    expect(wide).toBeLessThanOrEqual(1460)
    expect(wide).toBeGreaterThan(narrow)
  })

  test('does not stretch across a very large screen', async ({ page }) => {
    await page.setViewportSize({ width: 2800, height: 900 })
    await gotoApp(page)
    await openHelp(page, 'User Guide')
    expect(await panelWidth(page)).toBeLessThanOrEqual(1800)
  })

  test('every tab uses the same width', async ({ page }) => {
    await page.setViewportSize({ width: 1400, height: 800 })
    await gotoApp(page)
    await openHelp(page, 'User Guide')
    const guide = await panelWidth(page)
    for (const tab of ['Format Spec', 'Example Scenario', 'AI Prompt', 'What We Check']) {
      await page.getByRole('button', { name: tab }).click()
      expect(Math.abs((await panelWidth(page)) - guide)).toBeLessThan(1)
    }
  })
})
