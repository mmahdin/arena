import React from 'react';

/* Brand mark — soft double-lobed outline (recreated placeholder mark) */
export const Mark = ({ size = 30, color = 'currentColor', sw = 5 }) => (
  <svg width={size} height={size} viewBox="0 0 100 100" fill="none" aria-hidden>
    <path
      d="M50 6c15 0 25 8 25 17 0 8-8 11-8 18 0 9 17 12 17 26 0 16-15 27-34 27S16 83 16 67c0-14 17-17 17-26 0-7-8-10-8-18C25 14 35 6 50 6Z"
      stroke={color} strokeWidth={sw} strokeLinejoin="round"
    />
  </svg>
);

export const Back = ({ size = 26 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.2" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M20 12H4" /><path d="m10 6-6 6 6 6" />
  </svg>
);

export const Check = ({ size = 14, sw = 3 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth={sw} strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="m4.5 12.5 5 5 10-11" />
  </svg>
);

export const Info = ({ size = 26 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.7" aria-hidden>
    <circle cx="12" cy="12" r="9.2" /><path d="M12 10.6v6" strokeLinecap="round" /><circle cx="12" cy="7.6" r="1.15" fill="currentColor" stroke="none" />
  </svg>
);

export const CheckCircle = ({ size = 26 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
    <circle cx="12" cy="12" r="9.2" /><path d="m8 12.4 2.7 2.7L16 9.6" strokeLinecap="round" strokeLinejoin="round" />
  </svg>
);

export const ChatCheck = ({ size = 76 }) => (
  <svg width={size} height={size} viewBox="0 0 48 48" fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinejoin="round" aria-hidden>
    <path d="M24 6c10.5 0 18 7.2 18 16.6S34.5 39.2 24 39.2c-2.2 0-4.4-.3-6.3-.9L9 42l1.6-7.4C7.7 31.7 6 27.6 6 22.6 6 13.2 13.5 6 24 6Z" />
    <path d="m17 22.6 4.6 4.6L31 17.6" strokeLinecap="round" />
  </svg>
);

export const Mic = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <rect x="9" y="2.6" width="6" height="11.2" rx="3" />
    <path d="M5.5 11.4a6.5 6.5 0 0 0 13 0" /><path d="M12 17.9v3.5" />
  </svg>
);

export const Menu = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.1" strokeLinecap="round" aria-hidden>
    <path d="M4 7h16M4 12h16M4 17h16" />
  </svg>
);

export const Sliders = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" aria-hidden>
    <path d="M8 4v6M8 14v6M16 4v10M16 18v2" />
    <circle cx="8" cy="12" r="2.1" /><circle cx="16" cy="16" r="2.1" />
  </svg>
);

export const Lock = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
    <path d="M12 2.8a4.6 4.6 0 0 0-4.6 4.6v2.4H6.6A1.6 1.6 0 0 0 5 11.4v8A1.6 1.6 0 0 0 6.6 21h10.8a1.6 1.6 0 0 0 1.6-1.6v-8a1.6 1.6 0 0 0-1.6-1.6h-.8V7.4A4.6 4.6 0 0 0 12 2.8Zm-2.4 7V7.4a2.4 2.4 0 1 1 4.8 0v2.4H9.6Z" />
  </svg>
);

export const Waveform = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden>
    <path d="M4 10v4M8 7v10M12 4v16M16 8v8M20 10.5v3" />
    <path d="M17.6 4.6 19.4 3M19.4 3v2.6M19.4 3h-2.5" strokeWidth="1.6" />
  </svg>
);

export const Sprout = ({ size = 44 }) => (
  <svg width={size} height={size} viewBox="0 0 48 48" fill="none" stroke="currentColor" strokeWidth="2.4" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M24 42V22" />
    <path d="M24 24c0-8-5-12-12-12 0 8 5 12 12 12Z" />
    <path d="M24 18c0-7 4.5-10.5 10.5-10.5C34.5 14.5 30 18 24 18Z" />
    <path d="M14 42h20" />
  </svg>
);

export const History = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" aria-hidden>
    <path d="M3.5 12a8.5 8.5 0 1 0 2.6-6.1L3.5 8.4" /><path d="M3.5 3.8v4.6h4.6" /><path d="M12 7.6V12l3 1.8" />
  </svg>
);

