import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('CLI Command Builder', () => {
  test('builds, explains and shows a command per shell', async ({ page }) => {
    await gotoApp(page)

    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await expect(page.getByRole('heading', { name: 'CLI Command Builder' })).toBeVisible()
    await expect(page.getByRole('tab', { name: /CLI Builder/ })).toBeVisible()

    const command = page.getByTestId('cli-command')
    await expect(command).toContainText('Fill in the fields')

    await page.getByLabel('Choose Scenario from the workspace').selectOption('/workspace/scenarios/user-lifecycle.yaml')
    await page.getByLabel('Coverage report').check()
    await expect(command).toContainText('clrsig run scenarios/user-lifecycle.yaml')
    await expect(command).toContainText('--coverage')

    await page.getByLabel('Choose Environment from the workspace').selectOption('/workspace/environments/ci.yaml')
    await expect(page.getByRole('region', { name: 'Environment variables' })).toContainText('API_TOKEN')

    await page.getByRole('tab', { name: 'cmd' }).click()
    await expect(command).toContainText('scenarios\\user-lifecycle.yaml')

    await expect(page.getByRole('region', { name: 'Explanation' })).toContainText('The pipeline step fails')
  })

  test('suggests a file name for a non-text report and uses it', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await page.getByLabel('Choose Scenario from the workspace').selectOption('/workspace/scenarios/user-lifecycle.yaml')
    await page.getByLabel('Report format').selectOption('html')

    const command = page.getByTestId('cli-command')
    await expect(command).toContainText('--format html')
    await expect(page.getByText(/printed to the screen/)).toBeVisible()

    await page.getByRole('button', { name: 'Use reports/run-user-lifecycle.html' }).click()
    await expect(page.getByLabel('Save report to')).toHaveValue('reports/run-user-lifecycle.html')
    await expect(command).toContainText('--output reports/run-user-lifecycle.html')
    await expect(page.getByText(/printed to the screen/)).toHaveCount(0)
  })

  test('blocks copying until required inputs are filled', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await page.getByLabel('Coverage report').check()
    await expect(page.getByText('Scenario is required')).toBeVisible()
    await expect(page.getByRole('button', { name: /Copy/ })).toBeDisabled()
  })

  test('is a single page tab', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await page.getByRole('button', { name: 'CLI Command Builder' }).click()
    await expect(page.getByRole('tab', { name: /CLI Builder/ })).toHaveCount(1)
  })
})
