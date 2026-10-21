import { Link } from 'react-router-dom';
import { MapPin } from 'lucide-react';
import type { CaseFile } from '@/types';
import { cn } from '@/lib/cn';
import { CASE_STAGE, CASE_TYPE, PRIORITY } from '@/lib/labels';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';

/** Cases are reference material, not another task dashboard. */
export function CaseCard({ caseFile, className }: { caseFile: CaseFile; className?: string }) {
  const closed = caseFile.stage === 'closed';
  const important = caseFile.priority === 'critical' || caseFile.priority === 'high';

  return (
    <Link
      to={`/cases/${caseFile.id}`}
      className={cn('press block rounded-lg border border-line bg-surface p-4 shadow-xs hover:border-line-strong', closed && 'opacity-75', className)}
    >
      <div className="flex items-start gap-3">
        <Avatar name={caseFile.familyName} tone="neutral" className="mt-0.5" />
        <div className="min-w-0 flex-1">
          <div className="flex flex-wrap items-start justify-between gap-2">
            <h3 className="min-w-0 flex-1 text-base font-bold leading-relaxed text-ink">
              <PrivacyText value={caseFile.familyName} interactive={false} />
            </h3>
            {!closed && important && <Badge tone={PRIORITY[caseFile.priority].tone}>{PRIORITY[caseFile.priority].label}</Badge>}
          </div>
          <p className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ink-3">
            <span className="tabular-nums">{caseFile.code}</span>
            <span aria-hidden>·</span>
            <span className="inline-flex items-center gap-1"><MapPin className="size-3.5" aria-hidden />{caseFile.district ?? caseFile.city}</span>
          </p>
          <p className="mt-2 text-sm text-ink-2">
            {CASE_TYPE[caseFile.type].label} <span className="mx-1 text-ink-3">·</span> {CASE_STAGE[caseFile.stage].label}
          </p>
        </div>
      </div>
    </Link>
  );
}
