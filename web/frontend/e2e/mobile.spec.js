// @ts-check
import { test, expect, devices } from '@playwright/test';

// Phone viewport smoke tests: every main view fits the screen width (no horizontal page scroll),
// all top-level tabs are reachable, and card actions are touch-sized.
// defaultBrowserType can't be set inside a test file, so drop it from the device descriptor.
const { defaultBrowserType: _browser, ...pixel7 } = devices['Pixel 7'];
test.use(pixel7);

async function expectNoHorizontalOverflow(page, view) {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(scrollWidth, `${view}: page must not scroll horizontally`).toBeLessThanOrEqual(clientWidth);
}

test.describe('Mobile layout (Pixel 7)', () => {
  test('main views fit the viewport and all tabs are visible', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Group1', { exact: true })).toBeVisible({ timeout: 15000 });
    const width = page.viewportSize()?.width ?? 0;
    for (const name of ['Groups', 'Channels', 'Processing', 'Watch', 'Users']) {
      const box = await page.getByRole('tab', { name, exact: true }).boundingBox();
      expect(box, `tab ${name} rendered`).not.toBeNull();
      expect((box?.x ?? 0) + (box?.width ?? 0), `tab ${name} within viewport`).toBeLessThanOrEqual(width);
    }
    await expectNoHorizontalOverflow(page, 'Groups');

    for (const name of ['Channels', 'Processing', 'Watch', 'Users']) {
      await page.getByRole('tab', { name, exact: true }).click();
      await page.waitForTimeout(500);
      await expectNoHorizontalOverflow(page, name);
    }

    await page.getByRole('tab', { name: 'Processing', exact: true }).click();
    await page.getByRole('tab', { name: 'Inclusions & exclusions' }).click();
    await expect(page.getByLabel('Section', { exact: true })).toBeVisible();
    await expectNoHorizontalOverflow(page, 'Processing > Inclusions & exclusions');

    await page.goto('/settings');
    await expect(page.getByRole('heading', { name: /Settings \(settings\.json\)/i })).toBeVisible();
    await page.waitForTimeout(500);
    await expectNoHorizontalOverflow(page, 'Settings');
  });

  test('group and channel cards have touch-sized actions and a sort control', async ({ page }) => {
    await page.goto('/');
    await expect(page.getByText('Group1', { exact: true })).toBeVisible({ timeout: 15000 });
    await expect(page.getByLabel('Sort by')).toBeVisible();
    const viewChannels = await page.getByRole('button', { name: 'View channels' }).first().boundingBox();
    expect(viewChannels?.height ?? 0).toBeGreaterThanOrEqual(40);

    await page.getByRole('tab', { name: 'Channels', exact: true }).click();
    await expect(page.getByText('Test Channel', { exact: true })).toBeVisible({ timeout: 15000 });
    const exclude = await page.getByRole('button', { name: 'Add to exclusions' }).first().boundingBox();
    expect(exclude?.height ?? 0).toBeGreaterThanOrEqual(40);
    const play = page.getByRole('link', { name: 'Open stream' }).first();
    await expect(play).toBeVisible();
    expect((await play.boundingBox())?.height ?? 0).toBeGreaterThanOrEqual(40);
  });
});