export const Stack = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinejoin="round" aria-hidden>
    <rect x="5" y="8.5" width="14" height="11" rx="2.4" />
    <path d="M7.5 5.5h9" strokeLinecap="round" /><path d="M9 2.9h6" strokeLinecap="round" />
  </svg>
);

export const Person = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" aria-hidden>
    <circle cx="12" cy="7.4" r="3.6" /><path d="M5 20c.8-4 3.6-6 7-6s6.2 2 7 6" />
  </svg>
);

export const Clock = ({ size = 18 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" aria-hidden>
    <circle cx="12" cy="12" r="8.6" /><path d="M12 7.4V12l3.2 2" />
  </svg>
);

export const Calendar = ({ size = 18 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" aria-hidden>
    <rect x="3.6" y="5" width="16.8" height="15" rx="2.6" /><path d="M3.6 9.8h16.8M8.4 3v4M15.6 3v4" />
  </svg>
);

export const X = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.1" strokeLinecap="round" aria-hidden>
    <path d="m5.5 5.5 13 13M18.5 5.5l-13 13" />
  </svg>
);

export const Dots = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
    <circle cx="5" cy="12" r="1.7" /><circle cx="12" cy="12" r="1.7" /><circle cx="19" cy="12" r="1.7" />
  </svg>
);

export const Eye = ({ size = 30 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" aria-hidden>
    <path d="M2.5 12S6 5.8 12 5.8 21.5 12 21.5 12 18 18.2 12 18.2 2.5 12 2.5 12Z" /><circle cx="12" cy="12" r="3" />
  </svg>
);

export const Camera = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" aria-hidden>
    <path d="M4 8.2h3l1.6-2.4h6.8L17 8.2h3a1.4 1.4 0 0 1 1.4 1.4v8A1.4 1.4 0 0 1 20 19H4a1.4 1.4 0 0 1-1.4-1.4v-8A1.4 1.4 0 0 1 4 8.2Z" /><circle cx="12" cy="13.2" r="3.2" />
  </svg>
);

export const Video = ({ size = 22 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinejoin="round" aria-hidden>
    <rect x="2.8" y="6.4" width="12.6" height="11.2" rx="2.4" /><path d="m15.4 10.6 5.8-3v8.8l-5.8-3" />
  </svg>
);

export const Google = ({ size = 24 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" aria-hidden>
    <path fill="#EA4335" d="M12 5.04c1.62 0 3.06.56 4.2 1.66l3.12-3.12C17.46 1.8 14.96.75 12 .75 7.62.75 3.84 3.27 2 6.94l3.66 2.84C6.54 7.02 9.04 5.04 12 5.04Z" />
    <path fill="#4285F4" d="M23.25 12.27c0-.92-.08-1.6-.26-2.31H12v4.51h6.44c-.13 1.08-.83 2.7-2.39 3.79l3.57 2.77c2.14-1.97 3.63-4.88 3.63-8.76Z" />
    <path fill="#FBBC05" d="M5.66 14.22a7.2 7.2 0 0 1 0-4.44L2 6.94a11.26 11.26 0 0 0 0 10.12l3.66-2.84Z" />
    <path fill="#34A853" d="M12 23.25c3.04 0 5.6-1 7.46-2.72l-3.57-2.77c-.96.67-2.25 1.14-3.89 1.14-2.96 0-5.46-1.98-6.34-4.68L2 17.06c1.84 3.67 5.62 6.19 10 6.19Z" />
  </svg>
);

export const Apple = ({ size = 26 }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" aria-hidden>
    <path d="M16.36 12.94c.03 3.02 2.65 4.03 2.68 4.04-.02.07-.42 1.44-1.39 2.85-.84 1.22-1.71 2.43-3.08 2.46-1.35.02-1.78-.79-3.32-.79-1.54 0-2.02.76-3.3.81-1.33.05-2.34-1.31-3.18-2.53C3.05 17.3 1.7 12.7 3.47 9.98c.88-1.35 2.45-2.2 4.16-2.23 1.3-.02 2.53.88 3.32.88.8 0 2.29-1.09 3.86-.93.66.03 2.51.27 3.7 2.01-.1.06-2.21 1.29-2.15 3.23ZM13.83 4.9c.7-.85 1.18-2.04 1.05-3.22-1.01.04-2.24.68-2.97 1.53-.65.75-1.23 1.96-1.07 3.12 1.13.09 2.29-.58 2.99-1.43Z" />
  </svg>
);
