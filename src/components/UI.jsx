import React, { useEffect, useRef } from 'react';
import { ArrowRight, Check, Plus, X } from 'lucide-react';
import { LetterSymbol, TheiaMark } from './Brand';

/** Keeps the device's safe area; device chrome belongs to the OS. */
export function StatusBar() {
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
      <ArrowRight size={22} strokeWidth={1.7} aria-hidden="true" />
    </button>
  );
}

export function PrimaryButton({ children, onClick, disabled = false, light = false, className = '', type = 'button', ariaLabel }) {
  return (
    <button
      type={type}
      className={`primary-button ${light ? 'primary-button--light' : ''} ${className}`.trim()}
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

/** Brand lockup: mark + wordmark, with an optional hairline rule. */
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

/** Accessible native-dialog bottom sheet used for conversation actions. */
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
            <X size={18} aria-hidden="true" />
          </button>
        </header>
        {children}
      </div>
    </dialog>
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
          {selected ? <Check size={14} strokeWidth={2.4} /> : <Plus size={14} strokeWidth={1.8} />}
        </span>
      )}
    </button>
  );
}

/** A letter sealed with light — the brand's most tactile small moment. */
export function MiniEnvelope() {
  return <LetterSymbol size={26} />;
}

export function VoiceWave({ path }) {
  return (
    <div className="voice-wave" aria-hidden="true">
      <svg viewBox="0 0 280 140" preserveAspectRatio="none">
        <path d={path} fill="none" strokeLinecap="round" strokeLinejoin="round" />
      </svg>
    </div>
  );
}
