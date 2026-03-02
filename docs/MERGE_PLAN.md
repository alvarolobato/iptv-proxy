# Merge Plan: Bringing Fork Features into Master

This document is the detailed plan for consolidating fork features into `master` via new PRs. No CI changes are included. Each PR is self-contained with one commit per logical group; overlapping features use the single chosen implementation.

---

## Decisions

| Decision | Choice | Reason |
|----------|--------|--------|
| **Struct/VOD/EPG/HLS/startup** | Yagoor_master | Clean vendor changes, HLS fix, optional types; no debug/cache clutter. |
| **M3U patch parsing** | Gibby_patch-parsing | Only branch with this feature. Fix: declare `name`, use log instead of println. |
| **Regex filter + replacement + resolution** | ridgarou_master | Only branch with all three (filter, replacement, DivideByRes). Fix: compile regex only when non-empty; use stdlib or avoid x/exp/slices; keep upstream module path and CustomId. |
| **XMLTV cache + Range** | chernandezweb_master | Only branch with XMLTV cache/retry and Range header. Omit README/docker overhaul; bring only code. |
| **Debug/cache/advanced/m3u4u/error details** | jtdevops_master | Has unit tests and error_utils. Default USE_XTREAM_ADVANCED_PARSING to false. |
| **get.php play fix** | ridgarou_master | Single, clear behavioral fix. |
| **CI** | None | Explicitly not bringing any CI from any branch. |

---

## PRs to Create (order)

### PR1: Xtream struct, VOD/EPG fixes, HLS fix, startup logs
- **Branch:** `pr/1-xtream-struct-vod-epg-hls`
- **Source:** Yagoor_master
- **Contents:** Vendor `go.xtream-codes` UnmarshalJSON and optional `*Info` types; HLS URL fix (token as query); VOD/EPG fixes for incomplete data; startup "Starting"/"Started" logs in server; route tweak if any.
- **Exclude:** CD, goreleaser, README from Yagoor.
- **Fixes:** None beyond using upstream module path.
- **Commit:** One commit: "feat: xtream struct robustness, HLS fix, VOD/EPG fixes, startup logs"

---

### PR2: M3U patch parsing (tvg-name, tvg-logo)
- **Branch:** `pr/2-m3u-patch-parsing`
- **Base:** master (or PR1 if merged first; independent so base master)
- **Source:** Gibby_patch-parsing
- **Contents:** In `marshallInto`: use tvg-name as display name for tracks named `dpr_auto`, `h_256`, or containing `320"`; strip tvg-logo value when it contains a comma.
- **Fixes:** Declare `name := track.Name` at start of track loop; set `name = track.Tags[i].Value` when tag is tvg-name; use `log.Printf` instead of `println`. Keep upstream module path.
- **Commit:** One commit: "feat: M3U patch parsing for tvg-name and tvg-logo"

---

### PR3: Regex filter, replacement, resolution groups
- **Branch:** `pr/3-regex-filter-replacement-resolution`
- **Base:** master
- **Source:** ridgarou_master (filter + replacement + DivideByRes)
- **Contents:** Config: GroupRegex, ChannelRegex, JSONFolder, DivideByRes. Server: compile regex only when non-empty (empty = match all); apply replacements from JSON (Global, Names, Groups); optional "Divide by Resolution" (FHD/HD/SD). Add Replacements struct and loadReplacements/applyReplacements (in server or handlers); optional tvg-id tag injection. Keep CustomId and master’s NewServer (trimmedCustomId).
- **Fixes:** (1) Only compile GroupRegex/ChannelRegex when non-empty; use nil or match-all semantics. (2) Use simple loop instead of slices.IndexFunc (no x/exp/slices). (3) Upstream module path. (4) pathExists and loadReplacements: use os.Stat and log instead of fmt. (5) Add example replacements.json under docs or in README.
- **Commit 1:** "feat: regex filter and replacement for M3U (group, channel, names)"  
- **Commit 2:** "feat: optional divide groups by resolution (FHD/HD/SD)"  
  (Or one commit: "feat: regex filter, replacement, and resolution groups for M3U")
- **Commit:** One commit preferred: "feat: regex filter, replacement, and resolution groups for M3U"

---

