import { test, expect, gotoApp as _gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'

// All scenario tests need the Scenarios activity-rail tab to be active.
// This local wrapper navigates to the app and then opens the Scenarios panel.
async function gotoApp(page: Page) {
  await _gotoApp(page)
  await page.getByTitle('Scenarios').click()
}

const SCENARIO = { title: 'User flow', path: '/ws/scenarios/user-flow.yaml', stepCount: 2 }

const SCENARIO_DETAIL = {
  path: '/ws/scenarios/user-flow.yaml',
  title: 'User flow',
  description: 'End-to-end user lifecycle',
  error: '',
  steps: [
    {
      label: 'Create user',
      method: 'POST',
      url: 'https://api.example.com/users',
      requestRef: '',
      failPolicy: '',
      assertions: [{ type: 'status', path: '', operator: 'eq', expected: 201, schema: '' }],
      extractions: [{ name: 'userId', source: 'jsonpath', path: '$.id' }],
    },
    {
      label: 'Fetch user',
      method: 'GET',
      url: 'https://api.example.com/users/{{userId}}',
      requestRef: '',
      failPolicy: '',
      assertions: [
        { type: 'status', path: '', operator: 'eq', expected: 200, schema: '' },
        { type: 'jsonpath', path: '$.name', operator: 'eq', expected: 'Alice', schema: '' },
      ],
      extractions: [],
    },
  ],
}

const SCENARIO_RESULT = {
  title: 'User flow',
  error: '',
  passed: true,
  durationMs: 150,
  steps: [
    {
      label: 'Create user',
      error: '',
      passed: true,
      skipped: false,
      statusCode: 201,
      durationMs: 80,
      extracted: { userId: '42' },
      assertions: [{ type: 'status', message: 'status = 201', passed: true }],
    },
    {
      label: 'Fetch user',
      error: '',
      passed: true,
      skipped: false,
      statusCode: 200,
      durationMs: 70,
      extracted: {},
      assertions: [
        { type: 'status', message: 'status = 200', passed: true },
        { type: 'jsonpath', message: '$.name = Alice', passed: true },
      ],
    },
  ],
}

async function withScenarios(page: Page, opts: { detail?: object; result?: object; initialScenarios?: object[] } = {}) {
  const scenarios = opts.initialScenarios ?? [SCENARIO]
  const script = `
    (function() {
      var b = window.__bridge;
      b.workspacePath = '/workspace';
      b.scenarios = ${JSON.stringify(scenarios)};
      ${opts.detail ? `b.scenarioDetail = ${JSON.stringify(opts.detail)};` : ''}
      ${opts.result ? `b.scenarioResult = ${JSON.stringify(opts.result)};` : ''}
    })();
  `
  await page.addInitScript(script)
}

async function withWorkspaceOnly(page: Page) {
  await page.addInitScript(`
    (function() {
      var b = window.__bridge;
      b.workspacePath = '/workspace';
      b.scenarios = [];
    })();
  `)
}

test.describe('Scenarios', () => {
  test('no scenarios section shown by default', async ({ page }) => {
    await gotoApp(page)
    // With no scenarios in the workspace, no scenario buttons appear in the list
    await expect(page.locator('aside').getByRole('button', { name: 'User flow' })).not.toBeVisible()
  })

  test('scenarios section appears in sidebar when list has items', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByText('Scenarios', { exact: true })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'User flow' })).toBeVisible()
  })

  test('clicking a scenario opens a SCN tab', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    const tab = page.getByRole('tab').filter({ hasText: 'User flow' })
    await expect(tab).toBeVisible()
    await expect(tab).toContainText('SCN')
    await expect(tab).toHaveAttribute('aria-selected', 'true')
  })

  test('scenario tab is never dirty', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    const tab = page.getByRole('tab').filter({ hasText: 'User flow' })
    await expect(tab.getByTitle('Unsaved changes')).not.toBeVisible()
  })

  test('clicking same scenario twice does not open a second tab', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    const btn = page.locator('aside').getByRole('button', { name: 'User flow' })
    await btn.click()
    await btn.click()
    await expect(page.getByRole('tab').filter({ hasNotText: /^(Definition|Result)/ })).toHaveCount(1)
  })

  test('scenario viewer shows step names', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await expect(page.getByText('Fetch user').first()).toBeVisible()
  })

  test('step editor is shown before first run', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  })

  test('Run button shows PASSED badge on success', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('PASSED').first()).toBeVisible()
  })

  test('shows total duration in header after run', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText(/150 ms/)).toBeVisible()
  })

  test('step result shows HTTP status code after run', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('201', { exact: true }).first()).toBeVisible()
  })

  test('assertion results shown in step result strip after expanding', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    // Expand the result strip for step 1 (collapsed by default for passing steps)
    await page.getByRole('region', { name: 'Step details' }).getByRole('button').first().click()
    await expect(page.getByText('status = 201').first()).toBeVisible()
  })

  test('extracted variables shown in step result strip after expanding', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    // Expand the result strip for step 1 (collapsed by default for passing steps)
    await page.getByRole('region', { name: 'Step details' }).getByRole('button').first().click()
    await expect(page.getByText('→ userId:').first()).toBeVisible()
  })

  test('FAILED badge shown when scenario fails', async ({ page }) => {
    const failResult = {
      title: 'User flow',
      error: '',
      passed: false,
      durationMs: 80,
      steps: [{
        label: 'Create user',
        error: '',
        passed: false,
        skipped: false,
        statusCode: 500,
        durationMs: 80,
        extracted: {},
        assertions: [{ type: 'status', message: 'status = 201 (got 500)', passed: false }],
      }],
    }
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: failResult })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('FAILED').first()).toBeVisible()
    await expect(page.getByText('status = 201 (got 500)')).toBeVisible()
  })
})

