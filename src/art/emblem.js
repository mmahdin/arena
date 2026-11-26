/**
 * Theia's emblem — «نشانهٔ تیا».
 *
 * The old identity depicted Theia, the Greek Titaness of sight and heavenly light,
 * as a woman. The redesigned identity renders her through her three mythic gifts:
 *
 *   1. Shamseh (شمسه)  — the Persian sunburst medallion. Theia is the mother of
 *      Helios; the radiating disc is her radiance, set in the same geometry that
 *      Persian illuminators used for the sun.
 *   2. The quiet eye  — Theia is the goddess of sight (thea, "to see"). The almond
 *      inside the ring is an unblinking, unhurried witness — seeing without judging.
 *   3. The pupil-sun  — the still point at the centre of both: the light and the
 *      gaze are one. Gold is her oldest attribute (Pindar, Isthmian 5).
 *
 * 16 rays (8 long, 8 short) keep the 8-fold Persian khatam rhythm. Everything is
 * generated from this one module so the React component, the favicon and the
 * install icons can never drift apart.
 */

const TAU = Math.PI * 2;

export const EMBLEM_VIEWBOX = '0 0 64 64';

const round = (n) => Number(n.toFixed(3));

export function emblemRays() {
  const rays = [];
  for (let i = 0; i < 16; i += 1) {
    const long = i % 2 === 0;
    // Start on the vertical axis so the star reads upright in every context.
    const angle = -Math.PI / 2 + (TAU * i) / 16;
    const r1 = 17.4;
    const r2 = long ? 27.2 : 22.6;
    const cos = Math.cos(angle);
    const sin = Math.sin(angle);
    rays.push({
      x1: round(32 + r1 * cos),
      y1: round(32 + r1 * sin),
      x2: round(32 + r2 * cos),
      y2: round(32 + r2 * sin),
      long,
    });
  }
  return rays;
}

export const EYE_PATH = 'M 22.3 32 Q 32 24.5 41.7 32 Q 32 39.5 22.3 32 Z';

/**
 * Build the emblem as a framework-agnostic tree of SVG elements.
 * `luminous` swaps flat currentColor strokes/fills for a gold gradient.
 */
export function buildEmblem({ luminous = false } = {}) {
  const ink = 'currentColor';
  const ray = (node) => ({ ...node, attrs: { ...node.attrs, stroke: luminous ? 'url(#theia-gold)' : ink } });
  const nodes = [];
  const children = [];

  if (luminous) {
    children.push({
      tag: 'defs',
      attrs: {},
      children: [
        {
          tag: 'radialGradient',
          attrs: { id: 'theia-pupil', cx: '0.5', cy: '0.45', r: '0.65' },
          children: [
            { tag: 'stop', attrs: { offset: '0', 'stop-color': '#fff6dc' }, children: [] },
            { tag: 'stop', attrs: { offset: '0.6', 'stop-color': '#f8cf92' }, children: [] },
            { tag: 'stop', attrs: { offset: '1', 'stop-color': '#e5ab5e' }, children: [] },
          ],
        },
        {
          tag: 'radialGradient',
          attrs: { id: 'theia-gold', cx: '0.5', cy: '0.42', r: '0.72' },
          children: [
            { tag: 'stop', attrs: { offset: '0', 'stop-color': '#ffe9b8' }, children: [] },
            { tag: 'stop', attrs: { offset: '0.45', 'stop-color': '#f2c078' }, children: [] },
            { tag: 'stop', attrs: { offset: '1', 'stop-color': '#c08a44' }, children: [] },
          ],
        },
      ],
    });
  }

  for (const r of emblemRays()) {
    children.push(ray({
      tag: 'line',
      attrs: {
        x1: r.x1, y1: r.y1, x2: r.x2, y2: r.y2,
        'stroke-width': r.long ? 2.15 : 1.5,
        'stroke-linecap': 'round',
        opacity: r.long ? 1 : 0.82,
      },
      children: [],
    }));
  }

  children.push({
    tag: 'circle',
    attrs: {
      cx: 32, cy: 32, r: 12.6, fill: 'none',
      stroke: luminous ? 'url(#theia-gold)' : ink,
      'stroke-width': 2.45,
    },
    children: [],
  });
  children.push({
    tag: 'path',
    attrs: {
      d: EYE_PATH, fill: 'none',
      stroke: luminous ? 'url(#theia-gold)' : ink,
      'stroke-width': 2.15,
      'stroke-linejoin': 'round',
      'stroke-linecap': 'round',
    },
    children: [],
  });
  children.push({
    tag: 'circle',
    attrs: { cx: 32, cy: 32, r: 4.55, fill: luminous ? 'url(#theia-pupil)' : ink },
    children: [],
  });

  nodes.push({ tag: 'g', attrs: {}, children });
  return { tag: 'svg', attrs: { viewBox: EMBLEM_VIEWBOX }, children: nodes };
}

