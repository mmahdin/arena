import { useEffect, useMemo, useState } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  Baby,
  CalendarClock,
  CheckCircle2,
  CircleAlert,
  ClipboardList,
  FileText,
  MapPin,
  Navigation,
  Phone,
  RotateCcw,
  XCircle,
} from 'lucide-react';
import type { VisitStatus } from '@/types';
import { useApp } from '@/store/app-store';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { KeyValue } from '@/components/ui/Field';
import { TextArea } from '@/components/ui/Field';
import { ChecklistRow } from '@/components/ui/Checklist';
import { ProgressLine } from '@/components/ui/Progress';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { IconButton } from '@/components/ui/IconButton';
import { IconTile } from '@/components/ui/DomainIcon';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { EmptyState } from '@/components/ui/EmptyState';
import { VISIT_PURPOSE, VISIT_STATUS } from '@/lib/labels';
import { visitDisplayState } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { faDate, faDuration, faTimeRange, relativeTime } from '@/lib/jalali';

const OUTCOME_OPTIONS: { status: VisitStatus; icon: typeof CheckCircle2; description: string }[] = [
  { status: 'completed', icon: CheckCircle2, description: 'بازدید انجام شد و مشاهدات ثبت می‌گردد' },
  { status: 'family-absent', icon: CircleAlert, description: 'در منزل حاضر نبودند؛ هماهنگی مجدد لازم است' },
  { status: 'reschedule', icon: RotateCcw, description: 'زمان دیگری با خانواده هماهنگ می‌شود' },
  { status: 'cancelled', icon: XCircle, description: 'بازدید به دستور سرپرست لغو شد' },
];

/**
 * Visit detail — the field screen.
 *
 * Built for one-handed use while standing at a door: what/where/who at the top,
 * the checklist in the middle, and the outcome action pinned to the bottom so
 * it never requires scrolling to find. Recording an outcome closes the task,
 * writes to the case timeline and opens a report draft.
 */
