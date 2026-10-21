import type { HTMLAttributes, ReactNode } from "react";
import { cn } from "@/lib/cn";
import "./Card.css";

export interface CardProps extends HTMLAttributes<HTMLDivElement> {
  padding?: "none" | "sm" | "md" | "lg";
  tone?: "default" | "muted" | "brand" | "inverse";
  bordered?: boolean;
  children: ReactNode;
}

export function Card({
  padding = "md",
  tone = "default",
  bordered = true,
  className,
  children,
  ...rest
}: CardProps) {
  return (
    <div
      className={cn(
        "card",
        `card--pad-${padding}`,
        `card--${tone}`,
        bordered && "card--bordered",
        className
      )}
      {...rest}
    >
      {children}
    </div>
  );
}
