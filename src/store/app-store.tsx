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
import { myCaseIds, myVisits as selectMyVisits, poolVisits as selectPoolVisits } from '@/lib/domain';

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
  tasks: Task[];
  reports: Report[];
  notifications: AppNotification[];
}

interface AppContextValue extends AppData {
  reload: () => Promise<void>;
  /**
   * Derived scopes, computed once here so every screen agrees on what is
   * "mine". `visits`/`cases` stay the raw collections (detail screens look
   * anything up by id); lists and counts should use these instead.
   */
  /** Visits I am responsible for — «بازدیدهای من». */
  myVisits: Visit[];
  /** Claimable unassigned visits — «بازدیدهای آزاد». */
  poolVisits: Visit[];
  /** Families in my caseload — at least one of their visits is/was mine. */
  myCases: CaseFile[];
  /* tasks */
  setTaskStatus: (id: string, status: TaskStatus) => Promise<void>;
  postponeTask: (id: string, days: number) => Promise<void>;
  /* visits */
  toggleChecklistItem: (visitId: string, itemId: string) => void;
  recordVisitOutcome: (visitId: string, status: VisitStatus, note: string) => Promise<{ reportId?: string }>;
  /** Take responsibility for a pool visit; returns it once it is mine. */
  claimVisit: (visitId: string) => Promise<Visit | undefined>;
  /** Give one of my upcoming visits back to the pool. */
  releaseVisit: (visitId: string) => Promise<Visit | undefined>;
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
    const [user, cases, visits, tasks, reports, notifications] = await Promise.all([
      api.getCurrentUser(),
      api.listCases(),
      api.listVisits(),
      api.listTasks(),
      api.listReports(),
      api.listNotifications(),
    ]);
    setData({ status: 'ready', user, cases, visits, tasks, reports, notifications });
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


  // Claiming or releasing only flips `assignedToId` in the db; reloading the
  // two collections updates the pool, my schedule, my families and every
  // badge at once, because all of them derive from that single field.
  const claimVisit = useCallback(async (visitId: string) => {
    const claimed = await api.claimVisit(visitId);
    if (!claimed) return undefined;
    const [visits, cases, notifications] = await Promise.all([
      api.listVisits(),
      api.listCases(),
      api.listNotifications(),
    ]);
    setData((d) => ({ ...d, visits, cases, notifications }));
    return claimed;
  }, []);

  const releaseVisit = useCallback(async (visitId: string) => {
    const released = await api.releaseVisit(visitId);
    if (!released) return undefined;
    const [visits, cases] = await Promise.all([api.listVisits(), api.listCases()]);
    setData((d) => ({ ...d, visits, cases }));
    return released;
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
      myVisits: data.user ? selectMyVisits(data.visits, data.user.id) : [],
      poolVisits: selectPoolVisits(data.visits),
      myCases: data.user
        ? data.cases.filter((c) => myCaseIds(data.visits, data.user!.id).has(c.id))
        : [],
      reload,
      setTaskStatus,
      postponeTask,
      toggleChecklistItem,
      recordVisitOutcome,
      claimVisit,
      releaseVisit,
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
      releaseVisit,
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
