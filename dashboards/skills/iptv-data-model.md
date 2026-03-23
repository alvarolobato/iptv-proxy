# IPTV-Proxy Elasticsearch Data Model

Field reference for all Elasticsearch indices written by iptv-proxy2. Use this when writing ES|QL queries or Lens aggregations in Kibana dashboards.

> **Index prefix:** The default prefix is `iptv`. If the proxy was started with `--es-index-prefix myprefix`, replace `iptv` with `myprefix` in all index names below.

---

## `{prefix}.sessions` — Session lifecycle events

**Type:** Regular data stream (not TSDB)
**Purpose:** Every session event — start, end, and error — is written here.

### Fields

| Field | Type | Description |
|---|---|---|
| `@timestamp` | date | When the event was recorded |
| `event_kind` | keyword | `session_start`, `session_end`, or `session_error` |
| `session_id` | keyword | UUID identifying the session across events |
| `duration_seconds` | long | Filled on `session_end` — watch time in seconds |
| `bytes_transferred` | long | Filled on `session_end` — bytes proxied |
| `client_ip` | keyword | IP address of the streaming client |
| `user_agent` | keyword | HTTP User-Agent header sent by the client |
| `error_message` | text | Filled on `session_error` — error description |
| `channel_id` | keyword | Unique channel identifier |
| `channel_name` | keyword | Human-readable channel name |
| `channel_group` | keyword | Channel group/category |
| `channel_type` | keyword | `live`, `movie`, or `series` |
| `channel_stream_id` | keyword | Provider stream ID |
| `user_name` | keyword | Authenticated proxy user |
| `proxy_mode` | keyword | `m3u` or `xtream` |

### Common ES|QL patterns

```esql
# Active sessions right now
FROM iptv.sessions
| WHERE @timestamp > NOW() - 5 minutes AND event_kind == "session_start"
| STATS active = COUNT(DISTINCT session_id)

# Sessions started over time
FROM iptv.sessions
| WHERE event_kind == "session_start"
| STATS sessions = COUNT(*) BY time_bucket = BUCKET(@timestamp, 20, ?_tstart, ?_tend)
| SORT time_bucket ASC

# Error rate by channel
FROM iptv.sessions
| WHERE event_kind IN ("session_end", "session_error")
| STATS
    total = COUNT(*),
    errors = COUNT(*) WHERE event_kind == "session_error"
  BY channel_name
| EVAL error_rate_pct = ROUND(errors * 100.0 / total, 1)
| SORT errors DESC
```

---

## `{prefix}.user_history` — Completed sessions per user

**Type:** Regular data stream (not TSDB)
**Purpose:** One document per completed session. Use for user-level analytics and leaderboards. Populated on `session_end`.

### Fields

| Field | Type | Description |
|---|---|---|
| `@timestamp` | date | Session start time |
| `session_id` | keyword | UUID of the session |
| `session_end_time` | date | When the session ended |
| `duration_seconds` | long | Watch time in seconds |
| `bytes_transferred` | long | Bytes proxied |
| `channel_id` | keyword | Unique channel identifier |
| `channel_name` | keyword | Human-readable channel name |
| `channel_group` | keyword | Channel group/category |
| `channel_type` | keyword | `live`, `movie`, or `series` |
| `channel_stream_id` | keyword | Provider stream ID |
| `user_name` | keyword | Authenticated proxy user |
| `client_ip` | keyword | IP of the streaming client |
| `user_agent` | keyword | HTTP User-Agent header |
| `proxy_mode` | keyword | `m3u` or `xtream` |

### Common ES|QL patterns

```esql
# User leaderboard — top watchers by total hours
FROM iptv.user_history
| STATS
    sessions = COUNT(*),
    total_hours = SUM(duration_seconds) / 3600.0,
    total_gb = SUM(bytes_transferred) / 1073741824.0,
    unique_channels = COUNT_DISTINCT(channel_id)
  BY user_name
| SORT total_hours DESC
| LIMIT 20

# Per-user channel preferences — top channels per user
FROM iptv.user_history
| STATS watch_hours = SUM(duration_seconds) / 3600.0 BY user_name, channel_name
| SORT watch_hours DESC

# Viewing heatmap — activity by hour of day × day of week
FROM iptv.user_history
| EVAL hour_of_day = DATE_EXTRACT("hour", @timestamp)
| EVAL day_of_week = DATE_EXTRACT("day_of_week", @timestamp)
| STATS sessions = COUNT(*) BY hour_of_day, day_of_week
| SORT day_of_week ASC, hour_of_day ASC

# Recent sessions across all users
FROM iptv.user_history
| KEEP @timestamp, user_name, channel_name, channel_group, duration_seconds, bytes_transferred
| SORT @timestamp DESC
| LIMIT 50
```

