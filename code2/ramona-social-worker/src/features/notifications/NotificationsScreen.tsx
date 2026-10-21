import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BellOff, CheckCheck } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile, NOTIFICATION_ICON } from '@/components/ui/DomainIcon';
import { NOTIFICATION_TYPE } from '@/lib/labels';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { dayKey, faDate, isSameDay, relativeTime } from '@/lib/jalali';

/**
 * Notifications.
 *
 * Grouped by day and ordered newest-first, with unread items carrying a dot
 * *and* a filled background — again, two channels rather than colour alone.
 * Tapping a notification marks it read and goes straight to the thing it is
 * about; there is no "open notification" intermediate screen.
 */
export function NotificationsScreen() {
  const { notifications, unreadCount, markNotificationRead, markAllNotificationsRead, toast } = useApp();
  const navigate = useNavigate();
  const [scope, setScope] = useState<'all' | 'unread'>('all');

  const visible = useMemo(() => {
    const rows = scope === 'unread' ? notifications.filter((n) => !n.read) : notifications;
    return rows.slice().sort((a, b) => b.at.localeCompare(a.at));
  }, [notifications, scope]);

  const groups = useMemo(() => {
    const now = new Date();
    const map = new Map<string, typeof visible>();
    for (const n of visible) {
      const d = new Date(n.at);
      const key = isSameDay(d, now) ? 'today' : isSameDay(d, new Date(now.getTime() - 86400000)) ? 'yesterday' : dayKey(d);
      const list = map.get(key) ?? [];
      list.push(n);
      map.set(key, list);
    }
    return [...map.entries()];
  }, [visible]);

  const groupLabel = (key: string) =>
    key === 'today' ? 'امروز' : key === 'yesterday' ? 'دیروز' : faDate(key, { weekday: true, year: false });

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="اعلان‌ها"
        subtitle={unreadCount > 0 ? `${fa(unreadCount)} اعلان خوانده‌نشده` : 'همه اعلان‌ها خوانده شده‌اند'}
        actions={
          unreadCount > 0 ? (
            <IconButton
              label="خواندن همه اعلان‌ها"
              onClick={() => {
                markAllNotificationsRead();
                toast({ title: 'همه اعلان‌ها خوانده شد', tone: 'success' });
              }}
            >
              <CheckCheck className="size-[20px]" strokeWidth={2.1} />
            </IconButton>
          ) : undefined
        }
      >
        <div className="mt-3">
          <SegmentedControl<'all' | 'unread'>
            value={scope}
            onChange={setScope}
            ariaLabel="محدوده اعلان‌ها"
            items={[
              { value: 'all', label: 'همه', count: notifications.length },
              { value: 'unread', label: 'خوانده‌نشده', count: unreadCount },
            ]}
          />
        </div>
      </ScreenHeader>

      <div className="pt-4 pb-8">
        {visible.length === 0 ? (
          <EmptyState
            icon={<BellOff className="size-6" />}
            title="اعلانی وجود ندارد"
            description={scope === 'unread' ? 'همه اعلان‌ها خوانده شده‌اند.' : 'رویداد جدیدی برای شما ثبت نشده است.'}
          />
        ) : (
          groups.map(([key, items]) => (
            <section key={key} className="mt-5 first:mt-0">
              <h2 className="mb-2 px-1 text-sm font-bold text-ink-2">{groupLabel(key)}</h2>
              <ul className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
                {items.map((n, i) => {
                  const meta = NOTIFICATION_TYPE[n.type];
                  const Icon = NOTIFICATION_ICON[n.type];
                  return (
                    <li key={n.id}>
                      <button
                        type="button"
                        onClick={() => {
                          if (!n.read) markNotificationRead(n.id);
                          if (n.href) navigate(n.href);
                        }}
                        className={cn(
                          'press flex w-full items-start gap-3 px-4 py-3.5 text-start hover:bg-sunken',
                          i > 0 && 'border-t border-line-soft',
                          !n.read && 'bg-brand-soft/40',
                        )}
                      >
                        <IconTile icon={Icon} tone={meta.tone} size="sm" />
                        <span className="min-w-0 flex-1">
                          <span className="flex items-center gap-2">
                            <span className={cn('truncate text-base', !n.read ? 'font-bold text-ink' : 'font-semibold text-ink-2')}>
                              {n.title}
                            </span>
                            {!n.read && <span className="size-2 shrink-0 rounded-full bg-brand" aria-label="خوانده‌نشده" />}
                          </span>
                          <span className="mt-1 block text-sm leading-relaxed text-ink-2 clamp-2">{n.body}</span>
                          <span className="mt-1.5 flex items-center gap-2 text-2xs text-ink-3">
                            <span>{meta.label}</span>
                            <span aria-hidden className="size-1 rounded-full bg-line-strong" />
                            <span>{relativeTime(n.at)}</span>
                          </span>
                        </span>
                      </button>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))
        )}
      </div>
    </Screen>
  );
}
