import type { AppNotification } from '@/types';
import { db, replace } from '@/data/db';
import { latency, nowIso } from './_utils';

export function listNotifications(onlyUnread = false): Promise<AppNotification[]> {
  const rows = onlyUnread ? db.notifications.filter((n) => !n.read) : db.notifications.slice();
  rows.sort((a, b) => b.at.localeCompare(a.at));
  return latency(rows);
}

export function unreadCount(): number {
  return db.notifications.filter((n) => !n.read).length;
}

export function markRead(id: string): Promise<AppNotification | undefined> {
  const n = db.notifications.find((x) => x.id === id);
  if (!n) return latency(undefined, 0);
  const next = { ...n, read: true };
  db.notifications = replace(db.notifications, next);
  return latency(next, 0);
}

export function markAllRead(): Promise<number> {
  db.notifications = db.notifications.map((n) => ({ ...n, read: true }));
  return latency(db.notifications.length, 120);
}

export function push(notification: Omit<AppNotification, 'id' | 'at' | 'read'>): void {
  db.notifications = [{ ...notification, id: nowIso(), at: nowIso(), read: false }, ...db.notifications];
}
