// Smoke test of a built server with the embedded GUI: first-run setup,
// dashboard, log out, log back in. The server starts on an empty database
// (start-server.mjs), so the setup step only works once per run.
import { test, expect } from '@playwright/test';

const admin = {
  name: 'E2E Admin',
  email: 'admin@example.com',
  password: 'e2e-password-0123',
};

test.describe.configure({ mode: 'serial' });

let errors = [];
test.beforeEach(({ page }) => {
  errors = [];
  page.on('console', (msg) => {
    // A signed-out page probes the session and gets a 401: that's expected.
    if (msg.type() === 'error' && !/status of 401 \(Unauthorized\)/.test(msg.text())) errors.push(msg.text());
  });
  page.on('pageerror', (err) => errors.push(err.message));
});
test.afterEach(() => {
  expect(errors, 'browser console errors').toEqual([]);
});

test('first run sends you to setup and creates the admin', async ({ page }) => {
  await page.goto('/');
  await expect(page).toHaveURL(/\/setup$/);

  await page.locator('#name').fill(admin.name);
  await page.locator('#email').fill(admin.email);
  await page.locator('#password').fill(admin.password);
  await page.locator('#confirmPassword').fill(admin.password);
  await page.getByRole('button', { name: 'Create account' }).click();

  await expect(page).toHaveURL(/\/dashboard$/, { timeout: 15_000 });
  await expect(page.getByText(admin.email).first()).toBeVisible();
});

test('the admin can log out and sign back in', async ({ page }) => {
  await page.goto('/login');
  await page.locator('#email').fill(admin.email);
  await page.locator('#password').fill(admin.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/dashboard$/, { timeout: 15_000 });

  await page.getByText(admin.email).first().click();
  await page.getByRole('menuitem', { name: 'Log out' }).click();
  await expect(page).toHaveURL(/\/login$/);

  // Signed out: the app sends you back to the login page.
  await page.goto('/agents');
  await expect(page).toHaveURL(/\/login/);
});

test('the main pages load for a signed-in user', async ({ page }) => {
  await page.goto('/login');
  await page.locator('#email').fill(admin.email);
  await page.locator('#password').fill(admin.password);
  await page.getByRole('button', { name: 'Sign in' }).click();
  await expect(page).toHaveURL(/\/dashboard$/, { timeout: 15_000 });

  for (const path of ['/agents', '/policies', '/destinations', '/snapshots', '/jobs', '/settings']) {
    await page.goto(path);
    await expect(page).toHaveURL(new RegExp(`${path}$`));
    await expect(page.locator('main')).toBeVisible();
  }
});
