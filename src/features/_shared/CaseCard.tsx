import { Link } from 'react-router-dom';
import { MapPin } from 'lucide-react';
import type { CaseFile, Visit } from '@/types';
import { cn } from '@/lib/cn';
import { CASE_TYPE } from '@/lib/labels';
import { nextVisit } from '@/lib/visit-workflow';
import { visitDisplayState } from '@/lib/domain';
import { faDate } from '@/lib/jalali';
import { fa } from '@/lib/digits';
import { Avatar } from '@/components/ui/Avatar';
import { PrivacyText } from '@/components/ui/PrivacyText';

/** Family directory: who they are and the state of their visit history. */
export function CaseCard({ caseFile, visits, className }: { caseFile: CaseFile; visits: Visit[]; className?: string }) {
  const scheduled = nextVisit(visits);
  const completed = visits.filter((v) => v.status === 'completed').length;
  const withoutOutcome = visits.filter((v) => visitDisplayState(v) === 'needs-outcome').length;

  return (
    <Link
      to={`/cases/${caseFile.id}`}
      className={cn('press block rounded-lg border border-line bg-surface p-4 shadow-xs hover:border-line-strong', caseFile.stage === 'closed' && 'opacity-75', className)}
    >
      <div className="flex items-start gap-3">
        <Avatar name={caseFile.familyName} tone="neutral" className="mt-0.5" />
        <div className="min-w-0 flex-1">
          <h3 className="truncate text-base font-bold text-ink"><PrivacyText value={caseFile.familyName} interactive={false} /></h3>
          <p className="mt-1 flex items-center gap-1.5 truncate text-xs text-ink-3">
            <span>{CASE_TYPE[caseFile.type].label}</span><span>·</span><span>{caseFile.code}</span>
            <span>·</span><MapPin className="size-3.5 shrink-0" aria-hidden />
            <span className="truncate">{caseFile.district ?? caseFile.city}</span>
          </p>
          <p className="mt-2.5 text-xs leading-relaxed text-ink-2">
            {fa(completed)} بازدید انجام‌شده
            <span className="mx-1.5 text-ink-3">·</span>
            {withoutOutcome > 0
              ? `${fa(withoutOutcome)} بازدید نیازمند ثبت نتیجه`
              : scheduled ? `بازدید بعدی: ${faDate(scheduled.date, { year: false })}` : 'بازدید بعدی تعیین نشده'}
          </p>
        </div>
      </div>
    </Link>
  );
}
