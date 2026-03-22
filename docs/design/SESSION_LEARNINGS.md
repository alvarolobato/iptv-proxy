# Session learnings log

Captures hurdles, mistakes, and non-obvious discoveries made during development sessions. Each entry is a brief post-mortem that helps future sessions avoid the same pitfalls.

> **When to add an entry:** At the end of a session where you hit a non-obvious problem, wasted time on a wrong approach, or discovered something that isn't documented elsewhere. Skip if the session was straightforward.

---

## Format

```markdown
## YYYY-MM-DD — <short title>

**PR/commit:** link
**What happened:** 1-2 sentences on the problem or mistake.
**Root cause:** Why it happened.
**Fix/workaround:** What resolved it.
**Takeaway:** One-liner for future sessions.
```

---

## 2026-03-22 — ES serverless rejects metrics- prefix on non-TSDB indices

**PR/commit:** [#29](https://github.com/alvarolobato/iptv-proxy/pull/29)
**What happened:** Elasticsearch serverless rejected `metrics-iptv.sessions` and `metrics-iptv.user_history` indices — they aren't TSDB data streams.
**Root cause:** The `metrics-` prefix is reserved for TSDB in ES serverless. Only `channel_metrics` (with gauge/counter fields) qualifies as TSDB.
**Fix/workaround:** Renamed to `iptv.sessions` and `iptv.user_history`. Kept `metrics-iptv.channel_metrics` as the only true TSDB stream.
**Takeaway:** Only use the `metrics-` prefix for actual TSDB data streams in Elasticsearch.

---

## 2026-03-03 — Exclusions worked in UI but not in served M3U

**PR/commit:** [#21](https://github.com/alvarolobato/iptv-proxy/pull/21)
**What happened:** UI showed channels as excluded, but the M3U downloaded by devices still contained all channels.
**Root cause:** `marshallInto()` was re-filtering an already-filtered playlist. After the first filter pass, the original tracks were lost.
**Fix/workaround:** Preserve the original unfiltered playlist and always re-filter from it.
**Takeaway:** When applying filters that reduce a list, always keep the original and filter from it — never mutate in place.

---

## 2026-03-04 — Xtream filtering silently did nothing

**PR/commit:** [#25](https://github.com/alvarolobato/iptv-proxy/pull/25)
**What happened:** Group/channel inclusion and exclusion rules worked for M3U but were completely ignored for Xtream `player_api.php` responses.
**Root cause:** The Xtream handler returned raw upstream responses without passing them through any filtering.
**Fix/workaround:** Added `filterXtreamResponse()` and `applyXtreamReplacements()` reusing the same matching logic as M3U.
**Takeaway:** When adding filtering/transformation logic, verify it applies to all code paths (M3U and Xtream) — not just the one you tested.

---

## 2026-03-02 — Missing settings.go broke CI

**PR/commit:** Commit `8be8b44`
**What happened:** CI failed with "undefined: SettingsJSON" after a PR that didn't include `pkg/config/settings.go`.
**Root cause:** `settings_load.go` references types from `settings.go`. The file existed locally but wasn't committed.
**Fix/workaround:** Added `settings.go` to the commit.
**Takeaway:** When splitting types across files, make sure all files are committed. Run `go build` before pushing.

---

## 2026-03-02 — EUI icons not rendering

**PR/commit:** PR [#19](https://github.com/alvarolobato/iptv-proxy/pull/19)
**What happened:** Play and copy icons in the configuration UI rendered as empty/missing.
**Root cause:** EUI requires icons to be explicitly registered via `appendIconComponentCache`. The icons were used in JSX but not registered in `icons_hack.jsx`.
**Fix/workaround:** Added icon imports and registration in `web/frontend/src/icons_hack.jsx`.
**Takeaway:** When adding new EUI icons, always register them in `icons_hack.jsx` and rebuild.

---

## 2026-03-02 — Open stream link opened about:blank#blocked

**PR/commit:** Commit `7810704`
**What happened:** Clicking "Open stream" in the Channels table opened `about:blank#blocked` instead of the stream URL.
**Root cause:** `EuiButtonEmpty` with `href` didn't produce a proper anchor element — browser blocked the navigation.
**Fix/workaround:** Replaced with a native `<a href={streamUrl}>` element.
**Takeaway:** For links that should navigate (especially to non-page URLs like streams), use native `<a>` elements instead of EUI button components with `href`.
