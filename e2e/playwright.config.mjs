import { defineConfig, devices } from '@playwright/test';
import { HTTP_PORT } from './ports.mjs';

const baseURL = `http://127.0.0.1:${HTTP_PORT}`;

export default defineConfig({
  testDir: './tests',
  // One server, one database: the specs run in order (first-run setup first).
  workers: 1,
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: 0,
  timeout: 60_000,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
  },
  projects: [{ name: 'chromium', use: { ...devices['Desktop Chrome'] } }],
  webServer: {
    command: 'node start-server.mjs',
    url: `${baseURL}/health/live`,
    timeout: 60_000,
    reuseExistingServer: false,
    stdout: 'pipe',
    stderr: 'pipe',
  },
});
