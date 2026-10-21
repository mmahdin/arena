import type { ReactNode } from "react";
import { Phone } from "lucide-react";
import { Avatar } from "@/components/ui/Avatar";
import { telHref } from "@/lib/phone";
import { cn } from "@/lib/cn";
import "./People.css";

export function PersonRow({
  name,
  subtitle,
  phone,
  badge,
  tone,
  className,
}: {
  name: string;
  subtitle?: ReactNode;
  phone?: string;
  badge?: ReactNode;
  tone?: "brand" | "info" | "accent" | "warning" | "neutral";
  className?: string;
}) {
  return (
    <div className={cn("person-row", className)}>
      <Avatar name={name} tone={tone} size="md" />
      <div className="person-row__identity">
        <p className="person-row__name">{name}</p>
        {subtitle ? <p className="person-row__subtitle">{subtitle}</p> : null}
      </div>
      {badge}
      {phone ? (
        <a
          className="person-row__call"
          href={telHref(phone)}
          aria-label={`تماس با ${name}`}
        >
          <Phone size={16} aria-hidden="true" />
        </a>
      ) : null}
    </div>
  );
}

export function InfoRow({
  icon,
  label,
  value,
  action,
  className,
}: {
  icon?: ReactNode;
  label: string;
  value: ReactNode;
  action?: ReactNode;
  className?: string;
}) {
  return (
    <div className={cn("info-row", className)}>
      {icon ? <span className="info-row__icon" aria-hidden="true">{icon}</span> : null}
      <div className="info-row__text">
        <p className="info-row__label">{label}</p>
        <p className="info-row__value">{value}</p>
      </div>
      {action ? <div className="info-row__action">{action}</div> : null}
    </div>
  );
}

export function DataGrid({ children }: { children: ReactNode }) {
  return <div className="data-grid">{children}</div>;
}
