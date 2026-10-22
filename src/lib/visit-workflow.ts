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
