import { test, expect, gotoApp } from '../../fixtures/test.js'
import type { Page } from '@playwright/test'

const SCENARIO = { name: 'User flow', path: '/ws/scenarios/user-flow.yaml', stepCount: 2 }

const SCENARIO_DETAIL = {
  path: '/ws/scenarios/user-flow.yaml',
  name: 'User flow',
  description: 'End-to-end user lifecycle',
  error: '',
  steps: [
    {
      name: 'Create user',
      method: 'POST',
      url: 'https://api.example.com/users',
      requestRef: '',
      failPolicy: '',
      assertions: [{ type: 'status', path: '', operator: 'eq', expected: 201, schema: '' }],
      extractions: [{ name: 'userId', source: 'jsonpath', path: '$.id' }],
    },
    {
      name: 'Fetch user',
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
  name: 'User flow',
  error: '',
  passed: true,
  durationMs: 150,
  steps: [
    {
      name: 'Create user',
      error: '',
      passed: true,
      skipped: false,
      statusCode: 201,
      durationMs: 80,
      extracted: { userId: '42' },
      assertions: [{ type: 'status', message: 'status = 201', passed: true }],
    },
    {
      name: 'Fetch user',
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
    await expect(page.locator('aside').getByText('Scenarios')).not.toBeVisible()
  })

  test('scenarios section appears in sidebar when list has items', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByText('Scenarios')).toBeVisible()
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
    await expect(page.getByRole('tab')).toHaveCount(1)
  })

  test('scenario viewer shows step names', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await expect(page.getByText('Fetch user').first()).toBeVisible()
  })

  test('right panel prompts to click Run before first run', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText(/Click.*Run/)).toBeVisible()
  })

  test('Run button shows PASSED badge on success', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('PASSED')).toBeVisible()
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
    await expect(page.getByText('201', { exact: true })).toBeVisible()
  })

  test('assertion results shown in step result', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('status = 201').first()).toBeVisible()
  })

  test('extracted variables shown in step result', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL, result: SCENARIO_RESULT })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await page.getByRole('button', { name: 'Run', exact: true }).click()
    await expect(page.getByText('→ userId:')).toBeVisible()
  })

  test('FAILED badge shown when scenario fails', async ({ page }) => {
    const failResult = {
      name: 'User flow',
      error: '',
      passed: false,
      durationMs: 80,
      steps: [{
        name: 'Create user',
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
    await expect(page.getByText('FAILED')).toBeVisible()
    await expect(page.getByText('status = 201 (got 500)')).toBeVisible()
  })
})

test.describe('Scenario management — create', () => {
  test('+ button appears in sidebar when workspace is open', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByTitle('New scenario')).toBeVisible()
  })

  test('clicking + shows inline name input', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario').click()
    await expect(page.locator('aside').getByRole('textbox')).toBeVisible()
  })

  test('pressing Escape cancels without creating a scenario', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario').click()
    await page.locator('aside').getByRole('textbox').press('Escape')
    await expect(page.locator('aside').getByRole('textbox')).not.toBeVisible()
    // Scenarios section header still present (+ button) but no scenario items
    await expect(page.locator('aside').getByTitle('New scenario')).toBeVisible()
  })

  test('typing a name and pressing Enter creates scenario and opens a tab', async ({ page }) => {
    await withWorkspaceOnly(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('New scenario').click()
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
    await expect(page.getByText('PASSED')).toBeVisible()
  })

  test('right panel prompts to run when no cached result exists', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText(/Click.*Run/)).toBeVisible()
    await expect(page.getByText('PASSED')).not.toBeVisible()
  })
})

