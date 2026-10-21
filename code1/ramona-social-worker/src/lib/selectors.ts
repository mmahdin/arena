import type { Child, Case, Report, Task, Visit } from "@/types";
import type { WorkspaceSnapshot } from "@/services/api";
import { diffDays, isSameDay, parse, startOfDay } from "@/lib/date";
import type { Tone } from "@/lib/labels";

/* ------------------------------------------------------------------ */
/* Visits                                                              */
/* ------------------------------------------------------------------ */

export function scheduledVisits(data: WorkspaceSnapshot): Visit[] {
  return data.visits
    .filter((v) => v.status === "scheduled")
    .sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime());
}

export function visitsOnDay(data: WorkspaceSnapshot, day: Date): Visit[] {
  return data.visits
    .filter((v) => isSameDay(parse(v.startsAt), day))
    .sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime());
}

export function todayVisits(data: WorkspaceSnapshot): Visit[] {
  return visitsOnDay(data, parse(data.now));
}

export function upcomingVisits(data: WorkspaceSnapshot): Visit[] {
  const today = startOfDay(parse(data.now)).getTime();
  return scheduledVisits(data)
    .filter((v) => startOfDay(parse(v.startsAt)).getTime() >= today)
    .sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime());
}

export function pastVisits(data: WorkspaceSnapshot): Visit[] {
  const today = startOfDay(parse(data.now)).getTime();
  return data.visits
    .filter((v) => startOfDay(parse(v.startsAt)).getTime() < today || v.status === "completed")
    .sort((a, b) => parse(b.startsAt).getTime() - parse(a.startsAt).getTime());
}

export function nextVisit(data: WorkspaceSnapshot): Visit | undefined {
  return upcomingVisits(data).find((v) => v.status === "scheduled");
}

/**
 * Case timeline order: upcoming visits first (soonest first), then past visits
 * (most recent first) — matching how a social worker scans the file.
 */
export function visitsForCase(data: WorkspaceSnapshot, caseId: string): Visit[] {
  const today = startOfDay(parse(data.now)).getTime();
  const all = data.visits.filter((v) => v.caseId === caseId);
  const upcoming = all
    .filter((v) => startOfDay(parse(v.startsAt)).getTime() >= today)
    .sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime());
  const past = all
    .filter((v) => startOfDay(parse(v.startsAt)).getTime() < today)
    .sort((a, b) => parse(b.startsAt).getTime() - parse(a.startsAt).getTime());
  return [...upcoming, ...past];
}

/* ------------------------------------------------------------------ */
/* Tasks                                                               */
/* ------------------------------------------------------------------ */

export function openTasks(data: WorkspaceSnapshot): Task[] {
  return data.tasks
    .filter((t) => t.status !== "done")
    .sort((a, b) => {
      const due = parse(a.dueAt).getTime() - parse(b.dueAt).getTime();
      return due !== 0 ? due : 0;
    });
}

export function doneTasks(data: WorkspaceSnapshot): Task[] {
  return data.tasks
    .filter((t) => t.status === "done")
    .sort((a, b) => parse(b.dueAt).getTime() - parse(a.dueAt).getTime());
}

export function overdueTasks(data: WorkspaceSnapshot): Task[] {
  const now = parse(data.now);
  return openTasks(data).filter((t) => startOfDay(parse(t.dueAt)) < startOfDay(now));
}

export function tasksDueToday(data: WorkspaceSnapshot): Task[] {
  const now = parse(data.now);
  return openTasks(data).filter((t) => isSameDay(parse(t.dueAt), now));
}

export function tasksForCase(data: WorkspaceSnapshot, caseId: string): Task[] {
  return data.tasks.filter((t) => t.caseId === caseId);
}

/* ------------------------------------------------------------------ */
/* Reports                                                             */
/* ------------------------------------------------------------------ */

export function reportsByStatus(data: WorkspaceSnapshot, status: Report["status"]): Report[] {
  return data.reports
    .filter((r) => r.status === status)
    .sort((a, b) => parse(b.updatedAt).getTime() - parse(a.updatedAt).getTime());
}

export function reportsForCase(data: WorkspaceSnapshot, caseId: string): Report[] {
  return data.reports
    .filter((r) => r.caseId === caseId)
    .sort((a, b) => parse(b.updatedAt).getTime() - parse(a.updatedAt).getTime());
}

/* ------------------------------------------------------------------ */
/* Monitoring                                                          */
/* ------------------------------------------------------------------ */

export interface MonitoredChild {
  child: Child;
  familyCase: Case;
}

