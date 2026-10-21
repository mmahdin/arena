import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useReducer,
  type ReactNode,
} from "react";
import { api, type WorkspaceSnapshot } from "@/services/api";
import type { ID, Report, TaskStatus, VisitStatus } from "@/types";

/* ------------------------------------------------------------------ */
/* State                                                               */
/* ------------------------------------------------------------------ */

interface State {
  status: "loading" | "ready" | "error";
  data: WorkspaceSnapshot | null;
}

type Action =
  | { type: "hydrate"; data: WorkspaceSnapshot }
  | { type: "error" }
  | { type: "task/status"; id: ID; status: TaskStatus }
  | { type: "task/checklist"; taskId: ID; index: number }
  | { type: "visit/status"; id: ID; status: VisitStatus }
  | { type: "visit/checklist"; visitId: ID; itemId: string }
  | { type: "notification/read"; id: ID }
  | { type: "notifications/read-all" }
  | { type: "report/save"; id: ID; patch: Partial<Report> }
  | { type: "report/create"; report: Report }
  | { type: "profile/update"; patch: Partial<WorkspaceSnapshot["profile"]> };

const initialState: State = { status: "loading", data: null };

export function computeReportProgress(report: Report): number {
  if (report.sections.length === 0) return 0;
  const filled = report.sections.filter((s) => s.value.trim().length > 24).length;
  return Math.round((filled / report.sections.length) * 100);
}

function reducer(state: State, action: Action): State {
  if (!state.data && action.type !== "hydrate" && action.type !== "error") return state;

  switch (action.type) {
    case "hydrate":
      return { status: "ready", data: action.data };
    case "error":
      return { status: "error", data: null };
    case "task/status":
      return {
        ...state,
        data: {
          ...state.data!,
          tasks: state.data!.tasks.map((t) =>
            t.id === action.id ? { ...t, status: action.status } : t
          ),
        },
      };
    case "task/checklist":
      return {
        ...state,
        data: {
          ...state.data!,
          tasks: state.data!.tasks.map((t) =>
            t.id === action.taskId && t.checklist
              ? {
                  ...t,
                  checklist: t.checklist.map((item, i) =>
                    i === action.index ? { ...item, done: !item.done } : item
                  ),
                }
              : t
          ),
        },
      };
    case "visit/status":
      return {
        ...state,
        data: {
          ...state.data!,
          visits: state.data!.visits.map((v) =>
            v.id === action.id ? { ...v, status: action.status } : v
          ),
        },
      };
    case "visit/checklist":
      return {
        ...state,
        data: {
          ...state.data!,
          visits: state.data!.visits.map((v) =>
            v.id === action.visitId
              ? {
                  ...v,
                  checklist: v.checklist.map((item) =>
                    item.id === action.itemId ? { ...item, done: !item.done } : item
                  ),
                }
              : v
          ),
        },
      };
    case "notification/read":
      return {
        ...state,
        data: {
          ...state.data!,
          notifications: state.data!.notifications.map((n) =>
            n.id === action.id ? { ...n, read: true } : n
          ),
        },
      };
    case "notifications/read-all":
      return {
        ...state,
        data: {
          ...state.data!,
          notifications: state.data!.notifications.map((n) => ({ ...n, read: true })),
        },
      };
    case "report/save": {
      const now = state.data!.now;
      return {
        ...state,
        data: {
          ...state.data!,
          reports: state.data!.reports.map((r) => {
            if (r.id !== action.id) return r;
            const next = { ...r, ...action.patch, updatedAt: now };
            return { ...next, progress: computeReportProgress(next) };
          }),
        },
      };
    }
    case "report/create":
      return {
        ...state,
        data: {
          ...state.data!,
          reports: [action.report, ...state.data!.reports],
        },
      };
    case "profile/update":
      return {
        ...state,
        data: { ...state.data!, profile: { ...state.data!.profile, ...action.patch } },
      };
    default:
      return state;
  }
}

/* ------------------------------------------------------------------ */
/* Context                                                             */
/* ------------------------------------------------------------------ */

interface Actions {
  setTaskStatus: (id: ID, status: TaskStatus) => void;
  toggleTaskChecklist: (taskId: ID, index: number) => void;
  setVisitStatus: (id: ID, status: VisitStatus) => void;
  toggleVisitChecklist: (visitId: ID, itemId: string) => void;
  markNotificationRead: (id: ID) => void;
  markAllNotificationsRead: () => void;
  saveReport: (id: ID, patch: Partial<Report>) => void;
  createReport: (report: Report) => void;
  updateProfile: (patch: Partial<WorkspaceSnapshot["profile"]>) => void;
}

interface AppContextValue {
  status: State["status"];
  data: WorkspaceSnapshot | null;
  actions: Actions;
}

const AppContext = createContext<AppContextValue | null>(null);

export function AppDataProvider({ children }: { children: ReactNode }) {
  const [state, dispatch] = useReducer(reducer, initialState);

  useEffect(() => {
    let alive = true;
    api
      .loadWorkspace()
      .then((data) => {
        if (alive) dispatch({ type: "hydrate", data });
      })
      .catch(() => {
        if (alive) dispatch({ type: "error" });
      });
    return () => {
      alive = false;
    };
  }, []);

  const actions = useMemo<Actions>(
    () => ({
      setTaskStatus: (id, status) => {
        dispatch({ type: "task/status", id, status });
        void api.updateTaskStatus(id, status);
      },
      toggleTaskChecklist: (taskId, index) => {
        dispatch({ type: "task/checklist", taskId, index });
      },
      setVisitStatus: (id, status) => {
        dispatch({ type: "visit/status", id, status });
        void api.updateVisitStatus(id, status);
      },
      toggleVisitChecklist: (visitId, itemId) => {
        dispatch({ type: "visit/checklist", visitId, itemId });
        const visit = state.data?.visits.find((v) => v.id === visitId);
        const item = visit?.checklist.find((c) => c.id === itemId);
        void api.updateVisitChecklist(visitId, itemId, !item?.done);
      },
      markNotificationRead: (id) => {
        dispatch({ type: "notification/read", id });
      },
      markAllNotificationsRead: () => {
        dispatch({ type: "notifications/read-all" });
        void api.markAllNotificationsRead();
      },
      saveReport: (id, patch) => {
        dispatch({ type: "report/save", id, patch });
        const report = state.data?.reports.find((r) => r.id === id);
        if (report) void api.saveReport({ ...report, ...patch }, patch.status ?? report.status);
      },
      createReport: (report) => {
        dispatch({ type: "report/create", report });
        void api.saveReport(report, report.status);
      },
      updateProfile: (patch) => dispatch({ type: "profile/update", patch }),
    }),
    [state.data]
  );

  const value = useMemo<AppContextValue>(
    () => ({ status: state.status, data: state.data, actions }),
    [state.status, state.data, actions]
  );

  return <AppContext.Provider value={value}>{children}</AppContext.Provider>;
}

export function useAppContext() {
  const ctx = useContext(AppContext);
  if (!ctx) throw new Error("useAppContext must be used inside <AppDataProvider>");
  return ctx;
}

/** Workspace data for screens that render only after hydration. */
export function useWorkspace(): WorkspaceSnapshot {
  const { data } = useAppContext();
  if (!data) throw new Error("Workspace accessed before hydration");
  return data;
}

export function useAppActions(): Actions {
  return useAppContext().actions;
}

export function useNow(): Date {
  const { data } = useAppContext();
  return useMemo(() => new Date(data?.now ?? Date.now()), [data?.now]);
}

export type { Actions as AppActions };
