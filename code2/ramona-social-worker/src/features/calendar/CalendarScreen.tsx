import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { CalendarCheck2, ChevronLeft, ChevronRight, Clock3, MapPin } from 'lucide-react';
import type { CalendarEvent } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { EmptyState } from '@/components/ui/EmptyState';
import { TONE } from '@/components/ui/tone';
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
  toJalali,
  WEEKDAYS_SHORT,
} from '@/lib/jalali';
import { buildEvents } from '@/services/calendar';

/**
 * Calendar.
 *
 * A Jalali month grid with per-day load indicators, and the selected day's
 * agenda underneath. The grid answers "how busy is that week?", the list
 * answers "what exactly do I do then?" — both without leaving the screen.
 */
export function CalendarScreen() {
  const { visits, tasks } = useApp();
  const today = useMemo(() => new Date(), []);
  const [cursor, setCursor] = useState(() => toJalali(today));
  const [selected, setSelected] = useState(() => dayKey(today));

  // Rebuilt from the store so recording a visit outcome updates the grid.
  const events = useMemo(() => buildEvents(), [visits, tasks]);

  const grid = useMemo(() => buildMonthGrid(cursor.jy, cursor.jm, today), [cursor, today]);

  const byDay = useMemo(() => {
    const map = new Map<string, CalendarEvent[]>();
    for (const e of events) {
      const list = map.get(e.date) ?? [];
      list.push(e);
      map.set(e.date, list);
    }
    return map;
  }, [events]);

  const selectedEvents = useMemo(
    () =>
      (byDay.get(selected) ?? []).slice().sort((a, b) => (a.time ?? '99:99').localeCompare(b.time ?? '99:99')),
    [byDay, selected],
  );

  const monthCount = useMemo(() => {
    const { jy, jm } = cursor;
    return events.filter((e) => {
      const j = toJalali(new Date(`${e.date}T00:00:00`));
      return j.jy === jy && j.jm === jm;
    }).length;
  }, [events, cursor]);

  const isCurrentMonth = cursor.jy === toJalali(today).jy && cursor.jm === toJalali(today).jm;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="تقویم کاری"
        subtitle={`${fa(monthCount)} برنامه در ${faMonthTitle(cursor.jy, cursor.jm)}`}
        actions={
          <IconButton label="بازگشت به امروز" onClick={() => { setCursor(toJalali(today)); setSelected(dayKey(today)); }}>
            <CalendarCheck2 className="size-[20px]" strokeWidth={2} />
          </IconButton>
        }
      >
        {/* month navigation */}
        <div className="mt-3 flex items-center justify-between gap-2 rounded-md border border-line bg-surface p-1.5">
          <IconButton label="ماه قبل" size="sm" onClick={() => setCursor((c) => addMonthsJalali(c, -1))}>
            <ChevronRight className="size-[18px]" strokeWidth={2.4} />
          </IconButton>
          <p className="flex-1 text-center text-md font-bold text-ink">{faMonthTitle(cursor.jy, cursor.jm)}</p>
          <IconButton label="ماه بعد" size="sm" onClick={() => setCursor((c) => addMonthsJalali(c, 1))}>
            <ChevronLeft className="size-[18px]" strokeWidth={2.4} />
          </IconButton>
        </div>
      </ScreenHeader>

      <div className="pt-3">
        <div className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
          {/* weekday header */}
          <div className="grid grid-cols-7 border-b border-line-soft bg-sunken/60">
            {WEEKDAYS_SHORT.map((d, i) => (
              <span
                key={d}
                className={cn(
                  'py-2 text-center text-2xs font-bold',
                  i === 6 ? 'text-danger/80' : 'text-ink-3',
                )}
              >
                {d}
              </span>
            ))}
          </div>

          {/* day grid */}
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
                  onClick={() => setSelected(key)}
                  aria-pressed={isSelected}
                  aria-label={`${faDate(cell.date, { weekday: true })} — ${fa(dayEvents.length)} برنامه`}
                  className={cn(
                    'relative flex h-[52px] flex-col items-center justify-center gap-1 transition-colors',
                    !cell.inMonth && 'opacity-35',
                    isSelected ? 'bg-brand-soft' : 'hover:bg-sunken',
                  )}
                >
                  <span
                    className={cn(
                      'grid size-7 place-items-center rounded-full text-sm tabular-nums transition-colors',
                      cell.isToday && !isSelected && 'bg-ink text-bg font-extrabold',
                      cell.isToday && isSelected && 'bg-brand text-white font-extrabold',
                      !cell.isToday && (isSelected ? 'font-bold text-brand-ink' : isFriday ? 'text-danger/85' : 'text-ink'),
                    )}
                  >
                    {fa(cell.jalali.jd)}
                  </span>
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

        {/* legend */}
        <div className="mt-2.5 flex flex-wrap items-center gap-x-3.5 gap-y-1 px-1 text-2xs text-ink-3">
          <LegendDot className="bg-brand" label="بازدید" />
          <LegendDot className="bg-warning" label="مهلت مهم" />
          <LegendDot className="bg-info" label="مهلت" />
          <LegendDot className="bg-success" label="انجام‌شده" />
          <LegendDot className="bg-ink-3" label="جلسه" />
        </div>

        {/* selected day */}
        <section className="mt-6 pb-8">
          <div className="mb-2.5 flex items-baseline justify-between">
            <h2 className="text-md font-bold text-ink">
              {isSameDay(new Date(`${selected}T00:00:00`), today) ? 'امروز' : faDate(selected, { weekday: true })}
            </h2>
            <span className="text-2xs text-ink-3 tabular-nums">
              {!isCurrentMonth && `${faMonthTitle(cursor.jy, cursor.jm)} · `}
              {fa(selectedEvents.length)} برنامه
            </span>
          </div>

          {selectedEvents.length === 0 ? (
            <div className="rounded-lg border border-dashed border-line-strong bg-surface/60">
              <EmptyState
                icon={<CalendarCheck2 className="size-6" />}
                title="برنامه‌ای ثبت نشده"
                description="این روز خالی است؛ می‌توانید از فهرست کارها موارد پیش‌رو را ببینید."
              />
            </div>
          ) : (
            <ul className="space-y-2">
              {selectedEvents.map((e) => (
                <EventCard key={e.id} event={e} />
              ))}
            </ul>
          )}
        </section>
      </div>
    </Screen>
  );
}

