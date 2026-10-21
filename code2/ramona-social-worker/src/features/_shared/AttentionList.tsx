import { Link } from 'react-router-dom';
import { AlertTriangle, CalendarClock, HeartPulse, PenLine, Timer } from 'lucide-react';
import type { AttentionItem } from '@/services/dashboard';
import { cn } from '@/lib/cn';
import { IconTile } from '@/components/ui/DomainIcon';
import { ChevronLeft } from 'lucide-react';

const ICON = {
  revision: PenLine,
  overdue: Timer,
  'visit-outcome': CalendarClock,
  wellbeing: HeartPulse,
  deadline: AlertTriangle,
} as const;

/**
 * Attention block.
 *
 * The first thing on the home screen when something is wrong. Deliberately
 * capped at four items and ordered by severity: a list that can grow without
 * limit stops being a signal and becomes noise.
 */
export function AttentionList({ items }: { items: AttentionItem[] }) {
  if (items.length === 0) return null;

  return (
    <div className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
      {items.map((item, i) => {
        const Icon = ICON[item.kind];
        return (
          <Link
            key={item.id}
            to={item.href}
            className={cn(
              'press flex items-center gap-3 px-4 py-3.5 hover:bg-sunken',
              i > 0 && 'border-t border-line-soft',
            )}
          >
            <IconTile icon={Icon} tone={item.tone} />
            <span className="min-w-0 flex-1">
              <span className="block truncate text-base font-semibold text-ink">{item.title}</span>
              <span className="mt-0.5 block truncate text-sm text-ink-3">{item.detail}</span>
            </span>
            <ChevronLeft className="size-[18px] shrink-0 text-ink-3" strokeWidth={2.25} aria-hidden />
          </Link>
        );
      })}
    </div>
  );
}
