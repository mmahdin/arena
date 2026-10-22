import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { CalendarClock, CheckCircle2, CircleAlert, ClipboardList, FileText, MapPin, Navigation, Phone, RotateCcw, XCircle } from 'lucide-react';
import type { VisitStatus } from '@/types';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { KeyValue, TextArea } from '@/components/ui/Field';
import { ChecklistRow } from '@/components/ui/Checklist';
import { ProgressLine } from '@/components/ui/Progress';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { EmptyState } from '@/components/ui/EmptyState';
import { VISIT_PURPOSE, VISIT_STATUS, REPORT_STATUS } from '@/lib/labels';
import { visitDisplayState } from '@/lib/domain';
import { reportForVisit, visitKindLabel, visitProgress, nextVisit } from '@/lib/visit-workflow';
import { fa, toLatinDigits } from '@/lib/digits';
import { faDate, faTimeRange } from '@/lib/jalali';

const OUTCOMES: { status: VisitStatus; icon: typeof CheckCircle2; description: string }[] = [
  { status: 'completed', icon: CheckCircle2, description: 'بازدید انجام شد؛ گزارش آن را تکمیل کنید' },
  { status: 'family-absent', icon: CircleAlert, description: 'خانواده در محل حاضر نبود' },
  { status: 'reschedule', icon: RotateCcw, description: 'نیاز به هماهنگی زمان تازه دارد' },
  { status: 'cancelled', icon: XCircle, description: 'بازدید لغو شده است' },
];

