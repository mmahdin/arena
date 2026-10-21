import { useEffect, useRef } from 'react';
import { Navigate, Route, Routes, useLocation } from 'react-router-dom';
import { Moon, Sun } from 'lucide-react';
import { TabBar } from '@/components/layout/TabBar';
import { Logo } from '@/components/layout/Logo';
import { ToastHost } from '@/components/ui/Toasts';
import { OVERLAY_ROOT_ID } from '@/components/ui/Sheet';
import { useSettings } from '@/store/settings';

import { TodayScreen } from '@/features/today/TodayScreen';
import { CalendarScreen } from '@/features/calendar/CalendarScreen';
import { CasesScreen } from '@/features/cases/CasesScreen';
import { CaseDetailScreen } from '@/features/cases/CaseDetailScreen';
import { ChildDetailScreen } from '@/features/cases/ChildDetailScreen';
import { VisitDetailScreen } from '@/features/visits/VisitDetailScreen';
import { ReportsScreen } from '@/features/reports/ReportsScreen';
import { ReportEditorScreen } from '@/features/reports/ReportEditorScreen';
import { MonitoringScreen } from '@/features/monitoring/MonitoringScreen';
import { NotificationsScreen } from '@/features/notifications/NotificationsScreen';
import { MoreScreen } from '@/features/more/MoreScreen';
import { ProfileScreen } from '@/features/profile/ProfileScreen';
import { SettingsScreen } from '@/features/profile/SettingsScreen';
import { SearchScreen } from '@/features/search/SearchScreen';
import { DesignSystemScreen } from '@/features/design-system/DesignSystemScreen';
import { NotFoundScreen } from '@/features/misc/NotFoundScreen';

/** Destinations that own a tab in the primary navigation. */
const TAB_ROUTES = ['/', '/calendar', '/cases', '/reports', '/more'];

export function AppShell() {
  const { pathname, search } = useLocation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const { resolvedTheme, setTheme } = useSettings();

  // Every navigation starts at the top of the new screen.
  useEffect(() => {
    scrollRef.current?.scrollTo({ top: 0, behavior: 'auto' });
  }, [pathname, search]);

  const showTabs = TAB_ROUTES.includes(pathname);

  return (
    <div className="stage h-full">
      <div className="device">
        <div className="device-viewport">
          <div className="relative flex h-full flex-col">
            <div id="app-scroll" ref={scrollRef} className="app-scroll">
              <Routes>
                <Route path="/" element={<TodayScreen />} />
                <Route path="/work" element={<Navigate to="/calendar" replace />} />
                <Route path="/calendar" element={<CalendarScreen />} />
                <Route path="/cases" element={<CasesScreen />} />
                <Route path="/cases/:caseId" element={<CaseDetailScreen />} />
                <Route path="/children/:childId" element={<ChildDetailScreen />} />
                <Route path="/visits/:visitId" element={<VisitDetailScreen />} />
                <Route path="/reports" element={<ReportsScreen />} />
                <Route path="/reports/:reportId" element={<ReportEditorScreen />} />
                <Route path="/monitoring" element={<MonitoringScreen />} />
                <Route path="/notifications" element={<NotificationsScreen />} />
                <Route path="/more" element={<MoreScreen />} />
                <Route path="/profile" element={<ProfileScreen />} />
                <Route path="/settings" element={<SettingsScreen />} />
                <Route path="/search" element={<SearchScreen />} />
                <Route path="/design-system" element={<DesignSystemScreen />} />
                <Route path="*" element={<NotFoundScreen />} />
              </Routes>
            </div>

            {showTabs && <TabBar />}

            {/* Overlays (sheets, toasts) live inside the viewport, never on
                document.body, so they stay inside the device frame. */}
            <div id={OVERLAY_ROOT_ID} className="pointer-events-none absolute inset-0 z-50" />
            <ToastHost />
          </div>
        </div>
      </div>

      {/* Desktop-only chrome, outside the phone. */}
      <div className="pointer-events-none fixed inset-x-0 top-0 z-40 hidden items-center justify-between px-8 py-6 lg:flex">
        <div className="pointer-events-auto flex items-center gap-3 text-ink">
          <span className="grid size-9 place-items-center rounded-md bg-brand text-white shadow-sm">
            <Logo size={20} />
          </span>
          <span className="leading-tight">
            <span className="block text-sm font-extrabold">رمونا — فضای کاری مددکار اجتماعی</span>
            <span className="block text-xs text-ink-3">نمونه طراحی · تجربه موبایل‌اول و راست‌به‌چپ</span>
          </span>
        </div>
        <button
          type="button"
          onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}
          className="press pointer-events-auto grid size-10 place-items-center rounded-md border border-line bg-surface text-ink-2 shadow-sm hover:text-ink"
          aria-label={resolvedTheme === 'dark' ? 'حالت روشن' : 'حالت تیره'}
        >
          {resolvedTheme === 'dark' ? <Sun className="size-[18px]" /> : <Moon className="size-[18px]" />}
        </button>
      </div>
    </div>
  );
}
