import { useMemo, useState, type ReactNode } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import {
  BadgeCheck,
  Building2,
  ClipboardPlus,
  FileText,
  Flag,
  MapPin,
  MoreVertical,
  Paperclip,
  PenLine,
  Phone,
  Users,
} from 'lucide-react';
import type { CaseEvent, CaseFile, Tone, VisitPurpose } from '@/types';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Card } from '@/components/ui/Card';
import { KeyValue } from '@/components/ui/Field';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { Stepper } from '@/components/ui/Stepper';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile } from '@/components/ui/DomainIcon';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { Avatar } from '@/components/ui/Avatar';
import { ChildFollowUpRow } from '@/features/_shared/ChildRow';
import { ReportRow } from '@/features/_shared/ReportRow';
import { CASE_STAGE, CASE_TYPE, PRIORITY, VISIT_PURPOSE } from '@/lib/labels';
import { fa } from '@/lib/digits';
import { dayKey, faDate, relativeTime } from '@/lib/jalali';
import { cn } from '@/lib/cn';
import { TONE } from '@/components/ui/tone';

type Tab = 'summary' | 'reports' | 'activity';

/**
 * Case file.
 *
 * A case is what everything else hangs off: visits, reports and monitoring.
 * The page therefore opens with "where is this case and what happens next",
 * then offers depth through three tabs rather than one long wall of data.
 */
