import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useSearchParams } from 'react-router-dom';
import { Bell, CalendarCheck2, ChevronLeft } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Card } from '@/components/ui/Card';
import { IconButton } from '@/components/ui/IconButton';
import { Avatar } from '@/components/ui/Avatar';
import { VisitRow } from '@/features/_shared/VisitRow';
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

/** A single date drives the calendar, the agenda, and its visit details. */
export function VisitsScreen() {
  const { visits, cases, reports, user, unreadCount } = useApp();
  const navigate = useNavigate();
  const [params, setParams] = useSearchParams();
  const focusedDate = params.get('date');
  const today = useMemo(() => new Date(), []);
  const [selected, setSelected] = useState(() => dayKey(parseDate(focusedDate) ?? today));
  const [expanded, setExpanded] = useState(false);

  // Also handle dates supplied by other screens (e.g. /visits?date=...).
  useEffect(() => {
    const linked = parseDate(focusedDate);
    if (linked) setSelected(dayKey(linked));
  }, [focusedDate]);

  const byDay = useMemo(() => {
    const dates = new Map<string, typeof visits>();
    for (const visit of visits) {
      const dayVisits = dates.get(visit.date) ?? [];
      dayVisits.push(visit);
      dates.set(visit.date, dayVisits);
    }
    for (const dayVisits of dates.values()) dayVisits.sort(visitChronological);
    return dates;
  }, [visits]);

  const selectedDate = parseDate(selected) ?? today;
  const selectedVisits = byDay.get(selected) ?? [];
  const nextDateWithVisits = [...byDay.keys()].filter((date) => date > selected).sort()[0];
  const nextVisitDate = parseDate(nextDateWithVisits ?? null);

  const selectDate = (date: Date) => {
    const key = dayKey(date);
    setSelected(key);
    setExpanded(false);
    setParams({ date: key }, { replace: true });
  };

  const move = (direction: -1 | 1) => {
    const next = expanded
      ? toGregorian(addMonthsJalali(toJalali(selectedDate), direction))
      : addDays(selectedDate, 7 * direction);
    const key = dayKey(next);
    setSelected(key);
    setParams({ date: key }, { replace: true });
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="بازدیدها"
        subtitle="برنامه بازدیدها به تفکیک روز"
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
      />

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
            <Card padded={false}>
              {selectedVisits.map((visit, index) => (
                <VisitRow
                  key={visit.id}
                  visit={visit}
                  family={cases.find((family) => family.id === visit.caseId)}
                  reports={reports}
                  showDate={false}
                  divider={index < selectedVisits.length - 1}
                />
              ))}
            </Card>
          ) : (
            <div className="rounded-lg border border-line bg-surface px-5 py-9 text-center">
              <CalendarCheck2 className="mx-auto mb-2 size-6 text-ink-3" aria-hidden />
              <p className="text-sm font-semibold text-ink-2">برای این روز بازدیدی ثبت نشده</p>
              {nextVisitDate && (
                <button
                  type="button"
                  onClick={() => selectDate(nextVisitDate)}
                  className="press mt-3 inline-flex min-h-9 items-center gap-1 rounded-md px-2 text-xs font-bold text-brand-ink hover:bg-brand-soft"
                >
                  بازدید بعدی · {faDate(nextVisitDate, { year: false })}
                  <ChevronLeft className="size-4" aria-hidden />
                </button>
              )}
            </div>
          )}
        </section>
      </div>
    </Screen>
  );
}