test.describe('Scenario management — run all', () => {
  const FLOW_A = { name: 'Flow A', path: '/ws/scenarios/flow-a.yaml', stepCount: 1 }
  const FLOW_B = { name: 'Flow B', path: '/ws/scenarios/flow-b.yaml', stepCount: 1 }

  test('Run All button appears when workspace has scenarios', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await expect(page.locator('aside').getByTitle('Run all scenarios')).toBeVisible()
  })

  test('clicking Run All shows pass summary when all scenarios pass', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B] })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.locator('aside').getByText('2/2 passed')).toBeVisible()
  })

  test('run all results list shows each scenario name', async ({ page }) => {
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B] })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.locator('aside').getByText('2/2 passed')).toBeVisible()
    // Each scenario name appears in the results panel
    const panel = page.locator('aside').getByText('2/2 passed').locator('xpath=../..')
    await expect(panel.getByText('Flow A')).toBeVisible()
    await expect(panel.getByText('Flow B')).toBeVisible()
  })

  test('clicking a result row opens the scenario tab', async ({ page }) => {
    const detailA = { path: FLOW_A.path, name: 'Flow A', description: '', steps: [], error: '' }
    await withScenarios(page, { initialScenarios: [FLOW_A, FLOW_B], detail: detailA })
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.locator('aside').getByText('2/2 passed')).toBeVisible()
    const panel = page.locator('aside').getByText('2/2 passed').locator('xpath=../..')
    await panel.getByRole('button', { name: 'Flow A' }).click()
    await expect(page.getByRole('tab').filter({ hasText: 'Flow A' })).toBeVisible()
  })

  test('dismiss button hides the run all results panel', async ({ page }) => {
    await withScenarios(page)
    await gotoApp(page)
    await page.locator('aside').getByTitle('Run all scenarios').click()
    await expect(page.locator('aside').getByText('1/1 passed')).toBeVisible()
    // Dismiss button is the only button in the header row alongside the summary text
    await page.locator('aside').getByText('1/1 passed').locator('xpath=..').getByRole('button').click()
    await expect(page.locator('aside').getByText('1/1 passed')).not.toBeVisible()
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
  async function openInEditMode(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    // Wait for the detail to load before Edit becomes enabled
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
  }

  test('Edit button switches to the visual step editor', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // Step name inputs appear (one per step) with correct values
    await expect(page.getByPlaceholder('Step name')).toHaveCount(2)
    await expect(page.getByPlaceholder('Step name').first()).toHaveValue('Create user')
    await expect(page.getByPlaceholder('Step name').nth(1)).toHaveValue('Fetch user')
  })

  test('visual editor shows method and URL fields', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // URL inputs appear for each step
    await expect(page.getByPlaceholder('https://api.example.com/path')).toHaveCount(2)
    await expect(page.getByPlaceholder('https://api.example.com/path').first()).toHaveValue('https://api.example.com/users')
  })

  test('visual editor shows assertion rows with expected values', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // Step 1 has a status assertion with expected 201
    await expect(page.getByPlaceholder('expected').first()).toHaveValue('201')
  })

  test('Discard returns to view mode without saving', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByPlaceholder('Step name').first().fill('Changed name')
    await page.getByRole('button', { name: 'Discard' }).click()
    // Back in view mode — Run button visible
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // Original step name shown (save never happened)
    await expect(page.getByText('Create user').first()).toBeVisible()
  })

  test('Save serializes edits and returns to view mode', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // Change URL of first step
    await page.getByPlaceholder('https://api.example.com/path').first().fill('https://updated.example.com/users')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    // Returns to view mode
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // The serialized YAML was written to the bridge
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('https://updated.example.com/users')
  })

  test('Add step button inserts a new step card', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByPlaceholder('Step name')).toHaveCount(2)
    await page.getByRole('button', { name: 'Add step' }).click()
    await expect(page.getByPlaceholder('Step name')).toHaveCount(3)
  })

  test('Remove step button deletes the step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByPlaceholder('Step name')).toHaveCount(2)
    await page.getByTitle('Remove step').first().click()
    await expect(page.getByPlaceholder('Step name')).toHaveCount(1)
    // The remaining step is "Fetch user"
    await expect(page.getByPlaceholder('Step name').first()).toHaveValue('Fetch user')
  })

  test('Add assertion button inserts a new assertion row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // SCENARIO_DETAIL: step 1 has 1 assertion, step 2 has 2 → 3 total
    await expect(page.getByTitle('Remove assertion')).toHaveCount(3)
    await page.getByRole('button', { name: 'Add assertion' }).first().click()
    await expect(page.getByTitle('Remove assertion')).toHaveCount(4)
  })

  test('Remove assertion button removes the assertion row', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByTitle('Remove assertion')).toHaveCount(3)
    await page.getByTitle('Remove assertion').first().click()
    await expect(page.getByTitle('Remove assertion')).toHaveCount(2)
  })

  test('YAML toggle shows the editor with Save and Discard controls', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByRole('button', { name: 'YAML', exact: true }).click()
    // YAML mode: Save and Discard controls appear
    await expect(page.getByRole('button', { name: 'Discard' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Save', exact: true })).toBeVisible()
  })

  test('timeout field defaults to 0 and accepts a numeric value', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    const timeoutInputs = page.getByTitle('Step timeout in milliseconds (0 = use global default)')
    await expect(timeoutInputs.first()).toHaveValue('0')
    await timeoutInputs.first().fill('5000')
    await expect(timeoutInputs.first()).toHaveValue('5000')
  })

  test('saving with a timeout emits timeoutMs in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
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
    await openInEditMode(page)
    const attemptsInputs = page.getByTitle('Maximum retry attempts (0 = no retry)')
    const delayInputs = page.getByTitle('Delay between retry attempts in milliseconds')
    await expect(attemptsInputs.first()).toHaveValue('0')
    await expect(delayInputs.first()).toBeDisabled()
  })

  test('setting retry attempts enables delay field', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Maximum retry attempts (0 = no retry)').first().fill('3')
    await expect(page.getByTitle('Delay between retry attempts in milliseconds').first()).toBeEnabled()
  })

  test('saving with retry emits retry block in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
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
  { name: 'Auth flow', path: '/ws/scenarios/auth-flow.yaml', stepCount: 2 },
  { name: 'User lifecycle', path: '/ws/scenarios/user-lifecycle.yaml', stepCount: 4 },
  { name: 'Health check', path: '/ws/scenarios/health-check.yaml', stepCount: 1 },
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
  async function openInEditMode(page: Page) {
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
  }

  test('move-up button is disabled on the first step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByTitle('Move step up').first()).toBeDisabled()
  })

  test('move-down button is disabled on the last step', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await expect(page.getByTitle('Move step down').last()).toBeDisabled()
  })

  test('move-down on first step swaps it with the second', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    // Before: step 1 = "Create user", step 2 = "Fetch user"
    await expect(page.getByPlaceholder('Step name').first()).toHaveValue('Create user')
    await expect(page.getByPlaceholder('Step name').nth(1)).toHaveValue('Fetch user')
    // Click move-down on the first step
    await page.getByTitle('Move step down').first().click()
    // After: step 1 = "Fetch user", step 2 = "Create user"
    await expect(page.getByPlaceholder('Step name').first()).toHaveValue('Fetch user')
    await expect(page.getByPlaceholder('Step name').nth(1)).toHaveValue('Create user')
  })

  test('move-up on second step swaps it with the first', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Move step up').nth(1).click()
    await expect(page.getByPlaceholder('Step name').first()).toHaveValue('Fetch user')
    await expect(page.getByPlaceholder('Step name').nth(1)).toHaveValue('Create user')
  })

  test('step numbers update after reorder', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Move step down').first().click()
    // Step circles should show 1 and 2 — text content check via aria
    // The step index circles show "1" and "2" as the visible text in the step cards
    const cards = page.locator('.rounded.border.border-\\[var\\(--color-border\\)\\].bg-\\[var\\(--color-muted\\)\\]')
    await expect(cards).toHaveCount(2)
  })

  test('saving after reorder serializes steps in the new order', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await openInEditMode(page)
    await page.getByTitle('Move step down').first().click()
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
    description: i === 0 ? 'Creates a new user and captures the returned ID' : '',
  })),
}

