import { useState } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import {
  CalendarDays,
  Check,
  ChevronLeft,
  Clock,
  FolderClosed,
  ListChecks,
  Navigation,
  Phone,
  Share2,
  ShieldAlert,
  ShieldCheck,
  StickyNote,
  Target,
  UserRound,
  XCircle,
  RotateCcw,
  CheckCircle2,
} from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import {
  addMinutes,
  formatDayLabel,
  formatDayMonth,
  formatRelative,
  formatTime,
  parse,
} from "@/lib/date";
import { relationLabels, visitKindLabels, visitStatusLabels } from "@/lib/labels";
import { telHref } from "@/lib/phone";
import { cn } from "@/lib/cn";
import type { VisitStatus } from "@/types";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { Button } from "@/components/ui/Button";
import { Alert } from "@/components/ui/Alert";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { EmptyState } from "@/components/ui/EmptyState";
import { Sheet } from "@/components/ui/Sheet";
import { useToast } from "@/components/ui/Toast";
import { LocationCard } from "@/components/domain/LocationCard";
import { PersonRow } from "@/components/domain/People";
import {
  CaseStageBadge,
  ReportStatusBadge,
  VisitKindIcon,
  VisitStatusBadge,
} from "@/components/domain/Badges";
import "./VisitDetailPage.css";

