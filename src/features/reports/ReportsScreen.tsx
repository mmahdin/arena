import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FileCheck2, FileText } from 'lucide-react';
import type { Report } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Card } from '@/components/ui/Card';
import { EmptyState } from '@/components/ui/EmptyState';
import { ReportRow } from '@/features/_shared/ReportRow';
import { REPORT_QUEUE_LABEL, reportQueue, type ReportQueue } from '@/lib/domain';
import { fa } from '@/lib/digits';

/**
 * «گزارش‌ها» — the report queue, and only the report queue.
 *
 * Three piles from the author's seat: what waits on me, what waits on the
 * reviewer, what is approved. Visits are never listed here — a visit that
 * still needs its outcome is handled in «بازدیدهای من», and recording
 * «انجام شد» there is exactly what creates the draft that appears in this
 * queue. One object, one place to act on it; the two lifecycles stay
 * distinguishable everywhere else as separate status signals.
 */
export function ReportsScreen() {
  const { reports, user, cases, myVisits } = useApp();
  const myReports = useMemo(() => reports.filter((r) => r.author === user?.name), [reports, user?.name]);
  const [params, setParams] = useSearchParams();
  const requested = params.get('filter');
  const filter: ReportQueue = requested === 'review' || requested === 'approved' ? requested : 'action';

  const byRecent = (rows: Report[]) => [...rows].sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const revision = byRecent(myReports.filter((r) => r.status === 'revision'));
  const drafts = byRecent(myReports.filter((r) => r.status === 'draft'));
  const reviewing = byRecent(myReports.filter((r) => reportQueue(r) === 'review'));
  const approved = byRecent(myReports.filter((r) => reportQueue(r) === 'approved'));
  const actionCount = revision.length + drafts.length;
  const visibleCount = filter === 'action' ? actionCount : filter === 'review' ? reviewing.length : approved.length;

  const visitOf = (report: Report) => myVisits.find((v) => v.id === report.visitId || v.reportId === report.id);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="گزارش‌ها"
        subtitle={actionCount > 0 ? `${fa(actionCount)} گزارش نیازمند اقدام شما` : 'صف گزارش‌های بازدیدهای شما'}
      >
        <SegmentedControl<ReportQueue>
          className="mt-3"
          ariaLabel="صف گزارش‌ها"
          value={filter}
          onChange={(value) => setParams({ filter: value }, { replace: true })}
          items={[
            { value: 'action', label: 'اقدام من', count: actionCount },
            { value: 'review', label: 'در بررسی', count: reviewing.length },
            { value: 'approved', label: 'تأییدشده', count: approved.length },
          ]}
        />
      </ScreenHeader>

      <div className="space-y-6 pb-8 pt-5">
        {visibleCount === 0 && (
          <EmptyState
            icon={filter === 'action' ? <FileCheck2 className="size-6" /> : <FileText className="size-6" />}
            title={`گزارشی در «${REPORT_QUEUE_LABEL[filter]}» نیست`}
            description={
              filter === 'action'
                ? 'با ثبت نتیجه «انجام شد» برای هر بازدید، پیش‌نویس گزارش آن همین‌جا آماده تکمیل می‌شود.'
                : undefined
            }
          />
        )}
        {filter === 'action' && <>
          {revision.length > 0 && <ReportGroup title="نیازمند اصلاح" rows={revision} cases={cases} visitOf={visitOf} />}
          {drafts.length > 0 && <ReportGroup title="پیش‌نویس‌ها · منتظر تکمیل و ارسال" rows={drafts} cases={cases} visitOf={visitOf} />}
        </>}
        {filter === 'review' && reviewing.length > 0 && <ReportGroup title="ارسال‌شده برای بررسی" rows={reviewing} cases={cases} visitOf={visitOf} />}
        {filter === 'approved' && approved.length > 0 && <ReportGroup title="گزارش‌های تأییدشده" rows={approved} cases={cases} visitOf={visitOf} />}
      </div>
    </Screen>
  );
}

function ReportGroup({ title, rows, cases, visitOf }: {
  title: string;
  rows: Report[];
  cases: ReturnType<typeof useApp>['cases'];
  visitOf: (report: Report) => ReturnType<typeof useApp>['myVisits'][number] | undefined;
}) {
  return <section>
    <h2 className="mb-2.5 px-1 text-sm font-bold text-ink">{title}</h2>
    <Card padded={false}>
      {rows.map((report, i) => (
        <ReportRow
          key={report.id}
          report={report}
          familyName={cases.find((c) => c.id === report.caseId)?.familyName}
          visit={visitOf(report)}
          divider={i < rows.length - 1}
        />
      ))}
    </Card>
  </section>;
}
