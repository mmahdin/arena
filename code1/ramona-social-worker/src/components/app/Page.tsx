import type { ReactNode } from "react";
import { cn } from "@/lib/cn";
import "./Page.css";

export function Page({
  children,
  className,
  gutter = true,
}: {
  children: ReactNode;
  className?: string;
  gutter?: boolean;
}) {
  return <div className={cn("page", !gutter && "page--flush", className)}>{children}</div>;
}

export function Section({
  children,
  className,
  as: Tag = "section",
}: {
  children: ReactNode;
  className?: string;
  as?: "section" | "div";
}) {
  return <Tag className={cn("section", className)}>{children}</Tag>;
}

export interface PageHeaderProps {
  eyebrow?: ReactNode;
  title: string;
  description?: ReactNode;
  actions?: ReactNode;
  className?: string;
}

export function PageHeader({ eyebrow, title, description, actions, className }: PageHeaderProps) {
  return (
    <header className={cn("page-header", className)}>
      <div className="page-header__top">
        <div className="page-header__text">
          {eyebrow ? <p className="page-header__eyebrow">{eyebrow}</p> : null}
          <h1 className="page-header__title">{title}</h1>
        </div>
        {actions ? <div className="page-header__actions">{actions}</div> : null}
      </div>
      {description ? <p className="page-header__description">{description}</p> : null}
    </header>
  );
}
