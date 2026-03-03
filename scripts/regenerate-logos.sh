#!/usr/bin/env bash
# Regenerate logo PNGs and favicon from images/iptv-proxy.png (max quality, Lanczos).
# Requires: python3, Pillow (pip install Pillow)
set -e
cd "$(dirname "$0")/.."
python3 scripts/regenerate-logos.py
