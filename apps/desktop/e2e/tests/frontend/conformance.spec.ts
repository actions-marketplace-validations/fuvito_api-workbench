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

  test('an operation with no value for a placeholder is not sent, and the user is told what to set', async ({ page }) => {
    await page.addInitScript(() => { (window as unknown as { __bridge: Record<string, unknown> }).__bridge['conformNeedsId'] = true })
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await page.getByRole('button', { name: 'Run conformance check' }).click()

    await expect(page.getByTestId('conform-headline')).toHaveText('1 of 5 operations could not be checked')
    await expect(page.getByRole('article', { name: 'GET /items/{{id}} fails' })).toContainText('NOT SENT')
    await expect(page.getByRole('region', { name: 'Passing operations' })).not.toContainText('/items/{{id}}')
    await expect(page.getByRole('region', { name: 'Missing values' })).toContainText('no value for {{id}}')

    // One click puts the name in the overrides; typing a value and running again sends it.
    await page.getByRole('button', { name: 'Add these to Variable overrides' }).click()
    await expect(page.getByLabel('Variable overrides')).toHaveValue('id=')
    await page.getByLabel('Variable overrides').fill('id=1')
    await page.getByRole('button', { name: 'Run conformance check' }).click()
    await expect(page.getByRole('region', { name: 'Missing values' })).toHaveCount(0)
    await expect(page.getByTestId('conform-headline')).toHaveText('1 of 5 operations break the spec')
  })

  test('Get CLI command carries the page choices, read-only included', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await page.getByText('Options', { exact: true }).click()
    await page.getByLabel('Tags').fill('items, orders')
    await page.getByLabel(/Also run negative probes/).check()
    await page.getByRole('button', { name: 'Get CLI command' }).click()

    await expect(page.getByRole('heading', { name: 'CLI Command Builder' })).toBeVisible()
    const command = page.getByTestId('cli-command')
    await expect(command).toContainText('clrsig conform specs/shop.yaml')
    await expect(command).toContainText('--base-url http://localhost:8120')
    await expect(command).toContainText('--env environments/local.yaml')
    await expect(command).toContainText('--tags items,orders')
    await expect(command).toContainText('--negative')
    await expect(command).toContainText('--read-only')
  })

  test('Help > Dev Servers > Start Conformance Server is one step, like the Demo Server', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('menuitem', { name: 'Help' }).click()
    await page.getByRole('menuitem', { name: 'Dev Servers' }).hover()
    await expect(page.getByRole('menuitem', { name: 'Open Conformance Workspace' })).toHaveCount(0)
    await page.getByRole('menuitem', { name: 'Start Conformance Server' }).click()

    // The Check Conformance page opens and the server shows in the status bar.
    await expect(page.getByRole('heading', { name: 'Check Conformance' })).toBeVisible()
    await expect(page.getByText('Conformance :8120')).toBeVisible()

    await page.getByRole('menuitem', { name: 'Help' }).click()
    await page.getByRole('menuitem', { name: 'Dev Servers' }).hover()
    await expect(page.getByRole('menuitem', { name: 'Stop Conformance Server' })).toBeVisible()
  })

  test('with writes ticked the command has no --read-only', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await page.getByLabel(/^Spec/).selectOption('/workspace/specs/shop.yaml')
    await page.getByLabel(/Also send write requests/).check()
    await page.getByRole('button', { name: 'Get CLI command' }).click()
    const command = page.getByTestId('cli-command')
    await expect(command).toContainText('clrsig conform specs/shop.yaml')
    await expect(command).not.toContainText('--read-only')
  })
})
