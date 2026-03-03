#!/usr/bin/env bash
# Generate lower-resolution logo images and favicon from images/iptv-proxy.png
# Requires: sips (macOS) or ImageMagick (convert)
set -e
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
SRC="$REPO_ROOT/images/iptv-proxy.png"
OUT="$REPO_ROOT/images"

if [[ ! -f "$SRC" ]]; then
  echo "Source image not found: $SRC" >&2
  exit 1
fi

mkdir -p "$OUT"

# Resize with sips (macOS) or convert (ImageMagick)
resize() {
  local out=$1
  local size=$2
  if command -v sips &>/dev/null; then
    sips -z "$size" "$size" "$SRC" --out "$out"
  elif command -v convert &>/dev/null; then
    convert "$SRC" -resize "${size}x${size}" "$out"
  else
    echo "Need sips (macOS) or convert (ImageMagick)" >&2
    exit 1
  fi
}

# Logo variants
resize "$OUT/logo-16.png"  16
resize "$OUT/logo-32.png"  32
resize "$OUT/logo-64.png"  64
resize "$OUT/logo-128.png" 128
resize "$OUT/logo-256.png" 256

# Favicon: 32x32 as primary (browsers use it)
cp "$OUT/logo-32.png" "$OUT/favicon-32.png"

# favicon.ico: copy 32x32 PNG (some browsers accept PNG in .ico path)
cp "$OUT/logo-32.png" "$OUT/favicon.ico"

# Copy assets used by frontend (README uses images/ via repo root)
FRONTEND_PUBLIC="$REPO_ROOT/web/frontend/public"
if [[ -d "$(dirname "$FRONTEND_PUBLIC")" ]]; then
  mkdir -p "$FRONTEND_PUBLIC"
  cp "$OUT/favicon.ico" "$OUT/favicon-32.png" "$OUT/logo-128.png" "$FRONTEND_PUBLIC/"
  echo "Copied favicon and logo-128 to web/frontend/public/"
fi

echo "Generated: logo-16.png, logo-32.png, logo-64.png, logo-128.png, logo-256.png, favicon-32.png, favicon.ico"
