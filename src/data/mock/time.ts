import type { DateKey, ISODateTime } from '@/types';

/**
 * Mock clock.
 *
 * Every seeded date is expressed as an offset from *now*, so the demo always
 * shows a coherent "today" — overdue tasks are overdue, tomorrow's visit is
 * tomorrow. When the real API replaces this data the offsets disappear with it.
 */

const NOW = new Date();

function pad(n: number) {
  return String(n).padStart(2, '0');
}

/** Local `YYYY-MM-DD` for `now + offsetDays`. */
export function day(offsetDays = 0): DateKey {
  const d = new Date(NOW);
  d.setDate(d.getDate() + offsetDays);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

/** Local naive `YYYY-MM-DDTHH:mm` — parsed back as local time everywhere. */
export function at(offsetDays: number, hour: number, minute = 0): ISODateTime {
  return `${day(offsetDays)}T${pad(hour)}:${pad(minute)}`;
}

export function hoursAgo(hours: number): ISODateTime {
  const d = new Date(NOW.getTime() - hours * 3600_000);
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(
    d.getMinutes(),
  )}`;
}

export function minutesAgo(minutes: number): ISODateTime {
  return hoursAgo(minutes / 60);
}

export const TODAY = day(0);
export const NOW_ISO = hoursAgo(0);
