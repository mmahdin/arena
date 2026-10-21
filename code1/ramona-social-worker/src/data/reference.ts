import { startOfDay, toISO } from "@/lib/date";

/**
 * Demo clock.
 *
 * Mock timestamps are generated relative to "today at 08:15" so the workspace
 * always opens on a plausible working morning, no matter when the app is run.
 * Replace `NOW` with `new Date()` once real data is wired in.
 */
const reference = new Date();
reference.setHours(8, 15, 0, 0);

export const NOW = reference;

/** ISO string for a day offset from today at a given time. */
export function at(dayOffset: number, hour: number, minute = 0): string {
  const d = startOfDay(NOW);
  d.setDate(d.getDate() + dayOffset);
  d.setHours(hour, minute, 0, 0);
  return toISO(d);
}
