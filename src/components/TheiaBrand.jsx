import React from 'react';

/** The user-approved logo. Keep the supplied artwork and colors intact. */
export const THEIA_LOGO_SRC = '/brand/photo_2026-11-12_19-20-10.jpg';

export function TheiaMark({ size = 24, luminous = false, className = '' }) {
  return (
    <span className={`theia-mark ${luminous ? 'theia-mark--luminous' : ''} ${className}`.trim()} style={{ width: size, height: size }} aria-hidden="true">
      <img src={THEIA_LOGO_SRC} width="1254" height="1254" alt="" draggable="false" />
    </span>
  );
}

export function TheiaSealIcon({ size = 11, className = '' }) { return <TheiaMark size={size} className={className} />; }

/** A small light held within a quiet horizon; the onboarding's visual story. */
export function TheiaStory({ chapter = 'threshold', className = '' }) {
  const progress = { threshold: 0, listening: 1, reflection: 2, privacy: 3, checkin: 4, arrival: 5 }[chapter] ?? 0;
  return (
    <div className={`theia-story theia-story--${chapter} ${className}`} role="img" aria-label="نور آرامی که از دل تاریکی به افق می‌رسد">
      <svg viewBox="0 0 360 300" fill="none" aria-hidden="true">
        <defs>
          <radialGradient id={`glow-${chapter}`}><stop stopColor="#e9c79d" stopOpacity=".25"/><stop offset="1" stopColor="#e9c79d" stopOpacity="0"/></radialGradient>
          <linearGradient id={`light-${chapter}`} x1="180" y1="80" x2="180" y2="230"><stop stopColor="#f4dfc2" stopOpacity=".85"/><stop offset="1" stopColor="#c8b394" stopOpacity=".08"/></linearGradient>
        </defs>
        <circle cx="180" cy="150" r="122" fill={`url(#glow-${chapter})`} />
        <ellipse cx="180" cy="225" rx="117" ry="29" stroke="#c9b69c" strokeOpacity=".16" />
        <ellipse cx="180" cy="225" rx="83" ry="20" stroke="#c9b69c" strokeOpacity=".24" />
        <path d="M180 82v143" stroke={`url(#light-${chapter})`} strokeWidth="1" />
        <circle cx="180" cy="143" r="38" stroke="#e6d5bd" strokeOpacity=".48" />
        <circle cx="180" cy="143" r="27" stroke="#e6d5bd" strokeOpacity=".28" />
        <circle cx="180" cy="143" r="7" fill="#f2d7b3" fillOpacity={progress > 1 ? '.9' : '.65'} />
        <path d="M74 225h212" stroke="#e6d5bd" strokeOpacity=".22" />
        {progress > 0 && <path d="M115 225a65 65 0 0 1 130 0" stroke="#e9c79d" strokeOpacity=".42" />}
        {progress > 2 && <circle cx="180" cy="143" r="52" stroke="#e9c79d" strokeOpacity=".22" strokeDasharray="2 7" />}
        {progress > 4 && <path d="M180 76v-10m0 165v-10" stroke="#f2d7b3" strokeOpacity=".55" />}
      </svg>
      <span className="theia-story-caption" aria-hidden="true">{['آغازِ روشن‌شدن','جایی برای شنیدن','دیدنِ آنچه درون توست','فضایی امن برای مکث','بازگشت به خود','هر گفت‌وگو، یک سپیده'][progress]}</span>
    </div>
  );
}

export function TheiaHomeArt() {
  return <span className="home-conversation-art" aria-hidden="true"><TheiaMark size="100%" luminous className="home-conversation-mark theia-home-emblem" /></span>;
}

export function TheiaLetterEmpty({ size = 44 }) {
  return <svg width={size} height={size} viewBox="0 0 48 48" fill="none" className="theia-letter-empty" aria-hidden="true" focusable="false">
    <path d="M9 17.5C9 15.57 10.57 14 12.5 14H35.5C37.43 14 39 15.57 39 17.5V33.5C39 35.43 37.43 37 35.5 37H12.5C10.57 37 9 35.43 9 33.5V17.5Z" stroke="currentColor" strokeWidth="1.35" />
    <path d="M10.5 16L24 26.2L37.5 16" stroke="currentColor" strokeWidth="1.35" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M28.5 5.2A5.2 5.2 0 1 0 32.8 11.4A4.1 4.1 0 0 1 28.5 5.2Z" stroke="#d5d9f7" strokeOpacity=".65" strokeWidth="1.25" strokeLinecap="round" strokeLinejoin="round" />
    <path d="M21 6.5L21.65 8.35L23.5 9L21.65 9.65L21 11.5L20.35 9.65L18.5 9L20.35 8.35L21 6.5Z" fill="#f6cf98" fillOpacity=".78" />
  </svg>;
}
