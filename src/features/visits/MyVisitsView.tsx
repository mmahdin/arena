import { useMemo } from 'react';
import { useNavigate, useSearchParams } from 'react-router-dom';
import { CalendarCheck2, ClipboardCheck, FileWarning, Inbox } from 'lucide-react';
import type { CaseFile, Report, Visit } from '@/types';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { EmptyState } from '@/components/ui/EmptyState';
import { VisitRow } from '@/features/_shared/VisitRow';
import { fa } from '@/lib/digits';
import {
  VISIT_FILTERS,
  sortVisitQueue,
  visitActionReason,
  visitChronological,
  visitQueue,
  visitQueueCounts,
  type VisitFilter,
} from '@/lib/visit-workflow';

const FILTER_HINT: Record<Exclude<VisitFilter, 'all'>, string> = {
  action: 'بازدیدهای گذشته که نتیجه یا گزارش آن‌ها کامل نشده است.',
  upcoming: 'بازدیدهای پیش‌رو به ترتیب زمان.',
  done: 'بازدیدهای بسته‌شده و گزارش‌های ارسال‌شده.',
  'not-done': 'بازدیدهایی که انجام نشدند یا لغو شدند.',
};

function parseFilter(value: string | null): VisitFilter {
  return VISIT_FILTERS.some((item) => item.value === value) ? (value as VisitFilter) : 'all';
}

/**
 * All personal visits, grouped by what they ask of the worker.
 *
 * This is a management view of the same records as the day programme — never a
 * second source of truth. Report text lives in the report queue; here a visit
 * only shows whether its report is still owed.
 */
export function MyVisitsView({
  visits,
  cases,
  reports,
  poolCount = 0,
}: {
  visits: Visit[];
  cases: CaseFile[];
  reports: Report[];
  poolCount?: number;
}) {
  const [params, setParams] = useSearchParams();
  const navigate = useNavigate();
  const filter = parseFilter(params.get('filter'));
  const now = useMemo(() => new Date(), []);
  const counts = useMemo(() => visitQueueCounts(visits, reports, now), [visits, reports, now]);

  const rows = useMemo(
    () => sortVisitQueue(visits.filter((visit) => filter === 'all' || visitQueue(visit, reports, now) === filter), filter),
    [visits, reports, filter, now],
  );

  const changeFilter = (next: VisitFilter) => {
    setParams(next === 'all' ? { view: 'all' } : { view: 'all', filter: next }, { replace: true });
  };

  return (
    <div className="pb-8 pt-4">
      <ChipRail>
        {VISIT_FILTERS.map((item) => (
          <Chip
            key={item.value}
            selected={filter === item.value}
            count={counts[item.value]}
            onClick={() => changeFilter(item.value)}
          >
            {item.label}
          </Chip>
        ))}
      </ChipRail>

      <p className="mb-4 mt-3 px-1 text-xs leading-relaxed text-ink-3">
        {filter === 'all'
          ? `همه بازدیدهای شما · ${fa(counts.action)} مورد نیازمند اقدام`
          : FILTER_HINT[filter]}
      </p>

      {rows.length === 0 ? (
        <EmptyState
          icon={filter === 'not-done' ? <Inbox className="size-6" /> : filter === 'action' ? <ClipboardCheck className="size-6" /> : <CalendarCheck2 className="size-6" />}
          title={emptyTitle(filter)}
          description={emptyDescription(filter)}
          action={
            poolCount > 0 && (filter === 'upcoming' || filter === 'all') ? (
              <Button variant="secondary" onClick={() => navigate('/pool')}>
                دیدن {fa(poolCount)} بازدید آزاد
              </Button>
            ) : undefined
          }
        />
      ) : filter === 'all' ? (
        <div className="space-y-6">
          <VisitGroup
            title="نیازمند اقدام"
            rows={rows
              .filter((visit) => visitQueue(visit, reports, now) === 'action')
              .sort(visitChronological)}
            cases={cases}
            reports={reports}
          />
          <VisitGroup
            title="پیش‌رو"
            rows={rows
              .filter((visit) => visitQueue(visit, reports, now) === 'upcoming')
              .sort(visitChronological)}
            cases={cases}
            reports={reports}
          />
          <VisitGroup
            title="گذشته"
            rows={rows
              .filter((visit) => ['done', 'not-done'].includes(visitQueue(visit, reports, now)))
              .sort((a, b) => (b.date + b.time).localeCompare(a.date + a.time))}
            cases={cases}
            reports={reports}
          />
        </div>
      ) : filter === 'action' ? (
        <div className="space-y-6">
          <VisitGroup
            title="ثبت نتیجه بازدید"
            hint="زمان بازدید گذشته و نتیجه آن ثبت نشده است."
            rows={rows.filter((visit) => visitActionReason(visit, reports, now) === 'outcome')}
            cases={cases}
            reports={reports}
            tone="warning"
          />
          <VisitGroup
            title="گزارش ناقص"
            hint="بازدید انجام شده؛ گزارش آن باید تکمیل و ارسال شود."
            rows={rows.filter((visit) => visitActionReason(visit, reports, now) === 'report')}
            cases={cases}
            reports={reports}
            tone="warning"
          />
        </div>
      ) : (
        <VisitGroup
          title={filter === 'upcoming' ? 'بازدیدهای پیش‌رو' : filter === 'done' ? 'بازدیدهای انجام‌شده' : 'بازدیدهای انجام‌نشده'}
          rows={rows}
          cases={cases}
          reports={reports}
        />
      )}
    </div>
  );
}

function VisitGroup({
  title,
  hint,
  rows,
  cases,
  reports,
  tone,
}: {
  title: string;
  hint?: string;
  rows: Visit[];
  cases: CaseFile[];
  reports: Report[];
  tone?: 'warning';
}) {
  if (rows.length === 0) return null;
  return (
    <section>
      <div className="mb-2.5 flex items-baseline justify-between gap-3 px-1">
        <h2 className="flex items-center gap-1.5 text-sm font-bold text-ink">
          {tone === 'warning' && <FileWarning className="size-4 text-warning" aria-hidden />}
          {title}
        </h2>
        <span className="shrink-0 text-xs text-ink-3">{fa(rows.length)} مورد</span>
      </div>
      {hint && <p className="mb-2.5 px-1 text-xs leading-relaxed text-ink-3">{hint}</p>}
      <Card padded={false}>
        {rows.map((visit, index) => (
          <VisitRow
            key={visit.id}
            visit={visit}
            family={cases.find((family) => family.id === visit.caseId)}
            reports={reports}
            divider={index < rows.length - 1}
          />
        ))}
      </Card>
    </section>
  );
}

function emptyTitle(filter: VisitFilter): string {
  if (filter === 'action') return 'کاری در انتظار شما نیست';
  if (filter === 'upcoming') return 'بازدید پیش‌رویی ندارید';
  if (filter === 'done') return 'هنوز بازدیدی انجام نشده';
  if (filter === 'not-done') return 'بازدید انجام‌نشده‌ای نیست';
  return 'بازدیدی ثبت نشده';
}

function emptyDescription(filter: VisitFilter): string {
  if (filter === 'action') return 'همه نتیجه‌ها ثبت و گزارش‌ها تکمیل شده‌اند.';
  if (filter === 'upcoming') return 'بازدید آزاد را ببینید و متناسب با برنامه خود بپذیرید.';
  if (filter === 'done') return 'پس از ثبت نتیجه، بازدیدها اینجا نمایش داده می‌شوند.';
  return 'با تغییر فیلتر، بازدیدهای دیگر را ببینید.';
}
