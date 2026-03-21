// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Users tab', () => {
  test('Users tab is visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('tab', { name: 'Users' })).toBeVisible();
  });

  test('Users table shows default user and test users', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    // Wait for table content to load
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('default')).toBeVisible();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    await expect(page.getByText('disableduser')).toBeVisible({ timeout: 10000 });
  });

  test('Default user has no Delete button', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    // Find the row with the default user
    const defaultRow = page.locator('table tbody tr').filter({ hasText: 'default' });
    await expect(defaultRow).toBeVisible();
    // Default user row should have Edit but no Delete
    await expect(defaultRow.getByText('Edit')).toBeVisible();
    await expect(defaultRow.getByText('Delete')).not.toBeVisible();
  });

  test('Non-default user has Delete button', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'testuser1' });
    await expect(userRow.getByText('Delete')).toBeVisible();
  });

  test('Disabled user shows Disabled badge', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('disableduser')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'disableduser' });
    await expect(userRow.getByText('Disabled')).toBeVisible();
  });

  test('Add user button opens modal', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('usertest')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: '+ Add user' }).click();
    await expect(page.getByText('Add user')).toBeVisible();
    await expect(page.getByText('Username')).toBeVisible();
    await expect(page.getByText('Password')).toBeVisible();
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
  test('Watch tab shows user dropdown when multiple users exist', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Watch' }).click();
    // Should show the user dropdown
    await expect(page.getByText('Show connection details for user:')).toBeVisible({ timeout: 10000 });
  });
});
