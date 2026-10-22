import { forwardRef, type ButtonHTMLAttributes, type ReactNode } from 'react';
import { cn } from '@/lib/cn';
import { Loader } from './Loader';

export type ButtonVariant = 'primary' | 'secondary' | 'ghost' | 'quiet' | 'danger';
export type ButtonSize = 'sm' | 'md' | 'lg';

export interface ButtonProps extends ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: ButtonVariant;
  size?: ButtonSize;
  /** Full-width inside its container — the default for page-level actions. */
  block?: boolean;
  loading?: boolean;
  iconStart?: ReactNode;
  iconEnd?: ReactNode;
}

const VARIANTS: Record<ButtonVariant, string> = {
  primary:
    'bg-primary text-primary-fg shadow-sm hover:bg-primary-hover active:bg-primary-hover disabled:bg-line-strong disabled:text-ink-3',
  secondary:
    'bg-surface text-ink border border-line-strong hover:bg-sunken active:bg-sunken disabled:text-ink-3',
  ghost: 'text-brand-ink hover:bg-brand-soft active:bg-brand-muted disabled:text-ink-3',
  quiet: 'bg-sunken text-ink hover:bg-line active:bg-line disabled:text-ink-3',
  danger: 'bg-danger-soft text-danger border border-danger/25 hover:bg-danger/15 active:bg-danger/20',
};

const SIZES: Record<ButtonSize, string> = {
  sm: 'h-9 px-3 text-sm rounded-sm gap-1.5',
  md: 'h-11 px-4 text-base rounded-md gap-2',
  lg: 'h-12 px-5 text-base rounded-md gap-2',
};

/**
 * The single button of the product.
 *
 * Height is never below 44px for `md`/`lg` so it stays a comfortable touch
 * target in the field — including with gloves or in a moving vehicle.
 */
export const Button = forwardRef<HTMLButtonElement, ButtonProps>(function Button(
  {
    variant = 'primary',
    size = 'md',
    block,
    loading,
    iconStart,
    iconEnd,
    className,
    children,
    disabled,
    type = 'button',
    ...rest
  },
  ref,
) {
  return (
    <button
      ref={ref}
      type={type}
      disabled={disabled || loading}
      aria-busy={loading || undefined}
      className={cn(
        'press inline-flex items-center justify-center font-semibold select-none',
        'disabled:cursor-not-allowed disabled:shadow-none',
        VARIANTS[variant],
        SIZES[size],
        block && 'w-full',
        className,
      )}
      {...rest}
    >
      {loading ? <Loader className="size-4" /> : iconStart}
      {children}
      {!loading && iconEnd}
    </button>
  );
});
