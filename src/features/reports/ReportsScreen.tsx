import { useMemo } from 'react';
import { useSearchParams } from 'react-router-dom';
import { FileText } from 'lucide-react';
import type { Report } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Card } from '@/components/ui/Card';
import { EmptyState } from '@/components/ui/EmptyState';
import { ReportRow } from '@/features/_shared/ReportRow';
import { VisitRow } from '@/features/_shared/VisitRow';
import { reportForVisit } from '@/lib/visit-workflow';
import { fa } from '@/lib/digits';

type Filter = 'action' | 'review' | 'approved';
const isAction = (r: Report) => r.status === 'draft' || r.status === 'revision';
const isReview = (r: Report) => r.status === 'submitted' || r.status === 'in-review';

/** Reports are a work queue: act, wait for review, or view approved results. */
export function ReportsScreen() {
  const { reports, user, cases, visits } = useApp();
  const myReports = useMemo(() => reports.filter((r) => r.author === user?.name), [reports, user?.name]);
  const [params, setParams] = useSearchParams();
  const requested = params.get('filter');
  const filter: Filter = requested === 'review' || requested === 'approved' ? requested : 'action';

  const missing = visits.filter((v) => v.status === 'completed' && !reportForVisit(v, reports));
  const revision = myReports.filter((r) => r.status === 'revision').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const drafts = myReports.filter((r) => r.status === 'draft').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const reviewing = myReports.filter(isReview).sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const approved = myReports.filter((r) => r.status === 'approved').sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  const empty = filter === 'action' ? revision.length + drafts.length + missing.length === 0 : filter === 'review' ? reviewing.length === 0 : approved.length === 0;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="گزارش‌ها"
        subtitle={`${fa(myReports.filter(isAction).length + missing.length)} مورد نیازمند اقدام شما`}
      >
        <SegmentedControl<Filter>
          className="mt-3"
          ariaLabel="وضعیت گزارش‌ها"
          value={filter}
          onChange={(value) => setParams({ filter: value }, { replace: true })}
          items={[
            { value: 'action', label: 'اقدام من', count: revision.length + drafts.length + missing.length },
            { value: 'review', label: 'در بررسی', count: reviewing.length },
            { value: 'approved', label: 'تأییدشده', count: approved.length },
          ]}
        />
      </ScreenHeader>

      <div className="space-y-6 pb-8 pt-5">
        {empty && <EmptyState icon={<FileText className="size-6" />} title="گزارشی در این وضعیت نیست" />}
        {filter === 'action' && <>
          {revision.length > 0 && <ReportGroup title="نیازمند اصلاح" rows={revision} cases={cases} />}
          {missing.length > 0 && <section>
            <h2 className="mb-2.5 px-1 text-sm font-bold text-ink">بازدیدهای بدون گزارش</h2>
            <Card padded={false}>
              {missing.map((visit, i) => <VisitRow key={visit.id} visit={visit}
                family={cases.find((c) => c.id === visit.caseId)} reports={reports}
                divider={i < missing.length - 1} />)}
            </Card>
          </section>}
          {drafts.length > 0 && <ReportGroup title="پیش‌نویس‌ها · منتظر تکمیل و ارسال" rows={drafts} cases={cases} />}
        </>}
        {filter === 'review' && reviewing.length > 0 && <ReportGroup title="ارسال‌شده برای بررسی" rows={reviewing} cases={cases} />}
        {filter === 'approved' && approved.length > 0 && <ReportGroup title="گزارش‌های تأییدشده" rows={approved} cases={cases} />}
      </div>
    </Screen>
  );
}

function ReportGroup({ title, rows, cases }: {
  title: string;
  rows: Report[];
  cases: ReturnType<typeof useApp>['cases'];
}) {
  return <section>
    <h2 className="mb-2.5 px-1 text-sm font-bold text-ink">{title}</h2>
    <Card padded={false}>
      {rows.map((report, i) => (
        <ReportRow key={report.id} report={report} familyName={cases.find((c) => c.id === report.caseId)?.familyName} divider={i < rows.length - 1} />
      ))}
    </Card>
  </section>;
}