export function VisitDetailPage() {
  const { visitId } = useParams();
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const actions = useAppActions();
  const toast = useToast();
  const [sheetOpen, setSheetOpen] = useState(false);

  const visit = data.visits.find((v) => v.id === visitId);
  if (!visit) {
    return (
      <>
        <TopBar title="جزئیات بازدید" onBack={() => navigate("/visits")} />
        <Page>
          <EmptyState
            icon={<CalendarDays size={22} />}
            title="این بازدید یافت نشد"
            description="ممکن است بازدید حذف شده باشد یا دسترسی شما به آن محدود شده است."
            action={
              <Button variant="secondary" onClick={() => navigate("/visits")}>
                بازگشت به بازدیدها
              </Button>
            }
          />
        </Page>
      </>
    );
  }

  const familyCase = data.cases.find((c) => c.id === visit.caseId);
  const report = visit.reportId ? data.reports.find((r) => r.id === visit.reportId) : undefined;
  const start = parse(visit.startsAt);
  const end = addMinutes(start, visit.durationMin);
  const doneCount = visit.checklist.filter((item) => item.done).length;
  const checklistProgress = visit.checklist.length
    ? (doneCount / visit.checklist.length) * 100
    : 0;
  const children = familyCase?.children ?? [];
  const isScheduled = visit.status === "scheduled";
  const isCompleted = visit.status === "completed";

  const handleOutcome = (status: VisitStatus) => {
    actions.setVisitStatus(visit.id, status);
    setSheetOpen(false);
    if (status === "completed") {
      toast.show({
        message: "نتیجه بازدید ثبت شد.",
        tone: "success",
        action: {
          label: report ? "مشاهده گزارش" : "نوشتن گزارش",
          onClick: () =>
            navigate(report ? `/reports/${report.id}` : `/reports/new?visitId=${visit.id}`),
        },
      });
    } else if (status === "needs_followup") {
      toast.show({ message: "بازدید به «نیاز به پیگیری» تغییر کرد.", tone: "default" });
    } else {
      toast.show({ message: "بازدید لغو شد.", tone: "default" });
    }
  };

  const share = async () => {
    try {
      await navigator.clipboard.writeText(`${location.origin}${location.pathname}#/visits/${visit.id}`);
      toast.show({ message: "پیوند بازدید کپی شد.", tone: "success" });
    } catch {
      toast.show({ message: "پیوند این بازدید آماده اشتراک‌گذاری است.", tone: "default" });
    }
  };

  return (
    <>
      <TopBar
        title="جزئیات بازدید"
        actions={
          <button
            type="button"
            className="icon-btn icon-btn--plain icon-btn--sm"
            aria-label="اشتراک‌گذاری بازدید"
            onClick={share}
          >
            <Share2 size={18} />
          </button>
        }
      />

      <Page className="page--with-action-bar">
        <section className="visit-hero">
          <div className="visit-hero__top">
            <span className="visit-hero__kind">
              <VisitKindIcon kind={visit.kind} size={15} />
              {visitKindLabels[visit.kind]}
            </span>
            <VisitStatusBadge status={visit.status} />
          </div>

          <div>
            <Link to={`/cases/${visit.caseId}`} className="visit-hero__title">
              {visit.caseTitle}
            </Link>
            <p className="visit-hero__meta">
              {children.length > 0
                ? children
                    .map(
                      (child) =>
                        `${child.fullName}، ${new Intl.NumberFormat("fa-IR").format(child.age)} ساله`
                    )
                    .join(" · ")
                : "بازدید متقاضیان"}
              {familyCase ? (
                <>
                  <span className="dot-sep" aria-hidden="true">·</span>
                  پرونده <span className="num">{familyCase.caseNumber}</span>
                </>
              ) : null}
            </p>
          </div>

          <div className="visit-hero__when">
            <div className="visit-hero__when-block">
              <CalendarDays size={16} aria-hidden="true" />
              <div>
                <span className="visit-hero__when-label">تاریخ</span>
                <span className="visit-hero__when-value">
                  {formatDayLabel(start, now)}، {formatDayMonth(start)}
                </span>
              </div>
            </div>
            <div className="visit-hero__when-block">
              <Clock size={16} aria-hidden="true" />
              <div>
                <span className="visit-hero__when-label">زمان</span>
                <span className="visit-hero__when-value num">
                  {formatTime(start)} تا {formatTime(end)}
                </span>
              </div>
            </div>
          </div>

          <div className="visit-hero__actions">
            <a className="btn btn--secondary" href={telHref(visit.contactPhone ?? "")}>
              <Phone size={16} aria-hidden="true" />
              تماس تلفنی
            </a>
            <button
              type="button"
              className="btn btn--soft"
              onClick={() =>
                document.getElementById("visit-location")?.scrollIntoView({ behavior: "smooth" })
              }
            >
              <Navigation size={16} aria-hidden="true" />
              مسیر و آدرس
            </button>
          </div>
        </section>

        {visit.mandatory ? (
          <Alert tone="warning" icon={<ShieldAlert size={15} />}>
            <strong>بازدید الزامی است.</strong> لغو یا جابجایی زمان این بازدید نیازمند تأیید سرپرست
            تیم است.
          </Alert>
        ) : null}

        <section className="panel">
          <div className="panel__head">
            <h2 className="panel__title">
              <Target size={15} aria-hidden="true" />
              هدف بازدید
            </h2>
          </div>
          <p className="panel__body">{visit.objective}</p>
        </section>

        <section className="panel">
          <div className="panel__head">
            <h2 className="panel__title">
              <ListChecks size={15} aria-hidden="true" />
              چک‌لیست بازدید
            </h2>
            <span className="panel__caption num">
              {new Intl.NumberFormat("fa-IR").format(doneCount)} از{" "}
              {new Intl.NumberFormat("fa-IR").format(visit.checklist.length)}
            </span>
          </div>
          {visit.checklist.length > 0 ? (
            <>
              <ProgressBar value={checklistProgress} label="پیشرفت چک‌لیست" />
              <ul className="checklist">
                {visit.checklist.map((item) => (
                  <li key={item.id}>
                    <button
                      type="button"
                      role="checkbox"
                      aria-checked={item.done}
                      className={cn("checklist__item", item.done && "is-done")}
                      onClick={() => actions.toggleVisitChecklist(visit.id, item.id)}
                    >
                      <span className="checklist__box" aria-hidden="true">
                        <Check size={12} strokeWidth={3} />
                      </span>
                      <span>{item.label}</span>
                    </button>
                  </li>
                ))}
              </ul>
            </>
          ) : (
            <p className="panel__body">برای این بازدید چک‌لیستی تعریف نشده است.</p>
          )}
        </section>

        {report ? (
          <Link to={`/reports/${report.id}`} className="visit-report-link">
            <span className="visit-report-link__head">
              <span className="panel__title">گزارش مرتبط</span>
              <ReportStatusBadge status={report.status} />
            </span>
            <span className="visit-report-link__title">{report.title}</span>
            <span className="visit-report-link__meta">
              آخرین ویرایش {formatRelative(parse(report.updatedAt), now)}
              <ChevronLeft size={15} aria-hidden="true" />
            </span>
          </Link>
        ) : null}

        {visit.notes ? (
          <section className="panel panel--muted">
            <h2 className="panel__title">
              <StickyNote size={15} aria-hidden="true" />
              یادداشت هماهنگی
            </h2>
            <p className="panel__body">{visit.notes}</p>
          </section>
        ) : null}

        <div id="visit-location">
          <LocationCard address={visit.address} title={visit.contactName} />
        </div>

        <section className="panel">
          <h2 className="panel__title">
            <UserRound size={15} aria-hidden="true" />
            تماس و همراهی
          </h2>
          {familyCase?.guardians.map((guardian) => (
            <PersonRow
              key={guardian.id}
              name={guardian.fullName}
              subtitle={
                <>
                  {relationLabels[guardian.relation]}
                  {guardian.job ? <span className="dot-sep">·</span> : null}
                  {guardian.job}
                </>
              }
              phone={guardian.phone}
            />
          ))}
          {!familyCase ? (
            <p className="panel__body">{visit.contactName} — {visit.contactPhone}</p>
          ) : null}
        </section>

        {familyCase ? (
          <Link to={`/cases/${familyCase.id}`} className="case-link">
            <span className="icon-tile icon-tile--brand" aria-hidden="true">
              <FolderClosed size={17} />
            </span>
            <span className="case-link__text">
              <span className="case-link__title">پرونده {familyCase.title}</span>
              <span className="case-link__sub">
                <CaseStageBadge stage={familyCase.stage} />
              </span>
            </span>
            <ChevronLeft size={17} className="case-link__chev" aria-hidden="true" />
          </Link>
        ) : null}

        <p className="privacy-note">
          <ShieldCheck size={14} aria-hidden="true" />
          اطلاعات این بازدید محرمانه است و تنها برای افراد مجاز پرونده قابل مشاهده است.
        </p>
      </Page>

      <div className="action-bar">
        <button
          type="button"
          className="action-bar__icon"
          aria-label="جزئیات پرونده"
          onClick={() => navigate(`/cases/${visit.caseId}`)}
        >
          <FolderClosed size={19} aria-hidden="true" />
        </button>
        {isCompleted || report ? (
          <Button
            className="action-bar__primary"
            onClick={() =>
              report
                ? navigate(`/reports/${report.id}`)
                : navigate(`/reports/new?visitId=${visit.id}`)
            }
          >
            {report ? "مشاهده گزارش بازدید" : "نوشتن گزارش بازدید"}
          </Button>
        ) : visit.status === "canceled" ? (
          <Button className="action-bar__primary" onClick={() => setSheetOpen(true)}>
            زمان‌بندی مجدد بازدید
          </Button>
        ) : (
          <Button className="action-bar__primary" onClick={() => setSheetOpen(true)}>
            ثبت نتیجه بازدید
          </Button>
        )}
      </div>

      <Sheet
        open={sheetOpen}
        onClose={() => setSheetOpen(false)}
        title="ثبت نتیجه بازدید"
        description={`${visit.caseTitle} · ${formatDayLabel(start, now)}، ساعت ${formatTime(start)}`}
      >
        <div className="outcomes">
          <button type="button" className="outcome" onClick={() => handleOutcome("completed")}>
            <span className="outcome__icon is-success" aria-hidden="true">
              <CheckCircle2 size={18} />
            </span>
            <span className="outcome__text">
              <span className="outcome__title">بازدید انجام شد</span>
              <span className="outcome__desc">
                نتیجه در تاریخچه پرونده ثبت می‌شود و می‌توانید گزارش را بنویسید.
              </span>
            </span>
            <ChevronLeft size={17} className="outcome__chev" aria-hidden="true" />
          </button>
          <button type="button" className="outcome" onClick={() => handleOutcome("needs_followup")}>
            <span className="outcome__icon is-warning" aria-hidden="true">
              <RotateCcw size={18} />
            </span>
            <span className="outcome__text">
              <span className="outcome__title">نیاز به بازدید مجدد</span>
              <span className="outcome__desc">
                شرایط نیازمند پیگیری است؛ بازدید بعدی باید زمان‌بندی شود.
              </span>
            </span>
            <ChevronLeft size={17} className="outcome__chev" aria-hidden="true" />
          </button>
          <button type="button" className="outcome" onClick={() => handleOutcome("canceled")}>
            <span className="outcome__icon is-neutral" aria-hidden="true">
              <XCircle size={18} />
            </span>
            <span className="outcome__text">
              <span className="outcome__title">بازدید لغو شد</span>
              <span className="outcome__desc">
                دلیل لغو در پرونده ثبت می‌شود و هماهنگی مجدد لازم است.
              </span>
            </span>
            <ChevronLeft size={17} className="outcome__chev" aria-hidden="true" />
          </button>
        </div>
        <p className="outcomes__note">
          <ShieldCheck size={14} aria-hidden="true" />
          وضعیت ثبت‌شده برای سرپرست تیم و در تاریخچه پرونده قابل مشاهده است.
        </p>
        <p className="outcomes__hint">
          وضعیت فعلی: {visitStatusLabels[visit.status].label}
          {isScheduled ? "" : " — با ثبت نتیجه، وضعیت به‌روزرسانی می‌شود."}
        </p>
      </Sheet>
    </>
  );
}
