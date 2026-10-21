import { useEffect } from "react";
import { HashRouter, Route, Routes, useLocation } from "react-router-dom";
import { AppDataProvider } from "@/app/store";
import { ToastProvider } from "@/components/ui/Toast";
import { AppShell } from "@/components/app/AppShell";
import { HomePage } from "@/features/home/HomePage";
import { VisitsPage } from "@/features/visits/VisitsPage";
import { VisitDetailPage } from "@/features/visits/VisitDetailPage";
import { TasksPage } from "@/features/tasks/TasksPage";
import { TaskDetailPage } from "@/features/tasks/TaskDetailPage";
import { ReportsPage } from "@/features/reports/ReportsPage";
import { ReportDetailPage } from "@/features/reports/ReportDetailPage";
import { ReportEditorPage } from "@/features/reports/ReportEditorPage";
import { CasesPage } from "@/features/cases/CasesPage";
import { CaseDetailPage } from "@/features/cases/CaseDetailPage";
import { CalendarPage } from "@/features/calendar/CalendarPage";
import { NotificationsPage } from "@/features/notifications/NotificationsPage";
import { ProfilePage } from "@/features/profile/ProfilePage";
import { DesignSystemPage } from "@/features/design/DesignSystemPage";
import { NotFoundPage } from "@/features/NotFoundPage";

/**
 * Hash routing keeps the built SPA portable (static hosting, file previews)
 * without needing a server-side history fallback.
 */
export function App() {
  return (
    <AppDataProvider>
      <ToastProvider>
        <HashRouter>
          <RouteEffects />
          <AppShell>
            <Routes>
              <Route path="/" element={<HomePage />} />
              <Route path="/visits" element={<VisitsPage />} />
              <Route path="/visits/:visitId" element={<VisitDetailPage />} />
              <Route path="/tasks" element={<TasksPage />} />
              <Route path="/tasks/:taskId" element={<TaskDetailPage />} />
              <Route path="/reports" element={<ReportsPage />} />
              <Route path="/reports/new" element={<ReportEditorPage mode="create" />} />
              <Route path="/reports/:reportId" element={<ReportDetailPage />} />
              <Route path="/reports/:reportId/edit" element={<ReportEditorPage mode="edit" />} />
              <Route path="/cases" element={<CasesPage />} />
              <Route path="/cases/:caseId" element={<CaseDetailPage />} />
              <Route path="/calendar" element={<CalendarPage />} />
              <Route path="/notifications" element={<NotificationsPage />} />
              <Route path="/profile" element={<ProfilePage />} />
              <Route path="/design" element={<DesignSystemPage />} />
              <Route path="*" element={<NotFoundPage />} />
            </Routes>
          </AppShell>
        </HashRouter>
      </ToastProvider>
    </AppDataProvider>
  );
}

function RouteEffects() {
  const location = useLocation();

  useEffect(() => {
    window.scrollTo({ top: 0, behavior: "instant" as ScrollBehavior });
  }, [location.pathname]);

  return null;
}
