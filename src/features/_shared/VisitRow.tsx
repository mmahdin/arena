import { Link } from 'react-router-dom';
import { ChevronLeft, MapPin } from 'lucide-react';
import type { CaseFile, Report, Visit } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { cn } from '@/lib/cn';
import { faDate, faTime } from '@/lib/jalali';
import { visitKindLabel, visitProgress } from '@/lib/visit-workflow';

/** Consistent entry point: home, visit list and family history all lead to a visit. */
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
  const progress = visitProgress(visit, reports);
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
        <p className="mt-1 flex items-center gap-1 truncate text-xs text-ink-3">
          <MapPin className="size-3.5 shrink-0" aria-hidden />
          <span className="truncate">{visit.city}</span>
        </p>
      </div>
      <div className="flex shrink-0 items-center gap-1.5">
        <Badge tone={progress.tone}>{progress.label}</Badge>
        <ChevronLeft className="size-4 text-ink-3" aria-hidden />
      </div>
    </Link>
  );
}

function VISIT_PURPOSE_LABEL(visit: Visit): string {
  return visit.purpose === 'introduction' ? 'جلسه' : 'بازدید';
}
