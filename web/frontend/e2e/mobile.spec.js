// @ts-check
import { test, expect, devices } from '@playwright/test';

// Phone viewport smoke tests: every main view fits the screen width (no horizontal page scroll),
// all top-level tabs are reachable, and card actions are touch-sized.
// Run for the widest and narrowest supported phones (Pixel 7: 412px, iPhone 13: 390px).
// defaultBrowserType can't be set inside a test file, so drop it from the device descriptors.
const PHONES = ['Pixel 7', 'iPhone 13'].map((name) => {
  const { defaultBrowserType: _browser, ...descriptor } = devices[name];
  return { name, descriptor };
});

async function expectNoHorizontalOverflow(page, view) {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(scrollWidth, `${view}: page must not scroll horizontally`).toBeLessThanOrEqual(clientWidth);
}

// Content that proves each tab has rendered its data before checking overflow (no fixed delays).
const TAB_READY = {
  Channels: (page) => page.getByText('Test Channel', { exact: true }),
  Processing: (page) => page.getByRole('heading', { name: 'Processing order' }),
  Watch: (page) => page.getByText('Show connection details for user:'),
  // testuser1 is seeded by start-e2e-server.mjs goldenSettings.
  Users: (page) => page.getByRole('row', { name: /testuser1/ }),
};

for (const { name: phone, descriptor } of PHONES) {
  test.describe(`Mobile layout (${phone})`, () => {
    test.use(descriptor);

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

      for (const [name, ready] of Object.entries(TAB_READY)) {
        await page.getByRole('tab', { name, exact: true }).click();
        await expect(ready(page).first(), `${name}: content loaded`).toBeVisible({ timeout: 15000 });
        await expectNoHorizontalOverflow(page, name);
      }

      await page.getByRole('tab', { name: 'Processing', exact: true }).click();
      await page.getByRole('tab', { name: 'Inclusions & exclusions' }).click();
      await expect(page.getByLabel('Section', { exact: true })).toBeVisible();
      await expectNoHorizontalOverflow(page, 'Processing > Inclusions & exclusions');

      await page.goto('/settings');
      await expect(page.getByRole('heading', { name: /Settings \(settings\.json\)/i })).toBeVisible();
      await page.waitForLoadState('networkidle');
      await expectNoHorizontalOverflow(page, 'Settings');
    });

    test('group and channel cards are condensed, touch-sized, with a sort control', async ({ page }) => {
      await page.goto('/');
      await expect(page.getByText('Group1', { exact: true })).toBeVisible({ timeout: 15000 });
      await expect(page.getByLabel('Sort by')).toBeVisible();
      const viewChannels = await page.getByRole('button', { name: 'View channels' }).first().boundingBox();
      expect(Math.round(viewChannels?.height ?? 0)).toBeGreaterThanOrEqual(40);
      await expectCondensedRows(page, 'Groups');
      await expectOverflowMenuActions(page);
      await expectEditFocusesInput(page);

      await page.getByRole('tab', { name: 'Channels', exact: true }).click();
      await expect(page.getByText('Test Channel', { exact: true })).toBeVisible({ timeout: 15000 });
      await expectCondensedRows(page, 'Channels');
      // Show both included (with stream) and excluded (without stream) channels: the menu must not shift.
      await expectMenusAligned(page, 'Channels');
      const play = page.getByRole('link', { name: 'Open stream' }).first();
      await expect(play).toBeVisible();
      expect(Math.round((await play.boundingBox())?.height ?? 0)).toBeGreaterThanOrEqual(40);
      await expectOverflowMenuActions(page);
    });
  });
}

// Each Groups/Channels card is one condensed row: at most 72px tall including the row chrome, and the row
// fills the card (actions pinned right) without sticking out of it — EUI's inline cell span would otherwise
// shrink-wrap short names and let long names overflow the page.
async function expectCondensedRows(page, view) {
  const rows = await page.locator('.euiTableRow:has([data-testid="compact-row"])').evaluateAll((els) =>
    els.map((r) => {
      const card = r.getBoundingClientRect();
      const row = r.querySelector('[data-testid="compact-row"]').getBoundingClientRect();
      return { height: card.height, gapRight: card.right - row.right };
    })
  );
  expect(rows.length, `${view}: condensed rows rendered`).toBeGreaterThan(0);
  expect(Math.max(...rows.map((r) => r.height)), `${view}: row height`).toBeLessThanOrEqual(72);
  for (const r of rows) {
    expect(r.gapRight, `${view}: row fills the card`).toBeGreaterThanOrEqual(0);
    expect(r.gapRight, `${view}: row fills the card`).toBeLessThanOrEqual(24);
  }
}

// Every visible row has a status icon (not color alone) and the "More actions" button at the same x position.
async function expectMenusAligned(page, view) {
  const rows = page.locator('[data-testid="compact-row"]');
  const count = await rows.count();
  expect(count, `${view}: rows`).toBeGreaterThan(0);
  expect(await page.locator('[data-testid="compact-row-status"]').count(), `${view}: status icon per row`).toBe(count);
  const xs = await page.getByRole('button', { name: 'More actions' }).evaluateAll((els) => els.map((e) => Math.round(e.getBoundingClientRect().right)));
  expect(new Set(xs).size, `${view}: More actions aligned (${xs.join(',')})`).toBe(1);
}

// Secondary actions live in a 40px "More actions" menu that exposes include/exclude as 40px items.
async function expectOverflowMenuActions(page) {
  const more = page.getByRole('button', { name: 'More actions' }).first();
  expect(Math.round((await more.boundingBox())?.height ?? 0)).toBeGreaterThanOrEqual(40);
  await more.click();
  for (const name of ['Edit', 'Add to inclusions', 'Add to exclusions']) {
    const item = page.getByRole('button', { name, exact: true }).last();
    await expect(item, `menu item ${name}`).toBeVisible();
    expect(Math.round((await item.boundingBox())?.height ?? 0), `menu item ${name} height`).toBeGreaterThanOrEqual(40);
  }
  await page.keyboard.press('Escape');
  await expect(page.getByRole('button', { name: 'Add to exclusions', exact: true })).toHaveCount(0);
}

// Choosing Edit from the ⋯ menu must leave focus in the inline input (the popover must not return focus to the
// menu button), so the phone keyboard opens and Enter/Escape work.
async function expectEditFocusesInput(page) {
  await page.getByRole('button', { name: 'More actions' }).first().click();
  await page.getByRole('button', { name: 'Edit', exact: true }).last().click();
  const input = page.locator('.euiTableRow input[type="text"]').first();
  await expect(input).toBeVisible();
  // Wait past the popover's 250ms closing transition, when a returned focus would land on the button.
  await page.waitForTimeout(400);
  await expect(input).toBeFocused();
  await page.keyboard.press('Escape');
  await expect(page.locator('.euiTableRow input[type="text"]')).toHaveCount(0);
}
