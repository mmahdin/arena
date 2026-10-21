import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { CalendarX2, ChevronLeft, ChevronRight, Clock } from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import {
  addDays,
  fa,
  formatDate,
  formatDayMonth,
  formatLongDate,
  parse,
  startOfDay,
  toISO,
  WEEKDAYS_SHORT,
} from "@/lib/date";
import { buildPersianMonth } from "@/lib/calendar";
import { visitsOnDay } from "@/lib/selectors";
import type { Visit } from "@/types";
import { cn } from "@/lib/cn";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { VisitRow } from "@/components/domain/VisitRow";
import "./CalendarPage.css";

export function CalendarPage() {
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const [anchor, setAnchor] = useState<Date>(startOfDay(now));
  const [selected, setSelected] = useState<Date>(startOfDay(now));

  const month = useMemo(() => buildPersianMonth(anchor, now), [anchor, now]);

  const visitsByDay = useMemo(() => {
    const map = new Map<string, Visit[]>();
    data.visits.forEach((visit) => {
      const key = toISO(startOfDay(parse(visit.startsAt))).slice(0, 10);
      if (!map.has(key)) map.set(key, []);
      map.get(key)!.push(visit);
    });
    map.forEach((list) =>
      list.sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime())
    );
    return map;
  }, [data.visits]);

  const selectedVisits = visitsOnDay(data, selected);
  const monthVisitCount = month.cells
    .filter((cell) => cell.inMonth)
    .reduce((total, cell) => {
      const list = visitsByDay.get(cell.key) ?? [];
      return total + list.filter((visit) => visit.status !== "canceled").length;
    }, 0);

  const goToMonth = (direction: -1 | 1) => {
    const nextAnchor = direction === 1 ? addDays(month.last, 1) : addDays(month.first, -1);
    setAnchor(nextAnchor);
    setSelected(nextAnchor);
  };

  const goToday = () => {
    setAnchor(startOfDay(now));
    setSelected(startOfDay(now));
  };

  return (
    <>
      <TopBar
        title="تقویم کاری"
        actions={
          <button type="button" className="calendar-today-btn" onClick={goToday}>
            امروز
          </button>
        }
      />

      <Page>
        <section className="calendar-card" aria-label={`تقویم ${month.label}`}>
          <header className="calendar-head">
            <button
              type="button"
              className="calendar-nav"
              aria-label="ماه قبل"
              onClick={() => goToMonth(-1)}
            >
              <ChevronRight size={18} />
            </button>
            <div className="calendar-head__title">
              <span className="calendar-head__month">{month.label}</span>
              <span className="calendar-head__meta num">
                {fa(monthVisitCount)} بازدید در این ماه
              </span>
            </div>
            <button
              type="button"
              className="calendar-nav"
              aria-label="ماه بعد"
              onClick={() => goToMonth(1)}
            >
              <ChevronLeft size={18} />
            </button>
          </header>

          <div className="calendar-weekdays" aria-hidden="true">
            {WEEKDAYS_SHORT.map((day) => (
              <span key={day} className="calendar-weekday">
                {day}
              </span>
            ))}
          </div>

          <div className="calendar-grid" role="grid" aria-label={`روزهای ${month.label}`}>
            {month.cells.map((cell) => {
              const list = visitsByDay.get(cell.key) ?? [];
              const active = list.filter((visit) => visit.status !== "canceled");
              const selectedDay = cell.key === toISO(selected).slice(0, 10);
              return (
                <button
                  key={cell.key}
                  type="button"
                  role="gridcell"
                  aria-selected={selectedDay}
                  aria-label={`${formatDate(cell.date)}${active.length ? `، ${fa(active.length)} بازدید` : ""}`}
                  className={cn(
                    "calendar-day",
                    !cell.inMonth && "is-outside",
                    cell.isToday && "is-today",
                    selectedDay && "is-selected"
                  )}
                  onClick={() => {
                    setSelected(cell.date);
                    if (!cell.inMonth) setAnchor(cell.date);
                  }}
                >
                  <span className="calendar-day__num">{fa(cell.day)}</span>
                  <span className="calendar-day__dots" aria-hidden="true">
                    {active.slice(0, 3).map((visit, index) => (
                      <span
                        key={`${visit.id}-${index}`}
                        className={cn(
                          "calendar-day__dot",
                          visit.status === "completed" && "is-done"
                        )}
                      />
                    ))}
                  </span>
                </button>
              );
            })}
          </div>

          <div className="calendar-legend">
            <span className="calendar-legend__item">
              <span className="calendar-day__dot" aria-hidden="true" /> بازدید برنامه‌ریزی‌شده
            </span>
            <span className="calendar-legend__item">
              <span className="calendar-day__dot is-done" aria-hidden="true" /> انجام‌شده
            </span>
          </div>
        </section>

        <section className="calendar-agenda">
          <header className="calendar-agenda__head">
            <div>
              <h2 className="calendar-agenda__title">{formatDayMonth(selected)}</h2>
              <p className="calendar-agenda__sub">{formatLongDate(selected)}</p>
            </div>
            <span className="calendar-agenda__count num">
              {fa(selectedVisits.length)} بازدید
            </span>
          </header>

          {selectedVisits.length > 0 ? (
            <ul className="calendar-agenda__list">
              {selectedVisits.map((visit) => (
                <li key={visit.id}>
                  <VisitRow visit={visit} compact />
                </li>
              ))}
            </ul>
          ) : (
            <div className="calendar-empty">
              <span className="calendar-empty__icon" aria-hidden="true">
                <CalendarX2 size={18} />
              </span>
              <div>
                <p className="calendar-empty__title">برنامه‌ای برای این روز نیست</p>
                <p className="calendar-empty__text">
                  می‌توانید روز دیگری را انتخاب کنید یا <Clock size={11} aria-hidden="true" />{" "}
                  بازدیدهای پیش رو را در بخش بازدیدها ببینید.
                </p>
              </div>
            </div>
          )}
        </section>

        <button
          type="button"
          className="calendar-all-link"
          onClick={() => navigate("/visits")}
        >
          مشاهده فهرست کامل بازدیدها
          <ChevronLeft size={15} aria-hidden="true" />
        </button>
      </Page>
    </>
  );
}