test.describe('Scenario management — create', () => {
  test('+ button appears in sidebar when workspace is open', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByTitle('New scenario', { exact: true })).toBeVisible()
  })

  test('clicking + shows inline name input', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario', { exact: true }).click()
    await expect(page.locator('aside').getByRole('textbox')).toBeVisible()
  })

  test('pressing Escape cancels without creating a scenario', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario', { exact: true }).click()
    await page.locator('aside').getByRole('textbox').press('Escape')
    await expect(page.locator('aside').getByRole('textbox')).not.toBeVisible()
    // Scenarios section header still present (+ button) but no scenario items
    await expect(page.locator('aside').getByTitle('New scenario', { exact: true })).toBeVisible()
  })

  test('typing a name and pressing Enter creates scenario and opens a tab', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario', { exact: true }).click()
    await page.locator('aside').getByRole('textbox').fill('My Test Scenario')
    await page.locator('aside').getByRole('textbox').press('Enter')
    // New item appears in sidebar
    await expect(page.locator('aside').getByRole('button', { name: 'My Test Scenario' })).toBeVisible()
    // A tab opens for the new scenario
    const tab = page.getByRole('tab').filter({ hasText: 'My Test Scenario' })
    await expect(tab).toBeVisible()
    await expect(tab).toContainText('SCN')
  })
})

test.describe('Scenario management — delete', () => {
  test('hovering a scenario row reveals the delete button', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await expect(page.locator('aside').getByTitle('Delete scenario')).toBeVisible()
  })

  test('clicking delete removes the scenario from the sidebar', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await page.locator('aside').getByTitle('Delete scenario').click()
    // Confirm in the delete dialog
    await page.locator('.fixed.inset-0').getByRole('button', { name: 'Delete' }).click()
    await expect(page.locator('aside').getByRole('button', { name: 'User flow' })).not.toBeVisible()
  })

  test('deleting an open scenario tab closes that tab', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    // Open the scenario tab first
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByRole('tab').filter({ hasText: 'User flow' })).toBeVisible()
    // Delete it
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await page.locator('aside').getByTitle('Delete scenario').click()
    // Confirm in the delete dialog
    await page.locator('.fixed.inset-0').getByRole('button', { name: 'Delete' }).click()
    await expect(page.getByRole('tab').filter({ hasText: 'User flow' })).not.toBeVisible()
  })
})

test.describe('Scenario management — rename', () => {
  test('double-clicking a scenario name enters rename mode', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).dblclick()
    // The rename input is the last textbox in aside (filter input is the first)
    await expect(page.locator('aside').getByRole('textbox').last()).toBeVisible()
  })

  test('pressing Escape cancels rename without changing name', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).dblclick()
    await page.locator('aside').getByRole('textbox').last().press('Escape')
    await expect(page.locator('aside').getByRole('button', { name: 'User flow' })).toBeVisible()
    // Only the filter input remains (one textbox)
    await expect(page.locator('aside').getByRole('textbox')).toHaveCount(1)
  })

  test('typing a new name and pressing Enter renames the item', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).dblclick()
    const input = page.locator('aside').getByRole('textbox').last()
    await input.fill('Renamed Flow')
    await input.press('Enter')
    await expect(page.locator('aside').getByRole('button', { name: 'Renamed Flow' })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'User flow' })).not.toBeVisible()
  })

  test('renaming updates an open tab title', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    // Open the tab
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByRole('tab').filter({ hasText: 'User flow' })).toBeVisible()
    // Rename in sidebar
    await page.locator('aside').getByRole('button', { name: 'User flow' }).dblclick()
    const input = page.locator('aside').getByRole('textbox').last()
    await input.fill('Renamed Flow')
    await input.press('Enter')
    // Tab title updates
    await expect(page.getByRole('tab').filter({ hasText: 'Renamed Flow' })).toBeVisible()
    await expect(page.getByRole('tab').filter({ hasText: 'User flow' })).not.toBeVisible()
  })
})

