import { NavLink, useLocation } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { CalendarDays, ClipboardList, FileText, Inbox } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useApp } from '@/store/app-store';

/**
 * Primary navigation.
 *
 * Four destinations, each with one job: the personal programme, the open pool,
 * the families in the worker's care, and the report queue. The pool earns a tab
 * because accepting visits is a daily, high-frequency activity — burying it
 * one level down would cost a tap on every shift.
 */
interface TabDef {
  to: string;
  label: string;
  icon: LucideIcon;
}

const TABS: TabDef[] = [
  { to: '/visits', label: 'بازدیدها', icon: CalendarDays },
  { to: '/pool', label: 'بازدید آزاد', icon: Inbox },
  { to: '/cases', label: 'خانواده‌ها', icon: ClipboardList },
  { to: '/reports', label: 'گزارش‌ها', icon: FileText },
];

export function TabBar() {
  const { pathname } = useLocation();
  const { reports, poolVisits } = useApp();
  const revisions = reports.filter((r) => r.status === 'revision').length;
  const pool = poolVisits.length;

  return (
    <nav
      aria-label="ناوبری اصلی"
      className="relative z-30 shrink-0 border-t border-line bg-surface/95 backdrop-blur-xl"
      style={{ paddingBottom: 'env(safe-area-inset-bottom, 0px)' }}
    >
      <ul className="flex items-stretch" style={{ height: 'var(--tabbar-h)' }}>
        {TABS.map((tab) => {
          const active = pathname === tab.to || pathname.startsWith(`${tab.to}/`);
          const Icon = tab.icon;
          const badge = tab.to === '/pool' && pool > 0 ? pool : undefined;
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
                  {typeof badge === 'number' && (
                    <span
                      className="absolute -end-1 -top-1.5 grid h-4 min-w-4 place-items-center rounded-pill bg-brand px-1 text-[9px] font-bold leading-none text-white ring-2 ring-surface tabular-nums"
                      aria-label={`${badge.toLocaleString('fa-IR')} بازدید آزاد`}
                    >
                      {badge.toLocaleString('fa-IR')}
                    </span>
                  )}
                  {tab.to === '/reports' && revisions > 0 && (
                    <span className="absolute end-1 top-0.5 size-1.5 rounded-full bg-danger" aria-label="گزارش نیازمند اصلاح" />
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
