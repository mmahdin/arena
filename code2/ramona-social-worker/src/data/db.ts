import type { AppNotification, CaseFile, Report, Task, UserProfile, Visit } from '@/types';
import { seed } from './index';

/**
 * In-memory database.
 *
 * A deep clone of the seed, mutated by the service layer so interactions
 * (completing a task, recording a visit outcome, submitting a report) feel
 * real during the session. This module is the seam that disappears when the
 * backend arrives — `src/services/*` is the only code that touches it.
 */
function clone<T>(value: T): T {
  return JSON.parse(JSON.stringify(value)) as T;
}

export interface Db {
  user: UserProfile;
  cases: CaseFile[];
  visits: Visit[];
  tasks: Task[];
  reports: Report[];
  notifications: AppNotification[];
}

export const db: Db = {
  user: seed.user,
  cases: clone(seed.cases),
  visits: clone(seed.visits),
  tasks: clone(seed.tasks),
  reports: clone(seed.reports),
  notifications: clone(seed.notifications),
};

/* ------------------------------------------------------------------ *
 * Lookup helpers used by the services
 * ------------------------------------------------------------------ */

export const findCase = (id: string) => db.cases.find((c) => c.id === id);
export const findVisit = (id: string) => db.visits.find((v) => v.id === id);
export const findTask = (id: string) => db.tasks.find((t) => t.id === id);
export const findReport = (id: string) => db.reports.find((r) => r.id === id);

export const allChildren = () => db.cases.flatMap((c) => c.children.map((ch) => ({ ...ch, caseCode: c.code, familyName: c.familyName })));
export const findChild = (id: string) => allChildren().find((ch) => ch.id === id);

/** Replace an entity inside a collection after a mutation. */
export function replace<T extends { id: string }>(list: T[], next: T): T[] {
  const i = list.findIndex((x) => x.id === next.id);
  if (i === -1) return list;
  const copy = list.slice();
  copy[i] = next;
  return copy;
}

/** Reset to the original seed — handy while developing. */
export function resetDb(): void {
  db.cases = clone(seed.cases);
  db.visits = clone(seed.visits);
  db.tasks = clone(seed.tasks);
  db.reports = clone(seed.reports);
  db.notifications = clone(seed.notifications);
}
