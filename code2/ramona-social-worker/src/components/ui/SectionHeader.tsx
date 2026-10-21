import type { ReactNode } from 'react';
import { Link } from 'react-router-dom';
import { cn } from '@/lib/cn';

export interface SectionHeaderProps {
  title: string;
  /** Short helper under the title — use sparingly. */
  hint?: string;
  action?: ReactNode;
  className?: string;
}

/**
 * Section header: the typographic anchor of a screen block.
 * Title is the only heavy element in a section, which is what gives the page
 * its hierarchy without extra decoration.
 */
export function SectionHeader({ title, hint, action, className }: SectionHeaderProps) {
  return (
    <div className={cn('mb-2.5 flex items-end justify-between gap-3', className)}>
      <div className="min-w-0">
        <h2 className="text-md font-bold text-ink leading-tight">{title}</h2>
        {hint && <p className="mt-0.5 text-xs text-ink-3">{hint}</p>}
      </div>
      {action && <div className="shrink-0 pb-0.5">{action}</div>}
    </div>
  );
}

/** Text-only link action used as a section header trailing element. */
export function SectionAction({ onClick, children, to }: { onClick?: () => void; children: ReactNode; to?: string }) {
  const cls = 'press text-sm font-semibold text-brand-ink hover:underline underline-offset-4';
  if (to) {
    return (
      <Link to={to} className={cls}>
        {children}
      </Link>
    );
  }
  return (
    <button type="button" onClick={onClick} className={cls}>
      {children}
    </button>
  );
}
