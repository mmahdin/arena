import type { Report, Visit, VisitStatus } from '@/types';
import { db, findCase, findTask, findVisit, replace } from '@/data/db';
import { dayKey } from '@/lib/jalali';
import { latency, nowIso, uid } from './_utils';

export interface VisitQuery {
  caseId?: string;
  /** 'upcoming' = today onwards and still scheduled */
  scope?: 'upcoming' | 'past' | 'today' | 'all';
}

export function listVisits(q: VisitQuery = {}): Promise<Visit[]> {
  const today = dayKey(new Date());
  let rows = db.visits.slice();
  if (q.caseId) rows = rows.filter((v) => v.caseId === q.caseId);
  if (q.scope === 'today') rows = rows.filter((v) => v.date === today);
  if (q.scope === 'upcoming') rows = rows.filter((v) => v.date >= today);
  if (q.scope === 'past') rows = rows.filter((v) => v.date < today);
  rows.sort((a, b) => (a.date + a.time).localeCompare(b.date + b.time));
  return latency(rows);
}

export function getVisit(id: string): Promise<Visit | undefined> {
  return latency(db.visits.find((v) => v.id === id));
}

export function toggleChecklistItem(visitId: string, itemId: string): Promise<Visit | undefined> {
  const visit = findVisit(visitId);
  if (!visit) return latency(undefined, 0);
  const next: Visit = {
    ...visit,
    checklist: visit.checklist.map((c) => (c.id === itemId ? { ...c, done: !c.done } : c)),
  };
  db.visits = replace(db.visits, next);
  return latency(next, 0);
}

export interface VisitOutcomeInput {
  status: VisitStatus;
  note: string;
}

export interface VisitOutcomeResult {
  visit: Visit;
  /** Draft report created for the visit, when applicable. */
  reportId?: string;
  taskId?: string;
}

/**
 * Record what happened at a visit.
 *
 * This is the single most important action in the field, so it does the whole
 * chain at once: the visit is closed, the linked task is completed, a case
 * timeline entry is written and a report draft is prepared for the office.
 */
export async function recordVisitOutcome(
  visitId: string,
  input: VisitOutcomeInput,
): Promise<VisitOutcomeResult | undefined> {
  const visit = findVisit(visitId);
  if (!visit) return undefined;

  const next: Visit = {
    ...visit,
    status: input.status,
    outcome: { status: input.status, note: input.note, at: nowIso() },
    checklist: visit.checklist,
  };
  db.visits = replace(db.visits, next);

  // Close the related work item.
  const task = db.tasks.find((t) => t.visitId === visitId);
  if (task) {
    db.tasks = replace(db.tasks, {
      ...task,
      status: input.status === 'completed' ? 'done' : task.status === 'done' ? task.status : 'todo',
      completedAt: input.status === 'completed' ? nowIso() : undefined,
    });
  }

  // Log it on the case.
  const caseFile = findCase(visit.caseId);
  if (caseFile) {
    caseFile.timeline = [
      {
        id: uid('e'),
        at: nowIso(),
        kind: 'visit',
        title:
          input.status === 'completed'
            ? 'بازدید انجام و نتیجه ثبت شد'
            : 'بازدید انجام نشد — نتیجه ثبت شد',
        detail: input.note || undefined,
        actor: db.user.name,
      },
      ...caseFile.timeline,
    ];
    caseFile.lastActivityAt = nowIso();
  }

  // A completed visit always produces a report draft to finish later.
  let reportId: string | undefined;
  if (input.status === 'completed' && !next.reportId) {
    reportId = uid('r');
    const report: Report = {
      id: reportId,
      title: `گزارش بازدید — ${caseFile?.familyName ?? 'پرونده'}`,
      type: visit.purpose === 'initial-assessment' ? 'home-assessment' : visit.purpose === 'periodic-monitoring' ? 'monitoring' : 'visit',
      status: 'draft',
      caseId: visit.caseId,
      caseCode: caseFile?.code ?? '',
      visitId,
      childId: visit.childIds?.[0],
      author: db.user.name,
      createdAt: nowIso(),
      updatedAt: nowIso(),
      summary: input.note,
      sections: [],
      comments: [],
      attachments: [],
    };
    db.reports = [report, ...db.reports];
    next.reportId = reportId;
    db.visits = replace(db.visits, next);
  }

  return latency({ visit: next, reportId, taskId: task?.id }, 260);
}

export function visitsByCase(caseId: string): Visit[] {
  return db.visits.filter((v) => v.caseId === caseId);
}

export function visitById(id: string): Visit | undefined {
  return findVisit(id);
}

export function taskForVisit(id: string) {
  return findTask(db.tasks.find((t) => t.visitId === id)?.id ?? '');
}
