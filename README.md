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

## Install and run

### Docker Compose (recommended)

No need to clone the repo. Download the Compose file, edit it, and start:

```bash
curl -sSL -o docker-compose.yml https://raw.githubusercontent.com/alvarolobato/iptv-proxy/master/docker-compose.yml
```

Edit `docker-compose.yml` and set at least:

| Variable | What to set |
|----------|-------------|
| `M3U_URL` | Your provider’s M3U URL (e.g. `http://provider.com/get.php?username=USER&password=PASS&type=m3u_plus&output=m3u8`) or path to a local file (e.g. `./iptv/playlist.m3u`). |
| `HOSTNAME` | Hostname or IP used in generated URLs (e.g. `localhost` or your server’s public hostname). |
| `USER` | Username for proxy auth (playlist and streams). |
| `PASSWORD` | Password for proxy auth. |

Optional (Xtream proxy): `XTREAM_USER`, `XTREAM_PASSWORD`, `XTREAM_BASE_URL`.

The Compose file already mounts `./data:/data` and sets `JSON_FOLDER: /data` so you can put `replacements.json` in `./data` for name/group rewriting.

Then start:

```bash
docker-compose up -d
```

Playlist URL: `http://<HOSTNAME>:8080/iptv.m3u?username=<USER>&password=<PASSWORD>`.

---

### Docker run (single container)

Use a volume for data (e.g. `replacements.json`) and set `JSON_FOLDER`:

```bash
mkdir -p ./data
docker run -d \
  --name iptv-proxy \
  -p 8080:8080 \
  -v "$(pwd)/data:/data" \
  -e M3U_URL="http://your-provider.com/get.php?username=user&password=pass&type=m3u_plus&output=m3u8" \
  -e HOSTNAME=localhost \
  -e USER=myuser \
  -e PASSWORD=mypass \
  -e JSON_FOLDER=/data \
  alobato/iptv-proxy2:latest
```

(If you don’t have a pre-built image, build from a clone: `docker build -t iptv-proxy2 .` and use that image name.)

---

### Binary (from release)

1. Download the latest [release](https://github.com/alvarolobato/iptv-proxy/releases) for your OS/arch (e.g. `iptv-proxy_linux_amd64.tar.gz`), unpack it.
2. Create a data directory and run (use it for `replacements.json` and optional cache):

   ```bash
   mkdir -p ./data
   ./iptv-proxy --m3u-url "http://provider.com/get.php?username=u&password=p&type=m3u_plus&output=m3u8" \
     --port 8080 --hostname localhost --user myuser --password mypass \
     --json-folder ./data
   ```

3. Playlist URL: `http://localhost:8080/iptv.m3u?username=myuser&password=mypass`.

**Building from source:** `go install` in the repo root (Go 1.17+). Config file and env vars: see [Configuration](#configuration).

---

## Configuration

All options can be set via **command-line flags**, a **config file** (`~/.iptv-proxy.yaml` or path from `--iptv-proxy-config`), or **environment variables** (`IPTV_PROXY_` + flag name with hyphens as underscores, e.g. `IPTV_PROXY_M3U_URL`).

### Main options (summary)

| Flag | Required? | Default | Description |
|------|------------|---------|-------------|
| `--iptv-proxy-config` | No | — | Config file path (default: `$HOME/.iptv-proxy.yaml`). |
| `--m3u-url`, `-u` | Yes (M3U) | — | M3U URL or path. For Xtream, often a get.php URL. |
| `--m3u-file-name` | No | `iptv.m3u` | Proxified playlist filename. |
| `--custom-endpoint` | No | — | Path prefix for M3U (e.g. `api` → `…/api/iptv.m3u`). |
| `--custom-id` | No | — | Anti-collision path for track URLs. |
| `--port` | No | 8080 | Listen port. |
| `--advertised-port` | No | 0 (= port) | Port in generated URLs (e.g. 443 behind reverse proxy). |
| `--hostname` | No* | — | Hostname or IP in generated URLs. *Set for correct playlist URLs. |
| `--https` | No | false | Use `https` in generated URLs. |
| `--user` | No | usertest | Proxy auth username. |
| `--password` | No | passwordtest | Proxy auth password. |
| `--xtream-user` | Yes (Xtream) | — | Xtream provider username (can be inferred from get.php URL). |
| `--xtream-password` | Yes (Xtream) | — | Xtream provider password. |
| `--xtream-base-url` | Yes (Xtream) | — | Xtream provider base URL. |
| `--xtream-api-get` | No | false | Serve get.php from Xtream API. |
| `--m3u-cache-expiration` | No | 1 | M3U cache TTL (hours). |
| `--xmltv-cache-ttl` | No | — | XMLTV cache TTL (e.g. `1h`, `30m`). Empty = no cache. |
| `--xmltv-cache-max-entries` | No | 100 | Max cached XMLTV responses. |
| `--group-regex` | No | — | Include only tracks whose `group-title` matches this regex. |
| `--channel-regex` | No | — | Include only tracks whose channel name matches this regex. |
| `--json-folder` | No | — | Folder for `replacements.json` (see [Replacements](docs/replacements.md)). |
| `--divide-by-res` | No | false | Add resolution suffix to groups (FHD/HD/SD). |
| `--debug-logging` | No | false | Verbose debug logs. |
| `--cache-folder` | No | — | Folder for saving provider responses (debug). |
| `--use-xtream-advanced-parsing` | No | false | Alternate Xtream parsing for some providers. |
| `--ui-port` | No | 0 | Port for the configuration UI (0 = disabled). See [Configuration UI](docs/ui.md). |

Full reference and examples: **[docs/configuration.md](docs/configuration.md)**.

- **Filtering and renaming:** [docs/replacements.md](docs/replacements.md) (format of `replacements.json`).
- **Configuration UI:** [docs/ui.md](docs/ui.md) (manage groups, channels, and replacements in the browser; requires `--ui-port` and `--json-folder`).
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
| [docs/release.md](docs/release.md) | How to create a release and build binaries. |
| [docs/ui.md](docs/ui.md) | Configuration UI (groups, channels, replacements). |

---

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).

---

## Credits

- Original project: [Pierre-Emmanuel Jacquier](https://github.com/pierre-emmanuelJ/iptv-proxy).
- Built with [Cobra](https://github.com/spf13/cobra), [go.xtream-codes](https://github.com/tellytv/go.xtream-codes), [Gin](https://github.com/gin-gonic/gin).
