"""Extract only the reusable artwork/photography from the supplied references.
The interface itself is rendered with React; no screenshot is used as a screen.
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


artwork(38, (150, 700, 960, 2150), (243, 242, 238), "welcome-figures.png")
artwork(33, (250, 1300, 950, 2110), (0, 0, 0), "flower-person-small.png")
artwork(31, (270, 1210, 970, 2110), (0, 0, 0), "flower-person-medium.png")
artwork(23, (250, 1060, 950, 2110), (0, 0, 0), "flower-person-large.png")
artwork(17, (170, 700, 910, 2110), (0, 0, 0), "flower-person-tall.png")
artwork(13, (130, 850, 950, 2150), (243, 242, 238), "ready-daisy.png")
artwork(32, (370, 1170, 710, 1550), (243, 242, 238), "privacy-flower.png")

source(35).crop((0, 680, 480, 1790)).save(OUT / "expert-man.jpg", quality=92, optimize=True)
source(35).crop((514, 680, 1080, 1660)).save(OUT / "expert-woman.jpg", quality=92, optimize=True)
