import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ReactNode,
} from 'react';
import type {
  AppNotification,
  CaseFile,
  Report,
  ReportField,
  ReportType,
  Task,
  TaskStatus,
  Tone,
  UserProfile,
  Visit,
  VisitStatus,
} from '@/types';
import * as api from '@/services';
import { uid } from '@/services/_utils';

/**
 * Application store.
 *
 * One provider owns the loaded data and every mutation ("intent"). Screens
 * read from it and call intents; they never touch `src/services` or
 * `src/data` directly. That keeps the swap to a real backend contained and
 * makes optimistic UI updates trivial to add.
 */

export interface ToastMessage {
  id: string;
  title: string;
  description?: string;
  tone?: Tone;
  action?: { label: string; onClick: () => void };
}

interface AppData {
  status: 'loading' | 'ready';
  user: UserProfile | null;
  cases: CaseFile[];
  visits: Visit[];
  /** Unassigned visits published to the open pool. */
  poolVisits: Visit[];
  tasks: Task[];
  reports: Report[];
  notifications: AppNotification[];
}

interface AppContextValue extends AppData {
  reload: () => Promise<void>;
  /* tasks */
  setTaskStatus: (id: string, status: TaskStatus) => Promise<void>;
  postponeTask: (id: string, days: number) => Promise<void>;
  /* visits */
  toggleChecklistItem: (visitId: string, itemId: string) => void;
  recordVisitOutcome: (visitId: string, status: VisitStatus, note: string) => Promise<{ reportId?: string }>;
  /** Accept an open-pool visit; it becomes part of the worker's programme. */
  claimVisit: (visitId: string) => Promise<boolean>;
  /* reports */
  editReportField: (reportId: string, sectionId: string, fieldId: string, value: string | number | string[]) => void;
  editReportMeta: (reportId: string, patch: { title?: string; summary?: string }) => void;
  submitReport: (reportId: string) => Promise<void>;
  createReportDraft: (input: {
    type: ReportType;
    caseId: string;
    childId?: string;
    visitId?: string;
  }) => Promise<string | undefined>;
  /* notifications */
  markNotificationRead: (id: string) => void;
  markAllNotificationsRead: () => void;
  /* toasts */
  toast: (message: Omit<ToastMessage, 'id'>) => void;
  dismissToast: (id: string) => void;
  toasts: ToastMessage[];
  unreadCount: number;
}

const AppContext = createContext<AppContextValue | null>(null);

const empty: AppData = {
  status: 'loading',
  user: null,
  cases: [],
  visits: [],
  poolVisits: [],
  tasks: [],
  reports: [],
  notifications: [],
};

