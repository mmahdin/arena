import type { Report, Tone, Visit, VisitPurpose } from '@/types';
import { visitDisplayState } from './domain';
import { VISIT_PURPOSE, VISIT_STATUS } from './labels';

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

export interface StatusMeta {
  label: string;
  tone: Tone;
}

/**
 * The visit's own status — and nothing else.
 *
 * A completed visit used to display its *report's* status («تأییدشده»,
 * «تکمیل گزارش»…), which merged two different lifecycles into one badge and
 * made lists unreadable: the same word meant "the visit happened" in one row
 * and "the paperwork is approved" in another. Visit status and report status
 * are now always rendered as two separate signals.
 */
export function visitStatusMeta(visit: Visit, now: Date = new Date()): StatusMeta {
  const state = visitDisplayState(visit, now);
  if (state === 'needs-outcome') return { label: 'ثبت نتیجه', tone: 'warning' };
  if (state === 'scheduled') return { label: 'برنامه‌ریزی‌شده', tone: 'brand' };
  return VISIT_STATUS[visit.status];
}

/**
 * The report belonging to a completed visit, as a secondary signal.
 * Returns `null` while the visit itself is not completed — there is no
 * report to talk about yet, and rows stay clean.
 */
export function visitReportMeta(
  visit: Visit,
  reports: Report[],
): (StatusMeta & { report?: Report }) | null {
  if (visit.status !== 'completed') return null;
  const report = reportForVisit(visit, reports);
  if (!report) return { label: 'گزارش ثبت نشده', tone: 'warning' };
  switch (report.status) {
    case 'draft':
      return { label: 'گزارش: پیش‌نویس', tone: 'warning', report };
    case 'revision':
      return { label: 'گزارش: نیازمند اصلاح', tone: 'danger', report };
    case 'submitted':
    case 'in-review':
      return { label: 'گزارش: در بررسی', tone: 'info', report };
    case 'approved':
      return { label: 'گزارش: تأییدشده', tone: 'success', report };
  }
}

export const visitChronological = (a: Visit, b: Visit) =>
  (a.date + a.time).localeCompare(b.date + b.time);

export function nextVisit(visits: Visit[]): Visit | undefined {
  const now = new Date();
  return visits
    .filter((v) => visitDisplayState(v, now) === 'scheduled')
    .sort(visitChronological)[0];
}