export function CaseDetailScreen() {
  const { caseId } = useParams();
  const navigate = useNavigate();
  const { cases, visits, reports, toast, createReportDraft } = useApp();
  const { fieldPrivacy } = useSettings();
  const [tab, setTab] = useState<Tab>('summary');
  const [menuOpen, setMenuOpen] = useState(false);

  const caseFile = cases.find((c) => c.id === caseId);

  const caseVisits = useMemo(
    () =>
      visits
        .filter((v) => v.caseId === caseId)
        .sort((a, b) => (b.date + b.time).localeCompare(a.date + a.time)),
    [visits, caseId],
  );
  const caseReports = useMemo(() => reports.filter((r) => r.caseId === caseId), [reports, caseId]);

  if (!caseFile) {
    return (
      <Screen>
        <ScreenHeader back title="پرونده" />
        <EmptyState title="پرونده یافت نشد" description="ممکن است دسترسی شما به این پرونده محدود شده باشد." />
        <Button block onClick={() => navigate('/cases')}>
          بازگشت به پرونده‌ها
        </Button>
      </Screen>
    );
  }

  const todayKey = dayKey(new Date());
  const upcoming = caseVisits.find((v) => v.status === 'scheduled' && v.date >= todayKey);
  const stage = CASE_STAGE[caseFile.stage];

  const openNewReport = async () => {
    setMenuOpen(false);
    const id = await createReportDraft({
      type:
        caseFile.stage === 'initial-assessment'
          ? 'home-assessment'
          : caseFile.stage === 'post-placement'
            ? 'monitoring'
            : 'visit',
      caseId: caseFile.id,
      childId: caseFile.children[0]?.id,
    });
    if (id) {
      toast({ title: 'پیش‌نویس گزارش ایجاد شد', tone: 'success' });
      navigate(`/reports/${id}`);
    }
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title={<PrivacyText value={caseFile.familyName} interactive={false} />}
        subtitle={`${caseFile.code} · ${stage.label}`}
        actions={
          <>
            {caseFile.phone && (
              <IconButton
                label="تماس با خانواده"
                onClick={() => toast({ title: 'در حال برقراری تماس', description: caseFile.phone, tone: 'info' })}
              >
                <Phone className="size-[18px]" strokeWidth={2.1} />
              </IconButton>
            )}
            <IconButton label="گزینه‌های پرونده" onClick={() => setMenuOpen(true)}>
              <MoreVertical className="size-[18px]" strokeWidth={2.1} />
            </IconButton>
          </>
        }
      />

      <div className="pt-4">
        <Card className="space-y-4">
          <div className="flex flex-wrap items-center gap-1.5">
            <Badge tone={stage.tone}>{stage.label}</Badge>
            <Badge tone={CASE_TYPE[caseFile.type].tone} variant="outline">
              {CASE_TYPE[caseFile.type].label}
            </Badge>
            {caseFile.stage !== 'closed' && (
              <Badge tone={PRIORITY[caseFile.priority].tone} variant="outline">
                اولویت {PRIORITY[caseFile.priority].label}
              </Badge>
            )}
          </div>

          <Stepper stage={caseFile.stage} />

          {caseFile.nextAction && caseFile.stage !== 'closed' && (
            <div className="rounded-md bg-sunken p-3.5">
              <p className="text-2xs font-bold text-ink-3">اقدام بعدی</p>
              <p className="mt-1 text-sm font-semibold leading-relaxed text-ink">{caseFile.nextAction.label}</p>
              {upcoming && (
                <Button
                  size="sm"
                  variant="ghost"
                  className="mt-1.5 -ms-2"
                  iconStart={<MapPin className="size-4" />}
                  onClick={() => navigate(`/visits/${upcoming.id}`)}
                >
                  جزئیات بازدید
                </Button>
              )}
            </div>
          )}

          {caseFile.notes && (
            <p className="border-s-2 border-line-strong ps-3 text-sm leading-relaxed text-ink-2">{caseFile.notes}</p>
          )}
        </Card>

        <SegmentedControl<Tab>
          className="mt-4"
          value={tab}
          onChange={setTab}
          ariaLabel="بخش‌های پرونده"
          items={[
            { value: 'summary', label: 'خلاصه' },
            { value: 'reports', label: 'گزارش‌ها', count: caseReports.length },
            { value: 'activity', label: 'فعالیت‌ها', count: caseFile.timeline.length },
          ]}
        />
      </div>

      {tab === 'summary' && (
        <>
          <Section>
            <SectionTitle icon={Users} title="اطلاعات خانوار" />
            <Card className="divide-y divide-line-soft px-4 py-1">
              <KeyValue label="سرپرست خانوار" value={<PrivacyText value={caseFile.headOfHousehold} />} />
              <KeyValue
                label="شماره تماس"
                icon={<Phone className="size-3.5 text-ink-3" />}
                value={caseFile.phone ? <PrivacyText value={caseFile.phone} /> : '—'}
              />
              <KeyValue
                label="نشانی"
                icon={<MapPin className="size-3.5 text-ink-3" />}
                value={<PrivacyText value={`${caseFile.city}، ${caseFile.address}`} />}
              />
              {caseFile.housing && (
                <KeyValue
                  label="وضعیت سکونت"
                  icon={<Building2 className="size-3.5 text-ink-3" />}
                  value={`${caseFile.housing.tenure} · ${fa(caseFile.housing.rooms)} اتاق`}
                />
              )}
              {caseFile.housing && <KeyValue label="شرایط منزل" value={caseFile.housing.condition} />}
              {caseFile.income && <KeyValue label="وضعیت درآمد" value={caseFile.income} />}
              <KeyValue label="گشایش پرونده" value={faDate(caseFile.openedAt)} />
            </Card>
          </Section>

          {caseFile.members.length > 0 && (
            <Section>
              <SectionTitle icon={Users} title="اعضای خانوار" />
              <Card padded={false} className="divide-y divide-line-soft">
                {caseFile.members.map((m) => (
                  <div key={m.id} className="flex items-center gap-3 px-4 py-3">
                    <Avatar name={m.name} size="sm" tone="neutral" />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-base font-semibold text-ink">
                        <PrivacyText value={m.name} interactive={false} />
                      </p>
                      <p className="text-xs text-ink-3">
                        {m.relation} · {fa(m.age)} ساله{m.occupation ? ` · ${m.occupation}` : ''}
                      </p>
                    </div>
                  </div>
                ))}
              </Card>
            </Section>
          )}

          <Section>
            <SectionTitle icon={Users} title="کودکان پرونده" />
            {caseFile.children.length === 0 ? (
              <Card className="border-dashed">
                <p className="text-sm leading-relaxed text-ink-3">
                  این پرونده در مرحله ارزیابی متقاضیان است و هنوز کودکی به آن تخصیص نیافته است.
                </p>
              </Card>
            ) : (
              <Card padded={false} className="divide-y divide-line-soft">
                {caseFile.children.map((ch) => (
                  <ChildFollowUpRow
                    key={ch.id}
                    child={{ ...ch, caseCode: caseFile.code, familyName: caseFile.familyName, caseId: caseFile.id }}
                  />
                ))}
              </Card>
            )}
          </Section>

          <Section>
            <SectionTitle
              icon={MapPin}
              title="بازدیدها"
              action={caseVisits.length > 4 ? <span className="text-2xs text-ink-3">{fa(caseVisits.length)} مورد</span> : undefined}
            />
            {caseVisits.length === 0 ? (
              <Card className="border-dashed">
                <p className="text-sm text-ink-3">بازدیدی برای این پرونده ثبت نشده است.</p>
              </Card>
            ) : (
              <Card padded={false} className="divide-y divide-line-soft">
                {caseVisits.slice(0, 4).map((v) => (
                  <Link key={v.id} to={`/visits/${v.id}`} className="press flex items-center gap-3 px-4 py-3 hover:bg-sunken">
                    <IconTile
                      icon={v.status === 'completed' ? BadgeCheck : MapPin}
                      tone={v.status === 'completed' ? 'success' : v.status === 'scheduled' ? 'brand' : 'warning'}
                      size="sm"
                    />
                    <div className="min-w-0 flex-1">
                      <p className="truncate text-base font-semibold text-ink tabular-nums">
                        {faDate(v.date, { year: false })} — {fa(v.time)}
                      </p>
                      <p className="truncate text-xs text-ink-3">{VISIT_PURPOSE[v.purpose as VisitPurpose].label}</p>
                    </div>
                    <Badge tone={v.status === 'completed' ? 'success' : v.status === 'scheduled' ? 'info' : 'warning'}>
                      {v.status === 'completed' ? 'انجام شد' : v.status === 'scheduled' ? 'زمان‌بندی‌شده' : 'ثبت‌نشده'}
                    </Badge>
                  </Link>
                ))}
              </Card>
            )}
          </Section>

          {caseFile.documents.length > 0 && (
            <Section>
              <SectionTitle icon={Paperclip} title="مستندات" />
              <Card padded={false} className="divide-y divide-line-soft">
                {caseFile.documents.map((d) => (
                  <button
                    key={d.id}
                    type="button"
                    onClick={() => toast({ title: 'نمایش مستند', description: d.title, tone: 'info' })}
                    className="press flex w-full items-center gap-3 px-4 py-3 text-start hover:bg-sunken"
                  >
                    <IconTile icon={d.kind === 'image' ? Paperclip : FileText} tone="neutral" size="sm" />
                    <span className="min-w-0 flex-1">
                      <span className="block truncate text-base font-medium text-ink">{d.title}</span>
                      <span className="block text-xs text-ink-3">{relativeTime(d.updatedAt)}</span>
                    </span>
                  </button>
                ))}
              </Card>
            </Section>
          )}
        </>
      )}

      {tab === 'reports' && (
        <Section>
          {caseReports.length === 0 ? (
            <EmptyState
              icon={<FileText className="size-6" />}
              title="گزارشی ثبت نشده"
              description="پس از نخستین بازدید، پیش‌نویس گزارش به‌صورت خودکار ایجاد می‌شود."
            />
          ) : (
            <Card padded={false} className="divide-y divide-line-soft">
              {caseReports.map((r) => (
                <ReportRow key={r.id} report={r} />
              ))}
            </Card>
          )}
        </Section>
      )}

      {tab === 'activity' && (
        <Section>
          <Timeline events={caseFile.timeline} />
        </Section>
      )}

      <ActionFooter hint={fieldPrivacy ? 'حالت محرمانگی فعال است.' : undefined}>
        <Button block iconStart={<ClipboardPlus className="size-[18px]" />} onClick={() => setMenuOpen(true)}>
          ثبت فعالیت
        </Button>
      </ActionFooter>

      <Sheet open={menuOpen} onClose={() => setMenuOpen(false)} title="ثبت فعالیت" description={caseFile.familyName}>
        <div className="space-y-2 pb-4">
          {upcoming && (
            <SheetOption
              icon={<MapPin className="size-[18px]" />}
              title="ثبت نتیجه بازدید"
              description={`${faDate(upcoming.date, { year: false })} — ${fa(upcoming.time)}`}
              onClick={() => {
                setMenuOpen(false);
                navigate(`/visits/${upcoming.id}`);
              }}
            />
          )}
          <SheetOption
            icon={<FileText className="size-[18px]" />}
            title="گزارش جدید برای این پرونده"
            description="پیش‌نویس بر اساس قالب ارزیابی ساخته می‌شود"
            onClick={openNewReport}
          />
          <SheetOption
            icon={<PenLine className="size-[18px]" />}
            title="یادداشت داخلی"
            description="در فازهای بعدی فعال می‌شود"
            onClick={() => {
              setMenuOpen(false);
              toast({ title: 'یادداشت داخلی', description: 'این قابلیت در فاز بعدی پیاده‌سازی می‌شود.', tone: 'info' });
            }}
          />
        </div>
      </Sheet>
    </Screen>
  );
}

