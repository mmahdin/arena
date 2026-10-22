import { Link } from 'react-router-dom';
import { Baby, CalendarClock, CircleAlert, CircleCheck, MapPin } from 'lucide-react';
import type { CaseFile, Visit } from '@/types';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { scheduleConflicts, visitsOnSameDay } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { faDuration, faTimeRange } from '@/lib/jalali';
import { CASE_TYPE, PRIORITY } from '@/lib/labels';
import { visitKindLabel } from '@/lib/visit-workflow';

/**
 * A pool visit, shown the way a claim decision is made: what kind of visit,
 * when, roughly where, how long — and whether it fits my day.
 *
 * Identity is deliberately absent. Before claiming, a social worker inside
 * the coverage area only sees what she needs to decide; the family's name,
 * exact address and phone become available with the claim itself.
 */
export function PoolVisitCard({
  visit,
  family,
  myVisits,
  onClaim,
}: {
  visit: Visit;
  family?: CaseFile;
  myVisits: Visit[];
  onClaim: (visit: Visit) => void;
}) {
  const conflicts = scheduleConflicts(visit, myVisits);
  const sameDay = visitsOnSameDay(visit, myVisits);
  const child = family?.children.find((c) => visit.childIds?.includes(c.id)) ?? family?.children[0];
  const urgent = family?.priority === 'critical';

  return (
    <div className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
      <Link to={`/visits/${visit.id}`} className="press block p-4 pb-3 hover:bg-sunken/60">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={visit.purpose === 'initial-assessment' ? 'info' : 'brand'}>
            {visitKindLabel(visit)}
          </Badge>
          {family && <span className="text-xs font-medium text-ink-2">{CASE_TYPE[family.type].label}</span>}
          {urgent && (
            <Badge tone="danger" variant="outline" className="ms-auto">
              {PRIORITY.critical.label}
            </Badge>
          )}
        </div>

        <p className="mt-3 flex items-center gap-2 text-base font-bold text-ink">
          <CalendarClock className="size-[18px] shrink-0 text-brand" aria-hidden />
          <span className="tabular-nums">ساعت {faTimeRange(visit.time, visit.durationMin)}</span>
          <span className="text-xs font-medium text-ink-3">({faDuration(visit.durationMin)})</span>
        </p>

        <p className="clamp-2 mt-1.5 text-sm leading-6 text-ink-2">{visit.summary}</p>

        <div className="mt-2.5 flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-ink-3">
          <span className="flex items-center gap-1">
            <MapPin className="size-3.5 shrink-0" aria-hidden />
            {visit.city}
          </span>
          {child && (
            <span className="flex items-center gap-1">
              <Baby className="size-3.5 shrink-0" aria-hidden />
              کودک {fa(child.age)} ساله
            </span>
          )}
        </div>

        <p className={`mt-2.5 flex items-center gap-1.5 text-xs font-semibold ${conflicts.length ? 'text-warning' : 'text-success'}`}>
          {conflicts.length ? (
            <>
              <CircleAlert className="size-4 shrink-0" aria-hidden />
              هم‌پوشانی با بازدید دیگری از برنامه شما
            </>
          ) : (
            <>
              <CircleCheck className="size-4 shrink-0" aria-hidden />
              {sameDay.length > 0
                ? `بدون تداخل · آن روز ${fa(sameDay.length)} بازدید دارید`
                : 'آن روز برنامه‌تان آزاد است'}
            </>
          )}
        </p>
      </Link>

      <div className="flex items-center justify-between gap-3 border-t border-line-soft px-4 py-2.5">
        <span className="min-w-0 truncate text-2xs text-ink-3">ارجاع: {visit.assignedBy}</span>
        <Button size="sm" onClick={() => onClaim(visit)}>
          پذیرش بازدید
        </Button>
      </div>
    </div>
  );
}
