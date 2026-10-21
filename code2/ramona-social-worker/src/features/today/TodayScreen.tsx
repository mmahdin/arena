import { useMemo } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Bell, EyeOff, ShieldCheck } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { buildToday } from '@/services/dashboard';
import { Screen, Section } from '@/components/layout/Screen';
import { SectionAction, SectionHeader } from '@/components/ui/SectionHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Avatar } from '@/components/ui/Avatar';
import { ListGroup } from '@/components/ui/ListRow';
import { EmptyState } from '@/components/ui/EmptyState';
import { Skeleton, RowSkeleton } from '@/components/ui/Skeleton';
import { TaskRow } from '@/features/_shared/TaskRow';
import { AgendaTimeline } from '@/features/_shared/AgendaTimeline';
import { AttentionList } from '@/features/_shared/AttentionList';
import { ChildFollowUpRow } from '@/features/_shared/ChildRow';
import { fa } from '@/lib/digits';
import { Inbox } from 'lucide-react';

/**
 * Today — the landing screen.
 *
 * Reading order answers three questions in sequence: what is broken, what is
 * my day, what is next. Nothing here is decorative, and nothing requires the
 * social worker to think about the platform's internal structure.
 */
export function TodayScreen() {
  const { tasks, visits, reports, notifications, user, status, unreadCount } = useApp();
  const { fieldPrivacy, toggleFieldPrivacy } = useSettings();
  const navigate = useNavigate();

  // Recomputed from the store, which the service layer keeps authoritative.
  const today = useMemo(
    () => buildToday(),
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [tasks, visits, reports, notifications, status],
  );

  if (status === 'loading' || !user) {
    return (
      <Screen>
        <div className="space-y-3 pt-6">
          <Skeleton className="h-4 w-24" />
          <Skeleton className="h-7 w-48" />
          <Skeleton className="h-16 w-full rounded-lg" />
        </div>
        <div className="mt-6">
          <RowSkeleton />
          <RowSkeleton />
          <RowSkeleton />
        </div>
      </Screen>
    );
  }

  const firstName = user.name.split(' ')[0];
  const { agenda, attention, upcomingTasks, monitoring, stats } = today;

  return (
    <Screen bottomGap={false}>
      {/* ---------------------------------------------------------------- */}
      <header className="sticky top-0 z-30 -mx-gutter bg-bg/85 px-gutter pb-3 pt-4 backdrop-blur-xl">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-medium text-ink-3">{today.dateLabel}</p>
            <h1 className="mt-0.5 truncate text-xl font-extrabold tracking-tight text-ink">
              {today.greeting}، {firstName}
            </h1>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            {fieldPrivacy && (
              <button
                type="button"
                onClick={toggleFieldPrivacy}
                className="press flex h-10 items-center gap-1.5 rounded-md bg-accent-soft px-2.5 text-2xs font-bold text-accent-ink"
                aria-label="خاموش کردن حالت محرمانگی"
              >
                <EyeOff className="size-4" strokeWidth={2.4} aria-hidden />
                محرمانه
              </button>
            )}
            <IconButton label="اعلان‌ها" badge={unreadCount} onClick={() => navigate('/notifications')}>
              <Bell className="size-[20px]" strokeWidth={2} />
            </IconButton>
            <Link to="/profile" aria-label="پروفایل" className="press ms-0.5">
              <Avatar name={user.name} size="sm" tone="brand" />
            </Link>
          </div>
        </div>
      </header>

      {/* ---------------------------------------------------------------- */}
      <div className="mt-1 grid grid-cols-3 overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
        <StatCell value={stats.agendaTotal} label="برنامه امروز" to="/calendar" />
        <StatCell value={stats.openTasks} label="کار باز" to="/work" divider />
        <StatCell value={stats.activeCases} label="پرونده فعال" to="/cases" divider />
      </div>

      {/* ---------------------------------------------------------------- */}
      {attention.length > 0 && (
        <Section>
          <SectionHeader
            title="نیازمند توجه شما"
            hint={attention.length > 1 ? `${fa(attention.length)} مورد برای رسیدگی` : undefined}
          />
          <AttentionList items={attention} />
        </Section>
      )}

      {/* ---------------------------------------------------------------- */}
      <Section>
        <SectionHeader title="برنامه امروز" action={<SectionAction to="/calendar">تقویم</SectionAction>} />
        <AgendaTimeline items={agenda} />
      </Section>

      {/* ---------------------------------------------------------------- */}
      <Section>
        <SectionHeader title="کارهای پیش‌رو" action={<SectionAction to="/work">همه کارها</SectionAction>} />
        {upcomingTasks.length === 0 ? (
          <div className="rounded-lg border border-dashed border-line-strong bg-surface/60">
            <EmptyState
              icon={<Inbox className="size-6" />}
              title="کاری در پیش ندارید"
              description="همه کارهای با مهلت نزدیک انجام شده‌اند."
            />
          </div>
        ) : (
          <ListGroup>
            {upcomingTasks.map((t, i) => (
              <TaskRow key={t.id} task={t} divider={i < upcomingTasks.length - 1} />
            ))}
          </ListGroup>
        )}
      </Section>

      {/* ---------------------------------------------------------------- */}
      {monitoring.length > 0 && (
        <Section>
          <SectionHeader title="پایش کودکان" action={<SectionAction to="/monitoring">همه موارد</SectionAction>} />
          <ListGroup>
            {monitoring.map((child, i) => (
              <ChildFollowUpRow key={child.id} child={child} divider={i < monitoring.length - 1} />
            ))}
          </ListGroup>
        </Section>
      )}

      {/* ---------------------------------------------------------------- */}
      <p className="mt-8 flex items-start gap-2 px-1 pb-8 text-2xs leading-relaxed text-ink-3">
        <ShieldCheck className="mt-0.5 size-4 shrink-0 text-brand" strokeWidth={2.25} aria-hidden />
        دسترسی شما به پرونده‌ها بر اساس مسئولیت محوله محدود است. تصمیم‌گیری درباره صلاحیت، جایگذاری
        و امور حقوقی در صلاحیت مراجع ذی‌صلاح است.
      </p>
    </Screen>
  );
}

function StatCell({
  value,
  label,
  to,
  divider,
}: {
  value: number;
  label: string;
  to: string;
  divider?: boolean;
}) {
  return (
    <Link
      to={to}
      className={`press flex flex-col items-center gap-0.5 px-2 py-3 hover:bg-sunken ${divider ? 'border-s border-line-soft' : ''}`}
    >
      <span className="text-xl font-extrabold text-ink tabular-nums">{fa(value)}</span>
      <span className="text-2xs font-medium text-ink-3">{label}</span>
    </Link>
  );
}
