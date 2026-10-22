import type { Report, Task, Visit, VisitStatus } from '@/types';
import { db, findCase, findTask, findVisit, replace } from '@/data/db';
import { dayKey } from '@/lib/jalali';
import { latency, nowIso, uid } from './_utils';
import { templateFor } from '@/data/mock/report-templates';
import { visitKindLabel } from '@/lib/visit-workflow';

export interface VisitQuery {
  caseId?: string;
  /**
   * `mine` (default) = visits owned by the signed-in worker;
   * `pool` = visits published to the open pool and not yet claimed.
   */
  assignment?: 'mine' | 'pool' | 'all';
  /** 'upcoming' = today onwards and still scheduled */
  scope?: 'upcoming' | 'past' | 'today' | 'all';
}

const isPool = (visit: Visit) => Boolean(visit.pool);
const isMine = (visit: Visit) => !visit.pool && visit.assigneeId === db.user.id;

export function listVisits(q: VisitQuery = {}): Promise<Visit[]> {
  const today = dayKey(new Date());
  let rows = db.visits.slice();
  if (q.assignment === 'pool') rows = rows.filter(isPool);
  else if (q.assignment === 'all') rows = rows.slice();
  else rows = rows.filter(isMine);
  if (q.caseId) rows = rows.filter((v) => v.caseId === q.caseId);
  if (q.scope === 'today') rows = rows.filter((v) => v.date === today);
  if (q.scope === 'upcoming') rows = rows.filter((v) => v.date >= today);
  if (q.scope === 'past') rows = rows.filter((v) => v.date < today);
  rows.sort((a, b) => (a.date + a.time).localeCompare(b.date + b.time));
  return latency(rows);
}

/** Visits that still wait for a social worker, soonest first. */
export function listPoolVisits(): Promise<Visit[]> {
  const rows = db.visits
    .filter(isPool)
    .sort((a, b) => (a.date + a.time).localeCompare(b.date + b.time));
  return latency(rows);
}

export function getVisit(id: string): Promise<Visit | undefined> {
  return latency(db.visits.find((v) => v.id === id));
}

/**
 * Claim a visit from the open pool.
 *
 * One transaction: the visit leaves the pool and gains an owner, a work item
 * is created for the day, and the case timeline records who accepted it — so
 * the calendar, the visit queue and the family history agree immediately.
 */
export async function claimVisit(visitId: string): Promise<Visit | undefined> {
  const visit = findVisit(visitId);
  if (!visit || !visit.pool) return latency(undefined, 0);

  const claimed: Visit = {
    ...visit,
    assigneeId: db.user.id,
    assignedBy: 'پذیرش شخصی از مخزن آزاد',
    pool: undefined,
  };
  db.visits = replace(db.visits, claimed);

  const caseFile = findCase(visit.caseId);
  const workItem: Task = {
    id: uid('t'),
    title: `${visitKindLabel(visit)} — ${caseFile?.familyName ?? 'خانواده'}`,
    type: 'visit',
    status: 'todo',
    priority: caseFile?.priority ?? 'medium',
    dueDate: visit.date,
    createdAt: nowIso(),
    caseId: visit.caseId,
    caseCode: caseFile?.code,
    childId: visit.childIds?.[0],
    visitId: visit.id,
    assignedBy: 'پذیرش شخصی از مخزن آزاد',
    description: visit.summary,
    estimatedMin: visit.durationMin,
  };
  db.tasks = [workItem, ...db.tasks];

  if (caseFile) {
    caseFile.timeline = [
      {
        id: uid('e'),
        at: nowIso(),
        kind: 'visit',
        title: 'بازدید از مخزن آزاد پذیرفته شد',
        detail: `${db.user.name} — ${visit.city}`,
        actor: db.user.name,
      },
      ...caseFile.timeline,
    ];
    caseFile.lastActivityAt = nowIso();
  }

  return latency(claimed, 240);
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
  if (!visit || visit.pool) return undefined;

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

  // A completed visit links to exactly one report. Legacy mock data may
  // contain report.visitId without the reverse visit.reportId reference.
  let reportId: string | undefined;
  if (input.status === 'completed') {
    const existing = db.reports.find((r) => r.id === next.reportId) ?? db.reports.find((r) => r.visitId === visitId);
    if (existing) {
      reportId = existing.id;
    } else {
      reportId = uid('r');
      const type: Report['type'] = visit.purpose === 'initial-assessment' ? 'home-assessment' : visit.purpose === 'periodic-monitoring' ? 'monitoring' : 'visit';
      const { title, sections } = templateFor(type);
      const report: Report = {
        id: reportId,
        title: `${title} — ${caseFile?.familyName ?? 'خانواده'}`,
        type,
        status: 'draft',
        caseId: visit.caseId,
        caseCode: caseFile?.code ?? '',
        visitId,
        childId: visit.childIds?.[0],
        author: db.user.name,
        createdAt: nowIso(),
        updatedAt: nowIso(),
        summary: input.note,
        sections,
        comments: [],
        attachments: [],
      };
      db.reports = [report, ...db.reports];
    }
    next.reportId = reportId;
    db.visits = replace(db.visits, next);
  }

  return latency({ visit: next, reportId, taskId: task?.id }, 260);
}

export function visitsByCase(caseId: string): Visit[] {
  return db.visits.filter((v) => v.caseId === caseId && isMine(v));
}

export function visitById(id: string): Visit | undefined {
  return findVisit(id);
}

export function taskForVisit(id: string) {
  return findTask(db.tasks.find((t) => t.visitId === id)?.id ?? '');
}
