# Configuration UI

The configuration UI lets you view groups and channels from your playlist and manage `replacements.json` in the browser. The UI runs on a **separate port** from the main proxy.

## Enabling the UI

1. Set **`--ui-port`** to a port number (e.g. `9090`). The UI server will listen on that port.
2. Set **`--json-folder`** so the proxy (and UI) can read and write `replacements.json` (e.g. `./data` or `/data` in Docker).

Example:

```bash
./iptv-proxy --m3u-url "http://..." --port 8080 --ui-port 9090 --json-folder ./data
```

Then open **http://localhost:9090** in your browser.

With Docker, expose the UI port and set both env vars:

```bash
docker run -d -p 8080:8080 -p 9090:9090 \
  -v "$(pwd)/data:/data" \
  -e M3U_URL="..." -e JSON_FOLDER=/data -e IPTV_PROXY_UI_PORT=9090 \
  alobato/iptv-proxy2:latest
```

## Stub replacements.json

When the server starts and `--json-folder` is set, it creates an empty **replacements.json** in that folder if the file does not exist. The stub contains:

```json
{
  "global-replacements": [],
  "names-replacements": [],
  "groups-replacements": []
}
```

You can then edit it via the UI or by hand.

## Tabs

- **Groups** — Table of all unique `group-title` values from the current playlist (from the M3U source). Empty if no M3U is loaded or the playlist is empty.
- **Channels** — Table of all channels: name, group, tvg-id, tvg-name, tvg-logo. Use this to see what names and groups you might want to rewrite.
- **Replacements** — Edit the three rule sections (global, names, groups). Add or remove regex rules, then click **Save replacements.json**. Changes are written to the file in `--json-folder`; **restart the proxy** for them to take effect on the playlist.

## API (for integrations)

The UI is backed by a simple JSON API on the same port:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/groups` | List unique group titles from the playlist. |
| GET | `/api/channels` | List channels (name, group, tvg_id, tvg_name, tvg_logo). |
| GET | `/api/replacements` | Current `replacements.json` content. |
| PUT | `/api/replacements` | Save `replacements.json` (body: JSON with `global-replacements`, `names-replacements`, `groups-replacements` arrays). |

No authentication is applied to the UI or API; keep the UI port behind a firewall or reverse proxy if the server is exposed.