export function VisitDetailScreen() {
  const { visitId } = useParams();
  const navigate = useNavigate();
  const { visits, cases, toast, toggleChecklistItem, recordVisitOutcome } = useApp();

  const visit = visits.find((v) => v.id === visitId);
  const caseFile = cases.find((c) => c.id === visit?.caseId);

  const [sheetOpen, setSheetOpen] = useState(false);
  const [selected, setSelected] = useState<VisitStatus>('completed');
  const [note, setNote] = useState('');
  const [fieldNote, setFieldNote] = useState('');
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    if (visit?.outcome) setFieldNote(visit.outcome.note);
  }, [visit?.outcome]);

  const children = useMemo(
    () => (caseFile?.children ?? []).filter((c) => visit?.childIds?.includes(c.id)),
    [caseFile, visit],
  );

  if (!visit) {
    return (
      <Screen>
        <ScreenHeader back title="بازدید" />
        <EmptyState title="بازدید یافت نشد" description="ممکن است برنامه بازدید تغییر کرده باشد." />
        <Button block onClick={() => navigate('/calendar')}>
          بازگشت به تقویم
        </Button>
      </Screen>
    );
  }

  const purpose = VISIT_PURPOSE[visit.purpose];
  const state = visitDisplayState(visit);
  const doneCount = visit.checklist.filter((c) => c.done).length;
  const closed = visit.status !== 'scheduled';

  const submitOutcome = async () => {
    setSaving(true);
    const result = await recordVisitOutcome(visit.id, selected, note.trim());
    setSaving(false);
    setSheetOpen(false);
    setNote('');
    toast({
      title: 'نتیجه بازدید ثبت شد',
      description: VISIT_STATUS[selected].label,
      tone: selected === 'completed' ? 'success' : 'warning',
      action: result.reportId
        ? { label: 'ادامه گزارش', onClick: () => navigate(`/reports/${result.reportId}`) }
        : undefined,
    });
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title="بازدید"
        subtitle={`${faDate(visit.date, { weekday: true })} · ${fa(visit.time)}`}
        actions={
          <IconButton
            label="مسیریابی"
            onClick={() => toast({ title: 'مسیریابی', description: 'اتصال به نقشه در فاز بعدی فعال می‌شود.', tone: 'info' })}
          >
            <Navigation className="size-[18px]" strokeWidth={2.1} />
          </IconButton>
        }
      />

      <div className="pt-4">
        {/* current status */}
        <Card className={cnStatus(state)}>
          <div className="flex items-start gap-3">
            <IconTile
              icon={state === 'completed' ? CheckCircle2 : state === 'needs-outcome' ? ClipboardList : CalendarClock}
              tone={state === 'completed' ? 'success' : state === 'needs-outcome' ? 'warning' : 'brand'}
            />
            <div className="min-w-0 flex-1">
              <p className="text-md font-bold text-ink">
                {state === 'completed'
                  ? 'بازدید انجام شده'
                  : state === 'needs-outcome'
                    ? 'زمان این بازدید گذشته است'
                    : state === 'not-done'
                      ? 'بازدید انجام نشد'
                      : 'بازدید برنامه‌ریزی‌شده'}
              </p>
              <p className="mt-1 text-sm leading-relaxed text-ink-2">
                {state === 'needs-outcome'
                  ? 'نتیجه بازدید را ثبت کنید تا کار مرتبط بسته شود و پیش‌نویس گزارش ساخته شود.'
                  : visit.outcome?.note || purpose.label}
              </p>
              <div className="mt-2 flex flex-wrap items-center gap-1.5">
                <Badge tone={VISIT_STATUS[visit.status].tone}>{VISIT_STATUS[visit.status].label}</Badge>
                <Badge tone={purpose.tone} variant="outline">
                  {purpose.label}
                </Badge>
                {visit.outcome && <span className="text-2xs text-ink-3">{relativeTime(visit.outcome.at)}</span>}
              </div>
            </div>
          </div>
        </Card>
      </div>

      <Section>
        <Card className="space-y-1">
          <KeyValue label="تاریخ" value={faDate(visit.date)} icon={<CalendarClock className="size-3.5 text-ink-3" />} />
          <KeyValue label="ساعت" value={faTimeRange(visit.time, visit.durationMin)} />
          <KeyValue label="مدت زمان" value={faDuration(visit.durationMin)} />
          <KeyValue label="نشانی" value={<PrivacyText value={`${visit.city} — ${visit.address}`} />} icon={<MapPin className="size-3.5 text-ink-3" />} />
          <KeyValue label="هماهنگی با" value={<PrivacyText value={visit.contactName} />} icon={<Phone className="size-3.5 text-ink-3" />} />
          <KeyValue label="ارجاع‌دهنده" value={visit.assignedBy} />

          <div className="mt-2 grid grid-cols-2 gap-2 pt-1">
            <Button
              variant="secondary"
              size="sm"
              iconStart={<Phone className="size-4" />}
              onClick={() => toast({ title: 'تماس با خانواده', description: visit.contactPhone, tone: 'info' })}
            >
              تماس
            </Button>
            <Button
              variant="secondary"
              size="sm"
              iconStart={<Navigation className="size-4" />}
              onClick={() => toast({ title: 'مسیریابی', description: 'اتصال به نقشه در فاز بعدی فعال می‌شود.', tone: 'info' })}
            >
              مسیریابی
            </Button>
          </div>
        </Card>
      </Section>

      {caseFile && (
        <Section>
          <Card>
            <div className="flex items-center justify-between gap-3">
              <div className="min-w-0">
                <p className="text-2xs font-bold text-ink-3">پرونده مرتبط</p>
                <p className="mt-0.5 truncate text-md font-bold text-ink">
                  <PrivacyText value={caseFile.familyName} interactive={false} />
                </p>
                <p className="text-xs text-ink-3 tabular-nums">{caseFile.code}</p>
              </div>
              <Link to={`/cases/${caseFile.id}`}>
                <Button variant="ghost" size="sm">
                  مشاهده پرونده
                </Button>
              </Link>
            </div>

            {children.length > 0 && (
              <div className="mt-3 flex flex-wrap gap-1.5 border-t border-line-soft pt-3">
                {children.map((c) => (
                  <Link key={c.id} to={`/children/${c.id}`}>
                    <span className="press flex items-center gap-1.5 rounded-pill bg-sunken px-3 py-1.5 text-sm font-medium text-ink-2">
                      <Baby className="size-3.5 text-brand" strokeWidth={2.4} aria-hidden />
                      {c.firstName} · {fa(c.age)} ساله
                    </span>
                  </Link>
                ))}
              </div>
            )}
          </Card>
        </Section>
      )}

      <Section>
        <Card>
          <p className="text-2xs font-bold text-ink-3">هدف و زمینه بازدید</p>
          <p className="mt-1.5 text-sm leading-relaxed text-ink-2">{visit.summary}</p>
        </Card>
      </Section>

      <Section>
        <div className="mb-2.5 flex items-end justify-between gap-3">
          <h2 className="text-md font-bold text-ink">چک‌لیست بازدید</h2>
        </div>
        <Card className="space-y-1">
          <ProgressLine
            done={doneCount}
            total={visit.checklist.length}
            unit={doneCount === visit.checklist.length ? 'همه موارد بررسی شد' : 'موارد بررسی‌شده'}
            tone={doneCount === visit.checklist.length ? 'success' : 'brand'}
          />
          <div className="-mx-1 mt-2 divide-y divide-line-soft">
            {visit.checklist.map((item) => (
              <ChecklistRow
                key={item.id}
                checked={item.done}
                required={item.required}
                label={item.label}
                onChange={() => toggleChecklistItem(visit.id, item.id)}
              />
            ))}
          </div>
        </Card>
      </Section>

      {!closed && (
        <Section>
          <Card>
            <TextArea
              label="یادداشت میدانی"
              hint="این یادداشت فقط روی دستگاه شما ذخیره می‌شود و در گزارش نهایی قابل ویرایش است."
              rows={4}
              value={fieldNote}
              onChange={(e) => setFieldNote(e.target.value)}
              placeholder="مشاهدات کلیدی را همین‌جا یادداشت کنید…"
            />
          </Card>
        </Section>
      )}

      {visit.reportId && (
        <Section>
          <Link to={`/reports/${visit.reportId}`}>
            <Card className="press flex items-center gap-3 hover:border-line-strong">
              <IconTile icon={FileText} tone="info" />
              <div className="min-w-0 flex-1">
                <p className="truncate text-base font-semibold text-ink">گزارش مرتبط با این بازدید</p>
                <p className="text-xs text-ink-3">برای تکمیل و ارسال باز کنید</p>
              </div>
            </Card>
          </Link>
        </Section>
      )}

      <ActionFooter
        hint={closed ? 'نتیجه این بازدید ثبت شده است.' : 'ثبت نتیجه، کار مرتبط را می‌بندد و پیش‌نویس گزارش می‌سازد.'}
      >
        {closed ? (
          <>
            <Button variant="secondary" className="flex-1" onClick={() => setSheetOpen(true)}>
              ویرایش نتیجه
            </Button>
            {visit.reportId && (
              <Button className="flex-1" onClick={() => navigate(`/reports/${visit.reportId}`)}>
                گزارش بازدید
              </Button>
            )}
          </>
        ) : (
          <Button block iconStart={<ClipboardList className="size-[18px]" />} onClick={() => setSheetOpen(true)}>
            ثبت نتیجه بازدید
          </Button>
        )}
      </ActionFooter>

      <Sheet
        open={sheetOpen}
        onClose={() => setSheetOpen(false)}
        title="نتیجه بازدید"
        description={`${caseFile?.familyName ?? ''} · ${faDate(visit.date, { year: false })}`}
        footer={
          <div className="flex gap-2.5">
            <Button variant="secondary" className="flex-1" onClick={() => setSheetOpen(false)}>
              انصراف
            </Button>
            <Button className="flex-1" loading={saving} onClick={submitOutcome}>
              ثبت نتیجه
            </Button>
          </div>
        }
      >
        <div className="space-y-2 pb-4">
          {OUTCOME_OPTIONS.map((o) => (
            <SheetOption
              key={o.status}
              icon={<o.icon className="size-[18px]" />}
              title={VISIT_STATUS[o.status].label}
              description={o.description}
              selected={selected === o.status}
              onClick={() => setSelected(o.status)}
            />
          ))}

          <div className="pt-2">
            <TextArea
              label="توضیح کوتاه"
              rows={3}
              value={note}
              onChange={(e) => setNote(e.target.value)}
              placeholder={selected === 'completed' ? 'خلاصه مشاهدات و وضعیت کودک…' : 'دلیل انجام نشدن بازدید…'}
              hint="این متن در پیش‌نویس گزارش و سابقه پرونده ثبت می‌شود."
            />
          </div>
        </div>
      </Sheet>
    </Screen>
  );
}

function cnStatus(state: ReturnType<typeof visitDisplayState>) {
  if (state === 'completed') return 'border-success/30 bg-success-soft/40';
  if (state === 'needs-outcome') return 'border-warning/35 bg-warning-soft/50';
  if (state === 'not-done' || state === 'cancelled') return 'border-line bg-sunken';
  return '';
}
