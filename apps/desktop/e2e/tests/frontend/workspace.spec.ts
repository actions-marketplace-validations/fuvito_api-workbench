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
    // A workspace that was just opened starts on the Workspace tab with the Workspace sidebar showing.
    await expect(page.getByTestId('sidebar-workspace')).toBeVisible()
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

  // Every rail icon works the same way: click shows its sidebar, clicking the lit icon collapses it.
  test('every rail icon opens its own sidebar, and the lit icon collapses it', async ({ page }) => {
    await gotoApp(page)
    const sidebar = page.getByTestId('sidebar')
    const rail = { getByRole: (_: 'button', o: { name: string }) => page.locator(`button[aria-label="${o.name}"]`) }
    const icons: [string, string | null][] = [
      ['Workspace', 'sidebar-workspace'], ['Check spec conformance', 'sidebar-conformance'], ['CLI Command Builder', 'sidebar-cli'],
      ['Requests', null], ['Scenarios', null], ['Contracts', null],
    ]
    for (const [name, id] of icons) {
      const icon = rail.getByRole('button', { name })
      await icon.click()
      await expect(sidebar).toBeVisible()
      if (id) await expect(page.getByTestId(id)).toBeVisible()
      await expect(page.locator('[aria-current="page"]')).toHaveCount(1)
      await expect(icon).toHaveAttribute('aria-current', 'page')
    }
    // The lit icon collapses the sidebar and stays lit; clicking it again shows the sidebar.
    const contracts = rail.getByRole('button', { name: 'Contracts' })
    await contracts.click()
    await expect(sidebar).toHaveCount(0)
    await expect(contracts).toHaveAttribute('aria-current', 'page')
    await contracts.click()
    await expect(sidebar).toBeVisible()
  })

  test('each icon brings back the last tab opened under it', async ({ page }) => {
    await page.addInitScript(`
      window.__bridge.scenarios = [{ title: 'User flow', path: '/workspace/scenarios/user-flow.yaml', stepCount: 1 }];
      window.__bridge.scenarioDetail = { path: '/workspace/scenarios/user-flow.yaml', title: 'User flow', description: '', error: '', steps: [] };
    `)
    await gotoApp(page)
    const rail = { getByRole: (_: 'button', o: { name: string }) => page.locator(`button[aria-label="${o.name}"]`) }
    const selected = (name: RegExp) => expect(page.getByRole('tab', { name })).toHaveAttribute('aria-selected', 'true')

    await rail.getByRole('button', { name: 'Scenarios' }).click()
    await page.getByTestId('sidebar').getByText('User flow').click()
    await selected(/User flow/)

    await rail.getByRole('button', { name: 'Workspace' }).click()
    await selected(/Workspace/)
    await expect(page.getByTestId('workspace-page')).toBeVisible()

    await rail.getByRole('button', { name: 'Scenarios' }).click()
    await selected(/User flow/)
    await rail.getByRole('button', { name: 'Check spec conformance' }).click()
    await selected(/Conformance/)
    await rail.getByRole('button', { name: 'Workspace' }).click()
    await selected(/Workspace/)

    // An icon with nothing open under it shows its sidebar and keeps the current tab.
    await rail.getByRole('button', { name: 'Contracts' }).click()
    await expect(page.getByTestId('sidebar')).toBeVisible()
    await selected(/Workspace/)
  })

  // The Workspace sidebar is a summary that drives the page: the same snapshot, an environment
  // selector, and attention items that show the matching card on the page.
  test('the Workspace sidebar lists what needs attention and shows its card on the page', async ({ page }) => {
    await page.addInitScript(o => {
      (window as any).__bridge.workspaceOverview = o;
      (window as any).__bridge.environments = [{ name: 'local', path: '/workspace/environments/local.yaml' }, { name: 'staging', path: '/workspace/environments/staging.yaml' }]
    }, overview)
    await gotoApp(page)
    await page.getByRole('button', { name: 'Workspace', exact: true }).click()
    const side = page.getByTestId('sidebar-workspace')
    await expect(side).toBeVisible()
    await expect(side.getByRole('list', { name: 'Needs attention' })).toContainText('does not exist')
    await expect(side.getByTestId('sidebar-env-source')).toContainText('remembered for this workspace')
    await expect(side.getByRole('combobox', { name: 'Environment' })).toBeVisible()
    await expect(side.getByRole('button', { name: '3 requests' })).toBeVisible()

    // The page and the sidebar come from one snapshot.
    await expect(page.getByTestId('workspace-page').getByRole('article', { name: /does not exist/ })).toBeVisible()

    // Clicking an item shows the Workspace page with that card.
    await side.getByRole('button', { name: /does not exist/ }).click()
    await expect(page.getByRole('tab', { name: /Workspace/ })).toHaveAttribute('aria-selected', 'true')
    await expect(page.getByRole('article', { name: /does not exist/ })).toBeVisible()
  })

  // The Check Conformance and CLI Builder sidebars pick inputs on their pages.
  test('the Check Conformance sidebar picks the spec on the page and shows its last result', async ({ page }) => {
    await gotoApp(page)
    await page.locator('button[aria-label="Check spec conformance"]').click()
    const side = page.getByTestId('sidebar-conformance')
    await expect(side.getByRole('list', { name: 'Workspace specs' })).toContainText('specs/shop.yaml')
    await expect(side.getByRole('button', { name: 'Active environment' })).toHaveAttribute('aria-pressed', 'true')

    await side.getByRole('button', { name: 'specs/shop.yaml' }).click()
    await expect(page.locator('#conf-spec')).toHaveValue('/workspace/specs/shop.yaml')
    await expect(side.getByRole('button', { name: 'specs/shop.yaml' })).toHaveAttribute('aria-pressed', 'true')

    await side.getByRole('button', { name: 'environments/ci.yaml' }).or(side.getByRole('button', { name: 'environments/local.yaml' })).click()
    await expect(page.locator('#conf-env')).toHaveValue('/workspace/environments/local.yaml')

    await page.getByRole('button', { name: /Run conformance check/ }).click()
    await expect(side.getByTestId('sidebar-last-result')).toBeVisible()
  })

  test('the CLI Builder sidebar lists the commands and selects one on the page', async ({ page }) => {
    await gotoApp(page)
    await page.locator('button[aria-label="CLI Command Builder"]').click()
    const side = page.getByTestId('sidebar-cli')
    await expect(side.getByRole('button', { name: /clrsig run/ })).toBeVisible()
    await expect(side.getByRole('button', { name: /clrsig run/ })).toHaveAttribute('aria-pressed', 'true')
    await side.getByRole('button', { name: /clrsig conform/ }).click()
    await expect(page.getByRole('radio', { name: /Check an API against its OpenAPI spec/ })).toHaveAttribute('aria-checked', 'true')
    await expect(side.getByRole('button', { name: /clrsig conform/ })).toHaveAttribute('aria-pressed', 'true')
  })

  test('Refresh reads the workspace again', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Workspace', exact: true }).click()
    await expect(page.getByTestId('attention-empty')).toBeVisible()
    await page.evaluate(o => { (window as any).__bridge.workspaceOverview = o }, overview)
    await page.getByTestId('workspace-page').getByRole('button', { name: 'Refresh' }).click()
    await expect(page.getByRole('article', { name: /does not exist/ })).toBeVisible()
  })

  // Switching tabs lights the tab's icon and points the sidebar at its panel; it never opens or
  // closes the sidebar, which is the user's choice.
  test('clicking tabs lights the matching rail icon and leaves the sidebar as the user had it', async ({ page }) => {
    await gotoApp(page)
    const rail = page.getByRole('navigation', { name: 'Activity' })
    const lit = async (name: string) => expect(page.getByRole('button', { name, exact: true }).first()).toHaveAttribute('aria-current', 'page')
    const notLit = async (name: string) => expect(page.getByRole('button', { name, exact: true }).first()).not.toHaveAttribute('aria-current', 'page')
    const sidebar = page.getByTestId('sidebar')

    await rail.getByRole('button', { name: 'Workspace' }).click()
    await lit('Workspace')
    await expect(sidebar).toBeVisible()
    await expect(page.getByTestId('sidebar-workspace')).toBeVisible()

    await page.getByRole('tab', { name: /New Request/ }).click()
    await lit('Requests')
    await notLit('Workspace')
    await expect(sidebar).toBeVisible()
    await expect(sidebar.getByText('Filter requests…').or(sidebar.getByText('No request files yet'))).toBeVisible()

    // A sidebar the user collapses stays collapsed while tabs change; the active tab's icon is lit.
    await rail.getByRole('button', { name: 'Requests' }).click()
    await expect(sidebar).toHaveCount(0)
    await page.getByRole('tab', { name: /Workspace/ }).click()
    await lit('Workspace')
    await expect(sidebar).toHaveCount(0)
    await page.getByRole('button', { name: 'Check spec conformance' }).click()
    await expect(page.getByRole('tab', { name: /Check Conformance|Conformance/ }).first()).toBeVisible()
    await lit('Check spec conformance')
    await notLit('Workspace')
    // Clicking an icon shows its sidebar; clicking a tab afterwards leaves it showing and re-points it.
    await expect(sidebar).toBeVisible()
    await expect(page.getByTestId('sidebar-conformance')).toBeVisible()
    await page.getByRole('tab', { name: /New Request/ }).click()
    await lit('Requests')
    await expect(sidebar).toBeVisible()
  })

  test('a scenario tab clicked after the Workspace tab shows the Scenarios sidebar', async ({ page }) => {
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
    await expect(page.getByTestId('sidebar-workspace')).toBeVisible()

    await page.getByRole('tab', { name: /User flow/ }).click()
    await expect(sidebar).toBeVisible()
    await expect(rail.getByRole('button', { name: 'Scenarios' })).toHaveAttribute('aria-current', 'page')
    await expect(sidebar.getByRole('button', { name: 'Run Log' }).or(sidebar.getByText('Run Log'))).toBeVisible()
  })
})