function SectionTitle({ icon: Icon, title, action }: { icon: typeof Users; title: string; action?: ReactNode }) {
  return (
    <div className="mb-2.5 flex items-center justify-between gap-3">
      <h2 className="flex items-center gap-2 text-md font-bold text-ink">
        <Icon className="size-4 text-ink-3" strokeWidth={2.25} aria-hidden />
        {title}
      </h2>
      {action}
    </div>
  );
}

function Timeline({ events }: { events: CaseEvent[] }) {
  if (events.length === 0) {
    return (
      <Card className="border-dashed">
        <p className="text-sm text-ink-3">فعالیتی ثبت نشده است.</p>
      </Card>
    );
  }

  const meta: Record<CaseEvent['kind'], { icon: typeof Flag; tone: Tone }> = {
    visit: { icon: MapPin, tone: 'brand' },
    report: { icon: FileText, tone: 'info' },
    stage: { icon: Flag, tone: 'accent' },
    note: { icon: PenLine, tone: 'neutral' },
    decision: { icon: BadgeCheck, tone: 'success' },
  };

  return (
    <ol className="relative space-y-4">
      <span className="absolute bottom-3 start-[15px] top-3 w-px bg-line" aria-hidden />
      {events.map((e) => {
        const m = meta[e.kind];
        return (
          <li key={e.id} className="relative flex gap-3.5">
            <span
              className={cn(
                'relative z-10 grid size-8 shrink-0 place-items-center rounded-full ring-4 ring-bg',
                TONE[m.tone].soft,
                TONE[m.tone].text,
              )}
            >
              <m.icon className="size-4" strokeWidth={2.25} aria-hidden />
            </span>
            <div className="min-w-0 flex-1 pb-1">
              <p className="text-base font-semibold leading-snug text-ink">{e.title}</p>
              {e.detail && <p className="mt-1 text-sm leading-relaxed text-ink-2">{e.detail}</p>}
              <p className="mt-1 text-2xs text-ink-3">
                {relativeTime(e.at)}
                {e.actor ? ` · ${e.actor}` : ''}
              </p>
            </div>
          </li>
        );
      })}
    </ol>
  );
}
