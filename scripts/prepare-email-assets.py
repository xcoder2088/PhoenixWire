#!/usr/bin/env python3
"""Email-sized copies of approved logos. Requires Pillow only when refreshing assets."""
from pathlib import Path
from PIL import Image
root = Path(__file__).resolve().parent.parent
assets = root / 'internal/cards/assets'
for source, target, width in [
    (root / 'internal/web/static/brand/phoenix-wire.webp', 'email-wire.png', 322),
    (assets / 'phoenixmail-email-icon.png', 'email-icon.png', 60),
    (assets / 'phoenixmail-email-wordmark.png', 'email-wordmark.png', 231),
]:
    with Image.open(source) as image:
        image.thumbnail((width, 1000), Image.Resampling.LANCZOS)
        image.convert('RGBA').quantize(colors=128, method=Image.Quantize.FASTOCTREE).save(assets / target, optimize=True)
