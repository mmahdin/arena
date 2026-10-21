import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { CheckCheck, Inbox } from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import { parse } from "@/lib/date";
import { doneTasks, openTasks, overdueTasks, tasksDueToday } from "@/lib/selectors";
import type { Task, TaskType } from "@/types";
import { cn } from "@/lib/cn";
import { Page, PageHeader } from "@/components/app/Page";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilterChips } from "@/components/ui/FilterChips";
import { EmptyState } from "@/components/ui/EmptyState";
import { Button } from "@/components/ui/Button";
import { TaskRow } from "@/components/domain/TaskRow";
import { useToast } from "@/components/ui/Toast";
import { ReportsPanel } from "@/features/reports/ReportsPanel";
import "./TasksPage.css";

type View = "tasks" | "reports";
type TaskFilter = "all" | "urgent" | TaskType;

const taskFilters: { value: TaskFilter; label: string }[] = [
  { value: "all", label: "همه" },
  { value: "urgent", label: "فوری" },
  { value: "visit", label: "بازدید" },
  { value: "report", label: "گزارش" },
  { value: "assessment", label: "ارزیابی" },
  { value: "admin", label: "اداری" },
];

export function TasksPage() {
  const data = useWorkspace();
  const now = useNow();
  const actions = useAppActions();
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();
  const [filter, setFilter] = useState<TaskFilter>("all");
  const [showDone, setShowDone] = useState(false);

  const view: View = searchParams.get("view") === "reports" ? "reports" : "tasks";

  const open = openTasks(data);
  const overdue = overdueTasks(data);
  const dueToday = tasksDueToday(data);
  const done = doneTasks(data);

  const filtered = useMemo(() => {
    if (filter === "all") return open;
    if (filter === "urgent") return open.filter((t) => t.priority === "urgent" || t.priority === "high");
    return open.filter((t) => t.type === filter);
  }, [filter, open]);

  const groups = useMemo(() => {
    if (filter !== "all") return [{ key: "results", label: "", items: filtered }];
    const overdueIds = new Set(overdue.map((t) => t.id));
    const todayIds = new Set(dueToday.map((t) => t.id));
    const rest = filtered.filter((t) => !overdueIds.has(t.id) && !todayIds.has(t.id));
    return [
      { key: "overdue", label: "عقب‌افتاده", items: filtered.filter((t) => overdueIds.has(t.id)) },
      { key: "today", label: "امروز", items: filtered.filter((t) => todayIds.has(t.id)) },
      {
        key: "next",
        label: "پیش رو",
        items: rest.sort((a, b) => parse(a.dueAt).getTime() - parse(b.dueAt).getTime()),
      },
    ].filter((group) => group.items.length > 0);
  }, [filter, filtered, overdue, dueToday]);

  const handleToggle = (task: Task) => {
    const next = task.status === "done" ? "todo" : "done";
    actions.setTaskStatus(task.id, next);
    if (next === "done") {
      toast.show({
        message: "کار انجام شد.",
        tone: "success",
        action: { label: "بازگرداندن", onClick: () => actions.setTaskStatus(task.id, task.status) },
      });
    }
  };

  const filterCounts: Record<TaskFilter, number> = {
    all: open.length,
    urgent: open.filter((t) => t.priority === "urgent" || t.priority === "high").length,
    visit: open.filter((t) => t.type === "visit").length,
    report: open.filter((t) => t.type === "report").length,
    assessment: open.filter((t) => t.type === "assessment").length,
    monitoring: open.filter((t) => t.type === "monitoring").length,
    admin: open.filter((t) => t.type === "admin").length,
  };

  return (
    <Page>
      <PageHeader
        eyebrow="همه کارهای محول‌شده به شما"
        title="کارتابل"
        description={
          open.length > 0
            ? `${new Intl.NumberFormat("fa-IR").format(open.length)} کار باز، ${new Intl.NumberFormat("fa-IR").format(dueToday.length + overdue.length)} مورد در محدوده امروز`
            : "در حال حاضر کار بازی ندارید."
        }
      />

      <SegmentedControl<View>
        ariaLabel="انتخاب بخش کارتابل"
        value={view}
        onChange={(next) =>
          setSearchParams(next === "reports" ? { view: "reports" } : {}, { replace: true })
        }
        items={[
          { value: "tasks", label: "کارها", count: open.length },
          { value: "reports", label: "گزارش‌ها", count: data.reports.length },
        ]}
      />

      {view === "tasks" ? (
        <div className="tasks-view">
          <FilterChips<TaskFilter>
            ariaLabel="فیلتر کارها"
            items={taskFilters
              .filter((f) => f.value === "all" || f.value === "urgent" || filterCounts[f.value] > 0)
              .map((f) => ({ ...f, count: filterCounts[f.value] }))}
            value={filter}
            onChange={setFilter}
          />

          {groups.length === 0 ? (
            <EmptyState
              icon={<Inbox size={22} />}
              title={filter === "all" ? "کار بازی ندارید" : "کاری با این فیلتر نیست"}
              description={
                filter === "all"
                  ? "کارهای جدید از سوی سرپرست یا سامانه به کارتابل شما اضافه می‌شود."
                  : "فیلتر دیگری را امتحان کنید یا همه کارها را ببینید."
              }
              compact
              action={
                filter !== "all" ? (
                  <Button variant="secondary" size="sm" onClick={() => setFilter("all")}>
                    نمایش همه کارها
                  </Button>
                ) : undefined
              }
            />
          ) : (
            <div className="task-groups">
              {groups.map((group) => (
                <section className="task-group" key={group.key} aria-label={group.label || "نتایج"}>
                  {group.label ? (
                    <header className="task-group__head">
                      <span
                        className={cn(
                          "task-group__label",
                          group.key === "overdue" && "is-overdue",
                          group.key === "today" && "is-today"
                        )}
                      >
                        {group.label}
                      </span>
                      <span className="task-group__count num">
                        {new Intl.NumberFormat("fa-IR").format(group.items.length)}
                      </span>
                    </header>
                  ) : null}
                  <ul className="task-group__list">
                    {group.items.map((task) => (
                      <TaskRow key={task.id} task={task} now={now} onToggle={handleToggle} />
                    ))}
                  </ul>
                </section>
              ))}
            </div>
          )}

          {done.length > 0 ? (
            <section className="done-tasks">
              <button
                type="button"
                className="done-tasks__toggle"
                onClick={() => setShowDone((value) => !value)}
                aria-expanded={showDone}
              >
                <span className="done-tasks__title">
                  <CheckCheck size={16} aria-hidden="true" />
                  انجام‌شده
                  <span className="done-tasks__count num">{done.length}</span>
                </span>
                <span className="done-tasks__chevron">{showDone ? "بستن" : "نمایش"}</span>
              </button>
              {showDone ? (
                <ul className="task-group__list done-tasks__list">
                  {done.map((task) => (
                    <TaskRow key={task.id} task={task} now={now} onToggle={handleToggle} />
                  ))}
                </ul>
              ) : null}
            </section>
          ) : null}
        </div>
      ) : (
        <ReportsPanel />
      )}
    </Page>
  );
}
