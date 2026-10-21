import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { CheckCircle2, ChevronDown, CloudUpload, Lock, MessageSquareQuote, Paperclip } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card, ToneCard } from '@/components/ui/Card';
import { ProgressBar } from '@/components/ui/Progress';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile } from '@/components/ui/DomainIcon';
import { ReportFieldControl } from './ReportFieldControl';
import { REPORT_STATUS } from '@/lib/labels';
import { reportCompletion } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { relativeTime } from '@/lib/jalali';

/**
 * Report editor.
 *
 * One generic editor for every professional form. Completion is always
 * visible, fields update in the session, and the review conversation (supervisor
 * comments) sits at the top of the returned report — the social worker should
 * never have to hunt for the reason it came back.
 */
export function ReportEditorScreen() {
  const { reportId } = useParams();
  const navigate = useNavigate();
  const { reports, cases, editReportField, submitReport, toast } = useApp();

  const report = reports.find((r) => r.id === reportId);
  const caseFile = cases.find((c) => c.id === report?.caseId);
  const [saving, setSaving] = useState(false);
  const [activeSectionId, setActiveSectionId] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  // Keeps the "saved …" label honest as time passes.
  useEffect(() => {
    const t = window.setInterval(() => setTick((x) => x + 1), 30_000);
    return () => window.clearInterval(t);
  }, []);

  const completion = useMemo(() => (report ? reportCompletion(report) : null), [report, tick]);

  // Open one meaningful section at a time, without jumping away while typing.
  useEffect(() => {
    if (!report) return;
    const firstIncomplete = report.sections.find((section) => section.fields.some((f) =>
      f.kind === 'checklist' ? f.value.length === 0 : f.kind === 'scale' ? f.value <= 0 : !String(f.value).trim(),
    ));
    setActiveSectionId(firstIncomplete?.id ?? report.sections[0]?.id ?? null);
    // The active section should change when the report changes, not on each field edit.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [report?.id]);

  if (!report || !completion) {
    return (
      <Screen>
        <ScreenHeader back title="گزارش" />
        <EmptyState title="گزارش یافت نشد" description="ممکن است این گزارش حذف یا بایگانی شده باشد." />
        <Button block onClick={() => navigate('/reports')}>
          بازگشت به گزارش‌ها
        </Button>
      </Screen>
    );
  }

  const status = REPORT_STATUS[report.status];
  const locked = report.status === 'approved' || report.status === 'in-review' || report.status === 'submitted';
  const lastComment = report.comments[report.comments.length - 1];

  const onSubmit = async () => {
    if (completion.pct < 100) {
      toast({
        title: 'گزارش کامل نیست',
        description: `${fa(100 - completion.pct)}٪ از بخش‌ها باقی مانده است.`,
        tone: 'warning',
      });
      return;
    }
    setSaving(true);
    await submitReport(report.id);
    setSaving(false);
    toast({
      title: 'گزارش برای بازبینی ارسال شد',
      description: 'سرپرست واحد آن را بررسی خواهد کرد.',
      tone: 'success',
      action: { label: 'مشاهده گزارش‌ها', onClick: () => navigate('/reports') },
    });
    navigate('/reports');
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title={report.title}
        subtitle={caseFile ? (
          <Link to={`/cases/${caseFile.id}`} className="press hover:underline underline-offset-4">
            {caseFile.familyName} · {report.caseCode}
          </Link>
        ) : report.caseCode}
        actions={<Badge tone={status.tone}>{status.label}</Badge>}
      />

      <div className="pt-3">
        {/* This demo has an in-memory store, not durable server-side autosave. */}
        <div className="flex items-center justify-between px-1 text-2xs text-ink-3">
          <span className="flex items-center gap-1.5">
            <CloudUpload className="size-3.5 text-success" strokeWidth={2.4} aria-hidden />
            ثبت در این نشست · {relativeTime(report.updatedAt)}
          </span>
          <span className="tabular-nums">
            {fa(completion.filled)} از {fa(completion.total)} فیلد
          </span>
        </div>

        <ProgressBar value={completion.pct} tone={completion.pct === 100 ? 'success' : 'brand'} className="mt-2" />
      </div>

      {locked && (
        <Section>
          <ToneCard tone="info" className="flex items-start gap-3">
            <Lock className="mt-0.5 size-5 shrink-0" strokeWidth={2.25} aria-hidden />
            <div>
              <p className="text-sm font-bold">این گزارش قفل شده است</p>
              <p className="mt-1 text-xs leading-relaxed opacity-85">
                پس از ارسال برای بازبینی، ویرایش ممکن نیست. در صورت بازگشت گزارش، ویرایش دوباره فعال می‌شود.
              </p>
            </div>
          </ToneCard>
        </Section>
      )}

      {report.status === 'revision' && lastComment && (
        <Section>
          <ToneCard tone="warning">
            <div className="flex items-start gap-3">
              <IconTile icon={MessageSquareQuote} tone="warning" />
              <div className="min-w-0">
                <p className="text-sm font-bold">درخواست اصلاح — {lastComment.author}</p>
                <p className="text-2xs opacity-80">{lastComment.role} · {relativeTime(lastComment.at)}</p>
                <p className="mt-2 text-sm leading-relaxed text-ink-2">{lastComment.body}</p>
              </div>
            </div>
          </ToneCard>
        </Section>
      )}

      {report.status === 'approved' && lastComment && (
        <Section>
          <ToneCard tone="success" className="flex items-start gap-3">
            <CheckCircle2 className="mt-0.5 size-5 shrink-0" strokeWidth={2.25} aria-hidden />
            <div>
              <p className="text-sm font-bold">تأیید سرپرست</p>
              <p className="mt-1 text-sm leading-relaxed opacity-90">{lastComment.body}</p>
            </div>
          </ToneCard>
        </Section>
      )}

      {report.summary && (
        <Section>
          <div className="rounded-md bg-sunken px-4 py-3">
            <p className="mb-1 text-xs font-bold text-ink-3">خلاصه ثبت‌شده</p>
            <p className="text-sm leading-relaxed text-ink-2">{report.summary}</p>
          </div>
        </Section>
      )}

      {report.sections.length === 0 && (
        <Section>
          <Card className="border-dashed">
            <p className="text-sm leading-relaxed text-ink-3">
              این گزارش از یک بازدید ثبت شده و هنوز بخشی ندارد. بخش‌های ارزیابی در فاز بعدی به‌صورت قالب‌های
              تخصصی اضافه می‌شوند.
            </p>
          </Card>
        </Section>
      )}

      {report.sections.length > 0 && (
        <Section>
          <h2 className="mb-2.5 text-md font-bold text-ink">بخش‌های گزارش</h2>
          <div className="space-y-2">
            {report.sections.map((section, index) => {
              const filled = section.fields.filter((f) =>
                f.kind === 'checklist' ? f.value.length > 0 : f.kind === 'scale' ? f.value > 0 : String(f.value).trim().length > 0,
              ).length;
              const complete = filled === section.fields.length;
              const expanded = activeSectionId === section.id;

              return (
                <div key={section.id} className="overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
                  <button
                    type="button"
                    onClick={() => setActiveSectionId(expanded ? null : section.id)}
                    aria-expanded={expanded}
                    aria-controls={`report-section-${section.id}`}
                    className="press flex min-h-14 w-full items-center gap-3 px-4 py-3 text-start"
                  >
                    <span className="grid size-7 shrink-0 place-items-center rounded-md bg-brand-soft text-sm font-extrabold text-brand-ink tabular-nums">
                      {fa(index + 1)}
                    </span>
                    <span className="min-w-0 flex-1 text-sm font-bold leading-relaxed text-ink">{section.title}</span>
                    {complete ? <CheckCircle2 className="size-4 shrink-0 text-success" aria-label="تکمیل‌شده" /> : (
                      <span className="shrink-0 text-xs text-ink-3 tabular-nums">{fa(filled)}/{fa(section.fields.length)}</span>
                    )}
                    <ChevronDown className={`size-4 shrink-0 text-ink-3 transition-transform ${expanded ? 'rotate-180' : ''}`} aria-hidden />
                  </button>
                  {expanded && (
                    <div id={`report-section-${section.id}`} className="space-y-5 border-t border-line-soft px-4 pb-5 pt-4">
                      {section.fields.map((field) => (
                        <ReportFieldControl
                          key={field.id}
                          field={field}
                          disabled={locked}
                          onChange={(value) => editReportField(report.id, section.id, field.id, value)}
                        />
                      ))}
                    </div>
                  )}
                </div>
              );
            })}
          </div>
        </Section>
      )}

      {report.attachments.length > 0 && (
        <Section>
          <h2 className="mb-2.5 text-md font-bold text-ink">پیوست‌های موجود</h2>
          <Card padded={false} className="divide-y divide-line-soft">
            {report.attachments.map((a) => (
              <div key={a.id} className="flex items-center gap-3 px-4 py-3">
                <IconTile icon={Paperclip} tone="neutral" size="sm" />
                <div className="min-w-0 flex-1">
                  <p className="text-sm font-medium text-ink">{a.name}</p>
                  <p className="text-xs text-ink-3">{a.size}</p>
                </div>
              </div>
            ))}
          </Card>
        </Section>
      )}

      {!locked && (
        <ActionFooter hint="تغییرات فقط تا پایان این نشست نگهداری می‌شوند.">
          <Button block loading={saving} onClick={onSubmit} iconStart={<CloudUpload className="size-4" />}>
            ارسال برای بازبینی
          </Button>
        </ActionFooter>
      )}
    </Screen>
  );
}
