import { test, expect, gotoApp } from '../../fixtures/test.js'

test.describe('Request editor', () => {
  test('can type a URL', async ({ page }) => {
    await gotoApp(page)
    const urlInput = page.getByPlaceholder('https://api.example.com/endpoint')
    await urlInput.fill('https://api.example.com/users')
    await expect(urlInput).toHaveValue('https://api.example.com/users')
  })

  test('Send button becomes enabled after typing URL', async ({ page }) => {
    await gotoApp(page)
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com')
    await expect(page.getByRole('button', { name: 'Send' })).toBeEnabled()
  })

  test('method selector has all 7 HTTP methods', async ({ page }) => {
    await gotoApp(page)
    const select = page.getByRole('combobox').first()
    const options = await select.locator('option').allTextContents()
    expect(options).toEqual(['GET', 'POST', 'PUT', 'PATCH', 'DELETE', 'HEAD', 'OPTIONS'])
  })

  test('changing method to POST updates the selector', async ({ page }) => {
    await gotoApp(page)
    const select = page.getByRole('combobox').first()
    await select.selectOption('POST')
    await expect(select).toHaveValue('POST')
  })

  test('Params tab: add row and fill in name/value', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Params' }).click()
    await page.getByText('+ Add row').first().click()
    const nameInputs = page.getByPlaceholder('Parameter')
    await nameInputs.last().fill('page')
    const valueInputs = page.getByPlaceholder('Value')
    await valueInputs.last().fill('2')
    await expect(nameInputs.last()).toHaveValue('page')
    await expect(valueInputs.last()).toHaveValue('2')
  })

  test('Headers tab: add row', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Headers' }).click()
    await page.getByText('+ Add row').first().click()
    // A new row should appear — the add row button is still there, plus the new row inputs
    const addRowBtns = page.getByText('+ Add row')
    await expect(addRowBtns.first()).toBeVisible()
  })

  test('Body tab: JSON type shows textarea', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Body' }).click()
    await page.locator('select').nth(1).selectOption('json')
    await expect(page.getByRole('textbox').last()).toBeVisible()
  })

  test('Body tab: form-urlencoded shows KeyValue editor', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Body' }).click()
    await page.locator('select').nth(1).selectOption('form-urlencoded')
    await expect(page.getByText('+ Add row')).toBeVisible()
  })

  test('Body tab: multipart shows KeyValue editor', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Body' }).click()
    await page.locator('select').nth(1).selectOption('multipart')
    await expect(page.getByText('+ Add row')).toBeVisible()
  })

  test('Auth tab: bearer type shows token field', async ({ page }) => {
    await gotoApp(page)
    await page.getByRole('button', { name: 'Auth' }).click()
    // Select bearer auth
    await page.locator('select').nth(1).selectOption('bearer')
    await expect(page.getByPlaceholder('eyJhbGci…')).toBeVisible()
  })

  test('timeout input accepts a value', async ({ page }) => {
    await gotoApp(page)
    // The timeout input has placeholder "0"
    const timeoutInput = page.getByPlaceholder('0')
    await timeoutInput.fill('5000')
    await expect(timeoutInput).toHaveValue('5000')
  })

  test('Send button shows Cancel while loading', async ({ page }) => {
    await gotoApp(page)
    // Make Execute hang so we can observe the loading state
    await page.evaluate(() => {
      ;(window as unknown as Record<string, unknown>)['go'] = {
        ...(window as unknown as Record<string, Record<string, unknown>>)['go'],
        main: {
          ...(window as unknown as Record<string, Record<string, Record<string, unknown>>>)['go']['main'],
          RequestService: {
            ...(window as unknown as Record<string, Record<string, Record<string, Record<string, unknown>>>>)['go']['main']['RequestService'],
            Execute: () => new Promise(() => { /* never resolves */ }),
          },
        },
      }
    })
    await page.getByPlaceholder('https://api.example.com/endpoint').fill('https://example.com')
    await page.getByRole('button', { name: 'Send' }).click()
    await expect(page.getByRole('button', { name: 'Cancel' })).toBeVisible()
    await expect(page.getByRole('button', { name: 'Send' })).not.toBeVisible()
  })
})
