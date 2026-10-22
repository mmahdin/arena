import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';

export function ProgressBar({
  value,
  tone = 'brand',
  size = 'md',
  className,
}: {
  value: number;
  tone?: 'brand' | 'success' | 'warning' | 'danger' | 'info';
  size?: 'sm' | 'md';
  className?: string;
}) {
  const pct = Math.max(0, Math.min(100, Math.round(value)));
  const fill = {
    brand: 'bg-brand',
    success: 'bg-success',
    warning: 'bg-warning',
    danger: 'bg-danger',
    info: 'bg-info',
  }[tone];
  return (
    <div
      role="progressbar"
      aria-valuenow={pct}
      aria-valuemin={0}
      aria-valuemax={100}
      className={cn('w-full overflow-hidden rounded-pill bg-line', size === 'sm' ? 'h-1' : 'h-1.5', className)}
    >
      <div
        className={cn('h-full rounded-pill transition-[width] duration-slow ease-out', fill)}
        style={{ width: `${pct}%` }}
      />
    </div>
  );
}

/** Numeric progress with a label — «۳ از ۴». */
export function ProgressLine({
  done,
  total,
  unit,
  tone = 'brand',
  className,
}: {
  done: number;
  total: number;
  unit: string;
  tone?: 'brand' | 'success' | 'warning' | 'danger' | 'info';
  className?: string;
}) {
  const pct = total === 0 ? 0 : (done / total) * 100;
  return (
    <div className={cn('space-y-1.5', className)}>
      <div className="flex items-baseline justify-between">
        <span className="text-xs font-medium text-ink-3">{unit}</span>
        <span className="text-xs font-bold text-ink tabular-nums">
          {fa(done)} از {fa(total)}
        </span>
      </div>
      <ProgressBar value={pct} tone={done === total ? 'success' : tone} size="sm" />
    </div>
  );
}

/**
 * Circular completion indicator (report drafts).
 * Pure SVG so it scales crisply and inherits the tone colour.
 */
export function ProgressRing({
  value,
  size = 36,
  children,
  className,
}: {
  value: number;
  size?: number;
  children?: ReactNode;
  className?: string;
}) {
  const r = (size - 4) / 2;
  const c = 2 * Math.PI * r;
  const pct = Math.max(0, Math.min(100, value));
  return (
    <div className={cn('relative grid place-items-center', className)} style={{ width: size, height: size }}>
      <svg width={size} height={size} className="-rotate-90" aria-hidden>
        <circle cx={size / 2} cy={size / 2} r={r} fill="none" stroke="currentColor" strokeWidth="3" className="text-line" />
        <circle
          cx={size / 2}
          cy={size / 2}
          r={r}
          fill="none"
          stroke="currentColor"
          strokeWidth="3"
          strokeLinecap="round"
          strokeDasharray={c}
          strokeDashoffset={c - (c * pct) / 100}
          className="text-brand transition-[stroke-dashoffset] duration-slow ease-out"
        />
      </svg>
      <span className="absolute text-2xs font-bold text-ink tabular-nums">{children ?? `${fa(pct)}٪`}</span>
    </div>
  );
}
