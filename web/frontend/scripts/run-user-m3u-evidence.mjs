#!/usr/bin/env node
/**
 * Starts iptv-proxy with a real M3U URL from env USER_M3U_URL,
 * waits for /api/ready, runs the evidence Playwright test, then stops the proxy.
 * Run from repo root (set USER_M3U_URL in env or GitHub Actions secrets):
 *   USER_M3U_URL="http://..." node web/frontend/scripts/run-user-m3u-evidence.mjs
 *
 * Evidence is written to web/frontend/e2e/evidence/
 */
import { spawn } from 'child_process';
import path from 'path';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));
const repoRoot = path.resolve(__dirname, '../../..');
const uiPort = 18081;
const readyUrl = `http://localhost:${uiPort}/api/ready`;

const USER_M3U_URL = process.env.USER_M3U_URL;
if (!USER_M3U_URL || USER_M3U_URL.trim() === '') {
  console.error('[run-user-m3u-evidence] USER_M3U_URL env var is required. Set it in GitHub Actions secrets or locally.');
  process.exit(1);
}

async function waitForReady(timeoutMs = 120000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const r = await fetch(readyUrl);
      if (r.ok) return true;
    } catch (_) {}
    await new Promise((r) => setTimeout(r, 2000));
  }
  return false;
}

async function main() {
  console.log('[run-user-m3u-evidence] Starting proxy with provided M3U URL...');
  const proxy = spawn(
    'go',
    [
      'run', '.',
      '--m3u-url', USER_M3U_URL,
      '--hostname', 'localhost',
      '--port', '18080',
      '--ui-port', String(uiPort),
    ],
    {
      cwd: repoRoot,
      stdio: ['ignore', 'pipe', 'pipe'],
      shell: process.platform === 'win32',
    }
  );

  let stderr = '';
  proxy.stderr.on('data', (d) => {
    stderr += d;
    process.stderr.write(d);
  });
  proxy.stdout.on('data', (d) => process.stdout.write(d));

  const cleanup = () => {
    try {
      proxy.kill('SIGTERM');
    } catch (_) {}
  };
  process.on('SIGINT', () => { cleanup(); process.exit(130); });
  process.on('SIGTERM', () => { cleanup(); process.exit(0); });

  const ready = await waitForReady();
  if (!ready) {
    console.error('[run-user-m3u-evidence] Proxy did not become ready in time. Stderr:', stderr.slice(-2000));
    cleanup();
    process.exit(1);
  }

  console.log('[run-user-m3u-evidence] Proxy ready. Running evidence test...');
  const { execSync } = await import('child_process');
  try {
    execSync(
      `npx playwright test evidence-channel-link.spec.js --config=playwright.user-m3u.config.cjs --project=chromium`,
      {
        cwd: path.join(repoRoot, 'web/frontend'),
        stdio: 'inherit',
        env: { ...process.env },
      }
    );
  } catch (e) {
    cleanup();
    process.exit(e.status ?? 1);
  }

  cleanup();
  console.log('[run-user-m3u-evidence] Done. Evidence in web/frontend/e2e/evidence/');
}

main();
