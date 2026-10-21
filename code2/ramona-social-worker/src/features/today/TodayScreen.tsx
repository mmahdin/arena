import { useMemo, useState } from 'react';
import { Link, useNavigate } from 'react-router-dom';
import { Bell, EyeOff } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { buildToday } from '@/services/dashboard';
import { Screen, Section } from '@/components/layout/Screen';
import { SectionAction, SectionHeader } from '@/components/ui/SectionHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Avatar } from '@/components/ui/Avatar';
import { Skeleton, RowSkeleton } from '@/components/ui/Skeleton';
import { AgendaTimeline } from '@/features/_shared/AgendaTimeline';
import { AttentionList } from '@/features/_shared/AttentionList';
import { fa } from '@/lib/digits';

/** Today is a focused brief: what must be addressed, and today's agenda. */
export function TodayScreen() {
  const { tasks, visits, reports, notifications, user, status, unreadCount } = useApp();
  const { fieldPrivacy, toggleFieldPrivacy } = useSettings();
  const navigate = useNavigate();
  const [showAllAttention, setShowAllAttention] = useState(false);

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
        </div>
        <div className="mt-6"><RowSkeleton /><RowSkeleton /><RowSkeleton /></div>
      </Screen>
    );
  }

  const visibleAttention = showAllAttention ? today.attention : today.attention.slice(0, 3);
  const moreAttention = today.attention.length - visibleAttention.length;

  return (
    <Screen bottomGap={false}>
      <header className="sticky top-0 z-30 -mx-gutter bg-bg/90 px-gutter pb-3 pt-4 backdrop-blur-xl">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <p className="text-xs font-medium text-ink-3">{today.dateLabel}</p>
            <h1 className="mt-0.5 truncate text-xl font-extrabold tracking-tight text-ink">{user.name}</h1>
          </div>
          <div className="flex shrink-0 items-center gap-1">
            {fieldPrivacy && (
              <button
                type="button"
                onClick={toggleFieldPrivacy}
                className="press flex h-10 items-center gap-1 rounded-md bg-accent-soft px-2.5 text-xs font-bold text-accent-ink"
                aria-label="خاموش کردن حالت محرمانگی"
              >
                <EyeOff className="size-4" strokeWidth={2.3} aria-hidden />
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

      {today.attention.length > 0 && (
        <Section>
          <SectionHeader
            title="نیازمند رسیدگی"
            hint={`${fa(today.attention.length)} اقدام باقی‌مانده`}
          />
          <AttentionList items={visibleAttention} />
          {today.attention.length > 3 && (
            <button
              type="button"
              onClick={() => setShowAllAttention((v) => !v)}
              className="press mt-2 w-full rounded-md py-2 text-center text-sm font-semibold text-brand-ink"
              aria-expanded={showAllAttention}
            >
              {showAllAttention ? 'نمایش کمتر' : `نمایش ${fa(moreAttention)} مورد دیگر`}
            </button>
          )}
        </Section>
      )}

      <Section className="pb-8">
        <SectionHeader title="برنامه امروز" action={<SectionAction to="/calendar">تقویم کاری</SectionAction>} />
        <AgendaTimeline items={today.agenda} />
      </Section>
    </Screen>
  );
}
