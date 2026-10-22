import type { CaseStage, Report, Task, Visit, Wellbeing } from '@/types';
import { CASE_STAGE_ORDER } from './labels';
import { addDays, dayKey, daysUntil, isSameDay, persianWeekdayIndex } from './jalali';

/**
 * Pure derivations over domain data.
 *
 * Nothing here is async and nothing here mutates. Screens and services both
 * rely on these so a rule like "a scheduled visit whose time has passed needs
 * an outcome" is defined exactly once.
 */

/* ------------------------------------------------------------------ *
 * Visits
 * ------------------------------------------------------------------ */

export type VisitDisplayState = 'scheduled' | 'needs-outcome' | 'completed' | 'not-done' | 'cancelled';

/** A scheduled visit whose start time has already passed needs an outcome. */
export function visitNeedsOutcome(visit: Visit, now: Date = new Date()): boolean {
  if (visit.status !== 'scheduled') return false;
  const [h, m] = visit.time.split(':').map(Number);
  const start = new Date(`${visit.date}T00:00:00`);
  start.setHours(h, m, 0, 0);
  return now.getTime() > start.getTime();
}

export function visitDisplayState(visit: Visit, now: Date = new Date()): VisitDisplayState {
  switch (visit.status) {
    case 'completed':
      return 'completed';
    case 'family-absent':
    case 'reschedule':
      return 'not-done';
    case 'cancelled':
      return 'cancelled';
    default:
      return visitNeedsOutcome(visit, now) ? 'needs-outcome' : 'scheduled';
  }
}

export function isUpcoming(visit: Visit, now: Date = new Date()): boolean {
  return visit.status === 'scheduled' && !visitNeedsOutcome(visit, now);
}

/**
 * The visit lifecycle, seen from "what do I do with it" — the buckets of the
 * «همه بازدیدها» list. Report state is intentionally *not* a bucket: a
 * completed visit with an unfinished report is still a completed visit, and
 * the report is managed in the reports queue. The two lifecycles only ever
 * meet as a secondary signal on a row, never as one merged status.
 */
export type VisitBucket = 'needs-outcome' | 'upcoming' | 'completed' | 'not-done';

export function visitBucket(visit: Visit, now: Date = new Date()): VisitBucket {
  const state = visitDisplayState(visit, now);
  if (state === 'needs-outcome') return 'needs-outcome';
  if (state === 'scheduled') return 'upcoming';
  if (state === 'completed') return 'completed';
  return 'not-done';
}

export const VISIT_BUCKET_LABEL: Record<VisitBucket, string> = {
  'needs-outcome': 'نیازمند ثبت نتیجه',
  upcoming: 'پیش‌رو',
  completed: 'انجام‌شده',
  'not-done': 'انجام‌نشده و لغوشده',
};

/** Urgency first — the order filters and grouped lists are read in. */
export const VISIT_BUCKET_ORDER: VisitBucket[] = ['needs-outcome', 'upcoming', 'completed', 'not-done'];

/* ------------------------------------------------------------------ *
 * Assignment — «بازدیدهای من» در برابر «بازدیدهای آزاد»
 * ------------------------------------------------------------------ */

export const isMyVisit = (visit: Visit, userId: string): boolean => visit.assignedToId === userId;

export const isPoolVisit = (visit: Visit): boolean => visit.assignedToId === null;

/** Visits the signed-in social worker is responsible for. */
export function myVisits(visits: Visit[], userId: string): Visit[] {
  return visits.filter((v) => isMyVisit(v, userId));
}

/**
 * The pool: unassigned visits that are still claimable. A pool visit whose
 * time has passed is not offered for claiming — it goes back to the unit for
 * re-scheduling instead of silently expiring in someone's list.
 */
export function poolVisits(visits: Visit[], now: Date = new Date()): Visit[] {
  return visits
    .filter((v) => isPoolVisit(v) && v.status === 'scheduled' && !visitNeedsOutcome(v, now))
    .sort((a, b) => (a.date + a.time).localeCompare(b.date + b.time));
}

