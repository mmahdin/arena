/**
 * Domain model for the Social Worker app.
 *
 * These types are deliberately backend-agnostic: they describe what the UI
 * needs, not how it is stored. When the real API lands, the mapping happens in
 * `src/services/*` and these shapes can stay as they are (or be replaced by
 * generated DTOs) without touching a single screen.
 */

export type ID = string;
/** Local calendar day, `YYYY-MM-DD`. */
export type DateKey = string;
export type ISODateTime = string;

/* ------------------------------------------------------------------ *
 * Enumerations
 * ------------------------------------------------------------------ */

/** میزبانی (foster hosting) or فرزندخواندگی (adoption). */
export type CaseType = 'hosting' | 'adoption';

/** Position of a case in the Ramona process. */
export type CaseStage =
  | 'initial-assessment'
  | 'introduction'
  | 'placement'
  | 'post-placement'
  | 'closed';

export type Priority = 'critical' | 'high' | 'medium' | 'low';

export type TaskType = 'visit' | 'report' | 'assessment' | 'monitoring' | 'document' | 'meeting';

export type TaskStatus = 'todo' | 'in-progress' | 'revision' | 'blocked' | 'done';

export type VisitPurpose =
  | 'initial-assessment'
  | 'periodic-monitoring'
  | 'follow-up'
  | 'introduction'
  | 're-assessment';

export type VisitStatus =
  | 'scheduled'
  | 'completed'
  | 'family-absent'
  | 'reschedule'
  | 'cancelled';

export type ReportStatus = 'draft' | 'submitted' | 'in-review' | 'revision' | 'approved';

export type ReportType =
  | 'home-assessment'
  | 'visit'
  | 'monitoring'
  | 'placement'
  | 'incident';

/** Post-placement wellbeing signal for a child. */
export type Wellbeing = 'stable' | 'attention' | 'critical';

export type NotificationType =
  | 'assignment'
  | 'schedule'
  | 'revision'
  | 'message'
  | 'deadline'
  | 'system';

/** Visual intent — the only thing a component needs to pick a colour. */
export type Tone = 'neutral' | 'brand' | 'success' | 'warning' | 'danger' | 'info' | 'accent';

/* ------------------------------------------------------------------ *
 * Entities
 * ------------------------------------------------------------------ */

export interface Child {
  id: ID;
  firstName: string;
  age: number;
  gender: 'male' | 'female';
  caseId: ID;
  /** Where the child currently lives. */
  placement: 'family' | 'center' | 'temporary';
  wellbeing: Wellbeing;
  /** 1–5 self-assessed by the social worker; drives the trend chart. */
  wellbeingHistory: { date: DateKey; score: number; note?: string }[];
  health?: string;
  education?: string;
  needs: string[];
  nextFollowUp: DateKey;
  followUpCount: number;
  /** Restricted fields are withheld unless the user has the permission. */
  restricted?: boolean;
}

export interface FamilyMember {
  id: ID;
  name: string;
  relation: string;
  age: number;
  occupation?: string;
  phone?: string;
}

export interface CaseFile {
  id: ID;
  /** Human reference shown in headers, e.g. `RAM-0431`. */
  code: string;
  familyName: string;
  type: CaseType;
  stage: CaseStage;
  city: string;
  district?: string;
  address: string;
  phone?: string;
  headOfHousehold: string;
  members: FamilyMember[];
  children: Child[];
  priority: Priority;
  openedAt: DateKey;
  lastActivityAt: ISODateTime;
  nextAction?: { label: string; taskId?: ID; visitId?: ID };
  housing?: {
    tenure: string;
    rooms: number;
    condition: string;
  };
  income?: string;
  notes?: string;
  timeline: CaseEvent[];
  documents: CaseDocument[];
}

export interface CaseEvent {
  id: ID;
  at: ISODateTime;
  kind: 'visit' | 'report' | 'stage' | 'note' | 'decision';
  title: string;
  detail?: string;
  actor?: string;
}

export interface CaseDocument {
  id: ID;
  title: string;
  kind: 'pdf' | 'image' | 'form';
  updatedAt: ISODateTime;
}

export interface Visit {
  id: ID;
  caseId: ID;
  childIds?: ID[];
  purpose: VisitPurpose;
  status: VisitStatus;
  date: DateKey;
  time: string;
  durationMin: number;
  address: string;
  city: string;
  contactName: string;
  contactPhone: string;
  summary: string;
  checklist: ChecklistItem[];
  reportId?: ID;
  assignedBy: string;
  /** Populated once the social worker records an outcome. */
  outcome?: { status: VisitStatus; note: string; at: ISODateTime };
}

export interface ChecklistItem {
  id: ID;
  label: string;
  done: boolean;
  required?: boolean;
}

export interface Task {
  id: ID;
  title: string;
  type: TaskType;
  status: TaskStatus;
  priority: Priority;
  dueDate: DateKey;
  createdAt: ISODateTime;
  caseId?: ID;
  caseCode?: string;
  childId?: ID;
  visitId?: ID;
  reportId?: ID;
  assignedBy: string;
  description?: string;
  estimatedMin?: number;
  completedAt?: ISODateTime;
}

export interface ReportReviewComment {
  id: ID;
  author: string;
  role: string;
  at: ISODateTime;
  body: string;
}

export interface ReportSection {
  id: string;
  title: string;
  /** Filled fields / total fields — drives the completion ring. */
  fields: ReportField[];
}

export type ReportField =
  | { id: string; kind: 'text'; label: string; value: string; multiline?: boolean; hint?: string }
  | { id: string; kind: 'select'; label: string; value: string; options: string[] }
  | { id: string; kind: 'scale'; label: string; value: number; max: number; lowLabel: string; highLabel: string }
  | { id: string; kind: 'checklist'; label: string; value: string[]; options: string[] };

export interface Report {
  id: ID;
  title: string;
  type: ReportType;
  status: ReportStatus;
  caseId: ID;
  caseCode: string;
  visitId?: ID;
  childId?: ID;
  author: string;
  reviewer?: string;
  createdAt: ISODateTime;
  updatedAt: ISODateTime;
  submittedAt?: ISODateTime;
  sections: ReportSection[];
  summary: string;
  comments: ReportReviewComment[];
  attachments: { id: ID; name: string; size: string }[];
}

export interface AppNotification {
  id: ID;
  type: NotificationType;
  title: string;
  body: string;
  at: ISODateTime;
  read: boolean;
  /** Deep link used when the notification is tapped. */
  href?: string;
  priority?: Priority;
}

export interface CalendarEvent {
  id: ID;
  title: string;
  kind: 'visit' | 'task' | 'meeting' | 'deadline';
  date: DateKey;
  time?: string;
  durationMin?: number;
  href?: string;
  tone: Tone;
  caseCode?: string;
  /** Present only on deadline entries, for related task actions and status. */
  taskId?: ID;
  taskStatus?: TaskStatus;
}

export interface UserProfile {
  id: ID;
  name: string;
  role: string;
  organization: string;
  unit: string;
  employeeCode: string;
  licenseNumber: string;
  phone: string;
  email: string;
  supervisor: string;
  coverageAreas: string[];
  permissions: string[];
  stats: { activeCases: number; visitsThisMonth: number; approvedReports: number };
}
