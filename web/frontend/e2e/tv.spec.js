// @ts-check
import fs from 'fs';
import { test, expect, devices } from '@playwright/test';

// TV view: dedicated play view listing only playable channels (included, with a stream_url), and the
// player sheet with in-browser playback plus platform-specific external player links.
// defaultBrowserType can't be set inside a test file, so drop it from the device descriptors.
const { defaultBrowserType: _pixelBrowser, ...pixel7 } = devices['Pixel 7'];
const { defaultBrowserType: _iphoneBrowser, ...iphone13 } = devices['iPhone 13'];

const PLAY_STORE = 'https://play.google.com/store/apps/details?id=org.videolan.vlc';

// Keep tests offline: fixture stream URLs are proxied to example.com.
async function blockStreams(page) {
  await page.route('http://localhost:18080/**', (route) => route.abort());
}

async function fixtureChannels(request) {
  const res = await request.get('/api/channels');
  expect(res.ok()).toBeTruthy();
  const channels = await res.json();
  const playable = channels.filter((c) => c.excluded !== true && c.stream_url);
  const excluded = channels.filter((c) => c.excluded === true);
  expect(playable.length, 'fixture needs an included channel with stream_url').toBeGreaterThanOrEqual(1);
  expect(excluded.length, 'fixture needs an excluded channel').toBeGreaterThanOrEqual(1);
  return { playable: playable[0], excluded: excluded[0] };
}

const playButton = (page, name) => page.getByTestId('tv-list').getByRole('button', { name: `Play ${name}`, exact: true });

async function expectNoHorizontalOverflow(page, view) {
  const { scrollWidth, clientWidth } = await page.evaluate(() => ({
    scrollWidth: document.documentElement.scrollWidth,
    clientWidth: document.documentElement.clientWidth,
  }));
  expect(scrollWidth, `${view}: page must not scroll horizontally`).toBeLessThanOrEqual(clientWidth);
}

async function openSheet(page, name) {
  await playButton(page, name).click();
  const sheet = page.getByTestId('player-sheet');
  await expect(sheet).toBeVisible();
  return sheet;
}

