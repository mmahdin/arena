import React, { useEffect, useRef } from 'react';
import {
  ArrowRight,
  Check,
  Plus,
  Star,
  X,
} from 'lucide-react';

/**
 * Kenar mark — two stones resting side by side on a shore line.
 * "کنار" means *beside*: the shoreline, and the one who stays at your side.
 * Drawn inline so it inherits currentColor and stays crisp at any size.
 */
export function KenarMark({ size = 24, className = '', title, ...rest }) {
  return (
    <svg
      className={`kenar-mark ${className}`}
      width={size}
      height={size * 0.75}
      viewBox="0 0 160 120"
      fill="none"
      role={title ? 'img' : undefined}
      aria-hidden={title ? undefined : 'true'}
      aria-label={title}
      focusable="false"
      {...rest}
    >
      <path
        d="M106.6 71.4C111.3 71.4 114.7 74.4 115.6 78.2C116.9 83.8 112.6 89.2 105.2 89.2C97.8 89.2 96.4 86 96.4 82.6C96.4 76.4 101.2 71.4 106.6 71.4Z"
        fill="currentColor"
        opacity=".55"
      />
      <path
        d="M52 34.5C68.5 34.5 83 46.5 83 62.2C83 71.8 78 78 68.5 82.6C61.8 86 49 90.5 37.6 90.5C28.6 90.5 21 86.4 21 76.2C21 53 32.5 34.5 52 34.5Z"
        fill="currentColor"
      />
      <path
        d="M20 97C46 105 96 105 132 94"
        stroke="currentColor"
        strokeWidth="5.4"
        strokeLinecap="round"
      />
    </svg>
  );
}

/** Wordmark: the product name set as its own typographic signature. */
export function KenarWordmark({ className = '' }) {
  return (
    <span className={`kenar-wordmark ${className}`} aria-label="کنار">
      <span aria-hidden="true">کنار</span>
    </span>
  );
}

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

