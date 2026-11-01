import React from 'react';
import { Mark, Back, Check, Info } from './Icons.jsx';

export const BRAND = 'رهراه';

export function Pill({ variant = 'dark', disabled, onClick, children, auto, className = '' }) {
  return (
    <button
      className={`pill ${variant} ${auto ? 'auto' : ''} ${className}`}
      disabled={disabled}
      onClick={onClick}
      type="button"
    >
      {children}
    </button>
  );
}

export function TopBar({ dark, back, onBack, right, hideBrand }) {
  return (
    <div className={`topbar ${dark ? 'on-dark' : ''}`}>
      {back && (
        <button className="iconbtn" onClick={onBack} aria-label="بازگشت" type="button">
          <Back />
        </button>
      )}
      {!hideBrand && (
        <span className="brand">
          <Mark size={27} sw={6} />
          {BRAND}
        </span>
      )}
      {right && <span className="iconbtn end" style={{ cursor: 'default' }}>{right}</span>}
    </div>
  );
}

/* 4-point progress: p in [0..3], nodes at 1,2,3 */
const NODE_AT = [0.35, 0.63, 0.91];
export function Stepper({ p = 0 }) {
  const frac = (p) => {
    if (p <= 0) return 0;
    if (p >= 3) return 1;
    const i = Math.floor(p);
    const t = p - i;
    const a = i === 0 ? 0 : NODE_AT[i - 1];
    const b = NODE_AT[i] ?? 1;
    return a + (b - a) * t;
  };
  return (
    <div className="stepper" aria-hidden>
      <div className="rail" />
      <div className="fill" style={{ width: `${frac(p) * 100}%` }} />
      {NODE_AT.map((x, i) => (
        <span key={i} className={`node ${p >= i + 1 ? 'done' : ''}`} style={{ insetInlineStart: `${x * 100}%` }}>
          {p >= i + 1 && <Check size={13} sw={3.4} />}
        </span>
      ))}
    </div>
  );
}

export function Options({ items, value, onChange, circles = true }) {
  return (
    <div className="opts">
      {items.map((it) => (
        <button
          key={it}
          type="button"
          className={`opt ${circles ? '' : 'plain'} ${value === it ? 'selected' : ''}`}
          onClick={() => onChange(it)}
        >
          {circles && <span className="radio"><i /></span>}
          <span className="lbl">{it}</span>
        </button>
      ))}
    </div>
  );
}

export function Note({ children }) {
  return (
    <div className="note">
      <Info />
      <div>{children}</div>
    </div>
  );
}

export function faNum(n) {
  return String(n).replace(/\d/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[d]);
}
