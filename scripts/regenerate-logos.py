#!/usr/bin/env python3
"""
Regenerate logo PNGs and favicon from images/iptv-proxy.png with max quality
(Lanczos resampling, aspect ratio preserved).
"""
import shutil
from pathlib import Path

from PIL import Image

ROOT = Path(__file__).resolve().parent.parent
SRC = ROOT / "images" / "iptv-proxy.png"
IMAGES = ROOT / "images"
PUBLIC = ROOT / "web" / "frontend" / "public"

# Lanczos = highest quality resampling for downscaling
RESAMPLING = getattr(Image, "Resampling", Image).LANCZOS

SIZES = [16, 32, 64, 128, 256]


def main():
    im = Image.open(SRC).convert("RGBA")
    for size in SIZES:
        out = im.resize((size, size), RESAMPLING)
        path = IMAGES / f"logo-{size}.png"
        out.save(path, "PNG", optimize=False)
        print(path)
    favicon_32 = im.resize((32, 32), RESAMPLING)
    favicon_32.save(IMAGES / "favicon-32.png", "PNG", optimize=False)
    favicon_32.save(IMAGES / "favicon.ico", format="ICO", sizes=[(32, 32)])
    # Copy to frontend public
    PUBLIC.mkdir(parents=True, exist_ok=True)
    for name in ["logo-128.png", "favicon-32.png", "favicon.ico"]:
        shutil.copy2(IMAGES / name, PUBLIC / name)
    print("Done. All logos and favicon regenerated (Lanczos, max quality).")


if __name__ == "__main__":
    main()