/**
 * A family belongs to my caseload once at least one of its visits is (or was)
 * mine. Families that only have pool visits stay out of «خانواده‌ها» until a
 * visit is claimed — claiming is the moment the family becomes my
 * responsibility, and its full record becomes visible to me.
 */
export function myCaseIds(visits: Visit[], userId: string): Set<string> {
  return new Set(myVisits(visits, userId).map((v) => v.caseId));
}

/* ------------------------------------------------------------------ *
 * Schedule fit — does a visit fit into my day?
 * ------------------------------------------------------------------ */

export function visitStart(visit: Visit): Date {
  const [h, m] = visit.time.split(':').map(Number);
  const start = new Date(`${visit.date}T00:00:00`);
  start.setHours(h, m, 0, 0);
  return start;
}

export const visitEnd = (visit: Visit): Date =>
  new Date(visitStart(visit).getTime() + visit.durationMin * 60_000);

/** Cancelled / not-done visits no longer occupy the day. */
const occupiesSchedule = (visit: Visit): boolean =>
  visit.status === 'scheduled' || visit.status === 'completed';

export function visitsOverlap(a: Visit, b: Visit): boolean {
  if (a.date !== b.date) return false;
  return visitStart(a) < visitEnd(b) && visitStart(b) < visitEnd(a);
}

/** My visits that collide with a candidate — shown before claiming. */
export function scheduleConflicts(candidate: Visit, mine: Visit[]): Visit[] {
  return mine
    .filter((v) => v.id !== candidate.id && occupiesSchedule(v) && visitsOverlap(v, candidate))
    .sort((a, b) => a.time.localeCompare(b.time));
}

/** My visits on the candidate's day — the load a claim would add to. */
export function visitsOnSameDay(visit: Visit, mine: Visit[]): Visit[] {
  return mine
    .filter((v) => v.id !== visit.id && v.date === visit.date && occupiesSchedule(v))
    .sort((a, b) => a.time.localeCompare(b.time));
}

/* ------------------------------------------------------------------ *
 * Tasks
 * ------------------------------------------------------------------ */

export type TaskBucket = 'overdue' | 'today' | 'tomorrow' | 'week' | 'later';

export const OPEN_TASK_STATUSES: Task['status'][] = ['todo', 'in-progress', 'revision', 'blocked'];

export function isOpenTask(task: Task): boolean {
  return OPEN_TASK_STATUSES.includes(task.status);
}

export function taskBucket(task: Task, now: Date = new Date()): TaskBucket {
  const d = daysUntil(task.dueDate, now);
  if (d < 0) return 'overdue';
  if (d === 0) return 'today';
  if (d === 1) return 'tomorrow';
  if (d <= 6) return 'week';
  return 'later';
}

export const TASK_BUCKET_LABEL: Record<TaskBucket, string> = {
  overdue: 'از مهلت گذشته',
  today: 'امروز',
  tomorrow: 'فردا',
  week: 'این هفته',
  later: 'بعدتر',
};

export const TASK_BUCKET_ORDER: TaskBucket[] = ['overdue', 'today', 'tomorrow', 'week', 'later'];

const PRIORITY_WEIGHT = { critical: 0, high: 1, medium: 2, low: 3 } as const;

/** Sort a work queue the way a social worker reads it: urgency, then priority. */
export function sortTasks(list: Task[]): Task[] {
  return [...list].sort((a, b) => {
    if (a.status === 'done' && b.status !== 'done') return 1;
    if (b.status === 'done' && a.status !== 'done') return -1;
    const byDate = a.dueDate.localeCompare(b.dueDate);
    if (byDate !== 0) return byDate;
    return PRIORITY_WEIGHT[a.priority] - PRIORITY_WEIGHT[b.priority];
  });
}

/* ------------------------------------------------------------------ *
 * Reports
 * ------------------------------------------------------------------ */

