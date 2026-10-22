import { useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { CalendarClock, CheckCircle2, CircleAlert, ClipboardList, FileText, Inbox, MapPin, Navigation, Phone, RotateCcw, ShieldCheck, Undo2, XCircle } from 'lucide-react';
import type { Visit, VisitStatus } from '@/types';
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
import { ClaimVisitSheet } from '@/features/_shared/ClaimVisitSheet';
import { VISIT_PURPOSE, VISIT_STATUS, REPORT_STATUS, CASE_TYPE, PRIORITY } from '@/lib/labels';
import { isPoolVisit, scheduleConflicts, visitDisplayState, visitsOnSameDay } from '@/lib/domain';
import { reportForVisit, visitKindLabel, visitStatusMeta, nextVisit } from '@/lib/visit-workflow';
import { fa, toLatinDigits } from '@/lib/digits';
import { deadlineLabel, faDate, faDuration, faTime, faTimeRange } from '@/lib/jalali';

const OUTCOMES: { status: VisitStatus; icon: typeof CheckCircle2; description: string }[] = [
  { status: 'completed', icon: CheckCircle2, description: 'بازدید انجام شد؛ گزارش آن را تکمیل کنید' },
  { status: 'family-absent', icon: CircleAlert, description: 'خانواده در محل حاضر نبود' },
  { status: 'reschedule', icon: RotateCcw, description: 'نیاز به هماهنگی زمان تازه دارد' },
  { status: 'cancelled', icon: XCircle, description: 'بازدید لغو شده است' },
];

/**
 * One visit, one screen — for every state the visit can be in:
 *
 * - a pool visit: a limited, identity-free preview whose only action is
 *   claiming. The moment it is claimed this same screen becomes the full
 *   visit below, because the visit itself never changed — only its owner.
 * - my visit: plan → checklist → outcome → its report. The report is a
 *   separate object with its own status and its own screen; here it is just
 *   linked, never duplicated.
 */
export function VisitDetailScreen() {
  const { visitId } = useParams();
  const navigate = useNavigate();
  const { visits, myVisits, cases, reports, user, toast, toggleChecklistItem, recordVisitOutcome, createReportDraft, releaseVisit } = useApp();
  const { fieldPrivacy } = useSettings();
  const visit = visits.find((v) => v.id === visitId);
  const family = cases.find((c) => c.id === visit?.caseId);
  const report = visit ? reportForVisit(visit, reports) : undefined;
  const [sheetOpen, setSheetOpen] = useState(false);
  const [releaseOpen, setReleaseOpen] = useState(false);
  const [selected, setSelected] = useState<VisitStatus>('completed');
  const [note, setNote] = useState('');
  const [saving, setSaving] = useState(false);

  const otherVisits = useMemo(
    () => myVisits.filter((v) => v.caseId === visit?.caseId && v.id !== visitId),
    [myVisits, visit?.caseId, visitId],
  );

  if (!visit) return (
    <Screen><ScreenHeader back title="بازدید" />
      <EmptyState title="بازدید یافت نشد" description="ممکن است برنامه بازدید تغییر کرده باشد." />
      <Button block onClick={() => navigate('/visits')}>بازگشت به بازدیدها</Button>
    </Screen>
  );

  if (isPoolVisit(visit)) {
    return <PoolVisitDetail visit={visit} />;
  }

  if (user && visit.assignedToId !== user.id) return (
    <Screen><ScreenHeader back title="بازدید" />
      <EmptyState title="این بازدید در حوزه مسئولیت شما نیست" description="بازدید به مددکار دیگری محول شده است." />
      <Button block onClick={() => navigate('/visits')}>بازگشت به بازدیدهای من</Button>
    </Screen>
  );

  const state = visitDisplayState(visit);
  const status = visitStatusMeta(visit);
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

  const confirmRelease = async () => {
    setSaving(true);
    try {
      const released = await releaseVisit(visit.id);
      setReleaseOpen(false);
      if (released) {
        toast({
          title: 'بازدید به مخزن آزاد بازگشت',
          description: 'مددکاران دیگر می‌توانند مسئولیت آن را بپذیرند.',
          tone: 'success',
        });
        navigate('/visits');
      }
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
          <Badge tone={status.tone}>{status.label}</Badge>
          <span className="text-xs text-ink-3">{VISIT_PURPOSE[visit.purpose].label}</span>
          {visit.claimedAt && (
            <span className="flex items-center gap-1 text-xs text-ink-3">
              <Inbox className="size-3.5" aria-hidden />
              پذیرفته‌شده از بازدیدهای آزاد
            </span>
          )}
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

        {state === 'scheduled' && (
          <div className="mt-8 border-t border-line-soft pt-3">
            <button
              type="button"
              onClick={() => setReleaseOpen(true)}
              className="press flex min-h-11 items-center gap-2 rounded-md px-2 text-sm font-semibold text-ink-3 hover:text-ink-2"
            >
              <Undo2 className="size-4" aria-hidden />
              امکان انجام این بازدید را ندارم
            </button>
          </div>
        )}
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

      <Sheet open={releaseOpen} onClose={() => setReleaseOpen(false)} title="بازگرداندن به بازدیدهای آزاد"
        description={`${visitKindLabel(visit)} · ${faDate(visit.date, { weekday: true, year: false })} · ساعت ${faTime(visit.time)}`}
        footer={<div className="flex gap-2.5">
          <Button variant="secondary" className="flex-1" onClick={() => setReleaseOpen(false)}>انصراف</Button>
          <Button className="flex-1" loading={saving} onClick={() => { void confirmRelease(); }}>بازگرداندن به مخزن</Button>
        </div>}
      >
        <p className="pb-4 text-sm leading-7 text-ink-2">
          این بازدید با همان تاریخ و ساعت به «بازدیدهای آزاد» برمی‌گردد تا مددکار دیگری مسئولیت آن را بپذیرد.
          بازدید از برنامه شما خارج می‌شود؛ اگر این تنها بازدید خانواده نزد شما باشد، پرونده خانواده نیز از حوزه مسئولیت شما خارج خواهد شد.
        </p>
      </Sheet>
    </Screen>
  );
}

/**
 * The pool preview of a visit: everything needed to decide, nothing that
 * identifies the family. The single action is claiming — after which this
 * route renders the full visit above.
 */
function PoolVisitDetail({ visit }: { visit: Visit }) {
  const navigate = useNavigate();
  const { cases, myVisits, toast } = useApp();
  const [claimOpen, setClaimOpen] = useState(false);
  const family = cases.find((c) => c.id === visit.caseId);
  const child = family?.children.find((c) => visit.childIds?.includes(c.id)) ?? family?.children[0];
  const sameDay = visitsOnSameDay(visit, myVisits);
  const conflictIds = new Set(scheduleConflicts(visit, myVisits).map((v) => v.id));

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        onBack={() => navigate('/available')}
        title={visitKindLabel(visit)}
        subtitle={`بازدید آزاد · ${faDate(visit.date, { weekday: true, year: false })}`}
      />

      <div className="pt-4">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone="accent">بدون مددکار</Badge>
          <span className="text-xs text-ink-3">{VISIT_PURPOSE[visit.purpose].label}</span>
          {family?.priority === 'critical' && (
            <Badge tone="danger" variant="outline">{PRIORITY.critical.label}</Badge>
          )}
          <span className="ms-auto text-xs font-semibold text-brand-ink">{deadlineLabel(visit.date)}</span>
        </div>

        <Section>
          <Card className="space-y-1">
            <KeyValue label="زمان" icon={<CalendarClock className="size-3.5 text-ink-3" />}
              value={`${faDate(visit.date)} · ${faTimeRange(visit.time, visit.durationMin)}`} />
            <KeyValue label="مدت تقریبی" value={faDuration(visit.durationMin)} />
            <KeyValue label="محدوده" icon={<MapPin className="size-3.5 text-ink-3" />}
              value={family ? [family.city, family.district].filter(Boolean).join('، ') : visit.city} />
            {family && <KeyValue label="نوع پرونده" value={CASE_TYPE[family.type].label} />}
            {child && <KeyValue label="کودک مرتبط" value={`${fa(child.age)} ساله`} />}
            <KeyValue label="ارجاع" value={visit.assignedBy} />
          </Card>
        </Section>

        <Section>
          <h2 className="mb-2.5 text-sm font-bold text-ink">هدف بازدید</h2>
          <p className="text-sm leading-relaxed text-ink-2">{visit.summary}</p>
        </Section>

        <Section>
          <h2 className="mb-2.5 text-sm font-bold text-ink">برنامه شما در این روز</h2>
          {sameDay.length > 0 ? (
            <Card padded={false} className="divide-y divide-line-soft">
              {sameDay.map((mine) => (
                <div key={mine.id} className="flex items-center gap-3 px-4 py-3">
                  <span className="shrink-0 text-sm font-bold text-ink tabular-nums">{faTime(mine.time)}</span>
                  <span className="min-w-0 flex-1 truncate text-sm text-ink-2">{visitKindLabel(mine)} · {mine.city}</span>
                  {conflictIds.has(mine.id) && (
                    <span className="flex shrink-0 items-center gap-1 text-xs font-semibold text-warning">
                      <CircleAlert className="size-4" aria-hidden />
                      هم‌پوشانی
                    </span>
                  )}
                </div>
              ))}
            </Card>
          ) : (
            <p className="rounded-lg bg-sunken px-4 py-3 text-sm text-ink-2">در این روز بازدید دیگری ندارید.</p>
          )}
        </Section>

        <p className="mt-2 flex items-start gap-2 px-1 pb-2 text-xs leading-6 text-ink-3">
          <ShieldCheck className="mt-1 size-4 shrink-0" aria-hidden />
          نام، نشانی دقیق و شماره تماس خانواده پس از پذیرش بازدید نمایش داده می‌شود و سوابق کامل خانواده در دسترس قرار می‌گیرد.
        </p>
      </div>

      <ActionFooter>
        <Button block onClick={() => setClaimOpen(true)} iconStart={<Inbox className="size-4" />}>
          پذیرش این بازدید
        </Button>
      </ActionFooter>

      <ClaimVisitSheet
        visit={visit}
        open={claimOpen}
        onClose={() => setClaimOpen(false)}
        onClaimed={() => {
          setClaimOpen(false);
          toast({ title: 'بازدید به برنامه شما اضافه شد', description: 'اطلاعات کامل خانواده اکنون در دسترس است.', tone: 'success' });
        }}
      />
    </Screen>
  );
}
