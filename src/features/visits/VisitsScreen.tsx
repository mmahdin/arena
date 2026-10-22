import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { Bell, CalendarCheck2, ChevronLeft, CircleAlert, Inbox } from 'lucide-react';
import type { Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Card } from '@/components/ui/Card';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconButton } from '@/components/ui/IconButton';
import { Avatar } from '@/components/ui/Avatar';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { VisitRow } from '@/features/_shared/VisitRow';
import { VISIT_BUCKET_LABEL, visitBucket, type VisitBucket } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { addDays, addMonthsJalali, dayKey, faDate, toGregorian, toJalali } from '@/lib/jalali';
import { visitChronological } from '@/lib/visit-workflow';
import { VisitCalendar } from './VisitCalendar';

/** Ignore malformed deep links rather than showing an invalid calendar date. */
function parseDate(value: string | null): Date | undefined {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return undefined;
  const date = new Date(`${value}T12:00:00`);
  return !Number.isNaN(date.getTime()) && dayKey(date) === value ? date : undefined;
}

type ViewMode = 'day' | 'list';
type ListFilter = VisitBucket | 'all';

const FILTERS: ListFilter[] = ['upcoming', 'needs-outcome', 'completed', 'not-done', 'all'];
/** Grouped «همه» reads by urgency, not by the chip order. */
const GROUP_ORDER: VisitBucket[] = ['needs-outcome', 'upcoming', 'completed', 'not-done'];

const filterLabel = (f: ListFilter) => (f === 'all' ? 'همه' : VISIT_BUCKET_LABEL[f]);

/** Needs-action and upcoming read forward in time; history reads backwards. */
function sortForBucket(bucket: VisitBucket, rows: Visit[]): Visit[] {
  const sorted = [...rows].sort(visitChronological);
  return bucket === 'completed' || bucket === 'not-done' ? sorted.reverse() : sorted;
}

function isListFilter(value: string | null): value is ListFilter {
  return value === 'all' || value === 'upcoming' || value === 'needs-outcome' || value === 'completed' || value === 'not-done';
}

/**
 * «بازدیدهای من» — one destination with two views of the same visits:
 *
 * - «برنامه روزانه» is the home screen: the calendar and one day at a time.
 * - «همه بازدیدها» is the management view: every visit I own, by where it
 *   stands in its lifecycle.
 *
 * They are two views, not two places — switching never changes what a visit
 * is, and every row in both views opens the same visit screen. Visits that
 * are not mine yet live in «بازدیدهای آزاد», not here.
 */
