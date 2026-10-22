import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { ChevronLeft } from 'lucide-react';
import { cn } from '@/lib/cn';

export interface ListRowProps {
  leading?: ReactNode;
  title: ReactNode;
  meta?: ReactNode;
  trailing?: ReactNode;
  /** Show the RTL "go deeper" affordance. */
  chevron?: boolean;
  to?: string;
  onClick?: () => void;
  /** Hairline separator that respects the leading element's width. */
  divider?: boolean;
  /** Dims the row — used for completed/archived items. */
  muted?: boolean;
  /** Emphasised leading edge (priority rail). */
  accent?: ReactNode;
  className?: string;
  titleClassName?: string;
  dense?: boolean;
  disabled?: boolean;
}

/**
 * The list row is the most repeated component in the app, so it owns the
 * rhythm: 44px+ tap height, 16px padding, consistent 12px gap, and a strict
 * leading / content / trailing structure. Anything that looks like a row
 * should be built from this.
 */
export function ListRow({
  leading,
  title,
  meta,
  trailing,
  chevron,
  to,
  onClick,
  divider,
  muted,
  accent,
  className,
  titleClassName,
  dense,
  disabled,
}: ListRowProps) {
  const content = (
    <>
      {accent}
      {leading && <div className="shrink-0 self-center">{leading}</div>}
      <div className="min-w-0 flex-1">
        <div className={cn('text-base font-semibold text-ink clamp-1', muted && 'text-ink-2', titleClassName)}>
          {title}
        </div>
        {meta && <div className="mt-0.5 flex flex-wrap items-center gap-x-1.5 text-sm text-ink-3">{meta}</div>}
      </div>
      {(trailing || chevron) && (
        <div className="flex shrink-0 items-center gap-1 self-center">
          {trailing}
          {chevron && <ChevronLeft className="size-[18px] text-ink-3" strokeWidth={2.25} aria-hidden />}
        </div>
      )}
    </>
  );

  const classes = cn(
    'relative flex w-full items-start gap-3 text-start transition-colors',
    dense ? 'px-4 py-2.5' : 'px-4 py-3.5',
    (to || onClick) && !disabled && 'press hover:bg-sunken active:bg-sunken',
    disabled && 'opacity-60',
    divider && 'after:absolute after:inset-x-4 after:bottom-0 after:h-px after:bg-line-soft',
    className,
  );

  if (disabled) return <div className={classes}>{content}</div>;
  if (to) {
    return (
      <Link to={to} className={classes} onClick={onClick}>
        {content}
      </Link>
    );
  }
  if (onClick) {
    return (
      <button type="button" className={classes} onClick={onClick}>
        {content}
      </button>
    );
  }
  return <div className={classes}>{content}</div>;
}

/** Container that groups rows into one bordered surface. */
export function ListGroup({ children, className }: { children: ReactNode; className?: string }) {
  return (
    <div className={cn('overflow-hidden rounded-lg border border-line bg-surface shadow-xs', className)}>
      {children}
    </div>
  );
}
