import type {
  CaseStage,
  CaseType,
  NotificationType,
  Priority,
  ReportStatus,
  ReportType,
  TaskStatus,
  TaskType,
  Tone,
  VisitPurpose,
  VisitStatus,
  Wellbeing,
} from '@/types';

/**
 * Domain vocabulary.
 *
 * Every enumeration has exactly one Persian label and one visual tone here.
 * Screens read from this map instead of hard-coding strings or colours, so a
 * new status added later shows up consistently everywhere at once.
 */

interface Meta {
  label: string;
  tone: Tone;
}

export const CASE_TYPE: Record<CaseType, Meta> = {
  hosting: { label: 'میزبانی', tone: 'accent' },
  adoption: { label: 'فرزندخواندگی', tone: 'brand' },
};

/** Ordered — the case detail screen renders this as a progress stepper. */
export const CASE_STAGE_ORDER: CaseStage[] = [
  'initial-assessment',
  'introduction',
  'placement',
  'post-placement',
  'closed',
];

export const CASE_STAGE: Record<CaseStage, Meta> = {
  'initial-assessment': { label: 'ارزیابی اولیه', tone: 'info' },
  introduction: { label: 'معرفی و آشنایی', tone: 'brand' },
  placement: { label: 'جایگذاری', tone: 'accent' },
  'post-placement': { label: 'پایش پس از جایگذاری', tone: 'success' },
  closed: { label: 'پایان‌یافته', tone: 'neutral' },
};

export const CASE_STAGE_SHORT: Record<CaseStage, string> = {
  'initial-assessment': 'ارزیابی اولیه',
  introduction: 'معرفی',
  placement: 'جایگذاری',
  'post-placement': 'پایش',
  closed: 'پایان‌یافته',
};

export const PRIORITY: Record<Priority, Meta> = {
  critical: { label: 'فوری', tone: 'danger' },
  high: { label: 'بالا', tone: 'warning' },
  medium: { label: 'متوسط', tone: 'info' },
  low: { label: 'پایین', tone: 'neutral' },
};

export const TASK_TYPE: Record<TaskType, Meta> = {
  visit: { label: 'بازدید', tone: 'brand' },
  report: { label: 'گزارش', tone: 'info' },
  assessment: { label: 'ارزیابی', tone: 'accent' },
  monitoring: { label: 'پایش', tone: 'success' },
  document: { label: 'مستندات', tone: 'neutral' },
  meeting: { label: 'جلسه', tone: 'neutral' },
};

export const TASK_STATUS: Record<TaskStatus, Meta> = {
  todo: { label: 'در انتظار اقدام', tone: 'neutral' },
  'in-progress': { label: 'در جریان', tone: 'info' },
  revision: { label: 'نیازمند اصلاح', tone: 'warning' },
  blocked: { label: 'مسدود', tone: 'danger' },
  done: { label: 'انجام‌شده', tone: 'success' },
};

export const VISIT_PURPOSE: Record<VisitPurpose, Meta> = {
  'initial-assessment': { label: 'ارزیابی اولیه منزل', tone: 'info' },
  'periodic-monitoring': { label: 'پایش دوره‌ای', tone: 'success' },
  'follow-up': { label: 'پیگیری موردی', tone: 'brand' },
  introduction: { label: 'جلسه آشنایی', tone: 'accent' },
  're-assessment': { label: 'ارزیابی مجدد', tone: 'warning' },
};

export const VISIT_STATUS: Record<VisitStatus, Meta> = {
  scheduled: { label: 'زمان‌بندی‌شده', tone: 'info' },
  completed: { label: 'انجام شد', tone: 'success' },
  'family-absent': { label: 'خانواده در منزل نبود', tone: 'warning' },
  reschedule: { label: 'نیاز به هماهنگی مجدد', tone: 'accent' },
  cancelled: { label: 'لغو شد', tone: 'neutral' },
};

export const REPORT_STATUS: Record<ReportStatus, Meta> = {
  draft: { label: 'پیش‌نویس', tone: 'neutral' },
  submitted: { label: 'ارسال‌شده', tone: 'info' },
  'in-review': { label: 'در بازبینی سرپرست', tone: 'brand' },
  revision: { label: 'نیازمند اصلاح', tone: 'warning' },
  approved: { label: 'تأییدشده', tone: 'success' },
};

export const REPORT_TYPE: Record<ReportType, Meta> = {
  'home-assessment': { label: 'ارزیابی شرایط خانواده', tone: 'info' },
  visit: { label: 'گزارش بازدید', tone: 'brand' },
  monitoring: { label: 'گزارش پایش کودک', tone: 'success' },
  placement: { label: 'گزارش جایگذاری', tone: 'accent' },
  incident: { label: 'گزارش مورد خاص', tone: 'danger' },
};

export const WELLBEING: Record<Wellbeing, Meta> = {
  stable: { label: 'پایدار', tone: 'success' },
  attention: { label: 'نیازمند توجه', tone: 'warning' },
  critical: { label: 'نیازمند مداخله', tone: 'danger' },
};

export const NOTIFICATION_TYPE: Record<NotificationType, Meta> = {
  assignment: { label: 'مأموریت جدید', tone: 'brand' },
  schedule: { label: 'تغییر برنامه', tone: 'info' },
  revision: { label: 'درخواست اصلاح', tone: 'warning' },
  message: { label: 'پیام', tone: 'neutral' },
  deadline: { label: 'یادآوری مهلت', tone: 'danger' },
  system: { label: 'اطلاعیه', tone: 'neutral' },
};

export const label = (m: Meta) => m.label;
export const tone = (m: Meta) => m.tone;
