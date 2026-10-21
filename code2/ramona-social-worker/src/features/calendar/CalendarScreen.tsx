import { useEffect, useMemo, useState } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { CalendarCheck2, ChevronLeft, ChevronRight, Clock3, MapPin, MoreVertical } from 'lucide-react';
import type { CalendarEvent } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Badge } from '@/components/ui/Badge';
import { ListGroup } from '@/components/ui/ListRow';
import { EmptyState } from '@/components/ui/EmptyState';
import { TONE } from '@/components/ui/tone';
import { TaskMenu } from '@/features/_shared/TaskRow';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import {
  addMonthsJalali,
  buildMonthGrid,
  dayKey,
  faDate,
  faDuration,
  faMonthTitle,
  faTime,
  isSameDay,
  persianWeekdayIndex,
  toGregorian,
  toJalali,
  WEEKDAYS_SHORT,
} from '@/lib/jalali';
import { buildEvents } from '@/services/calendar';

/** Read a linked date without accepting invalid or normalized calendar dates. */
function linkedDate(value: string | null): Date | undefined {
  if (!value || !/^\d{4}-\d{2}-\d{2}$/.test(value)) return undefined;
  const date = new Date(`${value}T12:00:00`);
  return !Number.isNaN(date.getTime()) && dayKey(date) === value ? date : undefined;
}

/** One purpose: select a date and work with that day's schedule and tasks. */
export function CalendarScreen() {
  const { visits, tasks } = useApp();
  const [params, setParams] = useSearchParams();
  const focusedDate = params.get('date');
  const today = useMemo(() => new Date(), []);
  const [cursor, setCursor] = useState(() => toJalali(linkedDate(focusedDate) ?? today));
  const [selected, setSelected] = useState(() => dayKey(linkedDate(focusedDate) ?? today));
  const events = useMemo(() => buildEvents(), [visits, tasks]);
  const grid = useMemo(() => buildMonthGrid(cursor.jy, cursor.jm, today), [cursor, today]);

  // Deep links from alerts, search and notifications open the relevant day.
  useEffect(() => {
    const date = linkedDate(focusedDate);
    if (!date) return;
    setCursor(toJalali(date));
    setSelected(dayKey(date));
  }, [focusedDate]);

  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const e of events) map.set(e.date, [...(map.get(e.date) ?? []), e]);
    return map;
  }, [events]);

  const selectedEvents = useMemo(
    () => (byDay.get(selected) ?? []).slice().sort((a, b) => (a.time ?? '99:99').localeCompare(b.time ?? '99:99')),
    [byDay, selected],
  );

  const changeMonth = (offset: number) => {
    const next = addMonthsJalali(cursor, offset);
    setCursor(next);
    setSelected(dayKey(toGregorian({ ...next, jd: 1 })));
  };

  const selectDate = (key: string, date: Date) => {
    setSelected(key);
    const next = toJalali(date);
    if (next.jy !== cursor.jy || next.jm !== cursor.jm) setCursor(next);
  };

  const returnToday = () => {
    setCursor(toJalali(today));
    setSelected(dayKey(today));
    setParams({});
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="تقویم کاری"
        subtitle="برنامه‌ها و مسئولیت‌های شما"
        actions={(
          <IconButton label="بازگشت به امروز" onClick={returnToday}>
            <CalendarCheck2 className="size-[20px]" strokeWidth={2} />
          </IconButton>
        )}
      />

      <div className="pt-4 pb-8">
        <div className="mb-2 flex items-center justify-between gap-2 rounded-md border border-line bg-surface p-1.5">
          <IconButton label="ماه قبل" size="sm" onClick={() => changeMonth(-1)}>
            <ChevronRight className="size-[18px]" strokeWidth={2.4} />
          </IconButton>
          <p className="flex-1 text-center text-md font-bold text-ink">{faMonthTitle(cursor.jy, cursor.jm)}</p>
          <IconButton label="ماه بعد" size="sm" onClick={() => changeMonth(1)}>
            <ChevronLeft className="size-[18px]" strokeWidth={2.4} />
          </IconButton>
        </div>

        <div className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
          <div className="grid grid-cols-7 border-b border-line-soft bg-sunken/60">
            {WEEKDAYS_SHORT.map((d, i) => (
              <span key={d} className={cn('py-2 text-center text-xs font-bold', i === 6 ? 'text-danger' : 'text-ink-2')}>{d}</span>
            ))}
          </div>
          <div className="grid grid-cols-7">
            {grid.map((cell) => {
              const key = dayKey(cell.date);
              const dayEvents = byDay.get(key) ?? [];
              const isSelected = key === selected;
              const isFriday = persianWeekdayIndex(cell.date) === 6;
              return (
                <button
                  key={key}
                  type="button"
                  onClick={() => selectDate(key, cell.date)}
                  aria-pressed={isSelected}
                  aria-label={`${faDate(cell.date, { weekday: true })} — ${fa(dayEvents.length)} مورد`}
                  className={cn(
                    'relative flex h-12 min-w-0 flex-col items-center justify-center gap-1 transition-colors',
                    !cell.inMonth && 'opacity-40',
                    isSelected ? 'bg-brand-soft' : 'hover:bg-sunken',
                  )}
                >
                  <span className={cn(
                    'grid size-7 place-items-center rounded-full text-sm tabular-nums transition-colors',
                    cell.isToday && !isSelected && 'bg-ink text-bg font-extrabold',
                    isSelected && 'bg-brand text-white font-extrabold',
                    !cell.isToday && !isSelected && (isFriday ? 'text-danger' : 'text-ink'),
                  )}>{fa(cell.jalali.jd)}</span>
                  <span className="flex h-1.5 items-center gap-0.5">
                    {dayEvents.slice(0, 3).map((e) => (
                      <span key={e.id} className={cn('size-1.5 rounded-full', TONE[e.tone].dot)} />
                    ))}
                  </span>
                </button>
              );
            })}
          </div>
        </div>

        <section className="mt-6" aria-label="برنامه روز انتخاب‌شده">
          <div className="mb-3 flex items-center justify-between gap-2">
            <h2 className="text-md font-bold text-ink">
              {isSameDay(new Date(`${selected}T00:00:00`), today) ? 'امروز' : faDate(selected, { weekday: true })}
            </h2>
            <span className="text-xs text-ink-3 tabular-nums">{fa(selectedEvents.length)} مورد</span>
          </div>
          {selectedEvents.length === 0 ? (
            <div className="rounded-lg border border-dashed border-line-strong bg-surface/60">
              <EmptyState
                icon={<CalendarCheck2 className="size-6" />}
                title="موردی برای این روز ثبت نشده"
                description="برای دیدن برنامه‌ها و کارهای یک روز دیگر، تاریخ آن را انتخاب کنید."
              />
            </div>
          ) : (
            <ListGroup>
              {selectedEvents.map((e, i) => (
                <EventRow key={e.id} event={e} divider={i < selectedEvents.length - 1} />
              ))}
            </ListGroup>
          )}
        </section>
      </div>
    </Screen>
  );
}

