# Agent context: iptv-proxy2

This file gives AI agents and future sessions application context and where to find things.

**UI framework:** The configuration UI uses [Elastic UI (EUI)](https://eui.elastic.co/docs/components/). Refer to the [EUI component docs](https://eui.elastic.co/docs/components/) for patterns (tables, filter groups, buttons, etc.).

---

## What the application does

- **iptv-proxy2** is a reverse proxy for:
  1. **M3U playlists** — Rewrites track URLs to point at the proxy; serves a single proxified M3U and proxies each track stream. Supports filtering (group/channel regex), replacements (regex rewrite of names and groups), and optional resolution grouping (FHD/HD/SD).
  2. **Xtream Codes client API** — Proxies the Xtream API (live, VOD, series, EPG/xmltv.php, get.php, player_api.php, HLS, etc.) and rewrites credentials and base URL so clients use the proxy instead of the provider.

- **Tech:** Go, Cobra/Viper for CLI and config, Gin for HTTP, vendored deps (e.g. `github.com/tellytv/go.xtream-codes`, `github.com/jamesnetherton/m3u`). Module path: `github.com/alvarolobato/iptv-proxy`.

---

## Repository layout

```
cmd/root.go           # CLI entry, flags, config construction, server.NewServer + Serve
main.go               # calls cmd.Execute()
pkg/
  config/config.go    # ProxyConfig, CredentialString, HostConfiguration
  config/settings.go  # SettingsJSON, ReplacementsInSettings, ReplacementRule (settings.json shape)
  config/settings_load.go  # LoadSettings, ApplyTo, EnsureStubSettings (depends on settings.go)
  server/
    server.go         # NewServer, Serve, playlist init, marshallInto (M3U writing), replaceURL
    startup.go        # ServeWithContext, startup summary
    ui_static.go      # embedded frontend (uistatic/*), serveStaticUI
    ui.go             # UI API (/api/ready, /api/groups, /api/channels, settings), channelsProcessed
    handlers.go       # getM3U, reverseProxy, m3u8ReverseProxy, stream (HTTP proxy), xtreamStream, auth
    routes.go         # routes, xtreamRoutes, m3uRoutes
    xtreamHandles.go  # Xtream: get.php, apiget, player_api, xmltv.php, stream handlers, HLS
    cache.go          # responseCache for XMLTV
    replacements.go   # loadReplacements, applyReplacements, Replacements struct
  xtream-proxy/       # Xtream API client (GetLiveCategories, GetXMLTV, etc.)
vendor/               # Vendored deps
web/frontend/         # Configuration UI (React, Vite, EUI); build output → pkg/server/uistatic/
docs/
  configuration.md   # All parameters and options
  replacements.md    # Replacements file format
  traefik.md         # Running behind Traefik with TLS
  replacements-example.json
```

---

## Architecture (high level)

1. **Startup:** `cmd/root.go` parses flags and config, builds `config.ProxyConfig`, calls `server.NewServer(conf)`, then `server.Serve()`.
2. **M3U mode:** If `RemoteURL` (m3u-url) is set, server parses the M3U, applies optional filter/replacement (inclusions/exclusions and replacements from settings, data-folder, divide-by-res), writes a proxified M3U, and registers M3U route + per-track proxy routes.
3. **Xtream mode:** If `XtreamBaseURL` (+ credentials) is set, server registers Xtream routes: get.php, player_api.php, xmltv.php, live/movie/series stream URLs, HLS, play route. Stream requests are proxied with Range header; XMLTV can be cached and retried.
4. **Auth:** M3U and Xtream endpoints use the same `user`/`password`; auth middleware in routes.
5. **Data folder:** Use `--data-folder /data` (e.g. in Docker mount a volume at `/data`) for `replacements.json` and other data.

---

## Conventions

- **Config:** Viper binds flags and env; env key is `IPTV_PROXY_` + flag name with `-` → `_`. Config file: `~/.iptv-proxy.yaml` or path from `--iptv-proxy-config`.
- **CustomId:** Trimmed and used as `endpointAntiColision` for M3U track paths when set.
- **Filter:** Inclusions and exclusions (regex lists) from settings only. Processing order: inclusions → exclusions → replacements. Empty list = no filter. Invalid regex logs and skips that pattern.
- **Replacements:** Loaded from `{JSONFolder}/replacements.json`. See docs/replacements.md.
- **XMLTV cache:** Key is request query string (canonical). TTL from `--xmltv-cache-ttl`. Retries with backoff; on failure returns empty `<tv></tv>`.

---

## Where to look

| Need | Location |
|------|----------|
| CLI flags and config construction | `cmd/root.go` |
| M3U writing and filter/replacement | `pkg/server/server.go`, `pkg/server/replacements.go` |
| Xtream API client | `pkg/xtream-proxy/xtream-proxy.go`, `vendor/.../go.xtream-codes` |
| Xtream HTTP handlers and XMLTV | `pkg/server/xtreamHandles.go` |
| Stream proxy (Range, etc.) | `pkg/server/handlers.go` (`stream`) |
| Routes | `pkg/server/routes.go` |
| Cache | `pkg/server/cache.go` |
| User-facing configuration and options | `docs/configuration.md`, README.md |
| Replacements file | `docs/replacements.md` |
| Settings (settings.json) types and load | `pkg/config/settings.go`, `pkg/config/settings_load.go` |
| UI API and channel/group processed data | `pkg/server/ui.go` |
| Stream URL building | `pkg/server/server.go` (`replaceURL`) |
| Embedded UI and static serve | `pkg/server/ui_static.go` |

**Settings categories (UI and CLI/docs):** The Settings UI groups options as Input, Serving, Output, Xtream, Cache & EPG, Other. Use the same grouping in CLI help and documentation where possible.

---

## Testing (Playwright)

E2E tests live in `web/frontend/e2e/`. Run them with:

```bash
cd web/frontend && npm run e2e
```

The Playwright config starts the server via `webServer` (see `web/frontend/scripts/start-e2e-server.mjs`), so you don’t need to start it manually.

### Test data

- **Location:** `web/frontend/e2e/testdata/` (e.g. `settings.json`) and `web/frontend/e2e/fixtures/` (e.g. `test.m3u`).
- **Keep test data sufficient for all cases:** The data must cover what the tests assert. For example:
  - **Exclusions:** Include at least one exclusion pattern in `settings.json` (e.g. `group_exclusions: ["^Group2$"]`) so tests that filter by “Excluded” or remove an exclusion pattern have data to work with.
  - **M3U:** The fixture M3U should have groups/channels that match those patterns (e.g. a group titled `Group2` so it appears as excluded when the pattern is `^Group2$`).
- If a test skips or fails due to missing data, add or adjust the test data so the scenario is covered; don’t rely on skip when the scenario is important.

### When you change code or UI

1. **Update tests** for any change that affects behavior or UI (new buttons, new flows, renamed labels, etc.).
2. **Rebuild and re-run:** The webServer runs `go run .`, which serves the **embedded** frontend. After frontend changes, build and re-embed so e2e runs against the latest UI:
   - `cd web/frontend && npm run build`
   - Update embedded assets if the project has an embed/generate step, then run `npm run e2e`.
3. **Validate that all tests pass** with the current test data. Fix any failing or flaky tests and ensure test data stays adequate (see above).

### E2E and screenshot learnings

- **Don’t wait only for `table tbody tr`.** EUI’s `EuiBasicTable` empty state still renders a `tbody` with a single row (the “no items” message). So `waitForSelector('table tbody tr')` can resolve before any real data is loaded and screenshots will show empty tables. Always wait for **content that proves data has loaded**, e.g. text like “Group1” or both “Included” and “Excluded” inside the table body.
- **Verifying excluded items:** The backend must return an `excluded` flag on groups/channels (e.g. `/api/groups`, `/api/channels`). In the UI, wait for real rows to appear, then assert that both “Included” and “Excluded” appear in the table (or that toggling the filter shows the expected subset). Rely on API tests to assert that at least one item has `excluded: true` and one has `excluded !== true` when exclusions are configured.
- **Server readiness:** The Playwright `webServer` uses `http://localhost:18081/api/ready`. The backend returns 200 only when the playlist has been loaded (at least one track). This avoids starting tests before the proxy has fetched the M3U. The start script (`start-e2e-server.mjs`) writes golden testdata, ensures the M3U server responds, waits briefly, then starts the proxy; `reuseExistingServer: false` so the proxy always runs against the fixture M3U.

### Debugging

- Use the API from tests: `request.get('/api/groups')`, `request.get('/api/settings')`, etc.
- Take screenshots when debugging: `await page.screenshot({ path: 'screenshot.png' })` or `test.info().attach('screenshot', await page.screenshot())`.
- Check that excluded groups/channels appear when the Excluded filter is on and the backend has exclusion rules configured (and that test data provides those rules).

---

## Learnings from sessions (CI, UI, stream URLs)

### Build and CI

- **`pkg/config/settings.go` must be committed.** `settings_load.go` references `SettingsJSON`, `ReplacementsInSettings`, and `ReplacementRule` defined in `settings.go`. If `settings.go` is missing, `go build` and golangci-lint fail with "undefined: SettingsJSON" etc. on CI.

### Stream URLs and Channels table

- **Valid proxy URLs:** In `pkg/server/server.go`, `replaceURL()` builds the proxy stream URL. If `HostConfig.Hostname` is empty, the URL would be `http://:9090/...` (invalid). Use a fallback: when hostname is empty, use `"localhost"`; when `AdvertisedPort` is 0, use `HostConfig.Port`. This ensures stream URLs are always valid so "Open stream" opens the real URL and "Copy link" works.
- **Stream URL for Xtream M3U:** When the input is an Xtream-style M3U URL (`get.php?type=m3u_plus`), the server still parses the M3U and runs `marshallInto(..., false)`, so `trackIndexInPlaylist` is set. In `channelsProcessed()` (ui.go), set `stream_url` whenever `uriToIndex` is available, not only when `!xtream`, so the Channels table gets Open stream links for Xtream M3U too.

### Channels table: Open stream link (UI)

- **Use a native `<a href={streamUrl}>`** for "Open stream". Avoid using `EuiButtonEmpty` with `href` for this link: ensure the element is a plain anchor so the browser shows the URL on hover, right-click "Copy link" works, and opening in a new tab doesn’t result in `about:blank#blocked`. Set `title={streamUrl}` and tooltip content to the URL for visibility.
- **Action order:** Put the play (Open stream) action to the **right** of "Add to inclusions" and "Add to exclusions" so that when a channel has no `stream_url`, the other two buttons don’t shift and alignment stays consistent.

### EUI icon hack

- The configuration UI uses Elastic UI (EUI). Icons used in the app (e.g. `play`, `plusInCircleFilled`, `copyClipboard`) must be **registered** in `web/frontend/src/icons_hack.jsx` (import from EUI assets and add to `appendIconComponentCache`). If an icon isn’t registered, it may not render. After adding an icon, rebuild the frontend (`npm run build`) so the embedded UI in `pkg/server/uistatic/` is updated.
