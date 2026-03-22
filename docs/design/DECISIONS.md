# Architecture Decision Log

Lightweight ADR (Architecture Decision Record) entries for iptv-proxy.
Newest entries first.

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
