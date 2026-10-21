import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';

/**
 * Meta line — the secondary text under every row title.
 * Items are separated by a dot rather than pipes or slashes, which reads
 * calmer in Persian and keeps the line scannable at 13.5px.
 */
export function Meta({ children, className }: { children: ReactNode; className?: string }) {
  const items = Array.isArray(children) ? children : [children];
  return (
    <span className={cn('flex flex-wrap items-center gap-x-1.5 gap-y-0.5 text-sm text-ink-3', className)}>
      {items
        .filter((x) => x !== null && x !== undefined && x !== false && x !== '')
        .map((item, i, arr) => (
          <span key={i} className="flex items-center gap-x-1.5">
            {item}
            {i < arr.length - 1 && <span aria-hidden className="size-1 shrink-0 rounded-full bg-line-strong" />}
          </span>
        ))}
    </span>
  );
}

/** A single emphasised meta item (deadline, priority) with its own tone. */
export function MetaTone({ tone = 'neutral', children }: { tone?: 'neutral' | 'danger' | 'warning' | 'success' | 'brand'; children: ReactNode }) {
  const map = {
    neutral: 'text-ink-3',
    danger: 'text-danger font-semibold',
    warning: 'text-warning font-semibold',
    success: 'text-success font-semibold',
    brand: 'text-brand-ink font-semibold',
  } as const;
  return <span className={map[tone]}>{children}</span>;
}

/** Vertical priority rail used at the start of high-priority rows. */
export function PriorityRail({ tone }: { tone: 'danger' | 'warning' | 'brand' }) {
  const map = { danger: 'bg-danger', warning: 'bg-warning', brand: 'bg-brand' } as const;
  return <span aria-hidden className={cn('-my-1 w-[3px] shrink-0 self-stretch rounded-pill', map[tone])} />;
}
