# Agent context: iptv-proxy2

This file gives AI agents and future sessions application context and where to find things.

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
  server/
    server.go         # NewServer, Serve, playlist init, marshallInto (M3U writing), replaceURL
    handlers.go       # getM3U, reverseProxy, m3u8ReverseProxy, stream (HTTP proxy), xtreamStream, auth
    routes.go         # routes, xtreamRoutes, m3uRoutes
    xtreamHandles.go  # Xtream: get.php, apiget, player_api, xmltv.php, stream handlers, HLS
    cache.go          # responseCache for XMLTV
    replacements.go   # loadReplacements, applyReplacements, Replacements struct
  xtream-proxy/       # Xtream API client (GetLiveCategories, GetXMLTV, etc.)
vendor/               # Vendored deps
docs/
  configuration.md   # All parameters and options
  replacements.md    # Replacements file format
  traefik.md         # Running behind Traefik with TLS
  replacements-example.json
```

---

## Architecture (high level)

1. **Startup:** `cmd/root.go` parses flags and config, builds `config.ProxyConfig`, calls `server.NewServer(conf)`, then `server.Serve()`.
2. **M3U mode:** If `RemoteURL` (m3u-url) is set, server parses the M3U, applies optional filter/replacement (group-regex, channel-regex, json-folder, divide-by-res), writes a proxified M3U, and registers M3U route + per-track proxy routes.
3. **Xtream mode:** If `XtreamBaseURL` (+ credentials) is set, server registers Xtream routes: get.php, player_api.php, xmltv.php, live/movie/series stream URLs, HLS, play route. Stream requests are proxied with Range header; XMLTV can be cached and retried.
4. **Auth:** M3U and Xtream endpoints use the same `user`/`password`; auth middleware in routes.
5. **Data folder:** Use `--json-folder /data` (e.g. in Docker mount a volume at `/data`) for `replacements.json` and other data.

---

## Conventions

- **Config:** Viper binds flags and env; env key is `IPTV_PROXY_` + flag name with `-` → `_`. Config file: `~/.iptv-proxy.yaml` or path from `--iptv-proxy-config`.
- **CustomId:** Trimmed and used as `endpointAntiColision` for M3U track paths when set.
- **Regex filter:** Empty `GroupRegex` or `ChannelRegex` means match all; regexes are compiled once. Invalid regex logs and disables that filter.
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
