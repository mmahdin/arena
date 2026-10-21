import {
  addDays,
  isSameDay,
  persianParts,
  startOfDay,
  startOfWeek,
  toISO,
  toNumber,
} from "./date";

export interface CalendarCell {
  date: Date;
  key: string;
  day: number;
  inMonth: boolean;
  isToday: boolean;
}

export interface PersianMonth {
  label: string;
  /** First and last day of the month (noon-safe local midnights). */
  first: Date;
  last: Date;
  /** 1–31, Jalali month length. */
  length: number;
  cells: CalendarCell[];
}

/** Day of month of a date inside a Persian month, as a number. */
export function persianDayOfMonth(date: Date): number {
  return toNumber(persianParts(date).day);
}

/**
 * Builds a Saturday-first grid for the Persian month containing `anchor`.
 * Month boundaries are discovered by walking day-by-day (max 32 steps), which
 * is accurate for every Jalali month and avoids a date-conversion dependency.
 */
export function buildPersianMonth(anchor: Date, today: Date): PersianMonth {
  const target = startOfDay(anchor);
  const parts = persianParts(target);

  const sameMonth = (date: Date) => {
    const p = persianParts(date);
    return p.year === parts.year && p.month === parts.month;
  };

  let first = target;
  for (let i = 0; i < 32; i++) {
    const previous = addDays(first, -1);
    if (!sameMonth(previous)) break;
    first = previous;
  }

  let last = target;
  for (let i = 0; i < 32; i++) {
    const next = addDays(last, 1);
    if (!sameMonth(next)) break;
    last = next;
  }

  const gridStart = startOfWeek(first);
  const gridEnd = addDays(startOfWeek(last), 6);
  const cells: CalendarCell[] = [];
  let cursor = gridStart;

  while (cursor.getTime() <= gridEnd.getTime()) {
    const p = persianParts(cursor);
    cells.push({
      date: cursor,
      key: toISO(cursor).slice(0, 10),
      day: toNumber(p.day),
      inMonth: p.year === parts.year && p.month === parts.month,
      isToday: isSameDay(cursor, today),
    });
    cursor = addDays(cursor, 1);
  }

  return {
    label: `${parts.month} ${parts.year}`,
    first,
    last,
    length: toNumber(persianParts(last).day),
    cells,
  };
}

/** True when both dates belong to the same Jalali month & year. */
export function isSamePersianMonth(a: Date, b: Date): boolean {
  const pa = persianParts(a);
  const pb = persianParts(b);
  return pa.year === pb.year && pa.month === pb.month;
}
