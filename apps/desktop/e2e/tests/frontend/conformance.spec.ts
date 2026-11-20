import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('Check Conformance page', () => {
  test('runs a read-only check and shows the totals and the violation', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await expect(page.getByRole('heading', { name: 'Check Conformance' })).toBeVisible()
    await expect(page.getByRole('tab', { name: /Check Conformance/ })).toBeVisible()

    const run = page.getByRole('button', { name: 'Run conformance check' })
    await expect(run).toBeDisabled()

    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await expect(page.getByLabel(/^Base URL/)).toHaveValue('http://localhost:8120')
    await expect(page.getByTestId('conform-send-note')).toContainText('Will send 3 read requests. 2 write operations are skipped.')

    await run.click()
    const summary = page.getByRole('region', { name: 'Summary' })
    for (const label of ['Covered', 'Not covered', 'Passing', 'Failing']) await expect(summary).toContainText(label)
    await expect(summary).toContainText('read requests only')

    await expect(page.getByTestId('conform-headline')).toHaveText('1 of 5 operations break the spec')
    const problem = page.getByRole('article', { name: 'GET /items/{{id}} fails' })
    await expect(problem).toContainText('"price" is text, the spec says number')
    await expect(problem).toContainText('Send a number from the server')
    await expect(page.getByRole('region', { name: 'Passing operations' })).toContainText('/items')
    await expect(page.getByRole('region', { name: 'Not run' })).toContainText('write request, not sent')

    const runs = await page.evaluate(() => (window as unknown as { __bridge: { conformRuns: Array<{ allowWrites: boolean }> } }).__bridge.conformRuns)
    expect(runs).toHaveLength(1)
    expect(runs[0].allowWrites).toBe(false)
  })

  test('sending writes is an explicit choice and says how many', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')

    await page.getByLabel(/Also send write requests/).check()
    await expect(page.getByTestId('conform-send-note')).toContainText('Will send 3 read and 2 write requests')

    await page.getByRole('button', { name: 'Run conformance check' }).click()
    await expect(page.getByRole('region', { name: 'Summary' })).toContainText('reads and writes')
    await expect(page.getByRole('region', { name: 'Not run' })).toHaveCount(0)
  })

  test('a base URL that is not on this computer needs confirmation', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await page.getByLabel(/^Base URL/).fill('https://api.example.com')

    const run = page.getByRole('button', { name: 'Run conformance check' })
    await expect(run).toBeDisabled()
    await page.getByRole('checkbox', { name: /not on your computer/ }).check()
    await expect(run).toBeEnabled()
    await run.click()
    await expect(page.getByRole('region', { name: 'Summary' })).toContainText('https://api.example.com')
  })

  test('negative probes are listed with the ones the server accepted', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await page.getByText('Options', { exact: true }).click()
    await page.getByLabel(/Also run negative probes/).check()
    await page.getByRole('button', { name: 'Run conformance check' }).click()
    await expect(page.getByRole('region', { name: 'Negative probes' })).toContainText('1 accepted by the server')
  })

  test('Get CLI command opens the CLI builder for conform', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByRole('button', { name: 'Get CLI command' }).click()
    await expect(page.getByRole('heading', { name: 'CLI Command Builder' })).toBeVisible()
  })
})