/**
 * The report lifecycle from the author's seat: something waits on me, on the
 * reviewer, or is finished. This queue is about reports only — a visit that
 * has no report yet is handled from the visit (recording «انجام شد» creates
 * the draft), so visits never appear as rows inside the reports queue.
 */
export type ReportQueue = 'action' | 'review' | 'approved';

export function reportQueue(report: Report): ReportQueue {
  if (report.status === 'approved') return 'approved';
  if (report.status === 'submitted' || report.status === 'in-review') return 'review';
  return 'action'; // draft | revision
}

export const REPORT_QUEUE_LABEL: Record<ReportQueue, string> = {
  action: 'نیازمند اقدام من',
  review: 'در انتظار بررسی',
  approved: 'تأییدشده',
};

export function reportCompletion(report: Report): { filled: number; total: number; pct: number } {
  let filled = 0;
  let total = 0;
  for (const section of report.sections) {
    for (const field of section.fields) {
      total += 1;
      if (field.kind === 'text' && field.value.trim().length > 0) filled += 1;
      else if (field.kind === 'select' && field.value.trim().length > 0) filled += 1;
      else if (field.kind === 'scale' && field.value > 0) filled += 1;
      else if (field.kind === 'checklist' && field.value.length > 0) filled += 1;
    }
  }
  return { filled, total, pct: total === 0 ? 0 : Math.round((filled / total) * 100) };
}

export function reportSectionsDone(report: Report): number {
  return report.sections.filter((s) =>
    s.fields.every((f) => {
      if (f.kind === 'checklist') return f.value.length > 0;
      if (f.kind === 'scale') return f.value > 0;
      return String(f.value).trim().length > 0;
    }),
  ).length;
}

/* ------------------------------------------------------------------ *
 * Cases & children
 * ------------------------------------------------------------------ */

export function caseStageIndex(stage: CaseStage): number {
  return CASE_STAGE_ORDER.indexOf(stage);
}

export function wellbeingTrend(history: { date: string; score: number }[]): 'up' | 'down' | 'flat' {
  if (history.length < 2) return 'flat';
  const sorted = [...history].sort((a, b) => a.date.localeCompare(b.date));
  const first = sorted[0].score;
  const last = sorted[sorted.length - 1].score;
  if (last > first) return 'up';
  if (last < first) return 'down';
  return 'flat';
}

export const wellbeingFromScore = (score: number): Wellbeing =>
  score >= 4 ? 'stable' : score === 3 ? 'attention' : 'critical';

/** Cases that still need the social worker's attention. */
export function isCaseActive(stage: CaseStage): boolean {
  return stage !== 'closed';
}

/* ------------------------------------------------------------------ *
 * Calendar / scheduling
 * ------------------------------------------------------------------ */

export function eventsOn<T extends { date: string }>(list: T[], date: Date | string): T[] {
  const key = typeof date === 'string' ? date : dayKey(date);
  return list.filter((x) => x.date === key);
}

export function weekRange(date: Date): { start: Date; end: Date } {
  const idx = persianWeekdayIndex(date);
  return { start: addDays(date, -idx), end: addDays(date, 6 - idx) };
}

export function isThisWeek(date: string, now: Date = new Date()): boolean {
  const { start, end } = weekRange(now);
  const d = new Date(`${date}T00:00:00`);
  return d.getTime() >= start.getTime() && d.getTime() <= end.getTime();
}

export function isTodayKey(date: string, now: Date = new Date()): boolean {
  return isSameDay(new Date(`${date}T00:00:00`), now);
}

/** Group any list by a key, preserving first-seen order. */
export function groupBy<T>(list: T[], keyFn: (item: T) => string): Map<string, T[]> {
  const map = new Map<string, T[]>();
  for (const item of list) {
    const key = keyFn(item);
    const bucket = map.get(key);
    if (bucket) bucket.push(item);
    else map.set(key, [item]);
  }
  return map;
}
