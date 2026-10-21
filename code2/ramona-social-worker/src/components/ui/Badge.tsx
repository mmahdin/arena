import type { ReactNode } from 'react';
import type { Tone } from '@/types';
import { cn } from '@/lib/cn';
import { TONE } from './tone';

export interface BadgeProps {
  tone?: Tone;
  /** Solid badges are for the single most important status on a card. */
  variant?: 'soft' | 'outline' | 'solid';
  size?: 'sm' | 'md';
  icon?: ReactNode;
  children: ReactNode;
  className?: string;
}

/**
 * Status vocabulary. One shape, one tone system — a user learns the colour
 * language once and then reads any screen instantly.
 */
export function Badge({ tone = 'neutral', variant = 'soft', size = 'sm', icon, children, className }: BadgeProps) {
  const t = TONE[tone];
  return (
    <span
      className={cn(
        'inline-flex items-center gap-1 rounded-pill font-semibold whitespace-nowrap',
        size === 'sm' ? 'h-6 px-2 text-2xs' : 'h-7 px-3 text-xs',
        variant === 'soft' && cn(t.soft, t.text),
        variant === 'outline' && cn('border bg-transparent', t.border, t.text),
        variant === 'solid' && cn(t.solid, 'text-white'),
        className,
      )}
    >
      {icon}
      {children}
    </span>
  );
}

/** A bare colour dot — the most compact status signal in the system. */
export function Dot({ tone = 'neutral', className }: { tone?: Tone; className?: string }) {
  return <span className={cn('size-2 shrink-0 rounded-full', TONE[tone].dot, className)} />;
}
