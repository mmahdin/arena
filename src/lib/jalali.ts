/**
 * Jalali (Solar Hijri) date engine.
 *
 * One small, well-tested module owns every calendar concern: conversion,
 * month grids, and Persian date/time formatting. Screens never do date maths
 * themselves — they call these helpers, which keeps the whole app coherent
 * when the calendar logic eventually moves server-side.
 */
import jalaali from 'jalaali-js';
import { fa } from './digits';

export interface JalaliDate {
  jy: number;
  jm: number;
  jd: number;
}

export const JALALI_MONTHS = [
  'فروردین',
  'اردیبهشت',
  'خرداد',
  'تیر',
  'مرداد',
  'شهریور',
  'مهر',
  'آبان',
  'آذر',
  'دی',
  'بهمن',
  'اسفند',
] as const;

/** Week starts on Saturday, as everywhere in Iran. */
export const WEEKDAYS_SHORT = ['ش', 'ی', 'د', 'س', 'چ', 'پ', 'ج'] as const;
export const WEEKDAYS_LONG = [
  'شنبه',
  'یکشنبه',
  'دوشنبه',
  'سه‌شنبه',
  'چهارشنبه',
  'پنجشنبه',
  'جمعه',
] as const;

/* ------------------------------------------------------------------ *
 * Conversion
 * ------------------------------------------------------------------ */

export function toJalali(date: Date): JalaliDate {
  return jalaali.toJalaali(date.getFullYear(), date.getMonth() + 1, date.getDate());
}

export function toGregorian(j: JalaliDate): Date {
  const g = jalaali.toGregorian(j.jy, j.jm, j.jd);
  return new Date(g.gy, g.gm - 1, g.gd);
}

export function jalaliMonthLength(jy: number, jm: number): number {
  return jalaali.jalaaliMonthLength(jy, jm);
}

/** Persian weekday index for a Date — 0 = شنبه … 6 = جمعه */
export function persianWeekdayIndex(date: Date): number {
  return (date.getDay() + 1) % 7;
}

/* ------------------------------------------------------------------ *
 * Normalisation
 * ------------------------------------------------------------------ */

export function startOfDay(d: Date | string): Date {
  const x = typeof d === 'string' ? new Date(d) : new Date(d);
  x.setHours(0, 0, 0, 0);
  return x;
}

export function addDays(d: Date | string, days: number): Date {
  const x = startOfDay(d);
  x.setDate(x.getDate() + days);
  return x;
}

export function addMonthsJalali(j: JalaliDate, months: number): JalaliDate {
  let jy = j.jy;
  let jm = j.jm + months;
  while (jm < 1) {
    jm += 12;
    jy -= 1;
  }
  while (jm > 12) {
    jm -= 12;
    jy += 1;
  }
  const len = jalaliMonthLength(jy, jm);
  return { jy, jm, jd: Math.min(j.jd, len) };
}

/** Stable local day key — used for grouping and map lookups. */
export function dayKey(d: Date | string): string {
  const x = typeof d === 'string' ? new Date(d) : d;
  return `${x.getFullYear()}-${String(x.getMonth() + 1).padStart(2, '0')}-${String(
    x.getDate(),
  ).padStart(2, '0')}`;
}

export function isSameDay(a: Date | string, b: Date | string): boolean {
  return dayKey(a) === dayKey(b);
}

export function isToday(d: Date | string, now: Date = new Date()): boolean {
  return isSameDay(d, now);
}

export function daysBetween(from: Date | string, to: Date | string): number {
  return Math.round((startOfDay(to).getTime() - startOfDay(from).getTime()) / 86400000);
}

/* ------------------------------------------------------------------ *
 * Formatting
 * ------------------------------------------------------------------ */

export interface FaDateOptions {
  /** «شنبه ۱۲ آبان ۱۴۰۴» */
  weekday?: boolean;
  /** Include the year. Defaults to true. */
  year?: boolean;
  /** «۱۴۰۴/۰۸/۱۲» instead of «۱۲ آبان ۱۴۰۴» */
  numeric?: boolean;
}

/** Format a date in Persian. `faDate(d)` → «۱۲ آبان ۱۴۰۴» */
export function faDate(d: Date | string, opts: FaDateOptions = {}): string {
  const date = typeof d === 'string' ? new Date(d) : d;
  const { jy, jm, jd } = toJalali(date);
  const { weekday = false, year = true, numeric = false } = opts;

  const parts: string[] = [];
  if (weekday) parts.push(WEEKDAYS_LONG[persianWeekdayIndex(date)]);
  if (numeric) {
    parts.push(`${fa(jy)}/${fa(String(jm).padStart(2, '0'))}/${fa(String(jd).padStart(2, '0'))}`);
  } else {
    parts.push(`${fa(jd)} ${JALALI_MONTHS[jm - 1]}`);
    if (year) parts.push(fa(jy));
  }
  return parts.join(' ');
}

