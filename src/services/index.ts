import type { AppNotification, CaseFile, Report, Task, UserProfile, Visit } from '@/types';
import { db } from '@/data/db';
import { latency } from './_utils';

/**
 * Service facade — the app's single entry point to data.
 *
 * Every screen and the store go through `@/services`. Each function is async
 * and returns plain domain objects, so replacing the in-memory `db` with HTTP
 * calls (fetch/axios + DTO mapping) is a local change inside these files.
 */

export * from './cases';
export * from './tasks';
export * from './visits';
export * from './reports';
export * from './notifications';
export * from './calendar';
export * from './dashboard';

export function getCurrentUser(): Promise<UserProfile> {
  return latency(db.user);
}

/** Everything the app shell needs on first paint, in one round-trip. */
export async function bootstrap(): Promise<{
  user: UserProfile;
  cases: CaseFile[];
  visits: Visit[];
  poolVisits: Visit[];
  tasks: Task[];
  reports: Report[];
  notifications: AppNotification[];
}> {
  return latency({
    user: db.user,
    cases: db.cases,
    visits: db.visits.filter((visit) => !visit.pool && visit.assigneeId === db.user.id),
    poolVisits: db.visits.filter((visit) => visit.pool),
    tasks: db.tasks,
    reports: db.reports,
    notifications: db.notifications,
  });
}
