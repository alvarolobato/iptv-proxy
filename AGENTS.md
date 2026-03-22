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
  config/config.go    # ProxyConfig, CredentialString, HostConfiguration, User struct
  config/users.go     # User helpers: FindUser, ValidateCredentials, AllUsers, ValidUsername
  config/settings.go  # SettingsJSON, ReplacementsInSettings, ReplacementRule (settings.json shape)
  config/settings_load.go  # LoadSettings, ApplyTo, EnsureStubSettings (depends on settings.go)
  server/
    server.go         # NewServer, Serve, playlist init, marshallInto (M3U writing), replaceURL
    startup.go        # ServeWithContext, startup summary
    ui_static.go      # embedded frontend (uistatic/*), serveStaticUI
    ui.go             # UI API (/api/ready, /api/groups, /api/channels, settings, users), channelsProcessed
    handlers.go       # getM3U, reverseProxy, m3u8ReverseProxy, stream (HTTP proxy), xtreamStream, auth
    routes.go         # routes, xtreamRoutes, m3uRoutes
    xtreamHandles.go  # Xtream: get.php, apiget, player_api, xmltv.php, stream handlers, HLS
    cache.go          # responseCache for XMLTV
    replacements.go   # loadReplacements, applyReplacements, Replacements struct
  stats/              # Elasticsearch stats collector (sessions, channel metrics, user history)
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
4. **Auth:** Multi-user auth — the default user comes from `--user`/`--password` (CLI flags or `user`/`password` in settings.json). Additional users are stored in the `users` array in `settings.json` and managed via the UI or `/api/users` endpoints. Auth middleware (`authenticate`, `appAuthenticate`, `authenticatePath`) validates against all users and stores the matched username in the Gin context (`ctx.Set("authenticated_user", ...)`).
5. **Data folder:** Use `--data-folder /data` (e.g. in Docker mount a volume at `/data`) for `settings.json`, `replacements.json`, and other data.

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
| User management API | `pkg/server/ui.go` (`apiListUsers`, `apiCreateUser`, etc.) |
| User helpers (find, validate, list) | `pkg/config/users.go` |
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

## Conventions and gotchas

### Go build

- **All type-definition files must be committed together.** `settings_load.go` depends on types in `settings.go`. Run `go build ./...` locally before pushing to catch missing files.

### Filtering and replacements

- **Always filter from the original unfiltered source.** `server.go` keeps `fullPlaylistTracks` — never mutate or discard it. When settings change, re-filter from the full list, not from a previously filtered copy.
- **Filtering must apply to all code paths.** Both M3U (`marshallInto`) and Xtream (`filterXtreamResponse`, `applyXtreamReplacements`) must run the same inclusion/exclusion/replacement logic. When adding or changing filter behavior, verify it works in both paths.

### Stream URLs

- **`replaceURL()` must always produce a valid URL.** When `HostConfig.Hostname` is empty, fall back to `"localhost"`; when `AdvertisedPort` is 0, use `HostConfig.Port`.
- **Xtream M3U also gets stream URLs.** In `channelsProcessed()` (ui.go), set `stream_url` whenever `uriToIndex` is available, not only when `!xtream`.

### UI (EUI / React)

- **Register every EUI icon** in `web/frontend/src/icons_hack.jsx` via `appendIconComponentCache`. Unregistered icons render as empty. Rebuild the frontend after adding icons.
- **Use native `<a href>` for stream links**, not `EuiButtonEmpty` with `href` (which causes `about:blank#blocked`). Set `title` to the URL for hover visibility.
- **Action order in tables:** Put "Open stream" to the right of filter actions so missing stream URLs don’t shift button alignment.

### Multi-user and auth

- **Xtream route conflicts.** Gin does not allow conflicting wildcards at the same path segment. When converting literal user/pass to `:user/:password` params, routes like `/play/:token/:type` and `/play/:user/:password/:id` will panic. Use a catch-all dispatcher (`/play/*path`) for conflicting patterns, similar to the existing `/hls/*path` dispatcher. Routes under unique fixed prefixes (`/live/`, `/movie/`, `/series/`, `/timeshift/`) can safely use `:user/:password` params. Always add a `TestXtreamRouteRegistration` test to catch these panics.
- **Route params, not literals.** Stream routes use `:user/:password` path params (not hardcoded user/pass literals). Auth is validated by `authenticatePath` middleware. The M3U file is generated with the default user’s credentials and rewritten per-user in `getM3U`.
- **Credential escaping.** When building proxy URLs in `replaceURL`, always use `url.PathEscape` for credentials. When rewriting credentials in `getM3U`, use the same `url.PathEscape` so the search string matches what was written. Mismatch between generation and rewrite will silently fail for passwords with special characters.
- **`Config` struct contains `sync.RWMutex`.** Never copy `Config` by value (`tmp := *c`) — this copies the mutex and triggers the `govet` copylocks lint error. Instead, construct a new `&Config{...}` with the fields you need. This applies to `cacheXtreamM3u` and any similar pattern.
- **Per-track `Config` in routes.** When creating `trackConfig` for per-track handlers, copy `statsCollector` from the parent config. Without it, `streamWithStats` will panic on nil interface dereference.
- **`writeSettingsFile` and users.** The settings API (`PUT /api/settings`) and user API (`/api/users`) both write to `settings.json`. The settings form doesn’t include users, so `writeSettingsFile` must preserve users from in-memory state to avoid wiping them. Users are managed via their own API endpoints.
- **E2E golden settings.** The E2E startup script (`start-e2e-server.mjs`) overwrites `settings.json` with `goldenSettings` on every run. Any test data (including `users`) must be in `goldenSettings`, not just in the testdata JSON file.
- **Constant-time comparison.** For `ValidateCredentials`, evaluate both username and password comparisons into variables before the conditional branch. `&&` short-circuits, which can leak whether the username matched via timing differences.
- **CLI user migration.** The CLI `--user`/`--password` is migrated into the `Users` slice at startup via `MigrateDefaultUser()`. All user operations (CRUD, auth, watch) go through the `Users` slice only — no dual-path logic for "default" vs "additional" users. This keeps the code simple and ensures all users are treated equally in the UI.

### Elasticsearch

- **Only use the `metrics-` index prefix for TSDB data streams.** ES serverless rejects non-TSDB indices with that prefix. Current layout: `metrics-iptv.channel_metrics` (TSDB), `iptv.sessions` and `iptv.user_history` (regular).

---

## Documentation maintenance

### Decision log

When a change picks one approach over another, introduces a dependency, changes the data model, or modifies the public API, add an entry to [`docs/design/DECISIONS.md`](docs/design/DECISIONS.md). Use the lightweight ADR format already in that file.

### When you discover a gotcha

If you hit a non-obvious problem or discover an undocumented constraint, **update the "Conventions and gotchas" section above** with the preventive rule — not a log of the problem, just the guidance to avoid it. Keep entries short and imperative.

### Documentation updates required

| Change type | Update |
|---|---|
| New feature or flag | `docs/configuration.md`, `DECISIONS.md` entry |
| Architecture change | `AGENTS.md` (architecture section), `DECISIONS.md` entry |
| Bug fix with non-obvious cause | `AGENTS.md` conventions section (add the preventive rule) |
| UI change | `AGENTS.md` conventions section if gotcha found |
| New dependency | `DECISIONS.md` entry |
