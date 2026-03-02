# Fork Branches Comparison

This document compares the branches brought in from forks and their corresponding PRs into `master`. It summarizes what each PR introduces, where they overlap, and which branches bring in the same or similar commits/features.

**Implementation plan:** See [docs/MERGE_PLAN.md](MERGE_PLAN.md) for the consolidated merge strategy. New feature PRs (no CI) are below.

**New PRs (consolidated features, base: `master`):**

| PR | Branch | Title |
|----|--------|--------|
| [#10](https://github.com/alvarolobato/iptv-proxy/pull/10) | pr/1-xtream-struct-vod-epg-hls | Xtream struct robustness, HLS fix, VOD/EPG fixes, startup logs |
| [#11](https://github.com/alvarolobato/iptv-proxy/pull/11) | pr/2-m3u-patch-parsing | M3U patch parsing (tvg-name, tvg-logo) |
| [#12](https://github.com/alvarolobato/iptv-proxy/pull/12) | pr/3-regex-filter-replacement-resolution | Regex filter, replacement, resolution groups for M3U |
| [#13](https://github.com/alvarolobato/iptv-proxy/pull/13) | pr/4-xmltv-cache-range | XMLTV caching with retry and Range header support |
| [#15](https://github.com/alvarolobato/iptv-proxy/pull/15) | pr/5-debug-advanced-m3u4u | Debug/cache/advanced-parsing flags (config only) |
| [#14](https://github.com/alvarolobato/iptv-proxy/pull/14) | pr/6-get-php-play-fix | /play/user/password/:id route for client compatibility |

**Original fork-branch PRs (reference only; superseded by the above):**

| PR | Branch | Title |
|----|--------|--------|
| [#1](https://github.com/alvarolobato/iptv-proxy/pull/1) | Gibby_main | CI/CD, Release Please, Docker, module path |
| [#2](https://github.com/alvarolobato/iptv-proxy/pull/2) | Gibby_patch-parsing | M3U patch parsing (tvg-name, tvg-logo) |
| [#3](https://github.com/alvarolobato/iptv-proxy/pull/3) | Gibby_regex-filters | Regex filter for M3U group and channel |
| [#4](https://github.com/alvarolobato/iptv-proxy/pull/4) | Yagoor_master | UnmarshalJSON, HLS fix, VOD/EPG fixes, startup logs |
| [#5](https://github.com/alvarolobato/iptv-proxy/pull/5) | chernandezweb_master | XMLTV cache/retry, Range header, README/docker |
| [#6](https://github.com/alvarolobato/iptv-proxy/pull/6) | jtdevops_master | Debug, cache folder, advanced parsing, m3u4u, VOD/EPG |
| [#7](https://github.com/alvarolobato/iptv-proxy/pull/7) | michbeck100_master | Regex filters, debug, cache, advanced parsing, m3u4u |
| [#8](https://github.com/alvarolobato/iptv-proxy/pull/8) | ridgarou_master | Filter/replacement, resolution groups, get.php play fix |

---

## 1. What each PR introduces

### Gibby_main (PR #1)
- **Scope:** CI/CD and fork identity only; no new app features.
- **Content:** Replaces existing CI with Dependabot, CodeQL, Docker workflows, Release Please; simplifies goreleaser; changes Go module path to `github.com/gibby/iptv-proxy`.
- **Merge recommendation:** Do not merge as-is; adopt selected workflows and keep upstream module path.

### Gibby_patch-parsing (PR #2)
- **Scope:** M3U output only.
- **Content:** Patches M3U generation: strip commas from `tvg-logo`; use `tvg-name` as display name for tracks named `dpr_auto`, `h_256`, or containing `320"`.
- **Note:** Builds on Gibby_main (shared commits). Contains a bug: variable `name` is undefined in outer scope (must be fixed before merge).

### Gibby_regex-filters (PR #3)
- **Scope:** M3U filtering.
- **Content:** Adds `GroupRegex` and `ChannelRegex` config; filters M3U tracks by group-title and channel name. Builds on Gibby_main.
- **Improvement:** Compile regexes once before the track loop, not per track.

### Yagoor_master (PR #4)
- **Scope:** Xtream API robustness and HLS.
- **Content:** Custom `UnmarshalJSON` in vendor `go.xtream-codes` for inconsistent JSON; optional `*Info` types for VOD/Series; HLS URL fix (token as query `?token=`); EPG and VOD (Shows & Movies) fixes for incomplete data; startup logs; CD/README tweaks.
- **Merge recommendation:** Valuable; prefer pushing struct changes upstream to go.xtream-codes long term.

### chernandezweb_master (PR #5)
- **Scope:** Resilience, streaming, and docs.
- **Content:** XMLTV caching with retry and empty XMLTV on failure; Range header support for partial content; docker-compose template; README overhaul; error logging (e.g. persist response body for 509); fix empty series. Many small “.” commits.
- **Overlap:** Vendor/struct changes similar to Yagoor/jtdevops; otherwise mostly unique (cache, Range, README).

### jtdevops_master (PR #6)
- **Scope:** Debugging, logging, parsing, URL handling.
- **Content:** DEBUG_LOGGING, CACHE_FOLDER, USE_XTREAM_ADVANCED_PARSING; save provider responses to files; advanced parsing to preserve raw responses; error-details handling; UnmarshalJSON and struct fixes (same lineage as Yagoor); VOD/EPG fixes and startup logs; m3u4u.com URL option and URL handling fixes; unit tests and test data.
- **Merge recommendation:** Unify with Yagoor and michbeck100 to avoid duplicate struct/VOD/EPG code.

### michbeck100_master (PR #7)
- **Scope:** Combined feature branch (regex + debug + parsing + fixes).
- **Content:** Same ideas as jtdevops (debug, cache folder, advanced parsing, UnmarshalJSON, VOD/EPG, startup logs, m3u4u, URL fixes) **plus** regex filters for channel/group (same concept as Gibby_regex-filters). Different commit hashes than jtdevops but same feature set plus regex.
- **Merge recommendation:** Do not merge as a second copy of jtdevops; consolidate with jtdevops and Yagoor for app logic, and with Gibby_regex-filters/ridgarou for filtering.

### ridgarou_master (PR #8)
- **Scope:** Filtering, replacement, routing, CI.
- **Content:** Filter M3U by regex (group/channel); **replacement** of channel/group names (including Xtream); “Divide groups by Resolution”; get.php fix so video routes use “play”; CI/CD and dependency updates; author change pierre → ridgarou.
- **Overlap:** Filter concept overlaps with Gibby_regex-filters and michbeck100; replacement is an extra feature.

---

## 2. Overlap and same-commit lineage

### Branches that share commit history
- **Gibby_main**, **Gibby_patch-parsing**, **Gibby_regex-filters**  
  - Patch-parsing and regex-filters are built **on top of** Gibby_main. They share commits such as `719607e`, `2e8bab4`, `69d8911` (path changes, gitignore, CI). Any merge of patch-parsing or regex-filters will pull in Gibby_main’s history unless rebased onto master and stripped of fork-specific changes.

### Same features, different commits (no shared hash)
- **Yagoor_master**, **jtdevops_master**, **michbeck100_master**  
  - All three introduce:
    - UnmarshalJSON and struct changes in vendor `go.xtream-codes`
    - VOD (Shows & Movies) and EPG fixes for incomplete data
    - Startup “Starting” / “Started” logs  
  - So they are **bringing in the same logical changes** from a common “enhancements” lineage, implemented in separate forks. Merging more than one of these without consolidation will duplicate these changes and cause conflicts.
- **jtdevops_master** and **michbeck100_master**  
  - Both add: DEBUG_LOGGING, CACHE_FOLDER, USE_XTREAM_ADVANCED_PARSING, advanced parsing, response saving, m3u4u URL option, URL handling fixes. Same features, different commits. Only one implementation should be merged (or one consolidated branch).

### Same feature area, different implementations
- **Regex / filter by group and channel:**
  - **Gibby_regex-filters:** GroupRegex, ChannelRegex; filter only.
  - **michbeck100_master:** Same idea (regex filter channel/group) plus the rest of its stack.
  - **ridgarou_master:** Filter by regex **and** replacement of names; “Divide groups by Resolution.”  
  - Overlap: filtering. Ridgarou adds replacement and resolution grouping. Recommendation: one unified “filter (and optional replacement)” design.

### Largely unique
- **chernandezweb_master:** XMLTV cache/retry, Range header, 509 body logging, docker-compose template, README – mostly distinct; only vendor/struct overlap with Yagoor/jtdevops.
- **Gibby_patch-parsing:** M3U tvg-name/tvg-logo patch only; unique except for dependency on Gibby_main.

---

## 3. Summary table: features by branch

| Feature | Gibby_main | patch-parsing | regex-filters | Yagoor | chernandezweb | jtdevops | michbeck100 | ridgarou |
|--------|------------|---------------|---------------|--------|---------------|----------|--------------|----------|
| CI/CD / release (fork) | ✓ | (via main) | (via main) | minor | - | - | - | ✓ |
| Module path → fork | ✓ | ✓ | ✓ | - | - | - | - | - |
| M3U tvg-name / tvg-logo patch | - | ✓ | - | - | - | - | - | - |
| M3U regex filter (group/channel) | - | - | ✓ | - | - | - | ✓ | ✓ |
| Name replacement | - | - | - | - | - | - | - | ✓ |
| Resolution groups | - | - | - | - | - | - | - | ✓ |
| UnmarshalJSON / struct fixes | - | - | - | ✓ | ✓ | ✓ | ✓ | - |
| HLS token query fix | - | - | - | ✓ | - | - | - | - |
| VOD/EPG/startup fixes | - | - | - | ✓ | - | ✓ | ✓ | - |
| XMLTV cache + retry | - | - | - | - | ✓ | - | - | - |
| Range header (partial content) | - | - | - | - | ✓ | - | - | - |
| Debug/cache folder/advanced parsing | - | - | - | - | - | ✓ | ✓ | - |
| m3u4u URL / URL handling | - | - | - | - | - | ✓ | ✓ | - |
| get.php “play” fix | - | - | - | - | - | - | - | ✓ |
| Error details / 509 body | - | - | - | - | ✓ | ✓ | ✓ | - |

---

## 4. Recommended merge strategy

1. **Do not merge** Gibby_main as-is; reuse only desired CI pieces and keep upstream module path.
2. **Fix and merge** Gibby_patch-parsing after fixing the `name` variable and reverting fork paths; consider squashing onto master without Gibby_main history.
3. **Unify regex/filter:** Pick one implementation (e.g. Gibby_regex-filters or ridgarou’s), add optional “replacement” and “resolution groups” from ridgarou if desired, then merge a single PR. Close or supersede the other filter PRs.
4. **Unify struct/VOD/EPG/debug:** One branch that has UnmarshalJSON, VOD/EPG/startup fixes, and (optionally) debug/cache/advanced parsing/m3u4u. Prefer one of Yagoor + jtdevops or michbeck100, or a new branch that cherry-picks/consolidates these. Merge one PR; close the other two for this set of features.
5. **Merge** chernandezweb_master’s unique parts (XMLTV cache/retry, Range, README, docker-compose) either as one PR or split; resolve vendor overlap with the chosen struct branch.
6. **Merge** ridgarou’s get.php “play” fix and (if not already in the chosen filter PR) resolution groups and replacement in a coordinated way with the filter PR.
7. **Revert** fork-specific bits (module path, author changes, fork CI) in every merged branch before merging to master.

This comparison and the per-PR improvement lists in each PR body should be enough to merge selectively and avoid duplicate or conflicting changes.

---

## 5. What we are NOT bringing in (and why)

| Item | Reason |
|------|--------|
| **Any CI/CD** | User requested no CI from any branch. |
| **Module path changes** | Keep upstream path (e.g. `github.com/pierre-emmanuelJ/iptv-proxy`). |
| **Author/attribution change** (pierre → ridgarou) | Keep original attribution. |
| **README overhaul / docker-compose template** (chernandezweb) | Only code features brought in; docs separate. |
| **Trivial "." commits** (chernandezweb) | Not brought in. |
| **Duplicate struct/VOD/EPG/startup** (jtdevops, michbeck100) | Covered by PR #10 (Yagoor). |
| **Duplicate regex-only** (Gibby_regex-filters, michbeck100) | PR #12 uses ridgarou’s full implementation. |
| **Full response-saving to files** (jtdevops) | PR #15 adds config/flags only; wiring deferred. |
| **m3u4u.com URL option** (jtdevops, michbeck100) | Deferred; can be a follow-up. |
| **509 response body persistence** (chernandezweb) | Not in PR #13. |
| **error_utils / error details** (jtdevops) | Deferred; would require pkg/utils and handler changes. |
