import { useId, type InputHTMLAttributes, type ReactNode, type TextareaHTMLAttributes } from 'react';
import { ChevronDown } from 'lucide-react';
import { cn } from '@/lib/cn';

/**
 * Form primitives.
 *
 * One shared label/hint/error structure across inputs, selects and textareas,
 * so a report form and a filter field feel like the same product. Labels are
 * real `<label>` elements bound by id — never placeholders pretending to be
 * labels.
 */

export interface FieldShellProps {
  label: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: ReactNode;
  className?: string;
  counter?: string;
}

export function FieldShell({ label, hint, error, required, children, className, counter }: FieldShellProps) {
  return (
    <div className={cn('space-y-1.5', className)}>
      <div className="flex items-baseline justify-between gap-3">
        <label className="text-sm font-semibold text-ink">
          {label}
          {required && <span className="ms-1 text-danger">*</span>}
        </label>
        {counter && <span className="text-2xs text-ink-3 tabular-nums">{counter}</span>}
      </div>
      {children}
      {hint && !error && <p className="text-xs leading-relaxed text-ink-3">{hint}</p>}
      {error && (
        <p className="text-xs font-medium text-danger" role="alert">
          {error}
        </p>
      )}
    </div>
  );
}

const controlBase =
  'w-full rounded-md border bg-surface px-3.5 text-base text-ink placeholder:text-ink-3/70 transition-colors focus:border-brand focus:outline-none focus:ring-2 focus:ring-brand/25 disabled:bg-sunken disabled:text-ink-3';

export interface TextInputProps extends InputHTMLAttributes<HTMLInputElement> {
  label: string;
  hint?: string;
  error?: string;
  leadingIcon?: ReactNode;
}

export function TextInput({ label, hint, error, leadingIcon, className, id, ...rest }: TextInputProps) {
  const autoId = useId();
  const inputId = id ?? autoId;
  return (
    <FieldShell label={label} hint={hint} error={error}>
      <div className="relative">
        {leadingIcon && (
          <span className="pointer-events-none absolute inset-y-0 start-3 grid place-items-center text-ink-3">
            {leadingIcon}
          </span>
        )}
        <input
          id={inputId}
          aria-invalid={!!error}
          className={cn(controlBase, 'h-12', error ? 'border-danger' : 'border-line-strong', leadingIcon && 'ps-10', className)}
          {...rest}
        />
      </div>
    </FieldShell>
  );
}

export interface TextAreaProps extends TextareaHTMLAttributes<HTMLTextAreaElement> {
  label: string;
  hint?: string;
  error?: string;
  rows?: number;
  max?: number;
}

export function TextArea({ label, hint, error, rows = 4, max, className, value, ...rest }: TextAreaProps) {
  const length = typeof value === 'string' ? value.length : 0;
  return (
    <FieldShell label={label} hint={hint} error={error} counter={max ? `${length.toLocaleString('fa-IR')} / ${max.toLocaleString('fa-IR')}` : undefined}>
      <textarea
        rows={rows}
        value={value}
        aria-invalid={!!error}
        className={cn(
          controlBase,
          'resize-none py-3 leading-relaxed',
          error ? 'border-danger' : 'border-line-strong',
          className,
        )}
        {...rest}
      />
    </FieldShell>
  );
}

export interface SelectInputProps {
  label: string;
  value: string;
  options: string[];
  onChange: (value: string) => void;
  hint?: string;
  placeholder?: string;
}

/**
 * Option group rendered as chips rather than a native `<select>`: a native
 * picker hides the remaining choices, and in this product seeing all options
 * at once prevents mistakes.
 */
export function SelectInput({ label, value, options, onChange, hint, placeholder }: SelectInputProps) {
  return (
    <FieldShell label={label} hint={hint}>
      <div className="relative">
        <select
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className={cn(controlBase, 'h-12 appearance-none border-line-strong pe-10', !value && 'text-ink-3')}
        >
          {placeholder && <option value="">{placeholder}</option>}
          {options.map((o) => (
            <option key={o} value={o}>
              {o}
            </option>
          ))}
        </select>
        <ChevronDown className="pointer-events-none absolute inset-y-0 end-3 my-auto size-4 text-ink-3" aria-hidden />
      </div>
    </FieldShell>
  );
}

/** Read-only key/value line — the backbone of every detail screen. */
export function KeyValue({
  label,
  value,
  icon,
  masked,
  className,
}: {
  label: string;
  value: ReactNode;
  icon?: ReactNode;
  masked?: boolean;
  className?: string;
}) {
  return (
    <div className={cn('flex items-start justify-between gap-4 py-2.5', className)}>
      <span className="flex shrink-0 items-center gap-2 text-sm text-ink-3">
        {icon}
        {label}
      </span>
      <span className={cn('min-w-0 text-end text-sm font-medium text-ink', masked && 'masked')}>{value}</span>
    </div>
  );
}
