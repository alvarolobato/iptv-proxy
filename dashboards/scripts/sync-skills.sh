#!/usr/bin/env bash
# Sync skill markdown files from upstream repos into dashboards/skills/.
#
# Sources:
#   alvarolobato/grafana-import-cli — curated Kibana/ES|QL reference docs
#   (strawgate/kb-yaml-to-lens and elastic/agent-tools are private; grafana-import-cli
#    already contains the combined and curated versions of those docs.)
#
# Idempotent: safe to re-run. IPTV-specific files (iptv-data-model.md) are never
# overwritten by this script — edit them directly in dashboards/skills/.
#
# Requirements: gh (GitHub CLI authenticated), python3
# Run from the repo root or from dashboards/scripts/:
#   bash dashboards/scripts/sync-skills.sh

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"
SKILLS_DIR="$REPO_ROOT/dashboards/skills"
mkdir -p "$SKILLS_DIR"

GRAFANA_IMPORT_OWNER="alvarolobato"
GRAFANA_IMPORT_REPO="grafana-import-cli"

fetch_file() {
  local owner="$1" repo="$2" path="$3" dest="$4"
  local json content
  if json=$(gh api "repos/${owner}/${repo}/contents/${path}" 2>/dev/null); then
    content=$(python3 -c "
import json, sys, base64
d = json.load(sys.stdin)
raw = (d.get('content') or '').replace('\n', '')
sys.stdout.buffer.write(base64.b64decode(raw))
" <<< "$json")
    echo "$content" > "$dest"
    echo "  fetched ${path} -> $(basename "$dest")"
  else
    echo "  skipping ${owner}/${repo}/${path} (no access or not found)"
  fi
}

echo "==> Syncing skills from ${GRAFANA_IMPORT_OWNER}/${GRAFANA_IMPORT_REPO} ..."

SKILLS_SOURCE="skills"

fetch_file "$GRAFANA_IMPORT_OWNER" "$GRAFANA_IMPORT_REPO" \
  "${SKILLS_SOURCE}/kibana-dashboard-yaml.md"    "$SKILLS_DIR/kibana-dashboard-yaml.md"

fetch_file "$GRAFANA_IMPORT_OWNER" "$GRAFANA_IMPORT_REPO" \
  "${SKILLS_SOURCE}/esql-language-reference.md"  "$SKILLS_DIR/esql-language-reference.md"

fetch_file "$GRAFANA_IMPORT_OWNER" "$GRAFANA_IMPORT_REPO" \
  "${SKILLS_SOURCE}/esql-query-patterns.md"      "$SKILLS_DIR/esql-query-patterns.md"

fetch_file "$GRAFANA_IMPORT_OWNER" "$GRAFANA_IMPORT_REPO" \
  "${SKILLS_SOURCE}/kibana-dashboard-style-guide.md" "$SKILLS_DIR/kibana-dashboard-style-guide.md"

fetch_file "$GRAFANA_IMPORT_OWNER" "$GRAFANA_IMPORT_REPO" \
  "${SKILLS_SOURCE}/kb-dashboard-cli-usage.md"   "$SKILLS_DIR/kb-dashboard-cli-usage.md"

echo ""
echo "NOTE: dashboards/skills/iptv-data-model.md is IPTV-specific — not synced."
echo "Sync complete. Review changes with: git diff dashboards/skills/"
