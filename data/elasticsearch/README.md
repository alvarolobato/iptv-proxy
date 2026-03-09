# Elasticsearch Index Mappings

These files contain the index template + data stream definitions for iptv-proxy statistics.
Copy the content of each file and paste it directly into **Kibana Dev Console** (Stack Management → Dev Tools).

## Data streams

| File | Data stream name | Type | Purpose |
|---|---|---|---|
| `iptv.sessions.json` | `iptv.sessions` | Standard data stream | Raw session lifecycle events (`session_start`, `session_end`, `session_error`) |
| `metrics-iptv.channel_metrics.json` | `metrics-iptv.channel_metrics` | **TSDB data stream** | Per-channel per-minute aggregates |
| `iptv.user_history.json` | `iptv.user_history` | Standard data stream | Completed sessions per user |

**Naming convention:**
- `metrics-iptv.channel_metrics` uses the `metrics-*` prefix required by Elasticsearch serverless for TSDB data streams.
- `iptv.sessions` and `iptv.user_history` are plain event-log data streams and do not need the `metrics-*` prefix.

Replace `iptv` with your `--es-index-prefix` value (e.g. `iptv.sessions` → `myprefix.sessions`, `metrics-iptv.channel_metrics` → `metrics-myprefix.channel_metrics`).

## TSDB — channel_metrics

`metrics-iptv.channel_metrics` uses `index.mode: time_series` (TSDB).

Each unique combination of **dimension** values defines a separate time series:

| Dimension field | Description |
|---|---|
| `channel_id` | Canonical channel identifier (tvg-name) |
| `channel_name` | Channel display name |
| `channel_group` | Group / category |
| `channel_type` | `live`, `movie`, `series`, `m3u` |

**Metric fields** (can be used with `rate()` in ES|QL and Kibana Lens):

| Field | Metric type | Description |
|---|---|---|
| `session_count` | gauge | Sessions started in this minute |
| `active_sessions` | gauge | Concurrent active sessions |
| `unique_users` | gauge | Distinct users watching |
| `total_duration_seconds` | counter | Cumulative watch seconds |
| `bytes_transferred` | counter | Cumulative bytes proxied |
| `error_count` | counter | Stream errors |

## Recreate all data streams

Run each file in Kibana Dev Console in order:

```
# 1. iptv.sessions.json                (PUT template + PUT _data_stream)
# 2. metrics-iptv.channel_metrics.json (PUT template + PUT _data_stream)
# 3. iptv.user_history.json            (PUT template + PUT _data_stream)
```

## Delete and recreate

```
DELETE _data_stream/iptv.sessions
DELETE _data_stream/metrics-iptv.channel_metrics
DELETE _data_stream/iptv.user_history

DELETE _index_template/iptv.sessions
DELETE _index_template/metrics-iptv.channel_metrics
DELETE _index_template/iptv.user_history

# Then paste each .json file
```

> **Note:** The proxy creates these data streams automatically at startup when `--es-url` is set.
> Use these files to recreate them manually or provision a new environment.
