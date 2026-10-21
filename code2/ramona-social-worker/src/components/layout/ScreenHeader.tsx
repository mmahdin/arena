import type { ReactNode } from 'react';
import { useNavigate } from 'react-router-dom';
import { ChevronRight } from 'lucide-react';
import { cn } from '@/lib/cn';

export interface ScreenHeaderProps {
  title: ReactNode;
  /** Small line under the title: case code, date, status. */
  subtitle?: ReactNode;
  /** Show a back control. In RTL "back" points right. */
  back?: boolean;
  onBack?: () => void;
  actions?: ReactNode;
  /** Renders a soft hairline + blur once the page scrolls. */
  className?: string;
  children?: ReactNode;
}

/**
 * Top app bar.
 *
 * Height is fixed at 56px, the title is the only bold element, and actions are
 * capped at two — a third action belongs in an overflow menu. The bar is
 * translucent so content sliding underneath reads as depth, not as clutter.
 */
export function ScreenHeader({ title, subtitle, back, onBack, actions, className, children }: ScreenHeaderProps) {
  const navigate = useNavigate();
  return (
    <header
      className={cn(
        'sticky top-0 z-30 -mx-gutter border-b border-line-soft bg-bg/85 px-gutter pb-2 pt-3 backdrop-blur-xl',
        className,
      )}
    >
      <div className="flex min-h-11 items-center gap-2">
        {back && (
          <button
            type="button"
            aria-label="بازگشت"
            onClick={onBack ?? (() => navigate(-1))}
            className="press -ms-1.5 grid size-10 shrink-0 place-items-center rounded-md text-ink hover:bg-sunken"
          >
            <ChevronRight className="size-[22px]" strokeWidth={2.25} />
          </button>
        )}
        <div className="min-w-0 flex-1">
          <h1 className="truncate text-lg font-extrabold tracking-tight text-ink">{title}</h1>
          {subtitle && <div className="mt-0.5 truncate text-xs text-ink-3">{subtitle}</div>}
        </div>
        {actions && <div className="flex shrink-0 items-center gap-1">{actions}</div>}
      </div>
      {children}
    </header>
  );
}
