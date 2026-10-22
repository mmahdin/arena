import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { CalendarCheck2, ChevronLeft, ChevronRight, Clock3, MoreVertical } from 'lucide-react';
import type { CalendarEvent, Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Card } from '@/components/ui/Card';
import { Badge } from '@/components/ui/Badge';
import { EmptyState } from '@/components/ui/EmptyState';
import { VisitRow } from '@/features/_shared/VisitRow';
import { TaskMenu } from '@/features/_shared/TaskRow';
import { TONE } from '@/components/ui/tone';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { buildEvents } from '@/services/calendar';
import {
  addMonthsJalali, buildMonthGrid, dayKey, faDate, faMonthTitle,
  isSameDay, persianWeekdayIndex, toGregorian, toJalali, WEEKDAYS_SHORT, faTime, faDuration,
} from '@/lib/jalali';
import { visitChronological, visitStatusMeta } from '@/lib/visit-workflow';

function linkedDate(value: string | null): Date | undefined {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return undefined;
  const date = new Date(`${value}T12:00:00`);
  return !Number.isNaN(date.getTime()) && dayKey(date) === value ? date : undefined;
}

/** Calendar is an alternate view of visits, never a second task dashboard. */
export function CalendarScreen() {
  const { myVisits: visits, cases, reports, tasks } = useApp();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const focusedDate = params.get('date');
  const today = useMemo(() => new Date(), []);
  const [cursor, setCursor] = useState(() => toJalali(linkedDate(focusedDate) ?? today));
  const [selected, setSelected] = useState(() => dayKey(linkedDate(focusedDate) ?? today));
  const grid = useMemo(() => buildMonthGrid(cursor.jy, cursor.jm, today), [cursor, today]);

  useEffect(() => {
    const date = linkedDate(focusedDate);
    if (!date) return;
    setCursor(toJalali(date));
    setSelected(dayKey(date));
  }, [focusedDate]);

  const byDay = useMemo(() => {
    const map = new Map<string, Visit[]>();
    for (const visit of visits) map.set(visit.date, [...(map.get(visit.date) ?? []), visit]);
    return map;
  }, [visits]);
  const selectedVisits = (byDay.get(selected) ?? []).slice().sort(visitChronological);
  const otherEvents = useMemo(() => buildEvents().filter((e) => e.kind !== 'visit'), [tasks, visits]);
  const selectedOther = otherEvents.filter((e) => e.date === selected);

  const selectDate = (date: Date) => {
    setSelected(dayKey(date));
    setCursor(toJalali(date));
    setParams({ date: dayKey(date) }, { replace: true });
  };
  const changeMonth = (offset: number) => {
    const next = addMonthsJalali(cursor, offset);
    selectDate(toGregorian({ ...next, jd: 1 }));
  };
  const returnToday = () => selectDate(today);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="تقویم بازدیدها" back onBack={() => navigate('/visits')}
        subtitle="انتخاب روز برای مشاهده بازدیدها"
        actions={<IconButton label="رفتن به امروز" onClick={returnToday}>
          <CalendarCheck2 className="size-5" aria-hidden />
        </IconButton>}
      />
      <div className="pb-8 pt-4">
        <div className="mb-2 flex items-center justify-between gap-2 rounded-md border border-line bg-surface p-1.5">
          <IconButton label="ماه قبل" size="sm" onClick={() => changeMonth(-1)}><ChevronRight className="size-[18px]" /></IconButton>
          <p className="flex-1 text-center text-md font-bold text-ink">{faMonthTitle(cursor.jy, cursor.jm)}</p>
          <IconButton label="ماه بعد" size="sm" onClick={() => changeMonth(1)}><ChevronLeft className="size-[18px]" /></IconButton>
        </div>
        <div className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
          <div className="grid grid-cols-7 border-b border-line-soft bg-sunken/60">
            {WEEKDAYS_SHORT.map((name, i) => (
              <span key={name} className={cn('py-2 text-center text-xs font-bold', i === 6 ? 'text-danger' : 'text-ink-2')}>{name}</span>
            ))}
          </div>
          <div className="grid grid-cols-7">
            {grid.map((cell) => {
              const key = dayKey(cell.date);
              const dayVisits = byDay.get(key) ?? [];
              const active = key === selected;
              const friday = persianWeekdayIndex(cell.date) === 6;
              return <button
                key={key} type="button" onClick={() => selectDate(cell.date)} aria-pressed={active}
                aria-label={`${faDate(cell.date, { weekday: true })} — ${fa(dayVisits.length)} بازدید`}
                className={cn('flex h-12 min-w-0 flex-col items-center justify-center gap-1 transition-colors',
                  !cell.inMonth && 'opacity-40', active ? 'bg-brand-soft' : 'hover:bg-sunken')}
              >
                <span className={cn('grid size-7 place-items-center rounded-full text-sm tabular-nums',
                  cell.isToday && !active && 'bg-ink text-bg font-extrabold',
                  active && 'bg-brand text-white font-extrabold',
                  !cell.isToday && !active && (friday ? 'text-danger' : 'text-ink'))}>
                  {fa(cell.jalali.jd)}
                </span>
                <span className="flex h-1.5 items-center gap-0.5">
                  {dayVisits.slice(0, 3).map((v) => <span key={v.id} className={cn('size-1.5 rounded-full', TONE[visitStatusMeta(v).tone].dot)} />)}
                </span>
              </button>;
            })}
          </div>
        </div>
        <section className="mt-6">
          <div className="mb-3 flex items-center justify-between">
            <h2 className="text-md font-bold text-ink">{isSameDay(new Date(`${selected}T00:00:00`), today) ? 'امروز' : faDate(selected, { weekday: true })}</h2>
            <span className="text-xs text-ink-3">{fa(selectedVisits.length)} بازدید</span>
          </div>
          {selectedVisits.length ? <Card padded={false}>
            {selectedVisits.map((v, i) => <VisitRow key={v.id} visit={v}
              family={cases.find((c) => c.id === v.caseId)} reports={reports}
              showDate={false} divider={i < selectedVisits.length - 1} />)}
          </Card> : <EmptyState icon={<CalendarCheck2 className="size-6" />} title="بازدیدی برای این روز ثبت نشده" />}
        </section>
        {selectedOther.length > 0 && <details className="mt-5 overflow-hidden rounded-lg border border-line bg-surface">
          <summary className="press flex min-h-12 cursor-pointer list-none items-center justify-between gap-2 px-4 py-3 text-sm font-semibold text-ink-2 [&::-webkit-details-marker]:hidden">
            <span>سایر کارهای این روز</span><span className="text-xs text-ink-3">{fa(selectedOther.length)} مورد</span>
          </summary>
          <div className="border-t border-line-soft">
            {selectedOther.map((event, index) => <OtherWorkRow key={event.id} event={event} divider={index < selectedOther.length - 1} />)}
          </div>
        </details>}
      </div>
    </Screen>
  );
}

