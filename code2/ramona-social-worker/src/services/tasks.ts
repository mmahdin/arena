import type { Task, TaskStatus, TaskType } from '@/types';
import { db, findTask, replace } from '@/data/db';
import { isOpenTask, sortTasks } from '@/lib/domain';
import { dayKey, addDays } from '@/lib/jalali';
import { latency, nowIso } from './_utils';

export interface TaskQuery {
  query?: string;
  type?: TaskType | 'all';
  scope?: 'open' | 'done' | 'all';
}

export function listTasks(q: TaskQuery = {}): Promise<Task[]> {
  let rows = db.tasks.slice();
  if (q.scope === 'open') rows = rows.filter(isOpenTask);
  if (q.scope === 'done') rows = rows.filter((t) => t.status === 'done');
  if (q.type && q.type !== 'all') rows = rows.filter((t) => t.type === q.type);
  if (q.query) {
    const needle = q.query.trim().toLowerCase();
    rows = rows.filter((t) =>
      [t.title, t.caseCode ?? '', t.assignedBy, t.description ?? ''].join(' ').toLowerCase().includes(needle),
    );
  }
  return latency(sortTasks(rows));
}

export function getTask(id: string): Promise<Task | undefined> {
  return latency(db.tasks.find((t) => t.id === id));
}

export function setTaskStatus(id: string, status: TaskStatus): Promise<Task | undefined> {
  const task = findTask(id);
  if (!task) return latency(undefined);
  const next: Task = {
    ...task,
    status,
    completedAt: status === 'done' ? nowIso() : undefined,
  };
  db.tasks = replace(db.tasks, next);
  return latency(next, 120);
}

export const completeTask = (id: string) => setTaskStatus(id, 'done');
export const reopenTask = (id: string) => setTaskStatus(id, 'todo');

/** Push a deadline forward — the most common field action after a visit moves. */
export function postponeTask(id: string, days: number): Promise<Task | undefined> {
  const task = findTask(id);
  if (!task) return latency(undefined);
  const next: Task = { ...task, dueDate: dayKey(addDays(task.dueDate, days)), status: task.status === 'done' ? task.status : 'todo' };
  db.tasks = replace(db.tasks, next);
  return latency(next, 120);
}

export function countsByStatus(): Promise<Record<'open' | 'revision' | 'overdue' | 'done', number>> {
  const today = dayKey(new Date());
  return latency({
    open: db.tasks.filter(isOpenTask).length,
    revision: db.tasks.filter((t) => t.status === 'revision').length,
    overdue: db.tasks.filter((t) => isOpenTask(t) && t.dueDate < today).length,
    done: db.tasks.filter((t) => t.status === 'done').length,
  });
}
