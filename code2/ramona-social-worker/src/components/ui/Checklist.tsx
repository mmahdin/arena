import { Check } from 'lucide-react';
import { cn } from '@/lib/cn';

export interface ChecklistRowProps {
  checked: boolean;
  onChange: () => void;
  label: string;
  hint?: string;
  required?: boolean;
  disabled?: boolean;
  className?: string;
}

/**
 * Checklist row — used for visit preparation steps and multi-select fields.
 * The whole row is the tap target, which matters when it is ticked off while
 * standing in someone's hallway.
 */
export function ChecklistRow({
  checked,
  onChange,
  label,
  hint,
  required,
  disabled,
  className,
}: ChecklistRowProps) {
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={checked}
      disabled={disabled}
      onClick={onChange}
      className={cn(
        'press flex w-full items-start gap-3 rounded-md px-1 py-2.5 text-start',
        disabled && 'opacity-50',
        className,
      )}
    >
      <span
        className={cn(
          'mt-0.5 grid size-6 shrink-0 place-items-center rounded-sm border-2 transition-all duration-fast',
          checked ? 'border-brand bg-brand text-white' : 'border-line-strong bg-surface',
        )}
      >
        {checked && <Check className="size-4" strokeWidth={3.25} aria-hidden />}
      </span>
      <span className="min-w-0 flex-1">
        <span className={cn('block text-base leading-snug', checked ? 'font-semibold text-ink' : 'text-ink')}>
          {label}
          {required && <span className="ms-1.5 text-2xs font-medium text-danger">الزامی</span>}
        </span>
        {hint && <span className="mt-0.5 block text-xs leading-relaxed text-ink-3">{hint}</span>}
      </span>
    </button>
  );
}
