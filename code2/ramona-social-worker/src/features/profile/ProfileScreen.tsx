import { useNavigate } from 'react-router-dom';
import { BadgeCheck, Building2, Mail, MapPin, Phone, Settings, ShieldCheck, UserRound } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Avatar } from '@/components/ui/Avatar';
import { Button } from '@/components/ui/Button';
import { Card } from '@/components/ui/Card';
import { KeyValue } from '@/components/ui/Field';
import { SectionHeader } from '@/components/ui/SectionHeader';
import { fa } from '@/lib/digits';

/**
 * Profile.
 *
 * A social worker's professional identity matters in this product: reports and
 * assessments are signed with it, and access to confidential records is
 * derived from it. So the page shows credentials and permissions plainly —
 * including what the user is *not* allowed to do.
 */
export function ProfileScreen() {
  const { user, cases, visits, reports } = useApp();
  const navigate = useNavigate();

  if (!user) return <Screen />;

  const activeCases = cases.filter((c) => c.stage !== 'closed').length;
  const monthVisits = visits.filter((v) => v.status === 'completed').length;
  const approved = reports.filter((r) => r.status === 'approved').length;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader title="پروفایل حرفه‌ای" back />

      <div className="pt-4">
        <Card className="flex items-center gap-4">
          <Avatar name={user.name} size="lg" tone="brand" />
          <div className="min-w-0">
            <h2 className="truncate text-lg font-extrabold text-ink">{user.name}</h2>
            <p className="text-sm font-medium text-brand-ink">{user.role}</p>
            <p className="mt-0.5 truncate text-xs text-ink-3">{user.unit}</p>
          </div>
        </Card>

        <div className="mt-3 grid grid-cols-3 overflow-hidden rounded-lg border border-line bg-surface shadow-xs">
          <Stat value={activeCases} label="پرونده فعال" />
          <Stat value={monthVisits} label="بازدید ثبت‌شده" divider />
          <Stat value={approved} label="گزارش تأییدشده" divider />
        </div>
      </div>

      <Section>
        <SectionHeader title="اطلاعات حرفه‌ای" />
        <Card className="divide-y divide-line-soft px-4 py-1">
          <KeyValue label="سازمان" value={user.organization} icon={<Building2 className="size-3.5 text-ink-3" />} />
          <KeyValue label="کد پرسنلی" value={<span className="tabular-nums">{user.employeeCode}</span>} icon={<UserRound className="size-3.5 text-ink-3" />} />
          <KeyValue label="شماره پروانه" value={user.licenseNumber} icon={<BadgeCheck className="size-3.5 text-ink-3" />} />
          <KeyValue label="سرپرست واحد" value={user.supervisor} />
          <KeyValue label="تلفن" value={<span className="tabular-nums">{user.phone}</span>} icon={<Phone className="size-3.5 text-ink-3" />} />
          <KeyValue label="ایمیل سازمانی" value={user.email} icon={<Mail className="size-3.5 text-ink-3" />} />
        </Card>
      </Section>

      <Section>
        <SectionHeader title="حوزه فعالیت" hint="محدوده جغرافیایی پرونده‌های محوله" />
        <div className="flex flex-wrap gap-1.5">
          {user.coverageAreas.map((a) => (
            <span key={a} className="flex items-center gap-1.5 rounded-pill border border-line bg-surface px-3 py-1.5 text-sm text-ink-2">
              <MapPin className="size-3.5 text-brand" strokeWidth={2.25} aria-hidden />
              {a}
            </span>
          ))}
        </div>
      </Section>

      <Section>
        <SectionHeader title="دسترسی‌ها" />
        <Card>
          <ul className="space-y-2.5">
            {user.permissions.map((p) => (
              <li key={p} className="flex items-start gap-2.5">
                <BadgeCheck className="mt-0.5 size-4 shrink-0 text-success" strokeWidth={2.25} aria-hidden />
                <span className="text-sm leading-relaxed text-ink-2">{p}</span>
              </li>
            ))}
          </ul>
          <p className="mt-4 flex items-start gap-2 border-t border-line-soft pt-3 text-2xs leading-relaxed text-ink-3">
            <ShieldCheck className="mt-0.5 size-4 shrink-0" strokeWidth={2.25} aria-hidden />
            دسترسی شما محدود به پرونده‌هایی است که به شما محول شده‌اند. تصمیم‌گیری درباره صلاحیت،
            جایگذاری و امور حقوقی در صلاحیت مراجع ذی‌صلاح است.
          </p>
        </Card>
      </Section>

      <Section className="pb-8">
        <Button block variant="secondary" iconStart={<Settings className="size-[18px]" />} onClick={() => navigate('/settings')}>
          تنظیمات برنامه
        </Button>
      </Section>
    </Screen>
  );
}

function Stat({ value, label, divider }: { value: number; label: string; divider?: boolean }) {
  return (
    <div className={`px-2 py-3 text-center ${divider ? 'border-s border-line-soft' : ''}`}>
      <p className="text-xl font-extrabold text-ink tabular-nums">{fa(value)}</p>
      <p className="mt-0.5 text-2xs font-medium text-ink-3">{label}</p>
    </div>
  );
}