function LegendDot({ className, label }: { className: string; label: string }) {
  return (
    <span className="flex items-center gap-1.5">
      <span className={cn('size-1.5 rounded-full', className)} aria-hidden />
      {label}
    </span>
  );
}

function EventCard({ event }: { event: CalendarEvent }) {
  const Icon = event.kind === 'visit' ? MapPin : event.kind === 'meeting' ? Clock3 : CalendarCheck2;
  return (
    <li>
      <Link
        to={event.href ?? '#'}
        className="press flex items-center gap-3 rounded-lg border border-line bg-surface p-3.5 shadow-xs hover:border-line-strong"
      >
        <span className={cn('w-14 shrink-0 text-center text-sm font-bold tabular-nums', TONE[event.tone].text)}>
          {event.time ? faTime(event.time) : '—'}
        </span>
        <span className={cn('h-9 w-1 shrink-0 rounded-pill', TONE[event.tone].dot)} aria-hidden />
        <span className="min-w-0 flex-1">
          <span className="block truncate text-base font-semibold text-ink">{event.title}</span>
          <span className="mt-0.5 flex items-center gap-1.5 overflow-hidden text-xs text-ink-3">
            <Icon className="size-3.5 shrink-0" strokeWidth={2.25} aria-hidden />
            <span className="truncate">
              {event.kind === 'visit' ? 'بازدید' : event.kind === 'meeting' ? 'جلسه' : 'مهلت انجام'}
              {event.caseCode ? ` · ${event.caseCode}` : ''}
              {event.durationMin ? ` · ${faDuration(event.durationMin)}` : ''}
            </span>
          </span>
        </span>
      </Link>
    </li>
  );
}
