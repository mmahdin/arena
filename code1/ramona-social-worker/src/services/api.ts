/**
 * API boundary.
 *
 * The UI never imports mock fixtures directly — it talks to this module
 * (through `AppDataProvider`). To connect a real backend, replace the bodies
 * below with HTTP calls and map DTOs into the domain types in `src/types`.
 * Function signatures and the store contract should stay the same.
 */
import {
  cases,
  visits,
  tasks,
  reports,
  notifications,
  profile,
  NOW,
} from "@/data";
import type {
  AppNotification,
  Case,
  ID,
  Report,
  ReportStatus,
  SocialWorkerProfile,
  Task,
  TaskStatus,
  Visit,
  VisitStatus,
} from "@/types";
import { toISO } from "@/lib/date";

export interface WorkspaceSnapshot {
  /** Server "now" — the single clock used for all relative labels. */
  now: string;
  profile: SocialWorkerProfile;
  cases: Case[];
  visits: Visit[];
  tasks: Task[];
  reports: Report[];
  notifications: AppNotification[];
}

const clone = <T>(value: T): T => structuredClone(value);

export const api = {
  /** GET /workspace — everything the social worker may see, pre-filtered by permissions. */
  async loadWorkspace(): Promise<WorkspaceSnapshot> {
    return clone({
      now: toISO(NOW),
      profile: profile,
      cases,
      visits,
      tasks,
      reports,
      notifications,
    });
  },

  /* ------------------------------------------------------------------ */
  /* Mutations — optimistic in the client, authoritative on the server.  */
  /* ------------------------------------------------------------------ */

  /** PATCH /tasks/:id { status } */
  async updateTaskStatus(_id: ID, _status: TaskStatus): Promise<void> {},

  /** PATCH /visits/:id { status } */
  async updateVisitStatus(_id: ID, _status: VisitStatus): Promise<void> {},

  /** PATCH /visits/:id/checklist/:itemId { done } */
  async updateVisitChecklist(_visitId: ID, _itemId: string, _done: boolean): Promise<void> {},

  /** POST /reports  |  PATCH /reports/:id */
  async saveReport(_report: Report, _status: ReportStatus): Promise<void> {},

  /** POST /notifications/read-all */
  async markAllNotificationsRead(): Promise<void> {},
};
