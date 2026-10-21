import { NavLink, useLocation } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { CalendarDays, FolderHeart, ListChecks, Menu, Sunrise } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useApp } from '@/store/app-store';
import { isOpenTask, taskBucket } from '@/lib/domain';

/**
 * Primary navigation.
 *
 * Five destinations, no more: Today, Work, Calendar, Cases, More. Everything
 * else (reports, monitoring, notifications, profile) is reachable in one tap
 * from "More" and contextually from the screens where it matters. This keeps
 * the bar legible at a glance and the thumb's travel short.
 */
interface TabDef {
  to: string;
  label: string;
  icon: LucideIcon;
  exact?: boolean;
}

const TABS: TabDef[] = [
  { to: '/', label: 'امروز', icon: Sunrise, exact: true },
  { to: '/work', label: 'کارها', icon: ListChecks },
  { to: '/calendar', label: 'تقویم', icon: CalendarDays },
  { to: '/cases', label: 'پرونده‌ها', icon: FolderHeart },
  { to: '/more', label: 'بیشتر', icon: Menu },
]

export function TabBar() {
  const { pathname } = useLocation();
  const { tasks } = useApp();
  const urgent = tasks.filter((t) => isOpenTask(t) && ['overdue', 'today'].includes(taskBucket(t))).length;

  return (
    <nav
      aria-label="ناوبری اصلی"
      className="relative z-30 shrink-0 border-t border-line bg-surface/95 backdrop-blur-xl"
      style={{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }}
    >
      <ul className="flex items-stretch" style={{ height: 'var(--tabbar-h)' }}>
        {TABS.map((tab) => {
          const active = tab.exact ? pathname === tab.to : pathname.startsWith(tab.to);
          const Icon = tab.icon;
          return (
            <li key={tab.to} className="flex-1">
              <NavLink
                to={tab.to}
                aria-current={active ? 'page' : undefined}
                className="group flex h-full w-full flex-col items-center justify-center gap-1"
              >
                <span
                  className={cn(
                    'relative grid h-7 w-12 place-items-center rounded-pill transition-colors duration-base',
                    active ? 'bg-brand-soft text-brand-ink' : 'text-ink-3 group-hover:text-ink-2',
                  )}
                >
                  <Icon className="size-[20px]" strokeWidth={active ? 2.4 : 2} aria-hidden />
                  {tab.to === '/work' && urgent > 0 && (
                    <span className="absolute end-1.5 top-0.5 size-1.5 rounded-full bg-danger" />
                  )}
                </span>
                <span
                  className={cn(
                    'text-2xs leading-none transition-colors',
                    active ? 'font-bold text-brand-ink' : 'font-medium text-ink-3',
                  )}
                >
                  {tab.label}
                </span>
              </NavLink>
            </li>
          );
        })}
      </ul>
    </nav>
  );
}
