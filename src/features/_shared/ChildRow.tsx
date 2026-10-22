import { ListRow } from '@/components/ui/ListRow';
import { Avatar } from '@/components/ui/Avatar';
import { Badge } from '@/components/ui/Badge';
import { ScaleDots } from '@/components/ui/ScaleInput';
import { WELLBEING } from '@/lib/labels';
import { deadlineLabel } from '@/lib/jalali';
import { fa } from '@/lib/digits';
import type { ChildSummary } from '@/services/cases';
import { Meta, MetaTone } from './Meta';

/**
 * Child under monitoring.
 *
 * The wellbeing signal is the reason this list exists, so it appears twice —
 * as a colour-coded badge and as a 5-step scale — because a single abstract
 * colour is not enough when a child's situation is being judged.
 */
export function ChildRow({ child, divider }: { child: ChildSummary; divider?: boolean }) {
  const w = WELLBEING[child.wellbeing];
  const latest = child.wellbeingHistory[child.wellbeingHistory.length - 1]?.score ?? 0;

  return (
    <ListRow
      to={`/children/${child.id}`}
      divider={divider}
      chevron
      leading={<Avatar name={child.firstName} tone={w.tone} />}
      title={`${child.firstName} — ${fa(child.age)} ساله`}
      meta={
        <Meta>
          <span>{child.familyName}</span>
          <span className="tabular-nums">{child.caseCode}</span>
        </Meta>
      }
      trailing={
        <div className="flex flex-col items-end gap-1.5">
          <Badge tone={w.tone}>{w.label}</Badge>
          <ScaleDots
            value={latest}
            tone={child.wellbeing === 'stable' ? 'success' : child.wellbeing === 'attention' ? 'warning' : 'danger'}
          />
        </div>
      }
    />
  );
}

/** Compact variant used on the home screen and inside a case page. */
export function ChildFollowUpRow({ child, divider }: { child: ChildSummary; divider?: boolean }) {
  const w = WELLBEING[child.wellbeing];
  const label = deadlineLabel(child.nextFollowUp);
  return (
    <ListRow
      to={`/children/${child.id}`}
      divider={divider}
      chevron
      leading={<Avatar name={child.firstName} tone={w.tone} size="sm" />}
      title={`${child.firstName} — ${child.familyName}`}
      meta={
        <Meta>
          <MetaTone tone={label === 'امروز' || label === 'فردا' ? 'warning' : 'neutral'}>پیگیری: {label}</MetaTone>
          <span>{fa(child.followUpCount)} بازدید ثبت‌شده</span>
        </Meta>
      }
      trailing={<Badge tone={w.tone} variant="outline">{w.label}</Badge>}
    />
  );
}
