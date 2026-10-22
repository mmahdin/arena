import { NavLink, useLocation } from 'react-router-dom';
import type { LucideIcon } from 'lucide-react';
import { ClipboardList, FileText, House } from 'lucide-react';
import { cn } from '@/lib/cn';
import { useApp } from '@/store/app-store';

/** Each primary destination has a single, distinct responsibility. */
interface TabDef {
  to: string;
  label: string;
  icon: LucideIcon;
}

const TABS: TabDef[] = [
  { to: '/visits', label: 'بازدیدها', icon: House },
  { to: '/cases', label: 'خانواده‌ها', icon: ClipboardList },
  { to: '/reports', label: 'گزارش‌ها', icon: FileText },
];

export function TabBar() {
  const { pathname } = useLocation();
  const { reports } = useApp();
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