test.describe('TV view', () => {
  test('lists only included channels with a stream URL', async ({ page, request }) => {
    const { playable, excluded } = await fixtureChannels(request);
    await blockStreams(page);
    await page.goto('/tv');
    await expect(playButton(page, playable.name)).toBeVisible({ timeout: 15000 });
    await expect(playButton(page, excluded.name)).toHaveCount(0);
    await expect(page.getByTestId('tv-count')).toHaveText('1 channel');
  });

  test('search and group filters narrow the list', async ({ page, request }) => {
    const { playable } = await fixtureChannels(request);
    await blockStreams(page);
    await page.goto('/tv');
    await expect(playButton(page, playable.name)).toBeVisible({ timeout: 15000 });

    await page.getByPlaceholder('Search channels').fill('no-such-channel');
    await expect(page.getByText('No channels match')).toBeVisible();
    await page.getByPlaceholder('Search channels').fill(playable.name);
    await expect(playButton(page, playable.name)).toBeVisible();

    await page.getByPlaceholder('Search channels').fill('');
    await page.getByLabel('Category', { exact: true }).selectOption(playable.group);
    await expect(playButton(page, playable.name)).toBeVisible();
  });

  test('shows only the final list with a Live/VOD switch and categories (no way to reveal excluded)', async ({ page }) => {
    const stream = (path) => `http://localhost:18080/${path}`;
    const channels = [
      { name: 'Live News', group: 'News', type: 'play', excluded: false, stream_url: stream('play/u/p/1.ts') },
      { name: 'Hidden Live', group: 'News', type: 'play', excluded: true, stream_url: stream('play/u/p/2.ts') },
      { name: 'Live Sports', group: 'Sports', type: 'live', excluded: false, stream_url: stream('live/u/p/3.ts') },
      { name: 'A Movie', group: 'Films', type: 'movie', excluded: false, stream_url: stream('movie/u/p/4.mp4') },
      { name: 'Hidden Movie', group: 'Films', type: 'movie', excluded: true, stream_url: stream('movie/u/p/5.mp4') },
      { name: 'A Series', group: 'Shows', type: 'series', excluded: false, stream_url: stream('series/u/p/6.mkv') },
    ];
    await blockStreams(page);
    await page.route('**/api/channels', (route) => route.fulfill({ json: channels }));
    await page.goto('/tv');

    // Live by default; excluded channels never appear and there is no included/excluded toggle.
    await expect(playButton(page, 'Live News')).toBeVisible({ timeout: 15000 });
    await expect(playButton(page, 'Live Sports')).toBeVisible();
    await expect(playButton(page, 'A Movie')).toHaveCount(0);
    await expect(page.getByText(/Hidden/)).toHaveCount(0);
    await expect(page.getByRole('button', { name: /Included|Excluded/ })).toHaveCount(0);

    await page.getByLabel('Category', { exact: true }).selectOption('Sports');
    await expect(playButton(page, 'Live Sports')).toBeVisible();
    await expect(playButton(page, 'Live News')).toHaveCount(0);

    // VOD = movies + series, with a Movies/Series sub-switch; the category resets when it doesn't apply.
    await page.getByRole('button', { name: /^VOD/ }).click();
    await expect(playButton(page, 'A Movie')).toBeVisible();
    await expect(playButton(page, 'A Series')).toBeVisible();
    await expect(page.getByText(/Hidden/)).toHaveCount(0);
    await page.getByRole('button', { name: /^Series/ }).click();
    await expect(playButton(page, 'A Series')).toBeVisible();
    await expect(playButton(page, 'A Movie')).toHaveCount(0);

    // .mkv can't play in the browser: the sheet offers only external options.
    const sheet = await openSheet(page, 'A Series');
    await expect(sheet.getByTestId('player-unsupported')).toBeVisible();
    await expect(sheet.getByTestId('player-video')).toHaveCount(0);
  });

  test('player sheet offers the .m3u download on desktop and reports stream failures', async ({ page, request }) => {
    const { playable } = await fixtureChannels(request);
    await blockStreams(page);
    await page.goto('/tv');
    const sheet = await openSheet(page, playable.name);

    await expect(sheet.getByTestId('player-video')).toBeAttached();
    await expect(sheet.getByTestId('player-link-vlc')).toHaveCount(0);
    await expect(sheet.getByTestId('player-copy-url')).toBeVisible();

    const [download] = await Promise.all([page.waitForEvent('download'), sheet.getByTestId('player-download-m3u').click()]);
    expect(download.suggestedFilename()).toMatch(/\.m3u$/);
    const content = fs.readFileSync(await download.path(), 'utf8');
    expect(content).toBe(`#EXTM3U\n#EXTINF:-1,${playable.name}\n${playable.stream_url}\n`);

    // The stream request is blocked, so the player must surface an error (or stall notice) pointing to external players.
    await expect(sheet.getByTestId('player-error')).toBeVisible({ timeout: 20000 });

    await page.keyboard.press('Escape');
    await expect(sheet).toHaveCount(0);
    await expect(page.getByRole('region', { name: 'Recently played' }).getByRole('button', { name: playable.name, exact: true })).toBeVisible();
  });

  test('header TV button opens the TV view', async ({ page }) => {
    await page.goto('/');
    await page.getByRole('button', { name: 'TV', exact: true }).click();
    await expect(page).toHaveURL(/\/tv$/);
    await expect(page.getByRole('heading', { name: 'TV', exact: true })).toBeVisible();
  });

  test('Channels tab play action opens the player sheet instead of the raw stream', async ({ page, context }) => {
    await blockStreams(page);
    await page.goto('/');
    await page.getByRole('tab', { name: 'Channels', exact: true }).click();
    const link = page.getByTestId('channel-open-stream').first();
    await expect(link).toHaveAttribute('href', /^http:\/\/localhost:18080\//, { timeout: 15000 });
    await link.click();
    await expect(page.getByTestId('player-sheet')).toBeVisible();
    expect(context.pages()).toHaveLength(1);
  });
});

test.describe('TV view on Android (Pixel 7)', () => {
  test.use(pixel7);

  test('fits the screen and offers VLC intent links', async ({ page, request }) => {
    const { playable } = await fixtureChannels(request);
    await blockStreams(page);
    await page.goto('/tv');
    await expect(playButton(page, playable.name)).toBeVisible({ timeout: 15000 });
    await expectNoHorizontalOverflow(page, 'TV');

    const sheet = await openSheet(page, playable.name);
    await expectNoHorizontalOverflow(page, 'TV player sheet');
    const u = new URL(playable.stream_url);
    const target = `intent://${u.host}${u.pathname}${u.search}#Intent;scheme=http;type=video/*;`;
    await expect(sheet.getByTestId('player-link-vlc')).toHaveAttribute(
      'href',
      `${target}package=org.videolan.vlc;S.browser_fallback_url=${encodeURIComponent(PLAY_STORE)};end`
    );
    await expect(sheet.getByTestId('player-link-other-app')).toHaveAttribute('href', `${target}end`);
    await expect(sheet.getByTestId('player-download-m3u')).toBeVisible();
  });
});

test.describe('TV view on iPhone (iPhone 13)', () => {
  test.use(iphone13);

  test('fits the screen and offers VLC for iOS links', async ({ page, request }) => {
    const { playable } = await fixtureChannels(request);
    await blockStreams(page);
    await page.goto('/tv');
    await expect(playButton(page, playable.name)).toBeVisible({ timeout: 15000 });
    await expectNoHorizontalOverflow(page, 'TV');

    const sheet = await openSheet(page, playable.name);
    await expectNoHorizontalOverflow(page, 'TV player sheet');
    await expect(sheet.getByTestId('player-link-vlc')).toHaveAttribute(
      'href',
      `vlc-x-callback://x-callback-url/stream?url=${encodeURIComponent(playable.stream_url)}`
    );
    await expect(sheet.getByTestId('player-link-vlc-alt')).toHaveAttribute('href', `vlc://${playable.stream_url}`);
  });
});
