import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { FileText, FolderHeart, Plus } from 'lucide-react';
import type { ReportStatus } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { Card } from '@/components/ui/Card';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { EmptyState } from '@/components/ui/EmptyState';
import { ReportRow } from '@/features/_shared/ReportRow';
import { REPORT_STATUS } from '@/lib/labels';
import { fa } from '@/lib/digits';

type Filter = ReportStatus | 'all';

const FILTERS: { value: Filter; label: string }[] = [
  { value: 'all', label: 'همه' },
  { value: 'draft', label: 'پیش‌نویس' },
  { value: 'revision', label: 'نیازمند اصلاح' },
  { value: 'in-review', label: 'در بازبینی' },
  { value: 'submitted', label: 'ارسال‌شده' },
  { value: 'approved', label: 'تأییدشده' },
];

/**
 * Reports.
 *
 * The report lifecycle is the contract between the social worker and the
 * supervisor, so the list is organised by *state* — draft, returned, under
 * review, approved — not by date. State tells you what to do next.
 */
export function ReportsScreen() {
  const { reports, cases, toast, createReportDraft } = useApp();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>('all');
  const [pickerOpen, setPickerOpen] = useState(false);

  const counts = useMemo(() => {
    const base: Record<string, number> = { all: reports.length };
    for (const r of reports) base[r.status] = (base[r.status] ?? 0) + 1;
    return base;
  }, [reports]);

  const visible = useMemo(() => {
    const rows = filter === 'all' ? reports : reports.filter((r) => r.status === filter);
    return rows.slice().sort((a, b) => b.updatedAt.localeCompare(a.updatedAt));
  }, [reports, filter]);

  const draftCount = counts.draft ?? 0;
  const revisionCount = counts.revision ?? 0;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="گزارش‌ها"
        subtitle={`${fa(draftCount)} پیش‌نویس${revisionCount > 0 ? ` · ${fa(revisionCount)} نیازمند اصلاح` : ''}`}
        actions={
          <IconButton label="گزارش جدید" onClick={() => setPickerOpen(true)}>
            <Plus className="size-[22px]" strokeWidth={2.2} />
          </IconButton>
        }
      >
        <div className="mt-3">
          <ChipRail flush>
            {FILTERS.map((f) => (
              <Chip key={f.value} selected={filter === f.value} count={counts[f.value] ?? 0} onClick={() => setFilter(f.value)}>
                {f.label}
              </Chip>
            ))}
          </ChipRail>
        </div>
      </ScreenHeader>

      <div className="pt-4 pb-8">
        {visible.length === 0 ? (
          <EmptyState
            icon={<FileText className="size-6" />}
            title="گزارشی در این وضعیت نیست"
            description="با ثبت نتیجه یک بازدید، پیش‌نویس گزارش به‌صورت خودکار ساخته می‌شود."
          />
        ) : (
          <Card padded={false} className="divide-y divide-line-soft">
            {visible.map((r) => (
              <ReportRow key={r.id} report={r} />
            ))}
          </Card>
        )}

        <p className="mt-5 px-1 text-2xs leading-relaxed text-ink-3">
          گزارش‌های ارسالی برای بازبینی سرپرست ارسال می‌شوند و ممکن است برای تکمیل بازگردانده شوند.
          نظر نهایی درباره صلاحیت و جایگذاری در صلاحیت کمیته تخصصی و مراجع ذی‌صلاح است.
        </p>
      </div>

      <Sheet
        open={pickerOpen}
        onClose={() => setPickerOpen(false)}
        title="گزارش جدید"
        description="پرونده مورد نظر را انتخاب کنید؛ پیش‌نویس بر اساس قالب مناسب ساخته می‌شود."
        tall
      >
        <div className="space-y-2 pb-4">
          {cases
            .filter((c) => c.stage !== 'closed')
            .map((c) => (
              <SheetOption
                key={c.id}
                icon={<FolderHeart className="size-[18px]" />}
                title={c.familyName}
                description={`${c.code} · ${REPORT_STATUS.draft.label} · ${
                  c.stage === 'initial-assessment' ? 'قالب ارزیابی منزل' : c.stage === 'post-placement' ? 'قالب پایش کودک' : 'قالب گزارش بازدید'
                }`}
                onClick={async () => {
                  setPickerOpen(false);
                  const id = await createReportDraft({
                    type:
                      c.stage === 'initial-assessment'
                        ? 'home-assessment'
                        : c.stage === 'post-placement'
                          ? 'monitoring'
                          : 'visit',
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