---

## `metrics-{prefix}.channel_metrics` — Per-channel per-minute aggregates

**Type:** TSDB data stream (`index.mode: time_series`)
**Purpose:** Pre-aggregated channel metrics written every minute. Suitable for efficient time-series charts with TSDB `TS` queries and `RATE()`.

> **Important:** Because this is a TSDB index, use the `TS` source command (Elasticsearch 9.2+) for counter fields. For Kibana 8.x use `FROM` with `MAX()` on gauge fields.

### Dimensions (uniquely identify a time series)

| Field | Type | Description |
|---|---|---|
| `channel_id` | keyword | Unique channel identifier |
| `channel_name` | keyword | Human-readable channel name |
| `channel_group` | keyword | Channel group/category |
| `channel_type` | keyword | `live`, `movie`, or `series` |

### Gauge metrics (point-in-time values)

| Field | Type | Description |
|---|---|---|
| `session_count` | long (gauge) | Sessions started in this minute |
| `active_sessions` | long (gauge) | Currently active sessions |
| `unique_users` | long (gauge) | Distinct users watching in this minute |

### Counter metrics (monotonically increasing totals)

| Field | Type | Description |
|---|---|---|
| `total_duration_seconds` | long (counter) | Cumulative watch seconds |
| `bytes_transferred` | long (counter) | Cumulative bytes proxied |
| `error_count` | long (counter) | Cumulative stream errors |

### Common ES|QL patterns

```esql
# Top channels by concurrent viewers (Kibana 8.x — gauge via FROM)
FROM metrics-iptv.channel_metrics
| STATS max_viewers = MAX(active_sessions) BY channel_name
| SORT max_viewers DESC
| LIMIT 10

# Channel activity timeline — active sessions over time (Kibana 8.x)
FROM metrics-iptv.channel_metrics
| STATS viewers = MAX(active_sessions) BY
    time_bucket = BUCKET(@timestamp, 20, ?_tstart, ?_tend),
    channel_name
| SORT time_bucket ASC

# Group distribution — total watch time by channel group (Kibana 8.x)
FROM metrics-iptv.channel_metrics
| STATS total_hours = SUM(total_duration_seconds) / 3600.0 BY channel_group
| SORT total_hours DESC

# Bandwidth over time (Kibana 8.x)
FROM metrics-iptv.channel_metrics
| STATS total_gb = SUM(bytes_transferred) / 1073741824.0 BY
    time_bucket = BUCKET(@timestamp, 20, ?_tstart, ?_tend)
| SORT time_bucket ASC

# Error rate per channel using TSDB RATE() (Elasticsearch 9.2+)
TS metrics-iptv.channel_metrics
| STATS error_rate = SUM(RATE(error_count)) BY
    time_bucket = BUCKET(@timestamp, 20, ?_tstart, ?_tend), channel_name
| SORT time_bucket ASC
```

---

## Index naming quick reference

| Index | Purpose | Default name |
|---|---|---|
| Sessions stream | All session events | `iptv.sessions` |
| User history | Completed sessions | `iptv.user_history` |
| Channel metrics | TSDB per-minute aggregates | `metrics-iptv.channel_metrics` |

Substitute `iptv` with `--es-index-prefix` value if non-default.

---

## Recommended dashboard data sources

| Dashboard need | Best index | Why |
|---|---|---|
| Current active streams | `{prefix}.sessions` | Real-time `session_start` events |
| User leaderboard / history | `{prefix}.user_history` | Pre-joined per-session records |
| Channel timelines, trends | `metrics-{prefix}.channel_metrics` | Efficient TSDB aggregates |
| Error analysis | `{prefix}.sessions` | Has `error_message` text field |
| Per-user filters, controls | `{prefix}.user_history` | `user_name` keyword field |