test.describe('Scenario step count badge', () => {
  test('step count is shown next to scenario name in sidebar', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [{ ...SCENARIO, stepCount: 4 }] })
    await gotoApp(page)
    await expect(
      page.locator('aside').getByRole('button', { name: 'User flow' })
    ).toContainText('4')
  })

  test('step count badge is hidden when stepCount is 0', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [{ ...SCENARIO, stepCount: 0 }] })
    await gotoApp(page)
    const btn = page.locator('aside').getByRole('button', { name: 'User flow' })
    await expect(btn).toBeVisible()
    // Name is there but no "0" appended
    await expect(btn).not.toContainText('0')
  })
})

test.describe('Scenario management — duplicate', () => {
  test('hovering a scenario row reveals the duplicate button', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await expect(page.locator('aside').getByTitle('Duplicate scenario')).toBeVisible()
  })

  test('clicking duplicate adds a copy to the sidebar', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await page.locator('aside').getByTitle('Duplicate scenario').click()
    await expect(page.locator('aside').getByRole('button', { name: 'User flow Copy' })).toBeVisible()
  })

  test('duplicate opens a new SCN tab', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).hover()
    await page.locator('aside').getByTitle('Duplicate scenario').click()
    const tab = page.getByRole('tab').filter({ hasText: 'user-flow-copy' })
    await expect(tab).toBeVisible()
    await expect(tab).toContainText('SCN')
  })
})

test.describe('Scenario result persistence', () => {
  test('cached result is shown automatically when opening a scenario tab', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await page.addInitScript(`
      (function() {
        window.__bridge.scenarioResultCache['/ws/scenarios/user-flow.yaml'] = ${JSON.stringify(SCENARIO_RESULT)};
      })();
    `)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('PASSED').first()).toBeVisible()
  })

  test('step editor shown with no result badge when no cached result exists', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
    await expect(page.getByText('PASSED')).not.toBeVisible()
  })
})

test.describe('Scenario management — run all', () => {
  const FLOW_A = { title: 'Flow A', path: '/ws/scenarios/flow-a.yaml', stepCount: 1 }
  const FLOW_B = { title: 'Flow B', path: '/ws/scenarios/flow-b.yaml', stepCount: 1 }

  test('Run All button appears when workspace has scenarios', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByTitle('Run all scenarios')).toBeVisible()
  })

  // Run All results are shown in the single "Run Results" (Runner) tab, not in the sidebar.
  test('clicking Run All opens the Run Results tab with a pass summary', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B] })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.getByRole('tab', { name: /Run Results/ })).toBeVisible()
    await expect(page.getByText('All passed')).toBeVisible()
    await expect(page.getByText(/2\/2 scenarios/)).toBeVisible()
  })

  test('run all results list shows each scenario name', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B] })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.getByText('All passed')).toBeVisible()
    const table = page.getByRole('table')
    await expect(table.getByText('Flow A')).toBeVisible()
    await expect(table.getByText('Flow B')).toBeVisible()
  })

  // The scenario list also carries a placeholder per folder; those are not scenarios and must not run.
  test('Run All runs only scenarios, not the folders that hold them', async ({ page }) => {
    const folder = { isFolder: true, folder: 'group', path: '/ws/scenarios/group', title: '' }
    await withScenarios(page, { initialScenarios: [folder, { ...FLOW_A, folder: 'group', path: '/ws/scenarios/group/flow-a.yaml' }, FLOW_B] })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios', { exact: true }).click()
    await expect(page.getByText('All passed')).toBeVisible()
    await expect(page.getByText(/2\/2 scenarios/)).toBeVisible()
    await expect(page.getByRole('table').getByRole('row')).toHaveCount(3) // header + 2 scenarios
  })

  test('clicking a result row opens the scenario tab', async ({ page }) => {
    const detailA = { path: FLOW_A.path, title: 'Flow A', description: '', steps: [], error: '' }
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B], detail: detailA })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.getByText('All passed')).toBeVisible()
    await page.getByRole('table').getByText('Flow A').click()
    await expect(page.getByRole('tab').filter({ hasText: 'Flow A' })).toBeVisible()
  })

  test('Clear empties the Run Results tab', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.getByText(/1\/1 scenarios/)).toBeVisible()
    await page.getByRole('button', { name: 'Clear' }).click()
    await expect(page.getByText(/1\/1 scenarios/)).not.toBeVisible()
    await expect(page.getByText('No run in progress.')).toBeVisible()
  })
})

