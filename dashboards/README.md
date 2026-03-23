# IPTV-Proxy Kibana Dashboards

YAML-based Kibana dashboard definitions for iptv-proxy2 analytics. Dashboards are compiled to Kibana NDJSON using [`kb-yaml-to-lens`](https://github.com/strawgate/kb-yaml-to-lens).

## Quick start

```bash
# Compile a dashboard to NDJSON (dry run, no upload)
uvx kb-dashboard-cli compile --input-file dashboards/definitions/user-dashboard.yaml

# Compile and upload to Kibana
export KIBANA_URL=https://your-kibana:5601
export KIBANA_API_KEY=your-base64-api-key
uvx kb-dashboard-cli compile --input-file dashboards/definitions/user-dashboard.yaml --upload

# Lint for best practices
uvx kb-dashboard-lint check --input-file dashboards/definitions/user-dashboard.yaml
```

## Prerequisites

- Python 3.12+ with `uv`/`uvx` — see [uv docs](https://github.com/astral-sh/uv)
- Kibana 8.x or 9.x (for upload/screenshot)
- iptv-proxy2 running with `--es-url` configured (to have data)

## Folder structure

```
dashboards/
├── AGENTS.md                       # Dashboard expert agent instructions
├── README.md                       # This file
├── scripts/
│   └── sync-skills.sh              # Pull updated docs from upstream
├── skills/                         # Reference docs for AI agents
│   ├── iptv-data-model.md          # IPTV index schemas (maintained here)
│   ├── kibana-dashboard-yaml.md    # YAML format spec (synced from upstream)
│   ├── esql-language-reference.md  # ES|QL reference (synced from upstream)
│   ├── esql-query-patterns.md      # Query patterns (synced from upstream)
│   ├── kibana-dashboard-style-guide.md  # Layout guide (synced from upstream)
│   └── kb-dashboard-cli-usage.md   # CLI reference (synced from upstream)
└── definitions/
    ├── user-dashboard.yaml         # User leaderboard and activity dashboard
    └── channel-dashboard.yaml      # Channel metrics dashboard
```

## Available dashboards

### User Dashboard (`definitions/user-dashboard.yaml`)

Analytics focused on user activity:
- Active sessions (real-time count)
- Total unique users
- User leaderboard (top watchers by hours)
- User activity over time (stacked area)
- Recent sessions table
- Per-user channel preferences

**Data source:** `iptv.sessions`, `iptv.user_history`

### Channel Dashboard (`definitions/channel-dashboard.yaml`)

Channel performance metrics:
- Top channels by watch time
- Channel activity timeline
- Group distribution (pie)
- Error rate metric
- Bandwidth usage over time

**Data source:** `metrics-iptv.channel_metrics`, `iptv.sessions`

## Environment variables

```bash
export KIBANA_URL=https://your-kibana:5601

# API key (recommended)
export KIBANA_API_KEY=your-base64-api-key

# Or basic auth
export KIBANA_USERNAME=elastic
export KIBANA_PASSWORD=changeme

# Optional: target a specific Kibana space
export KIBANA_SPACE_ID=iptv
```

## Updating skill documentation

Skills in `dashboards/skills/` (except `iptv-data-model.md`) are sourced from [`alvarolobato/grafana-import-cli`](https://github.com/alvarolobato/grafana-import-cli). To pull the latest:

```bash
bash dashboards/scripts/sync-skills.sh
```

## Related

- Issue [#34](https://github.com/alvarolobato/iptv-proxy/issues/34) — Dashboard tooling epic
- Issue [#31](https://github.com/alvarolobato/iptv-proxy/issues/31) — Multi-user epic
- [`strawgate/kb-yaml-to-lens`](https://github.com/strawgate/kb-yaml-to-lens) — Dashboard compiler
- [`alvarolobato/grafana-import-cli`](https://github.com/alvarolobato/grafana-import-cli) — Skills source
