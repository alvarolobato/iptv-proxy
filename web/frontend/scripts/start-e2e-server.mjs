#!/usr/bin/env node
/**
 * Starts a minimal M3U HTTP server, then runs iptv-proxy. Used by Playwright webServer.
 * Run from repo root: node web/frontend/scripts/start-e2e-server.mjs
 */
import http from 'http';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';
import { spawn } from 'child_process';
const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../../..');
const fixturesDir = path.join(repoRoot, 'web/frontend/e2e/fixtures');
const testdataDir = path.join(repoRoot, 'web/frontend/e2e/testdata');

const m3uPort = 18765;
const uiPort = 18081;

// Ensure testdata exists and has known-good settings so proxy loads M3U and exclusions
try {
  fs.mkdirSync(testdataDir, { recursive: true });
} catch (e) {}
const settingsPath = path.join(testdataDir, 'settings.json');
const goldenSettings = {
  m3u_url: `http://127.0.0.1:${m3uPort}/test.m3u`,
  port: 18080,
  advertised_port: 18080,
  hostname: 'localhost',
  user: 'usertest',
  password: 'passwordtest',
  group_exclusions: ['^Group2$'],
  replacements: { 'global-replacements': [], 'names-replacements': [], 'groups-replacements': [] },
  ui_port: uiPort,
};
fs.writeFileSync(settingsPath, JSON.stringify(goldenSettings, null, 2), 'utf8');

const m3uServer = http.createServer((req, res) => {
  if (req.url === '/test.m3u' || req.url === '/') {
    const file = path.join(fixturesDir, 'test.m3u');
    res.setHeader('Content-Type', 'application/x-mpegurl');
    res.end(fs.readFileSync(file));
  } else {
    res.statusCode = 404;
    res.end();
  }
});

function startProxy() {
  const m3uUrl = `http://127.0.0.1:${m3uPort}/test.m3u`;
  const child = spawn('go', [
    'run', '.',
    '--m3u-url', m3uUrl,
    '--hostname', 'localhost',
    '--port', '18080',
    '--data-folder', testdataDir,
    '--ui-port', String(uiPort),
  ], {
    cwd: repoRoot,
    stdio: 'inherit',
    shell: process.platform === 'win32',
  });

  const shutdown = () => {
    child.kill();
    m3uServer.close();
    process.exit(0);
  };
  process.on('SIGINT', shutdown);
  process.on('SIGTERM', shutdown);

  child.on('exit', (code) => {
    m3uServer.close();
    process.exit(code ?? 0);
  });
}

m3uServer.listen(m3uPort, '127.0.0.1', () => {
  const m3uUrl = `http://127.0.0.1:${m3uPort}/test.m3u`;
  // Ensure M3U server is accepting before starting proxy (avoids race where proxy fetches before listen is ready)
  fetch(m3uUrl)
    .then((r) => { if (!r.ok) throw new Error(r.status); return r.text(); })
    .then((body) => { if (!body.includes('#EXTM3U')) throw new Error('Invalid M3U'); })
    .then(() => new Promise((r) => setTimeout(r, 1500)))
    .then(() => startProxy())
    .catch((e) => { console.error('[WebServer]', e.message); process.exit(1); });
});