test.describe('Scenario env picker', () => {
  async function withEnvironments(page: Page) {
    await page.addInitScript(`
      (function() {
        window.__bridge.environments = [
          { name: 'Staging', path: '/ws/environments/staging.yaml' },
          { name: 'Production', path: '/ws/environments/production.yaml' },
        ];
      })();
    `)
  }

  test('env picker is visible in ScenarioViewer when environments exist', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await withEnvironments(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByTitle('Environment for this run')).toBeVisible()
  })

  test('env picker is not shown when no environments are configured', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByTitle('Environment for this run')).not.toBeVisible()
  })

  test('env picker lists all environments with a no-environment option', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await withEnvironments(page)
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    const picker = page.getByTitle('Environment for this run')
    await expect(picker.getByRole('option', { name: 'No environment' })).toBeAttached()
    await expect(picker.getByRole('option', { name: 'Staging' })).toBeAttached()
    await expect(picker.getByRole('option', { name: 'Production' })).toBeAttached()
  })
})

test.describe('Scenario visual editor', () => {
  async function openScenario(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    // Wait for the step editor to appear (editor is always active — no Edit button needed)
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  }

  test('step editor shows first step by default', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // First step selected by default; editor shows its fields
    await expect(page.getByPlaceholder('Step label').first()).toHaveValue('Create user')
  })

  test('clicking a step in the list switches the editor to that step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByRole('option', { name: /Fetch user/ }).click()
    await expect(page.getByPlaceholder('Step label').first()).toHaveValue('Fetch user')
  })

  test('visual editor shows method and URL fields for selected step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await expect(page.getByPlaceholder('https://api.example.com/path').first()).toHaveValue('https://api.example.com/users')
  })

  test('visual editor shows assertion rows with expected values', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Step 1 has a status assertion with expected 201
    await expect(page.getByPlaceholder('expected').first()).toHaveValue('201')
  })

  test('Discard resets unsaved changes', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByPlaceholder('Step label').first().fill('Changed name')
    await page.getByRole('button', { name: 'Discard' }).click()
    // Confirm in the discard dialog
    await page.locator('.fixed.inset-0').getByRole('button', { name: 'Discard' }).click()
    // Run button always visible; original step name restored
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    await expect(page.getByPlaceholder('Step label').first()).toHaveValue('Create user')
  })

  test('Save serializes edits to YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByPlaceholder('https://api.example.com/path').first().fill('https://updated.example.com/users')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('https://updated.example.com/users')
  })

  test('Add step button inserts a new step in the list', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    const stepList = page.getByRole('listbox', { name: 'Scenario steps' })
    await expect(stepList.getByRole('option')).toHaveCount(2)
    await page.getByRole('button', { name: 'Add step' }).click()
    await expect(stepList.getByRole('option')).toHaveCount(3)
  })

  test('Remove step button deletes the step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    const stepList = page.getByRole('listbox', { name: 'Scenario steps' })
    await expect(stepList.getByRole('option')).toHaveCount(2)
    // Remove button in the step editor header removes the selected (first) step
    await page.getByTitle('Remove step').first().click()
    await expect(stepList.getByRole('option')).toHaveCount(1)
    await expect(page.getByPlaceholder('Step label').first()).toHaveValue('Fetch user')
  })

  test('Add assertion button inserts a new assertion row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Step 1 (selected) has 1 assertion visible in the editor
    await expect(page.getByTitle('Remove assertion')).toHaveCount(1)
    await page.getByRole('button', { name: 'Add assertion' }).first().click()
    await expect(page.getByTitle('Remove assertion')).toHaveCount(2)
  })

  test('Remove assertion button removes the assertion row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await expect(page.getByTitle('Remove assertion')).toHaveCount(1)
    await page.getByTitle('Remove assertion').first().click()
    await expect(page.getByTitle('Remove assertion')).toHaveCount(0)
  })

  test('YAML button opens a dialog with YAML content', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByTitle('View / edit YAML').click()
    // YAML dialog shows with a heading and Check/Save controls
    await expect(page.locator('.fixed.inset-0').getByText('YAML').first()).toBeVisible()
    await expect(page.locator('.fixed.inset-0').getByRole('button', { name: 'Check' })).toBeVisible()
  })

  test('timeout field defaults to 0 and accepts a numeric value', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Timeout field is inside the Advanced section — expand it first
    await page.getByRole('button', { name: 'Advanced' }).click()
    const timeoutInputs = page.getByTitle('Step timeout in milliseconds (0 = use global default)')
    await expect(timeoutInputs.first()).toHaveValue('0')
    await timeoutInputs.first().fill('5000')
    await expect(timeoutInputs.first()).toHaveValue('5000')
  })

  test('saving with a timeout emits timeoutMs in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByRole('button', { name: 'Advanced' }).click()
    await page.getByTitle('Step timeout in milliseconds (0 = use global default)').first().fill('3000')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('timeoutMs: 3000')
  })

  test('retry field defaults to 0 and delay input is disabled when attempts is 0', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByRole('button', { name: 'Advanced' }).click()
    const attemptsInputs = page.getByTitle('Maximum retry attempts (0 = no retry)')
    // When attempts is 0 the delay input uses a different title (it's also disabled)
    const delayInputs = page.getByTitle('Set Retry attempts > 0 to enable delay')
    await expect(attemptsInputs.first()).toHaveValue('0')
    await expect(delayInputs.first()).toBeDisabled()
  })

  test('setting retry attempts enables delay field', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByRole('button', { name: 'Advanced' }).click()
    await page.getByTitle('Maximum retry attempts (0 = no retry)').first().fill('3')
    await expect(page.getByTitle('Delay between retry attempts in milliseconds').first()).toBeEnabled()
  })

  test('saving with retry emits retry block in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByRole('button', { name: 'Advanced' }).click()
    await page.getByTitle('Maximum retry attempts (0 = no retry)').first().fill('3')
    await page.getByTitle('Delay between retry attempts in milliseconds').first().fill('500')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('retry:')
    expect(savedYaml).toContain('maxAttempts: 3')
    expect(savedYaml).toContain('delayMs: 500')
  })
})