export function AppProvider({ children }: { children: ReactNode }) {
  const [data, setData] = useState<AppData>(empty);
  const [toasts, setToasts] = useState<ToastMessage[]>([]);
  const timers = useRef<Record<string, number>>({});

  const dismissToast = useCallback((id: string) => {
    setToasts((list) => list.filter((t) => t.id !== id));
    window.clearTimeout(timers.current[id]);
    delete timers.current[id];
  }, []);

  const toast = useCallback(
    (message: Omit<ToastMessage, 'id'>) => {
      const id = uid('toast');
      setToasts((list) => [...list.slice(-2), { ...message, id }]);
      timers.current[id] = window.setTimeout(() => dismissToast(id), 3800);
    },
    [dismissToast],
  );

  const reload = useCallback(async () => {
    const [user, cases, visits, poolVisits, tasks, reports, notifications] = await Promise.all([
      api.getCurrentUser(),
      api.listCases(),
      api.listVisits(),
      api.listPoolVisits(),
      api.listTasks(),
      api.listReports(),
      api.listNotifications(),
    ]);
    setData({ status: 'ready', user, cases, visits, poolVisits, tasks, reports, notifications });
  }, []);

  useEffect(() => {
    void reload();
  }, [reload]);

  /* ---------------------------------------------------------------- *
   * Intents
   * ---------------------------------------------------------------- */

  const setTaskStatus = useCallback(
    async (id: string, status: TaskStatus) => {
      setData((d) => ({ ...d, tasks: d.tasks.map((t) => (t.id === id ? { ...t, status } : t)) }));
      const updated = await api.setTaskStatus(id, status);
      if (updated) setData((d) => ({ ...d, tasks: d.tasks.map((t) => (t.id === id ? updated : t)) }));
    },
    [],
  );

  const postponeTask = useCallback(async (id: string, days: number) => {
    const updated = await api.postponeTask(id, days);
    if (updated) setData((d) => ({ ...d, tasks: d.tasks.map((t) => (t.id === id ? updated : t)) }));
  }, []);

  const toggleChecklistItem = useCallback((visitId: string, itemId: string) => {
    setData((d) => ({
      ...d,
      visits: d.visits.map((v) =>
        v.id === visitId
          ? { ...v, checklist: v.checklist.map((c) => (c.id === itemId ? { ...c, done: !c.done } : c)) }
          : v,
      ),
    }));
    void api.toggleChecklistItem(visitId, itemId);
  }, []);

  const recordVisitOutcome = useCallback(
    async (visitId: string, status: VisitStatus, note: string) => {
      const result = await api.recordVisitOutcome(visitId, { status, note });
      if (!result) return {};
      const [visits, tasks, cases, reports] = await Promise.all([
        api.listVisits(),
        api.listTasks(),
        api.listCases(),
        api.listReports(),
      ]);
      setData((d) => ({ ...d, visits, tasks, cases, reports }));
      return { reportId: result.reportId };
    },
    [],
  );

  const claimVisit = useCallback(async (visitId: string) => {
    const claimed = await api.claimVisit(visitId);
    if (!claimed) return false;
    const [visits, poolVisits, tasks, cases] = await Promise.all([
      api.listVisits(),
      api.listPoolVisits(),
      api.listTasks(),
      api.listCases(),
    ]);
    setData((d) => ({ ...d, visits, poolVisits, tasks, cases }));
    return true;
  }, []);

  const editReportField = useCallback(
    (reportId: string, sectionId: string, fieldId: string, value: string | number | string[]) => {
      setData((d) => ({
        ...d,
        reports: d.reports.map((r) => {
          if (r.id !== reportId) return r;
          const next: Report = {
            ...r,
            updatedAt: new Date().toISOString(),
            sections: r.sections.map((s) =>
              s.id !== sectionId
                ? s
                : { ...s, fields: s.fields.map((f) => (f.id === fieldId ? ({ ...f, value } as ReportField) : f)) },
            ),
          };
          void api.updateField(reportId, sectionId, fieldId, value);
          return next;
        }),
      }));
    },
    [],
  );

  const editReportMeta = useCallback((reportId: string, patch: { title?: string; summary?: string }) => {
    setData((d) => ({
      ...d,
      reports: d.reports.map((r) => (r.id === reportId ? { ...r, ...patch } : r)),
    }));
    if (patch.title !== undefined) api.updateTitle(reportId, patch.title);
    if (patch.summary !== undefined) api.updateSummary(reportId, patch.summary);
  }, []);

  const createReportDraft = useCallback(
    async (input: { type: ReportType; caseId: string; childId?: string; visitId?: string }) => {
      const report = await api.createDraftReport(input);
      const [reports, visits] = await Promise.all([api.listReports(), api.listVisits()]);
      setData((d) => ({ ...d, reports, visits }));
      return report.id;
    },
    [],
  );

  const submitReport = useCallback(async (reportId: string) => {
    await api.setReportStatus(reportId, 'in-review');
    const reports = await api.listReports();
    setData((d) => ({ ...d, reports }));
  }, []);

  const markNotificationRead = useCallback((id: string) => {
    setData((d) => ({
      ...d,
      notifications: d.notifications.map((n) => (n.id === id ? { ...n, read: true } : n)),
    }));
    void api.markRead(id);
  }, []);

  const markAllNotificationsRead = useCallback(() => {
    setData((d) => ({ ...d, notifications: d.notifications.map((n) => ({ ...n, read: true })) }));
    void api.markAllRead();
  }, []);

  const value = useMemo<AppContextValue>(
    () => ({
      ...data,
      reload,
      setTaskStatus,
      postponeTask,
      toggleChecklistItem,
      recordVisitOutcome,
      claimVisit,
      editReportField,
      editReportMeta,
      submitReport,
      createReportDraft,
      markNotificationRead,
      markAllNotificationsRead,
      toast,
      dismissToast,
      toasts,
      unreadCount: data.notifications.filter((n) => !n.read).length,
    }),
    [
      data,
      reload,
      setTaskStatus,
      postponeTask,
      toggleChecklistItem,
      recordVisitOutcome,
      claimVisit,
      editReportField,
      editReportMeta,
      submitReport,
      createReportDraft,
      markNotificationRead,
      markAllNotificationsRead,
      toast,
      dismissToast,
      toasts,
    ],
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useApp(): AppContextValue {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error('useApp must be used inside <AppProvider>');
  return ctx;
}

/** Narrow selector hook — avoids re-rendering a screen on unrelated changes. */
export function useAppSelector<T>(select: (state: AppContextValue) => T): T {
  return select(useApp());
}
