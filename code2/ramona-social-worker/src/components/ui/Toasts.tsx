import { useEffect, useState } from 'react';
import { createPortal } from 'react-dom';
import { AlertTriangle, CheckCircle2, Info, X } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useApp } from '@/store/app-store';
import { OVERLAY_ROOT_ID } from './Sheet';

const ICONS = {
  success: CheckCircle2,
  warning: AlertTriangle,
  danger: AlertTriangle,
  info: Info,
  brand: CheckCircle2,
  neutral: Info,
  accent: Info,
} as const;

const SKIN = {
  success: 'bg-success-soft text-success border-success/30',
  warning: 'bg-warning-soft text-warning border-warning/30',
  danger: 'bg-danger-soft text-danger border-danger/30',
  info: 'bg-info-soft text-info border-info/30',
  brand: 'bg-brand-soft text-brand-ink border-brand-muted',
  neutral: 'bg-raised text-ink border-line',
  accent: 'bg-accent-soft text-accent-ink border-accent/30',
} as const;

/**
 * Confirmation layer.
 *
 * Every mutating action answers with a toast — the social worker is often
 * looking at a piece of paper, not the screen, when they tap. Toasts sit above
 * the tab bar and never block it.
 */
export function ToastHost() {
  const { toasts, dismissToast } = useApp();
  const [root, setRoot] = useState<HTMLElement | null>(null);

  useEffect(() => {
    setRoot(document.getElementById(OVERLAY_ROOT_ID));
  }, []);

  if (!root || toasts.length === 0) return null;

  return createPortal(
    <div className="pointer-events-none absolute inset-x-0 bottom-0 z-[60] flex flex-col items-center gap-2 px-gutter pb-28">
      {toasts.map((t) => {
        const Icon = ICONS[t.tone ?? 'success'];
        return (
          <div
            key={t.id}
            role="status"
            aria-live="polite"
            className={cn(
              'animate-toast-in pointer-events-auto flex w-full items-start gap-3 rounded-md border px-3.5 py-3 shadow-lg backdrop-blur-sm',
              SKIN[t.tone ?? 'success'],
            )}
          >
            <Icon className="mt-0.5 size-5 shrink-0" strokeWidth={2.25} aria-hidden />
            <div className="min-w-0 flex-1">
              <p className="text-sm font-bold leading-snug">{t.title}</p>
              {t.description && <p className="mt-0.5 text-xs leading-relaxed opacity-80">{t.description}</p>}
              {t.action && (
                <button
                  type="button"
                  onClick={() => {
                    t.action?.onClick();
                    dismissToast(t.id);
                  }}
                  className="press mt-2 text-xs font-bold underline underline-offset-4"
                >
                  {t.action.label}
                </button>
              )}
            </div>
            <button
              type="button"
              aria-label="بستن پیام"
              onClick={() => dismissToast(t.id)}
              className="press -me-1 grid size-7 shrink-0 place-items-center rounded-sm opacity-60 hover:opacity-100"
            >
              <X className="size-4" />
            </button>
          </div>
        );
      })}
    </div>,
    root,
  );
}
