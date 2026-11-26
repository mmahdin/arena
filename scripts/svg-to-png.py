"""Rasterise an SVG at a chosen width (height follows the viewBox)."""
import sys
import resvg_py

src, dst = sys.argv[1], sys.argv[2]
width = int(sys.argv[3]) if len(sys.argv) > 3 else 780
raw = open(src, encoding='utf-8').read()
vb = None
if 'viewBox="' in raw:
    vb = raw.split('viewBox="', 1)[1].split('"', 1)[0].split()
if vb and len(vb) == 4:
    height = round(width * float(vb[3]) / float(vb[2]))
else:
    height = round(width * 844 / 390)
data = resvg_py.svg_to_bytes(svg_path=src, width=width, height=height)
open(dst, 'wb').write(bytes(data))
print('ok', dst, width, height)