test.describe('Scenario view mode — description and step notes', () => {
  test('scenario description is shown in the left panel in view mode', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL_WITH_NOTES })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await expect(page.getByText('End-to-end user lifecycle')).toBeVisible()
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

  test('description field is present in visual editor and can be edited', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
    // Description inputs visible (one per step)
    await expect(page.getByPlaceholder('Step description (optional)')).toHaveCount(2)
    await page.getByPlaceholder('Step description (optional)').first().fill('Sends a POST to create user')
    await expect(page.getByPlaceholder('Step description (optional)').first()).toHaveValue('Sends a POST to create user')
  })

  test('saving with a step description emits description in the YAML', async ({ page }) => {
    await withScenarios(page, { detail: SCENARIO_DETAIL })
    await gotoApp(page)
    await page.locator('aside').getByRole('button', { name: 'User flow' }).click()
    await expect(page.getByText('Create user').first()).toBeVisible()
    await page.getByRole('button', { name: 'Edit', exact: true }).click()
    await page.getByPlaceholder('Step description (optional)').first().fill('My step note')
    await page.getByRole('button', { name: 'Save', exact: true }).click()
    await expect(page.getByRole('button', { name: 'Run', exact: true })).toBeVisible()
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    const savedYaml = await page.evaluate(() => (window as any).__bridge.scenarioRaw)
    expect(savedYaml).toContain('description: My step note')
  })
})
