import type { CalendarEvent } from '@/types';
import { db } from '@/data/db';
import { buildMonthGrid, dayKey } from '@/lib/jalali';
import { toLatinDigits } from '@/lib/digits';
import { VISIT_PURPOSE } from '@/lib/labels';
import { latency } from './_utils';

/**
 * Calendar feed.
 *
 * Visits, task deadlines and meetings are normalised into one `CalendarEvent`
 * shape, so the month view has a single source to render. When the backend
 * exposes a real agenda endpoint, only this module changes.
 */
export function buildEvents(): CalendarEvent[] {
  const events: CalendarEvent[] = [];

  for (const v of db.visits) {
    const c = db.cases.find((x) => x.id === v.caseId);
    events.push({
      id: `cal-${v.id}`,
      title: `${VISIT_PURPOSE[v.purpose].label} — ${c?.familyName ?? ''}`.trim(),
      kind: 'visit',
      date: v.date,
      time: v.time,
      durationMin: v.durationMin,
      href: `/visits/${v.id}`,
      tone: v.status === 'completed' ? 'success' : v.status === 'cancelled' ? 'neutral' : 'brand',
      caseCode: c?.code,
    });
  }

  for (const t of db.tasks) {
    if (t.visitId) continue; // visits already represent their own slot; retain other completed tasks
    // Only show a meeting time when the assignment explicitly contains one.
    const match = t.type === 'meeting' ? toLatinDigits(t.description ?? '').match(/ساعت\s*(\d{1,2}):(\d{2})/) : null;
    const scheduledTime = match ? `${match[1].padStart(2, '0')}:${match[2]}` : undefined;
    events.push({
      id: `cal-${t.id}`,
      title: t.title,
      kind: t.type === 'meeting' ? 'meeting' : 'deadline',
      date: t.dueDate,
      time: scheduledTime,
      href: t.reportId ? `/reports/${t.reportId}` : t.caseId ? `/cases/${t.caseId}` : `/calendar?date=${t.dueDate}`,
      tone: t.status === 'done' ? 'success' : t.status === 'revision' ? 'warning' : t.priority === 'critical' ? 'danger' : t.priority === 'high' ? 'warning' : 'info',
      caseCode: t.caseCode,
      taskId: t.id,
      taskStatus: t.status,
    });
  }

  return events.sort((a, b) => (a.date + (a.time ?? '')).localeCompare(b.date + (b.time ?? '')));
}

export function eventsForDay(date: Date | string): Promise<CalendarEvent[]> {
  const key = typeof date === 'string' ? date : dayKey(date);
  return latency(
    buildEvents()
      .filter((e) => e.date === key)
      .sort((a, b) => (a.time ?? '99:99').localeCompare(b.time ?? '99:99')),
    0,
  );
}

/** Events keyed by day for a whole Jalali month — powers the day indicators. */
export function monthEventIndex(jy: number, jm: number): Map<string, CalendarEvent[]> {
  const keys = new Set(buildMonthGrid(jy, jm).map((c) => dayKey(c.date)));
  const index = new Map<string, CalendarEvent[]>();
  for (const e of buildEvents()) {
    if (!keys.has(e.date)) continue;
    const list = index.get(e.date) ?? [];
    list.push(e);
    index.set(e.date, list);
  }
  return index;
}
