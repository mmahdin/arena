import type { Report, ReportField, ReportStatus } from '@/types';
import { db, findReport, replace } from '@/data/db';
import { templateFor } from '@/data/mock/report-templates';
import { latency, nowIso, uid } from './_utils';

export interface ReportQuery {
  status?: ReportStatus | 'all';
  caseId?: string;
  query?: string;
}

export function listReports(q: ReportQuery = {}): Promise<Report[]> {
  let rows = db.reports.slice();
  if (q.status && q.status !== 'all') rows = rows.filter((r) => r.status === q.status);
  if (q.caseId) rows = rows.filter((r) => r.caseId === q.caseId);
  if (q.query) {
    const needle = q.query.trim().toLowerCase();
    rows = rows.filter((r) => [r.title, r.caseCode, r.summary].join(' ').toLowerCase().includes(needle));
  }
  rows.sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  return latency(rows);
}

export function getReport(id: string): Promise<Report | undefined> {
  return latency(db.reports.find((r) => r.id === id));
}

export function reportsByCase(caseId: string): Report[] {
  return db.reports.filter((r) => r.caseId === caseId);
}

/** Persist a single field edit — the editor calls this on every change. */
export function updateField(
  reportId: string,
  sectionId: string,
  fieldId: string,
  value: string | number | string[],
): Report | undefined {
  const report = findReport(reportId);
  if (!report) return undefined;
  const next: Report = {
    ...report,
    updatedAt: nowIso(),
    sections: report.sections.map((s) =>
      s.id !== sectionId
        ? s
        : {
            ...s,
            fields: s.fields.map((f) =>
              f.id !== fieldId ? f : ({ ...f, value } as ReportField),
            ),
          },
    ),
  };
  db.reports = replace(db.reports, next);
  return next;
}

export function updateSummary(reportId: string, summary: string): Report | undefined {
  const report = findReport(reportId);
  if (!report) return undefined;
  const next = { ...report, summary, updatedAt: nowIso() };
  db.reports = replace(db.reports, next);
  return next;
}

export function updateTitle(reportId: string, title: string): Report | undefined {
  const report = findReport(reportId);
  if (!report) return undefined;
  const next = { ...report, title, updatedAt: nowIso() };
  db.reports = replace(db.reports, next);
  return next;
}

export function setReportStatus(reportId: string, status: ReportStatus): Promise<Report | undefined> {
  const report = findReport(reportId);
  if (!report) return latency(undefined);
  const next: Report = {
    ...report,
    status,
    updatedAt: nowIso(),
    submittedAt: status === 'submitted' || status === 'in-review' ? nowIso() : report.submittedAt,
    reviewer: status === 'in-review' ? (report.reviewer ?? 'دکتر رضا کاویانی') : report.reviewer,
  };
  db.reports = replace(db.reports, next);

  if (status === 'submitted' || status === 'in-review') {
    db.notifications = [
      {
        id: uid('n'),
        type: 'system',
        title: 'گزارش برای بازبینی ارسال شد',
        body: `«${next.title}» در انتظار بررسی سرپرست قرار گرفت.`,
        at: nowIso(),
        read: true,
        href: `/reports/${next.id}`,
      },
      ...db.notifications,
    ];
  }
  return latency(next, 260);
}

export function addAttachment(reportId: string, name: string): Report | undefined {
  const report = findReport(reportId);
  if (!report) return undefined;
  const next: Report = {
    ...report,
    updatedAt: nowIso(),
    attachments: [...report.attachments, { id: uid('at'), name, size: '—' }],
  };
  db.reports = replace(db.reports, next);
  return next;
}

export function statusCounts(): Promise<Record<ReportStatus, number>> {
  const base: Record<ReportStatus, number> = {
    draft: 0,
    submitted: 0,
    'in-review': 0,
    revision: 0,
    approved: 0,
  };
  for (const r of db.reports) base[r.status] += 1;
  return latency(base, 0);
}

/**
 * Create a draft from a template.
 *
 * The template is data (`src/data/mock/report-templates.ts`), so new
 * professional forms appear here without any new UI code.
 */
export function createDraftReport(input: {
  type: Report['type'];
  caseId: string;
  childId?: string;
  visitId?: string;
  title?: string;
}): Promise<Report> {
  // The report belongs to the visit. Opening its action twice must not
  // manufacture two drafts, even if a legacy link is only one-directional.
  if (input.visitId) {
    const visit = db.visits.find((v) => v.id === input.visitId);
    const existing = db.reports.find((r) => r.id === visit?.reportId) ?? db.reports.find((r) => r.visitId === input.visitId);
    if (existing) return latency(existing, 0);
  }
  const caseFile = db.cases.find((c) => c.id === input.caseId);
  const { title, sections } = templateFor(input.type);
  const child = caseFile?.children.find((c) => c.id === input.childId);
  const report: Report = {
    id: uid('r'),
    title: input.title ?? (child ? `${title} — ${child.firstName} ${caseFile?.familyName ?? ''}`.trim() : `${title} — ${caseFile?.familyName ?? ''}`.trim()),
    type: input.type,
    status: 'draft',
    caseId: input.caseId,
    caseCode: caseFile?.code ?? '',
    childId: input.childId,
    visitId: input.visitId,
    author: db.user.name,
    createdAt: nowIso(),
    updatedAt: nowIso(),
    summary: '',
    sections,
    comments: [],
    attachments: [],
  };
  db.reports = [report, ...db.reports];

  if (input.visitId) {
    const visit = db.visits.find((v) => v.id === input.visitId);
    if (visit) db.visits = replace(db.visits, { ...visit, reportId: report.id });
  }

  return latency(report, 200);
}