export function monitoringChildren(data: WorkspaceSnapshot): MonitoredChild[] {
  const items: MonitoredChild[] = [];
  data.cases.forEach((c) => {
    c.children.forEach((child) => {
      if (child.monitoring) items.push({ child, familyCase: c });
    });
  });
  const severity = { critical: 0, needs_attention: 1, stable: 2 };
  return items.sort((a, b) => {
    const risk = severity[a.child.monitoring!.wellbeing] - severity[b.child.monitoring!.wellbeing];
    if (risk !== 0) return risk;
    const aNext = a.child.monitoring?.nextVisitAt ?? "";
    const bNext = b.child.monitoring?.nextVisitAt ?? "";
    return aNext.localeCompare(bNext);
  });
}

/* ------------------------------------------------------------------ */
/* Attention queue — what the social worker must not miss              */
/* ------------------------------------------------------------------ */

export interface AttentionItem {
  id: string;
  title: string;
  meta: string;
  href: string;
  tone: Tone;
  rank: number;
  kind: "report" | "task" | "visit" | "monitoring";
}

export function attentionItems(data: WorkspaceSnapshot): AttentionItem[] {
  const now = parse(data.now);
  const items: AttentionItem[] = [];
  const returned = reportsByStatus(data, "needs_revision");
  const returnedIds = new Set(returned.map((report) => report.id));

  returned.forEach((report) => {
    const days = diffDays(now, parse(report.updatedAt));
    items.push({
      id: `attention-report-${report.id}`,
      title: `اصلاح گزارش «${report.caseTitle}»`,
      meta: `بازگردانده‌شده توسط سرپرست${days > 0 ? ` · ${new Intl.NumberFormat("fa-IR").format(days)} روز پیش` : ""}`,
      href: `/reports/${report.id}`,
      tone: "warning",
      rank: 0,
      kind: "report",
    });
  });

  overdueTasks(data)
    .filter((task) => !task.relatedReportId || !returnedIds.has(task.relatedReportId))
    .forEach((task) => {
    items.push({
      id: `attention-task-${task.id}`,
      title: task.title,
      meta: task.caseTitle ? `مهلت گذشته · ${task.caseTitle}` : "مهلت گذشته",
      href: `/tasks/${task.id}`,
      tone: "danger",
      rank: 1,
      kind: "task",
    });
  });

  const todayUrgent = tasksDueToday(data).filter(
    (task) =>
      (task.priority === "urgent" || task.priority === "high") &&
      (!task.relatedReportId || !returnedIds.has(task.relatedReportId))
  );
  todayUrgent.forEach((task) => {
    items.push({
      id: `attention-task-${task.id}`,
      title: task.title,
      meta: task.caseTitle ? `مهلت امروز · ${task.caseTitle}` : "مهلت امروز",
      href: `/tasks/${task.id}`,
      tone: "accent",
      rank: 2,
      kind: "task",
    });
  });

  monitoringChildren(data)
    .filter((m) => m.child.monitoring?.wellbeing === "needs_attention")
    .forEach(({ child, familyCase }) => {
      items.push({
        id: `attention-child-${child.id}`,
        title: `پیگیری وضعیت ${child.fullName}`,
        meta: `پایش نیازمند توجه · ${familyCase.title}`,
        href: `/cases/${familyCase.id}`,
        tone: "warning",
        rank: 3,
        kind: "monitoring",
      });
    });

  return items.sort((a, b) => a.rank - b.rank);
}

/* ------------------------------------------------------------------ */
/* Counts & summaries                                                  */
/* ------------------------------------------------------------------ */

export function unreadNotifications(data: WorkspaceSnapshot): number {
  return data.notifications.filter((n) => !n.read).length;
}

export function activeCasesCount(data: WorkspaceSnapshot): number {
  return data.cases.filter((c) => c.stage !== "closed").length;
}

export function openReportsCount(data: WorkspaceSnapshot): number {
  return data.reports.filter((r) => r.status === "draft" || r.status === "needs_revision").length;
}

export function monthlyVisitCount(data: WorkspaceSnapshot): number {
  const now = parse(data.now);
  return data.visits.filter((v) => {
    const d = parse(v.startsAt);
    return d.getFullYear() === now.getFullYear() && d.getMonth() === now.getMonth() && v.status !== "canceled";
  }).length;
}

export function caseProgressPercent(data: WorkspaceSnapshot, caseId: string): number {
  const familyCase = data.cases.find((c) => c.id === caseId);
  if (!familyCase) return 0;
  return familyCase.stageProgress ?? 0;
}
