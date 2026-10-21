import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import { cn } from '@/lib/cn';

export interface IconButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  label: string;
  size?: 'sm' | 'md';
  variant?: 'plain' | 'soft' | 'outline';
  /** Unread/attention indicator. */
  dot?: boolean;
  badge?: number;
  children: ReactNode;
}

/**
 * Icon-only control. Always carries an accessible `label` because an icon
 * alone is never sufficient for screen readers — or for a tired user.
 */
export const IconButton = forwardRef<HTMLButtonElement, IconButtonProps>(function IconButton(
  { label, size = 'md', variant = 'plain', dot, badge, className, children, ...rest },
  ref,
) {
  return (
    <button
      ref={ref}
      type="button"
      aria-label={label}
      title={label}
      className={cn(
        'press relative grid place-items-center rounded-md text-ink-2',
        size === 'md' ? 'size-10' : 'size-8',
        variant === 'plain' && 'hover:bg-sunken active:bg-line',
        variant === 'soft' && 'bg-sunken text-ink hover:bg-line active:bg-line-strong',
        variant === 'outline' && 'border border-line bg-surface hover:bg-sunken',
        className,
      )}
      {...rest}
    >
      {children}
      {dot && (
        <span className="absolute end-1.5 top-1.5 size-2 rounded-full bg-danger ring-2 ring-surface" />
      )}
      {typeof badge === 'number' && badge > 0 && (
        <span className="absolute -end-1 -top-1 grid h-5 min-w-5 place-items-center rounded-pill bg-danger px-1 text-2xs font-bold text-white ring-2 ring-surface">
          {badge > 9 ? '۹+' : badge.toLocaleString('fa-IR')}
        </span>
      )}
    </button>
  );
});
