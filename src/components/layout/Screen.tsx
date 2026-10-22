import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';

/**
 * Screen scaffolding.
 *
 * `Screen` owns horizontal rhythm (the gutter) and bottom breathing room;
 * `Section` owns vertical rhythm. Using them keeps every screen spaced
 * identically without anyone thinking about margins.
 */
export function Screen({
  children,
  className,
  padded = true,
  bottomGap = true,
}: {
  children?: ReactNode;
  className?: string;
  padded?: boolean;
  bottomGap?: boolean;
}) {
  return (
    <div className={cn(padded && 'px-gutter', bottomGap && 'pb-8', className)}>{children}</div>
  );
}

export function Section({
  children,
  className,
  gap = 'md',
}: {
  children: ReactNode;
  className?: string;
  gap?: 'sm' | 'md' | 'lg';
}) {
  return (
    <section
      className={cn(
        gap === 'sm' && 'mt-4',
        gap === 'md' && 'mt-6',
        gap === 'lg' && 'mt-8',
        'first:mt-0',
        className,
      )}
    >
      {children}
    </section>
  );
}

/** Sticky action bar pinned to the bottom of the scroll area (detail screens). */
export function ActionFooter({
  children,
  className,
  hint,
}: {
  children: ReactNode;
  className?: string;
  hint?: ReactNode;
}) {
  return (
    <div
      className={cn(
        'sticky bottom-0 z-20 -mx-gutter mt-6 border-t border-line bg-surface/95 px-gutter pb-4 pt-3 backdrop-blur-md',
        className,
      )}
      style={{ paddingBottom: 'max(1rem, env(safe-area-inset-bottom, 0px))' }}
    >
      {hint && <div className="mb-2.5 text-center text-xs leading-relaxed text-ink-3">{hint}</div>}
      <div className="flex items-center gap-2.5">{children}</div>
    </div>
  );
}
