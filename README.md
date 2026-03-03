# IPTV-Proxy

<p align="center"><img src="images/logo-128.png" width="128" alt="IPTV-Proxy logo" /></p>

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
| **Filter channels** | Use **inclusions** and **exclusions** (regex lists) in settings.json or the Processing tab in the UI. |
| **Rename channels/groups** | Apply find/replace rules from **settings.json** (or legacy `replacements.json`). See [Replacements](docs/replacements.md) and [Settings](docs/settings.md). |
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

The Compose file already mounts `./data:/data` and sets `DATA_FOLDER: /data` so you can put **settings.json** (and replacement rules) in `./data`.

Then start:

```bash
docker-compose up -d
```

Playlist URL: `http://<HOSTNAME>:8080/iptv.m3u?username=<USER>&password=<PASSWORD>`.

---

### Docker run (single container)

Use a volume for data (e.g. **settings.json**) and set `DATA_FOLDER`:

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
  -e DATA_FOLDER=/data \
  alobato/iptv-proxy2:latest
```

(If you don’t have a pre-built image, build from a clone: `docker build -t iptv-proxy2 .` and use that image name.)

---

### Binary (from release)

1. Download the latest [release](https://github.com/alvarolobato/iptv-proxy/releases) for your OS/arch (e.g. `iptv-proxy_linux_amd64.tar.gz`), unpack it.
2. Create a data directory and run (use it for **settings.json** and optional cache):

   ```bash
   mkdir -p ./data
   ./iptv-proxy --m3u-url "http://provider.com/get.php?username=u&password=p&type=m3u_plus&output=m3u8" \
     --port 8080 --hostname localhost --user myuser --password mypass \
     --data-folder ./data
   ```

3. Playlist URL: `http://localhost:8080/iptv.m3u?username=myuser&password=mypass`.

---

## Build from source

**To build the binary:** from the repo root, run:

```bash
./scripts/build.sh
```

This produces **`./iptv-proxy`** in the repo root. You need **Go 1.17+** and **Node.js 18+** (the configuration UI is embedded in the binary).

**Manual build:**

```bash
# 1. Build the configuration UI (required; embeds into the binary)
cd web/frontend && npm ci && npm run build && cd ../..

# 2. Build the binary
go build -o iptv-proxy .
```

To install into `$GOPATH/bin`: after step 1, run `go install .`. If the UI is already built and unchanged, you can run only `go build -o iptv-proxy .`.

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
| `--user` | No | (empty) | Proxy auth username; set via flag, env, or Settings UI. |
| `--password` | No | (empty) | Proxy auth password; set via flag, env, or Settings UI. |
| `--xtream-user` | Yes (Xtream) | — | Xtream provider username (can be inferred from get.php URL). |
| `--xtream-password` | Yes (Xtream) | — | Xtream provider password. |
| `--xtream-base-url` | Yes (Xtream) | — | Xtream provider base URL. |
| `--xtream-api-get` | No | false | Serve get.php from Xtream API. |
| `--m3u-cache-expiration` | No | 1 | M3U cache TTL (hours). |
| `--xmltv-cache-ttl` | No | — | XMLTV cache TTL (e.g. `1h`, `30m`). Empty = no cache. |
| `--xmltv-cache-max-entries` | No | 100 | Max cached XMLTV responses. |
| `--data-folder` | No | — | Folder for **settings.json** (processing: inclusions, exclusions, replacements). See [Settings](docs/settings.md). |
| `--divide-by-res` | No | false | Add resolution suffix to groups (FHD/HD/SD). |
| `--debug-logging` | No | false | Verbose debug logs. |
| `--cache-folder` | No | — | Folder for saving provider responses (debug). |
| `--use-xtream-advanced-parsing` | No | false | Alternate Xtream parsing for some providers. |
| `--ui-port` | No | 8081 | Port for the configuration UI (default 8081, one above proxy port; set 0 to disable). See [Configuration UI](docs/ui.md). |

Full reference and examples: **[docs/configuration.md](docs/configuration.md)**.

- **Filtering and renaming:** [docs/replacements.md](docs/replacements.md) (replacement rules). **Settings file:** [docs/settings.md](docs/settings.md) (settings.json, precedence). **Full reference:** [docs/configuration/reference.md](docs/configuration/reference.md).
- **Configuration UI:** [docs/ui.md](docs/ui.md) (manage groups, channels, replacements, and settings in the browser; requires `--ui-port` and `--data-folder`).
- **HTTPS / TLS behind Traefik:** [docs/traefik.md](docs/traefik.md).

---

## TLS (HTTPS)

To serve over HTTPS, run IPTV-Proxy behind a reverse proxy (e.g. Traefik). Set `--https`, `--advertised-port` (e.g. 443), and `--hostname` to your domain. Step-by-step with Traefik: **[docs/traefik.md](docs/traefik.md)**.

---

## Documentation

| Document | Description |
|----------|-------------|
| [docs/configuration.md](docs/configuration.md) | Full configuration reference and config file example. |
| [docs/replacements.md](docs/replacements.md) | M3U name/group replacement rules. |
| [docs/settings.md](docs/settings.md) | Settings file (settings.json), precedence, and reference. |
| [docs/configuration/reference.md](docs/configuration/reference.md) | Full configuration reference (all keys). |
| [docs/traefik.md](docs/traefik.md) | TLS/HTTPS with Traefik. |
| [docs/release.md](docs/release.md) | How to create a release and build binaries. |
| [docs/ui.md](docs/ui.md) | Configuration UI (groups, channels, replacements, settings). |

---

## License

GNU General Public License v3.0 — see [LICENSE](LICENSE).

---

## Developing and contributing

### Prerequisites

- **Go 1.17+**
- **Node.js 18+** (for the configuration UI)
- **npm** (for frontend deps and scripts)

### Building

From the repo root:

```bash
./scripts/build.sh
```

Output: `./iptv-proxy`. To build only the Go binary (no UI changes): `go build -o iptv-proxy .`.

### Running tests

- **Go:** `go test ./...` (or `go test ./pkg/... ./cmd/...`).
- **Configuration UI (Playwright):** from `web/frontend`, run `npm run e2e`. This starts the server with fixture data and runs the E2E suite. See [AGENTS.md](AGENTS.md) for test data and debugging tips.

### Code layout

- **`cmd/root.go`** — CLI entry, flags, config construction, server startup.
- **`pkg/config/`** — Proxy config, settings file (settings.json) types and loading.
- **`pkg/server/`** — HTTP server, M3U/Xtream handlers, configuration UI and API.
- **`web/frontend/`** — React configuration UI (Elastic UI); build output is embedded via `pkg/server/uistatic/`.

For more detail, see [AGENTS.md](AGENTS.md).

### Contributing

1. Open an issue or pick an existing one.
2. Fork the repo, create a branch, make your changes.
3. Run `go test ./...` and, if you changed the UI or API, `cd web/frontend && npm run e2e`.
4. Open a PR with a short description of the change and reference any issue.

### Credits

- Original project: [Pierre-Emmanuel Jacquier](https://github.com/pierre-emmanuelJ/iptv-proxy).
- Built with [Cobra](https://github.com/spf13/cobra), [go.xtream-codes](https://github.com/tellytv/go.xtream-codes), [Gin](https://github.com/gin-gonic/gin).
