// @ts-check
// Use when proxy is already running with a real M3U (e.g. run-user-m3u-evidence.mjs).
// No webServer; baseURL must point at the running proxy UI.
const { defineConfig, devices } = require('@playwright/test');
const path = require('path');

module.exports = defineConfig({
  testDir: path.join(__dirname, 'e2e'),
  fullyParallel: false,
  workers: 1,
  timeout: 60000,
  use: {
    baseURL: 'http://localhost:18081',
    trace: 'on-first-retry',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  // No webServer - proxy is started by run-user-m3u-evidence.mjs
});
