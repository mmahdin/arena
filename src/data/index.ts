import { cases } from './mock/cases';
import { notifications } from './mock/notifications';
import { reports } from './mock/reports';
import { tasks } from './mock/tasks';
import { currentUser } from './mock/user';
import { visits } from './mock/visits';

/**
 * The mock database.
 *
 * This is the ONLY place seed data is assembled. `src/services/*` reads from
 * here; screens never import `src/data` directly. Swapping to a real backend
 * therefore means rewriting the services, and nothing else.
 *
 * Assignment is applied here, not repeated on every seeded visit: everything
 * without an open-pool entry belongs to the signed-in worker, and pool visits
 * stay ownerless until they are claimed.
 */
export const seed = {
  user: currentUser,
  cases,
  visits: visits.map((visit) => (visit.pool ? visit : { ...visit, assigneeId: currentUser.id })),
  tasks,
  reports,
  notifications,
};

export type Seed = typeof seed;
