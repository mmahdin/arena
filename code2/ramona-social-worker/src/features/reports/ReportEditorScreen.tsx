import { useEffect, useMemo, useState } from 'react';
import { useNavigate, useParams } from 'react-router-dom';
import { CheckCircle2, CloudUpload, Lock, MessageSquareQuote, Paperclip, Plus, Save } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { IconButton } from '@/components/ui/IconButton';
import { Card, ToneCard } from '@/components/ui/Card';
import { ProgressBar } from '@/components/ui/Progress';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile, REPORT_TYPE_ICON } from '@/components/ui/DomainIcon';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { ReportFieldControl } from './ReportFieldControl';
import { REPORT_STATUS, REPORT_TYPE } from '@/lib/labels';
import { reportCompletion } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { relativeTime } from '@/lib/jalali';

/**
 * Report editor.
 *
 * One generic editor for every professional form. Completion is always
 * visible, drafts save themselves, and the review conversation (supervisor
 * comments) sits at the top of the returned report — the social worker should
 * never have to hunt for the reason it came back.
 */
export function ReportEditorScreen() {
  const { reportId } = useParams();
  const navigate = useNavigate();
  const { reports, cases, editReportField, editReportMeta, submitReport, toast } = useApp();

  const report = reports.find((r) => r.id === reportId);
  const caseFile = cases.find((c) => c.id === report?.caseId);
  const [saving, setSaving] = useState(false);
  const [menuOpen, setMenuOpen] = useState(false);
  const [tick, setTick] = useState(0);

  // Keeps the "saved …" label honest as time passes.
  useEffect(() => {
    const t = window.setInterval(() => setTick((x) => x + 1), 30_000);
    return () => window.clearInterval(t);
  }, []);

  const completion = useMemo(() => (report ? reportCompletion(report) : null), [report, tick]);

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
        subtitle={
          <span className="flex items-center gap-1.5">
            <span className="tabular-nums">{report.caseCode}</span>
            {caseFile && <span>· {caseFile.familyName}</span>}
          </span>
        }
        actions={
          <>
            <Badge tone={status.tone}>{status.label}</Badge>
            <IconButton label="گزینه‌های گزارش" onClick={() => setMenuOpen(true)}>
              <Paperclip className="size-[18px]" strokeWidth={2.1} />
            </IconButton>
          </>
        }
      />

      <div className="pt-3">
        {/* autosave line */}
        <div className="flex items-center justify-between px-1 text-2xs text-ink-3">
          <span className="flex items-center gap-1.5">
            <CloudUpload className="size-3.5 text-success" strokeWidth={2.4} aria-hidden />
            ذخیره خودکار · {relativeTime(report.updatedAt)}
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

      <Section>
        <Card className="space-y-3">
          <div className="flex items-center gap-3">
            <IconTile icon={REPORT_TYPE_ICON[report.type]} tone={REPORT_TYPE[report.type].tone} />
            <div className="min-w-0">
              <p className="truncate text-base font-bold text-ink">{REPORT_TYPE[report.type].label}</p>
              <p className="text-xs text-ink-3">نویسنده: {report.author}</p>
            </div>
          </div>
          {report.summary ? (
            <p className="border-s-2 border-line-strong ps-3 text-sm leading-relaxed text-ink-2">{report.summary}</p>
          ) : (
            <p className="text-sm text-ink-3">خلاصه‌ای برای این گزارش ثبت نشده است.</p>
          )}
        </Card>
      </Section>

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

      {report.sections.map((section, index) => {
        const filled = section.fields.filter((f) =>
          f.kind === 'checklist' ? f.value.length > 0 : f.kind === 'scale' ? f.value > 0 : String(f.value).trim().length > 0,
        ).length;
        const complete = filled === section.fields.length;

        return (
          <Section key={section.id}>
            <div className="mb-2.5 flex items-center justify-between gap-3 px-1">
              <h2 className="flex items-center gap-2 text-md font-bold text-ink">
                <span className="grid size-6 place-items-center rounded-sm bg-brand-soft text-2xs font-extrabold text-brand-ink tabular-nums">
                  {fa(index + 1)}
                </span>
                {section.title}
              </h2>
              {complete ? (
                <CheckCircle2 className="size-[18px] text-success" strokeWidth={2.4} aria-label="تکمیل شده" />
              ) : (
                <span className="text-2xs text-ink-3 tabular-nums">
                  {fa(filled)}/{fa(section.fields.length)}
                </span>
              )}
            </div>
            <Card className="space-y-5">
              {section.fields.map((field) => (
                <ReportFieldControl
                  key={field.id}
                  field={field}
                  disabled={locked}
                  onChange={(value) => editReportField(report.id, section.id, field.id, value)}
                />
              ))}
            </Card>
          </Section>
        );
      })}

      <Section>
        <div className="mb-2.5 flex items-center justify-between px-1">
          <h2 className="text-md font-bold text-ink">پیوست‌ها</h2>
          <Button
            size="sm"
            variant="ghost"
            iconStart={<Plus className="size-4" />}
            disabled={locked}
            onClick={() => toast({ title: 'افزودن پیوست', description: 'انتخاب فایل در فاز بعدی فعال می‌شود.', tone: 'info' })}
          >
            افزودن
          </Button>
        </div>
        {report.attachments.length === 0 ? (
          <Card className="border-dashed">
            <p className="text-sm text-ink-3">پیوستی اضافه نشده است.</p>
          </Card>
        ) : (
          <Card padded={false} className="divide-y divide-line-soft">
            {report.attachments.map((a) => (
              <div key={a.id} className="flex items-center gap-3 px-4 py-3">
                <IconTile icon={Paperclip} tone="neutral" size="sm" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-base font-medium text-ink">{a.name}</p>
                  <p className="text-xs text-ink-3">{a.size}</p>
                </div>
              </div>
            ))}
          </Card>
        )}
      </Section>

      {!locked && (
        <ActionFooter hint="پس از ارسال، گزارش در فهرست «در بازبینی» قرار می‌گیرد.">
          <Button
            variant="secondary"
            className="flex-1"
            iconStart={<Save className="size-4" />}
            onClick={() => toast({ title: 'پیش‌نویس ذخیره شد', tone: 'success' })}
          >
            ذخیره پیش‌نویس
          </Button>
          <Button className="flex-1" loading={saving} onClick={onSubmit} iconStart={<CloudUpload className="size-4" />}>
            ارسال برای بازبینی
          </Button>
        </ActionFooter>
      )}

      <Sheet open={menuOpen} onClose={() => setMenuOpen(false)} title="گزینه‌های گزارش">
        <div className="space-y-2 pb-4">
          {caseFile && (
            <SheetOption
              icon={<Paperclip className="size-[18px]" />}
              title="مشاهده پرونده"
              description={caseFile.familyName}
              onClick={() => {
                setMenuOpen(false);
                navigate(`/cases/${caseFile.id}`);
              }}
            />
          )}
          <SheetOption
            icon={<Save className="size-[18px]" />}
            title="ذخیره پیش‌نویس"
            onClick={() => {
              setMenuOpen(false);
              editReportMeta(report.id, { summary: report.summary });
              toast({ title: 'پیش‌نویس ذخیره شد', tone: 'success' });
            }}
          />
        </div>
      </Sheet>
    </Screen>
  );
}
