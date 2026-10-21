import { useEffect, useRef, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { X } from "lucide-react";
import { cn } from "@/lib/cn";
import "./Sheet.css";

export interface SheetProps {
  open: boolean;
  onClose: () => void;
  title?: string;
  description?: string;
  children: ReactNode;
  footer?: ReactNode;
  /** Hides the close button when the sheet is purely informational. */
  dismissible?: boolean;
}

/**
 * Bottom sheet — the product's standard pattern for confirmations and
 * short forms. It is mobile-native, keeps context visible and avoids
 * full-page navigations for micro-tasks.
 */
export function Sheet({
  open,
  onClose,
  title,
  description,
  children,
  footer,
  dismissible = true,
}: SheetProps) {
  const panelRef = useRef<HTMLDivElement>(null);
  const lastFocused = useRef<HTMLElement | null>(null);

  useEffect(() => {
    if (!open) return;
    lastFocused.current = document.activeElement as HTMLElement;
    const originalOverflow = document.body.style.overflow;
    document.body.style.overflow = "hidden";

    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape" && dismissible) onClose();
      if (event.key === "Tab" && panelRef.current) {
        const focusables = panelRef.current.querySelectorAll<HTMLElement>(
          'button:not([disabled]), [href], input:not([disabled]), textarea:not([disabled]), select:not([disabled]), [tabindex]:not([tabindex="-1"])'
        );
        if (focusables.length === 0) return;
        const first = focusables[0];
        const last = focusables[focusables.length - 1];
        if (!event.shiftKey && document.activeElement === last) {
          event.preventDefault();
          first.focus();
        } else if (event.shiftKey && document.activeElement === first) {
          event.preventDefault();
          last.focus();
        }
      }
    };

    document.addEventListener("keydown", onKeyDown);
    requestAnimationFrame(() => {
      const target = panelRef.current?.querySelector<HTMLElement>(
        'input, textarea, [data-autofocus], button'
      );
      (target ?? panelRef.current)?.focus();
    });

    return () => {
      document.removeEventListener("keydown", onKeyDown);
      document.body.style.overflow = originalOverflow;
      lastFocused.current?.focus?.();
    };
  }, [open, onClose, dismissible]);

  if (!open) return null;

  return createPortal(
    <div className="sheet-root">
      <button
        type="button"
        className="sheet-backdrop"
        aria-label="بستن"
        tabIndex={-1}
        onClick={dismissible ? onClose : undefined}
      />
      <div
        className="sheet-panel"
        role="dialog"
        aria-modal="true"
        aria-labelledby={title ? "sheet-title" : undefined}
        ref={panelRef}
        tabIndex={-1}
      >
        <div className="sheet-grabber" aria-hidden="true" />
        {title ? (
          <div className="sheet-head">
            <div>
              <h2 className="sheet-title" id="sheet-title">{title}</h2>
              {description ? <p className="sheet-description">{description}</p> : null}
            </div>
            {dismissible ? (
              <button type="button" className="sheet-close" onClick={onClose} aria-label="بستن">
                <X size={18} />
              </button>
            ) : null}
          </div>
        ) : null}
        <div className={cn("sheet-body", !title && "sheet-body--flush")}>{children}</div>
        {footer ? <div className="sheet-footer">{footer}</div> : null}
      </div>
    </div>,
    document.body
  );
}
