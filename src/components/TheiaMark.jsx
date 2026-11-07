import React from 'react';

/**
 * Theia's original brand mark: two reflective halves opened by a thread of light.
 * It inherits `currentColor` so the existing light/dark/beige surfaces keep working.
 * The optional gold thread is reserved for larger, brand-forward moments.
 */
export default function TheiaMark({ size = 24, className = '', thread = false, title, ...props }) {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 64 64"
      className={`theia-mark ${className}`.trim()}
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : true}
      aria-label={title}
      focusable="false"
      {...props}
    >
      {title && <title>{title}</title>}
      <g fill="currentColor">
        <path d="M31 56a22 22 0 0 1-15.6-37.6l31.2 31.2A21.9 21.9 0 0 1 31 56Z" />
        <path d="M37 8a22 22 0 0 1 15.6 37.6L21.4 14.4A21.9 21.9 0 0 1 37 8Z" />
      </g>
      {thread && <path d="M21.8 19.8 44.2 42.2" fill="none" stroke="#d7b98a" strokeWidth="1.7" strokeLinecap="round" />}
    </svg>
  );
}
