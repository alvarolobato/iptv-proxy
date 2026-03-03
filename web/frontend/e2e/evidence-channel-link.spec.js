// @ts-check
/**
 * Evidence test: uses real /api/channels data, opens Channels tab,
 * captures screenshot and Actions cell HTML, asserts Open stream link is visible.
 * Run: npm run e2e -- --grep "Evidence: channel Open stream link"
 * Output: evidence/channels-response.json, evidence/screenshot-channels.png, evidence/actions-cell.html
 */
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { test, expect } from '@playwright/test';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const EVIDENCE_DIR = path.join(__dirname, 'evidence');

test.describe('Evidence: channel Open stream link', () => {
  test('capture API data, open Channels, screenshot and HTML, assert link visible', async ({ page, request }) => {
    fs.mkdirSync(EVIDENCE_DIR, { recursive: true });

    // 1. Get real API response and save it
    const channelsRes = await request.get('/api/channels');
    expect(channelsRes.ok()).toBeTruthy();
    const channels = await channelsRes.json();
    const responsePath = path.join(EVIDENCE_DIR, 'channels-response.json');
    fs.writeFileSync(responsePath, JSON.stringify(channels, null, 2), 'utf8');

    const withStream = channels.filter((c) => c.stream_url && String(c.stream_url).length > 0);
    const sampleStreamUrl = withStream.length > 0 ? withStream[0].stream_url : null;

    // 2. Open app and go to Channels tab
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels' }).click();
    // Wait for table to have real data (Included/Excluded in table body)
    await page.locator('table tbody').locator('text=Included').first().waitFor({ state: 'visible', timeout: 15000 }).catch(() => {});
    await page.locator('table tbody tr').first().waitFor({ state: 'visible', timeout: 5000 });

    // 3. Screenshot with data populated
    const screenshotPath = path.join(EVIDENCE_DIR, 'screenshot-channels.png');
    await page.screenshot({ path: screenshotPath, fullPage: true });

    // 4. Capture first row Actions cell HTML
    const actionsCell = page.locator('table tbody tr').first().locator('td').last();
    const actionsHtml = await actionsCell.evaluate((el) => el.outerHTML);
    const htmlPath = path.join(EVIDENCE_DIR, 'actions-cell.html');
    fs.writeFileSync(htmlPath, `<!-- First channel row Actions cell -->\n${actionsHtml}\n`, 'utf8');

    // 5. Assert: API should return at least one stream_url and the link must be visible
    expect(withStream.length, `API returned no channel with stream_url. See ${responsePath}`).toBeGreaterThanOrEqual(1);

    const openStreamLink = page.getByRole('link', { name: 'Open stream' }).or(page.getByTestId('channel-open-stream'));
    await expect(openStreamLink.first(), 'Open stream link should be visible in Actions. See screenshot and actions-cell.html').toBeVisible({ timeout: 3000 });

    if (sampleStreamUrl) {
      const firstLinkHref = await openStreamLink.first().getAttribute('href');
      const allStreamUrls = new Set(withStream.map((c) => c.stream_url));
      expect(allStreamUrls.has(firstLinkHref), `Open stream link href should be one of the channel stream_urls. Got: ${firstLinkHref}`).toBeTruthy();
    }

    test.info().attach('channels-response.json', { path: responsePath });
    test.info().attach('screenshot-channels.png', { path: screenshotPath });
    test.info().attach('actions-cell.html', { path: htmlPath });
  });
});