export function VisitsScreen() {
  const { myVisits, poolVisits, cases, reports, user, unreadCount } = useApp();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const focusedDate = params.get('date');
  const view: ViewMode = params.get('view') === 'list' ? 'list' : 'day';
  const filterParam = params.get('status');
  const filter: ListFilter = isListFilter(filterParam) ? filterParam : 'upcoming';
  const today = useMemo(() => new Date(), []);
  const [selected, setSelected] = useState(() => dayKey(parseDate(focusedDate) ?? today));
  const [expanded, setExpanded] = useState(false);

  // Also handle dates supplied by other screens (e.g. /visits?date=...).
  useEffect(() => {
    const linked = parseDate(focusedDate);
    if (linked) setSelected(dayKey(linked));
  }, [focusedDate]);

  const updateParams = (patch: Record<string, string | null>) => {
    const next = new URLSearchParams(params);
    for (const [key, value] of Object.entries(patch)) {
      if (value === null) next.delete(key);
      else next.set(key, value);
    }
    setParams(next, { replace: true });
  };

  const byDay = useMemo(() => {
    const dates = new Map<string, Visit[]>();
    for (const visit of myVisits) {
      const dayVisits = dates.get(visit.date) ?? [];
      dayVisits.push(visit);
      dates.set(visit.date, dayVisits);
    }
    for (const dayVisits of dates.values()) dayVisits.sort(visitChronological);
    return dates;
  }, [myVisits]);

  const byBucket = useMemo(() => {
    const grouped = new Map<VisitBucket, Visit[]>();
    for (const visit of myVisits) {
      const bucket = visitBucket(visit, today);
      grouped.set(bucket, [...(grouped.get(bucket) ?? []), visit]);
    }
    return grouped;
  }, [myVisits, today]);

  const selectedDate = parseDate(selected) ?? today;
  const selectedVisits = byDay.get(selected) ?? [];
  const poolOnSelectedDay = poolVisits.filter((v) => v.date === selected).length;
  const nextDateWithVisits = [...byDay.keys()].filter((date) => date > selected).sort()[0];
  const nextVisitDate = parseDate(nextDateWithVisits ?? null);
  const needsOutcomeCount = byBucket.get('needs-outcome')?.length ?? 0;

  const selectDate = (date: Date) => {
    const key = dayKey(date);
    setSelected(key);
    setExpanded(false);
    updateParams({ date: key });
  };

  const move = (direction: -1 | 1) => {
    const next = expanded
      ? toGregorian(addMonthsJalali(toJalali(selectedDate), direction))
      : addDays(selectedDate, 7 * direction);
    const key = dayKey(next);
    setSelected(key);
    updateParams({ date: key });
  };

  const familyOf = (visit: Visit) => cases.find((family) => family.id === visit.caseId);

  const renderRows = (rows: Visit[], showDate = true) => (
    <Card padded={false}>
      {rows.map((visit, index) => (
        <VisitRow
          key={visit.id}
          visit={visit}
          family={familyOf(visit)}
          reports={reports}
          showDate={showDate}
          divider={index < rows.length - 1}
        />
      ))}
    </Card>
  );

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="بازدیدهای من"
        subtitle={view === 'day' ? 'برنامه روزانه شما' : 'همه بازدیدهای محول‌شده به شما'}
        actions={
          <>
            <IconButton label="اعلان‌ها" badge={unreadCount} onClick={() => navigate('/notifications')}>
              <Bell className="size-5" aria-hidden />
            </IconButton>
            {user && (
              <Link to="/profile" aria-label="پروفایل من" className="press ms-0.5 grid size-11 shrink-0 place-items-center rounded-md">
                <Avatar name={user.name} size="sm" tone="brand" />
              </Link>
            )}
          </>
        }
      >
        <SegmentedControl<ViewMode>
          className="mt-3"
          ariaLabel="نمای بازدیدها"
          value={view}
          onChange={(value) => updateParams({ view: value === 'list' ? 'list' : null })}
          items={[
            { value: 'day', label: 'برنامه روزانه' },
            { value: 'list', label: 'همه بازدیدها' },
          ]}
        />
      </ScreenHeader>

      {view === 'day' ? (
        <div className="space-y-5 pb-8 pt-4">
          <VisitCalendar
            selected={selectedDate}
            today={today}
            expanded={expanded}
            visitsByDay={byDay}
            onSelectDate={selectDate}
            onMove={move}
            onToggle={() => setExpanded((value) => !value)}
          />

          <section aria-labelledby="day-visits-heading">
            <div className="mb-3 flex min-h-9 items-center justify-between gap-3">
              <div className="min-w-0">
                <h2 id="day-visits-heading" aria-live="polite" className="truncate text-sm font-bold text-ink">
                  {selected === dayKey(today) ? 'بازدیدهای امروز' : `بازدیدهای ${faDate(selectedDate, { weekday: true, year: false })}`}
                </h2>
                <p className="mt-0.5 text-xs text-ink-3">{fa(selectedVisits.length)} بازدید</p>
              </div>
              {selected !== dayKey(today) && (
                <button
                  type="button"
                  onClick={() => selectDate(today)}
                  className="press min-h-9 shrink-0 rounded-md px-2.5 text-xs font-bold text-brand-ink hover:bg-brand-soft"
                >
                  بازگشت به امروز
                </button>
              )}
            </div>

            {selectedVisits.length > 0 ? (
              renderRows(selectedVisits, false)
            ) : (
              <div className="rounded-lg border border-line bg-surface px-5 py-9 text-center">
                <CalendarCheck2 className="mx-auto mb-2 size-6 text-ink-3" aria-hidden />
                <p className="text-sm font-semibold text-ink-2">برای این روز بازدیدی ندارید</p>
                <div className="mt-3 flex flex-wrap items-center justify-center gap-x-4 gap-y-1">
                  {poolOnSelectedDay > 0 && (
                    <Link
                      to="/available"
                      className="press inline-flex min-h-9 items-center gap-1.5 rounded-md px-2 text-xs font-bold text-brand-ink hover:bg-brand-soft"
                    >
                      <Inbox className="size-4" aria-hidden />
                      {fa(poolOnSelectedDay)} بازدید آزاد در این روز
                    </Link>
                  )}
                  {nextVisitDate && (
                    <button
                      type="button"
                      onClick={() => selectDate(nextVisitDate)}
                      className="press inline-flex min-h-9 items-center gap-1 rounded-md px-2 text-xs font-bold text-brand-ink hover:bg-brand-soft"
                    >
                      بازدید بعدی · {faDate(nextVisitDate, { year: false })}
                      <ChevronLeft className="size-4" aria-hidden />
                    </button>
                  )}
                </div>
              </div>
            )}
          </section>
        </div>
      ) : (
        <div className="pb-8 pt-4">
          {needsOutcomeCount > 0 && filter !== 'needs-outcome' && (
            <button
              type="button"
              onClick={() => updateParams({ status: 'needs-outcome' })}
              className="press mb-4 flex w-full items-center gap-2.5 rounded-lg bg-warning-soft px-4 py-3 text-start"
            >
              <CircleAlert className="size-5 shrink-0 text-warning" aria-hidden />
              <span className="min-w-0 flex-1 text-sm leading-6 text-ink-2">
                <strong className="font-bold text-ink">{fa(needsOutcomeCount)} بازدید گذشته</strong> هنوز نتیجه‌اش ثبت نشده است.
              </span>
              <span className="shrink-0 text-xs font-bold text-brand-ink">مشاهده</span>
            </button>
          )}

          <ChipRail>
            {FILTERS.map((value) => (
              <Chip
                key={value}
                selected={filter === value}
                count={value === 'all' ? myVisits.length : (byBucket.get(value)?.length ?? 0)}
                onClick={() => updateParams({ status: value === 'upcoming' ? null : value })}
              >
                {filterLabel(value)}
              </Chip>
            ))}
          </ChipRail>

          <div className="mt-4 space-y-6">
            {filter === 'all' ? (
              GROUP_ORDER.map((bucket) => {
                const rows = byBucket.get(bucket) ?? [];
                if (rows.length === 0) return null;
                return (
                  <section key={bucket} aria-label={VISIT_BUCKET_LABEL[bucket]}>
                    <div className="mb-2.5 flex items-baseline justify-between px-1">
                      <h2 className="text-sm font-bold text-ink">{VISIT_BUCKET_LABEL[bucket]}</h2>
                      <span className="text-xs text-ink-3">{fa(rows.length)} بازدید</span>
                    </div>
                    {renderRows(sortForBucket(bucket, rows))}
                  </section>
                );
              })
            ) : (byBucket.get(filter)?.length ?? 0) > 0 ? (
              renderRows(sortForBucket(filter, byBucket.get(filter) ?? []))
            ) : (
              <EmptyState
                icon={<CalendarCheck2 className="size-6" />}
                title={`بازدیدی در وضعیت «${filterLabel(filter)}» ندارید`}
                description={
                  filter === 'upcoming'
                    ? 'بازدید تازه از مسیر «بازدیدهای آزاد» به برنامه شما اضافه می‌شود.'
                    : undefined
                }
              />
            )}
          </div>
        </div>
      )}
    </Screen>
  );
}
