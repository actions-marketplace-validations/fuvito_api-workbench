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

  test('a workspace starts on the Workspace tab, then the first request, with the Workspace tab active', async ({ page }) => {
    await page.addInitScript(`
      window.__bridge.workspacePath = '';
      window.__bridge.fileTree = [{ name: 'health.yaml', path: '/workspace/requests/health.yaml', isDir: false, method: 'GET' }];
      window.__bridge.loadRequestResult = { title: 'Health Check', method: 'GET', url: 'http://localhost:8090/health', filePath: '/workspace/requests/health.yaml' };
    `)
    await page.goto('/')
    await page.getByRole('button', { name: /Open Workspace/ }).first().click()
    await expect(page.getByTestId('workspace-page')).toBeVisible()

    const tabs = page.getByRole('tab')
    await expect(tabs).toHaveCount(2)
    await expect(tabs.nth(0)).toContainText('Workspace')
    await expect(tabs.nth(1)).toContainText('Health Check')
    await expect(tabs.nth(0)).toHaveAttribute('aria-selected', 'true')
    await expect(tabs.nth(1)).toHaveAttribute('aria-selected', 'false')
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

  // The rail follows the tabs: the active tab decides which icon is lit. A page tab (Workspace,
  // Check Conformance) has no sidebar; coming back from a page to a request, scenario or Contracts
  // tab shows the sidebar again. A rail click still wins until the next tab switch.
  test('clicking tabs lights the matching rail icon and shows or hides the sidebar', async ({ page }) => {
    await gotoApp(page)
    const rail = page.getByRole('navigation', { name: 'Activity' })
    const lit = async (name: string) => expect(page.getByRole('button', { name, exact: true }).first()).toHaveAttribute('aria-current', 'page')
    const notLit = async (name: string) => expect(page.getByRole('button', { name, exact: true }).first()).not.toHaveAttribute('aria-current', 'page')
    const sidebar = page.getByTestId('sidebar')

    // Open the Workspace page: it is lit alone and the sidebar is gone.
    await rail.getByRole('button', { name: 'Workspace' }).click()
    await lit('Workspace')
    await expect(sidebar).toHaveCount(0)

    // Back to the request tab: Requests is lit and its sidebar is shown again.
    await page.getByRole('tab', { name: /New Request/ }).click()
    await lit('Requests')
    await notLit('Workspace')
    await expect(sidebar).toBeVisible()

    // A sidebar the user collapses stays collapsed, and the active tab's icon stays lit.
    await rail.getByRole('button', { name: 'Requests' }).click()
    await expect(sidebar).toHaveCount(0)
    await lit('Requests')

    // The Workspace tab, then Check Conformance, each light their own icon with no sidebar.
    await page.getByRole('tab', { name: /Workspace/ }).click()
    await lit('Workspace')
    await expect(sidebar).toHaveCount(0)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await lit('Check spec conformance')
    await notLit('Workspace')
    await expect(sidebar).toHaveCount(0)

    // Back to the request tab from a page: the sidebar returns, pointed at Requests.
    await page.getByRole('tab', { name: /New Request/ }).click()
    await lit('Requests')
    await notLit('Check spec conformance')
    await expect(sidebar).toBeVisible()
    await expect(sidebar.getByText('Filter requests…').or(sidebar.getByText('No request files yet'))).toBeVisible()
  })

  test('a scenario tab clicked after the Workspace page shows the Scenarios sidebar', async ({ page }) => {
    await page.addInitScript(`
      window.__bridge.scenarios = [{ title: 'User flow', path: '/workspace/scenarios/user-flow.yaml', stepCount: 1 }];
      window.__bridge.scenarioDetail = { path: '/workspace/scenarios/user-flow.yaml', title: 'User flow', description: '', error: '', steps: [] };
    `)
    await gotoApp(page)
    const rail = page.getByRole('navigation', { name: 'Activity' })
    const sidebar = page.getByTestId('sidebar')

    await rail.getByRole('button', { name: 'Scenarios' }).click()
    await sidebar.getByText('User flow').click()
    await expect(page.getByRole('tab', { name: /User flow/ })).toBeVisible()
    await expect(rail.getByRole('button', { name: 'Scenarios' })).toHaveAttribute('aria-current', 'page')

    await rail.getByRole('button', { name: 'Workspace' }).click()
    await expect(sidebar).toHaveCount(0)

    await page.getByRole('tab', { name: /User flow/ }).click()
    await expect(sidebar).toBeVisible()
    await expect(rail.getByRole('button', { name: 'Scenarios' })).toHaveAttribute('aria-current', 'page')
    await expect(sidebar.getByRole('button', { name: 'Run Log' }).or(sidebar.getByText('Run Log'))).toBeVisible()
  })
})
