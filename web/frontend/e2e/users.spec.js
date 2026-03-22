// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Users tab', () => {
  test('Users tab is visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('tab', { name: 'Users' })).toBeVisible();
  });

  test('Users table shows all users', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    // Wait for table content to load
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('disableduser')).toBeVisible({ timeout: 10000 });
  });

  test('All users have Edit and Delete buttons', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'usertest' });
    await expect(userRow.getByText('Edit')).toBeVisible();
    await expect(userRow.getByText('Delete')).toBeVisible();
  });

  test('Disabled user shows Disabled badge', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('disableduser')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'disableduser' });
    // Use exact match to avoid matching the username substring "disabled" in "disableduser"
    await expect(userRow.getByText('Disabled', { exact: true })).toBeVisible();
  });

  test('Add user button opens modal', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /Add user/ }).click();
    // Modal should open with form fields
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await expect(page.locator('label:has-text("Username")')).toBeVisible();
    await expect(page.locator('label:has-text("Password")')).toBeVisible();
  });

  test('Edit button opens edit modal', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'testuser1' });
    await userRow.getByText('Edit').click();
    await expect(page.getByText('Edit user: testuser1')).toBeVisible();
  });
});

test.describe('Watch tab user dropdown', () => {
  test('Watch tab shows user dropdown', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Watch' }).click();
    // Should show the user dropdown
    await expect(page.getByText('Show connection details for user:')).toBeVisible({ timeout: 10000 });
  });
});
