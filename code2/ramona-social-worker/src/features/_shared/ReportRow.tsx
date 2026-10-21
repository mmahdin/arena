import { ListRow } from '@/components/ui/ListRow';
import { Badge } from '@/components/ui/Badge';
import { REPORT_STATUS, REPORT_TYPE } from '@/lib/labels';
import { relativeTime } from '@/lib/jalali';
import type { Report } from '@/types';

/** Keep report lists scannable; review comments belong inside the report. */
export function ReportRow({ report, divider, showCase = true }: { report: Report; divider?: boolean; showCase?: boolean }) {
  const status = REPORT_STATUS[report.status];
  return (
    <ListRow
      to={`/reports/${report.id}`}
      chevron
      divider={divider}
      title={report.title}
      meta={
        <>
          <span>{showCase ? report.caseCode : REPORT_TYPE[report.type].label}</span>
          <span aria-hidden>·</span>
          <span>{relativeTime(report.updatedAt)}</span>
        </>
      }
      trailing={<Badge tone={status.tone}>{status.label}</Badge>}
    />
  );
}
