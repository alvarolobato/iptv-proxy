// @ts-check
const { defineConfig, devices } = require('@playwright/test');
const path = require('path');

module.exports = defineConfig({
  testDir: './e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: 1,
  reporter: 'html',
  use: {
    baseURL: 'http://localhost:18081',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: `node ${path.join(__dirname, 'scripts/start-e2e-server.mjs')}`,
    url: 'http://localhost:18081/api/ready',
    reuseExistingServer: false,
    timeout: 90000,
    cwd: path.resolve(__dirname, '../..'),
  },
});
