import { useMemo } from 'react';
import { Link, useSearchParams } from 'react-router-dom';
import { ChevronLeft, ClipboardCheck, FileText, Inbox } from 'lucide-react';
import type { CaseFile, Report } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Card } from '@/components/ui/Card';
import { EmptyState } from '@/components/ui/EmptyState';
import { ReportRow } from '@/features/_shared/ReportRow';
import { visitNeedsReport } from '@/lib/visit-workflow';
import { fa } from '@/lib/digits';

type Filter = 'action' | 'review' | 'approved';
const isAction = (r: Report) => r.status === 'draft' || r.status === 'revision';
const isReview = (r: Report) => r.status === 'submitted' || r.status === 'in-review';

/**
 * Report queue.
 *
 * Only reports live here: what I must finish, what waits for the supervisor and
 * what is approved. A completed visit without a report is a visit-level fact,
 * so it is surfaced as a single pointer into the visit queue instead of a
 * second list of the same visits.
 */
export function ReportsScreen() {
  const { reports, user, cases, visits } = useApp();
  const myReports = useMemo(() => reports.filter((r) => r.author === user?.name), [reports, user?.name]);
  const [params, setParams] = useSearchParams();
  const requested = params.get('filter');
  const filter: Filter = requested === 'review' || requested === 'approved' ? requested : 'action';

  const reportsNeedingVisit = visits.filter((visit) => visitNeedsReport(visit, reports));
  const revision = myReports.filter((r) => r.status === 'revision').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const drafts = myReports.filter((r) => r.status === 'draft').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const reviewing = myReports.filter(isReview).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const approved = myReports.filter((r) => r.status === 'approved').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const actionCount = revision.length + drafts.length;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="گزارش‌ها"
        subtitle={actionCount > 0 ? `${fa(actionCount)} گزارش نیازمند اقدام شما` : 'گزارشی در انتظار اقدام شما نیست'}
      >
        <SegmentedControl<Filter>
          className="mt-3"
          ariaLabel="وضعیت گزارش‌ها"
          value={filter}
          onChange={(value) => setParams(value === 'action' ? {} : { filter: value }, { replace: true })}
          items={[
            { value: 'action', label: 'اقدام من', count: actionCount },
            { value: 'review', label: 'در بررسی', count: reviewing.length },
            { value: 'approved', label: 'تأییدشده', count: approved.length },
          ]}
        />
      </ScreenHeader>

      <div className="space-y-6 pb-8 pt-5">
        {filter === 'action' && reportsNeedingVisit.length > 0 && (
          <Link
            to="/visits?view=all&filter=action"
            className="press flex items-center gap-2.5 rounded-lg border border-warning/25 bg-warning-soft px-3.5 py-3 text-start"
          >
            <ClipboardCheck className="size-4 shrink-0 text-warning" aria-hidden />
            <span className="min-w-0 flex-1 text-xs font-semibold leading-relaxed text-ink-2">
              {fa(reportsNeedingVisit.length)} بازدید انجام‌شده گزارش ناقص دارد — در بازدیدها تکمیل کنید
            </span>
            <ChevronLeft className="size-4 shrink-0 text-warning" aria-hidden />
          </Link>
        )}

        {filter === 'action' && actionCount === 0 && reportsNeedingVisit.length === 0 && (
          <EmptyState
            icon={<ClipboardCheck className="size-6" />}
            title="گزارشی نیازمند اقدام شما نیست"
            description="همه گزارش‌ها ارسال شده‌اند یا در حال بررسی‌اند."
          />
        )}
        {filter === 'review' && reviewing.length === 0 && (
          <EmptyState icon={<FileText className="size-6" />} title="گزارشی در حال بررسی نیست" />
        )}
        {filter === 'approved' && approved.length === 0 && (
          <EmptyState icon={<Inbox className="size-6" />} title="هنوز گزارشی تأیید نشده" />
        )}

        {filter === 'action' && (
          <>
            {revision.length > 0 && (
              <ReportGroup
                title="نیازمند اصلاح"
                hint="سرپرست گزارش را برای اصلاح بازگردانده است."
                rows={revision}
                cases={cases}
              />
            )}
            {drafts.length > 0 && (
              <ReportGroup
                title="پیش‌نویس‌ها"
                hint="گزارش‌های ناتمام؛ پس از تکمیل برای بررسی ارسال می‌شوند."
                rows={drafts}
                cases={cases}
              />
            )}
          </>
        )}
        {filter === 'review' && reviewing.length > 0 && (
          <ReportGroup title="ارسال‌شده برای بررسی" rows={reviewing} cases={cases} />
        )}
        {filter === 'approved' && approved.length > 0 && (
          <ReportGroup title="گزارش‌های تأییدشده" rows={approved} cases={cases} />
        )}
      </div>
    </Screen>
  );
}

function ReportGroup({
  title,
  hint,
  rows,
  cases,
}: {
  title: string;
  hint?: string;
  rows: Report[];
  cases: CaseFile[];
}) {
  return (
    <section>
      <div className="mb-2.5 flex items-baseline justify-between gap-3 px-1">
        <h2 className="text-sm font-bold text-ink">{title}</h2>
        <span className="shrink-0 text-xs text-ink-3">{fa(rows.length)} گزارش</span>
      </div>
      {hint && <p className="mb-2.5 px-1 text-xs leading-relaxed text-ink-3">{hint}</p>}
      <Card padded={false}>
        {rows.map((report, index) => (
          <ReportRow
            key={report.id}
            report={report}
            familyName={cases.find((c) => c.id === report.caseId)?.familyName}
            divider={index < rows.length - 1}
          />
        ))}
      </Card>
    </section>
  );
}
