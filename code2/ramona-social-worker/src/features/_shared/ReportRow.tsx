import { ListRow } from '@/components/ui/ListRow';
import { IconTile, REPORT_TYPE_ICON } from '@/components/ui/DomainIcon';
import { Badge } from '@/components/ui/Badge';
import { ProgressRing } from '@/components/ui/Progress';
import { REPORT_STATUS, REPORT_TYPE } from '@/lib/labels';
import { relativeTime } from '@/lib/jalali';
import { reportCompletion } from '@/lib/domain';
import type { Report } from '@/types';
import { Meta } from './Meta';

/**
 * Report row.
 *
 * A draft shows how much is left (ring), a report under review shows who has
 * it, and a returned report shows the reviewer's words — the three states a
 * social worker actually needs to distinguish at a glance.
 */
export function ReportRow({ report, divider }: { report: Report; divider?: boolean }) {
  const status = REPORT_STATUS[report.status];
  const completion = reportCompletion(report);
  const comment = report.comments[report.comments.length - 1];

  return (
    <div className={divider ? 'border-b border-line-soft last:border-0' : undefined}>
      <ListRow
        to={`/reports/${report.id}`}
        chevron
        leading={<IconTile icon={REPORT_TYPE_ICON[report.type]} tone={REPORT_TYPE[report.type].tone} />}
        title={report.title}
        meta={
          <Meta>
            <span className="tabular-nums">{report.caseCode}</span>
            <span>{REPORT_TYPE[report.type].label}</span>
            <span>{relativeTime(report.updatedAt)}</span>
          </Meta>
        }
        trailing={
          report.status === 'draft' ? (
            <ProgressRing value={completion.pct} size={38} />
          ) : (
            <Badge tone={status.tone}>{status.label}</Badge>
          )
        }
      />
      {report.status === 'revision' && comment && (
        <p className="mx-4 mb-3.5 -mt-2 rounded-md border-s-2 border-warning bg-warning-soft px-3 py-2 text-xs leading-relaxed text-warning">
          <span className="font-bold">{comment.author}: </span>
          <span className="clamp-2 text-ink-2">{comment.body}</span>
        </p>
      )}
      {report.status === 'draft' && (
        <p className="mx-4 mb-3.5 -mt-2 text-xs text-ink-3">
          {completion.pct === 100 ? 'آماده ارسال برای بازبینی' : `${(100 - completion.pct).toLocaleString('fa-IR')}٪ باقی‌مانده`}
        </p>
      )}
    </div>
  );
}