const MULTI_SCENARIOS = [
  { title: 'Auth flow', path: '/ws/scenarios/auth-flow.yaml', stepCount: 2 },
  { title: 'User lifecycle', path: '/ws/scenarios/user-lifecycle.yaml', stepCount: 4 },
  { title: 'Health check', path: '/ws/scenarios/health-check.yaml', stepCount: 1 },
]

test.describe('Scenario search/filter', () => {
  test('filter input is visible in the Scenarios section', async ({ page }) => {
    await withScenarios(page, { initialScenarios: MULTI_SCENARIOS })
    await gotoApp(page)
    await expect(page.locator('aside').getByPlaceholder('Filter scenarios…')).toBeVisible()
  })

  test('typing a query hides non-matching scenarios', async ({ page }) => {
    await withScenarios(page, { initialScenarios: MULTI_SCENARIOS })
    await gotoApp(page)
    await page.locator('aside').getByPlaceholder('Filter scenarios…').fill('health')
    await expect(page.locator('aside').getByRole('button', { name: 'Health check' })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'Auth flow' })).not.toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'User lifecycle' })).not.toBeVisible()
  })

  test('filter is case-insensitive', async ({ page }) => {
    await withScenarios(page, { initialScenarios: MULTI_SCENARIOS })
    await gotoApp(page)
    await page.locator('aside').getByPlaceholder('Filter scenarios…').fill('AUTH')
    await expect(page.locator('aside').getByRole('button', { name: 'Auth flow' })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'Health check' })).not.toBeVisible()
  })

  test('clearing the filter restores all scenarios', async ({ page }) => {
    await withScenarios(page, { initialScenarios: MULTI_SCENARIOS })
    await gotoApp(page)
    await page.locator('aside').getByPlaceholder('Filter scenarios…').fill('health')
    await expect(page.locator('aside').getByRole('button', { name: 'Auth flow' })).not.toBeVisible()
    // Click the × clear button
    await page.locator('aside').getByPlaceholder('Filter scenarios…').fill('')
    await expect(page.locator('aside').getByRole('button', { name: 'Auth flow' })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'User lifecycle' })).toBeVisible()
    await expect(page.locator('aside').getByRole('button', { name: 'Health check' })).toBeVisible()
  })

  test('a query with no matches shows the no-matches message', async ({ page }) => {
    await withScenarios(page, { initialScenarios: MULTI_SCENARIOS })
    await gotoApp(page)
    await page.locator('aside').getByPlaceholder('Filter scenarios…').fill('zzz')
    await expect(page.locator('aside').getByText('No matches.')).toBeVisible()
  })
})

test.describe('Scenario visual editor — step reordering', () => {
  async function openScenario(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  }

  test('move-up button is disabled on the first step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Step 0 (Create user) is selected by default; its editor's move-up is disabled
    await expect(page.getByTitle('Move step up')).toBeDisabled()
  })

  test('move-down button is disabled on the last step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Select the last step (Fetch user) to see its move-down button disabled
    await page.getByRole('option', { name: /Fetch user/ }).click()
    await expect(page.getByTitle('Move step down')).toBeDisabled()
  })

  test('move-down on first step swaps it with the second', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    const stepList = page.getByRole('listbox', { name: 'Scenario steps' })
    // Before: list order is Create user, Fetch user
    await expect(stepList.getByRole('option').first()).toContainText('Create user')
    // Click move-down on the selected step (Create user, index 0)
    await page.getByTitle('Move step down').click()
    // After: Create user moved to index 1; Fetch user is now first
    await expect(stepList.getByRole('option').first()).toContainText('Fetch user')
    await expect(stepList.getByRole('option').nth(1)).toContainText('Create user')
  })

  test('move-up on second step swaps it with the first', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    const stepList = page.getByRole('listbox', { name: 'Scenario steps' })
    // Select step 1 (Fetch user) and move it up
    await page.getByRole('option', { name: /Fetch user/ }).click()
    await page.getByTitle('Move step up').click()
    // Fetch user is now at index 0
    await expect(stepList.getByRole('option').first()).toContainText('Fetch user')
    await expect(stepList.getByRole('option').nth(1)).toContainText('Create user')
  })

  test('step numbers update after reorder', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByTitle('Move step down').click()
    // Both steps still appear in the step list after reorder
    await expect(page.getByRole('listbox', { name: 'Scenario steps' }).getByRole('option')).toHaveCount(2)
  })

  test('saving after reorder serializes steps in the new order', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByTitle('Move step down').click()
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    // Fetch user should now appear before Create user in the YAML
    const fetchIdx = savedYaml.indexOf('Fetch user')
    const createIdx = savedYaml.indexOf('Create user')
    expect(fetchIdx).toBeLessThan(createIdx)
  })
})

