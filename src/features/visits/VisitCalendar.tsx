import { useMemo } from 'react';
import { ChevronDown, ChevronLeft, ChevronRight } from 'lucide-react';
import type { Visit } from '@/types';
import { Card } from '@/components/ui/Card';
import { IconButton } from '@/components/ui/IconButton';
import { TONE } from '@/components/ui/tone';
import { cn } from '@/lib/cn';
import { visitStatusMeta } from '@/lib/visit-workflow';
import { fa } from '@/lib/digits';
import {
  buildMonthGrid, dayKey, faDate, faMonthTitle, isSameDay,
  jalaliMonthLength, persianWeekdayIndex, toGregorian, toJalali,
  WEEKDAYS_LONG, WEEKDAYS_SHORT,
} from '@/lib/jalali';

interface VisitCalendarProps {
  selected: Date;
  today: Date;
  expanded: boolean;
  visitsByDay: Map<string, Visit[]>;
  onSelectDate: (date: Date) => void;
  onMove: (direction: -1 | 1) => void;
  onToggle: () => void;
}

/**
 * The calendar is the primary visit navigator, not a separate destination.
 * Start in the compact week view with today selected. The month title expands
 * to the full month; choosing a date returns to the compact week view.
 */
export function VisitCalendar({
  selected, today, expanded, visitsByDay,
  onSelectDate, onMove, onToggle,
}: VisitCalendarProps) {
  const { jy, jm } = toJalali(selected);
  const cells = useMemo(() => buildMonthGrid(jy, jm, today), [jy, jm, today]);
  const firstWeekday = persianWeekdayIndex(toGregorian({ jy, jm, jd: 1 }));
  const monthCells = cells.slice(0, Math.ceil((firstWeekday + jalaliMonthLength(jy, jm)) / 7) * 7);
  const selectedIndex = cells.findIndex((cell) => isSameDay(cell.date, selected));
  const weekStart = Math.floor(selectedIndex / 7) * 7;
  const visibleCells = expanded ? monthCells : cells.slice(weekStart, weekStart + 7);

  return (
    <Card padded={false} className="overflow-hidden">
      <div className="flex min-h-12 items-center justify-between gap-1 px-2.5">
        <IconButton label={expanded ? 'ماه قبل' : 'هفته قبل'} onClick={() => onMove(-1)}>
          <ChevronRight className="size-[18px]" aria-hidden />
        </IconButton>
        <button
          type="button"
          onClick={onToggle}
          aria-expanded={expanded}
          aria-controls="visit-calendar-days"
          aria-label={expanded ? 'نمایش فقط هفته انتخاب‌شده' : 'نمایش تقویم کامل ماه'}
          className="press inline-flex min-h-10 min-w-0 items-center justify-center gap-1 rounded-md px-1.5 hover:bg-sunken"
        >
          <span className="truncate text-sm font-bold text-ink">{faMonthTitle(jy, jm)}</span>
          <span className="whitespace-nowrap text-2xs font-semibold text-brand-ink">{expanded ? 'نمایش هفته' : 'نمایش ماه'}</span>
          <ChevronDown className={cn('size-4 shrink-0 text-brand-ink transition-transform duration-base', expanded && 'rotate-180')} aria-hidden />
        </button>
        <IconButton label={expanded ? 'ماه بعد' : 'هفته بعد'} onClick={() => onMove(1)}>
          <ChevronLeft className="size-[18px]" aria-hidden />
        </IconButton>
      </div>

      <div className="border-t border-line-soft px-2 pb-2.5 pt-2">
        <div className="grid grid-cols-7" aria-hidden>
          {WEEKDAYS_SHORT.map((label, index) => (
            <span key={label} className={cn('py-1 text-center text-2xs font-semibold', index === 6 ? 'text-danger' : 'text-ink-3')} title={WEEKDAYS_LONG[index]}>
              {label}
            </span>
          ))}
        </div>
        <div id="visit-calendar-days" role="group" aria-label="انتخاب روز بازدید" className="grid grid-cols-7 gap-y-0.5">
          {visibleCells.map((cell) => {
            const date = dayKey(cell.date);
            const visits = visitsByDay.get(date) ?? [];
            const active = isSameDay(cell.date, selected);
            const friday = persianWeekdayIndex(cell.date) === 6;

            return (
              <button
                key={date}
                type="button"
                onClick={() => onSelectDate(cell.date)}
                aria-pressed={active}
                aria-current={cell.isToday ? 'date' : undefined}
                aria-label={`${faDate(cell.date, { weekday: true })}، ${fa(visits.length)} بازدید`}
                className={cn(
                  'press flex min-h-11 min-w-0 flex-col items-center justify-center gap-0.5 rounded-md py-1 transition-colors',
                  active ? 'bg-brand-soft' : 'hover:bg-sunken',
                  expanded && !cell.inMonth && 'opacity-40',
                )}
              >
                <span className={cn(
                  'grid size-[30px] place-items-center rounded-full text-sm tabular-nums',
                  active ? 'bg-brand font-extrabold text-white' : cell.isToday ? 'ring-1 ring-brand font-bold text-brand-ink' : friday ? 'text-danger' : 'text-ink',
                )}>
                  {fa(cell.jalali.jd)}
                </span>
                {/* One dot per visit that day (max 3), coloured by the visit's
                    own status — a warning dot reads as «نتیجه‌اش ثبت نشده». */}
                <span className="flex h-1.5 items-center gap-[3px]" aria-hidden>
                  {visits.slice(0, 3).map((visit) => (
                    <span key={visit.id} className={cn('size-1.5 rounded-full', TONE[visitStatusMeta(visit).tone].dot)} />
                  ))}
                </span>
              </button>
            );
          })}
        </div>
      </div>
    </Card>
  );
}
