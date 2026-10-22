import type { ReactNode } from 'react';
import type { Tone } from '@/types';
import { cn } from '@/lib/cn';
import { initials } from '@/lib/digits';
import { TONE } from './tone';

export interface AvatarProps {
  name: string;
  size?: 'xs' | 'sm' | 'md' | 'lg';
  tone?: Tone;
  /** Render an icon instead of initials (used for children/centres). */
  icon?: ReactNode;
  className?: string;
  masked?: boolean;
}

const SIZES = {
  xs: 'size-7 text-2xs rounded-sm',
  sm: 'size-9 text-xs rounded-md',
  md: 'size-11 text-sm rounded-md',
  lg: 'size-16 text-lg rounded-lg',
} as const;

/**
 * Identity marker. People are represented by initials on a tinted tile rather
 * than photography: no image assets to load in the field, and no risk of
 * exposing a child's face on a screen used in public.
 */
export function Avatar({ name, size = 'md', tone = 'brand', icon, className, masked }: AvatarProps) {
  return (
    <span
      aria-hidden={masked}
      className={cn(
        'grid shrink-0 place-items-center font-bold select-none',
        SIZES[size],
        TONE[tone].soft,
        TONE[tone].text,
        masked && 'masked',
        className,
      )}
    >
      {icon ?? initials(name)}
    </span>
  );
}

/** Small stack of children in a case row. */
export function AvatarStack({ names, tone = 'brand' }: { names: string[]; tone?: Tone }) {
  if (names.length === 0) return null;
  return (
    <div className="flex items-center -space-x-2 space-x-reverse">
      {names.slice(0, 3).map((n) => (
        <span key={n} className="ring-2 ring-surface rounded-sm">
          <Avatar name={n} size="xs" tone={tone} />
        </span>
      ))}
      {names.length > 3 && (
        <span className="grid size-7 place-items-center rounded-sm bg-sunken text-2xs font-bold text-ink-3 ring-2 ring-surface">
          {(names.length - 3).toLocaleString('fa-IR')}+
        </span>
      )}
    </div>
  );
}
