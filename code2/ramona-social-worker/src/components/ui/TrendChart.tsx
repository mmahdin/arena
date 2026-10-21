import { TrendingDown, TrendingUp, Minus } from 'lucide-react';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { faDate } from '@/lib/jalali';
import { wellbeingTrend } from '@/lib/domain';

/**
 * Wellbeing trend.
 *
 * A five-bar column chart, not a line graph: bars make each periodic visit a
 * discrete, comparable judgement, which is how the assessment actually works.
 * Rendered with plain divs — no chart dependency, no canvas, no loading state.
 */
export function TrendChart({
  history,
  className,
}: {
  history: { date: string; score: number; note?: string }[];
  className?: string;
}) {
  const rows = [...history].sort((a, b) => a.date.localeCompare(b.date)).slice(-6);
  const trend = wellbeingTrend(rows);
  const latest = rows[rows.length - 1]?.score ?? 0;
  const tone = latest >= 4 ? 'bg-success' : latest === 3 ? 'bg-warning' : 'bg-danger';

  const TrendIcon = trend === 'up' ? TrendingUp : trend === 'down' ? TrendingDown : Minus;
  const trendTone = trend === 'up' ? 'text-success' : trend === 'down' ? 'text-danger' : 'text-ink-3';
  const trendLabel = trend === 'up' ? 'روند بهبود' : trend === 'down' ? 'روند نزولی' : 'روند ثابت';

  return (
    <div className={cn('rounded-lg border border-line bg-surface p-4', className)}>
      <div className="flex items-start justify-between gap-3">
        <div>
          <p className="text-xs font-medium text-ink-3">روند وضعیت کودک</p>
          <p className="mt-1 flex items-baseline gap-1.5">
            <span className="text-2xl font-extrabold text-ink tabular-nums">{fa(latest)}</span>
            <span className="text-xs text-ink-3">از ۵</span>
          </p>
        </div>
        <span className={cn('flex items-center gap-1 text-xs font-semibold', trendTone)}>
          <TrendIcon className="size-4" strokeWidth={2.4} aria-hidden />
          {trendLabel}
        </span>
      </div>

      <div
        className="mt-4 flex h-24 items-end gap-2"
        role="img"
        aria-label={`روند امتیاز از ${fa(rows[0]?.score ?? 0)} به ${fa(latest)} از ۵`}
      >
        {rows.map((r) => (
          <div key={r.date} className="flex h-full flex-1 flex-col items-center justify-end gap-1">
            <span className="text-2xs font-bold text-ink-2 tabular-nums">{fa(r.score)}</span>
            <span
              className={cn(
                'w-full rounded-t-sm transition-all duration-slow ease-out',
                r.date === rows[rows.length - 1].date ? tone : 'bg-brand-muted',
              )}
              style={{ height: `${Math.max(6, (r.score / 5) * 68)}px` }}
            />
          </div>
        ))}
      </div>

      <div className="mt-2 flex gap-2">
        {rows.map((r) => (
          <span key={r.date} className="flex-1 text-center text-2xs text-ink-3">
            {faDate(r.date, { year: false })}
          </span>
        ))}
      </div>
    </div>
  );
}
