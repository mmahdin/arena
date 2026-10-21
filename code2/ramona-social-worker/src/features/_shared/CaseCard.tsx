import { Link } from 'react-router-dom';
import { ArrowLeft, Baby, MapPin } from 'lucide-react';
import type { CaseFile } from '@/types';
import { cn } from '@/lib/cn';
import { fa } from '@/lib/digits';
import { relativeTime } from '@/lib/jalali';
import { CASE_STAGE, CASE_TYPE, PRIORITY, WELLBEING } from '@/lib/labels';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';

/**
 * Case card.
 *
 * Three visual levels and no more: identity, status, next action. Everything
 * else about a family lives on the case page — a list card that tries to say
 * everything ends up saying nothing.
 */
export function CaseCard({ caseFile, className }: { caseFile: CaseFile; className?: string }) {
  const stage = CASE_STAGE[caseFile.stage];
  const type = CASE_TYPE[caseFile.type];
  const priority = PRIORITY[caseFile.priority];
  const closed = caseFile.stage === 'closed';

  return (
    <Link
      to={`/cases/${caseFile.id}`}
      className={cn(
        'press block rounded-lg border border-line bg-surface p-4 shadow-xs hover:border-line-strong',
        closed && 'opacity-75',
        className,
      )}
    >
      <div className="flex items-start gap-3">
        <Avatar name={caseFile.familyName} tone={caseFile.type === 'adoption' ? 'brand' : 'info'} className="mt-0.5" />
        <div className="min-w-0 flex-1">
          <div className="flex items-start justify-between gap-2">
            <h3 className="truncate text-md font-bold text-ink">
              <PrivacyText value={caseFile.familyName} interactive={false} />
            </h3>
            {!closed && caseFile.priority !== 'low' && <Badge tone={priority.tone}>{priority.label}</Badge>}
          </div>
          <p className="mt-1 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-sm text-ink-3">
            <span className="tabular-nums">{caseFile.code}</span>
            <span aria-hidden className="size-1 rounded-full bg-line-strong" />
            <span className="flex items-center gap-1">
              <MapPin className="size-3.5" strokeWidth={2.25} aria-hidden />
              {caseFile.district ?? caseFile.city}
            </span>
          </p>
        </div>
      </div>

      <div className="mt-3 flex flex-wrap items-center gap-1.5">
        <Badge tone={stage.tone}>{stage.label}</Badge>
        <Badge tone={type.tone} variant="outline">
          {type.label}
        </Badge>
        {caseFile.children.map((c) => (
          <span
            key={c.id}
            className="flex items-center gap-1.5 rounded-pill bg-sunken px-2 py-1 text-2xs font-medium text-ink-2"
          >
            <Baby className="size-3.5" strokeWidth={2.25} aria-hidden />
            {c.firstName} · {fa(c.age)} ساله
            {c.wellbeing !== 'stable' && (
              <span
                className={cn('size-1.5 rounded-full', WELLBEING[c.wellbeing].tone === 'warning' ? 'bg-warning' : 'bg-danger')}
                aria-label={WELLBEING[c.wellbeing].label}
              />
            )}
          </span>
        ))}
      </div>

      {caseFile.nextAction && !closed && (
        <div className="mt-3 flex items-center gap-2 rounded-md bg-sunken px-3 py-2.5">
          <ArrowLeft className="size-4 shrink-0 text-brand" strokeWidth={2.5} aria-hidden />
          <span className="min-w-0 flex-1 truncate text-sm font-medium text-ink-2">{caseFile.nextAction.label}</span>
        </div>
      )}

      <p className="mt-2.5 text-2xs text-ink-3">آخرین فعالیت {relativeTime(caseFile.lastActivityAt)}</p>
    </Link>
  );
}
