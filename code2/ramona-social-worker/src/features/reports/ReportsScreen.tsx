import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { FileText, FolderHeart, Plus } from 'lucide-react';
import type { Report } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Card } from '@/components/ui/Card';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { EmptyState } from '@/components/ui/EmptyState';
import { ReportRow } from '@/features/_shared/ReportRow';
import { fa } from '@/lib/digits';

type Filter = 'all' | 'action' | 'sent';
const needsAction = (r: Report) => r.status === 'draft' || r.status === 'revision';

/** Reports are their own workspace, with one clear actionable queue. */
export function ReportsScreen() {
  const { reports, cases, user, toast, createReportDraft } = useApp();
  const myReports = useMemo(() => reports.filter((r) => r.author === user?.name), [reports, user?.name]);
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>('all');
  const [pickerOpen, setPickerOpen] = useState(false);

  const counts = useMemo(() => ({
    all: myReports.length,
    action: myReports.filter(needsAction).length,
    sent: myReports.filter((r) => !needsAction(r)).length,
  }), [myReports]);

  const visible = useMemo(() => {
    const rows = myReports.filter((r) => filter === 'all' || (filter === 'action' ? needsAction(r) : !needsAction(r)));
    return rows.slice().sort((a, b) => {
      if (filter !== 'sent') {
        const weight = (r: Report) => r.status === 'revision' ? 0 : r.status === 'draft' ? 1 : 2;
        if (weight(a) !== weight(b)) return weight(a) - weight(b);
      }
      return b.updatedAt.localeCompare(a.updatedAt);
    });
  }, [myReports, filter]);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="گزارش‌ها"
        subtitle={counts.action > 0 ? `${fa(counts.action)} گزارش در انتظار اقدام شما` : 'گزارش‌های ثبت‌شده شما'}
        actions={
          <IconButton label="گزارش جدید" onClick={() => setPickerOpen(true)}>
            <Plus className="size-[22px]" strokeWidth={2.2} />
          </IconButton>
        }
      >
        <SegmentedControl<Filter>
          className="mt-3"
          ariaLabel="فیلتر گزارش‌ها"
          value={filter}
          onChange={setFilter}
          items={[
            { value: 'all', label: 'همه', count: counts.all },
            { value: 'action', label: 'نیازمند اقدام', count: counts.action },
            { value: 'sent', label: 'ارسال‌شده', count: counts.sent },
          ]}
        />
      </ScreenHeader>

      <div className="pt-4 pb-8">
        {visible.length === 0 ? (
          <EmptyState icon={<FileText className="size-6" />} title="گزارشی در این بخش نیست" />
        ) : (
          <Card padded={false}>
            {visible.map((r, i) => <ReportRow key={r.id} report={r} divider={i < visible.length - 1} />)}
          </Card>
        )}
      </div>

      <Sheet
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        title="گزارش جدید"
        description="پرونده مربوط را انتخاب کنید."
        tall
      >
        <div className="space-y-2 pb-4">
          {cases.filter((c) => c.stage !== 'closed').map((c) => (
            <SheetOption
              key={c.id}
              icon={<FolderHeart className="size-[18px]" />}
              title={c.familyName}
              description={c.code}
              onClick={async () => {
                setPickerOpen(false);
                const id = await createReportDraft({
                  type: c.stage === 'initial-assessment' ? 'home-assessment' : c.stage === 'post-placement' ? 'monitoring' : 'visit',
                  caseId: c.id,
                  childId: c.children[0]?.id,
                });
                if (id) {
                  toast({ title: 'پیش‌نویس گزارش ایجاد شد', tone: 'success' });
                  navigate(`/reports/${id}`);
                }
              }}
            />
          ))}
        </div>
      </Sheet>
    </Screen>
  );
}
