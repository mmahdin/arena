import type { Report, Tone, Visit, VisitPurpose } from '@/types';
import { visitDisplayState } from './domain';
import { REPORT_STATUS, VISIT_PURPOSE, VISIT_STATUS } from './labels';

/** Every visit has one place in the UI, regardless of where it is opened from. */
export function visitCategory(purpose: VisitPurpose): 'initial' | 'follow-up' | 'other' {
  if (purpose === 'initial-assessment') return 'initial';
  if (purpose === 'introduction') return 'other'; // A meeting is not a home visit.
  return 'follow-up';
}

export function visitKindLabel(visit: Visit): string {
  const kind = visitCategory(visit.purpose);
  return kind === 'initial' ? 'بازدید اولیه' : kind === 'follow-up' ? 'بازدید پیگیری' : VISIT_PURPOSE[visit.purpose].label;
}

/** Both legacy report.visitId and visit.reportId links are supported. */
export function reportForVisit(visit: Visit, reports: Report[]): Report | undefined {
  return reports.find((r) => r.id === visit.reportId) ?? reports.find((r) => r.visitId === visit.id);
}

/* ------------------------------------------------------------------ *
 * My visits — one queue per responsibility
 * ------------------------------------------------------------------ */

/** Mutually exclusive buckets, so the counts always add up to the total. */
export type VisitQueue = 'action' | 'upcoming' | 'done' | 'not-done';

export type VisitFilter = 'all' | VisitQueue;

export const VISIT_FILTERS: { value: VisitFilter; label: string }[] = [
  { value: 'all', label: 'همه' },
  { value: 'action', label: 'نیازمند اقدام' },
  { value: 'upcoming', label: 'پیش‌رو' },
  { value: 'done', label: 'انجام‌شده' },
  { value: 'not-done', label: 'انجام‌نشده' },
];

/** A completed visit whose report is still missing or unfinished. */
export function visitNeedsReport(visit: Visit, reports: Report[]): boolean {
  if (visit.status !== 'completed') return false;
  const report = reportForVisit(visit, reports);
  return !report || report.status === 'draft';
}

/**
 * Why a visit still asks something of the worker, if it does.
 *
 * Report revisions are deliberately absent: once a report is submitted the
 * visit is closed, and the remaining work belongs to the report queue.
 */
export function visitActionReason(
  visit: Visit,
  reports: Report[],
  now: Date = new Date(),
): 'outcome' | 'report' | null {
  if (visit.status === 'scheduled') return visitDisplayState(visit, now) === 'needs-outcome' ? 'outcome' : null;
  if (visitNeedsReport(visit, reports)) return 'report';
  return null;
}

export function visitQueue(visit: Visit, reports: Report[], now: Date = new Date()): VisitQueue {
  if (visitActionReason(visit, reports, now)) return 'action';
  if (visit.status === 'scheduled') return 'upcoming';
  if (visit.status === 'completed') return 'done';
  return 'not-done';
}

export function visitQueueCounts(
  visits: Visit[],
  reports: Report[],
  now: Date = new Date(),
): Record<VisitFilter, number> {
  const counts: Record<VisitFilter, number> = { all: visits.length, action: 0, upcoming: 0, done: 0, 'not-done': 0 };
  for (const visit of visits) counts[visitQueue(visit, reports, now)] += 1;
  return counts;
}

/** Soonest first for what is ahead; most recent first for what happened. */
export function sortVisitQueue(visits: Visit[], filter: VisitFilter): Visit[] {
  const rows = [...visits];
  return filter === 'upcoming' || filter === 'action'
    ? rows.sort(visitChronological)
    : rows.sort((a, b) => visitChronological(b, a));
}

export function visitProgress(visit: Visit, reports: Report[]): { label: string; tone: Tone } {
  const state = visitDisplayState(visit);
  if (state === 'needs-outcome') return { label: 'ثبت نتیجه', tone: 'warning' };
  if (state === 'scheduled') return { label: 'برنامه‌ریزی‌شده', tone: 'brand' };
  if (state !== 'completed') return VISIT_STATUS[visit.status];
  const report = reportForVisit(visit, reports);
  if (!report) return { label: 'گزارش ثبت نشده', tone: 'warning' };
  if (report.status === 'draft') return { label: 'تکمیل گزارش', tone: 'warning' };
  return REPORT_STATUS[report.status];
}

export const visitChronological = (a: Visit, b: Visit) =>
  (a.date + a.time).localeCompare(b.date + b.time);

export function nextVisit(visits: Visit[]): Visit | undefined {
  const now = new Date();
  return visits
    .filter((v) => visitDisplayState(v, now) === 'scheduled')
    .sort(visitChronological)[0];
}