/** «آبان ۱۴۰۴» — calendar header */
export function faMonthTitle(jy: number, jm: number): string {
  return `${JALALI_MONTHS[jm - 1]} ${fa(jy)}`;
}

/** 24-hour clock with Persian digits: «۰۹:۳۰» */
export function faTime(time: string | Date): string {
  if (typeof time === 'string' && /^\d{2}:\d{2}$/.test(time)) return fa(time);
  const d = typeof time === 'string' ? new Date(time) : time;
  return fa(`${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`);
}

/** «۰۹:۳۰ — ۱۰:۳۰» */
export function faTimeRange(start: string, durationMin: number): string {
  const [h, m] = start.split(':').map(Number);
  const endTotal = h * 60 + m + durationMin;
  const end = `${String(Math.floor(endTotal / 60) % 24).padStart(2, '0')}:${String(
    endTotal % 60,
  ).padStart(2, '0')}`;
  return `${faTime(start)} — ${faTime(end)}`;
}

/** Duration label: ۹۰ → «۱ ساعت و ۳۰ دقیقه» */
export function faDuration(minutes: number): string {
  if (minutes < 60) return `${fa(minutes)} دقیقه`;
  const h = Math.floor(minutes / 60);
  const m = minutes % 60;
  return m === 0 ? `${fa(h)} ساعت` : `${fa(h)} ساعت و ${fa(m)} دقیقه`;
}

/**
 * Relative past time for feeds: «هم‌اکنون», «۵ دقیقه پیش», «دیروز», «۳ روز پیش».
 */
export function relativeTime(iso: string | Date, now: Date = new Date()): string {
  const then = typeof iso === 'string' ? new Date(iso) : iso;
  const mins = Math.round((now.getTime() - then.getTime()) / 60000);
  if (mins < 1) return 'هم‌اکنون';
  if (mins < 60) return `${fa(mins)} دقیقه پیش`;
  const hours = Math.round(mins / 60);
  if (hours < 24) return `${fa(hours)} ساعت پیش`;
  const days = Math.round(hours / 24);
  if (days === 1) return 'دیروز';
  if (days < 7) return `${fa(days)} روز پیش`;
  const weeks = Math.round(days / 7);
  if (weeks < 5) return `${fa(weeks)} هفته پیش`;
  return faDate(then, { year: false });
}

/**
 * Deadline label with built-in urgency semantics for a work queue.
 * Returns the label; use `daysUntil` alongside it to pick a colour.
 */
export function deadlineLabel(iso: string | Date, now: Date = new Date()): string {
  const days = daysBetween(now, iso);
  if (days === 0) return 'امروز';
  if (days === 1) return 'فردا';
  if (days === -1) return 'دیروز';
  if (days > 1 && days <= 6) return `${fa(days)} روز دیگر`;
  if (days < -1 && days >= -6) return `${fa(Math.abs(days))} روز تأخیر`;
  return faDate(iso, { year: false });
}

export function daysUntil(iso: string | Date, now: Date = new Date()): number {
  return daysBetween(now, iso);
}

/* ------------------------------------------------------------------ *
 * Calendar grid
 * ------------------------------------------------------------------ */

export interface CalendarCell {
  jalali: JalaliDate;
  date: Date;
  inMonth: boolean;
  isToday: boolean;
}

/**
 * Build a 6×7 month grid starting on Saturday, padded with the neighbouring
 * months so the layout never shifts. `today` is injected to stay testable.
 */
export function buildMonthGrid(jy: number, jm: number, today: Date = new Date()): CalendarCell[] {
  const first = toGregorian({ jy, jm, jd: 1 });
  const leading = persianWeekdayIndex(first);
  const gridStart = addDays(first, -leading);
  const cells: CalendarCell[] = [];

  for (let i = 0; i < 42; i += 1) {
    const date = addDays(gridStart, i);
    const jalali = toJalali(date);
    cells.push({
      jalali,
      date,
      inMonth: jalali.jm === jm && jalali.jy === jy,
      isToday: isSameDay(date, today),
    });
  }
  return cells;
}

/** A greeting appropriate to the time of day. */
export function greeting(now: Date = new Date()): string {
  const h = now.getHours();
  if (h < 5) return 'شب بخیر';
  if (h < 12) return 'صبح بخیر';
  if (h < 17) return 'ظهر بخیر';
  if (h < 21) return 'عصر بخیر';
  return 'شب بخیر';
}
