import { useNavigate } from "react-router-dom";
import { Bell, CalendarDays, CalendarX2, CheckCircle2 } from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import { formatLongDate } from "@/lib/date";
import {
  attentionItems,
  monitoringChildren,
  openReportsCount,
  openTasks,
  overdueTasks,
  reportsByStatus,
  tasksDueToday,
  todayVisits,
  unreadNotifications,
} from "@/lib/selectors";
import type { Task } from "@/types";
import { Page, Section } from "@/components/app/Page";
import { SectionHeader } from "@/components/ui/SectionHeader";
import { IconButton } from "@/components/ui/Button";
import { Avatar } from "@/components/ui/Avatar";
import { EmptyState } from "@/components/ui/EmptyState";
import { TaskRow } from "@/components/domain/TaskRow";
import { VisitTimeline } from "@/components/domain/VisitRow";
import { MonitoringCard } from "@/components/domain/MonitoringCard";
import { useToast } from "@/components/ui/Toast";
import { AttentionCard } from "./AttentionCard";
import "./HomePage.css";

export function HomePage() {
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const toast = useToast();
  const actions = useAppActions();
  const firstName = data.profile.fullName.split(" ")[0];

  const visits = todayVisits(data);
  const attention = attentionItems(data).slice(0, 3);
  const revisionReportIds = reportsByStatus(data, "needs_revision").map((r) => r.id);
  const homeTasks = [...overdueTasks(data), ...tasksDueToday(data)]
    .filter((task, index, all) => all.findIndex((t) => t.id === task.id) === index)
    .filter((task) => !task.relatedReportId || !revisionReportIds.includes(task.relatedReportId))
    .slice(0, 3);
  const monitoring = monitoringChildren(data).slice(0, 2);
  const unread = unreadNotifications(data);
  const nextVisit = visits.find((v) => v.status === "scheduled");

  const handleToggle = (task: Task) => {
    const previous = task.status;
    actions.setTaskStatus(task.id, "done");
    toast.show({
      message: `«${task.title}» انجام شد`,
      tone: "success",
      action: { label: "بازگرداندن", onClick: () => actions.setTaskStatus(task.id, previous) },
    });
  };

  return (
    <>
      <header className="home-header">
        <div className="home-header__top">
          <div>
            <p className="home-header__day">{formatLongDate(now)}</p>
            <h1 className="home-header__greeting">سلام، {firstName}</h1>
          </div>
          <div className="home-header__actions">
            <IconButton
              variant="surface"
              className="home-header__bell"
              label={unread > 0 ? `اعلان‌ها، ${unread} خوانده‌نشده` : "اعلان‌ها"}
              onClick={() => navigate("/notifications")}
            >
              <Bell size={19} aria-hidden="true" />
              {unread > 0 ? (
                <span className="home-header__badge num" aria-hidden="true">{unread}</span>
              ) : null}
            </IconButton>
            <button
              type="button"
              className="home-header__avatar"
              aria-label="پروفایل من"
              onClick={() => navigate("/profile")}
            >
              <Avatar name={data.profile.fullName} size="sm" />
            </button>
          </div>
        </div>
        <p className="home-header__summary">
          <span>
            <strong className="num">{visits.length}</strong> بازدید امروز
          </span>
          <span className="dot-sep" aria-hidden="true">·</span>
          <span>
            <strong className="num">{openTasks(data).length}</strong> کار باز
          </span>
          <span className="dot-sep" aria-hidden="true">·</span>
          <span>
            <strong className="num">{openReportsCount(data)}</strong> گزارش در جریان
          </span>
        </p>
      </header>

      <Page>
        <AttentionCard items={attention} />

        <Section>
          <SectionHeader
            title="برنامه امروز"
            lead={
              nextVisit
                ? `${visits.length} بازدید · مورد بعدی ${nextVisit.caseTitle}`
                : visits.length
                  ? `${visits.length} بازدید برنامه‌ریزی شده`
                  : "برای امروز بازدیدی ثبت نشده است"
            }
            action={{ label: "تقویم", onClick: () => navigate("/calendar") }}
          />
          {visits.length > 0 ? (
            <VisitTimeline visits={visits} emphasisId={nextVisit?.id} />
          ) : (
            <EmptyState
              icon={<CalendarX2 size={22} />}
              title="برنامه امروز خالی است"
              description="بازدید یا جلسه‌ای برای امروز در تقویم شما ثبت نشده است."
              compact
              action={
                <button type="button" className="btn btn--secondary btn--sm" onClick={() => navigate("/calendar")}>
                  <CalendarDays size={15} aria-hidden="true" />
                  مشاهده تقویم هفته
                </button>
              }
            />
          )}
        </Section>

        <Section>
          <SectionHeader
            title="کارهای امروز"
            lead="کارهای باز نیازمند اقدام شما"
            action={{ label: "همه کارها", onClick: () => navigate("/tasks") }}
          />
          {homeTasks.length > 0 ? (
            <ul className="home-tasks">
              {homeTasks.map((task) => (
                <TaskRow key={task.id} task={task} now={now} onToggle={handleToggle} />
              ))}
            </ul>
          ) : (
            <div className="home-quiet">
              <CheckCircle2 size={17} aria-hidden="true" />
              <span>کاری برای امروز باقی نمانده است.</span>
            </div>
          )}
        </Section>

        <Section>
          <SectionHeader
            title="پایش کودکان"
            lead={`${monitoring.length} کودک در دوره پایش`}
            action={{ label: "همه کودکان", onClick: () => navigate("/cases?tab=monitoring") }}
          />
          <div className="stack stack-3">
            {monitoring.map(({ child, familyCase }) => (
              <MonitoringCard key={child.id} child={child} familyCase={familyCase} now={now} />
            ))}
          </div>
        </Section>
      </Page>
    </>
  );
}
