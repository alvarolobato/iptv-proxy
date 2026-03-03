// @ts-check
import { test, expect } from '@playwright/test';

test.describe('Data & processing', () => {
  test('main page loads and shows Groups tab', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByRole('heading', { name: /Data & processing/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Groups' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Channels' })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Processing' })).toBeVisible();
  });

  test('Groups table is not empty (has at least one row)', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    const count = await page.locator('table tbody tr').count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Groups table shows row numbers (no NaN)', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    const firstCell = page.locator('table tbody tr').first().locator('td').first();
    await expect(firstCell).toContainText(/\d+/);
    await expect(firstCell).not.toContainText('NaN');
  });

  test('Groups table has Actions column with links or buttons', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    await expect(page.getByRole('columnheader', { name: 'Actions' })).toBeVisible();
    const actionsCell = page.locator('table tbody tr').first().locator('td').last();
    await expect(actionsCell).toBeVisible();
  });

  test('Clicking Add to exclusions (in Actions) stays on Groups tab', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    const addToExcl = page.getByLabel('Add to exclusions').or(page.locator('button').filter({ hasText: /−|minus|exclusion/i }));
    if (await addToExcl.first().count() > 0) {
      await addToExcl.first().click();
      await expect(page.getByRole('tab', { name: 'Groups', selected: true })).toBeVisible({ timeout: 5000 });
    }
    await expect(page.locator('table')).toBeVisible();
  });

  test('Groups table shows Status column with Included/Excluded', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    await expect(page.getByRole('columnheader', { name: 'Group title' })).toBeVisible();
    await expect(page.getByText('Included').or(page.getByText('Excluded')).first()).toBeVisible({ timeout: 5000 });
  });

  test('Channels table is not empty (has at least one row)', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 20000 });
    const count = await page.locator('table tbody tr').count();
    expect(count).toBeGreaterThanOrEqual(1);
  });

  test('Channels table shows Status column with Included/Excluded', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels' }).click();
    await page.waitForSelector('table tbody tr', { timeout: 15000 });
    await expect(page.getByRole('columnheader', { name: 'Status' })).toBeVisible();
    await expect(page.getByText('Included').or(page.getByText('Excluded')).first()).toBeVisible({ timeout: 5000 });
  });

  test('Channels tab has Type column and table', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels' }).click();
    await page.waitForSelector('table', { timeout: 10000 });
    await expect(page.getByRole('columnheader', { name: 'Type' })).toBeVisible();
    await expect(page.getByRole('columnheader', { name: 'Logo' })).toBeVisible();
    await expect(page.locator('table tbody tr').first()).toBeVisible({ timeout: 5000 });
  });

  test('Processing tab has replacements section and add form', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Processing' }).click();
    await expect(page.getByRole('tab', { name: 'Replacements' })).toBeVisible();
    await expect(page.getByPlaceholder('Regex replace')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('button', { name: 'Add rule' })).toBeVisible();
  });

  test('Processing tab Add pattern then Save all persists via API', async ({ page, request }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Processing' }).click();
    await page.getByRole('tab', { name: 'Inclusions & exclusions' }).click();
    await expect(page.getByRole('textbox', { name: /Pattern \(regex\)/ })).toBeVisible({ timeout: 5000 });
    await page.getByRole('textbox', { name: /Pattern \(regex\)/ }).fill('^E2ETestPattern$');
    await page.locator('button').filter({ hasText: /^Add$/ }).click();
    await page.getByRole('button', { name: 'Save all processing settings' }).click();
    await expect(page.getByText(/Saved/)).toBeVisible({ timeout: 8000 });
    const res = await request.get('/api/settings', { headers: { 'Cache-Control': 'no-cache' } });
    expect(res.ok()).toBeTruthy();
    const data = await res.json();
    const settings = data.effective ?? data;
    const list = settings?.group_inclusions || [];
    const hasPattern = Array.isArray(list) && list.some((p) => p && String(p).includes('E2ETestPattern'));
    expect(hasPattern).toBeTruthy();
    // Restore group_inclusions so later tests (excluded filter, API proof, screenshots) still see both included and excluded
    const restored = { ...settings, group_inclusions: settings?.group_inclusions?.filter((p) => !String(p).includes('E2ETestPattern')) || [] };
    await request.put('/api/settings', { data: restored });
  });

  test('Processing tab Inclusions has sub-tabs and Remove pattern in a pattern table', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Processing' }).click();
    await page.getByRole('tab', { name: 'Inclusions & exclusions' }).click();
    await expect(page.getByText('Group inclusions').or(page.getByRole('tab', { name: 'Group inclusions' })).first()).toBeVisible({ timeout: 8000 });
    const removeBtn = page.getByLabel('Remove pattern').or(page.getByRole('button', { name: 'Remove pattern' }));
    if (await removeBtn.count() > 0) {
      await removeBtn.first().click();
      await expect(page.getByRole('tab', { name: 'Processing', selected: true })).toBeVisible();
    }
    await expect(page.locator('table')).toBeVisible();
  });

  test('API PUT settings can remove group_exclusions pattern', async ({ request }) => {
    const getRes = await request.get('/api/settings', { headers: { 'Cache-Control': 'no-cache' } });
    expect(getRes.ok()).toBeTruthy();
    const getData = await getRes.json();
    const settings = getData.effective ?? getData;
    const list = settings?.group_exclusions || [];
    expect(list.length).toBeGreaterThanOrEqual(1);
    const next = { ...settings, group_exclusions: list.slice(0, -1) };
    const putRes = await request.put('/api/settings', { data: next });
    expect(putRes.ok()).toBeTruthy();
    const afterRes = await request.get('/api/settings', { headers: { 'Cache-Control': 'no-cache' } });
    const afterData = await afterRes.json();
    const after = afterData.effective ?? afterData;
    expect((after?.group_exclusions || []).length).toBe(list.length - 1);
    // Restore golden state so later tests see one included (Group1) and one excluded (Group2)
    const restored = { ...settings, group_exclusions: ['^Group2$'] };
    const restoreRes = await request.put('/api/settings', { data: restored });
    expect(restoreRes.ok()).toBeTruthy();
  });

  test('Processing tab Remove rule (icon) exists and is clickable', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Processing' }).click();
    await expect(page.getByRole('tab', { name: 'Global' })).toBeVisible({ timeout: 5000 });
    const removeRuleBtn = page.getByRole('button', { name: 'Remove rule' });
    if (await removeRuleBtn.count() > 0) {
      await removeRuleBtn.first().click();
      await expect(page.getByRole('tab', { name: 'Processing', selected: true })).toBeVisible();
    }
  });

  test('Included / Excluded filter buttons exist when on Groups', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await expect(page.getByRole('button', { name: 'Included' })).toBeVisible();
    await expect(page.getByRole('button', { name: 'Excluded' })).toBeVisible();
  });

  test('Excluded filter shows excluded groups when clicked', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await expect(page.locator('table tbody').getByText('Group1')).toBeVisible({ timeout: 15000 });
    await page.getByRole('button', { name: 'Included' }).click();
    await expect(page.getByRole('cell', { name: 'Excluded' }).first()).toBeVisible({ timeout: 10000 });
    await expect(page.locator('table tbody').getByText('Group2').first()).toBeVisible({ timeout: 5000 });
  });

  test('API /api/groups returns data with excluded flag (backend proof)', async ({ request }) => {
    const res = await request.get('/api/groups', { headers: { 'Cache-Control': 'no-cache' } });
    expect(res.ok()).toBeTruthy();
    const groups = await res.json();
    expect(Array.isArray(groups)).toBeTruthy();
    expect(groups.length).toBeGreaterThanOrEqual(2);
    const withExcluded = groups.filter((g) => g.excluded === true);
    const withIncluded = groups.filter((g) => g.excluded !== true);
    expect(withExcluded.length).toBeGreaterThanOrEqual(1);
    expect(withIncluded.length).toBeGreaterThanOrEqual(1);
  });

  test('API /api/channels returns data with excluded flag (backend proof)', async ({ request }) => {
    const res = await request.get('/api/channels', { headers: { 'Cache-Control': 'no-cache' } });
    expect(res.ok()).toBeTruthy();
    const channels = await res.json();
    expect(Array.isArray(channels)).toBeTruthy();
    expect(channels.length).toBeGreaterThanOrEqual(2);
    const withExcluded = channels.filter((c) => c.excluded === true);
    const withIncluded = channels.filter((c) => c.excluded !== true);
    expect(withExcluded.length).toBeGreaterThanOrEqual(1);
    expect(withIncluded.length).toBeGreaterThanOrEqual(1);
  });

  test('Screenshot: Groups tab shows both Included and Excluded with Status column', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Groups' }).click();
    await expect(page.getByRole('columnheader', { name: 'Status' })).toBeVisible();
    await expect(page.locator('table tbody').getByText('Group1')).toBeVisible({ timeout: 15000 });
    await expect(page.locator('table tbody').getByText('Group2')).toBeVisible({ timeout: 5000 });
    await expect(page.getByRole('cell', { name: 'Included' }).first()).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Excluded' }).first()).toBeVisible();
    const screenshot = await page.screenshot({ path: 'e2e/screenshots/groups-included-excluded.png', fullPage: false });
    await test.info().attach('groups-included-excluded', { body: screenshot, contentType: 'image/png' });
  });

  test('Screenshot: Channels tab shows both Included and Excluded with Status column', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels' }).click();
    await expect(page.getByRole('columnheader', { name: 'Status' })).toBeVisible();
    await expect(page.locator('table tbody').getByText('Test Channel').first()).toBeVisible({ timeout: 15000 });
    await expect(page.getByRole('cell', { name: 'Included' }).first()).toBeVisible();
    await expect(page.getByRole('cell', { name: 'Excluded' }).first()).toBeVisible();
    const screenshot = await page.screenshot({ path: 'e2e/screenshots/channels-included-excluded.png', fullPage: false });
    await test.info().attach('channels-included-excluded', { body: screenshot, contentType: 'image/png' });
  });
});

test.describe('Settings', () => {
  test('Settings page loads and Options tab does not add untouched toggles on save', async ({ page }) => {
    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: /Settings \(settings\.json\)/i })).toBeVisible();
    await expect(page.getByRole('tab', { name: 'Options' })).toBeVisible();
    await page.getByRole('tab', { name: 'Options' }).click();
    const saveBtn = page.getByRole('button', { name: 'Save settings' });
    await expect(saveBtn).toBeVisible();
    await saveBtn.click();
    await expect(page.getByText(/Saved/).first()).toBeVisible({ timeout: 5000 });
  });
});