const SCENARIO_DETAIL_WITH_NOTES = {
  ...SCENARIO_DETAIL,
  description: 'End-to-end user lifecycle',
  steps: SCENARIO_DETAIL.steps.map((s, i) => ({
    ...s,
    notes: i === 0 ? 'Creates a new user and captures the returned ID' : undefined,
  })),
}

test.describe('Scenario view mode — description and step notes', () => {
  test('scenario description is shown in Setup', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL_WITH_NOTES })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    // The description lives in the Setup row of the step list
    await page.getByRole('button', { name: 'Setup' }).click()
    await expect(page.getByPlaceholder('What does this scenario test?')).toHaveValue('End-to-end user lifecycle')
  })

  test('step description is shown in italic below the step name', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL_WITH_NOTES })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Creates a new user and captures the returned ID')).toBeVisible()
  })

  test('step with no description shows no note text', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL_WITH_NOTES })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    // "Fetch user" step has no description — the step name appears but no note below it
    await expect(page.getByText('Fetch user')).toBeVisible()
  })

  test('notes field is present in step editor and can be edited', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    // Only the selected step's editor is shown (step 0 by default)
    await expect(page.getByPlaceholder('Notes (optional)')).toHaveCount(1)
    await page.getByPlaceholder('Notes (optional)').first().fill('Sends a POST to create user')
    await expect(page.getByPlaceholder('Notes (optional)').first()).toHaveValue('Sends a POST to create user')
  })

  test('saving with a step notes emits notes in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByPlaceholder('Notes (optional)').first().fill('My step note')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('notes: My step note')
  })
})

test.describe('Scenario visual editor — request body', () => {
  async function openScenario(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  }
  // Keep openInEditMode as alias so tests below don't need individual edits
  const openInEditMode = openScenario

  test('body type selector defaults to none', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    const bodySelects = page.getByTitle('Request body type')
    await expect(bodySelects.first()).toHaveValue('none')
  })

  test('selecting json body type shows the content textarea', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('json')
    await expect(page.getByPlaceholder('{"key": "value"}')).toBeVisible()
  })

  test('selecting form-urlencoded shows form placeholder', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('form-urlencoded')
    await expect(page.getByPlaceholder('key=value&other=value')).toBeVisible()
  })

  test('selecting none hides the textarea', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('json')
    await expect(page.getByPlaceholder('{"key": "value"}')).toBeVisible()
    await page.getByTitle('Request body type').first().selectOption('none')
    await expect(page.getByPlaceholder('{"key": "value"}')).not.toBeVisible()
  })

  test('Format button is enabled only for json type', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('raw')
    await expect(page.getByTitle('Format JSON').first()).toBeDisabled()
    await page.getByTitle('Request body type').first().selectOption('json')
    await expect(page.getByTitle('Format JSON').first()).toBeEnabled()
  })

  test('Format button pretty-prints valid JSON content', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('json')
    const textarea = page.getByPlaceholder('{"key": "value"}').first()
    await textarea.fill('{"name":"Alice","age":30}')
    await page.getByTitle('Format JSON').first().click()
    await expect(textarea).toHaveValue('{\n  "name": "Alice",\n  "age": 30\n}')
  })

  test('saving with a json body emits body block in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Request body type').first().selectOption('json')
    await page.getByPlaceholder('{"key": "value"}').first().fill('{"name":"Alice"}')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('body:')
    expect(savedYaml).toContain('type: json')
    // body content is quoted by the YAML serializer; check the key data is present
    expect(savedYaml).toContain('Alice')
  })

  test('body type shown in step editor when body is non-none', async ({ page }) => {
    const detailWithBody = {
      ...SCENARIO_DETAIL,
      steps: SCENARIO_DETAIL.steps.map((s, i) => i === 0 ? { ...s, bodyType: 'json', bodyContent: '{}' } : s),
    }
    await withScenarios(page, { detail: detailWithBody })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    // In always-edit mode the body type select reflects the loaded bodyType
    await expect(page.getByTitle('Request body type').first()).toHaveValue('json')
  })
})

