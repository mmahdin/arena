import type { ReactNode } from "react";
import { useNavigate } from "react-router-dom";
import { ArrowRight } from "lucide-react";
import { cn } from "@/lib/cn";
import "./TopBar.css";

export interface TopBarProps {
  title?: string;
  actions?: ReactNode;
  onBack?: () => void;
  /** Solid removes the translucency, used on long document screens. */
  variant?: "blur" | "solid";
  className?: string;
}

export function TopBar({ title, actions, onBack, variant = "blur", className }: TopBarProps) {
  const navigate = useNavigate();

  return (
    <header className={cn("topbar", `topbar--${variant}`, className)}>
      <div className="topbar__inner">
        <button
          type="button"
          className="topbar__back"
          aria-label="بازگشت"
          onClick={() => (onBack ? onBack() : navigate(-1))}
        >
          <ArrowRight size={20} aria-hidden="true" />
        </button>
        <h1 className="topbar__title">{title}</h1>
        <div className="topbar__actions">{actions}</div>
      </div>
    </header>
  );
}