export function PrimaryButton({ children, onClick, disabled = false, light = false, className = '', type = 'button', ariaLabel }) {
  return (
    <button
      type={type}
      className={`primary-button ${light ? 'primary-button--light' : ''} ${className}`}
      onClick={onClick}
      disabled={disabled}
      aria-label={ariaLabel}
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

export function PagerDots({ count, active, onChange, dark = false, label = 'انتخاب صفحه', labels = [] }) {
  return (
    <div className={`pager-dots ${dark ? 'pager-dots--dark' : ''}`} role="group" aria-label={label}>
      {Array.from({ length: count }, (_, index) => (
        <button
          key={index}
          type="button"
          className={index === active ? 'is-active' : ''}
          onClick={() => onChange?.(index)}
          aria-label={labels[index] || `گزینه ${index + 1}`}
          aria-current={index === active ? 'true' : undefined}
        />
      ))}
    </div>
  );
}

/** Accessible native-dialog bottom sheet used for conversation/message actions. */
export function BottomSheet({ open, onClose, title, children, className = '' }) {
  const ref = useRef(null);
  const returnFocus = useRef(null);
  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return undefined;
    if (open) {
      returnFocus.current = document.activeElement;
      if (!dialog.open) dialog.showModal();
      const first = dialog.querySelector('button, [href], input, textarea, select, [tabindex]:not([tabindex="-1"])');
      first?.focus();
    } else if (dialog.open) {
      dialog.close();
    }
    return undefined;
  }, [open]);
  useEffect(() => {
    const dialog = ref.current;
    if (!dialog) return undefined;
    const onCloseEvent = () => onClose?.();
    dialog.addEventListener('close', onCloseEvent);
    return () => dialog.removeEventListener('close', onCloseEvent);
  }, [onClose]);
  useEffect(() => () => {
    if (returnFocus.current instanceof HTMLElement) returnFocus.current.focus();
  }, []);
  return (
    <dialog
      ref={ref}
      className={`bottom-sheet ${className}`}
      aria-label={title}
      onCancel={(event) => { event.preventDefault(); onClose?.(); }}
      onClick={(event) => { if (event.target === event.currentTarget) onClose?.(); }}
    >
      <div className="bottom-sheet-panel">
        <header className="bottom-sheet-header">
          <h2>{title}</h2>
          <button type="button" className="bottom-sheet-close" onClick={onClose} aria-label="بستن">
            <X size={20} aria-hidden="true" />
          </button>
        </header>
        {children}
      </div>
    </dialog>
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

export function EnvelopeArt({ onClick, label = 'نامهٔ کنار', ariaLabel = 'باز کردن نامهٔ کنار' }) {
  const content = (
    <>
      <svg className="envelope-lines" viewBox="0 0 340 160" aria-hidden="true">
        <path d="M7 12 L170 93 L333 12" />
        <path d="M8 148 L126 69" />
        <path d="M332 148 L214 69" />
      </svg>
      <span className="envelope-date">{label}</span>
      <span className="wax-seal"><KenarMark size={34} /></span>
    </>
  );

  if (onClick) {
    return <button type="button" className="envelope-art" onClick={onClick} aria-label={ariaLabel}>{content}</button>;
  }
  return <div className="envelope-art">{content}</div>;
}

/**
 * A single companion stone resting on its shore, with two quiet ripples.
 * Shared decorative motif for panels that used to carry the floral artwork.
 */
export function ShoreStoneArt({ className = '' }) {
  return (
    <svg className={`shore-stone-art ${className}`} viewBox="0 0 260 190" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" opacity=".55">
        <path d="M18 132 C60 142 200 142 244 130" strokeWidth="4" />
        <path d="M56 150 C96 158 168 158 206 148" strokeWidth="2.6" opacity=".6" />
      </g>
      <path
        d="M130 132 C96 132 72 116 68 92 C63 62 88 40 118 40 C150 40 174 64 172 94 C171 112 158 132 130 132 Z"
        fill="currentColor"
        fillOpacity=".18"
        stroke="currentColor"
        strokeWidth="4.4"
        strokeLinejoin="round"
      />
      <g fill="none" stroke="currentColor" strokeWidth="3.4" strokeLinecap="round">
        <path d="M104 82 C109 89 116 89 121 82" />
        <path d="M136 82 C141 89 148 89 153 82" />
        <path d="M124 100 C128 104 133 104 137 100" />
      </g>
    </svg>
  );
}

/**
 * The "beside" constellation: two stones sharing one shore, drawn as points
 * of light. Replaces the previous astral-flower imagery on insight panels.
 */
export function ConstellationArt() {
  return (
    <svg className="constellation-art" viewBox="0 0 390 210" aria-hidden="true">
      <path d="M34 150 C120 168 268 168 356 146" fill="none" stroke="currentColor" strokeWidth="4" strokeLinecap="round" opacity=".5" />
      <g fill="currentColor" opacity=".9">
        <ellipse cx="152" cy="96" rx="24" ry="20" />
        <ellipse cx="204" cy="112" rx="13" ry="11" opacity=".6" />
      </g>
      <g fill="none" stroke="currentColor" strokeWidth="2.6" strokeLinecap="round" opacity=".55">
        <path d="M128 62 C136 50 152 46 164 52" />
        <path d="M118 116 C104 122 92 120 84 112" />
      </g>
      <g fill="currentColor" opacity=".45">
        <circle cx="94" cy="58" r="3" />
        <circle cx="286" cy="74" r="3.4" />
        <circle cx="312" cy="112" r="2.4" />
      </g>
    </svg>
  );
}

/** Concentric ripples — an idea settling, the way a stone settles the water. */
export function InsightSpiral() {
  return (
    <svg className="insight-spiral" viewBox="0 0 160 120" aria-hidden="true">
      <g fill="none" stroke="currentColor" strokeLinecap="round" opacity=".7">
        <ellipse cx="96" cy="62" rx="18" ry="13" strokeWidth="3.4" />
        <ellipse cx="96" cy="62" rx="36" ry="26" strokeWidth="2.6" opacity=".7" />
        <ellipse cx="96" cy="62" rx="54" ry="39" strokeWidth="2.2" opacity=".45" />
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
