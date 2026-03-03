#!/usr/bin/env bash
# Regenerate logo PNGs from images/iptv-proxy.png preserving aspect ratio (-Z = fit longest side).
set -e
SRC="images/iptv-proxy.png"
for size in 16 32 64 128 256; do
  sips -Z "$size" "$SRC" --out "images/logo-${size}.png"
done
sips -Z 32 "$SRC" --out "images/favicon-32.png"
# Copy assets used by frontend
cp images/logo-128.png web/frontend/public/logo-128.png
cp images/favicon-32.png web/frontend/public/favicon-32.png
# Keep existing favicon.ico in images/ and public/ (sips doesn't write ico)
echo "Done. Logo PNGs regenerated (aspect ratio preserved)."
