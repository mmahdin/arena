import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { Bell, BellOff, CheckCheck } from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import {
  diffDays,
  formatDate,
  formatRelative,
  isSameDay,
  parse,
} from "@/lib/date";
import { notificationKindLabels } from "@/lib/labels";
import type { AppNotification, NotificationKind } from "@/types";
import { cn } from "@/lib/cn";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { FilterChips } from "@/components/ui/FilterChips";
import { EmptyState } from "@/components/ui/EmptyState";
import { NotificationIcon } from "@/components/domain/Badges";
import { unreadNotifications } from "@/lib/selectors";
import "./NotificationsPage.css";

type Filter = "all" | "unread";

const kindTones: Record<NotificationKind, string> = {
  assignment: "brand",
  schedule_change: "info",
  report_revision: "warning",
  report_approved: "success",
  message: "accent",
  system: "neutral",
};

function groupLabel(notification: AppNotification, now: Date): string {
  const date = parse(notification.createdAt);
  if (isSameDay(date, now)) return "امروز";
  const delta = diffDays(now, date);
  if (delta === 1) return "دیروز";
  if (delta < 7) return "این هفته";
  return formatDate(date);
}

export function NotificationsPage() {
  const data = useWorkspace();
  const now = useNow();
  const actions = useAppActions();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>("all");

  const unread = unreadNotifications(data);
  const items = useMemo(
    () =>
      (filter === "unread" ? data.notifications.filter((n) => !n.read) : data.notifications).slice().sort(
        (a, b) => parse(b.createdAt).getTime() - parse(a.createdAt).getTime()
      ),
    [data.notifications, filter]
  );

  const groups = useMemo(() => {
    const map = new Map<string, AppNotification[]>();
    items.forEach((item) => {
      const label = groupLabel(item, now);
      if (!map.has(label)) map.set(label, []);
      map.get(label)!.push(item);
    });
    return [...map.entries()];
  }, [items, now]);

  const open = (notification: AppNotification) => {
    actions.markNotificationRead(notification.id);
    if (notification.href) navigate(notification.href);
  };

  return (
    <>
      <TopBar
        title="اعلان‌ها"
        actions={
          unread > 0 ? (
            <button
              type="button"
              className="notif-read-all"
              onClick={() => actions.markAllNotificationsRead()}
            >
              <CheckCheck size={14} aria-hidden="true" />
              خواندن همه
            </button>
          ) : null
        }
      />

      <Page>
        <div className="notif-summary">
          <span className="notif-summary__icon" aria-hidden="true">
            <Bell size={17} />
          </span>
          <div>
            <p className="notif-summary__title">
              {unread > 0
                ? `${new Intl.NumberFormat("fa-IR").format(unread)} اعلان خوانده‌نشده`
                : "همه اعلان‌ها خوانده شده‌اند"}
            </p>
            <p className="notif-summary__text">
              تغییر برنامه بازدیدها، اصلاح گزارش و ارجاع‌های جدید در این بخش ثبت می‌شود.
            </p>
          </div>
        </div>

        <FilterChips<Filter>
          ariaLabel="فیلتر اعلان‌ها"
          items={[
            { value: "all", label: "همه", count: data.notifications.length },
            { value: "unread", label: "خوانده‌نشده", count: unread },
          ]}
          value={filter}
          onChange={setFilter}
        />

        {items.length === 0 ? (
          <EmptyState
            icon={filter === "unread" ? <BellOff size={22} /> : <Bell size={22} />}
            title={filter === "unread" ? "اعلان خوانده‌نشده‌ای ندارید" : "اعلانی وجود ندارد"}
            description={
              filter === "unread"
                ? "همه اعلان‌ها را خوانده‌اید. اعلان‌های جدید در همین صفحه نمایش داده می‌شود."
                : "با ارجاع پرونده جدید یا تغییر برنامه، اعلان‌ها اینجا ظاهر می‌شوند."
            }
            compact
          />
        ) : (
          <div className="notif-groups">
            {groups.map(([label, groupItems]) => (
              <section className="notif-group" key={label} aria-label={label}>
                <h2 className="notif-group__label">{label}</h2>
                <ul className="notif-list">
                  {groupItems.map((notification) => (
                    <li key={notification.id}>
                      <button
                        type="button"
                        className={cn("notif-item", !notification.read && "is-unread")}
                        onClick={() => open(notification)}
                      >
                        <span
                          className={cn(
                            "notif-item__icon",
                            `is-${kindTones[notification.kind]}`
                          )}
                          aria-hidden="true"
                        >
                          <NotificationIcon kind={notification.kind} />
                        </span>
                        <span className="notif-item__body">
                          <span className="notif-item__top">
                            <span className="notif-item__kind">
                              {notificationKindLabels[notification.kind]}
                            </span>
                            <span className="notif-item__time">
                              {formatRelative(parse(notification.createdAt), now)}
                            </span>
                          </span>
                          <span className="notif-item__title">{notification.title}</span>
                          <span className="notif-item__text">{notification.body}</span>
                        </span>
                        {!notification.read ? (
                          <span className="notif-item__dot" aria-label="خوانده‌نشده" />
                        ) : null}
                      </button>
                    </li>
                  ))}
                </ul>
              </section>
            ))}
          </div>
        )}

        <p className="notif-footnote">
          تنظیمات اعلان‌ها را می‌توانید از بخش پروفایل تغییر دهید.
        </p>
      </Page>
    </>
  );
}
