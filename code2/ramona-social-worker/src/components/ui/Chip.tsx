import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';

export interface ChipProps {
  selected?: boolean;
  count?: number;
  icon?: ReactNode;
  onClick?: () => void;
  children: ReactNode;
  className?: string;
}

/**
 * Filter chip. Selection is signalled by fill *and* weight *and* colour —
 * never by colour alone, so it survives colour-blindness and bright sunlight.
 */
export function Chip({ selected, count, icon, onClick, children, className }: ChipProps) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={cn(
        'press inline-flex h-9 shrink-0 items-center gap-1.5 rounded-pill border px-3.5 text-sm font-medium whitespace-nowrap',
        selected
          ? 'border-transparent bg-ink text-bg shadow-xs'
          : 'border-line bg-surface text-ink-2 hover:border-line-strong hover:text-ink',
        className,
      )}
    >
      {icon}
      {children}
      {typeof count === 'number' && (
        <span
          className={cn(
            'grid h-5 min-w-5 place-items-center rounded-pill px-1 text-2xs font-bold tabular-nums',
            selected ? 'bg-white/20 text-white' : 'bg-sunken text-ink-3',
          )}
        >
          {count.toLocaleString('fa-IR')}
        </span>
      )}
    </button>
  );
}

/** Horizontal scroll rail for chip groups — no visible scrollbar, edge padding. */
export function ChipRail({
  children,
  className,
  /** Set inside containers that already provide the page gutter. */
  flush,
}: {
  children: ReactNode;
  className?: string;
  flush?: boolean;
}) {
  return (
    <div
      className={cn(
        'no-scrollbar rail-fade flex gap-2 overflow-x-auto pb-0.5',
        flush ? '' : '-mx-gutter px-gutter',
        className,
      )}
    >
      {children}
    </div>
  );
}
