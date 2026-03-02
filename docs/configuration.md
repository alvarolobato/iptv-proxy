# Configuration reference

All options can be set by:

- **Command-line flags:** `--flag-name value`
- **Config file:** `~/.iptv-proxy.yaml` (or path from `--iptv-proxy-config`). Keys match flag names (e.g. `m3u-url`, `port`).
- **Environment variables:** `IPTV_PROXY_` + flag name with hyphens replaced by underscores, e.g. `IPTV_PROXY_M3U_URL`, `IPTV_PROXY_GROUP_REGEX`.

---

## Core options

### M3U source and output

| Flag | Default | Description |
|------|---------|-------------|
| `--m3u-url`, `-u` | (none) | URL or path to the M3U playlist. Required for M3U mode. For Xtream providers this is often a get.php URL (e.g. `http://provider/get.php?username=...&password=...&type=m3u_plus&output=m3u8`). |
| `--m3u-file-name` | `iptv.m3u` | Name of the proxified M3U file. The playlist will be available at `http://host:port/<custom-endpoint>/<m3u-file-name>`. |
| `--custom-endpoint` | (none) | Optional path prefix for the M3U endpoint (e.g. `api` → `http://host:port/api/iptv.m3u`). |
| `--custom-id` | (none) | Anti-collision path segment used in per-track proxy URLs. Useful when running multiple proxies. |

### Network and auth

| Flag | Default | Description |
|------|---------|-------------|
| `--port` | 8080 | Port the server listens on. |
| `--advertised-port` | 0 | Port used in generated playlist and API URLs. Defaults to `port`; set when behind a reverse proxy (e.g. 443). |
| `--hostname` | (none) | Hostname or IP used in generated URLs. Set to your public hostname or IP. |
| `--https` | false | If true, generated URLs use `https` instead of `http`. |
| `--user` | usertest | Username for proxy auth (M3U and Xtream endpoints). |
| `--password` | passwordtest | Password for proxy auth. |

### Xtream Codes upstream

| Flag | Default | Description |
|------|---------|-------------|
| `--xtream-user` | (none) | Provider Xtream username. Can be auto-detected from get.php URL if not set. |
| `--xtream-password` | (none) | Provider Xtream password. |
| `--xtream-base-url` | (none) | Provider base URL (e.g. `http://provider.example.com:8080`). |
| `--xtream-api-get` | false | If true, serve get.php using the Xtream API instead of the provider’s get.php endpoint. |
| `--m3u-cache-expiration` | 1 | M3U cache expiration in hours (Xtream-generated M3U). |

---

## M3U filtering and rewriting

| Flag | Default | Description |
|------|---------|-------------|
| `--group-regex` | (none) | Include only tracks whose `group-title` tag matches this regex. Empty = all. |
| `--channel-regex` | (none) | Include only tracks whose channel name matches this regex. Empty = all. |
| `--json-folder` | (none) | Folder containing `replacements.json` for name/group replacement rules. Recommended: `/data` when using Docker; mount a volume and set `--json-folder /data`. |
| `--divide-by-res` | false | Add resolution suffix to group titles (FHD / HD / SD) and strip resolution from channel names. |

See [replacements.md](replacements.md) for the replacements file format.

---

## XMLTV and cache

| Flag | Default | Description |
|------|---------|-------------|
| `--xmltv-cache-ttl` | (none) | TTL for cached XMLTV (EPG) responses (e.g. `1h`, `30m`). Empty = no cache. |
| `--xmltv-cache-max-entries` | 100 | Maximum number of cached XMLTV responses; oldest are evicted when full. |

---

## Debug and advanced

| Flag | Default | Description |
|------|---------|-------------|
| `--debug-logging` | false | Enable verbose debug logging. |
| `--cache-folder` | (none) | Folder for saving provider/client responses when debug or advanced features are used. |
| `--use-xtream-advanced-parsing` | false | Use alternate Xtream response parsing for better compatibility with some providers. |

---

## Config file example

```yaml
m3u-url: "http://example.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8"
port: 8080
hostname: iptv.example.com
advertised-port: 443
https: true
user: myuser
password: mypass
json-folder: /data
xmltv-cache-ttl: 1h
```
