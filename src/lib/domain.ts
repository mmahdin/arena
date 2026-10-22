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

/** Minutes since midnight of the visit's start — ordering and overlap checks. */
export function visitStartMinutes(visit: Visit): number {
  const [h, m] = visit.time.split(':').map(Number);
  return h * 60 + m;
}

export function visitEndMinutes(visit: Visit): number {
  return visitStartMinutes(visit) + visit.durationMin;
}

/** Two visits on the same day whose time windows intersect. */
export function visitsOverlap(a: Visit, b: Visit): boolean {
  return a.date === b.date && visitStartMinutes(a) < visitEndMinutes(b) && visitStartMinutes(b) < visitEndMinutes(a);
}

/**
 * The next commitment that would collide with `visit`.
 *
 * Accepting an open-pool visit is a commitment, so the worker sees a conflict
 * before it happens rather than discovering a double booking on the way.
 */
export function conflictingVisit(visit: Visit, others: Visit[]): Visit | undefined {
  return others.find((other) => other.id !== visit.id && visitsOverlap(visit, other));
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
