"""Compose .design/spec.json screens into phone-sized QA stills.

Reviewer aid, not a browser. It emits ONE SVG per screen — the real scene art,
the real wash, the real palette and the real box model from src/onboarding.css
(22px gutter, 52px chrome, 56px button, bottom-anchored caption) — and lets resvg
shape the Persian. Line breaking comes from HarfBuzz advances.
"""
import json, os, html
import resvg_py
import uharfbuzz as hb
from PIL import Image

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
W, H, PAD = 390, 844, 22
FONT_FILES = [os.path.expanduser('~/.local/share/fonts/Estedad.ttf')]

_blob = hb.Blob.from_file_path(FONT_FILES[0])
_face = hb.Face(_blob)
_font = hb.Font(_face)
UPEM = _face.upem


def advance(text):
    buf = hb.Buffer()
    buf.add_str(text)
    buf.guess_segment_properties()
    hb.shape(_font, buf)
    return sum(p.x_advance for p in buf.glyph_positions) / UPEM


def wrap(text, size, width):
    """Greedy wrap measured in ems, exactly like a browser's line breaker."""
    if not text:
        return []
    limit = width / size
    words = text.split()
    lines, line = [], ''
    for word in words:
        cand = (line + ' ' + word).strip()
        if advance(cand) <= limit or not line:
            line = cand
        else:
            lines.append(line)
            line = word
    if line:
        lines.append(line)
    return lines


def esc(t):
    return html.escape(t or '', quote=False)


def scene_inner(scene):
    src = os.path.join(ROOT, '.design/screens', f'{scene}.svg')
    if not os.path.exists(src):
        src = os.path.join(ROOT, '.design/screens/welcome.svg')
    raw = open(src, encoding='utf-8').read()
    i = raw.find('<defs>')
    if i < 0:
        i = raw.find('>') + 1
    return raw[i:raw.rindex('</svg>')]


def txt(x, y, s, size, color, weight=400, anchor='end', family='Estedad', opacity=1.0, spacing=0):
    if not s:
        return ''
    return (f'<text x="{x}" y="{y}" font-size="{size}" fill="{color}" fill-opacity="{opacity}" '
            f'direction="rtl" text-anchor="{anchor}" font-family="{family}" font-weight="{weight}" '
            f'letter-spacing="{spacing}">{esc(s)}</text>')


def paragraph(x, y, lines, size, color, lh, weight=400, anchor='end'):
    out = []
    for i, line in enumerate(lines):
        out.append(txt(x, y + i * size * lh, line, size, color, weight, anchor))
    return '\n'.join(out)


