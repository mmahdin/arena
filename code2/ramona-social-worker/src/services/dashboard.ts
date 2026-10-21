import type { CalendarEvent, CaseFile, Tone, UserProfile, Visit } from '@/types';
import { db } from '@/data/db';
import { isOpenTask, sortTasks, visitDisplayState } from '@/lib/domain';
import { dayKey, daysUntil, faDate, greeting } from '@/lib/jalali';
import { WELLBEING } from '@/lib/labels';
import { latency } from './_utils';
import { buildEvents } from './calendar';
import { listChildrenSync, type ChildSummary } from './cases';

/**
 * The "today" payload.
 *
 * Everything the social worker needs the moment the app opens is assembled
 * here, in one call, already prioritised. Screens stay presentational; the
 * ranking rules live in this service where they can evolve safely.
 */

export type AttentionKind = 'revision' | 'overdue' | 'visit-outcome' | 'wellbeing' | 'deadline';

export interface AttentionItem {
  id: string;
  kind: AttentionKind;
  title: string;
  detail: string;
  tone: Tone;
  href: string;
}

export interface AgendaItem {
  id: string;
  kind: CalendarEvent['kind'];
  title: string;
  subtitle?: string;
  time?: string;
  durationMin?: number;
  href: string;
  tone: Tone;
  state: 'upcoming' | 'done' | 'needs-outcome';
  visit?: Visit;
}

export interface TodayPayload {
  user: UserProfile;
  greeting: string;
  dateLabel: string;
  agenda: AgendaItem[];
  attention: AttentionItem[];
  upcomingTasks: ReturnType<typeof sortTasks>;
  monitoring: ChildSummary[];
  stats: {
    agendaTotal: number;
    agendaDone: number;
    openTasks: number;
    unread: number;
    activeCases: number;
  };
}

const today = () => dayKey(new Date());

export function buildAttention(): AttentionItem[] {
  const items: AttentionItem[] = [];
  const now = new Date();
  const todayKey = today();

  // 1. Reports the supervisor sent back — these block a case moving forward.
  for (const r of db.reports.filter((x) => x.status === 'revision')) {
    items.push({
      id: `at-${r.id}`,
      kind: 'revision',
      title: 'گزارش نیازمند اصلاح',
      detail: `${r.title} — ${r.comments[0]?.author ?? 'سرپرست'}`,
      tone: 'danger',
      href: `/reports/${r.id}`,
    });
  }

  // 2. Overdue work.
  for (const t of db.tasks.filter((x) => isOpenTask(x) && x.dueDate < todayKey)) {
    items.push({
      id: `at-${t.id}`,
      kind: 'overdue',
      title: `${Math.abs(daysUntil(t.dueDate, now))} روز از مهلت گذشته`,
      detail: t.title,
      tone: 'danger',
      href: t.reportId ? `/reports/${t.reportId}` : '/work',
    });
  }

  // 3. Visits that happened but were never closed out.
  for (const v of db.visits.filter((x) => visitDisplayState(x, now) === 'needs-outcome')) {
    const c = db.cases.find((x) => x.id === v.caseId);
    items.push({
      id: `at-${v.id}`,
      kind: 'visit-outcome',
      title: 'ثبت نتیجه بازدید',
      detail: `${c?.familyName ?? ''} — ساعت ${v.time}`,
      tone: 'warning',
      href: `/visits/${v.id}`,
    });
  }

  // 4. A child whose wellbeing needs intervention.
  for (const ch of listChildrenSync().filter((c) => c.wellbeing === 'critical')) {
    items.push({
      id: `at-${ch.id}`,
      kind: 'wellbeing',
      title: 'وضعیت کودک نیازمند مداخله',
      detail: `${ch.firstName} — ${ch.familyName}`,
      tone: 'danger',
      href: `/children/${ch.id}`,
    });
  }

  // 5. Anything critical due today or tomorrow.
  for (const t of db.tasks.filter(
    (x) => isOpenTask(x) && x.priority === 'critical' && x.dueDate >= todayKey && daysUntil(x.dueDate, now) <= 1,
  )) {
    items.push({
      id: `at-${t.id}`,
      kind: 'deadline',
      title: daysUntil(t.dueDate, now) === 0 ? 'مهلت امروز' : 'مهلت فردا',
      detail: t.title,
      tone: 'warning',
      href: t.visitId ? `/visits/${t.visitId}` : t.reportId ? `/reports/${t.reportId}` : '/work',
    });
  }

  const severity: Record<Tone, number> = { danger: 0, warning: 1, accent: 2, brand: 3, info: 4, success: 5, neutral: 6 };
  return items.sort((a, b) => severity[a.tone] - severity[b.tone]).slice(0, 4);
}

export function buildAgenda(date: Date = new Date()): AgendaItem[] {
  const key = dayKey(date);
  const now = new Date();
  const items: AgendaItem[] = buildEvents()
    .filter((e) => e.date === key)
    .map<AgendaItem>((e) => {
      const visit = e.kind === 'visit' ? db.visits.find((v) => v.id === e.id.replace('cal-', '')) : undefined;
      const state = visit
        ? visitDisplayState(visit, now) === 'completed'
          ? 'done'
          : visitDisplayState(visit, now) === 'needs-outcome'
            ? 'needs-outcome'
            : 'upcoming'
        : e.kind === 'deadline'
          ? 'upcoming'
          : 'upcoming';
      const caseFile: CaseFile | undefined = e.caseCode
        ? db.cases.find((c) => c.code === e.caseCode)
        : undefined;
      return {
        id: e.id,
        kind: e.kind,
        title: e.kind === 'visit' ? (caseFile?.familyName ?? e.title) : e.title,
        subtitle: e.kind === 'visit' ? e.title.split('—')[0].trim() : caseFile?.code,
        time: e.time,
        durationMin: e.durationMin,
        href: e.href ?? '#',
        tone: e.tone,
        state,
        visit,
      };
    });

  return items.sort((a, b) => (a.time ?? '99:99').localeCompare(b.time ?? '99:99'));
}

export function buildToday(): TodayPayload {
  const now = new Date();
  const agenda = buildAgenda(now);
  const monitoring = listChildrenSync()
    .filter((ch) => daysUntil(ch.nextFollowUp, now) <= 10)
    .sort((a, b) => a.nextFollowUp.localeCompare(b.nextFollowUp))
    .slice(0, 3);

  return {
    user: db.user,
    greeting: greeting(now),
    dateLabel: faDate(now, { weekday: true }),
    agenda,
    attention: buildAttention(),
    upcomingTasks: sortTasks(db.tasks.filter((t) => isOpenTask(t) && t.dueDate >= today())).slice(0, 4),
    monitoring,
    stats: {
      agendaTotal: agenda.length,
      agendaDone: agenda.filter((a) => a.state === 'done').length,
      openTasks: db.tasks.filter(isOpenTask).length,
      unread: db.notifications.filter((n) => !n.read).length,
      activeCases: db.cases.filter((c) => c.stage !== 'closed').length,
    },
  };
}

export function getToday(): Promise<TodayPayload> {
  return latency(buildToday());
}

export function monitoringList(): ChildSummary[] {
  return listChildrenSync();
}

export const WELLBEING_LABEL = (w: keyof typeof WELLBEING) => WELLBEING[w].label;
