"""Extract the reusable photography from the supplied references.

Historical note (2026-11-08): this script used to cut the Ash-inspired floral
illustrations out of the reference images. That identity was replaced by the
original Kenar identity, whose artwork is vector and lives in
`src/brand/kenar.jsx` — nothing has to be extracted for it. Only the expert
portraits are still prepared here; the floral calls were removed on purpose.
"""
from pathlib import Path
from PIL import Image
import math

ROOT = Path(__file__).resolve().parents[1]
OUT = ROOT / "public" / "assets"
OUT.mkdir(parents=True, exist_ok=True)


def source(number: int) -> Image.Image:
    return Image.open(ROOT / "src" / f"photo_{number}_2026-11-01_11-26-16.jpg").convert("RGB")


def artwork(number: int, box: tuple[int, int, int, int], background: tuple[int, int, int], name: str) -> None:
    image = source(number).crop(box)
    pixels = []
    for red, green, blue in image.getdata():
        distance = math.sqrt((red-background[0])**2 + (green-background[1])**2 + (blue-background[2])**2)
        if distance <= 9:
            alpha = 0
        elif distance >= 43:
            alpha = 255
        else:
            alpha = round((distance - 9) / 34 * 255)
        pixels.append((red, green, blue, alpha))
    result = Image.new("RGBA", image.size)
    result.putdata(pixels)
    result.save(OUT / name, optimize=True)


source(35).crop((0, 680, 480, 1790)).save(OUT / "expert-man.jpg", quality=92, optimize=True)
source(35).crop((514, 680, 1080, 1660)).save(OUT / "expert-woman.jpg", quality=92, optimize=True)
