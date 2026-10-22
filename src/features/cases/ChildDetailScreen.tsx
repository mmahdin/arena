import { useMemo } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { Baby, CalendarClock, HeartPulse, MapPin, School, Stethoscope } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Badge } from '@/components/ui/Badge';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { KeyValue } from '@/components/ui/Field';
import { TrendChart } from '@/components/ui/TrendChart';
import { EmptyState } from '@/components/ui/EmptyState';
import { IconTile } from '@/components/ui/DomainIcon';
import { SectionHeader } from '@/components/ui/SectionHeader';
import { ReportRow } from '@/features/_shared/ReportRow';
import { WELLBEING } from '@/lib/labels';
import { fa } from '@/lib/digits';
import { deadlineLabel, faDate } from '@/lib/jalali';

const PLACEMENT_LABEL = {
  family: 'در خانواده میزبان',
  center: 'در مرکز نگهداری',
  temporary: 'جایگذاری موقت',
} as const;

/**
 * Child under monitoring.
 *
 * The whole point of this screen is one question: is this child doing well,
 * and is that getting better or worse? So the trend leads, the facts follow,
 * and the reporting action is always within reach of the thumb.
 */
export function ChildDetailScreen() {
  const { childId } = useParams();
  const navigate = useNavigate();
  const { myCases: cases, myVisits: visits, reports, toast, createReportDraft } = useApp();

  const found = useMemo(() => {
    for (const c of cases) {
      const child = c.children.find((ch) => ch.id === childId);
      if (child) return { child, caseFile: c };
    }
    return undefined;
  }, [cases, childId]);

  const childVisits = useMemo(
    () => visits.filter((v) => v.childIds?.includes(childId ?? '')).sort((a, b) => b.date.localeCompare(a.date)),
    [visits, childId],
  );
  const childReports = useMemo(() => reports.filter((r) => r.childId === childId), [reports, childId]);

  if (!found) {
    return (
      <Screen>
        <ScreenHeader back title="کودک" />
        <EmptyState title="موردی یافت نشد" description="این رکورد ممکن است از حوزه دسترسی شما خارج شده باشد." />
      </Screen>
    );
  }

  const { child, caseFile } = found;
  const w = WELLBEING[child.wellbeing];

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title={`${child.firstName} — ${fa(child.age)} ساله`}
        subtitle={
          <Link to={`/cases/${caseFile.id}`} className="press underline-offset-4 hover:underline">
            {caseFile.familyName} · {caseFile.code}
          </Link>
        }
      />

      <div className="pt-4">
        <Card className="flex items-center justify-between gap-3">
          <div className="flex items-center gap-3">
            <IconTile icon={Baby} tone={w.tone} size="lg" />
            <div>
              <p className="text-md font-bold text-ink">{child.firstName}</p>
              <p className="text-xs text-ink-3">{PLACEMENT_LABEL[child.placement]}</p>
            </div>
          </div>
          <Badge tone={w.tone} size="md">
            {w.label}
          </Badge>
        </Card>
      </div>

      <Section>
        <TrendChart history={child.wellbeingHistory} />
      </Section>

      <Section>
        <SectionHeader title="وضعیت کودک" />
        <Card className="divide-y divide-line-soft px-4 py-1">
          <KeyValue label="سن" value={`${fa(child.age)} سال`} icon={<Baby className="size-3.5 text-ink-3" />} />
          <KeyValue label="سلامت" value={child.health ?? '—'} icon={<Stethoscope className="size-3.5 text-ink-3" />} />
          <KeyValue label="تحصیل" value={child.education ?? '—'} icon={<School className="size-3.5 text-ink-3" />} />
          <KeyValue
            label="پیگیری بعدی"
            value={`${faDate(child.nextFollowUp, { year: false })} (${deadlineLabel(child.nextFollowUp)})`}
            icon={<CalendarClock className="size-3.5 text-ink-3" />}
          />
          <KeyValue label="بازدیدهای ثبت‌شده" value={`${fa(child.followUpCount)} مورد`} icon={<MapPin className="size-3.5 text-ink-3" />} />
        </Card>
      </Section>

      {child.needs.length > 0 && (
        <Section>
          <SectionHeader title="نیازهای شناسایی‌شده" />
          <div className="flex flex-wrap gap-1.5">
            {child.needs.map((n) => (
              <span key={n} className="rounded-pill border border-line bg-surface px-3 py-1.5 text-sm text-ink-2">
                {n}
              </span>
            ))}
          </div>
        </Section>
      )}

      <Section>
        <SectionHeader title="بازدیدهای پایش" action={<span className="text-2xs text-ink-3">{fa(childVisits.length)} مورد</span>} />
        {childVisits.length === 0 ? (
          <Card className="border-dashed">
            <p className="text-sm text-ink-3">بازدیدی برای این کودک ثبت نشده است.</p>
          </Card>
        ) : (
          <Card padded={false} className="divide-y divide-line-soft">
            {childVisits.slice(0, 4).map((v) => (
              <Link key={v.id} to={`/visits/${v.id}`} className="press flex items-center gap-3 px-4 py-3 hover:bg-sunken">
                <IconTile icon={HeartPulse} tone={v.status === 'completed' ? 'success' : 'brand'} size="sm" />
                <div className="min-w-0 flex-1">
                  <p className="truncate text-base font-semibold text-ink tabular-nums">{faDate(v.date, { year: false })}</p>
                  <p className="truncate text-xs text-ink-3">{v.summary}</p>
                </div>
              </Link>
            ))}
          </Card>
        )}
      </Section>

      <Section>
        <SectionHeader title="گزارش‌ها" />
        {childReports.length === 0 ? (
          <Card className="border-dashed">
            <p className="text-sm text-ink-3">گزارشی برای این کودک ثبت نشده است.</p>
          </Card>
        ) : (
          <Card padded={false} className="divide-y divide-line-soft">
            {childReports.map((r) => (
              <ReportRow key={r.id} report={r} />
            ))}
          </Card>
        )}
      </Section>

      <ActionFooter hint="پس از ارسال، گزارش برای بازبینی سرپرست ارسال می‌شود.">
        <Button
          block
          iconStart={<HeartPulse className="size-[18px]" />}
          onClick={async () => {
            const id = await createReportDraft({
              type: 'monitoring',
              caseId: caseFile.id,
              childId: child.id,
            });
            if (id) {
              toast({ title: 'پیش‌نویس گزارش پایش ایجاد شد', tone: 'success' });
              navigate(`/reports/${id}`);
            }
          }}
        >
          ثبت گزارش پایش
        </Button>
      </ActionFooter>
    </Screen>
  );
}
