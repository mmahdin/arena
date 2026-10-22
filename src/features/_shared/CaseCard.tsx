import { Link } from 'react-router-dom';
import { AlertCircle, CalendarDays, CheckCircle2, ChevronLeft, MapPin } from 'lucide-react';
import type { CaseFile, Visit } from '@/types';
import { cn } from '@/lib/cn';
import { useSettings } from '@/store/settings';
import { CASE_TYPE } from '@/lib/labels';
import { nextVisit } from '@/lib/visit-workflow';
import { visitDisplayState } from '@/lib/domain';
import { faDate } from '@/lib/jalali';
import { fa } from '@/lib/digits';
import { Avatar } from '@/components/ui/Avatar';
import { PrivacyText } from '@/components/ui/PrivacyText';

/** A scannable family summary: identity, location, then the relevant visit state. */
export function CaseCard({ caseFile, visits, className }: { caseFile: CaseFile; visits: Visit[]; className?: string }) {
  const { fieldPrivacy } = useSettings();
  const scheduled = nextVisit(visits);
  const completed = visits.filter((v) => v.status === 'completed').length;
  const withoutOutcome = visits.filter((v) => visitDisplayState(v) === 'needs-outcome').length;
  const location = [caseFile.district, caseFile.city].filter(Boolean).join('، ');

  return (
    <Link
      to={`/cases/${caseFile.id}`}
      className={cn(
        'press block rounded-lg border border-line bg-surface p-4 shadow-xs hover:border-brand/35 hover:shadow-sm',
        className,
      )}
    >
      <div className="flex items-start gap-3">
        <Avatar name={caseFile.familyName} tone={CASE_TYPE[caseFile.type].tone} masked={fieldPrivacy} className="mt-0.5" />
        <div className="min-w-0 flex-1">
          <h3 className="text-base font-bold leading-6 text-ink [overflow-wrap:anywhere]">
            <PrivacyText value={caseFile.familyName} interactive={false} />
          </h3>
          <p className="mt-0.5 text-xs font-medium text-ink-2">{CASE_TYPE[caseFile.type].label}</p>
        </div>
        <ChevronLeft className="mt-2 size-[18px] text-ink-3" aria-hidden />
      </div>

      {location && (
        <p className="mt-3 flex items-start gap-2 text-sm leading-6 text-ink-2">
          <MapPin className="mt-1 size-4 shrink-0 text-ink-3" aria-hidden />
          <span className="min-w-0 [overflow-wrap:anywhere]">
            <PrivacyText value={location} interactive={false} />
          </span>
        </p>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-x-4 gap-y-1.5 border-t border-line-soft pt-3 text-xs">
        {withoutOutcome > 0 ? (
          <span className="inline-flex items-center gap-1.5 font-semibold text-warning">
            <AlertCircle className="size-4" aria-hidden />
            {fa(withoutOutcome)} بازدید نیازمند ثبت نتیجه
          </span>
        ) : scheduled ? (
          <span className="inline-flex items-center gap-1.5 font-semibold text-brand-ink">
            <CalendarDays className="size-4" aria-hidden />
            بازدید بعدی: {faDate(scheduled.date, { year: false })}
          </span>
        ) : null}
        {completed > 0 ? (
          <span className="inline-flex items-center gap-1.5 text-ink-3">
            <CheckCircle2 className="size-4" aria-hidden />
            {fa(completed)} بازدید انجام‌شده
          </span>
        ) : !scheduled && withoutOutcome === 0 ? (
          <span className="text-ink-3">هنوز بازدیدی ثبت نشده</span>
        ) : null}
      </div>
    </Link>
  );
}
