import type { ReactNode } from 'react';
import { cn } from '@/lib/cn';

export interface EmptyStateProps {
  icon?: ReactNode;
  title: string;
  description?: string;
  action?: ReactNode;
  className?: string;
}

/**
 * Empty state. Every list in the product has one, because "nothing here" is
 * meaningful information for a work queue — it must never be a blank screen.
 */
export function EmptyState({ icon, title, description, action, className }: EmptyStateProps) {
  return (
    <div className={cn('flex flex-col items-center justify-center px-8 py-14 text-center', className)}>
      {icon && (
        <div className="mb-4 grid size-14 place-items-center rounded-lg bg-sunken text-ink-3">
          {icon}
        </div>
      )}
      <p className="text-md font-bold text-ink">{title}</p>
      {description && <p className="mt-1.5 max-w-[30ch] text-sm leading-relaxed text-ink-3">{description}</p>}
      {action && <div className="mt-5">{action}</div>}
    </div>
  );
}
