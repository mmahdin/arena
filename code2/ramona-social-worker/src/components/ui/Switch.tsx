import { cn } from '@/lib/cn';

export interface SwitchProps {
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
  disabled?: boolean;
  className?: string;
}

/**
 * Accessible switch. The label is always rendered as real text next to the
 * control (never only as `aria-label`) because settings rows are read more
 * often than they are tapped.
 */
export function Switch({ checked, onChange, label, description, disabled, className }: SwitchProps) {
  return (
    <div className={cn('flex items-center justify-between gap-4 px-4 py-3', className)}>
      <div className="min-w-0">
        <p className="text-base font-medium text-ink">{label}</p>
        {description && <p className="mt-0.5 text-xs leading-relaxed text-ink-3">{description}</p>}
      </div>
      <button
        type="button"
        role="switch"
        aria-checked={checked}
        aria-label={label}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className={cn(
          'relative h-7 w-12 shrink-0 rounded-pill transition-colors duration-base ease-out',
          checked ? 'bg-brand' : 'bg-line-strong',
          disabled && 'opacity-50',
        )}
      >
        <span
          className={cn(
            'absolute top-1 size-5 rounded-full bg-white shadow-sm transition-all duration-base ease-out',
            checked ? 'start-1' : 'start-6',
          )}
        />
      </button>
    </div>
  );
}
