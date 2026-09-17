# Architecture Decision Log

Lightweight ADR (Architecture Decision Record) entries for iptv-proxy.
Newest entries first.

---

## ADR-016: Buffered in-browser player settings instead of low latency

**Date:** 2026-09-14
**Status:** Implemented
**PR:** [#51](https://github.com/alvarolobato/iptv-proxy/pull/51)

**Context:** Users reported the in-browser TV player "always buffering". The mpegts.js config enabled `liveBufferLatencyChasing` with its defaults (max latency 1.5 s, min remain 0.5 s) and disabled the stash buffer, so the player kept ~0.5 s buffered and jumped forward whenever more arrived. Benchmark in Chrome against the production proxy (La 1). 45 s per config: previous config 28 stalls / 5.3 s stalled / 0.5 s buffered ahead / 9 forward jumps; stash buffer on without chasing 0 stalls / 11.7 s ahead; relaxed chasing (10 s / 4 s) 2 stalls / 4.7 s ahead. 5 min per config, sampled every 30 s: relaxed chasing held 5.7-6.3 s ahead for the whole run with 1 stall (8 ms); no chasing settled at 17.6-19.2 s ahead with no stalls on this channel — stable here, but nothing bounds it if a provider bursts faster than realtime.

**Decision:** Keep latency chasing but relax it (`liveBufferLatencyMaxLatency` 10 s, `liveBufferLatencyMinRemain` 4 s) so the forward buffer stays bounded, including while paused (`liveBufferLatencyChasingOnPaused`, off by default: mpegts.js keeps loading when paused but only trims behind the playhead), enable the stash buffer, and tighten `autoCleanupSourceBuffer` to 60 s / 30 s (library defaults are 180 s / 120 s). Note the smoothness comes from not chasing 1.5 s of latency, not from `stashInitialSize`: for live streams mpegts.js resets the stash size from the measured bitrate after the first samples.

**Consequences:**
- Smooth playback on jittery IPTV streams; the forward buffer holds ~6 s, so live TV runs roughly that far behind real time.
- Chasing must also be enabled while paused. Measured over a 150 s pause: with the library default (off while paused) the forward buffer grew from 7.6 s to 156 s, a second per second with no bound, heading for a full SourceBuffer; with it on the buffer stayed 4.0-5.2 s.
- Consequence of that: pausing live TV does not hold the frame. The player keeps up with live while paused, so resuming continues from live rather than from the pause point.
- The forward buffer stays bounded: with chasing disabled entirely, a provider that bursts faster than realtime grows the SourceBuffer until it is full, and mpegts.js suspends loading for live streams without ever resuming (the resume hook only exists on the lazyLoad path), which freezes playback silently.
- Memory per session is bounded by the cleanup windows.

---

## ADR-014: TV play view with mpegts.js player and VLC deep links

**Date:** 2026-09-13
**Status:** Implemented
**PR:** [#49](https://github.com/alvarolobato/iptv-proxy/pull/49)

**Context:** The Channels tab "Open stream" action opened the raw `.ts` in a new tab, which no browser plays. Viewers want to play channels from the browser, including on phones. The Channels tab is a configuration tool (included and excluded channels, include/exclude/rename actions). Provider streams are continuous MPEG-TS over HTTP (no usable live HLS); sampled channels are H.264 + AAC, some are HEVC.

**Decision:**
- Add a dedicated **`/tv` view** instead of changing the Channels tab defaults: it shows only the final processed list (included channels with replaced names/groups and a `stream_url`, no toggle or filter that reveals excluded ones), a Live/VOD switch (VOD = movies + series), category (group) selection and search, with a minimal mobile-first header. Configuration and viewing needs differ (excluded rows and edit actions are noise when watching), and a separate route is bookmarkable on a phone.
- Play **in the browser with mpegts.js** (transmuxes MPEG-TS to fMP4 into MSE/ManagedMediaSource) rather than server-side ffmpeg → HLS: no server CPU, no new process lifecycle or image size, ~6 s behind live with the buffered player settings (see ADR-016).
- Always offer **external players**: VLC deep links on iOS (`vlc-x-callback://x-callback-url/stream?url=`, `vlc://`) and Android (intent with `package=org.videolan.vlc` + Play Store fallback, and a chooser intent without package), a one-channel `.m3u` download (desktop VLC registers no URL scheme) and copy URL.
- The Channels tab play action opens the same player sheet (native `<a href>` kept for middle-click).

**Consequences:**
- New dependency: `mpegts.js` 1.8.2 (single maintainer; open issues on iPhone detection and long-session memory — `autoCleanupSourceBuffer` enabled).
- iPhone in-browser playback needs iOS 17.1+; HEVC plays only with a hardware decoder; AC-3/E-AC-3/MP2 audio generally doesn't play in Chrome/Firefox — those cases fall back to VLC. Interlaced channels are not deinterlaced.
- Each in-browser viewer consumes a provider connection; the player is destroyed when the sheet closes, when playback fails, and when an external player option is used (so the external app gets a connection). The TV view loads only the final list via `GET /api/channels?included=1`.
- Server-side HLS remux/transcode (ffmpeg, Intel Quick Sync on the N100 host) remains the fallback if real-device testing shows mpegts.js is not good enough on iPhone or for HEVC/AC-3 channels.

---

## ADR-013: Xtream-form stream URLs when the M3U comes from the Xtream get.php

**Date:** 2026-09-13
**Status:** Implemented
**PR:** [#46](https://github.com/alvarolobato/iptv-proxy/pull/46)

**Context:** With only Xtream credentials configured, the M3U source is the account's own `get.php`. In that mode `routes()` registers the Xtream routes and the auto-M3U route but not the per-track anti-collision routes (`/<id>/:user/:password/<index>/<file>`). The UI still generated anti-collision stream URLs, so every "Open stream" link returned 404. Separately, `/play/:user/:password/:id` forwarded upstream without the `/play` prefix, which providers reject for `.ts` ids.

**Decision:** A single predicate, `servesXtreamM3U()` (same host/port as the Xtream base URL, `get.php` path, matching non-empty credentials), selects the mode for both route registration and UI stream URLs. In that mode stream URLs use the Xtream form (provider credentials swapped for proxy credentials in the provider path, e.g. `/play/<user>/<pass>/<id>.ts`). The `/play` dispatcher forwards to `{base}/play/{xu}/{xp}/{id}` and sets the `id` route param so `.m3u8` requests use the HLS handler.

**Consequences:**
- UI links always match a registered route; covered by a router-level test against an `httptest` upstream.
- M3U clients using `/play` URLs work again.
- Anti-collision URLs remain for plain M3U sources.

---

## ADR-012: Add copyright headers and NOTICE file for fork

**Date:** 2026-03-22
**Status:** Implemented
**PR:** [#37](https://github.com/alvarolobato/iptv-proxy/pull/37)

**Context:** This is a fork of Pierre-Emmanuel Jacquier's iptv-proxy (GPL v3). New files added in the fork lacked copyright headers, and there was no clear attribution document.

**Decision:** Keep original GPL v3 headers on upstream files unchanged. Add GPL v3 headers to all fork-added files crediting both the original project and the fork author. Add a NOTICE file at repo root documenting the fork relationship.

**Consequences:**
- Clear license compliance and attribution
- Easy to audit which files are original vs fork-added

---

## ADR-011: Elasticsearch-backed channel statistics

**Date:** 2026-03-04
**Status:** Implemented
**PR:** [#28](https://github.com/alvarolobato/iptv-proxy/pull/28), [#29](https://github.com/alvarolobato/iptv-proxy/pull/29)

**Context:** Needed visibility into which channels are watched, session durations, and usage patterns. Required a time-series capable backend for metrics rollups.

**Decision:** Use Elasticsearch with TSDB-backed data streams for channel metrics (`metrics-iptv.channel_metrics`) and regular indices for sessions/user history. Use `tvg-name` as canonical channel identifier across both M3U and Xtream. No-op collector when ES is not configured.

**Consequences:**
- Zero overhead when ES is disabled (no-op collector)
- TSDB prefix (`metrics-`) reserved only for true time-series data streams — learned the hard way that ES serverless rejects non-TSDB indices with that prefix (PR #29 fix)
- Channel identification is consistent across M3U and Xtream modes

---

## ADR-010: Gate Docker images and binaries on all tests passing

**Date:** 2026-03-04
**Status:** Implemented
**PR:** [#27](https://github.com/alvarolobato/iptv-proxy/pull/27)

**Context:** Docker images and release binaries could be published even when tests were failing.

**Decision:** Add CI gates so Docker image and binary release jobs depend on all tests passing first.

**Consequences:**
- No broken releases reach users
- Slightly longer CI pipeline (tests must finish before publish)

---

## ADR-009: Apply filtering and replacements to Xtream API responses

**Date:** 2026-03-04
**Status:** Implemented
**PR:** [#25](https://github.com/alvarolobato/iptv-proxy/pull/25)

**Context:** Group/channel inclusions, exclusions, and replacement rules worked for M3U but were silently ignored for Xtream `player_api.php` responses. Xtream clients saw unfiltered content.

**Decision:** Add `filterXtreamResponse()` and `applyXtreamReplacements()` to the server config, called after `client.Action()` returns. Reuse existing `matchInclusionExclusion()` logic.

**Consequences:**
- Filtering behavior is now consistent between M3U and Xtream modes
- Single code path for inclusion/exclusion matching

---

## ADR-008: Apply exclusions to served M3U on settings change

**Date:** 2026-03-03
**Status:** Implemented
**PR:** [#21](https://github.com/alvarolobato/iptv-proxy/pull/21)

**Context:** Exclusions set via the UI showed correctly in the UI but were not applied to the actual M3U file served to clients. Root cause: `marshallInto` was re-filtering an already-filtered playlist instead of the original.

**Decision:** Keep the original unfiltered playlist and re-filter from it when settings change. `playlistInitialization()` now works from the full track list.

**Consequences:**
- Settings changes take effect immediately for M3U clients
- Original playlist is preserved for re-filtering

---

## ADR-007: Configuration UI with separate port

**Date:** 2026-03-02
**Status:** Implemented
**PR:** [#19](https://github.com/alvarolobato/iptv-proxy/pull/19)

**Context:** No way to manage groups, channels, and replacements without editing JSON files. Needed a web UI for configuration.

**Decision:** React + Vite + Elastic UI (EUI) frontend embedded in the Go binary. Served on a separate `--ui-port` (default 0 = disabled). API endpoints: `/api/groups`, `/api/channels`, `/api/replacements`, `/api/settings`. Auto-create stub `replacements.json` on startup.

**Consequences:**
- UI is opt-in (disabled by default)
- Embedded in binary — no separate deployment
- EUI provides consistent, accessible components
- Icons must be registered in `icons_hack.jsx` (EUI quirk)

---

## ADR-006: Documentation restructure and iptv-proxy2 branding

**Date:** 2026-03-02
**Status:** Implemented
**PR:** [#16](https://github.com/alvarolobato/iptv-proxy/pull/16), [#17](https://github.com/alvarolobato/iptv-proxy/pull/17)

**Context:** Fork had accumulated features but documentation was scattered and mixed with upstream. Needed clear branding and organized docs.

**Decision:** Rebrand to iptv-proxy2. Move detailed docs to `docs/` (configuration.md, replacements.md, traefik.md). README focuses on quick start with Docker. Add Required? column to config tables. Release workflow for binaries.

**Consequences:**
- Clear separation between quick-start README and detailed reference
- Docs are the source of truth for configuration options

---

## ADR-005: Debug logging, cache folder, and advanced parsing flags

**Date:** 2026-03-01
**Status:** Implemented
**PR:** [#15](https://github.com/alvarolobato/iptv-proxy/pull/15)

**Context:** Needed runtime toggles for debug output, configurable cache location, and optional advanced Xtream parsing.

**Decision:** Three new flags: `--debug-logging`, `--cache-folder`, `--use-xtream-advanced-parsing`. Sourced from jtdevops_master fork.

**Consequences:**
- Debug mode helps troubleshoot provider issues without code changes
- Cache folder is configurable for Docker volume mounts

---

## ADR-004: XMLTV caching with retry and Range header support

**Date:** 2026-03-01
**Status:** Implemented
**PR:** [#13](https://github.com/alvarolobato/iptv-proxy/pull/13)

**Context:** XMLTV fetches are slow and providers are unreliable. Clients seeking in streams need Range header support.

**Decision:** Add `responseCache` with configurable TTL (`--xmltv-cache-ttl`). Cache key is canonical query string. Retry up to 3 times with backoff. Return empty `<tv></tv>` on total failure. Forward `Range` header on stream proxy.

**Consequences:**
- Dramatically fewer upstream XMLTV requests
- Clients can seek in streams
- Graceful degradation on provider failures

---

## ADR-003: Regex filtering, replacements, and resolution groups for M3U

**Date:** 2026-03-01
**Status:** Implemented
**PR:** [#12](https://github.com/alvarolobato/iptv-proxy/pull/12)

**Context:** Users needed to filter channels by group/name regex, rewrite names/groups, and optionally split by resolution (FHD/HD/SD). Sourced from ridgarou_master fork.

**Decision:** Add `--group-regex`, `--channel-regex`, `--json-folder`, `--divide-by-res`. Regex compiled once at startup. `replacements.json` with global/names/groups sections. Empty regex = match all.

**Consequences:**
- Flexible filtering without modifying source M3U
- Replacements enable clean channel/group names for clients
- Resolution grouping useful for multi-device setups

---

## ADR-002: M3U patch parsing for tvg-name and tvg-logo

**Date:** 2026-03-01
**Status:** Implemented
**PR:** [#11](https://github.com/alvarolobato/iptv-proxy/pull/11)

**Context:** Some providers use unhelpful track names (`dpr_auto`, `h_256`) and malformed tvg-logo values (containing commas). Sourced from Gibby_patch-parsing.

**Decision:** Use `tvg-name` as display name when the track name matches known bad patterns. Clear `tvg-logo` when value contains comma (malformed).

**Consequences:**
- Better channel names for clients
- No broken image URLs from malformed logos

---

## ADR-001: Xtream struct robustness and HLS improvements

**Date:** 2026-03-01
**Status:** Implemented
**PR:** [#10](https://github.com/alvarolobato/iptv-proxy/pull/10)

**Context:** Xtream providers return inconsistent JSON (wrong types, missing fields). HLS streaming had URL issues. Sourced from Yagoor_master.

**Decision:** Custom `UnmarshalJSON` for Xtream structs with optional `*Info` types to tolerate missing/invalid nested data. HLS token as query param for provider compatibility.

**Consequences:**
- Proxy handles a wider range of Xtream providers
- HLS works with providers that don't support path-based tokens
