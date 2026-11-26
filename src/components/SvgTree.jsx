import React from 'react';

/**
 * Renders the plain element trees produced by src/art/* (emblem, dawn scene)
 * as React SVG nodes. Keeping the artwork in framework-agnostic builders lets
 * the app, the favicon generator and the QA rasterizer share one source.
 */
const toReactProps = (attrs = {}) => {
  const out = {};
  for (const [key, value] of Object.entries(attrs)) {
    if (key === 'class') { out.className = value; continue; }
    const camel = key.replace(/-([a-z])/g, (_, letter) => letter.toUpperCase());
    out[camel] = value;
  }
  return out;
};

export function renderNode(node, key = 0) {
  const children = (node.children || []).map((child, index) => renderNode(child, index));
  return React.createElement(node.tag, { ...toReactProps(node.attrs), key }, ...children);
}

export default function SvgTree({ tree, className, style, width, height, role, ariaLabel, focusable = 'false' }) {
  const base = toReactProps(tree.attrs);
  const svgProps = {
    ...base,
    className,
    style,
    width,
    height,
    role,
    'aria-label': ariaLabel,
    'aria-hidden': ariaLabel ? undefined : true,
    focusable,
  };
  return React.createElement(
    'svg',
    svgProps,
    ...(tree.children || []).map((child, index) => renderNode(child, index)),
  );
}
