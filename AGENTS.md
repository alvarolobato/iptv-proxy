# Agent context: iptv-proxy

This file gives AI agents and future sessions the application context, architecture, and decisions learned during the fork-consolidation work.

---

## What the application does

- **iptv-proxy** is a reverse proxy for:
  1. **M3U playlists** – Rewrites track URLs to point at the proxy; serves a single proxified M3U and proxies each track stream.
  2. **Xtream Codes client API** – Proxies the Xtream API (live, VOD, series, EPG/xmltv.php, get.php, player_api.php, HLS, etc.) and rewrites credentials and base URL so clients use the proxy instead of the provider.

- **Tech:** Go, Cobra/Viper for CLI and config, Gin for HTTP, vendored deps (e.g. `github.com/tellytv/go.xtream-codes`, `github.com/jamesnetherton/m3u`).
- **Module path:** `github.com/pierre-emmanuelJ/iptv-proxy` (upstream); this repo may be a fork (e.g. `alvarolobato/iptv-proxy`).

---

## Repository layout

```
cmd/root.go           # CLI entry, flags, config construction, server.NewServer + Serve
main.go               # calls cmd.Execute()
pkg/
  config/config.go    # ProxyConfig, CredentialString, HostConfiguration; optional globals (DebugLoggingEnabled, CacheFolder)
  server/
    server.go         # NewServer, Serve, playlist init, marshallInto (M3U writing), replaceURL
    handlers.go       # getM3U, reverseProxy, m3u8ReverseProxy, stream (HTTP proxy), xtreamStream, auth
    routes.go         # routes, xtreamRoutes, m3uRoutes
    xtreamHandles.go  # Xtream-specific: get.php, apiget, player_api, xmltv.php, stream handlers, HLS
    cache.go          # (PR #13) responseCache for XMLTV/metadata
    replacements.go   # (PR #12) loadReplacements, applyReplacements, Replacements struct
  xtream-proxy/       # Xtream API client (GetLiveCategories, GetXMLTV, etc.)
vendor/               # Vendored deps including go.xtream-codes, m3u, gin, cobra, viper
docs/
  FEATURES.md         # User-facing features and flags
  MERGE_PLAN.md      # Fork merge plan, PR list, “not bringing in”
  FORK_BRANCHES_COMPARISON.md  # Comparison of fork branches and overlap
  replacements-example.json   # (PR #12 branch) Example replacements JSON
```

---

## Architecture (high level)

1. **Startup:** `cmd/root.go` parses flags and config, builds `config.ProxyConfig`, calls `server.NewServer(conf)`, then `server.Serve()`.
2. **M3U mode:** If `RemoteURL` (m3u-url) is set, server parses the M3U, writes a proxified M3U to a temp file (with optional filter/replacement/patches from PRs #11, #12), and registers M3U route + per-track proxy routes.
3. **Xtream mode:** If `XtreamBaseURL` (+ credentials) is set, server registers Xtream routes: get.php, player_api.php, xmltv.php, live/movie/series stream URLs, HLS, etc. Stream requests are proxied via `handlers.stream()` (Range header forwarded in PR #13); XMLTV can be cached and retried (PR #13).
4. **Auth:** M3U and Xtream endpoints use the same `user`/`password`; auth middleware in routes.
5. **Vendor:** `go.xtream-codes` structs live in `vendor/`; PR #10 adds UnmarshalJSON and optional `*Info` types there for provider compatibility.

---

## Fork consolidation (session summary)

- **Goal:** Bring in features from fork branches (Gibby, Yagoor, chernandezweb, jtdevops, michbeck100, ridgarou) without CI and without duplicate/conflicting implementations.
- **Process:** For each feature area we chose one implementation, fixed issues noted in reviews, and opened new PRs (one commit per logical change) from branches `pr/1-*` … `pr/6-*`.
- **PRs created (all base: master):**
  - **#10** pr/1-xtream-struct-vod-epg-hls: Yagoor – UnmarshalJSON, HLS token query, VOD/EPG fixes, startup log.
  - **#11** pr/2-m3u-patch-parsing: Gibby – tvg-name/tvg-logo M3U patches (with `name` and log fixes).
  - **#12** pr/3-regex-filter-replacement-resolution: ridgarou – group/channel regex filter, replacements.json, divide-by-resolution.
  - **#13** pr/4-xmltv-cache-range: chernandezweb – cache.go, XMLTV cache + retry + empty on failure, Range header in stream.
  - **#15** pr/5-debug-advanced-m3u4u: jtdevops – flags only (debug-logging, cache-folder, use-xtream-advanced-parsing; default false).
  - **#14** pr/6-get-php-play-fix: ridgarou – route `GET /play/:user/:password/:id` for client compatibility.
- **Not brought in:** Any CI; module path or author changes; README/docker overhaul; full response-saving to files; m3u4u URL option; 509 body persistence; error_utils (deferred).
- **Docs:** `docs/MERGE_PLAN.md`, `docs/FORK_BRANCHES_COMPARISON.md`, `docs/FEATURES.md` (features and flags). New features/behaviours are documented in FEATURES.md; README points to it.

---

## Conventions and gotchas

- **Config:** Viper binds flags and env; env key is `IPTV_PROXY_` + flag name with `-` → `_`. Config file: `~/.iptv-proxy.yaml` or path from `--iptv-proxy-config`.
- **CustomId:** In server, `config.CustomId` (trimmed) is used as `endpointAntiColision` for M3U track paths when set.
- **Regex filter (PR #12):** Empty `GroupRegex` or `ChannelRegex` means “match all”; regexes are compiled once when non-empty. Replacements are loaded from `{JSONFolder}/replacements.json`.
- **XMLTV cache (PR #13):** Key is request query string; TTL is from `--xmltv-cache-ttl` (e.g. `1h`). Retries 3 times; on failure returns empty `<tv></tv>`.
- **Worktrees:** Feature PRs were built in separate worktrees (`../iptv-proxy-pr1` … `../iptv-proxy-pr6`) to avoid mixing changes.

---

## Where to look for what

| Need | Location |
|------|----------|
| CLI flags and config construction | `cmd/root.go` |
| M3U writing and filter/replacement/patch | `pkg/server/server.go`, `pkg/server/replacements.go` |
| Xtream API client | `pkg/xtream-proxy/xtream-proxy.go`, `vendor/.../go.xtream-codes` |
| Xtream HTTP handlers and XMLTV | `pkg/server/xtreamHandles.go` |
| Stream proxy (Range, etc.) | `pkg/server/handlers.go` (`stream`) |
| Routes | `pkg/server/routes.go` |
| Cache implementation | `pkg/server/cache.go` |
| User-facing feature list and flags | `docs/FEATURES.md` |
| Merge plan and “not bringing in” | `docs/MERGE_PLAN.md`, `docs/FORK_BRANCHES_COMPARISON.md` |

---

## For next time

- When adding a new feature, add it to `docs/FEATURES.md` (and README if it’s a major option).
- When touching vendor (e.g. go.xtream-codes), consider upstreaming or documenting in MERGE_PLAN/FORK_BRANCHES_COMPARISON if it’s a fork-specific patch.
- PRs #10–#15 are the canonical feature PRs; the original fork-branch PRs (#1–#8) are reference only and should not be merged as-is (they contain CI, path changes, or duplicates).
