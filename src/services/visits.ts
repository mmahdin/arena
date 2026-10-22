import type { Report, Visit, VisitStatus } from '@/types';
import { db, findCase, findTask, findVisit, replace } from '@/data/db';
import { dayKey } from '@/lib/jalali';
import { latency, nowIso, uid } from './_utils';
import { templateFor } from '@/data/mock/report-templates';

export interface VisitQuery {
  caseId?: string;
  /** 'upcoming' = today onwards and still scheduled */
  scope?: 'upcoming' | 'past' | 'today' | 'all';
  /**
   * Whose visits: mine, the unassigned pool, or everything the client is
   * allowed to see (the store loads 'all' once and derives both lists with
   * the pure selectors in `lib/domain`, so claiming a visit updates every
   * screen from a single state change).
   */
  assignment?: 'mine' | 'pool' | 'all';
}

export function listVisits(q: VisitQuery = {}): Promise<Visit[]> {
  const today = dayKey(new Date());
  let rows = db.visits.slice();
  if (q.caseId) rows = rows.filter((v) => v.caseId === q.caseId);
  if (q.assignment === 'mine') rows = rows.filter((v) => v.assignedToId === db.user.id);
  if (q.assignment === 'pool') rows = rows.filter((v) => v.assignedToId === null);
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

/**
 * Claim a pool visit.
 *
 * The visit already has a fixed date and time; claiming only changes who is
 * responsible for it. From this moment the family's full record is part of
 * the claimer's caseload (see `myCaseIds` in `lib/domain`), the visit shows
 * up in their daily schedule, and it leaves the pool for everyone else.
 */
export function claimVisit(visitId: string): Promise<Visit | undefined> {
  const visit = findVisit(visitId);
  if (!visit || visit.assignedToId !== null || visit.status !== 'scheduled') {
    return latency(undefined, 0);
  }
  const next: Visit = { ...visit, assignedToId: db.user.id, claimedAt: nowIso() };
  db.visits = replace(db.visits, next);

  const caseFile = findCase(visit.caseId);
  if (caseFile) {
    caseFile.timeline = [
      {
        id: uid('e'),
        at: nowIso(),
        kind: 'visit',
        title: 'بازدید از مخزن بازدیدهای آزاد پذیرفته شد',
        detail: 'مسئولیت انجام بازدید به مددکار پذیرنده واگذار شد.',
        actor: db.user.name,
      },
      ...caseFile.timeline,
    ];
    caseFile.lastActivityAt = nowIso();
  }

  // The unit (and the social worker's own feed) can see the visit is covered.
  db.notifications = [
    {
      id: uid('n'),
      type: 'assignment',
      title: 'بازدید آزاد پذیرفته شد',
      body: `بازدید ساعت ${next.time} به برنامه شما اضافه شد و اطلاعات کامل خانواده در دسترس است.`,
      at: nowIso(),
      read: true,
      href: `/visits/${next.id}`,
    },
    ...db.notifications,
  ];

  return latency(next, 260);
}

/**
 * Return one of my upcoming visits to the pool, so a colleague can take it.
 * Only visits that have not happened yet can be released — once an outcome
 * exists, the visit (and its report) stays with whoever performed it.
 */
export function releaseVisit(visitId: string): Promise<Visit | undefined> {
  const visit = findVisit(visitId);
  if (!visit || visit.assignedToId !== db.user.id || visit.status !== 'scheduled') {
    return latency(undefined, 0);
  }
  const next: Visit = { ...visit, assignedToId: null, claimedAt: undefined };
  db.visits = replace(db.visits, next);

  const caseFile = findCase(visit.caseId);
  if (caseFile) {
    caseFile.timeline = [
      {
        id: uid('e'),
        at: nowIso(),
        kind: 'visit',
        title: 'بازدید به مخزن بازدیدهای آزاد بازگردانده شد',
        detail: 'بازدید در انتظار پذیرش توسط مددکار دیگری است.',
        actor: db.user.name,
      },
      ...caseFile.timeline,
    ];
    caseFile.lastActivityAt = nowIso();
  }

  return latency(next, 260);
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
