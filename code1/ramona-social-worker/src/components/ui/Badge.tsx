import type { ReactNode } from "react";
import type { Tone } from "@/lib/labels";
import { cn } from "@/lib/cn";
import "./Badge.css";

export interface BadgeProps {
  children: ReactNode;
  tone?: Tone;
  variant?: "soft" | "outline" | "solid";
  size?: "sm" | "md";
  dot?: boolean;
  icon?: ReactNode;
  className?: string;
}

export function Badge({
  children,
  tone = "neutral",
  variant = "soft",
  size = "sm",
  dot,
  icon,
  className,
}: BadgeProps) {
  return (
    <span
      className={cn("badge", `badge--${tone}`, `badge--${variant}`, `badge--${size}`, className)}
    >
      {dot ? <span className="badge__dot" aria-hidden="true" /> : null}
      {icon ? <span className="badge__icon" aria-hidden="true">{icon}</span> : null}
      {children}
    </span>
  );
}
