import type { ReactNode } from "react";
import { ChevronLeft } from "lucide-react";
import { cn } from "@/lib/cn";
import "./SectionHeader.css";

export interface SectionHeaderProps {
  title: string;
  lead?: ReactNode;
  action?: { label: string; onClick: () => void; icon?: boolean };
  className?: string;
}

export function SectionHeader({ title, lead, action, className }: SectionHeaderProps) {
  return (
    <div className={cn("section-header", className)}>
      <div className="section-header__text">
        <h2 className="section-header__title">{title}</h2>
        {lead ? <p className="section-header__lead">{lead}</p> : null}
      </div>
      {action ? (
        <button type="button" className="section-header__action" onClick={action.onClick}>
          <span>{action.label}</span>
          {action.icon !== false ? <ChevronLeft size={15} aria-hidden="true" /> : null}
        </button>
      ) : null}
    </div>
  );
}
