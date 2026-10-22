import { Link } from 'react-router-dom';
import { ChevronLeft, FileText, MapPin } from 'lucide-react';
import type { CaseFile, Report, Visit } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { TONE } from '@/components/ui/tone';
import { cn } from '@/lib/cn';
import { faDate, faTime } from '@/lib/jalali';
import { visitKindLabel, visitReportMeta, visitStatusMeta } from '@/lib/visit-workflow';

/**
 * One visit, one row — used by the daily schedule, the «همه بازدیدها» list
 * and the family record, always leading to the same visit screen.
 *
 * The row carries two deliberately separate signals: the visit's own status
 * as the badge (what happened / will happen in the field), and — only for a
 * completed visit — the report's status as a quiet text line with a document
 * icon (where the paperwork stands). Acting on the report happens in the
 * reports queue; the row never pretends the two are one status.
 */
export function VisitRow({
  visit,
  family,
  reports,
  divider = false,
  showFamily = true,
  showDate = true,
}: {
  visit: Visit;
  family?: CaseFile;
  reports: Report[];
  divider?: boolean;
  showFamily?: boolean;
  showDate?: boolean;
}) {
  const status = visitStatusMeta(visit);
  const reportMeta = visitReportMeta(visit, reports);
  return (
    <Link
      to={`/visits/${visit.id}`}
      className={cn(
        'press flex min-h-[76px] items-center gap-3 px-4 py-3.5 text-start transition-colors hover:bg-sunken',
        divider && 'border-b border-line-soft',
      )}
    >
      <div className="min-w-0 flex-1">
        <p className="truncate text-sm font-bold text-ink">
          {showFamily && family ? <PrivacyText value={family.familyName} interactive={false} /> : visitKindLabel(visit)}
        </p>
        <p className="mt-0.5 truncate text-xs text-ink-2">
          {showFamily ? visitKindLabel(visit) : VISIT_PURPOSE_LABEL(visit)}
          <span className="mx-1.5 text-ink-3">·</span>
          {showDate && <>{faDate(visit.date, { year: false })} · </>}{faTime(visit.time)}
        </p>
        <p className="mt-1 flex items-center gap-x-3 gap-y-0.5 text-xs text-ink-3">
          <span className="flex min-w-0 items-center gap-1">
            <MapPin className="size-3.5 shrink-0" aria-hidden />
            <span className="truncate">{visit.city}</span>
          </span>
          {reportMeta && (
            <span className={cn('flex shrink-0 items-center gap-1 font-semibold', TONE[reportMeta.tone].text)}>
              <FileText className="size-3.5 shrink-0" aria-hidden />
              {reportMeta.label}
            </span>
          )}
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <Badge tone={status.tone}>{status.label}</Badge>
        <ChevronLeft className="size-4 text-ink-3" aria-hidden />
      </div>
    </Link>
  );
}

function VISIT_PURPOSE_LABEL(visit: Visit): string {
  return visit.purpose === 'introduction' ? 'جلسه' : 'بازدید';
}
