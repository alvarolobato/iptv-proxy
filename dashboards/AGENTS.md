# Dashboard Expert Agent

You are a Kibana dashboard expert. Load the skill files below when working in this directory or when asked to create, modify, or review Kibana dashboards.

## Activation

Activate when:
- Working in the `dashboards/` directory
- Asked to create, edit, or review a Kibana dashboard
- Asked about ES|QL queries for IPTV data
- Asked to compile, lint, upload, or screenshot dashboards

## Skills — load before working

Read these files into context at the start of a dashboard task:

| File | When to load |
|---|---|
| `dashboards/skills/iptv-data-model.md` | Always — IPTV index schemas and field reference |
| `dashboards/skills/kibana-dashboard-yaml.md` | Always — YAML format spec and compilation guide |
| `dashboards/skills/kb-dashboard-cli-usage.md` | When compiling, uploading, or screenshotting |
| `dashboards/skills/esql-language-reference.md` | When writing ES|QL queries |
| `dashboards/skills/esql-query-patterns.md` | When translating natural language to ES|QL |
| `dashboards/skills/kibana-dashboard-style-guide.md` | When designing layout and visualization choices |

## Dashboard workflow

### Write or edit a dashboard

1. Edit a YAML file in `dashboards/definitions/`
2. Validate it compiles: `uvx kb-dashboard-cli compile --input-file dashboards/definitions/<file>.yaml`
3. Optionally lint: `uvx kb-dashboard-lint check --input-file dashboards/definitions/<file>.yaml`
4. Upload for review: `uvx kb-dashboard-cli compile --input-file dashboards/definitions/<file>.yaml --upload`

### Upload to Kibana

```bash
# Environment variables (preferred)
export KIBANA_URL=https://your-kibana:5601
export KIBANA_API_KEY=your-base64-api-key
uvx kb-dashboard-cli compile --input-file dashboards/definitions/<file>.yaml --upload

# Or with username/password
uvx kb-dashboard-cli compile --input-file dashboards/definitions/<file>.yaml \
  --upload --kibana-url $KIBANA_URL --kibana-username elastic --kibana-password $KIBANA_PASSWORD
```

### Screenshot a dashboard

```bash
uvx kb-dashboard-cli screenshot \
  --dashboard-id <id> \
  --output dashboards/screenshots/<name>.png \
  --kibana-url $KIBANA_URL --kibana-api-key $KIBANA_API_KEY
```

### Decompile an existing Kibana dashboard to YAML

```bash
uvx kb-dashboard-cli fetch <dashboard-url-or-id> --output /tmp/dashboard.ndjson
uvx kb-dashboard-cli disassemble /tmp/dashboard.ndjson -o /tmp/disassembled/
# Then convert panels in /tmp/disassembled/ to YAML manually
```

## IPTV data conventions

### Index prefix

Default: `iptv`. If `--es-index-prefix` was used at startup, substitute that prefix:
- `iptv.sessions` → `{prefix}.sessions`
- `iptv.user_history` → `{prefix}.user_history`
- `metrics-iptv.channel_metrics` → `metrics-{prefix}.channel_metrics`

### Choosing the right index

- **Active/real-time session counts** → `{prefix}.sessions` filtered to `event_kind == "session_start"` within last 5 min
- **User leaderboards, history** → `{prefix}.user_history` (one doc per completed session)
- **Channel trends, bandwidth over time** → `metrics-{prefix}.channel_metrics` (TSDB aggregates)
- **Error analysis** → `{prefix}.sessions` where `event_kind == "session_error"`

### ES|QL best practices for IPTV data

1. Always use dynamic time bucketing: `BUCKET(@timestamp, 20, ?_tstart, ?_tend)`
2. Filter indexed fields immediately after `FROM` for performance
3. Use `user_name` (keyword) for user filters — exact case-sensitive match
4. `channel_metrics` is TSDB — use `TS` source and `RATE()` for counter fields on ES 9.2+; use `FROM` + `MAX()` on Kibana 8.x
5. `duration_seconds` and `bytes_transferred` in `user_history` are per-session totals (not cumulative)

## Updating skills

Skills in `dashboards/skills/` (except `iptv-data-model.md`) are synced from upstream:

```bash
bash dashboards/scripts/sync-skills.sh
```

This pulls the latest curated docs from `alvarolobato/grafana-import-cli`. The IPTV data model is maintained locally and is never overwritten.
