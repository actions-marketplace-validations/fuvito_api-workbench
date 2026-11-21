import { test, expect, gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'

// A sidebar list must only scroll when its content is taller than the panel.
async function scrollingContainers(page: Page) {
  return page.evaluate(() =>
    [...document.querySelectorAll('*')]
      .filter(e => ['auto', 'scroll'].includes(getComputedStyle(e as HTMLElement).overflowY))
      .filter(e => e.scrollHeight > e.clientHeight + 1)
      .map(e => (e as HTMLElement).className.toString().slice(0, 60)),
  )
}

async function withScenarios(page: Page, list: object[]) {
  await page.addInitScript(`window.__bridge.scenarios = ${JSON.stringify(list)};`)
}

const scenario = (n: number) => ({ title: `Scenario ${n}`, path: `/workspace/scenarios/s${n}.yaml`, stepCount: 2 })

test.describe('Sidebar scrolling', () => {
  test('the Scenarios panel has no scrollbar when only a few scenarios fit', async ({ page }) => {
    await withScenarios(page, [scenario(1), scenario(2), scenario(3)])
    await gotoApp(page)
    await page.getByTitle('Scenarios').click()
    await expect(page.getByText('Scenario 3')).toBeVisible()
    expect(await scrollingContainers(page)).toEqual([])
  })

  test('the Scenarios panel still scrolls when there are many scenarios', async ({ page }) => {
    await withScenarios(page, Array.from({ length: 60 }, (_, i) => scenario(i + 1)))
    await gotoApp(page)
    await page.getByTitle('Scenarios').click()
    await expect(page.getByText('Scenario 1', { exact: true })).toBeVisible()
    expect((await scrollingContainers(page)).join('|')).toContain('overflow-auto')
  })

  test('the empty space below the scenarios is still the root drop and context-menu area', async ({ page }) => {
    await withScenarios(page, [scenario(1)])
    await gotoApp(page)
    await page.getByTitle('Scenarios').click()
    const root = page.getByTestId('scenario-root')
    await expect(root).toBeVisible()
    const box = await root.boundingBox()
    const panel = await page.locator('[data-testid="scenario-root"]').evaluate(e => e.parentElement!.getBoundingClientRect().bottom)
    expect(box!.y + box!.height).toBeGreaterThanOrEqual(panel - 8)
  })

  test('the Requests panel has no scrollbar when only a few requests fit', async ({ page }) => {
    await page.addInitScript(`window.__bridge.fileTree = ${JSON.stringify([
      { name: 'health.yaml', path: '/workspace/requests/health.yaml', isDir: false },
      { name: 'users.yaml', path: '/workspace/requests/users.yaml', isDir: false },
    ])};`)
    await gotoApp(page)
    await expect(page.getByText('health')).toBeVisible()
    expect(await scrollingContainers(page)).toEqual([])
  })
})
