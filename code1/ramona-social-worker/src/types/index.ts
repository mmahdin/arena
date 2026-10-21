/**
 * Domain types — the shared vocabulary of the app.
 *
 * These mirror the shape we expect from the platform API. When the real API
 * lands, only `src/services` should need to change (map DTOs → these types),
 * while features and components stay untouched.
 */

export type ID = string;

/* ------------------------------------------------------------------ */
/* People & cases                                                      */
/* ------------------------------------------------------------------ */

export type CaseStage =
  | "pre_assessment"
  | "assessment"
  | "introduction"
  | "placement"
  | "monitoring"
  | "closed";

export type RiskLevel = "low" | "medium" | "high";

export type GuardianRelation = "mother" | "father" | "other";

export interface Guardian {
  id: ID;
  fullName: string;
  relation: GuardianRelation;
  phone: string;
  age?: number;
  job?: string;
}

export interface Child {
  id: ID;
  fullName: string;
  age: number;
  gender: "girl" | "boy";
  /** Only set while in a placement / monitoring cycle. */
  monitoring?: MonitoringState;
}

export interface MonitoringState {
  /** Well-being indicator of the latest monitoring cycle. */
  wellbeing: "stable" | "needs_attention" | "critical";
  /** ISO date of the last monitoring visit. */
  lastVisitAt?: string;
  /** ISO date of the next periodic visit. */
  nextVisitAt?: string;
  cycle: number;
  totalCycles: number;
  note?: string;
}

export interface Address {
  city: string;
  district: string;
  line: string;
  /** Filled when a location pin exists. */
  lat?: number;
  lng?: number;
  accessNote?: string;
}

export type CaseEventKind =
  | "referral"
  | "visit"
  | "report"
  | "stage_change"
  | "document"
  | "decision";

export interface CaseEvent {
  id: ID;
  at: string;
  kind: CaseEventKind;
  title: string;
  note?: string;
  actor?: string;
}

export interface Case {
  id: ID;
  caseNumber: string;
  title: string;
  stage: CaseStage;
  riskLevel: RiskLevel;
  guardians: Guardian[];
  children: Child[];
  address: Address;
  openedAt: string;
  lastActivityAt: string;
  nextVisitAt?: string;
  summary: string;
  tags: string[];
  /** Progress of the current stage, 0–100. Nil for closed cases. */
  stageProgress?: number;
  events: CaseEvent[];
}

/* ------------------------------------------------------------------ */
/* Work items                                                          */
/* ------------------------------------------------------------------ */

export type Priority = "urgent" | "high" | "normal" | "low";

export type TaskType = "visit" | "report" | "monitoring" | "assessment" | "admin";

export type TaskStatus = "todo" | "in_progress" | "done";

export interface Task {
  id: ID;
  title: string;
  description?: string;
  type: TaskType;
  status: TaskStatus;
  priority: Priority;
  /** ISO datetime. */
  dueAt: string;
  caseId?: ID;
  caseTitle?: string;
  /** Optional deep link to the related record. */
  relatedVisitId?: ID;
  relatedReportId?: ID;
  checklist?: { label: string; done: boolean }[];
}

/* ------------------------------------------------------------------ */
/* Visits                                                              */
/* ------------------------------------------------------------------ */

export type VisitKind =
  | "initial_assessment"
  | "introduction"
  | "placement"
  | "monitoring"
  | "follow_up"
  | "office_meeting";

export type VisitStatus = "scheduled" | "completed" | "canceled" | "needs_followup";

export interface ChecklistItem {
  id: string;
  label: string;
  done: boolean;
}

export interface Visit {
  id: ID;
  caseId: ID;
  caseTitle: string;
  kind: VisitKind;
  status: VisitStatus;
  startsAt: string;
  durationMin: number;
  address: Address;
  contactName?: string;
  contactPhone?: string;
  /** What the social worker needs to accomplish. */
  objective: string;
  notes?: string;
  checklist: ChecklistItem[];
  reportId?: ID;
  childIds?: ID[];
  /** True when the platform flags the visit as fixed and non-negotiable. */
  mandatory?: boolean;
}

/* ------------------------------------------------------------------ */
/* Reports                                                             */
/* ------------------------------------------------------------------ */

export type ReportType = "home_visit" | "social_assessment" | "monitoring" | "placement";

export type ReportStatus = "draft" | "submitted" | "needs_revision" | "approved";

export interface ReportSection {
  id: string;
  label: string;
  value: string;
}

export interface Report {
  id: ID;
  title: string;
  type: ReportType;
  status: ReportStatus;
  caseId: ID;
  caseTitle: string;
  visitId?: ID;
  authorName: string;
  updatedAt: string;
  submittedAt?: string;
  /** Free-text feedback from the supervisor when returned. */
  reviewNote?: string;
  reviewedBy?: string;
  /** Overall assessment on a 1–5 scale (assessments only). */
  rating?: number;
  sections: ReportSection[];
  progress?: number;
}

/* ------------------------------------------------------------------ */
/* Notifications                                                       */
/* ------------------------------------------------------------------ */

export type NotificationKind =
  | "assignment"
  | "schedule_change"
  | "report_revision"
  | "report_approved"
  | "message"
  | "system";

export interface AppNotification {
  id: ID;
  kind: NotificationKind;
  title: string;
  body: string;
  createdAt: string;
  read: boolean;
  caseId?: ID;
  href?: string;
}

/* ------------------------------------------------------------------ */
/* Profile                                                             */
/* ------------------------------------------------------------------ */

export interface SocialWorkerProfile {
  id: ID;
  fullName: string;
  title: string;
  employeeCode: string;
  office: string;
  email: string;
  phone: string;
  licenseNumber: string;
  joinedAt: string;
  supervisor: { name: string; title: string };
  stats: { activeCases: number; visitsThisMonth: number; openReports: number; approvalRate: number };
  permissions: { label: string; granted: boolean }[];
  availability: { day: string; hours: string }[];
}
