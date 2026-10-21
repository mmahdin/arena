import { Link } from "react-router-dom";
import { CalendarDays } from "lucide-react";
import type { Case, Child } from "@/types";
import { formatDayLabel, formatTime, parse } from "@/lib/date";
import { cn } from "@/lib/cn";
import { Avatar } from "@/components/ui/Avatar";
import { WellbeingBadge } from "./Badges";
import "./MonitoringCard.css";

export function CycleDots({
  cycle,
  total,
  tone = "brand",
}: {
  cycle: number;
  total: number;
  tone?: "brand" | "warning" | "danger";
}) {
  return (
    <span
      className={cn("cycle-dots", `cycle-dots--${tone}`)}
      role="img"
      aria-label={`دوره ${cycle} از ${total}`}
    >
      {Array.from({ length: total }).map((_, index) => {
        const step = index + 1;
        return (
          <span
            key={step}
            className={cn(
              "cycle-dots__dot",
              step < cycle && "is-done",
              step === cycle && "is-current"
            )}
          />
        );
      })}
    </span>
  );
}

export function MonitoringCard({
  child,
  familyCase,
  now,
}: {
  child: Child;
  familyCase: Case;
  now: Date;
}) {
  const monitoring = child.monitoring;
  if (!monitoring) return null;
  const tone =
    monitoring.wellbeing === "critical"
      ? "danger"
      : monitoring.wellbeing === "needs_attention"
        ? "warning"
        : "brand";
  const next = monitoring.nextVisitAt ? parse(monitoring.nextVisitAt) : null;

  return (
    <Link
      to={`/cases/${familyCase.id}?tab=summary`}
      className={cn("monitor-card", `monitor-card--${tone}`)}
    >
      <span className="monitor-card__head">
        <Avatar name={child.fullName} size="md" />
        <span className="monitor-card__identity">
          <span className="monitor-card__name">{child.fullName}</span>
          <span className="monitor-card__sub">
            <span className="num">{new Intl.NumberFormat("fa-IR").format(child.age)}</span> ساله
            <span aria-hidden="true">·</span>
            {familyCase.title}
          </span>
        </span>
        <WellbeingBadge wellbeing={monitoring.wellbeing} />
      </span>

      <span className="monitor-card__cycle">
        <span className="monitor-card__cycle-row">
          <span>
            دوره پایش{" "}
            <span className="num">{new Intl.NumberFormat("fa-IR").format(monitoring.cycle)}</span> از{" "}
            <span className="num">{new Intl.NumberFormat("fa-IR").format(monitoring.totalCycles)}</span>
          </span>
          {next ? (
            <span className="monitor-card__next">
              <CalendarDays size={12} aria-hidden="true" />
              {formatDayLabel(next, now)}، {formatTime(next)}
            </span>
          ) : null}
        </span>
        <CycleDots
          cycle={monitoring.cycle}
          total={monitoring.totalCycles}
          tone={tone === "brand" ? "brand" : tone}
        />
      </span>

      {monitoring.note ? (
        <span className="monitor-card__note">{monitoring.note}</span>
      ) : null}
    </Link>
  );
}
