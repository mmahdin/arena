import { ListRow } from '@/components/ui/ListRow';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { REPORT_STATUS } from '@/lib/labels';
import { reportCompletion } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { faDate, relativeTime } from '@/lib/jalali';
import { visitKindLabel } from '@/lib/visit-workflow';
import type { Report, Visit } from '@/types';

/**
 * A report row is about the report: where it stands and what it needs.
 * The visit it belongs to appears only as one context line («بازدید پیگیری ·
 * ۱۲ مهر») — enough to recognise it, never a second place to manage the
 * visit itself. Actionable reports also carry their completion, so a draft
 * at ۹۰٪ reads differently from one that was never opened.
 */
export function ReportRow({ report, divider, familyName, visit }: {
  report: Report;
  divider?: boolean;
  familyName?: string;
  visit?: Visit;
}) {
  const status = REPORT_STATUS[report.status];
  const actionable = report.status === 'draft' || report.status === 'revision';
  const pct = actionable ? reportCompletion(report).pct : null;
  return (
    <ListRow
      to={`/reports/${report.id}`}
      chevron
      divider={divider}
      title={<PrivacyText value={report.title} interactive={false} />}
      meta={<>
        {familyName && <><PrivacyText value={familyName} interactive={false} /><span aria-hidden>·</span></>}
        {visit
          ? <span>{visitKindLabel(visit)} · {faDate(visit.date, { year: false })}</span>
          : <span>{relativeTime(report.updatedAt)}</span>}
      </>}
      trailing={
        <span className="flex flex-col items-end gap-1">
          <Badge tone={status.tone}>{status.label}</Badge>
          {pct !== null && pct < 100 && (
            <span className="text-2xs font-medium text-ink-3 tabular-nums">{fa(pct)}٪ تکمیل‌شده</span>
          )}
        </span>
      }
    />
  );
}