test.describe('Scenario visual editor — headers and query params', () => {
  async function openScenario(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  }
  const openInEditMode = openScenario

  test('Headers section is present in visual editor', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByRole('button', { name: 'Add header' }).first()).toBeVisible()
  })

  test('Add header button inserts a new header row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByPlaceholder('Header', { exact: true })).toHaveCount(0)
    await page.getByRole('button', { name: 'Add header' }).first().click()
    await expect(page.getByPlaceholder('Header', { exact: true }).first()).toBeVisible()
  })

  test('Query Params section is present in visual editor', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByRole('button', { name: 'Add param' }).first()).toBeVisible()
  })

  test('Add param button inserts a new query param row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByRole('button', { name: 'Add param' }).first().click()
    await expect(page.getByPlaceholder('param').first()).toBeVisible()
  })

  test('saving with a header emits headers block in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByRole('button', { name: 'Add header' }).first().click()
    await page.getByPlaceholder('Header', { exact: true }).first().fill('Authorization')
    await page.getByPlaceholder('value').first().fill('Bearer {{token}}')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('headers:')
    expect(savedYaml).toContain('Authorization')
    expect(savedYaml).toContain('token')
  })

  test('saving with a query param emits queryParams block in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByRole('button', { name: 'Add param' }).first().click()
    await page.getByPlaceholder('param').first().fill('page')
    // the value input follows the param input in the same row
    await page.getByPlaceholder('value').first().fill('1')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('queryParams:')
    expect(savedYaml).toContain('page')
  })

  test('header value is visible in the step editor', async ({ page }) => {
    const detailWithHeaders = {
      ...SCENARIO_DETAIL,
      steps: SCENARIO_DETAIL.steps.map((s, i) =>
        i === 0 ? { ...s, headers: [{ name: 'Content-Type', value: 'application/json' }] } : s
      ),
    }
    await withScenarios(page, { detail: detailWithHeaders })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    // Step 0 (Create user) is selected by default; its header appears in the editor
    await expect(page.getByPlaceholder('Header', { exact: true }).first()).toHaveValue('Content-Type')
  })

  test('query param value is visible in the step editor', async ({ page }) => {
    const detailWithParams = {
      ...SCENARIO_DETAIL,
      steps: SCENARIO_DETAIL.steps.map((s, i) =>
        i === 1 ? { ...s, queryParams: [{ name: 'include', value: 'roles' }] } : s
      ),
    }
    await withScenarios(page, { detail: detailWithParams })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Fetch user').first()).toBeVisible()
    // Select step 1 (Fetch user) which has the query param
    await page.getByRole('option', { name: /Fetch user/ }).click()
    await expect(page.getByPlaceholder('param').first()).toHaveValue('include')
  })
})

