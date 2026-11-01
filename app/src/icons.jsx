import React from 'react';

const S = ({ size = 24, children, ...p }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor"
    strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" {...p}>{children}</svg>
);

export const ArrowBack = (p) => (
  <S {...p}><path d="M19 12H5" /><path d="M11 18l-6-6 6-6" /></S>
);
export const ArrowNext = (p) => (
  <S {...p}><path d="M5 12h14" /><path d="M13 6l6 6-6 6" /></S>
);
export const Close = (p) => (
  <S {...p}><path d="M18 6L6 18" /><path d="M6 6l12 12" /></S>
);
export const Check = (p) => (
  <S {...p} strokeWidth="2.4"><path d="M20 6L9 17l-5-5" /></S>
);
export const Mic = (p) => (
  <S {...p}><rect x="9" y="3" width="6" height="11" rx="3" /><path d="M5 11a7 7 0 0 0 14 0" /><path d="M12 18v3" /></S>
);
export const Lock = (p) => (
  <S size={16} {...p}><rect x="5" y="11" width="14" height="9" rx="2.5" /><path d="M8 11V8a4 4 0 0 1 8 0v3" /></S>
);
export const Info = (p) => (
  <S size={18} {...p}><circle cx="12" cy="12" r="9" /><path d="M12 11v5" /><circle cx="12" cy="8" r=".6" fill="currentColor" /></S>
);
export const Search = (p) => (
  <S size={17} {...p}><circle cx="11" cy="11" r="7" /><path d="M20 20l-3.5-3.5" /></S>
);
export const Wave = ({ size = 18, ...p }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="currentColor" {...p}>
    <rect x="3" y="10" width="2.4" height="4" rx="1.2" />
    <rect x="7.2" y="7" width="2.4" height="10" rx="1.2" />
    <rect x="11.4" y="4" width="2.4" height="16" rx="1.2" />
    <rect x="15.6" y="8" width="2.4" height="8" rx="1.2" />
    <rect x="19.8" y="10.5" width="2.4" height="3" rx="1.2" />
  </svg>
);
export const Clock = (p) => (
  <S size={13} {...p}><circle cx="12" cy="12" r="9" /><path d="M12 7v5l3 2" /></S>
);
export const Calendar = (p) => (
  <S size={13} {...p}><rect x="3" y="5" width="18" height="16" rx="3" /><path d="M3 10h18" /><path d="M8 3v4M16 3v4" /></S>
);
export const Dots = (p) => (
  <S size={20} {...p} fill="currentColor" stroke="none"><circle cx="5" cy="12" r="1.7" /><circle cx="12" cy="12" r="1.7" /><circle cx="19" cy="12" r="1.7" /></S>
);
export const Eye = (p) => (
  <S {...p}><path d="M2 12s3.5-6.5 10-6.5S22 12 22 12s-3.5 6.5-10 6.5S2 12 2 12z" /><circle cx="12" cy="12" r="2.6" /></S>
);
export const Camera = (p) => (
  <S size={20} {...p}><path d="M4 8h3l2-2.5h6L17 8h3a1 1 0 0 1 1 1v10a1 1 0 0 1-1 1H4a1 1 0 0 1-1-1V9a1 1 0 0 1 1-1z" /><circle cx="12" cy="13.5" r="3.2" /></S>
);
export const Video = (p) => (
  <S size={20} {...p}><rect x="3" y="7" width="12" height="10" rx="2.5" /><path d="M15 10.5l5.5-3v9l-5.5-3" /></S>
);
export const Sprout = (p) => (
  <S size={22} {...p}><path d="M12 21v-8" /><path d="M12 13c0-4 3-6 7-6 0 4-3 6-7 6z" /><path d="M12 13c0-4-3-6-7-6 0 4 3 6 7 6z" /></S>
);
export const Spark = (p) => (
  <S size={16} {...p}><path d="M12 3l1.8 5.2L19 10l-5.2 1.8L12 17l-1.8-5.2L5 10l5.2-1.8z" /></S>
);
export const Compass = (p) => (
  <S size={18} {...p}><circle cx="12" cy="12" r="9" /><path d="M15.5 8.5l-2 5-5 2 2-5z" /></S>
);
export const History = (p) => (
  <S size={18} {...p}><path d="M3 12a9 9 0 1 0 3-6.7" /><path d="M3 4v5h5" /><path d="M12 8v4l2.5 2" /></S>
);
export const Gear = (p) => (
  <S size={18} {...p}><circle cx="12" cy="12" r="3.2" /><path d="M12 2.8v2.4M12 18.8v2.4M4.9 4.9l1.7 1.7M17.4 17.4l1.7 1.7M2.8 12h2.4M18.8 12h2.4M4.9 19.1l1.7-1.7M17.4 6.6l1.7-1.7" /></S>
);
export const Google = (p) => (
  <svg width="18" height="18" viewBox="0 0 24 24" {...p}>
    <path fill="#4285F4" d="M23.5 12.27c0-.85-.08-1.66-.22-2.45H12v4.64h6.45a5.52 5.52 0 0 1-2.39 3.62v3h3.87c2.26-2.09 3.57-5.17 3.57-8.81z"/>
    <path fill="#34A853" d="M12 24c3.24 0 5.96-1.07 7.94-2.91l-3.87-3c-1.07.72-2.45 1.15-4.07 1.15-3.13 0-5.78-2.11-6.73-4.96H1.29v3.1A12 12 0 0 0 12 24z"/>
    <path fill="#FBBC05" d="M5.27 14.28A7.2 7.2 0 0 1 4.89 12c0-.79.14-1.56.38-2.28v-3.1H1.29a12 12 0 0 0 0 10.76l3.98-3.1z"/>
    <path fill="#EA4335" d="M12 4.77c1.76 0 3.34.6 4.58 1.8l3.44-3.44A11.98 11.98 0 0 0 12 0 12 12 0 0 0 1.29 6.62l3.98 3.1C6.22 6.88 8.87 4.77 12 4.77z"/>
  </svg>
);
export const Apple = (p) => (
  <svg width="18" height="18" viewBox="0 0 24 24" fill="currentColor" {...p}>
    <path d="M16.36 12.94c.03 3.05 2.68 4.06 2.71 4.08-.02.07-.42 1.45-1.4 2.87-.84 1.23-1.72 2.45-3.1 2.48-1.36.03-1.79-.8-3.34-.8-1.55 0-2.03.77-3.31.83-1.33.05-2.34-1.33-3.19-2.55C2.9 17.16 1.55 12.6 3.32 9.8a4.78 4.78 0 0 1 4-2.45c1.25-.02 2.42.84 3.18.84.76 0 2.19-1.03 3.7-.88.63.03 2.4.26 3.54 1.93-.09.06-2.11 1.23-2.09 3.68M13.9 4.38c.68-.82 1.14-1.96 1.01-3.1-.98.04-2.16.65-2.86 1.47-.63.73-1.18 1.89-1.03 3 .1.08 1 .14 2.88-1.37"/>
  </svg>
);

