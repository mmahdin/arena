/**
 * Date & number formatting for a Persian (Jalali) interface.
 *
 * All domain timestamps are local ISO-like strings ("2026-10-21T09:30:00").
 * Formatting is centralised here so a future locale switch is a one-file change.
 */

const FA_DIGITS = ["۰", "۱", "۲", "۳", "۴", "۵", "۶", "۷", "۸", "۹"];

export function fa(value: string | number): string {
  return String(value).replace(/\d/g, (d) => FA_DIGITS[Number(d)]);
}

export function toNumber(value: string | number): number {
  const map: Record<string, string> = {};
  FA_DIGITS.forEach((d, i) => (map[d] = String(i)));
  return Number(String(value).replace(/[۰-۹]/g, (d) => map[d] ?? d).replace(/[^\d.-]/g, ""));
}

/* ---------------------------------------------------------------- */
/* Core helpers                                                      */
/* ---------------------------------------------------------------- */

export function toISO(date: Date): string {
  const p = (n: number) => String(n).padStart(2, "0");
  return `${date.getFullYear()}-${p(date.getMonth() + 1)}-${p(date.getDate())}T${p(date.getHours())}:${p(date.getMinutes())}:00`;
}

export function parse(value: string): Date {
  return new Date(value);
}

export function startOfDay(date: Date): Date {
  const d = new Date(date);
  d.setHours(0, 0, 0, 0);
  return d;
}

export function addDays(date: Date, days: number): Date {
  const d = new Date(date);
  d.setDate(d.getDate() + days);
  return d;
}

export function addMinutes(date: Date, minutes: number): Date {
  return new Date(date.getTime() + minutes * 60000);
}

export function isSameDay(a: Date, b: Date): boolean {
  return (
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate()
  );
}

/** Whole-day difference, independent of clock time. */
export function diffDays(target: Date, base: Date): number {
  return Math.round(
    (startOfDay(target).getTime() - startOfDay(base).getTime()) / 86_400_000
  );
}

/** Iranian week starts on Saturday. */
export function startOfWeek(date: Date): Date {
  const d = startOfDay(date);
  const offset = (d.getDay() + 1) % 7; // Sat = 0 … Fri = 6
  return addDays(d, -offset);
}

/* ---------------------------------------------------------------- */
/* Persian calendar formatting                                       */
/* ---------------------------------------------------------------- */

type Parts = { year: string; month: string; day: string; weekday: string };

const formatter = new Intl.DateTimeFormat("fa-IR-u-ca-persian", {
  weekday: "long",
  day: "numeric",
  month: "long",
  year: "numeric",
});

export function persianParts(date: Date): Parts {
  const parts = formatter.formatToParts(date);
  const get = (type: string) => parts.find((p) => p.type === type)?.value ?? "";
  return {
    year: get("year"),
    month: get("month"),
    day: get("day"),
    weekday: get("weekday"),
  };
}

export function formatLongDate(date: Date): string {
  const { weekday, day, month, year } = persianParts(date);
  return `${weekday} ${day} ${month} ${year}`;
}

export function formatDate(date: Date): string {
  const { day, month, year } = persianParts(date);
  return `${day} ${month} ${year}`;
}

export function formatDayMonth(date: Date): string {
  const { day, month } = persianParts(date);
  return `${day} ${month}`;
}

export function formatMonthYear(date: Date): string {
  const { month, year } = persianParts(date);
  return `${month} ${year}`;
}

const timeFormatter = new Intl.DateTimeFormat("fa-IR", {
  hour: "2-digit",
  minute: "2-digit",
  hour12: false,
});

export function formatTime(date: Date): string {
  return timeFormatter.format(date).replace(/\u200f/g, "");
}

export function formatDateTime(date: Date): string {
  return `${formatDate(date)}، ساعت ${formatTime(date)}`;
}

/** «امروز»، «فردا»، «دیروز» or a short weekday + date. */
export function formatDayLabel(date: Date, now: Date): string {
  const delta = diffDays(date, now);
  if (delta === 0) return "امروز";
  if (delta === 1) return "فردا";
  if (delta === -1) return "دیروز";
  const { weekday, day, month } = persianParts(date);
  return `${weekday} ${day} ${month}`;
}

/* Weekday initials, Saturday first. */
export const WEEKDAYS_SHORT = ["ش", "ی", "د", "س", "چ", "پ", "ج"];
export const WEEKDAYS_LONG = [
  "شنبه",
  "یک‌شنبه",
  "دوشنبه",
  "سه‌شنبه",
  "چهارشنبه",
  "پنج‌شنبه",
  "جمعه",
];

export function weekdayIndex(date: Date): number {
  return (date.getDay() + 1) % 7;
}

/* ---------------------------------------------------------------- */
/* Relative & deadline phrasing                                      */
/* ---------------------------------------------------------------- */

export function formatRelative(date: Date, now: Date): string {
  const minutes = Math.round((now.getTime() - date.getTime()) / 60000);
  if (minutes < 1) return "همین حالا";
  if (minutes < 60) return `${fa(minutes)} دقیقه پیش`;
  const hours = Math.round(minutes / 60);
  if (hours < 24) return `${fa(hours)} ساعت پیش`;
  const delta = diffDays(now, date);
  if (delta === 1) return "دیروز";
  if (delta < 7) return `${fa(delta)} روز پیش`;
  if (delta < 30) return `${fa(Math.round(delta / 7))} هفته پیش`;
  return formatDate(date);
}

export type DeadlineTone = "overdue" | "today" | "soon" | "normal";

export interface DeadlineLabel {
  text: string;
  tone: DeadlineTone;
  days: number;
}

/** Human deadline used on tasks: «۲ روز گذشته»، «امروز»، «فردا»، «۳ روز مانده». */
export function deadlineLabel(dueAt: string, now: Date): DeadlineLabel {
  const due = parse(dueAt);
  const days = diffDays(due, now);
  if (days < 0) {
    return {
      text: days === -1 ? "۱ روز گذشته" : `${fa(Math.abs(days))} روز گذشته`,
      tone: "overdue",
      days,
    };
  }
  if (days === 0) {
    const hours = Math.max(0, Math.round((due.getTime() - now.getTime()) / 3_600_000));
    return {
      text: hours <= 3 ? `امروز تا ${formatTime(due)}` : "امروز",
      tone: "today",
      days,
    };
  }
  if (days === 1) return { text: "فردا", tone: "soon", days };
  if (days <= 3) return { text: `${fa(days)} روز مانده`, tone: "soon", days };
  return { text: `${fa(days)} روز مانده`, tone: "normal", days };
}

/** Duration in a friendly Persian form: «۴۵ دقیقه»، «۱ ساعت و ۳۰ دقیقه». */
export function formatDuration(minutes: number): string {
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  if (h && m) return `${fa(h)} ساعت و ${fa(m)} دقیقه`;
  if (h) return `${fa(h)} ساعت`;
  return `${fa(m)} دقیقه`;
}

export function formatNumber(n: number): string {
  return new Intl.NumberFormat("fa-IR").format(n);
}

export function formatPercent(n: number): string {
  return `${fa(new Intl.NumberFormat("fa-IR").format(Math.round(n)))}٪`;
}
