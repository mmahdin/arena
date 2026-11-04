import React from 'react';
import {
  ArrowRight,
  Check,
  Flower2,
  Plus,
  Star,
} from 'lucide-react';

export function StatusBar() {
  // Keep the shared header spacing and safe area; device chrome belongs to the OS.
  return <div className="screen-top-inset" aria-hidden="true" />;
}

export function BackButton({ onClick, dark = false, floating = false, label = 'بازگشت' }) {
  return (
    <button
      type="button"
      className={`back-button ${dark ? 'back-button--dark' : ''} ${floating ? 'back-button--floating' : ''}`}
      onClick={onClick}
      aria-label={label}
    >
      <ArrowRight size={25} strokeWidth={2.2} />
    </button>
  );
}

export function PrimaryButton({ children, onClick, disabled = false, light = false, className = '', type = 'button' }) {
  return (
    <button
      type={type}
      className={`primary-button ${light ? 'primary-button--light' : ''} ${className}`}
      onClick={onClick}
      disabled={disabled}
    >
      {children}
    </button>
  );
}

export function PageHeader({ title, description, className = '' }) {
  return (
    <header className={`page-header ${className}`}>
      <h1>{title}</h1>
      {description && <p>{description}</p>}
    </header>
  );
}

export function PagerDots({ count, active, onChange, dark = false, label = 'انتخاب صفحه' }) {
  return (
    <div className={`pager-dots ${dark ? 'pager-dots--dark' : ''}`} role="group" aria-label={label}>
      {Array.from({ length: count }, (_, index) => (
        <button
          key={index}
          type="button"
          className={index === active ? 'is-active' : ''}
          onClick={() => onChange?.(index)}
          aria-label={`گزینه ${index + 1}`}
          aria-current={index === active ? 'true' : undefined}
        />
      ))}
    </div>
  );
}

export function RatingStars() {
  return (
    <span className="rating-stars" aria-label="۵ از ۵ ستاره">
      {Array.from({ length: 5 }, (_, index) => (
        <Star key={index} size={14} fill="currentColor" strokeWidth={1.5} />
      ))}
    </span>
  );
}

export function ChoiceButton({ children, selected, onClick, multi = false, disabled = false }) {
  return (
    <button
      type="button"
      className={`choice-button ${multi ? 'choice-button--multi' : ''} ${selected ? 'is-selected' : ''}`}
      onClick={onClick}
      aria-pressed={selected}
      disabled={disabled}
    >
      <span>{children}</span>
      {multi && (
        <span className="choice-icon" aria-hidden="true">
          {selected ? <Check size={21} strokeWidth={1.8} /> : <Plus size={22} strokeWidth={1.6} />}
        </span>
      )}
    </button>
  );
}

export function EnvelopeArt({ onClick }) {
  const content = (
    <>
      <svg className="envelope-lines" viewBox="0 0 340 160" aria-hidden="true">
        <path d="M7 12 L170 93 L333 12" />
        <path d="M8 148 L126 69" />
        <path d="M332 148 L214 69" />
      </svg>
      <span className="envelope-date">۳۱ اکتبر</span>
      <span className="wax-seal"><Flower2 size={34} strokeWidth={1.25} /></span>
    </>
  );

  if (onClick) {
    return <button type="button" className="envelope-art" onClick={onClick} aria-label="دیدن نامه">{content}</button>;
  }
  return <div className="envelope-art">{content}</div>;
}

export function ScribbleArt() {
  return (
    <svg className="scribble-art" viewBox="0 0 260 190" aria-hidden="true">
      <path
        fill="#fff"
        d="M43 22 C67 8 91 16 116 11 C143 5 174 4 195 17 C218 31 218 52 224 72 C230 95 223 119 206 132 C187 147 158 139 132 144 C112 148 94 143 79 146 L62 179 C55 188 48 181 49 169 L52 139 C30 130 21 109 24 85 C26 59 26 35 43 22 Z"
      />
      <path
        d="M75 103 C96 114 119 93 110 71 C103 52 87 67 91 82 C98 105 143 91 179 87"
        fill="none"
        stroke="#111"
        strokeWidth="9"
        strokeLinecap="round"
        strokeLinejoin="round"
      />
      <path d="M151 88 C164 76 174 76 182 88" fill="none" stroke="#111" strokeWidth="8" strokeLinecap="round" />
    </svg>
  );
}

export function ConstellationArt() {
  return (
    <svg className="constellation-art" viewBox="0 0 390 210" aria-hidden="true">
      <path d="M92 53 C157 63 231 79 289 139" fill="none" stroke="#fff" strokeWidth="5" strokeLinecap="round" />
      <g fill="#17231a">
        <ellipse cx="92" cy="53" rx="10" ry="15" transform="rotate(24 92 53)" />
        <ellipse cx="192" cy="54" rx="8" ry="12" transform="rotate(24 192 54)" />
        <ellipse cx="287" cy="52" rx="8" ry="12" transform="rotate(24 287 52)" />
        <ellipse cx="88" cy="139" rx="7" ry="12" transform="rotate(20 88 139)" />
        <ellipse cx="192" cy="139" rx="7" ry="12" transform="rotate(20 192 139)" />
        <ellipse cx="290" cy="139" rx="9" ry="13" transform="rotate(20 290 139)" />
      </g>
      <g fill="none" stroke="#fff" strokeWidth="5">
        <ellipse cx="92" cy="53" rx="21" ry="20" transform="rotate(24 92 53)" />
        <ellipse cx="290" cy="139" rx="23" ry="20" transform="rotate(-20 290 139)" />
      </g>
    </svg>
  );
}

export function InsightSpiral() {
  return (
    <svg className="insight-spiral" viewBox="0 0 160 120" aria-hidden="true">
      <g fill="none" stroke="#9299b3" strokeWidth="5" strokeLinecap="round" opacity=".75">
        <path d="M155 12 C116 -4 54 5 30 38 C3 74 37 111 84 115 C126 119 157 94 151 61 C146 30 110 18 79 28 C48 38 39 67 55 86 C70 104 105 103 121 84 C137 65 124 42 102 41 C79 39 64 56 70 72 C75 86 96 90 105 78 C113 68 104 57 94 59" />
        <path d="M146 4 C105 -8 43 8 20 46" opacity=".45" />
      </g>
    </svg>
  );
}

export function MiniEnvelope() {
  return (
    <svg viewBox="0 0 100 72" className="mini-envelope" aria-hidden="true">
      <path d="M8 14 C27 8 63 13 91 18 L86 61 C59 65 30 57 9 53 Z" fill="#fff" />
      <g fill="none" stroke="#24231f" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round">
        <path d="M10 14 L51 42 L90 18" />
        <path d="M9 53 L39 31" />
        <path d="M86 60 L62 35" />
      </g>
    </svg>
  );
}

export function VoiceWave({ path }) {
  return (
    <div className="voice-wave" aria-hidden="true">
      <svg viewBox="0 0 280 140" preserveAspectRatio="none">
        <path d={path} fill="none" stroke="#171713" strokeWidth="3" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  );
}
