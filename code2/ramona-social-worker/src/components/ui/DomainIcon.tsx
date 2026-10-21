import type { LucideIcon } from 'lucide-react';
import {
  AlertTriangle,
  CalendarClock,
  ClipboardCheck,
  FileText,
  FolderOpen,
  HeartHandshake,
  HeartPulse,
  Home,
  MapPin,
  Megaphone,
  MessageSquare,
  PenLine,
  RefreshCw,
  Timer,
  UserPlus,
  Users,
} from 'lucide-react';
import type { NotificationType, ReportType, TaskType, VisitPurpose } from '@/types';
import { cn } from '@/lib/cn';
import { TONE, type ToneClasses } from './tone';
import type { Tone } from '@/types';

/**
 * Icon vocabulary.
 *
 * One icon per domain concept, declared once. A user learns "house = home
 * visit", "pulse = monitoring" and then reads any list, badge or calendar dot
 * without labels.
 */
export const TASK_ICON: Record<TaskType, LucideIcon> = {
  visit: Home,
  report: FileText,
  assessment: ClipboardCheck,
  monitoring: HeartPulse,
  document: FolderOpen,
  meeting: Users,
};

export const REPORT_TYPE_ICON: Record<ReportType, LucideIcon> = {
  'home-assessment': Home,
  visit: MapPin,
  monitoring: HeartPulse,
  placement: HeartHandshake,
  incident: AlertTriangle,
};

export const NOTIFICATION_ICON: Record<NotificationType, LucideIcon> = {
  assignment: UserPlus,
  schedule: CalendarClock,
  revision: PenLine,
  message: MessageSquare,
  deadline: Timer,
  system: Megaphone,
};

export const VISIT_PURPOSE_ICON: Record<VisitPurpose, LucideIcon> = {
  'initial-assessment': Home,
  'periodic-monitoring': HeartPulse,
  'follow-up': RefreshCw,
  introduction: HeartHandshake,
  're-assessment': ClipboardCheck,
};

/** Soft tinted square that carries a domain icon at the start of a row. */
export function IconTile({
  icon: Icon,
  tone = 'neutral',
  size = 'md',
  className,
}: {
  icon: LucideIcon;
  tone?: Tone;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
}) {
  const t: ToneClasses = TONE[tone];
  return (
    <span
      aria-hidden
      className={cn(
        'grid shrink-0 place-items-center rounded-md',
        size === 'sm' && 'size-8',
        size === 'md' && 'size-10',
        size === 'lg' && 'size-12',
        t.soft,
        t.text,
        className,
      )}
    >
      <Icon className={cn(size === 'sm' ? 'size-4' : size === 'md' ? 'size-[18px]' : 'size-5')} strokeWidth={2.1} />
    </span>
  );
}