test.describe('Scenario visual editor — assertions', () => {
  async function openScenario(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByPlaceholder('Step label').first()).toBeVisible()
  }

  test('selecting schema type shows schema input instead of operator and expected', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Step 0 (Create user) is selected; it has 1 status assertion → 1 operator select
    await expect(page.getByLabel('Assertion operator')).toHaveCount(1)
    await page.getByLabel('Assertion type').first().selectOption('schema')
    // Schema type hides operator + expected for that row → 0 operator selects remain
    await expect(page.getByLabel('Assertion operator')).toHaveCount(0)
    await expect(page.getByLabel('Schema value').first()).toBeVisible()
  })

  test('schema input accepts a text value', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByLabel('Assertion type').first().selectOption('schema')
    await page.getByLabel('Schema value').first().fill('getUserById')
    await expect(page.getByLabel('Schema value').first()).toHaveValue('getUserById')
  })

  test('saving with a schema assertion emits schema field in YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    await page.getByLabel('Assertion type').first().selectOption('schema')
    await page.getByLabel('Schema value').first().fill('listUsers')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('type: schema')
    expect(savedYaml).toContain('schema: listUsers')
  })

  test('status type operator select has only 4 options (eq/ne/gt/lt)', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Step 0 (Create user) has a status assertion — check its operator select
    const optionCount = await page.getByLabel('Assertion operator').first().evaluate(
      (el: HTMLSelectElement) => el.options.length
    )
    expect(optionCount).toBe(4)
  })

  test('jsonpath type operator select has all 6 options', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Select step 1 (Fetch user) which has a jsonpath assertion as its 2nd assertion
    await page.getByRole('option', { name: /Fetch user/ }).click()
    const optionCount = await page.getByLabel('Assertion operator').nth(1).evaluate(
      (el: HTMLSelectElement) => el.options.length
    )
    expect(optionCount).toBe(6)
  })

  test('switching assertion type resets path and expected', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openScenario(page)
    // Select step 1 (Fetch user); assertion 1 is jsonpath with expected=Alice
    await page.getByRole('option', { name: /Fetch user/ }).click()
    await expect(page.getByPlaceholder('expected').nth(1)).toHaveValue('Alice')
    // Switch to status type — expected should reset to empty
    await page.getByLabel('Assertion type').nth(1).selectOption('status')
    await expect(page.getByPlaceholder('expected').nth(1)).toHaveValue('')
  })

  test('schema assertion shown with schema value in step editor', async ({ page }) => {
    const detailWithSchema = {
      ...SCENARIO_DETAIL,
      steps: SCENARIO_DETAIL.steps.map((s, i) =>
        i === 0 ? { ...s, assertions: [{ type: 'schema', schema: 'createUser', operator: 'eq', expected: null }] } : s
      ),
    }
    await withScenarios(page, { detail: detailWithSchema })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    // Step 0 (Create user) is selected; its schema assertion value is in the schema input
    await expect(page.getByLabel('Schema value').first()).toHaveValue('createUser')
  })
  test('scenario variables are listed and offer dynamic functions', async ({ page }) => {
    const detail = { ...SCENARIO_DETAIL, variables: [{ name: 'email', value: 'user-{{$randomString 6}}@example.com' }] }
    await withScenarios(page, { detail })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Setup' }).click()
    await expect(page.getByLabel('Variable 1 name')).toHaveValue('email')

    await page.getByRole('button', { name: 'Add variable' }).click()
    await page.getByLabel('Variable 2 name').fill('count')
    const value = page.getByPlaceholder(/value, e\.g\./).nth(1)
    await value.fill('{{$ran')
    await expect(page.getByRole('button', { name: /\$randomInt/ })).toContainText('Random whole number')
    await page.getByRole('button', { name: /\$randomInt/ }).click()
    await expect(value).toHaveValue('{{$randomInt 1 100}}')
  })

  test('run result shows the spec conformance summary and the violations', async ({ page }) => {
    const detail = { ...SCENARIO_DETAIL, specFile: 'specs/openapi.yaml' }
    const result = {
      ...SCENARIO_RESULT,
      passed: false,
      steps: [
        { ...SCENARIO_RESULT.steps[0], specState: 'matched' },
        { ...SCENARIO_RESULT.steps[1], passed: false, specState: 'violations', specViolations: ['Error at "/name": value must be a string'] },
      ],
    }
    await withScenarios(page, { detail, result })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: /^Run$/ }).click()
    await page.getByRole('button', { name: 'Setup' }).click()
    await page.getByRole('tab', { name: 'result' }).click()

    const section = page.getByRole('region', { name: 'Spec conformance' })
    await expect(section.getByTestId('spec-conformance-summary')).toHaveText('1 of 2 match openapi.yaml, 1 step has violations')
    await expect(section).toContainText('Fetch user')
    await expect(section).toContainText('Error at "/name": value must be a string')
  })

  test('a scenario without a spec shows no conformance section', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: /^Run$/ }).click()
    await page.getByRole('button', { name: 'Setup' }).click()
    await page.getByRole('tab', { name: 'result' }).click()
    await expect(page.getByRole('region', { name: 'Spec conformance' })).toHaveCount(0)
  })

  test('run result shows the seed and generated variables', async ({ page }) => {
    const result = { ...SCENARIO_RESULT, seed: 4242, variables: [{ name: 'email', value: 'u-abc@example.com' }] }
    await withScenarios(page, { detail: SCENARIO_DETAIL, result })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: /^Run$/ }).click()
    await page.getByRole('button', { name: 'Setup' }).click()
    await page.getByRole('tab', { name: 'result' }).click()
    await expect(page.getByText('clrsig run --seed 4242')).toBeVisible()
    await expect(page.getByText('u-abc@example.com')).toBeVisible()
  })
  test('run shows the request as sent and the response for the selected step', async ({ page }) => {
    const result = {
      ...SCENARIO_RESULT,
      steps: SCENARIO_RESULT.steps.map((st, i) => i === 0 ? {
        ...st,
        requestMethod: 'POST', requestUrl: 'https://api.example.com/users?v=1',
        requestHeaders: [{ name: 'Authorization', value: 'Bearer ***' }], requestBody: '{"name":"Ada"}',
        responseHeaders: [{ name: 'Content-Type', value: 'application/json' }], responseBody: '{"id":42}',
      } : st),
    }
    await withScenarios(page, { detail: SCENARIO_DETAIL, result })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await page.getByRole('tab', { name: /^Result/ }).click()

    const req = page.getByRole('region', { name: 'Request', exact: true })
    const res = page.getByRole('region', { name: 'Response', exact: true })
    await expect(req).toContainText('https://api.example.com/users?v=1')
    await req.getByRole('button', { name: /Headers \(1\)/ }).click()
    await expect(req).toContainText('Bearer ***')
    await expect(req).toContainText('"name": "Ada"')
    await expect(res).toContainText('"id": 42')
    await res.getByRole('button', { name: /Headers \(1\)/ }).click()
    await expect(res).toContainText('application/json')
  })
})
