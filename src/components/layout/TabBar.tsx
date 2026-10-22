import { NavLink, useLocation } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { CalendarDays, FileText, Inbox, Users } from 'lucide-react';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { useApp } from '@/store/app-store';

/**
 * Primary navigation = the social worker's four recurring activities.
 * Each destination owns exactly one of them, and nothing is reachable
 * through two tabs:
 *
 * - «بازدیدهای من»   — my schedule and every visit I am responsible for
 * - «بازدیدهای آزاد» — unassigned visits waiting to be claimed
 * - «خانواده‌ها»     — the families in my caseload and their records
 * - «گزارش‌ها»       — my report queue (write, send, follow up review)
 */
interface TabDef {
  to: string;
  label: string;
  icon: LucideIcon;
}

const TABS: TabDef[] = [
  { to: '/visits', label: 'بازدیدهای من', icon: CalendarDays },
  { to: '/available', label: 'بازدیدهای آزاد', icon: Inbox },
  { to: '/cases', label: 'خانواده‌ها', icon: Users },
  { to: '/reports', label: 'گزارش‌ها', icon: FileText },
];

export function TabBar() {
  const { pathname } = useLocation();
  const { reports, poolVisits } = useApp();
  const revisions = reports.filter((r) => r.status === 'revision').length;

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
          return (
            <li key={tab.to} className="min-w-0 flex-1">
              <NavLink
                to={tab.to}
                aria-current={active ? 'page' : undefined}
                className="group flex h-full w-full flex-col items-center justify-center gap-1 px-0.5"
              >
                <span
                  className={cn(
                    'relative grid h-7 w-12 place-items-center rounded-pill transition-colors duration-base',
                    active ? 'bg-brand-soft text-brand-ink' : 'text-ink-3 group-hover:text-ink-2',
                  )}
                >
                  <Icon className="size-[20px]" strokeWidth={active ? 2.4 : 2} aria-hidden />
                  {tab.to === '/available' && poolVisits.length > 0 && (
                    <span
                      className="absolute -top-1 end-0 grid h-4 min-w-4 place-items-center rounded-pill bg-brand px-1 text-[10px] font-bold leading-none text-white tabular-nums"
                      aria-label={`${fa(poolVisits.length)} بازدید آزاد`}
                    >
                      {fa(poolVisits.length)}
                    </span>
                  )}
                  {tab.to === '/reports' && revisions > 0 && (
                    <span className="absolute end-1 top-0.5 size-1.5 rounded-full bg-danger" aria-label="گزارش نیازمند اصلاح" />
                  )}
                </span>
                <span
                  className={cn(
                    'max-w-full truncate text-2xs leading-none transition-colors',
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