def compose(spec):
    night = spec['surface'] == 'night'
    frame = spec['frame']
    ink = '#f4f1ea' if night else '#1b1a17'
    soft = '#c3cad7' if night else '#615c53'
    faint = '#6f7a8c' if night else '#8b857a'
    gold = '#c8a878' if night else '#8e6534'

    if night:
        if frame == 'band':
            wash = [(0, .94), (.22, .82), (.44, .34), (.6, 0)]
            top = .30
        else:
            wash = [(0, .92), (.16, .74), (.36, .28), (.52, 0)]
            top = .34
        base = '#04060b'
    else:
        wash = [(0, .97), (.26, .9), (.5, .4), (.68, 0)]
        top = .30
        base = '#f3efe6'

    stops = '\n'.join(
        f'<stop offset="{o}" stop-color="{base}" stop-opacity="{a}"/>' for o, a in wash)
    s = [f'<svg xmlns="http://www.w3.org/2000/svg" width="{W}" height="{H}" viewBox="0 0 {W} {H}">']
    s.append('<defs>')
    s.append(f'<linearGradient id="wash" x1="0" y1="1" x2="0" y2="0">{stops}</linearGradient>')
    s.append(f'<radialGradient id="top" cx="50%" cy="0%" r="62%">'
             f'<stop offset="0" stop-color="{base}" stop-opacity="{top}"/>'
             f'<stop offset="1" stop-color="{base}" stop-opacity="0"/></radialGradient>')
    s.append('</defs>')
    s.append(f'<rect width="{W}" height="{H}" fill="{"#05070d" if night else "#f3efe6"}"/>')

    # the still, ghosted on paper screens
    s.append(f'<g>{"<!--scene-->"}{scene_inner(spec["scene"])}</g>')
    if not night:
        s.append(f'<rect width="{W}" height="{H}" fill="#f3efe6" opacity="0.7"/>')
    s.append(f'<rect width="{W}" height="{H}" fill="url(#wash)"/>')
    s.append(f'<rect width="{W}" height="{H}" fill="url(#top)"/>')

    body = []
    # ---- chrome ----
    cy = 34
    if spec['route'] != 'welcome':
        body.append(f'<g stroke="{ink}" stroke-width="1.8" fill="none" opacity="0.8" stroke-linecap="round" '
                    f'stroke-linejoin="round"><path d="M{W-40} {cy-5} L{W-33} {cy} L{W-40} {cy+5}"/></g>')
    else:
        body.append(f'<g opacity="0.62"><circle cx="34" cy="{cy}" r="9" fill="none" stroke="{ink}" stroke-width="0.9" '
                    f'opacity="0.4"/><path d="M34 {cy-9} A9 9 0 1 1 30.6 {cy+7.6}" fill="none" stroke="{ink}" '
                    f'stroke-width="2" stroke-linecap="round"/></g>')
    # the moon progress: a ring that fills as the disc becomes legible
    mx, my = W / 2 - 26, cy
    body.append(f'<g opacity="0.75">'
                f'<circle cx="{mx}" cy="{my}" r="9.4" fill="none" stroke="{ink}" stroke-opacity="0.18" stroke-width="1"/>'
                f'<circle cx="{mx}" cy="{my}" r="9.4" fill="none" stroke="{ink}" stroke-opacity="0.62" stroke-width="1.1" '
                f'stroke-linecap="round" stroke-dasharray="20 60" transform="rotate(-90 {mx} {my})"/>'
                f'<circle cx="{mx + 4.4}" cy="{my - 1.4}" r="4.3" fill="#9fb3cf" fill-opacity="0.5"/>'
                f'<path d="M{mx+2.8} {my-4.6} A4.3 4.3 0 1 0 {mx+2.8} {my+1.8} A3.9 3.9 0 1 1 {mx+2.8} {my-4.6} Z" '
                f'fill="{ink}" fill-opacity="0.95"/></g>')
    if spec.get('chapter'):
        body.append(txt(mx + 20, cy + 4, spec['chapter'], 11.5, soft, 430, 'start'))

    # ---- footer first, so the caption can be measured against it ----
    btn_y = H - 14 - 56
    footer = []
    if spec.get('primary'):
        footer.append(f'<rect x="{PAD}" y="{btn_y}" width="{W-PAD*2}" height="56" rx="16" '
                      f'fill="{"#f1ece2" if night else "#14150f"}"/>')
        footer.append(txt(W / 2, btn_y + 34, spec['primary'], 15.5,
                          '#14150f' if night else '#f5f0e7', 520, 'middle'))
    if spec.get('skip'):
        footer.append(txt(W / 2, btn_y + 80, spec['skip'], 13, soft, 430, 'middle'))
    disclosure_lines = wrap(spec.get('disclosure', ''), 11.5, W - PAD * 2 - 24)

    bottom = btn_y - 14
    if disclosure_lines:
        bottom -= len(disclosure_lines) * 11.5 * 1.85 + 12
        for i, line in enumerate(disclosure_lines):
            footer.append(txt(W / 2, bottom + 12 + i * 11.5 * 1.85 + 9, line, 11.5, faint, 400, 'middle'))

    # ---- caption, bottom-anchored ----
    title_size = 27 if night else 26
    epi_size, lede_size, chap_size = 14.5, 14.5, 12
    parts = []
    if spec.get('epigraph'):
        parts.append(('epi', epi_size, 1.9, gold, golden_epi := 400, wrap(spec['epigraph'], epi_size, W - 2 * PAD)))
    if spec.get('title'):
        parts.append(('title', title_size, 1.62, ink, 470, wrap(spec['title'], title_size, W - 2 * PAD)))
    if spec.get('lede'):
        parts.append(('lede', lede_size, 2.02, soft, 380, wrap(spec['lede'], lede_size, W - 2 * PAD)))
    if spec.get('chapter'):
        parts.append(('chapter', chap_size, 1.0, gold, 430, [spec['chapter']]))

    gaps = {'epi': 11, 'title': 13, 'chapter': 26, 'lede': 0}
    total = sum(len(p[5]) * p[1] * p[2] for p in parts) + sum(gaps[p[0]] for p in parts[:-1])
    y = bottom - total
    top_copy = y
    header_parts = [p for p in parts if p[0] in ('epi', 'title', 'lede')] if spec.get('moodMoon') else []
    parts = [p for p in parts if p not in header_parts]
    if header_parts:
        hy = 96
        for kind, size, lh, color, weight, lines in header_parts:
            body.append(paragraph(W - PAD, hy + size, lines, size, color, lh, weight))
            hy += len(lines) * size * lh + 13

    for kind, size, lh, color, weight, lines in parts:
        if kind == 'chapter':
            body.append(f'<path d="M{W-PAD-150} {y+8} L{W-PAD-140} {y+8}" stroke="{gold}" stroke-opacity="0.5"/>')
            body.append(txt(W - PAD - 150, y + 11, lines[0], size, color, weight, 'start'))
            y += size + gaps['chapter']
        else:
            body.append(paragraph(W - PAD, y + size, lines, size, color, lh, weight))
            y += len(lines) * size * lh + gaps[kind]

    # ---- interactive blocks, immediately above the caption ----
    form_top = top_copy - 28
    if spec.get('choices'):
        labels = spec['choices'][:5]
        chip_h, gap = 52, 9
        start = form_top - len(labels) * (chip_h + gap)
        for i, label in enumerate(labels):
            yy = start + i * (chip_h + gap)
            body.append(f'<rect x="{PAD}" y="{yy}" width="{W-2*PAD}" height="{chip_h}" rx="14" '
                        f'fill="{"#0c111b" if night else "#fbf8f1"}" fill-opacity="{0.62 if night else 1}" '
                        f'stroke="{"#bacbe2" if night else "#ded4c3"}" stroke-opacity="{0.15 if night else 1}"/>')
            body.append(txt(W - PAD - 17, yy + 32, label, 15, '#e7e9ee' if night else '#23211c', 420))

    if spec.get('ageOptions'):
        labels = spec['ageOptions']
        start = form_top - len(labels) * 48
        for i, label in enumerate(labels):
            active = label == '۲۵–۲۹'
            yy = start + i * 48
            body.append(txt(W / 2, yy + 6, label, 20 if active else 18,
                            ink if active else soft, 500 if active else 400, 'middle',
                            opacity=1 if active else 0.55))
            if active:
                body.append(f'<path d="M{W/2-26} {yy+15} L{W/2+26} {yy+15}" stroke="{gold}" stroke-opacity="0.7"/>')

    if spec.get('privacyCard'):
        yy = form_top - 56
        body.append(f'<rect x="{PAD}" y="{yy}" width="{W-2*PAD}" height="56" rx="14" '
                    f'fill="{"#0c111b" if night else "#fbf8f1"}" fill-opacity="{0.68 if night else 1}" '
                    f'stroke="{"#bacbe2" if night else "#ded4c3"}" stroke-opacity="{0.15 if night else 1}"/>')
        body.append(f'<rect x="{W-PAD-18-46}" y="{yy+14}" width="46" height="27" rx="13.5" fill="#d7b284"/>')
        body.append(f'<circle cx="{W-PAD-18-23}" cy="{yy+27.5}" r="10.5" fill="#16150f"/>')
        body.append(txt(W - PAD - 76, yy + 33, 'اجازهٔ استفاده از گفت‌وگوها', 14, '#e7e9ee' if night else '#23211c', 420))
    if spec.get('privacy'):
        body.append(paragraph(W - PAD, bottom - 4, wrap(spec['privacy'], 12, W - 2 * PAD), 12, faint, 1.8, 400))

    if spec.get('namePlaceholder'):
        body.append(txt(W - PAD, form_top - 132, 'اسمت', 11.5, faint, 430))
        body.append(txt(W - PAD, form_top - 92, spec['namePlaceholder'], 26, faint, 400))
        body.append(f'<path d="M{PAD} {form_top-66} L{W-PAD} {form_top-66}" stroke="{ink}" stroke-opacity="0.24"/>')

    if spec.get('footnote'):
        body.append(paragraph(W - PAD, btn_y - 26, wrap(spec['footnote'], 13, W - 2 * PAD), 13, soft, 1.85, 400))
    if spec.get('moodMoon'):
        # .mood-stage is a centred flex column between the header and the footnote
        stage_top, stage_bottom = 300, btn_y - 60
        moon_h = min(280, int((stage_bottom - stage_top) * 0.62))
        moon_w = int(moon_h * 300 / 286)
        mx = (W - moon_w) / 2
        my = stage_top + max(0, (stage_bottom - stage_top - moon_h - 96) / 2)
        body.append(f'<g transform="translate({mx} {my}) scale({moon_w / 300})">{spec["moodMoon"]}</g>')
        if spec.get('moodLabel'):
            body.append(txt(W / 2, my + moon_h + 20, spec['moodLabel'], 18.5, ink, 460, 'middle'))
            body.append(txt(W / 2, my + moon_h + 40, spec.get('moodSea', ''), 11.5, gold, 430, 'middle'))
        # the slider, then the footnote
        sy = my + moon_h + 58
        body.append(f'<path d="M{W/2-104} {sy+17} L{W/2+104} {sy+17}" stroke="{ink}" stroke-opacity="0.3" stroke-width="1"/>')
        for k in range(5):
            x = W / 2 - 104 + k * 52
            on = k == spec.get('moodLevel', 2)
            body.append(f'<path d="M{x} {sy+11} L{x} {sy+23}" stroke="{gold if on else "#94a0b2"}" stroke-opacity="{0.95 if on else 0.4}" stroke-width="1"/>')
        thumb = W / 2 - 104 + spec.get('moodLevel', 2) * 52
        body.append(f'<circle cx="{thumb}" cy="{sy+17}" r="14" fill="url(#thumb)"/>')
        if spec.get('footnote'):
            body.append(paragraph(W - PAD, btn_y - 30, wrap(spec['footnote'], 13, W - 2 * PAD), 13, soft, 1.85, 400))
        body.insert(0, '<defs><radialGradient id="thumb" cx="38%" cy="32%"><stop offset="0" stop-color="#fdf6e8"/><stop offset="0.62" stop-color="#d7b284"/><stop offset="1" stop-color="#a67c4c"/></radialGradient></defs>')

    s.extend(body)
    s.extend(footer)
    s.append('</svg>')
    return '\n'.join(s)


specs = json.load(open(os.path.join(ROOT, '.design/spec.json'), encoding='utf-8'))
os.makedirs(os.path.join(ROOT, '.design/qa'), exist_ok=True)
for spec in specs:
    svg = compose(spec)
    path = os.path.join(ROOT, '.design/qa', f"{spec['route']}.svg")
    open(path, 'w', encoding='utf-8').write(svg)
    data = resvg_py.svg_to_bytes(svg_path=path, width=W * 2, height=H * 2,
                                 font_files=FONT_FILES, font_family='Estedad',
                                 skip_system_fonts=True, languages=['fa'])
    png = os.path.join(ROOT, '.design/qa', f"{spec['route']}.png")
    open(png, 'wb').write(bytes(data))
    Image.open(png).convert('RGB').resize((W, H), Image.LANCZOS).save(png)
    print('qa', spec['route'])