function EventRow({ event, divider }: { event: CalendarEvent; divider?: boolean }) {
  const { tasks } = useApp();
  const [menuOpen, setMenuOpen] = useState(false);
  const task = event.taskId ? tasks.find((t) => t.id === event.taskId) : undefined;
  const target = event.href ?? `/calendar?date=${event.date}`;
  const openInPlace = Boolean(task && target.startsWith('/calendar?date='));
  const Icon = event.kind === 'visit' ? MapPin : event.kind === 'meeting' ? Clock3 : CalendarCheck2;
  const done = event.taskStatus === 'done' || (event.kind === 'visit' && event.tone === 'success');
  const revision = event.taskStatus === 'revision';
  const blocked = event.taskStatus === 'blocked';
  const rowClass = 'press flex min-h-16 min-w-0 flex-1 items-center gap-3 px-3 py-3.5 text-start hover:bg-sunken';

  const content = (
    <>
      <span className="w-[3.3rem] shrink-0 text-center text-xs font-bold tabular-nums text-ink-2">
        {event.time ? faTime(event.time) : 'مهلت'}
      </span>
      <span className={cn('h-10 w-1 shrink-0 rounded-pill', TONE[event.tone].dot)} aria-hidden />
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-bold leading-relaxed text-ink">{event.title}</span>
        <span className="mt-0.5 flex items-center gap-1.5 text-xs leading-relaxed text-ink-3">
          <Icon className="size-3.5 shrink-0" strokeWidth={2.1} aria-hidden />
          <span>
            {event.kind === 'visit' ? 'بازدید' : event.kind === 'meeting' ? 'جلسه' : 'وظیفه'}
            {event.caseCode ? ` · ${event.caseCode}` : ''}
            {event.durationMin ? ` · ${faDuration(event.durationMin)}` : ''}
          </span>
        </span>
      </span>
      {done ? <Badge tone="success">انجام‌شده</Badge> : revision ? <Badge tone="warning">اصلاح</Badge> : blocked ? <Badge tone="danger">متوقف</Badge> : null}
    </>
  );

  return (
    <>
      <div className={cn('flex min-w-0 items-center', divider && 'border-b border-line-soft')}>
        {openInPlace ? (
          <button type="button" onClick={() => setMenuOpen(true)} className={rowClass}>
            {content}
            <MoreVertical className="size-4 shrink-0 text-ink-3" aria-hidden />
          </button>
        ) : (
          <Link to={target} className={rowClass}>{content}</Link>
        )}
        {task && !openInPlace && (
          <div className="pe-2">
            <IconButton label="گزینه‌های کار" size="sm" onClick={() => setMenuOpen(true)}>
              <MoreVertical className="size-4" />
            </IconButton>
          </div>
        )}
      </div>
      {task && <TaskMenu task={task} open={menuOpen} onClose={() => setMenuOpen(false)} />}
    </>
  );
}
