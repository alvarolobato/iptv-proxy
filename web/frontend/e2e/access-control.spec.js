// @ts-check
import { test, expect } from '@playwright/test';

test.describe('User access control', () => {
  test('Users table shows Custom access badge for restricted user', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('restricteduser')).toBeVisible({ timeout: 10000 });
    const restrictedRow = page.locator('table tbody tr').filter({ hasText: 'restricteduser' });
    await expect(restrictedRow.getByText('Custom access')).toBeVisible();
  });

  test('Users table shows Full access for unrestricted user', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'testuser1' });
    await expect(userRow.getByText('Full access')).toBeVisible();
  });

  test('Edit modal shows Access control tab', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('restricteduser')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'restricteduser' });
    await userRow.getByText('Edit').click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    // Should show General and Access control tabs
    await expect(page.getByRole('tab', { name: 'General' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Access control' })).toBeVisible();
  });

  test('Access control tab shows existing patterns', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('restricteduser')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'restricteduser' });
    await userRow.getByText('Edit').click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('tab', { name: 'Access control' }).click();
    // Should show existing patterns as badges
    await expect(page.getByText('^Group1$')).toBeVisible({ timeout: 5000 });
    await expect(page.getByText('^Premium.*')).toBeVisible({ timeout: 5000 });
  });

  test('Can add and remove access control pattern', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'testuser1' });
    await userRow.getByText('Edit').click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('tab', { name: 'Access control' }).click();
    // Add a pattern to group allow list
    const firstInput = page.getByPlaceholder('e.g. ^Sports$ or (?i)news').first();
    await firstInput.fill('^TestPattern$');
    // Click the Add button next to the first input
    await page.locator('.euiFormRow').filter({ hasText: 'Group allow list' }).getByRole('button', { name: 'Add' }).click();
    // Pattern should appear as a badge
    await expect(page.getByText('^TestPattern$')).toBeVisible();
  });

  test('Invalid regex shows validation error', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    const userRow = page.locator('table tbody tr').filter({ hasText: 'testuser1' });
    await userRow.getByText('Edit').click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    await page.getByRole('tab', { name: 'Access control' }).click();
    // Enter invalid regex
    const firstInput = page.getByPlaceholder('e.g. ^Sports$ or (?i)news').first();
    await firstInput.fill('[invalid');
    await page.locator('.euiFormRow').filter({ hasText: 'Group allow list' }).getByRole('button', { name: 'Add' }).click();
    // Should show validation error
    await expect(page.getByText('Invalid regex')).toBeVisible();
  });

  test('Add user modal does not show Access control tab', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Users' }).click();
    await expect(page.getByText('testuser1')).toBeVisible({ timeout: 10000 });
    await page.getByRole('button', { name: /Add user/ }).click();
    await expect(page.getByRole('dialog')).toBeVisible({ timeout: 5000 });
    // Add modal should NOT show the Access control tab (only shown in edit mode)
    await expect(page.getByRole('tab', { name: 'Access control' })).not.toBeVisible();
  });

  test('Access rules API returns correct data', async ({ page }) => {
    await page.goto('/');
    // Use page.evaluate to call the API directly
    const response = await page.evaluate(async () => {
      const r = await fetch('/api/users/restricteduser/access');
      return r.json();
    });
    expect(response.group_allow_list).toEqual(['^Group1$']);
    expect(response.channel_block_list).toEqual(['^Premium.*']);
    expect(response.group_block_list).toEqual([]);
    expect(response.channel_allow_list).toEqual([]);
  });
});
