import type {
  CaseStage,
  GuardianRelation,
  NotificationKind,
  Priority,
  ReportStatus,
  ReportType,
  RiskLevel,
  TaskType,
  VisitKind,
  VisitStatus,
} from "@/types";

export type Tone = "neutral" | "brand" | "success" | "warning" | "danger" | "info" | "accent";

/* ------------------------------------------------------------------ */
/* Cases                                                              */
/* ------------------------------------------------------------------ */

export const caseStageLabels: Record<CaseStage, string> = {
  pre_assessment: "پیش از ارزیابی",
  assessment: "ارزیابی اولیه",
  introduction: "معرفی و آشنایی",
  placement: "واگذاری",
  monitoring: "پایش پس از واگذاری",
  closed: "بسته‌شده",
};

export const caseStageShort: Record<CaseStage, string> = {
  pre_assessment: "پیش‌ارزیابی",
  assessment: "ارزیابی",
  introduction: "معرفی",
  placement: "واگذاری",
  monitoring: "پایش",
  closed: "بسته",
};

/** Order used by the case progress rail. */
export const caseStages: CaseStage[] = [
  "assessment",
  "introduction",
  "placement",
  "monitoring",
];

export const riskLabels: Record<RiskLevel, { label: string; tone: Tone }> = {
  low: { label: "ریسک پایین", tone: "success" },
  medium: { label: "ریسک متوسط", tone: "warning" },
  high: { label: "ریسک بالا", tone: "danger" },
};

export const relationLabels: Record<GuardianRelation, string> = {
  mother: "مادر",
  father: "پدر",
  other: "سرپرست",
};

/* ------------------------------------------------------------------ */
/* Tasks                                                              */
/* ------------------------------------------------------------------ */

export const taskTypeLabels: Record<TaskType, string> = {
  visit: "بازدید",
  report: "گزارش",
  monitoring: "پایش",
  assessment: "ارزیابی",
  admin: "اداری",
};

export const priorityLabels: Record<Priority, { label: string; tone: Tone; rank: number }> = {
  urgent: { label: "فوری", tone: "danger", rank: 0 },
  high: { label: "بالا", tone: "accent", rank: 1 },
  normal: { label: "معمولی", tone: "neutral", rank: 2 },
  low: { label: "پایین", tone: "neutral", rank: 3 },
};

/* ------------------------------------------------------------------ */
/* Visits                                                             */
/* ------------------------------------------------------------------ */

export const visitKindLabels: Record<VisitKind, string> = {
  initial_assessment: "ارزیابی اولیه",
  introduction: "جلسه معرفی",
  placement: "واگذاری",
  monitoring: "پایش دوره‌ای",
  follow_up: "بازدید پیگیری",
  office_meeting: "جلسه اداری",
};

export const visitStatusLabels: Record<VisitStatus, { label: string; tone: Tone }> = {
  scheduled: { label: "در انتظار انجام", tone: "info" },
  completed: { label: "انجام‌شده", tone: "success" },
  canceled: { label: "لغوشده", tone: "neutral" },
  needs_followup: { label: "نیاز به پیگیری", tone: "warning" },
};

/* ------------------------------------------------------------------ */
/* Reports                                                            */
/* ------------------------------------------------------------------ */

export const reportTypeLabels: Record<ReportType, string> = {
  home_visit: "گزارش بازدید منزل",
  social_assessment: "ارزیابی اجتماعی",
  monitoring: "گزارش پایش",
  placement: "گزارش واگذاری",
};

export const reportStatusLabels: Record<ReportStatus, { label: string; tone: Tone }> = {
  draft: { label: "پیش‌نویس", tone: "neutral" },
  submitted: { label: "در انتظار بررسی", tone: "info" },
  needs_revision: { label: "نیازمند اصلاح", tone: "warning" },
  approved: { label: "تأییدشده", tone: "success" },
};

/* ------------------------------------------------------------------ */
/* Monitoring                                                         */
/* ------------------------------------------------------------------ */

export const wellbeingLabels: Record<
  "stable" | "needs_attention" | "critical",
  { label: string; tone: Tone }
> = {
  stable: { label: "پایدار", tone: "success" },
  needs_attention: { label: "نیازمند توجه", tone: "warning" },
  critical: { label: "بحرانی", tone: "danger" },
};

/* ------------------------------------------------------------------ */
/* Notifications                                                      */
/* ------------------------------------------------------------------ */

export const notificationKindLabels: Record<NotificationKind, string> = {
  assignment: "ارجاع جدید",
  schedule_change: "تغییر برنامه",
  report_revision: "اصلاح گزارش",
  report_approved: "تأیید گزارش",
  message: "پیام",
  system: "سیستمی",
};
