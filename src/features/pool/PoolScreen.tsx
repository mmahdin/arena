import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CalendarClock, CheckCircle2, Clock3, Inbox, Lock, MapPin, TriangleAlert } from 'lucide-react';
import type { Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { EmptyState } from '@/components/ui/EmptyState';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { Sheet } from '@/components/ui/Sheet';
import { fa } from '@/lib/digits';
import { addDays, dayKey, faDate, faDuration, faTime, faTimeRange, relativeTime } from '@/lib/jalali';
import { VISIT_PURPOSE } from '@/lib/labels';
import { conflictingVisit } from '@/lib/domain';
import { visitKindLabel } from '@/lib/visit-workflow';

type PoolFilter = 'all' | 'today' | 'week';

/**
 * Open visit pool.
 *
 * Dispatch publishes visits that already have a date and a time but no owner.
 * The screen is a decision surface, not a case browser: enough information to
 * judge the trip and the slot, with family details withheld until the worker
 * takes responsibility for the visit.
 */
export function PoolScreen() {
  const { poolVisits, visits, cases, claimVisit, toast } = useApp();
  const navigate = useNavigate();
  const now = useMemo(() => new Date(), []);
  const todayKey = dayKey(now);
  const tomorrowKey = dayKey(addDays(now, 1));
  const weekEnd = dayKey(addDays(now, 6));
  const dayLabel = (date: string) =>
    date === todayKey ? 'امروز' : date === tomorrowKey ? 'فردا' : faDate(date, { weekday: true, year: false });
  const [filter, setFilter] = useState<PoolFilter>('all');
  const [candidate, setCandidate] = useState<Visit | null>(null);
  const [saving, setSaving] = useState(false);

  const counts: Record<PoolFilter, number> = {
    all: poolVisits.length,
    today: poolVisits.filter((visit) => visit.date === todayKey).length,
    week: poolVisits.filter((visit) => visit.date >= todayKey && visit.date <= weekEnd).length,
  };

  const rows = useMemo(
    () =>
      poolVisits.filter((visit) => {
        if (filter === 'today') return visit.date === todayKey;
        if (filter === 'week') return visit.date >= todayKey && visit.date <= weekEnd;
        return true;
      }),
    [poolVisits, filter, todayKey, weekEnd],
  );

  const groups = useMemo(() => {
    const map = new Map<string, Visit[]>();
    for (const visit of rows) map.set(visit.date, [...(map.get(visit.date) ?? []), visit]);
    return [...map.entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [rows]);

  const conflict = candidate ? conflictingVisit(candidate, visits) : undefined;
  const conflictFamily = conflict ? cases.find((family) => family.id === conflict.caseId) : undefined;

  const accept = async () => {
    if (!candidate) return;
    setSaving(true);
    try {
      const accepted = await claimVisit(candidate.id);
      if (!accepted) {
        toast({ title: 'این بازدید دیگر آزاد نیست', description: 'ممکن است همکاری آن را پذیرفته باشد.', tone: 'warning' });
        return;
      }
      const visit = candidate;
      setCandidate(null);
      toast({
        title: 'بازدید پذیرفته شد',
        description: `${visitKindLabel(visit)} · ${faDate(visit.date, { weekday: true, year: false })} · ${faTime(visit.time)}`,
        tone: 'success',
        action: { label: 'مشاهده در برنامه', onClick: () => navigate(`/visits?date=${visit.date}`) },
      });
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="بازدیدهای آزاد"
        subtitle={poolVisits.length > 0 ? `${fa(poolVisits.length)} بازدید بدون مددکار` : 'در حال حاضر خالی است'}
      />

      <div className="pb-8 pt-4">
        <p className="flex items-start gap-2 rounded-lg bg-sunken/70 px-3.5 py-3 text-xs leading-relaxed text-ink-2">
          <Lock className="mt-0.5 size-4 shrink-0 text-ink-3" aria-hidden />
          <span>
            بازدیدها با زمان مشخص منتشر می‌شوند. با پذیرش، بازدید به برنامه و تقویم شما اضافه می‌شود؛
            اطلاعات خانواده نیز پس از پذیرش نمایش داده می‌شود.
          </span>
        </p>

        <div className="mt-4">
          <ChipRail>
            <Chip selected={filter === 'all'} count={counts.all} onClick={() => setFilter('all')}>همه</Chip>
            <Chip selected={filter === 'today'} count={counts.today} onClick={() => setFilter('today')}>امروز</Chip>
            <Chip selected={filter === 'week'} count={counts.week} onClick={() => setFilter('week')}>این هفته</Chip>
          </ChipRail>
        </div>

        {groups.length === 0 ? (
          <EmptyState
            className="mt-2"
            icon={<Inbox className="size-6" />}
            title={filter === 'all' ? 'بازدید آزادی برای پذیرش نیست' : 'در این بازه بازدید آزادی نیست'}
            description="بازدیدهای جدید توسط واحد تخصیص منتشر می‌شوند و همین‌جا دیده می‌شوند."
            action={<Button variant="secondary" onClick={() => navigate('/visits')}>مشاهده برنامه من</Button>}
          />
        ) : (
          <div className="mt-5 space-y-6">
            {groups.map(([date, dayVisits]) => (
              <section key={date} aria-label={faDate(date, { weekday: true })}>
                <div className="mb-2.5 flex items-baseline justify-between gap-3 px-1">
                  <h2 className="flex items-center gap-1.5 text-sm font-bold text-ink">
                    <CalendarClock className="size-4 text-brand" aria-hidden />
                    {dayLabel(date)}
                  </h2>
                  <span className="shrink-0 text-xs text-ink-3">{fa(dayVisits.length)} بازدید</span>
                </div>
                <ul className="space-y-3">
                  {dayVisits.map((visit) => (
                    <li key={visit.id}>
                      <PoolCard visit={visit} onAccept={() => setCandidate(visit)} />
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        )}
      </div>

      <Sheet
        open={Boolean(candidate)}
        onClose={() => setCandidate(null)}
        title="پذیرش بازدید"
        description={candidate ? `${faDate(candidate.date, { weekday: true })} · ${faTimeRange(candidate.time, candidate.durationMin)}` : undefined}
        footer={
          <div className="flex gap-2.5">
            <Button variant="secondary" className="flex-1" onClick={() => setCandidate(null)}>انصراف</Button>
            <Button className="flex-1" loading={saving} onClick={() => void accept()}>پذیرش بازدید</Button>
          </div>
        }
      >
        {candidate && (
          <div className="space-y-4 pb-4">
            <div className="rounded-lg border border-line bg-surface px-4 py-3">
              <p className="text-sm font-bold text-ink">{VISIT_PURPOSE[candidate.purpose].label}</p>
              <p className="mt-1 flex items-center gap-1.5 text-xs text-ink-3">
                <MapPin className="size-3.5" aria-hidden />
                {candidate.city}
                <span aria-hidden>·</span>
                {faDuration(candidate.durationMin)}
              </p>
              {candidate.pool?.note && (
                <p className="mt-2.5 border-t border-line-soft pt-2.5 text-xs leading-relaxed text-ink-2">
                  {candidate.pool.note}
                </p>
              )}
            </div>

            {conflict ? (
              <div className="rounded-lg border border-warning/30 bg-warning-soft px-3.5 py-3 text-xs leading-relaxed text-ink-2">
                <p className="mb-1 flex items-center gap-1.5 font-bold text-warning">
                  <TriangleAlert className="size-4" aria-hidden />
                  تداخل با برنامه شما
                </p>
                <p>
                  این بازدید با
                  {conflictFamily ? <> <PrivacyText value={conflictFamily.familyName} interactive={false} /> </> : ' بازدید دیگری '}
                  از ساعت {faTimeRange(conflict.time, conflict.durationMin)} تداخل دارد. با پذیرش، دو بازدید هم‌زمان خواهید داشت.
                </p>
              </div>
            ) : (
              <p className="flex items-center gap-2 rounded-lg bg-sunken/70 px-3.5 py-3 text-xs font-semibold text-ink-2">
                <CheckCircle2 className="size-4 shrink-0 text-success" aria-hidden />
                با برنامه فعلی شما تداخل ندارد
              </p>
            )}

            <p className="text-xs leading-relaxed text-ink-3">
              با پذیرش، این بازدید از مخزن آزاد خارج و به برنامه شخصی و تقویم شما اضافه می‌شود.
              {candidate.pool?.publishedAt && ` منتشرشده ${relativeTime(candidate.pool.publishedAt)}.`}
            </p>
          </div>
        )}
      </Sheet>
    </Screen>
  );
}

function PoolCard({ visit, onAccept }: { visit: Visit; onAccept: () => void }) {
  return (
    <Card className="space-y-3">
      <div>
        <div className="flex flex-wrap items-center gap-x-2 gap-y-1.5">
          <span className="inline-flex items-center gap-1 text-sm font-extrabold text-ink tabular-nums">
            <Clock3 className="size-4 text-brand" aria-hidden />
            {faTime(visit.time)}
          </span>
          <Badge tone={VISIT_PURPOSE[visit.purpose].tone}>{VISIT_PURPOSE[visit.purpose].label}</Badge>
          <span className="text-2xs text-ink-3">{faDuration(visit.durationMin)}</span>
        </div>
        <p className="mt-2 flex items-center gap-1.5 text-sm font-semibold text-ink">
          <MapPin className="size-4 shrink-0 text-ink-3" aria-hidden />
          {visit.city}
        </p>
        {visit.pool?.note && (
          <p className="mt-2 rounded-md bg-sunken/70 px-3 py-2 text-xs leading-relaxed text-ink-2">{visit.pool.note}</p>
        )}
      </div>
      <div className="flex items-center justify-between gap-3 border-t border-line-soft pt-3">
        <span className="min-w-0 truncate text-2xs text-ink-3">
          {visit.pool?.publishedAt ? `منتشرشده ${relativeTime(visit.pool.publishedAt)}` : 'بدون مددکار'}
        </span>
        <Button onClick={onAccept} aria-label={`پذیرش بازدید ${faTime(visit.time)} — ${visit.city}`}>
          پذیرش بازدید
        </Button>
      </div>
    </Card>
  );
}