/** One visit, one outcome and one associated report. No parallel entry points. */
export function VisitDetailScreen() {
  const { visitId } = useParams();
  const navigate = useNavigate();
  const { visits, cases, reports, toast, toggleChecklistItem, recordVisitOutcome, createReportDraft } = useApp();
  const { fieldPrivacy } = useSettings();
  const visit = visits.find((v) => v.id === visitId);
  const family = cases.find((c) => c.id === visit?.caseId);
  const report = visit ? reportForVisit(visit, reports) : undefined;
  const [sheetOpen, setSheetOpen] = useState(false);
  const [selected, setSelected] = useState<VisitStatus>('completed');
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);

  const otherVisits = useMemo(
    () => visits.filter((v) => v.caseId === visit?.caseId && v.id !== visitId),
    [visits, visit?.caseId, visitId],
  );

  if (!visit) return (
    <Screen><ScreenHeader back title="بازدید" />
      <EmptyState title="بازدید یافت نشد" description="ممکن است برنامه بازدید تغییر کرده باشد." />
      <Button block onClick={() => navigate('/visits')}>بازگشت به بازدیدها</Button>
    </Screen>
  );

  const state = visitDisplayState(visit);
  const progress = visitProgress(visit, reports);
  const doneCount = visit.checklist.filter((item) => item.done).length;
  const closed = visit.status !== 'scheduled';
  const lockedReport = report && ['submitted', 'in-review', 'approved'].includes(report.status);
  const next = nextVisit(otherVisits);
  const contactHref = `tel:${toLatinDigits(visit.contactPhone).replace(/[^0-9+]/g, '')}`;
  const mapHref = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(`${visit.city} ${visit.address}`)}`;

  const openOutcomeSheet = () => {
    setNote(visit.outcome?.note ?? '');
    setSelected(closed ? visit.status : 'completed');
    setSheetOpen(true);
  };

  const submitOutcome = async () => {
    if (selected === 'completed' && visit.checklist.some((item) => item.required && !item.done)) {
      toast({ title: 'چک‌لیست کامل نیست', description: 'موارد الزامی را پیش از ثبت نتیجه تکمیل کنید.', tone: 'warning' });
      return;
    }
    setSaving(true);
    try {
      const result = await recordVisitOutcome(visit.id, selected, note.trim());
      setSheetOpen(false);
      if (selected === 'completed' && result.reportId) {
        toast({ title: 'بازدید ثبت شد', description: 'گزارش بازدید آماده تکمیل است.', tone: 'success' });
        navigate(`/reports/${result.reportId}`);
      } else {
        toast({ title: 'نتیجه بازدید ثبت شد', description: VISIT_STATUS[selected].label, tone: 'success' });
      }
    } finally {
      setSaving(false);
    }
  };

  const openReport = async () => {
    if (report) return navigate(`/reports/${report.id}`);
    if (saving || visit.status !== 'completed') return;
    setSaving(true);
    try {
      const type = visit.purpose === 'initial-assessment' ? 'home-assessment' : visit.purpose === 'periodic-monitoring' ? 'monitoring' : 'visit';
      const id = await createReportDraft({ type, caseId: visit.caseId, visitId: visit.id, childId: visit.childIds?.[0] });
      if (id) navigate(`/reports/${id}`);
    } finally {
      setSaving(false);
    }
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader back title={visitKindLabel(visit)}
        subtitle={<><PrivacyText value={family?.familyName ?? 'خانواده'} interactive={false} /> · {faDate(visit.date, { weekday: true })}</>} />

      <div className="pt-4">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={progress.tone}>{progress.label}</Badge>
          <span className="text-xs text-ink-3">{VISIT_PURPOSE[visit.purpose].label}</span>
        </div>
        {state === 'needs-outcome' && (
          <p className="mt-3 rounded-md bg-warning-soft px-3 py-2.5 text-sm leading-relaxed text-ink-2">
            زمان این بازدید گذشته است. نتیجه را ثبت کنید تا گزارش آن آماده شود.
          </p>
        )}

        <Section>
          <Card className="space-y-1">
            <KeyValue label="زمان" icon={<CalendarClock className="size-3.5 text-ink-3" />}
              value={`${faDate(visit.date)} · ${faTimeRange(visit.time, visit.durationMin)}`} />
            <KeyValue label="نشانی" icon={<MapPin className="size-3.5 text-ink-3" />}
              value={<PrivacyText value={`${visit.city}، ${visit.address}`} />} />
            <KeyValue label="هماهنگی با" icon={<Phone className="size-3.5 text-ink-3" />}
              value={<PrivacyText value={visit.contactName} />} />
            {!fieldPrivacy && !closed && <div className="flex gap-2 border-t border-line-soft pt-3">
              <a href={contactHref}
                className="press flex min-h-11 flex-1 items-center justify-center gap-2 rounded-md border border-line bg-sunken text-sm font-bold text-ink">
                <Phone className="size-4" aria-hidden /> تماس
              </a>
              <a href={mapHref} target="_blank" rel="noopener noreferrer"
                className="press flex min-h-11 flex-1 items-center justify-center gap-2 rounded-md border border-line bg-sunken text-sm font-bold text-ink">
                <Navigation className="size-4" aria-hidden /> مسیریابی
              </a>
            </div>}
          </Card>
        </Section>

        {family && <Section>
          <Link to={`/cases/${family.id}`} className="press flex min-h-14 items-center justify-between gap-3 rounded-lg border border-line bg-surface px-4 py-3 hover:border-line-strong">
            <span className="min-w-0">
              <span className="block text-sm font-bold text-ink"><PrivacyText value={family.familyName} interactive={false} /></span>
              <span className="mt-0.5 block text-xs text-ink-3">
                {fa(otherVisits.filter((v) => v.status === 'completed').length)} بازدید دیگر انجام‌شده
                {next && ` · بازدید بعدی: ${faDate(next.date, { year: false })}`}
              </span>
            </span>
            <span className="shrink-0 text-xs font-bold text-brand-ink">سوابق خانواده</span>
          </Link>
        </Section>}

        <Section>
          <h2 className="mb-2.5 text-sm font-bold text-ink">هدف بازدید</h2>
          <p className="text-sm leading-relaxed text-ink-2">{visit.summary}</p>
        </Section>

        {!closed && <Section>
          <h2 className="mb-2.5 text-sm font-bold text-ink">چک‌لیست بازدید</h2>
          <Card>
            <ProgressLine done={doneCount} total={visit.checklist.length}
              unit={doneCount === visit.checklist.length ? 'همه موارد بررسی شد' : 'موارد بررسی‌شده'}
              tone={doneCount === visit.checklist.length ? 'success' : 'brand'} />
            <div className="-mx-1 mt-2 divide-y divide-line-soft">
              {visit.checklist.map((item) => (
                <ChecklistRow key={item.id} checked={item.done} required={item.required}
                  label={item.label} onChange={() => toggleChecklistItem(visit.id, item.id)} />
              ))}
            </div>
          </Card>
        </Section>}

        {closed && <Section>
          <h2 className="mb-2.5 text-sm font-bold text-ink">نتیجه بازدید</h2>
          <Card>
            <p className="text-sm font-bold text-ink">{VISIT_STATUS[visit.status].label}</p>
            <p className="mt-1 text-sm leading-relaxed text-ink-2">{visit.outcome?.note || 'توضیحی ثبت نشده است.'}</p>
            {!lockedReport && <button type="button" onClick={openOutcomeSheet}
              className="press mt-2 min-h-10 text-sm font-semibold text-brand-ink">ویرایش نتیجه</button>}
          </Card>
        </Section>}

        {closed && visit.checklist.length > 0 && <details className="group mt-6 rounded-lg border border-line bg-surface">
          <summary className="press flex min-h-12 cursor-pointer list-none items-center justify-between px-4 py-3 text-sm font-bold text-ink [&::-webkit-details-marker]:hidden">
            چک‌لیست ثبت‌شده <span className="text-xs font-medium text-ink-3">{fa(doneCount)} از {fa(visit.checklist.length)} مورد</span>
          </summary>
          <div className="divide-y divide-line-soft border-t border-line-soft px-4">
            {visit.checklist.map((item) => (
              <ChecklistRow key={item.id} checked={item.done} required={item.required}
                label={item.label} onChange={() => {}} disabled />
            ))}
          </div>
        </details>}

        {visit.status === 'completed' && report && <Section>
          <Link to={`/reports/${report.id}`} className="press flex min-h-16 items-center gap-3 rounded-lg border border-line bg-surface px-4 py-3 hover:border-line-strong">
            <FileText className="size-5 shrink-0 text-brand" aria-hidden />
            <span className="min-w-0 flex-1">
              <span className="block text-sm font-bold text-ink">گزارش این بازدید</span>
              <span className="mt-0.5 block text-xs text-ink-3">{REPORT_STATUS[report.status].label}</span>
            </span>
            <span className="text-xs font-bold text-brand-ink">مشاهده</span>
          </Link>
        </Section>}
      </div>

      {(!closed || visit.status === 'completed') && <ActionFooter>
        {closed ? <Button block loading={saving} onClick={() => { void openReport(); }} iconStart={<FileText className="size-4" />}>
          {report ? (report.status === 'draft' || report.status === 'revision' ? 'تکمیل گزارش بازدید' : 'مشاهده گزارش بازدید') : 'نوشتن گزارش بازدید'}
        </Button> : <Button block onClick={openOutcomeSheet} iconStart={<ClipboardList className="size-4" />}>
          ثبت نتیجه بازدید
        </Button>}
      </ActionFooter>}

      <Sheet open={sheetOpen} onClose={() => setSheetOpen(false)} title="نتیجه بازدید"
        description={`${family?.familyName ?? ''} · ${faDate(visit.date, { year: false })}`}
        footer={<div className="flex gap-2.5">
          <Button variant="secondary" className="flex-1" onClick={() => setSheetOpen(false)}>انصراف</Button>
          <Button className="flex-1" loading={saving} onClick={() => { void submitOutcome(); }}>ثبت نتیجه</Button>
        </div>}
      >
        <div className="space-y-2 pb-4">
          {OUTCOMES.map((o) => <SheetOption key={o.status} icon={<o.icon className="size-[18px]" />}
            title={VISIT_STATUS[o.status].label} description={o.description}
            selected={selected === o.status} onClick={() => setSelected(o.status)} />)}
          <div className="pt-2">
            <TextArea label="توضیح کوتاه" rows={3} value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={selected === 'completed' ? 'خلاصه مشاهدات…' : 'دلیل انجام نشدن بازدید…'}
              hint="توضیح در سوابق بازدید نگهداری می‌شود." />
          </div>
        </div>
      </Sheet>
    </Screen>
  );
}
