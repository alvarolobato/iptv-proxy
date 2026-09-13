# Configuration UI

The configuration UI lets you view groups and channels from your playlist and manage `replacements.json` in the browser. The UI runs on a **separate port** from the main proxy.

## Enabling the UI

1. Set **`--ui-port`** to a port number (e.g. `9090`). The UI server will listen on that port.
2. The proxy uses **`--data-folder`** for settings and replacements (default **`/data`**, so with Docker you can mount a volume at `/data` and omit `IPTV_PROXY_DATA_FOLDER`).

Example:

```bash
./iptv-proxy --m3u-url "http://..." --port 8080 --ui-port 9090 --data-folder ./data
```

Then open **http://localhost:9090** in your browser.

With Docker, expose the UI port and mount a volume at `/data` (no need to set `IPTV_PROXY_DATA_FOLDER`):

```bash
docker run -d -p 8080:8080 -p 9090:9090 \
  -v "$(pwd)/data:/data" \
  -e M3U_URL="..." -e IPTV_PROXY_UI_PORT=9090 \
  alobato/iptv-proxy2:latest
```

## Stub replacements.json

When the server starts and `--data-folder` is set (it defaults to `/data`), it creates an empty **replacements.json** in that folder if the file does not exist. The stub contains:

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
- **Replacements** — Edit the three rule sections (global, names, groups). Add or remove regex rules, then click **Save replacements.json**. Changes are written to the file in `--data-folder`; **restart the proxy** for them to take effect on the playlist.

## TV view (play channels)

Open **`/tv`** on the UI port (e.g. `http://localhost:9090/tv`) or use the **TV** button in the header. It is a mobile-first view for watching, separate from the configuration tabs:

- Shows only the **final playlist**: included channels (after inclusions/exclusions) with replaced names and groups — the same list clients get. Excluded channels are never shown and there is no option to reveal them.
- Browse with **Live / VOD** (VOD can be narrowed to Movies or Series when both exist), a **category** (group) picker and search. Recently played channels are remembered in the browser.
- Tapping a channel opens the **player sheet**. The play button in the Channels tab opens the same sheet.

### In-browser player

Live channels (MPEG-TS) play directly in the page using [mpegts.js](https://github.com/xqq/mpegts.js). `.mp4` VOD uses the browser's native player; formats browsers can't play (e.g. `.mkv`) show only the external options.

| Browser | In-browser playback |
|---------|---------------------|
| Chrome / Edge / Firefox (desktop), Chrome on Android | Yes, for H.264 video with AAC audio |
| Safari on iPhone / iPad | iOS/iPadOS 17.1+ (ManagedMediaSource) |
| Any | HEVC only where the device has a hardware decoder; AC-3/E-AC-3 and MP2 audio usually not supported |

Video starts muted (browsers only allow muted autoplay); use **Tap to unmute**. Each open player uses one provider connection; it is released when the sheet closes. If a channel fails or doesn't start within 15 seconds, the sheet says so and points to the external options.

### Watch in another app

| Platform | Options |
|----------|---------|
| iPhone / iPad | **Open in VLC** (`vlc-x-callback://` link, alternative `vlc://` link). Requires [VLC for iOS](https://apps.apple.com/app/vlc-media-player/id650377962). |
| Android | **Open in VLC** (intent link; opens the Play Store if VLC isn't installed) and **Open with another app** (system app chooser). |
| Desktop | **Download .m3u** — a one-channel playlist that opens in VLC, IINA or mpv (desktop VLC has no link handler). |
| All | **Download .m3u** and **Copy stream URL** (e.g. VLC → Open Network Stream). |

The stream URL contains the proxy user's credentials, like the playlist URLs in the Watch tab.

## API (for integrations)

The UI is backed by a simple JSON API on the same port:

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/groups` | List unique group titles from the playlist. |
| GET | `/api/channels` | List channels (name, group, tvg_id, tvg_name, tvg_logo). |
| GET | `/api/replacements` | Current `replacements.json` content. |
| PUT | `/api/replacements` | Save `replacements.json` (body: JSON with `global-replacements`, `names-replacements`, `groups-replacements` arrays). |

No authentication is applied to the UI or API; keep the UI port behind a firewall or reverse proxy if the server is exposed.