/** Serialize the emblem tree to an SVG string (used for icons and QA renders). */
export function emblemToSVGString(tree, { size, title, background } = {}) {
  const esc = (value) => String(value).replace(/&/g, '&amp;').replace(/"/g, '&quot;').replace(/</g, '&lt;');
  const attrs = (node) => {
    const parts = [];
    const all = { ...node.attrs };
    if (node.tag === 'svg') {
      if (size) { all.width = size; all.height = size; }
      all.xmlns = 'http://www.w3.org/2000/svg';
    }
    for (const [key, value] of Object.entries(all)) parts.push(`${key}="${esc(value)}"`);
    return parts.join(' ');
  };
  const render = (node) => {
    if (node.text) return esc(node.text);
    const inner = (node.children || []).map(render).join('');
    if (!inner && node.tag !== 'stop') return `<${node.tag} ${attrs(node)}/>`;
    return `<${node.tag} ${attrs(node)}>${inner}</${node.tag}>`;
  };
  const inner = tree.children.map(render).join('');
  const viewBox = tree.attrs.viewBox;
  const bg = background ? `<rect x="0" y="0" width="64" height="64" fill="${background}"/>` : '';
  const titleTag = title ? `<title>${esc(title)}</title>` : '';
  return `<svg xmlns="http://www.w3.org/2000/svg" viewBox="${viewBox}"${size ? ` width="${size}" height="${size}"` : ''}>${titleTag}${bg}${inner}</svg>`;
}

/**
 * Dark launch tile (favicon / install icons). The emblem sits inside the
 * maskable safe zone on a near-black ground with a faint warm halo — the same
 * "light held inside the dark" idea as the onboarding scene.
 */
export function buildEmblemTile({ cornerRadius = 0 } = {}) {
  const emblem = buildEmblem({ luminous: true });
  return {
    tag: 'svg',
    attrs: { viewBox: '0 0 64 64' },
    children: [
      {
        tag: 'defs',
        attrs: {},
        children: [
          {
            tag: 'radialGradient',
            attrs: { id: 'tile-halo', cx: '0.5', cy: '0.44', r: '0.55' },
            children: [
              { tag: 'stop', attrs: { offset: '0', 'stop-color': '#f6cf98', 'stop-opacity': 0.22 }, children: [] },
              { tag: 'stop', attrs: { offset: '0.55', 'stop-color': '#f6cf98', 'stop-opacity': 0.05 }, children: [] },
              { tag: 'stop', attrs: { offset: '1', 'stop-color': '#f6cf98', 'stop-opacity': 0 }, children: [] },
            ],
          },
        ],
      },
      { tag: 'rect', attrs: { x: 0, y: 0, width: 64, height: 64, rx: cornerRadius, fill: '#12110d' }, children: [] },
      { tag: 'rect', attrs: { x: 0, y: 0, width: 64, height: 64, rx: cornerRadius, fill: 'url(#tile-halo)' }, children: [] },
      { tag: 'g', attrs: { transform: 'translate(32 32) scale(0.72) translate(-32 -32)' }, children: emblem.children },
    ],
  };
}
