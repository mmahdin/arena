import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { CalendarDays, CalendarX2, History } from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import { formatDayLabel, formatDayMonth, formatLongDate, parse, startOfDay, toISO } from "@/lib/date";
import { pastVisits, upcomingVisits } from "@/lib/selectors";
import type { Visit } from "@/types";
import { Page, PageHeader } from "@/components/app/Page";
import { IconButton } from "@/components/ui/Button";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { EmptyState } from "@/components/ui/EmptyState";
import { VisitRow } from "@/components/domain/VisitRow";
import "./VisitsPage.css";

type View = "upcoming" | "past";

function groupByDay(visits: Visit[]) {
  const groups = new Map<string, { date: Date; visits: Visit[] }>();
  visits.forEach((visit) => {
    const date = startOfDay(parse(visit.startsAt));
    const key = toISO(date).slice(0, 10);
    if (!groups.has(key)) groups.set(key, { date, visits: [] });
    groups.get(key)!.visits.push(visit);
  });
  return [...groups.values()];
}

export function VisitsPage() {
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const [view, setView] = useState<View>("upcoming");

  const upcoming = upcomingVisits(data);
  const past = pastVisits(data);
  const list = view === "upcoming" ? upcoming : past;
  const groups = groupByDay(list);

  return (
    <Page>
      <PageHeader
        eyebrow="برنامه کاری شما"
        title="بازدیدها"
        actions={
          <IconButton variant="surface" label="تقویم ماهانه" onClick={() => navigate("/calendar")}>
            <CalendarDays size={19} aria-hidden="true" />
          </IconButton>
        }
      />

      <div className="visits-toolbar">
        <SegmentedControl<View>
          ariaLabel="فیلتر بازدیدها"
          value={view}
          onChange={setView}
          items={[
            { value: "upcoming", label: "پیش رو", count: upcoming.length },
            { value: "past", label: "گذشته", count: past.length },
          ]}
        />
      </div>

      {groups.length === 0 ? (
        <EmptyState
          icon={view === "upcoming" ? <CalendarX2 size={22} /> : <History size={22} />}
          title={view === "upcoming" ? "بازدید پیش رویی ندارید" : "بازدید گذشته‌ای ثبت نشده"}
          description={
            view === "upcoming"
              ? "بازدیدهای جدید از سوی سرپرست یا سامانه به کارتابل شما اضافه می‌شود."
              : "بازدیدهای انجام‌شده و گزارش‌های مرتبط در این بخش نمایش داده می‌شوند."
          }
        />
      ) : (
        <div className="visit-groups">
          {groups.map(({ date, visits }) => {
            const label = formatDayLabel(date, now);
            const isNear = ["امروز", "فردا", "دیروز"].includes(label);
            const isToday = label === "امروز";
            return (
              <section className="day-group" key={toISO(date)} aria-label={formatLongDate(date)}>
                <header className="day-group__head">
                  <div className="day-group__date">
                    <span className={isToday ? "day-group__label is-today" : "day-group__label"}>
                      {label}
                    </span>
                    <span className="day-group__sub">
                      {isNear ? formatDayMonth(date) : formatLongDate(date)}
                    </span>
                  </div>
                  <span className="day-group__count num">
                    {new Intl.NumberFormat("fa-IR").format(visits.length)} بازدید
                  </span>
                </header>
                <ul className="day-group__list">
                  {visits.map((visit) => (
                    <li key={visit.id}>
                      <VisitRow visit={visit} />
                    </li>
                  ))}
                </ul>
              </section>
            );
          })}
        </div>
      )}
    </Page>
  );
}
