import { test, expect, gotoApp } from '../../fixtures/test.js'

const overview = {
  open: true, path: '/workspace', name: 'workspace', temporary: false,
  requests: 3, scenarios: 2, environments: 1,
  environmentName: 'Staging', environmentPath: '/workspace/environments/staging.yaml', environmentSource: 'remembered',
  envVars: [{ name: 'baseUrl', secret: false, set: true }, { name: 'token', secret: true, set: false, unsetRef: 'API_TOKEN' }],
  specs: [{ ref: '../specs/gone.yaml', refs: ['../specs/gone.yaml'], path: '/workspace/specs/gone.yaml', state: 'missing', scenarios: ['/workspace/scenarios/users.yaml'], coverage: [] }],
  lastRun: undefined,
  attention: [{
    id: 'spec-missing', severity: 'warning',
    what: 'The spec file "../specs/gone.yaml" does not exist.',
    where: 'used by scenarios/users.yaml',
    why: 'Responses are not checked against a spec.',
    fix: "Choose the spec file in the scenario's Setup row.",
    target: { kind: 'scenario', path: '/workspace/scenarios/users.yaml' },
  }],
  snapshotAt: '2026-11-21T10:00:00Z',
}

test.describe('Workspace page', () => {
  test('is the first rail item and shows the state of the workspace', async ({ page }) => {
    await gotoApp(page)
    await page.evaluate(o => { (window as any).__bridge.workspaceOverview = o }, overview)

    const first = page.getByRole('navigation', { name: 'Activity' }).getByRole('button').first()
    await expect(first).toHaveAccessibleName('Workspace')
    await first.click()

    await expect(page.getByRole('tab', { name: /Workspace/ })).toBeVisible()
    await expect(page.getByRole('region', { name: 'Contents' })).toContainText('3 requests')
    await expect(page.getByTestId('env-source')).toContainText('remembered for this workspace')
    await expect(page.getByTestId('workspace-reminder')).toContainText('Environment: Staging (remembered) · 1 thing needs attention')

    const card = page.getByRole('article', { name: /does not exist/ })
    await expect(card).toContainText('used by scenarios/users.yaml')
    await expect(card).toContainText('Responses are not checked against a spec.')
    await expect(page.getByRole('list', { name: 'Environment variables' })).toContainText('unset ${API_TOKEN}')
  })

  test('opens by itself when a workspace is opened and keeps the open tabs', async ({ page }) => {
    await page.addInitScript(() => { (window as any).__bridge.workspacePath = '' })
    await page.goto('/')
    await page.getByRole('button', { name: /Open Workspace/ }).first().click()

    await expect(page.getByTestId('workspace-page')).toBeVisible()
    await expect(page.getByRole('tab', { name: /Workspace/ })).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('navigation', { name: 'Activity' }).getByRole('button', { name: 'Workspace' })).toHaveAttribute('aria-current', 'page')
    // A workspace that was just opened starts on the full-width page, with the sidebar closed.
    await expect(page.getByTestId('sidebar')).toHaveCount(0)
  })

  test('shows what closing will remember when unsaved tabs ask for a confirmation', async ({ page }) => {
    await gotoApp(page)
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com/a')
    await page.getByRole('button', { name: 'Workspace', exact: true }).click()
    await page.getByRole('button', { name: 'Close workspace' }).click()
    await expect(page.getByText(/No environment will be remembered for this workspace\./)).toBeVisible()
  })

  test('reads the workspace again when you come back to the tab, so a fixed problem is gone', async ({ page }) => {
    await gotoApp(page)
    await page.evaluate(o => { (window as any).__bridge.workspaceOverview = o }, overview)
    await page.getByRole('button', { name: 'Workspace', exact: true }).click()
    await expect(page.getByRole('article', { name: /does not exist/ })).toBeVisible()

    // The user fixes it elsewhere, then returns to the Workspace tab without pressing Refresh.
    await page.getByRole('tab', { name: /New Request/ }).click()
    await page.evaluate(() => { (window as any).__bridge.workspaceOverview = { ...(window as any).__bridge.workspaceOverview, attention: [], specs: [] } })
    await page.getByRole('tab', { name: /Workspace/ }).click()
    await expect(page.getByTestId('attention-empty')).toBeVisible()
  })

  // The Workspace page is full width, not a side panel: the rail never shows it and a panel
  // icon highlighted at the same time, and the sidebar state follows the icon last clicked.
  test('the Workspace icon closes the sidebar and lights alone; a panel icon brings the sidebar back', async ({ page }) => {
    await gotoApp(page)
    const rail = page.getByRole('navigation', { name: 'Activity' })
    const workspace = rail.getByRole('button', { name: 'Workspace' })
    const requests = rail.getByRole('button', { name: 'Requests' })

    await expect(page.getByTestId('sidebar')).toBeVisible()
    await expect(requests).toHaveAttribute('aria-current', 'page')

    await workspace.click()
    await expect(page.getByTestId('workspace-page')).toBeVisible()
    await expect(page.getByTestId('sidebar')).toHaveCount(0)
    await expect(workspace).toHaveAttribute('aria-current', 'page')
    await expect(requests).not.toHaveAttribute('aria-current', 'page')

    await requests.click()
    await expect(page.getByTestId('sidebar')).toBeVisible()
    await expect(requests).toHaveAttribute('aria-current', 'page')
    await expect(workspace).not.toHaveAttribute('aria-current', 'page')
    // The page is still the open tab; only the rail's highlight moved.
    await expect(page.getByTestId('workspace-page')).toBeVisible()

    await rail.getByRole('button', { name: 'Scenarios' }).click()
    await expect(page.getByTestId('sidebar')).toBeVisible()
    await workspace.click()
    await expect(page.getByTestId('sidebar')).toHaveCount(0)
  })

  test('Refresh reads the workspace again', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Workspace', exact: true }).click()
    await expect(page.getByTestId('attention-empty')).toBeVisible()
    await page.evaluate(o => { (window as any).__bridge.workspaceOverview = o }, overview)
    await page.getByTestId('workspace-page').getByRole('button', { name: 'Refresh' }).click()
    await expect(page.getByRole('article', { name: /does not exist/ })).toBeVisible()
  })
})