/* Hourglass logo mark */
export const Hourglass = ({ size = 30, ...p }) => (
  <svg width={size} height={size} viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.9" strokeLinecap="round" strokeLinejoin="round" {...p}>
    <path d="M6 3h12" /><path d="M6 21h12" />
    <path d="M7 3c0 4.5 3.2 5.6 5 9-1.8 3.4-5 4.5-5 9" />
    <path d="M17 3c0 4.5-3.2 5.6-5 9 1.8 3.4 5 4.5 5 9" />
  </svg>
);

export const Logo = ({ dark = false, big = false }) => (
  <span className="brand" style={{ color: dark ? '#fff' : 'var(--ink)' }}>
    <Hourglass size={big ? 34 : 22} />
    <span style={big ? { fontSize: 34 } : undefined}>مسیر</span>
  </span>
);

/* Olive path ribbon for dark onboarding cards */
export const PathArt = ({ node = 0 }) => (
  <svg className="path-art" width="100%" height="190" viewBox="0 0 390 190" fill="none" preserveAspectRatio="xMidYMid meet">
    <defs>
      <linearGradient id="ribbon" x1="0" y1="0" x2="390" y2="0" gradientUnits="userSpaceOnUse">
        <stop stopColor="#909a5b" /><stop offset="1" stopColor="#597137" />
      </linearGradient>
    </defs>
    <path d="M-20 150 C 90 40, 190 230, 300 120 S 420 60, 430 90"
      stroke="url(#ribbon)" strokeWidth="34" strokeLinecap="round" opacity=".9" />
    <path d="M-20 150 C 90 40, 190 230, 300 120 S 420 60, 430 90"
      stroke="rgba(255,255,255,.18)" strokeWidth="34" strokeLinecap="round" strokeDasharray="1 14" />
    <circle cx="150" cy="128" r="26" fill="#e5e7e6" />
    <circle cx="150" cy="128" r="15" fill="#c9cdca" />
    <circle cx="300" cy="120" r="17" fill="#1b1b1b" stroke="#e5e7e6" strokeWidth="5" />
  </svg>
);
