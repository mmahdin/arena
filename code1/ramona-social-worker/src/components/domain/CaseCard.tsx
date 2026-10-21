import { Link } from "react-router-dom";
import { CalendarDays, ChevronLeft, FolderClosed, MapPin } from "lucide-react";
import type { Case } from "@/types";
import { formatDayLabel, formatTime, parse } from "@/lib/date";
import { cn } from "@/lib/cn";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { RiskBadge, CaseStageBadge } from "./Badges";
import "./CaseCard.css";

export function CaseCard({ familyCase, now }: { familyCase: Case; now: Date }) {
  const childCount = familyCase.children.length;
  const progress = familyCase.stageProgress ?? 0;
  const nextVisit = familyCase.nextVisitAt ? parse(familyCase.nextVisitAt) : null;

  return (
    <Link to={`/cases/${familyCase.id}`} className="case-card">
      <span className="case-card__head">
        <span className={cn("case-card__mark", familyCase.riskLevel === "high" && "is-alert")}>
          <FolderClosed size={18} aria-hidden="true" />
        </span>
        <span className="case-card__identity">
          <span className="case-card__title">{familyCase.title}</span>
          <span className="case-card__sub">
            پرونده <span className="num">{familyCase.caseNumber}</span>
            {childCount > 0 ? (
              <>
                <span className="case-card__sep" aria-hidden="true">·</span>
                {childCount === 1
                  ? familyCase.children[0].fullName
                  : `${new Intl.NumberFormat("fa-IR").format(childCount)} کودک`}
              </>
            ) : (
              <>
                <span className="case-card__sep" aria-hidden="true">·</span>
                متقاضی
              </>
            )}
          </span>
        </span>
        <ChevronLeft size={16} className="case-card__chev" aria-hidden="true" />
      </span>

      <span className="case-card__badges">
        <CaseStageBadge stage={familyCase.stage} />
        {familyCase.riskLevel !== "low" ? <RiskBadge risk={familyCase.riskLevel} /> : null}
      </span>

      {familyCase.stage !== "closed" ? (
        <span className="case-card__progress">
          <span className="case-card__progress-label">
            <span>پیشرفت مرحله</span>
            <span className="num">{new Intl.NumberFormat("fa-IR").format(progress)}٪</span>
          </span>
          <ProgressBar
            value={progress}
            tone={familyCase.riskLevel === "high" ? "warning" : "brand"}
            label={`پیشرفت ${familyCase.title}`}
          />
        </span>
      ) : null}

      <span className="case-card__foot">
        {nextVisit ? (
          <span className="case-card__next">
            <CalendarDays size={13} aria-hidden="true" />
            بازدید بعدی: {formatDayLabel(nextVisit, now)}، {formatTime(nextVisit)}
          </span>
        ) : (
          <span className="case-card__next case-card__next--muted">
            <MapPin size={13} aria-hidden="true" />
            {familyCase.address.district}
          </span>
        )}
        {nextVisit ? (
          <span className="case-card__district truncate">{familyCase.address.district}</span>
        ) : null}
      </span>
    </Link>
  );
}
