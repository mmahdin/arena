import { Link } from "react-router-dom";
import { ChevronLeft, MessageSquareQuote } from "lucide-react";
import type { Report } from "@/types";
import { formatRelative, parse } from "@/lib/date";
import { reportTypeLabels } from "@/lib/labels";
import { ReportStatusBadge } from "./Badges";
import { ProgressBar } from "@/components/ui/ProgressBar";
import "./ReportCard.css";

export function ReportCard({ report, now }: { report: Report; now: Date }) {
  const needsRevision = report.status === "needs_revision";
  const showProgress =
    (report.status === "draft" || report.status === "submitted") &&
    typeof report.progress === "number" &&
    report.progress < 100;

  return (
    <Link to={`/reports/${report.id}`} className={`report-card ${needsRevision ? "is-returned" : ""}`}>
      <span className="report-card__top">
        <ReportStatusBadge status={report.status} />
        <span className="report-card__type">{reportTypeLabels[report.type]}</span>
      </span>
      <span className="report-card__title">{report.title}</span>
      <span className="report-card__meta">
        {report.caseTitle}
        <span className="report-card__sep" aria-hidden="true">·</span>
        ویرایش {formatRelative(parse(report.updatedAt), now)}
      </span>

      {needsRevision && report.reviewNote ? (
        <span className="report-card__review">
          <MessageSquareQuote size={14} aria-hidden="true" />
          <span className="clamp-2">{report.reviewNote}</span>
        </span>
      ) : null}

      {showProgress ? (
        <span className="report-card__progress">
          <span className="report-card__progress-label">
            <span>تکمیل پیش‌نویس</span>
            <span className="num">{new Intl.NumberFormat("fa-IR").format(report.progress ?? 0)}٪</span>
          </span>
          <ProgressBar value={report.progress ?? 0} label={`پیشرفت ${report.title}`} />
        </span>
      ) : (
        <span className="report-card__footer">
          <span>
            {report.status === "approved"
              ? "تأییدشده توسط سرپرست"
              : report.status === "needs_revision"
                ? "نیازمند اصلاح و ارسال مجدد"
                : report.status === "draft"
                  ? "پیش‌نویس ذخیره‌شده"
                  : "ارسال‌شده برای بررسی"}
          </span>
          <ChevronLeft size={15} aria-hidden="true" />
        </span>
      )}
    </Link>
  );
}
