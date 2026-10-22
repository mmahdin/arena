import { useState } from 'react';
import { CalendarClock, CircleAlert, FolderOpen, MapPin, ShieldCheck } from 'lucide-react';
import type { Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { Button } from '@/components/ui/Button';
import { KeyValue } from '@/components/ui/Field';
import { Sheet } from '@/components/ui/Sheet';
import { scheduleConflicts, visitsOnSameDay } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { deadlineLabel, faDate, faDuration, faTime, faTimeRange } from '@/lib/jalali';
import { CASE_TYPE } from '@/lib/labels';
import { visitKindLabel } from '@/lib/visit-workflow';

/**
 * The single place a pool visit is claimed — opened from the pool list and
 * from the (limited) pool view of a visit. It answers the only question that
 * matters before accepting: does this fit into my day? Claiming itself is
 * one tap; the visit's date and time are already fixed by the unit.
 */
export function ClaimVisitSheet({
  visit,
  open,
  onClose,
  onClaimed,
}: {
  visit: Visit | null;
  open: boolean;
  onClose: () => void;
  /** Called after the visit has become mine (navigation/toast live here). */
  onClaimed?: (visit: Visit) => void;
}) {
  const { cases, myVisits, claimVisit } = useApp();
  const [saving, setSaving] = useState(false);

  if (!visit) return null;

  const family = cases.find((c) => c.id === visit.caseId);
  const conflicts = scheduleConflicts(visit, myVisits);
  const sameDay = visitsOnSameDay(visit, myVisits);
  const dayLabel = deadlineLabel(visit.date);

  const confirm = async () => {
    setSaving(true);
    try {
      const claimed = await claimVisit(visit.id);
      if (claimed) {
        onClose();
        onClaimed?.(claimed);
      }
    } finally {
      setSaving(false);
    }
  };

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title="پذیرش بازدید"
      description={`${visitKindLabel(visit)} · ${dayLabel}، ${faDate(visit.date, { year: false })}`}
      footer={
        <div className="flex gap-2.5">
          <Button variant="secondary" className="flex-1" onClick={onClose}>
            انصراف
          </Button>
          <Button className="flex-1" loading={saving} onClick={() => { void confirm(); }}>
            افزودن به برنامه من
          </Button>
        </div>
      }
    >
      <div className="space-y-4 pb-4">
        <div className="rounded-lg border border-line bg-surface px-4 py-1">
          <KeyValue
            label="زمان"
            icon={<CalendarClock className="size-3.5 text-ink-3" />}
            value={`ساعت ${faTimeRange(visit.time, visit.durationMin)} · ${faDuration(visit.durationMin)}`}
          />
          <KeyValue
            label="محدوده"
            icon={<MapPin className="size-3.5 text-ink-3" />}
            value={visit.city}
          />
          {family && (
            <KeyValue
              label="نوع پرونده"
              icon={<FolderOpen className="size-3.5 text-ink-3" />}
              value={CASE_TYPE[family.type].label}
            />
          )}
        </div>

        {conflicts.length > 0 ? (
          <p className="flex items-start gap-2 rounded-md bg-warning-soft px-3 py-2.5 text-sm leading-relaxed text-ink-2">
            <CircleAlert className="mt-0.5 size-4 shrink-0 text-warning" aria-hidden />
            <span>
              این بازدید با{' '}
              {conflicts.map((c) => `بازدید ساعت ${faTime(c.time)}`).join(' و ')}{' '}
              شما در همان روز هم‌پوشانی دارد. با پذیرش، هر دو در برنامه‌تان می‌مانند.
            </span>
          </p>
        ) : (
          <p className="rounded-md bg-sunken px-3 py-2.5 text-sm leading-relaxed text-ink-2">
            {sameDay.length > 0
              ? `با برنامه ${dayLabel} شما تداخلی ندارد؛ آن روز ${fa(sameDay.length)} بازدید دیگر دارید${sameDay.length ? ` (${sameDay.map((v) => `ساعت ${faTime(v.time)}`).join('، ')})` : ''}.`
              : `در ${dayLabel} بازدید دیگری ندارید.`}
          </p>
        )}

        <p className="flex items-start gap-2 px-1 text-xs leading-relaxed text-ink-3">
          <ShieldCheck className="mt-0.5 size-4 shrink-0" aria-hidden />
          نام، نشانی دقیق و شماره تماس خانواده فقط پس از پذیرش نمایش داده می‌شود و خانواده به حوزه مسئولیت شما اضافه می‌گردد.
        </p>
      </div>
    </Sheet>
  );
}
