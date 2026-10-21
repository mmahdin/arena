import { cn } from '@/lib/cn';

export interface SegmentItem<T extends string> {
  value: T;
  label: string;
  count?: number;
}

export interface SegmentedControlProps<T extends string> {
  items: SegmentItem<T>[];
  value: T;
  onChange: (value: T) => void;
  size?: 'sm' | 'md';
  className?: string;
  ariaLabel?: string;
}

/**
 * Segmented control — for switching between 2–5 mutually exclusive views
 * inside one screen. It never changes the URL of record, so back behaviour
 * stays predictable.
 */
export function SegmentedControl<T extends string>({
  items,
  value,
  onChange,
  size = 'md',
  className,
  ariaLabel,
}: SegmentedControlProps<T>) {
  return (
    <div
      role="tablist"
      aria-label={ariaLabel}
      className={cn(
        'flex gap-1 rounded-md border border-line bg-sunken p-1',
        size === 'sm' ? 'h-9' : 'h-11',
        className,
      )}
    >
      {items.map((item) => {
        const active = item.value === value;
        return (
          <button
            key={item.value}
            role="tab"
            type="button"
            aria-selected={active}
            onClick={() => onChange(item.value)}
            className={cn(
              'press relative flex flex-1 items-center justify-center gap-1.5 rounded-sm text-sm font-semibold transition-colors',
              active ? 'bg-surface text-ink shadow-xs' : 'text-ink-3 hover:text-ink-2',
            )}
          >
            <span className="truncate">{item.label}</span>
            {typeof item.count === 'number' && (
              <span
                className={cn(
                  'rounded-pill px-1.5 text-2xs font-bold tabular-nums',
                  active ? 'bg-brand-soft text-brand-ink' : 'bg-line text-ink-3',
                )}
              >
                {item.count.toLocaleString('fa-IR')}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
