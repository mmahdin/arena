import { ListRow } from '@/components/ui/ListRow';
import { Badge } from '@/components/ui/Badge';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { REPORT_STATUS } from '@/lib/labels';
import { relativeTime } from '@/lib/jalali';
import type { Report } from '@/types';

/** Review comments stay inside the editor; each row exposes who and what needs doing. */
export function ReportRow({ report, divider, familyName }: { report: Report; divider?: boolean; familyName?: string }) {
  const status = REPORT_STATUS[report.status];
  return (
    <ListRow
      to={`/reports/${report.id}`}
      chevron
      divider={divider}
      title={<PrivacyText value={report.title} interactive={false} />}
      meta={<>
        {familyName && <><PrivacyText value={familyName} interactive={false} /><span aria-hidden>·</span></>}
        <span>{relativeTime(report.updatedAt)}</span>
      </>}
      trailing={<Badge tone={status.tone}>{status.label}</Badge>}
    />
  );
}