### PR4: XMLTV cache, retry, Range header
- **Branch:** `pr/4-xmltv-cache-range`
- **Base:** master (or PR1 if structs needed; chernandezweb has its own vendor changes – prefer basing on master and including only cache/retry/Range/509 in application code; if vendor conflict, base on PR1)
- **Source:** chernandezweb_master
- **Contents:** pkg/server/cache.go (new); XMLTV handler with cache and retry; return empty XMLTV on failure; Range header support in stream handler; 509 response body persistence for debugging. Omit README overhaul and docker-compose template.
- **Fixes:** Omit "." commits; single clean commit. Resolve vendor if any (prefer minimal vendor change or depend on PR1).
- **Commit:** One commit: "feat: XMLTV caching with retry and Range header support"

---

### PR5: Debug, cache folder, advanced parsing, m3u4u, error details
- **Branch:** `pr/5-debug-advanced-m3u4u`
- **Base:** master (or PR1; jtdevops has struct overlap – base on PR1 after merge, or include only non-struct parts and depend on PR1)
- **Source:** jtdevops_master
- **Contents:** DEBUG_LOGGING, CACHE_FOLDER, USE_XTREAM_ADVANCED_PARSING (default false); save provider responses to files; error_utils and error details; m3u4u URL option and URL handling; pkg/utils (debug, error_utils, file_utils); unit tests. Omit duplicate struct/VOD/EPG (already in PR1).
- **Fixes:** Default USE_XTREAM_ADVANCED_PARSING to false; document CACHE_FOLDER; upstream module path.
- **Commit:** One commit: "feat: debug logging, cache folder, advanced parsing, m3u4u URL, error details"

---

### PR6: get.php play fix
- **Branch:** `pr/6-get-php-play-fix`
- **Base:** master
- **Source:** ridgarou_master
- **Contents:** Handler/routing change so video routes use "play" (fix del handler get.php para que las rutas de los videos lleven play).
- **Fixes:** Upstream module path only.
- **Commit:** One commit: "fix: get.php video routes use play for client compatibility"

---

## What We Are NOT Bringing In (and why)

| Item | Branches | Reason |
|------|----------|--------|
| **Any CI/CD** | Gibby_main, Gibby_patch-parsing, Gibby_regex-filters, ridgarou_master, Yagoor (cd.yml) | User requested no CI. |
| **Module path changes** | Gibby_*, ridgarou | Keep `github.com/alvarolobato/iptv-proxy` (or repo owner). |
| **Author/attribution change** | ridgarou (pierre → ridgarou) | Keep original attribution. |
| **README overhaul / docker-compose template** | chernandezweb | Only code features; docs can be updated separately. |
| **Trivial "." commits** | chernandezweb | Noise; not brought in. |
| **Duplicate struct/VOD/EPG/startup** | jtdevops, michbeck100 | Covered by PR1 (Yagoor). |
| **Duplicate regex-only (no replacement/resolution)** | Gibby_regex-filters, michbeck100 | PR3 uses ridgarou’s full implementation. |
| **Second copy of debug/advanced/m3u4u** | michbeck100 | Covered by PR5 (jtdevops). |
| **Full response-saving to files** | jtdevops, michbeck100 | PR5 adds config/flags only; wiring in handlers/xtream-proxy deferred. |
| **m3u4u.com URL option** | jtdevops, michbeck100 | Deferred; can be added in a follow-up. |
| **509 response body persistence** | chernandezweb | Deferred; not in PR4. |
| **error_utils / error details** | jtdevops | Deferred; would require pkg/utils and handler changes. |

---

## Task Checklist

- [x] **Plan** – MERGE_PLAN.md written (this doc)
- [x] **PR1** – [#10](https://github.com/alvarolobato/iptv-proxy/pull/10) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **PR2** – [#11](https://github.com/alvarolobato/iptv-proxy/pull/11) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **PR3** – [#12](https://github.com/alvarolobato/iptv-proxy/pull/12) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **PR4** – [#13](https://github.com/alvarolobato/iptv-proxy/pull/13) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **PR5** – [#15](https://github.com/alvarolobato/iptv-proxy/pull/15) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **PR6** – [#14](https://github.com/alvarolobato/iptv-proxy/pull/14) – Branch created, pushed, PR created, review requested (@copilot)
- [x] **Doc** – FORK_BRANCHES_COMPARISON.md updated with plan ref and "Not bringing in"; all tasks marked done.

---

## Implementation Notes

- Use **git worktree** (or `wt`) so each PR has its own working tree branching from `origin/master`.
- One commit per PR (or per logical group within a PR as specified above).
- All PRs target `master`. Chain only if a PR explicitly bases on another (e.g. PR5 could base on PR1 after it’s merged; for simplicity we base all on master and resolve conflicts at review/merge time).
- After each PR is created: `gh pr create ...` then `gh pr edit <number> --add-reviewer copilot` (or the correct GitHub username for Copilot review).
