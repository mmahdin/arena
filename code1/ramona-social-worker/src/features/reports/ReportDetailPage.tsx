import { useNavigate, useParams, Link } from "react-router-dom";
import {
  CalendarClock,
  CheckCircle2,
  ChevronLeft,
  FileQuestion,
  MessageSquareWarning,
  Pencil,
  Share2,
  ShieldCheck,
  Star,
  UserRound,
} from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import {
  formatDate,
  formatDateTime,
  formatRelative,
  parse,
} from "@/lib/date";
import { reportTypeLabels } from "@/lib/labels";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { Button } from "@/components/ui/Button";
import { Alert } from "@/components/ui/Alert";
import { Badge } from "@/components/ui/Badge";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { EmptyState } from "@/components/ui/EmptyState";
import { ReportStatusBadge } from "@/components/domain/Badges";
import { useToast } from "@/components/ui/Toast";
import "./ReportDetailPage.css";

export function ReportDetailPage() {
  const { reportId } = useParams();
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const toast = useToast();

  const report = data.reports.find((r) => r.id === reportId);

  if (!report) {
    return (
      <>
        <TopBar title="گزارش" onBack={() => navigate("/tasks?view=reports")} />
        <Page>
          <EmptyState
            icon={<FileQuestion size={22} />}
            title="این گزارش یافت نشد"
            description="ممکن است گزارش حذف شده باشد یا دسترسی شما به آن محدود شده است."
            action={
              <Button variant="secondary" onClick={() => navigate("/tasks?view=reports")}>
                بازگشت به گزارش‌ها
              </Button>
            }
          />
        </Page>
      </>
    );
  }

  const canEdit = report.status === "draft" || report.status === "needs_revision";
  const isReturned = report.status === "needs_revision";
  const familyCase = data.cases.find((c) => c.id === report.caseId);

  const share = async () => {
    try {
      await navigator.clipboard.writeText(`${location.origin}${location.pathname}#/reports/${report.id}`);
      toast.show({ message: "پیوند گزارش کپی شد.", tone: "success" });
    } catch {
      toast.show({ message: "پیوند این گزارش آماده اشتراک‌گذاری است.", tone: "default" });
    }
  };

  return (
    <>
      <TopBar
        title="گزارش مددکاری"
        actions={
          <>
            <button
              type="button"
              className="icon-btn icon-btn--plain icon-btn--sm"
              aria-label="اشتراک‌گذاری گزارش"
              onClick={share}
            >
              <Share2 size={17} />
            </button>
            {canEdit ? (
              <Link
                to={`/reports/${report.id}/edit`}
                className="icon-btn icon-btn--plain icon-btn--sm"
                aria-label="ویرایش گزارش"
              >
                <Pencil size={17} />
              </Link>
            ) : null}
          </>
        }
      />

      <Page className={canEdit ? "page--with-action-bar" : undefined}>
        <header className="report-head">
          <div className="report-head__top">
            <ReportStatusBadge status={report.status} />
            <span className="report-head__type">{reportTypeLabels[report.type]}</span>
          </div>
          <h1 className="report-head__title">{report.title}</h1>
          <div className="report-head__meta">
            <Link to={`/cases/${report.caseId}`} className="report-head__case">
              {report.caseTitle}
              <ChevronLeft size={13} aria-hidden="true" />
            </Link>
            <span className="dot-sep" aria-hidden="true">·</span>
            <span>{report.authorName}</span>
            <span className="dot-sep" aria-hidden="true">·</span>
            <span>ویرایش {formatRelative(parse(report.updatedAt), now)}</span>
          </div>
          {report.status === "draft" && typeof report.progress === "number" ? (
            <div className="report-head__progress">
              <div className="report-head__progress-label">
                <span>تکمیل پیش‌نویس</span>
                <span className="num">{new Intl.NumberFormat("fa-IR").format(report.progress)}٪</span>
              </div>
              <ProgressBar value={report.progress} label="پیشرفت گزارش" />
            </div>
          ) : null}
        </header>

        {isReturned ? (
          <Alert
            tone="warning"
            icon={<MessageSquareWarning size={15} />}
            title="این گزارش برای اصلاح بازگردانده شده است"
          >
            <p className="report-review-note">{report.reviewNote}</p>
            {report.reviewedBy ? (
              <p className="report-review-by">— {report.reviewedBy}، سرپرست تیم مددکاری</p>
            ) : null}
          </Alert>
        ) : null}

        {report.status === "draft" ? (
          <Alert tone="info" icon={<Pencil size={15} />} title="این گزارش پیش‌نویس است">
            تا زمانی که گزارش را برای بررسی ارسال نکنید، برای سرپرست تیم قابل مشاهده نیست.
          </Alert>
        ) : null}

        <article className="report-doc">
          <div className="doc">
            {report.sections.map((section) => (
              <section className="doc__section" key={section.id}>
                <h2 className="doc__section-title">{section.label}</h2>
                {section.value.trim() ? (
                  <p className="doc__text">{section.value}</p>
                ) : (
                  <p className="doc__text doc__text--empty">این بخش هنوز تکمیل نشده است.</p>
                )}
              </section>
            ))}
          </div>

          {typeof report.rating === "number" ? (
            <>
              <hr className="doc__divider" />
              <div className="report-rating">
                <span className="report-rating__label">ارزیابی کلی شرایط</span>
                <span className="rating-stars" role="img" aria-label={`امتیاز ${report.rating} از ۵`}>
                  {Array.from({ length: 5 }).map((_, index) => (
                    <Star
                      key={index}
                      size={16}
                      className={index < report.rating! ? "is-filled" : ""}
                      aria-hidden="true"
                    />
                  ))}
                </span>
              </div>
            </>
          ) : null}
        </article>

        {report.status === "approved" ? (
          <section className="report-approval">
            <span className="report-approval__icon" aria-hidden="true">
              <CheckCircle2 size={18} />
            </span>
            <div className="report-approval__text">
              <p className="report-approval__title">این گزارش تأیید شده است</p>
              <p className="report-approval__meta">
                {report.reviewedBy ?? "سرپرست تیم"}
                {report.submittedAt ? ` · ${formatDate(parse(report.submittedAt))}` : ""}
              </p>
            </div>
            <Badge tone="success" variant="outline">نهایی</Badge>
          </section>
        ) : null}

        <section className="panel">
          <h2 className="panel__title">سوابق گزارش</h2>
          <div className="kv-list">
            <div className="kv-row">
              <span className="kv-row__label">
                <UserRound size={13} aria-hidden="true" />
                نویسنده
              </span>
              <span className="kv-row__value">{report.authorName}</span>
            </div>
            {report.submittedAt ? (
              <div className="kv-row">
                <span className="kv-row__label">
                  <CalendarClock size={13} aria-hidden="true" />
                  ارسال برای بررسی
                </span>
                <span className="kv-row__value">{formatDateTime(parse(report.submittedAt))}</span>
              </div>
            ) : null}
            <div className="kv-row">
              <span className="kv-row__label">
                <CalendarClock size={13} aria-hidden="true" />
                آخرین ویرایش
              </span>
              <span className="kv-row__value">{formatDateTime(parse(report.updatedAt))}</span>
            </div>
            {familyCase ? (
              <div className="kv-row">
                <span className="kv-row__label">پرونده</span>
                <Link to={`/cases/${familyCase.id}`} className="kv-row__value text-brand">
                  {familyCase.title} · {familyCase.caseNumber}
                </Link>
              </div>
            ) : null}
          </div>
        </section>

        <p className="privacy-note">
          <ShieldCheck size={14} aria-hidden="true" />
          محتوای این گزارش محرمانه است و تنها برای مددکار، سرپرست تیم و افراد مجاز پرونده قابل
          مشاهده است.
        </p>
      </Page>

      {canEdit ? (
        <div className="action-bar">
          <Button
            className="action-bar__primary"
            onClick={() => navigate(`/reports/${report.id}/edit`)}
          >
            {isReturned ? "اصلاح و ارسال مجدد" : "ادامه نوشتن گزارش"}
          </Button>
        </div>
      ) : null}
    </>
  );
}
