import { useEffect, useRef, useState, type ReactNode } from 'react';
import { createPortal } from 'react-dom';
import { X } from 'lucide-react';
import { cn } from '@/lib/cn';

/**
 * Bottom sheet — the app's modal primitive.
 *
 * Sheets are used instead of centred dialogs: they keep the hand in its
 * natural position, they can host long content and keyboards, and they read as
 * "an extension of this screen" rather than an interruption.
 *
 * Portals into `#app-overlay` (inside the app viewport, never `document.body`)
 * so overlays stay inside the device frame on wide screens.
 */

export const OVERLAY_ROOT_ID = 'app-overlay';

let openSheets = 0;

function lockScroll(locked: boolean) {
  const el = document.getElementById('app-scroll');
  if (!el) return;
  if (locked) {
    openSheets += 1;
    el.style.overflow = 'hidden';
  } else {
    openSheets = Math.max(0, openSheets - 1);
    if (openSheets === 0) el.style.overflow = '';
  }
}

export interface SheetProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: ReactNode;
  /** Sticky action area at the bottom of the sheet. */
  footer?: ReactNode;
  /** Sheet grows to most of the viewport — for long forms. */
  tall?: boolean;
}

export function Sheet({ open, onClose, title, description, children, footer, tall }: SheetProps) {
  const [root, setRoot] = useState<HTMLElement | null>(null);
  const panelRef = useRef<HTMLDivElement>(null);
  const titleId = useRef(`sheet-${Math.random().toString(36).slice(2, 8)}`).current;

  useEffect(() => {
    setRoot(document.getElementById(OVERLAY_ROOT_ID));
  }, []);

  useEffect(() => {
    if (!open) return;
    lockScroll(true);
    const onKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
      if (e.key === 'Tab' && panelRef.current) {
        const nodes = panelRef.current.querySelectorAll<HTMLElement>(
          'button, [href], input, textarea, select, [tabindex]:not([tabindex="-1"])',
        );
        if (nodes.length === 0) return;
        const first = nodes[0];
        const last = nodes[nodes.length - 1];
        if (e.shiftKey && document.activeElement === first) {
          e.preventDefault();
          last.focus();
        } else if (!e.shiftKey && document.activeElement === last) {
          e.preventDefault();
          first.focus();
        }
      }
    };
    document.addEventListener('keydown', onKey);
    const t = window.setTimeout(() => panelRef.current?.querySelector<HTMLElement>('button, input, textarea')?.focus(), 60);
    return () => {
      document.removeEventListener('keydown', onKey);
      window.clearTimeout(t);
      lockScroll(false);
    };
  }, [open, onClose]);

  if (!open || !root) return null;

  return createPortal(
    <div className="pointer-events-auto absolute inset-0 z-50 flex items-end" role="presentation">
      <button
        type="button"
        aria-label="بستن"
        onClick={onClose}
        className="animate-fade-in absolute inset-0 bg-[rgb(var(--s-scrim)/0.45)] backdrop-blur-[2px]"
      />
      <div
        ref={panelRef}
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? titleId : undefined}
        className={cn(
          'animate-sheet-up relative flex w-full flex-col overflow-hidden rounded-t-2xl bg-surface shadow-lg',
          tall ? 'max-h-[92%]' : 'max-h-[86%]',
        )}
      >
        <div className="flex justify-center pt-2.5">
          <span className="h-1 w-10 rounded-pill bg-line-strong" aria-hidden />
        </div>

        {(title || description) && (
          <div className="flex items-start justify-between gap-3 px-5 pb-3 pt-3">
            <div className="min-w-0">
              <h2 id={titleId} className="text-lg font-bold text-ink leading-snug">
                {title}
              </h2>
              {description && <p className="mt-1 text-sm leading-relaxed text-ink-3">{description}</p>}
            </div>
            <button
              type="button"
              onClick={onClose}
              aria-label="بستن"
              className="press -me-1 grid size-9 shrink-0 place-items-center rounded-md text-ink-3 hover:bg-sunken"
            >
              <X className="size-5" />
            </button>
          </div>
        )}

        <div className="min-h-0 flex-1 overflow-y-auto px-5 pb-2">{children}</div>

        {footer && (
          <div className="safe-bottom border-t border-line-soft bg-surface px-5 pb-4 pt-3">{footer}</div>
        )}
      </div>
    </div>,
    root,
  );
}

/** A row inside a sheet that behaves like a selectable option. */
export function SheetOption({
  selected,
  onClick,
  icon,
  title,
  description,
}: {
  selected?: boolean;
  onClick: () => void;
  icon?: ReactNode;
  title: string;
  description?: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={selected}
      className={cn(
        'press flex w-full items-start gap-3 rounded-md border p-3.5 text-start',
        selected ? 'border-brand bg-brand-soft' : 'border-line bg-surface hover:bg-sunken',
      )}
    >
      {icon && (
        <span className={cn('grid size-9 shrink-0 place-items-center rounded-sm', selected ? 'bg-brand text-white' : 'bg-sunken text-ink-2')}>
          {icon}
        </span>
      )}
      <span className="min-w-0 flex-1">
        <span className={cn('block text-base font-semibold', selected ? 'text-brand-ink' : 'text-ink')}>{title}</span>
        {description && <span className="mt-0.5 block text-xs leading-relaxed text-ink-3">{description}</span>}
      </span>
    </button>
  );
}
