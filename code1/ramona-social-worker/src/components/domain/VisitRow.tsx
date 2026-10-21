import { Link } from "react-router-dom";
import { Clock, MapPin, Video, Building2 } from "lucide-react";
import type { Visit } from "@/types";
import { formatDuration, formatTime, parse } from "@/lib/date";
import { visitKindLabels } from "@/lib/labels";
import { cn } from "@/lib/cn";
import { VisitKindIcon, VisitStatusBadge } from "./Badges";
import "./VisitRow.css";

function placeLabel(visit: Visit): string {
  return visit.kind === "office_meeting" ? visit.address.district : visit.address.district;
}

/** Dense list row used on the Visits screen and inside case files. */
export function VisitRow({ visit, compact }: { visit: Visit; compact?: boolean }) {
  const start = parse(visit.startsAt);
  return (
    <Link to={`/visits/${visit.id}`} className={cn("visit-row", compact && "visit-row--compact")}>
      <span className="visit-row__time">
        <span className="visit-row__start num">{formatTime(start)}</span>
        <span className="visit-row__duration num">
          <Clock size={11} aria-hidden="true" />
          {formatDuration(visit.durationMin)}
        </span>
      </span>
      <span className="visit-row__body">
        <span className="visit-row__top">
          <span className="visit-row__kind">
            <VisitKindIcon kind={visit.kind} size={14} />
            {visitKindLabels[visit.kind]}
          </span>
          {visit.status !== "scheduled" ? (
            <VisitStatusBadge status={visit.status} />
          ) : visit.mandatory ? (
            <span className="visit-row__mandatory">الزامی</span>
          ) : null}
        </span>
        <span className="visit-row__title">{visit.caseTitle}</span>
        <span className="visit-row__meta">
          {visit.kind === "office_meeting" ? (
            <Building2 size={13} aria-hidden="true" />
          ) : (
            <MapPin size={13} aria-hidden="true" />
          )}
          <span className="truncate">{placeLabel(visit)}</span>
        </span>
      </span>
    </Link>
  );
}

/** Vertical agenda rail — the visual anchor of the home screen. */
export function VisitTimeline({
  visits,
  emphasisId,
}: {
  visits: Visit[];
  emphasisId?: string;
}) {
  return (
    <ol className="timeline">
      {visits.map((visit) => {
        const start = parse(visit.startsAt);
        const emphasized = visit.id === emphasisId;
        return (
          <li className="timeline__item" key={visit.id}>
            <div className="timeline__rail">
              <span className="timeline__line" aria-hidden="true" />
              <span className={cn("timeline__dot", emphasized && "is-emphasized")} aria-hidden="true" />
              <span className="timeline__time num">{formatTime(start)}</span>
            </div>
            <Link
              to={`/visits/${visit.id}`}
              className={cn("timeline__card", emphasized && "is-emphasized")}
            >
              <span className="timeline__card-top">
                <span className="timeline__kind">
                  <VisitKindIcon kind={visit.kind} size={14} />
                  {visitKindLabels[visit.kind]}
                  {visit.kind === "office_meeting" ? (
                    <Video size={12} aria-hidden="true" />
                  ) : null}
                </span>
                {visit.status !== "scheduled" ? (
                  <VisitStatusBadge status={visit.status} />
                ) : visit.mandatory ? (
                  <span className="visit-row__mandatory">الزامی</span>
                ) : null}
              </span>
              <span className="timeline__title">{visit.caseTitle}</span>
              <span className="timeline__meta">
                <MapPin size={12} aria-hidden="true" />
                {visit.address.district}
                <span className="timeline__separator" aria-hidden="true">·</span>
                <span>{formatDuration(visit.durationMin)}</span>
              </span>
            </Link>
          </li>
        );
      })}
    </ol>
  );
}