/** Secondary duties are available on demand, but never displace visits. */
function OtherWorkRow({ event, divider }: { event: CalendarEvent; divider?: boolean }) {
  const { tasks } = useApp();
  const [menuOpen, setMenuOpen] = useState(false);
  const task = event.taskId ? tasks.find((t) => t.id === event.taskId) : undefined;
  const target = event.href ?? `/calendar?date=${event.date}`;
  const openInPlace = Boolean(task && target.startsWith('/calendar?date='));
  const Icon = event.kind === 'meeting' ? Clock3 : CalendarCheck2;
  const content = <>
    <span className="min-w-0 flex-1">
      <span className="block text-sm font-semibold text-ink">{event.title}</span>
      <span className="mt-1 flex items-center gap-1.5 text-xs text-ink-3">
        <Icon className="size-3.5" aria-hidden /> {event.kind === 'meeting' ? 'جلسه' : 'وظیفه'}
        {event.time && ` · ${faTime(event.time)}`}
        {event.durationMin && ` · ${faDuration(event.durationMin)}`}
      </span>
    </span>
    {event.taskStatus === 'done' && <Badge tone="success">انجام‌شده</Badge>}
  </>;
  const className = cn('press flex min-h-14 flex-1 items-center gap-3 px-4 py-3 text-start hover:bg-sunken', divider && 'border-b border-line-soft');
  return <>
    <div className="flex items-center">
      {openInPlace ? <button type="button" className={className} onClick={() => setMenuOpen(true)}>{content}</button>
        : <Link to={target} className={className}>{content}</Link>}
      {task && !openInPlace && <IconButton label="گزینه‌های کار" size="sm" onClick={() => setMenuOpen(true)}>
        <MoreVertical className="size-4" />
      </IconButton>}
    </div>
    {task && <TaskMenu task={task} open={menuOpen} onClose={() => setMenuOpen(false)} />}
  </>;
}
