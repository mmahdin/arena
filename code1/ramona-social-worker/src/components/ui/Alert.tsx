import type { ReactNode } from "react";
import type { Tone } from "@/lib/labels";
import { cn } from "@/lib/cn";
import "./Alert.css";

export interface AlertProps {
  tone?: Extract<Tone, "brand" | "warning" | "danger" | "info" | "neutral">;
  icon?: ReactNode;
  title?: string;
  children: ReactNode;
  action?: ReactNode;
  className?: string;
}

export function Alert({ tone = "info", icon, title, children, action, className }: AlertProps) {
  return (
    <div className={cn("alert", `alert--${tone}`, className)}>
      {icon ? (
        <span className="alert__icon" aria-hidden="true">
          {icon}
        </span>
      ) : null}
      <div className="alert__body">
        {title ? <p className="alert__title">{title}</p> : null}
        <div className="alert__text">{children}</div>
        {action ? <div className="alert__action">{action}</div> : null}
      </div>
    </div>
  );
}
