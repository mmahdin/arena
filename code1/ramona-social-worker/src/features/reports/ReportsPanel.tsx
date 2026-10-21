import { useMemo, useState } from "react";
import { useNavigate } from "react-router-dom";
import { ChevronLeft, FilePlus2, Files } from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import { parse } from "@/lib/date";
import type { ReportStatus } from "@/types";
import { reportStatusLabels } from "@/lib/labels";
import { FilterChips } from "@/components/ui/FilterChips";
import { EmptyState } from "@/components/ui/EmptyState";
import { ReportCard } from "@/components/domain/ReportCard";
import "./ReportsPanel.css";

type Filter = "all" | ReportStatus;

const statusOrder: Record<ReportStatus, number> = {
  needs_revision: 0,
  draft: 1,
  submitted: 2,
  approved: 3,
};

const filters: { value: Filter; label: string }[] = [
  { value: "all", label: "همه" },
  { value: "needs_revision", label: "نیازمند اصلاح" },
  { value: "draft", label: "پیش‌نویس" },
  { value: "submitted", label: "در انتظار بررسی" },
  { value: "approved", label: "تأییدشده" },
];

export function ReportsPanel() {
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const [filter, setFilter] = useState<Filter>("all");

  const sorted = useMemo(
    () =>
      [...data.reports].sort((a, b) => {
        const order = statusOrder[a.status] - statusOrder[b.status];
        if (order !== 0) return order;
        return parse(b.updatedAt).getTime() - parse(a.updatedAt).getTime();
      }),
    [data.reports]
  );

  const counts = useMemo(() => {
    const base: Record<Filter, number> = {
      all: data.reports.length,
      needs_revision: 0,
      draft: 0,
      submitted: 0,
      approved: 0,
    };
    data.reports.forEach((r) => {
      base[r.status] += 1;
    });
    return base;
  }, [data.reports]);

  const filtered = filter === "all" ? sorted : sorted.filter((r) => r.status === filter);

  return (
    <div className="reports-panel">
      <button type="button" className="report-new" onClick={() => navigate("/reports/new")}>
        <span className="report-new__icon" aria-hidden="true">
          <FilePlus2 size={18} />
        </span>
        <span className="report-new__text">
          <span className="report-new__title">گزارش جدید</span>
          <span className="report-new__sub">نوشتن گزارش بازدید یا ارزیابی اجتماعی</span>
        </span>
        <ChevronLeft size={17} className="report-new__chev" aria-hidden="true" />
      </button>

      <FilterChips<Filter>
        ariaLabel="فیلتر گزارش‌ها"
        items={filters
          .filter((f) => f.value === "all" || counts[f.value] > 0)
          .map((f) => ({ ...f, count: counts[f.value] }))}
        value={filter}
        onChange={setFilter}
      />

      {filtered.length > 0 ? (
        <ul className="reports-list">
          {filtered.map((report) => (
            <li key={report.id}>
              <ReportCard report={report} now={now} />
            </li>
          ))}
        </ul>
      ) : (
        <EmptyState
          icon={<Files size={22} />}
          title="گزارشی با این وضعیت نیست"
          description={`گزارشی در وضعیت «${filter === "all" ? "همه" : reportStatusLabels[filter].label}» یافت نشد.`}
          compact
        />
      )}
    </div>
  );
}
