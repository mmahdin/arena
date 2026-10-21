import { Link } from 'react-router-dom';
import { Clock3, MapPin } from 'lucide-react';
import type { AgendaItem } from '@/services/dashboard';
import { cn } from '@/lib/cn';
import { faDuration, faTime } from '@/lib/jalali';
import { VISIT_STATUS } from '@/lib/labels';
import { Badge } from '@/components/ui/Badge';
import { EmptyState } from '@/components/ui/EmptyState';
import { CalendarDays } from 'lucide-react';

/**
 * Day timeline.
 *
 * The spine of the "today" screen: a time column, a rail, and one card per
 * commitment. Completed items recede, items awaiting an outcome are flagged —
 * so the day reads correctly at a glance even after a long field shift.
 */
export function AgendaTimeline({ items, className }: { items: AgendaItem[]; className?: string }) {
  if (items.length === 0) {
    return (
      <div className="rounded-lg border border-dashed border-line-strong bg-surface/60">
        <EmptyState
          icon={<CalendarDays className="size-6" />}
          title="برنامه‌ای برای امروز ثبت نشده"
          description="روز کاری شما خالی است. می‌توانید از تقویم برنامه هفته را ببینید."
        />
      </div>
    );
  }

  return (
    <ol className={cn('relative', className)}>
      {items.map((item, i) => {
        const last = i === items.length - 1;
        const dotTone =
          item.state === 'done'
            ? 'bg-success'
            : item.state === 'needs-outcome'
              ? 'bg-warning'
              : item.kind === 'meeting'
                ? 'bg-ink-3'
                : 'bg-brand';

        return (
          <li key={item.id} className="relative flex gap-2.5">
            <div className="w-12 shrink-0 pt-2.5 text-end">
              <span className="block text-sm font-bold text-ink tabular-nums">{item.time ? faTime(item.time) : '—'}</span>
              {item.durationMin && (
                <span className="mt-0.5 block text-2xs text-ink-3 tabular-nums">{faDuration(item.durationMin)}</span>
              )}
            </div>

            <div className="relative w-3 shrink-0" aria-hidden>
              {i > 0 && <span className="absolute start-1/2 top-0 h-3 w-px -translate-x-1/2 bg-line" />}
              {!last && <span className="absolute start-1/2 top-[1.6rem] bottom-0 w-px -translate-x-1/2 bg-line" />}
              <span
                className={cn(
                  'absolute start-1/2 top-[0.85rem] size-2.5 -translate-x-1/2 rounded-full ring-4 ring-bg',
                  dotTone,
                )}
              />
            </div>

            <div className="min-w-0 flex-1 pb-3">
              <AgendaCard item={item} />
            </div>
          </li>
        );
      })}
    </ol>
  );
}

function AgendaCard({ item }: { item: AgendaItem }) {
  const done = item.state === 'done';
  const needsOutcome = item.state === 'needs-outcome';

  return (
    <Link
      to={item.href}
      className={cn(
        'press block rounded-lg border bg-surface p-3.5 shadow-xs',
        done ? 'border-line-soft opacity-70' : needsOutcome ? 'border-warning/40' : 'border-line',
      )}
    >
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <p className={cn('truncate text-base font-bold', done ? 'text-ink-2 line-through decoration-line-strong' : 'text-ink')}>
            {item.title}
          </p>
          {item.subtitle && <p className="mt-0.5 truncate text-sm text-ink-3">{item.subtitle}</p>}
        </div>
        {done ? (
          <Badge tone="success">{VISIT_STATUS.completed.label}</Badge>
        ) : needsOutcome ? (
          <Badge tone="warning">ثبت نتیجه</Badge>
        ) : item.kind === 'meeting' ? (
          <Badge tone="neutral">جلسه</Badge>
        ) : null}
      </div>

      {item.visit && (
        <div className="mt-2 flex items-center gap-3.5 overflow-hidden text-xs text-ink-3">
          <span className="flex min-w-0 items-center gap-1.5">
            <MapPin className="size-3.5 shrink-0" strokeWidth={2.25} aria-hidden />
            <span className="truncate">{item.visit.city}</span>
          </span>
          <span className="flex shrink-0 items-center gap-1.5">
            <Clock3 className="size-3.5" strokeWidth={2.25} aria-hidden />
            <span className="tabular-nums">{faDuration(item.visit.durationMin)}</span>
          </span>
        </div>
      )}
    </Link>
  );
}
