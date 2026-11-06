import { defineConfig, devices } from '@playwright/test'

const SMOKE = process.env.E2E_SMOKE === '1'

export default defineConfig({
  testDir: './tests',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  reporter: [['html', { open: 'never' }], ['list']],
  use: { trace: 'on-first-retry' },

  projects: [
    {
      name: 'frontend',
      testMatch: /frontend\/.+\.spec\.ts/,
      use: {
        baseURL: 'http://localhost:5173',
        ...devices['Desktop Chrome'],
      },
    },
    {
      name: 'smoke',
      testMatch: /smoke\/.+\.spec\.ts/,
      use: {
        baseURL: 'http://localhost:34115',
        ...devices['Desktop Chrome'],
      },
    },
  ],

  webServer: SMOKE
    ? {
        command: 'wails dev',
        cwd: '../wails',
        url: 'http://localhost:34115',
        timeout: 120_000,
        reuseExistingServer: true,
      }
    : {
        command: 'pnpm dev',
        cwd: '../frontend',
        url: 'http://localhost:5173',
        reuseExistingServer: !process.env.CI,
      },
})
