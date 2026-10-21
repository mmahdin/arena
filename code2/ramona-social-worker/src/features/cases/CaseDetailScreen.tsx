import { useMemo, useState, type ReactNode } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ChevronDown, ClipboardPlus, FileText, MapPin, Phone, Users } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { ActionFooter, Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SectionHeader, SectionAction } from '@/components/ui/SectionHeader';
import { Button } from '@/components/ui/Button';
import { Badge } from '@/components/ui/Badge';
import { Card } from '@/components/ui/Card';
import { KeyValue } from '@/components/ui/Field';
import { EmptyState } from '@/components/ui/EmptyState';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { ReportRow } from '@/features/_shared/ReportRow';
import { CASE_STAGE, CASE_TYPE, PRIORITY, VISIT_PURPOSE, VISIT_STATUS } from '@/lib/labels';
import { visitDisplayState } from '@/lib/domain';
import { fa, toLatinDigits } from '@/lib/digits';
import { faDate, relativeTime } from '@/lib/jalali';

/** A case is a concise reference for a field worker, not a workflow console. */
export function CaseDetailScreen() {
  const { caseId } = useParams();
  const navigate = useNavigate();
  const { cases, visits, reports, user, toast, createReportDraft } = useApp();
  const { fieldPrivacy } = useSettings();
  const [showAllReports, setShowAllReports] = useState(false);
  const [creating, setCreating] = useState(false);
  const caseFile = cases.find((c) => c.id === caseId);

  const caseVisits = useMemo(
    () => visits.filter((v) => v.caseId === caseId).sort((a, b) => (b.date + b.time).localeCompare(a.date + a.time)),
    [visits, caseId],
  );
  const caseReports = useMemo(
    () => reports.filter((r) => r.caseId === caseId && r.author === user?.name).sort((a, b) => {
      const priority = (status: string) => status === 'revision' ? 0 : status === 'draft' ? 1 : 2;
      return priority(a.status) - priority(b.status) || b.updatedAt.localeCompare(a.updatedAt);
    }),
    [reports, caseId, user?.name],
  );

  if (!caseFile) {
    return (
      <Screen>
        <ScreenHeader back title="پرونده" />
        <EmptyState title="پرونده یافت نشد" description="ممکن است دسترسی شما به این پرونده محدود شده باشد." />
        <Button block onClick={() => navigate('/cases')}>بازگشت به پرونده‌ها</Button>
      </Screen>
    );
  }

  const pendingVisit = caseVisits.find((v) => visitDisplayState(v) === 'needs-outcome');
  const stage = CASE_STAGE[caseFile.stage];
  const closed = caseFile.stage === 'closed';
  const reportsToShow = showAllReports ? caseReports : caseReports.slice(0, 3);

  const openNewReport = async () => {
    if (creating || closed) return;
    setCreating(true);
    try {
      const id = await createReportDraft({
        type: caseFile.stage === 'initial-assessment' ? 'home-assessment' : caseFile.stage === 'post-placement' ? 'monitoring' : 'visit',
        caseId: caseFile.id,
        childId: caseFile.children[0]?.id,
      });
      if (id) {
        toast({ title: 'پیش‌نویس گزارش ایجاد شد', tone: 'success' });
        navigate(`/reports/${id}`);
      }
    } finally {
      setCreating(false);
    }
  };

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title={<PrivacyText value={caseFile.familyName} interactive={false} />}
        subtitle={`${caseFile.code} · ${CASE_TYPE[caseFile.type].label}`}
      />

      <div className="pt-4">
        <div className="flex flex-wrap items-center gap-2">
          <Badge tone={stage.tone}>{stage.label}</Badge>
          {!closed && caseFile.priority === 'critical' && <Badge tone={PRIORITY[caseFile.priority].tone}>اولویت فوری</Badge>}
        </div>

        <Section>
          <SectionHeader title="اطلاعات ضروری" />
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
          </Card>
          {caseFile.phone && !fieldPrivacy && (
            <a
              href={`tel:${toLatinDigits(caseFile.phone).replace(/[^0-9+]/g, '')}`}
              className="press mt-2 inline-flex min-h-10 items-center gap-2 px-1 text-sm font-semibold text-brand-ink"
            >
              <Phone className="size-4" aria-hidden /> تماس با خانواده
            </a>
          )}
        </Section>

        {caseFile.notes && (
          <p className="mt-4 rounded-md bg-sunken px-4 py-3 text-sm leading-relaxed text-ink-2">
            {caseFile.notes}
          </p>
        )}

        <Section>
          <SectionHeader
            title="گزارش‌های من"
            action={!closed ? <SectionAction onClick={() => { void openNewReport(); }}>گزارش جدید</SectionAction> : undefined}
          />
          {caseReports.length === 0 ? (
            <Card className="border-dashed">
              <p className="text-sm text-ink-3">هنوز گزارشی برای این پرونده ثبت نکرده‌اید.</p>
            </Card>
          ) : (
            <Card padded={false}>
              {reportsToShow.map((report, i) => (
                <ReportRow key={report.id} report={report} showCase={false} divider={i < reportsToShow.length - 1} />
              ))}
              {caseReports.length > 3 && (
                <button
                  type="button"
                  onClick={() => setShowAllReports((v) => !v)}
                  className="press w-full border-t border-line-soft px-4 py-3 text-center text-sm font-semibold text-brand-ink"
                  aria-expanded={showAllReports}
                >
                  {showAllReports ? 'نمایش کمتر' : `نمایش ${fa(caseReports.length - 3)} گزارش دیگر`}
                </button>
              )}
            </Card>
          )}
        </Section>

        <Disclosure title="اطلاعات تکمیلی" count={caseFile.members.length + caseFile.children.length}>
          {caseFile.housing && (
            <div className="mb-4 space-y-2">
              <h3 className="text-sm font-bold text-ink">شرایط سکونت</h3>
              <p className="text-sm leading-relaxed text-ink-2">
                {caseFile.housing.tenure} · {fa(caseFile.housing.rooms)} اتاق · {caseFile.housing.condition}
              </p>
            </div>
          )}
          {caseFile.income && (
            <div className="mb-4 space-y-1">
              <h3 className="text-sm font-bold text-ink">وضعیت اقتصادی ثبت‌شده</h3>
              <p className="text-sm leading-relaxed text-ink-2">{caseFile.income}</p>
            </div>
          )}
          {caseFile.members.length > 0 && (
            <div className="mb-4">
              <h3 className="mb-2 flex items-center gap-2 text-sm font-bold text-ink"><Users className="size-4" aria-hidden /> اعضای خانوار</h3>
              <ul className="divide-y divide-line-soft">
                {caseFile.members.map((m) => (
                  <li key={m.id} className="flex flex-wrap justify-between gap-2 py-2 text-sm">
                    <PrivacyText value={m.name} interactive={false} />
                    <span className="text-ink-3">{m.relation} · {fa(m.age)} سال</span>
                  </li>
                ))}
              </ul>
            </div>
          )}
          {caseFile.children.length > 0 && (
            <div className="mb-2">
              <h3 className="mb-2 text-sm font-bold text-ink">کودکان مرتبط</h3>
              {caseFile.children.map((ch) => (
                <Link key={ch.id} to={`/children/${ch.id}`} className="press flex min-h-10 items-center justify-between border-t border-line-soft py-2 text-sm text-brand-ink">
                  <span><PrivacyText value={ch.firstName} interactive={false} /> · {fa(ch.age)} سال</span>
                  <span>مشاهده</span>
                </Link>
              ))}
            </div>
          )}
          {caseFile.documents.length > 0 && (
            <div className="mt-4">
              <h3 className="mb-2 text-sm font-bold text-ink">مستندات ثبت‌شده</h3>
              <ul className="divide-y divide-line-soft">
                {caseFile.documents.map((d) => (
                  <li key={d.id} className="flex items-center justify-between gap-2 py-2 text-sm">
                    <span className="min-w-0 flex-1 text-ink-2">{d.title}</span>
                    <span className="shrink-0 text-xs text-ink-3">{relativeTime(d.updatedAt)}</span>
                  </li>
                ))}
              </ul>
              <p className="mt-2 text-xs text-ink-3">دریافت فایل‌های پیوست در نسخه نمایشی فعال نیست.</p>
            </div>
          )}
        </Disclosure>

        {caseVisits.length > 0 && (
          <Disclosure title="سوابق بازدید" count={caseVisits.length}>
            <div className="divide-y divide-line-soft">
              {caseVisits.map((visit) => (
                <Link key={visit.id} to={`/visits/${visit.id}`} className="press flex min-h-14 items-center justify-between gap-3 py-3">
                  <span className="min-w-0 flex-1">
                    <span className="block text-sm font-semibold text-ink">{VISIT_PURPOSE[visit.purpose].label}</span>
                    <span className="mt-1 block text-xs text-ink-3">{faDate(visit.date, { year: false })} · {fa(visit.time)}</span>
                  </span>
                  <span className="text-xs text-ink-2">{VISIT_STATUS[visit.status].label}</span>
                </Link>
              ))}
            </div>
          </Disclosure>
        )}
      </div>

      {!closed && (
        <ActionFooter>
          {pendingVisit ? (
            <Button block iconStart={<ClipboardPlus className="size-[18px]" />} onClick={() => navigate(`/visits/${pendingVisit.id}`)}>
              ثبت نتیجه بازدید
            </Button>
          ) : (
            <Button block loading={creating} iconStart={<FileText className="size-[18px]" />} onClick={() => { void openNewReport(); }}>
              ثبت گزارش جدید
            </Button>
          )}
        </ActionFooter>
      )}
    </Screen>
  );
}

function Disclosure({ title, count, children }: { title: string; count?: number; children: ReactNode }) {
  return (
    <details className="group mt-4 overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
      <summary className="press flex min-h-12 cursor-pointer list-none items-center justify-between gap-3 px-4 py-3 text-sm font-bold text-ink [&::-webkit-details-marker]:hidden">
        <span>{title}{typeof count === 'number' && <span className="ms-2 text-xs font-medium text-ink-3">{fa(count)}</span>}</span>
        <ChevronDown className="size-4 text-ink-3 transition-transform group-open:rotate-180" aria-hidden />
      </summary>
      <div className="border-t border-line-soft px-4 pb-4 pt-3">{children}</div>
    </details>
  );
}
