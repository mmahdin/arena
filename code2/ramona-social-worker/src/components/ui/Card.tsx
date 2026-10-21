import type { HTMLAttributes, ReactNode } from 'react';
import { cn } from '@/lib/cn';

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  /** Adds row padding — most cards are content containers. */
  padded?: boolean;
  /** Emphasised container for the single most important block on a screen. */
  emphasis?: boolean;
  children: ReactNode;
}

/**
 * Surface container. The card is deliberately quiet: a hairline border plus a
 * whisper of shadow. Elevation is reserved for overlays (sheets, toasts), so
 * the reading surface stays flat and calm.
 */
export function Card({ padded = true, emphasis, className, children, ...rest }: CardProps) {
  return (
    <div
      className={cn(
        'rounded-lg border border-line bg-surface shadow-xs',
        emphasis && 'shadow-sm',
        padded && 'p-4',
        className,
      )}
      {...rest}
    >
      {children}
    </div>
  );
}

/** Tinted card used for blocks that need an emotional register (attention, guidance). */
export function ToneCard({
  tone = 'brand',
  className,
  children,
  ...rest
}: HTMLAttributes<HTMLDivElement> & { tone?: 'brand' | 'accent' | 'warning' | 'danger' | 'info' | 'success' }) {
  const map = {
    brand: 'bg-brand-soft border-brand-muted text-brand-ink',
    accent: 'bg-accent-soft border-accent/25 text-accent-ink',
    warning: 'bg-warning-soft border-warning/25 text-warning',
    danger: 'bg-danger-soft border-danger/20 text-danger',
    info: 'bg-info-soft border-info/20 text-info',
    success: 'bg-success-soft border-success/25 text-success',
  } as const;
  return (
    <div className={cn('rounded-lg border p-4', map[tone], className)} {...rest}>
      {children}
    </div>
  );
}
