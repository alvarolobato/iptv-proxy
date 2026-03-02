# Features and configuration

This document describes optional features and behaviours introduced by the consolidated fork PRs (see [MERGE_PLAN.md](MERGE_PLAN.md)). Flags can be passed on the command line, set in config file (e.g. `~/.iptv-proxy.yaml`), or via environment variables (Cobra/Viper: `--flag-name` → `IPTV_PROXY_FLAG_NAME`).

---

## Core flags (existing)

| Flag | Default | Description |
|------|---------|-------------|
| `--m3u-url`, `-u` | (required for M3U) | M3U file URL or path |
| `--m3u-file-name` | `iptv.m3u` | Filename of the proxified M3U |
| `--custom-endpoint` | | Path prefix for M3U endpoint |
| `--custom-id` | | Anti-collision path segment for track URLs |
| `--port` | 8080 | Listen port |
| `--advertised-port` | 0 (= port) | Port used in generated URLs (e.g. behind reverse proxy) |
| `--hostname` | | Hostname or IP in generated URLs |
| `--https` | false | Use https in generated URLs |
| `--user` | usertest | Proxy auth username |
| `--password` | passwordtest | Proxy auth password |
| `--xtream-user`, `--xtream-password`, `--xtream-base-url` | | Xtream Codes upstream credentials |
| `--m3u-cache-expiration` | 1 | M3U cache expiration (hours) |
| `--xtream-api-get` | false | Serve get.php from Xtream API instead of upstream get.php |

---

## New features (from consolidated PRs)

### 1. Xtream robustness and HLS (PR #10)

**Behaviour (no new flags):**

- **UnmarshalJSON:** Provider JSON that sends numbers as strings or omits nested `info` is handled; optional `*Info` types for VOD/Series reduce crashes on incomplete data.
- **HLS:** Token is sent as query parameter (`?token=`) for providers that expect it there; URL format is `.../hls/{chunk}?token={token}`.
- **Startup:** Log line `Server is ready and listening on :port` after routes are registered.

---

### 2. M3U patch parsing (PR #11)

**Behaviour (no new flags):**

- **Display name:** For tracks whose name is `dpr_auto`, `h_256`, or contains `320"`, the displayed name in the M3U is taken from the `tvg-name` tag instead of the track name.
- **tvg-logo:** If a `tvg-logo` tag value contains a comma (malformed), it is cleared and a log line is written.

---

### 3. M3U filter, replacement, and resolution groups (PR #12)

| Flag | Default | Description |
|------|---------|-------------|
| `--group-regex` | (empty) | Include only tracks whose `group-title` matches this regex. Empty = all. |
| `--channel-regex` | (empty) | Include only tracks whose channel name matches this regex. Empty = all. |
| `--json-folder` | (empty) | Folder containing `replacements.json` for name/group replacement rules. |
| `--divide-by-res` | false | Add resolution suffix to group titles (FHD/HD/SD) and strip resolution from channel names. |

**replacements.json** (optional, in `--json-folder`):

- `global-replacements`: array of `{ "replace": "regex", "with": "replacement" }` applied to all relevant text.
- `names-replacements`: applied to channel names.
- `groups-replacements`: applied to group-title values.

Example:

```json
{
  "global-replacements": [ { "replace": "\\s+", "with": " " } ],
  "names-replacements": [ { "replace": "^ES ", "with": "ES: " } ],
  "groups-replacements": [ { "replace": "M\\+", "with": "MOVISTAR+" } ]
}
```

See `docs/replacements-example.json` in the repo for a full example (when PR #12 is merged).

---

### 4. XMLTV cache and Range header (PR #13)

| Flag | Default | Description |
|------|---------|-------------|
| `--xmltv-cache-ttl` | (empty) | XMLTV response cache TTL (e.g. `1h`, `30m`). Empty = no cache. |

**Behaviour:**

- **XMLTV:** Responses for `xmltv.php` are cached by request query when TTL is set. On upstream failure, the handler retries up to 3 times; if all fail, it returns an empty `<tv></tv>` document instead of 5xx.
- **Streaming:** The `Range` request header is forwarded to the upstream stream; useful for seeking (partial content).

---

### 5. Debug and advanced parsing flags (PR #15)

| Flag | Default | Description |
|------|---------|-------------|
| `--debug-logging` | false | Enable debug logging (when wired). |
| `--cache-folder` | (empty) | Folder for saving provider/client responses (when wired). |
| `--use-xtream-advanced-parsing` | false | Use alternate Xtream parsing to preserve raw provider response (when wired). |

**Note:** These flags are in config only; full wiring (file saving, advanced parsing code path) may follow in a later PR.

---

### 6. Play route for clients (PR #14)

**Behaviour (no new flags):**

- **Route:** `GET /play/:user/:password/:id` is added and handled like `/:user/:password/:id` (same stream handler). Some clients expect video URLs to include `play` in the path.

---

## Environment variables

All flags are exposed as environment variables by Viper: replace `--` with `_` and uppercase with `IPTV_PROXY_` prefix. Examples:

- `IPTV_PROXY_GROUP_REGEX`
- `IPTV_PROXY_XMLTV_CACHE_TTL`
- `IPTV_PROXY_DEBUG_LOGGING`

Config file keys use the same names as flags (with Viper’s default mapping).
