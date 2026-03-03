#!/usr/bin/env bash
# Build IPTV-Proxy from source: frontend (UI) then Go binary.
# Run from repo root: ./scripts/build.sh

set -e

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
cd "$REPO_ROOT"

echo "==> Building configuration UI (web/frontend)..."
cd web/frontend
npm ci
npm run build
cd "$REPO_ROOT"

echo "==> Building Go binary..."
go build -o iptv-proxy .

echo "==> Done. Binary: $REPO_ROOT/iptv-proxy"
