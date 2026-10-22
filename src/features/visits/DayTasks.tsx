import { useMemo, useState } from 'react';
import { Link } from 'react-router-dom';
import { CalendarCheck2, Clock3, MoreVertical } from 'lucide-react';
import type { CalendarEvent } from '@/types';
import { useApp } from '@/store/app-store';
import { Badge } from '@/components/ui/Badge';
import { IconButton } from '@/components/ui/IconButton';
import { buildEvents } from '@/services/calendar';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { faDuration, faTime } from '@/lib/jalali';
import { TaskMenu } from '@/features/_shared/TaskRow';

/**
 * The rest of the day.
 *
 * Meetings and paperwork are real commitments, but the visit programme is the
 * reason the screen exists. They stay collapsed so the day still reads as
 * "visits first" without losing information.
 */
export function DayTasks({ date }: { date: string }) {
  const { tasks, visits } = useApp();
  const events = useMemo(
    () => buildEvents().filter((event) => event.kind !== 'visit' && event.date === date),
    // `tasks` and `visits` are the reactive inputs; buildEvents reads the same db snapshot.
    [tasks, visits, date],
  );

  if (events.length === 0) return null;

  return (
    <details className="overflow-hidden rounded-lg border border-line bg-surface">
      <summary className="press flex min-h-12 cursor-pointer list-none items-center justify-between gap-2 px-4 py-3 text-sm font-semibold text-ink-2 [&::-webkit-details-marker]:hidden">
        <span>سایر کارهای این روز</span>
        <span className="text-xs text-ink-3">{fa(events.length)} مورد</span>
      </summary>
      <div className="border-t border-line-soft">
        {events.map((event, index) => (
          <OtherWorkRow key={event.id} event={event} divider={index < events.length - 1} />
        ))}
      </div>
    </details>
  );
}

/** Secondary duties stay available on demand, but never displace visits. */
function OtherWorkRow({ event, divider }: { event: CalendarEvent; divider?: boolean }) {
  const { tasks } = useApp();
  const [menuOpen, setMenuOpen] = useState(false);
  const task = event.taskId ? tasks.find((t) => t.id === event.taskId) : undefined;
  const target = event.href ?? `/visits?date=${event.date}`;
  const openInPlace = Boolean(task && event.href?.startsWith('/visits?date='));
  const Icon = event.kind === 'meeting' ? Clock3 : CalendarCheck2;

  const content = (
    <>
      <span className="min-w-0 flex-1">
        <span className="block text-sm font-semibold text-ink">{event.title}</span>
        <span className="mt-1 flex items-center gap-1.5 text-xs text-ink-3">
          <Icon className="size-3.5" aria-hidden /> {event.kind === 'meeting' ? 'جلسه' : 'وظیفه'}
          {event.time && ` · ${faTime(event.time)}`}
          {event.durationMin && ` · ${faDuration(event.durationMin)}`}
        </span>
      </span>
      {event.taskStatus === 'done' && <Badge tone="success">انجام‌شده</Badge>}
    </>
  );

  const className = cn(
    'press flex min-h-14 flex-1 items-center gap-3 px-4 py-3 text-start hover:bg-sunken',
    divider && 'border-b border-line-soft',
  );

  return (
    <>
      <div className="flex items-center">
        {openInPlace ? (
          <button type="button" className={className} onClick={() => setMenuOpen(true)}>
            {content}
          </button>
        ) : (
          <Link to={target} className={className}>
            {content}
          </Link>
        )}
        {task && !openInPlace && (
          <IconButton label="گزینه‌های کار" size="sm" onClick={() => setMenuOpen(true)}>
            <MoreVertical className="size-4" />
          </IconButton>
        )}
      </div>
      {task && <TaskMenu task={task} open={menuOpen} onClose={() => setMenuOpen(false)} />}
    </>
  );
}
