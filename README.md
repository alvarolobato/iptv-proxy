# IPTV-Proxy

[![CI](https://github.com/alvarolobato/iptv-proxy/workflows/CI/badge.svg)](https://github.com/alvarolobato/iptv-proxy/actions?query=workflow%3ACI)

A reverse proxy for IPTV M3U playlists and **Xtream Codes** API. Expose your provider’s streams under your own URLs, with optional filtering, replacements, and HTTPS.

---

## What is IPTV-Proxy?

IPTV-Proxy sits between your IPTV provider and your apps (players, Plex, Jellyfin, etc.). It fetches the provider’s M3U playlist or Xtream API and **rewrites all URLs** to point to the proxy. Viewers then connect to your server instead of the provider, so you can:

- Use a single public URL and change providers behind the scenes
- Add basic auth to protect playlists and streams
- Put the proxy behind HTTPS (e.g. with Traefik)
- Filter channels by name or group (regex) and rewrite names/groups via a JSON file
- Cache M3U and XMLTV (EPG) to reduce load on the provider

It supports **M3U/M3U8** (plain playlists) and full **Xtream Codes** (live, VOD, series, EPG). No database required; configuration is flags, config file, or environment variables.

---

## What can you do with it?

| Use case | What IPTV-Proxy does |
|----------|----------------------|
| **M3U proxy** | Fetches a remote or local M3U, rewrites track URLs to your host/port, serves the playlist at e.g. `http://yourserver:8080/iptv.m3u` with optional auth. |
| **Xtream proxy** | Proxies the Xtream API (live, VOD, series, EPG). You give clients your URL and proxy credentials; they use the same apps as with a normal Xtream server. |
| **Filter channels** | Keep only channels or groups matching regex (`--group-regex`, `--channel-regex`). |
| **Rename channels/groups** | Apply find/replace rules from a `replacements.json` file (see [Replacements](docs/replacements.md)). |
| **HTTPS** | Run behind Traefik (or another reverse proxy) and set `--https` and `--advertised-port` so generated URLs use `https`. See [TLS with Traefik](docs/traefik.md). |
| **EPG cache** | Cache XMLTV (EPG) responses with `--xmltv-cache-ttl` to avoid hammering the provider. |

---

## Quick start

### Docker (recommended)

**Option A — Docker Compose (good for a persistent setup)**

1. Clone the repo and go to its directory:
   ```bash
   git clone https://github.com/alvarolobato/iptv-proxy.git
   cd iptv-proxy
   ```
2. Edit `docker-compose.yml`: set `M3U_URL` (or use a local file in `./iptv/`), `HOSTNAME`, `USER`, `PASSWORD`, and optionally Xtream vars (`XTREAM_USER`, `XTREAM_PASSWORD`, `XTREAM_BASE_URL`).
3. Start:
   ```bash
   docker-compose up -d
   ```
4. Open `http://<HOSTNAME>:8080/iptv.m3u?username=<USER>&password=<PASSWORD>` (or the port you mapped).

**Option B — Single `docker run`**

```bash
docker run -d \
  --name iptv-proxy \
  -p 8080:8080 \
  -e M3U_URL="http://your-provider.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8" \
  -e HOSTNAME=localhost \
  -e USER=myuser \
  -e PASSWORD=mypass \
  alvarolobato/iptv-proxy:latest
```

If you don’t have a pre-built image, build and run from the repo root:

```bash
docker build -t iptv-proxy .
docker run -d --name iptv-proxy -p 8080:8080 \
  -e M3U_URL="http://..." -e HOSTNAME=localhost -e USER=myuser -e PASSWORD=mypass \
  iptv-proxy
```

---

### Binary (from release)

1. Download the latest [release](https://github.com/alvarolobato/iptv-proxy/releases) for your OS/arch (e.g. `iptv-proxy_linux_amd64.tar.gz`).
2. Unpack and run, for example:
   ```bash
   ./iptv-proxy --m3u-url "http://provider.com/get.php?username=u&password=p&type=m3u_plus&output=m3u8" \
     --port 8080 --hostname localhost --user myuser --password mypass
   ```
3. Playlist URL: `http://localhost:8080/iptv.m3u?username=myuser&password=mypass`.

**Building from source:** `go install` in the repo root (requires Go 1.17+). You can also use a config file (e.g. `~/.iptv-proxy.yaml`) or environment variables; see [Configuration](#configuration).

---

## Configuration

All options can be set via **command-line flags**, a **config file** (`~/.iptv-proxy.yaml` or path from `--iptv-proxy-config`), or **environment variables** (`IPTV_PROXY_` + flag name with hyphens as underscores, e.g. `IPTV_PROXY_M3U_URL`).

### Main options (summary)

| Flag | Default | Description |
|------|---------|-------------|
| `--iptv-proxy-config` | — | Config file path (default: `$HOME/.iptv-proxy.yaml`). |
| `--m3u-url`, `-u` | — | M3U URL or path (required for M3U). For Xtream, often a get.php URL. |
| `--m3u-file-name` | `iptv.m3u` | Proxified playlist filename. |
| `--custom-endpoint` | — | Path prefix for M3U (e.g. `api` → `…/api/iptv.m3u`). |
| `--custom-id` | — | Anti-collision path for track URLs. |
| `--port` | 8080 | Listen port. |
| `--advertised-port` | 0 (= port) | Port in generated URLs (e.g. 443 behind reverse proxy). |
| `--hostname` | — | Hostname or IP in generated URLs. |
| `--https` | false | Use `https` in generated URLs. |
| `--user` | usertest | Proxy auth username. |
| `--password` | passwordtest | Proxy auth password. |
| `--xtream-user` | — | Xtream provider username (can be inferred from get.php URL). |
| `--xtream-password` | — | Xtream provider password. |
| `--xtream-base-url` | — | Xtream provider base URL. |
| `--xtream-api-get` | false | Serve get.php from Xtream API. |
| `--m3u-cache-expiration` | 1 | M3U cache TTL (hours). |
| `--xmltv-cache-ttl` | — | XMLTV cache TTL (e.g. `1h`, `30m`). Empty = no cache. |
| `--xmltv-cache-max-entries` | 100 | Max cached XMLTV responses. |
| `--group-regex` | — | Include only tracks whose `group-title` matches this regex. |
| `--channel-regex` | — | Include only tracks whose channel name matches this regex. |
| `--json-folder` | — | Folder containing `replacements.json` (see [Replacements](docs/replacements.md)). |
| `--divide-by-res` | false | Add resolution suffix to groups (FHD/HD/SD). |
| `--debug-logging` | false | Verbose debug logs. |
| `--cache-folder` | — | Folder for saving provider responses (debug). |
| `--use-xtream-advanced-parsing` | false | Alternate Xtream parsing for some providers. |

Full reference and examples: **[docs/configuration.md](docs/configuration.md)**.

- **Filtering and renaming:** [docs/replacements.md](docs/replacements.md) (format of `replacements.json`).
- **HTTPS / TLS behind Traefik:** [docs/traefik.md](docs/traefik.md).

---

## TLS (HTTPS)

To serve over HTTPS, run IPTV-Proxy behind a reverse proxy (e.g. Traefik). Set `--https`, `--advertised-port` (e.g. 443), and `--hostname` to your domain. Step-by-step with Traefik: **[docs/traefik.md](docs/traefik.md)**.

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/configuration.md](docs/configuration.md) | Full configuration reference and config file example. |
| [docs/replacements.md](docs/replacements.md) | M3U name/group replacement rules (`replacements.json`). |
| [docs/traefik.md](docs/traefik.md) | TLS/HTTPS with Traefik. |

---

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).

---

## Credits

- Original project: [Pierre-Emmanuel Jacquier](https://github.com/pierre-emmanuelJ/iptv-proxy).
- Built with [Cobra](https://github.com/spf13/cobra), [go.xtream-codes](https://github.com/tellytv/go.xtream-codes), [Gin](https://github.com/gin-gonic/gin).
