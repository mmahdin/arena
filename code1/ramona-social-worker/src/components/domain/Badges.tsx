import {
  Activity,
  BadgeCheck,
  CalendarClock,
  ClipboardCheck,
  ClipboardList,
  FileText,
  FileWarning,
  FolderKanban,
  Handshake,
  HeartPulse,
  KeyRound,
  MapPin,
  MessageSquare,
  Repeat2,
  ShieldCheck,
  UserPlus,
  Users,
} from "lucide-react";
import { Badge } from "@/components/ui/Badge";
import type { Tone } from "@/lib/labels";
import {
  caseStageLabels,
  priorityLabels,
  reportStatusLabels,
  riskLabels,
  visitStatusLabels,
  wellbeingLabels,
} from "@/lib/labels";
import type {
  CaseStage,
  NotificationKind,
  Priority,
  ReportStatus,
  RiskLevel,
  TaskType,
  VisitKind,
  VisitStatus,
} from "@/types";
import "./Badges.css";

export function VisitStatusBadge({ status }: { status: VisitStatus }) {
  const config = visitStatusLabels[status];
  return <Badge tone={config.tone} dot>{config.label}</Badge>;
}

export function ReportStatusBadge({ status }: { status: ReportStatus }) {
  const config = reportStatusLabels[status];
  return <Badge tone={config.tone} dot>{config.label}</Badge>;
}

export function CaseStageBadge({ stage }: { stage: CaseStage }) {
  const tone: Tone = stage === "monitoring" ? "brand" : stage === "closed" ? "neutral" : "info";
  return <Badge tone={tone}>{caseStageLabels[stage]}</Badge>;
}

export function RiskBadge({ risk }: { risk: RiskLevel }) {
  const config = riskLabels[risk];
  return <Badge tone={config.tone} dot>{config.label}</Badge>;
}

export function PriorityBadge({ priority }: { priority: Priority }) {
  const config = priorityLabels[priority];
  return <Badge tone={config.tone}>{config.label}</Badge>;
}

export function WellbeingBadge({
  wellbeing,
}: {
  wellbeing: "stable" | "needs_attention" | "critical";
}) {
  const config = wellbeingLabels[wellbeing];
  return <Badge tone={config.tone} dot>{config.label}</Badge>;
}

/* ------------------------------------------------------------------ */
/* Icon mapping — keeps icon choice consistent wherever a type appears */
/* ------------------------------------------------------------------ */

const taskTypeIcons: Record<TaskType, typeof MapPin> = {
  visit: MapPin,
  report: FileText,
  monitoring: Activity,
  assessment: ClipboardCheck,
  admin: FolderKanban,
};

const visitKindIcons: Record<VisitKind, typeof MapPin> = {
  initial_assessment: ClipboardList,
  introduction: Handshake,
  placement: KeyRound,
  monitoring: HeartPulse,
  follow_up: Repeat2,
  office_meeting: Users,
};

const notificationIcons: Record<NotificationKind, typeof MapPin> = {
  assignment: UserPlus,
  schedule_change: CalendarClock,
  report_revision: FileWarning,
  report_approved: BadgeCheck,
  message: MessageSquare,
  system: ShieldCheck,
};

export function TaskTypeIcon({ type, size = 16 }: { type: TaskType; size?: number }) {
  const Icon = taskTypeIcons[type];
  return <Icon size={size} aria-hidden="true" />;
}

export function VisitKindIcon({ kind, size = 16 }: { kind: VisitKind; size?: number }) {
  const Icon = visitKindIcons[kind];
  return <Icon size={size} aria-hidden="true" />;
}

export function NotificationIcon({ kind, size = 17 }: { kind: NotificationKind; size?: number }) {
  const Icon = notificationIcons[kind];
  return <Icon size={size} aria-hidden="true" />;
}

export function IconTile({
  children,
  tone = "brand",
  size = "md",
}: {
  children: React.ReactNode;
  tone?: Tone;
  size?: "sm" | "md";
}) {
  return <span className={`icon-tile icon-tile--${tone} icon-tile--${size}`}>{children}</span>;
}
