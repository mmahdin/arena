import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { BellOff, CheckCheck, ChevronDown, ChevronLeft } from 'lucide-react';
import type { AppNotification } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile, NOTIFICATION_ICON } from '@/components/ui/DomainIcon';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { relativeTime } from '@/lib/jalali';

/**
 * A quiet inbox: one-line previews, with details expanding in the list.
 * No overlay or second page is needed to read a short notification.
 */
export function NotificationsScreen() {
  const { notifications, unreadCount, markNotificationRead, markAllNotificationsRead, toast } = useApp();
  const navigate = useNavigate();
  const [scope, setScope] = useState<'unread' | 'all'>('unread');
  const [expandedId, setExpandedId] = useState<string | null>(null);

  const visible = useMemo(() => {
    // Keep an opened notification in place after it is marked read. It leaves
    // the unread view once the user collapses it or opens another item.
    const rows = scope === 'unread'
      ? notifications.filter((notification) => !notification.read || notification.id === expandedId)
      : notifications;
    return rows.slice().sort((a, b) => b.at.localeCompare(a.at));
  }, [notifications, scope, expandedId]);

  const toggleNotification = (notification: AppNotification) => {
    if (expandedId === notification.id) {
      setExpandedId(null);
      return;
    }
    setExpandedId(notification.id);
    if (!notification.read) markNotificationRead(notification.id);
  };

  const navigateToRelated = (href: string) => {
    setExpandedId(null);
    navigate(href);
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="اعلان‌ها"
        back
        actions={
          unreadCount > 0 ? (
            <button
              type="button"
              onClick={() => {
                setExpandedId(null);
                markAllNotificationsRead();
                toast({ title: 'همه اعلان‌ها خوانده شد', tone: 'success' });
              }}
              className="press inline-flex min-h-10 items-center gap-1.5 rounded-md px-2 text-xs font-semibold text-ink-2 hover:bg-sunken hover:text-ink"
              aria-label="علامت‌گذاری همه اعلان‌ها به‌عنوان خوانده‌شده"
            >
              <CheckCheck className="size-4" aria-hidden />
              خواندن همه
            </button>
          ) : undefined
        }
      />

      <div className="pb-8 pt-4">
        <div className="mb-2 flex items-center justify-between gap-3">
          <h2 className="text-sm font-bold text-ink-2">
            {scope === 'unread' ? `خوانده‌نشده (${fa(unreadCount)})` : 'همه اعلان‌ها'}
          </h2>
          <button
            type="button"
            className="press inline-flex min-h-10 items-center gap-1 rounded-md px-1 text-sm font-semibold text-brand-ink hover:bg-brand-soft"
            onClick={() => {
              setExpandedId(null);
              setScope((current) => (current === 'unread' ? 'all' : 'unread'));
            }}
          >
            {scope === 'unread' ? 'همه اعلان‌ها' : 'فقط خوانده‌نشده‌ها'}
            <ChevronLeft className="size-4" aria-hidden />
          </button>
        </div>

        {visible.length === 0 ? (
          <EmptyState
            icon={<BellOff className="size-6" />}
            title={scope === 'unread' ? 'اعلان خوانده‌نشده‌ای ندارید' : 'اعلانی وجود ندارد'}
          />
        ) : (
          <ul className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
            {visible.map((notification, index) => {
              const Icon = NOTIFICATION_ICON[notification.type];
              const expanded = expandedId === notification.id;
              const detailId = `notification-detail-${notification.id}`;
              return (
                <li key={notification.id} className={cn(index > 0 && 'border-t border-line-soft')}>
                  <button
                    type="button"
                    onClick={() => toggleNotification(notification)}
                    className="press flex min-h-[72px] w-full items-center gap-3 px-3.5 py-3 text-start hover:bg-sunken/50"
                    aria-expanded={expanded}
                    aria-controls={expanded ? detailId : undefined}
                  >
                    <IconTile icon={Icon} tone="neutral" size="sm" />
                    <span className="min-w-0 flex-1">
                      <span className={cn('block clamp-2 text-sm leading-6', notification.read ? 'font-medium text-ink-2' : 'font-bold text-ink')}>
                        {notification.title}
                      </span>
                      <span className="mt-0.5 block text-xs text-ink-3">{relativeTime(notification.at)}</span>
                    </span>
                    {scope === 'all' && !notification.read && (
                      <span className="size-2 shrink-0 rounded-full bg-brand" aria-hidden />
                    )}
                    <ChevronDown
                      className={cn('size-4 shrink-0 text-ink-3 transition-transform duration-200', expanded && 'rotate-180')}
                      aria-hidden
                    />
                  </button>

                  {expanded && (
                    <div id={detailId} className="animate-fade-in mx-4 border-t border-line-soft pb-3 pt-3">
                      <p className="whitespace-pre-line text-sm leading-7 text-ink-2">{notification.body}</p>
                      {notification.href && (
                        <button
                          type="button"
                          onClick={() => notification.href && navigateToRelated(notification.href)}
                          className="press mt-2 inline-flex min-h-10 items-center gap-1.5 rounded-sm px-1 text-sm font-semibold text-brand-ink hover:bg-brand-soft"
                        >
                          {notification.href.startsWith('/reports/')
                            ? 'مشاهده گزارش'
                            : notification.href.startsWith('/visits/')
                              ? 'مشاهده بازدید'
                              : notification.href.startsWith('/cases/')
                                ? 'مشاهده پرونده'
                                : 'مشاهده جزئیات'}
                          <ChevronLeft className="size-4" aria-hidden />
                        </button>
                      )}
                    </div>
                  )}
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </Screen>
  );
}
