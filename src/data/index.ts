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
 */
export const seed = {
  user: currentUser,
  cases,
  visits,
  tasks,
  reports,
  notifications,
};

export type Seed = typeof seed;
