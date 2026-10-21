import { useNavigate, useParams, useSearchParams, Link } from "react-router-dom";
import {
  CalendarCheck2,
  CalendarClock,
  ChevronLeft,
  FileCheck2,
  FileText,
  FolderSearch,
  Gavel,
  GitBranch,
  ShieldCheck,
  Share2,
  UserPlus,
  Users,
} from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import {
  formatDate,
  formatDayLabel,
  formatRelative,
  formatTime,
  parse,
  startOfDay,
} from "@/lib/date";
import { reportsForCase, visitsForCase } from "@/lib/selectors";
import type { CaseEventKind } from "@/types";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { EmptyState } from "@/components/ui/EmptyState";
import { Badge } from "@/components/ui/Badge";
import { Button } from "@/components/ui/Button";
import { Alert } from "@/components/ui/Alert";
import { useToast } from "@/components/ui/Toast";
import { RiskBadge, WellbeingBadge } from "@/components/domain/Badges";
import { PersonRow } from "@/components/domain/People";
import { CycleDots } from "@/components/domain/MonitoringCard";
import { LocationCard } from "@/components/domain/LocationCard";
import { VisitRow } from "@/components/domain/VisitRow";
import { ReportCard } from "@/components/domain/ReportCard";
import { CaseStageRail } from "./CaseStageRail";
import "./CaseDetailPage.css";

type Tab = "summary" | "visits" | "reports" | "history";

const eventIcons: Record<CaseEventKind, typeof FileText> = {
  referral: UserPlus,
  visit: CalendarCheck2,
  report: FileText,
  stage_change: GitBranch,
  document: FileCheck2,
  decision: Gavel,
};

export function CaseDetailPage() {
  const { caseId } = useParams();
  const data = useWorkspace();
  const now = useNow();
  const navigate = useNavigate();
  const toast = useToast();
  const [searchParams, setSearchParams] = useSearchParams();

  const familyCase = data.cases.find((c) => c.id === caseId);
  const tabParam = searchParams.get("tab") ?? "summary";
  const tab: Tab = (["summary", "visits", "reports", "history"] as const).includes(
    tabParam as Tab
  )
    ? (tabParam as Tab)
    : "summary";

  if (!familyCase) {
    return (
      <>
        <TopBar title="پرونده" onBack={() => navigate("/cases")} />
        <Page>
          <EmptyState
            icon={<FolderSearch size={22} />}
            title="این پرونده یافت نشد"
            description="ممکن است دسترسی شما به این پرونده محدود شده باشد یا پرونده جابه‌جا شده باشد."
            action={
              <Button variant="secondary" onClick={() => navigate("/cases")}>
                بازگشت به پرونده‌ها
              </Button>
            }
          />
        </Page>
      </>
    );
  }

  const caseVisits = visitsForCase(data, familyCase.id);
  const caseReports = reportsForCase(data, familyCase.id);
  const nextVisit = [...caseVisits]
    .filter((v) => v.status === "scheduled" && parse(v.startsAt) >= startOfDay(now))
    .sort((a, b) => parse(a.startsAt).getTime() - parse(b.startsAt).getTime())[0];

  const setTab = (next: Tab) =>
    setSearchParams(next === "summary" ? {} : { tab: next }, { replace: true });

  const share = async () => {
    try {
      await navigator.clipboard.writeText(
        `${location.origin}${location.pathname}#/cases/${familyCase.id}`
      );
      toast.show({ message: "پیوند پرونده کپی شد.", tone: "success" });
    } catch {
      toast.show({ message: "پیوند این پرونده آماده اشتراک‌گذاری است.", tone: "default" });
    }
  };

  return (
    <>
      <TopBar
        title="پرونده خانواده"
        actions={
          <button
            type="button"
            className="icon-btn icon-btn--plain icon-btn--sm"
            aria-label="اشتراک‌گذاری پرونده"
            onClick={share}
          >
            <Share2 size={17} />
          </button>
        }
      />

      <Page>
        <section className="case-hero">
          <div className="case-hero__head">
            <span className="case-hero__mark" aria-hidden="true">
              {familyCase.title.replace(/^خانواده\s+/, "").slice(0, 1)}
            </span>
            <div className="case-hero__identity">
              <h1 className="case-hero__title">{familyCase.title}</h1>
              <p className="case-hero__sub">
                پرونده <span className="num">{familyCase.caseNumber}</span>
                <span className="dot-sep" aria-hidden="true">·</span>
                گشایش {formatDate(parse(familyCase.openedAt))}
              </p>
            </div>
          </div>

          <div className="case-hero__badges">
            <RiskBadge risk={familyCase.riskLevel} />
            {familyCase.tags.map((tag) => (
              <Badge key={tag} tone="neutral">{tag}</Badge>
            ))}
          </div>

          <CaseStageRail stage={familyCase.stage} />
        </section>

        <div className="case-tabs">
          <SegmentedControl<Tab>
            ariaLabel="بخش‌های پرونده"
            value={tab}
            onChange={setTab}
            items={[
              { value: "summary", label: "خلاصه" },
              { value: "visits", label: "بازدیدها" },
              { value: "reports", label: "گزارش‌ها" },
              { value: "history", label: "تاریخچه" },
            ]}
          />
        </div>

        {tab === "summary" ? (
          <div className="case-section">
            <section className="panel">
              <h2 className="panel__title">خلاصه پرونده</h2>
              <p className="panel__body">{familyCase.summary}</p>
            </section>

            <section className="panel">
              <div className="panel__head">
                <h2 className="panel__title">
                  <Users size={15} aria-hidden="true" />
                  کودکان
                </h2>
                <span className="panel__caption num">{familyCase.children.length}</span>
              </div>
              {familyCase.children.length > 0 ? (
                <div className="case-children">
                  {familyCase.children.map((child) => {
                    const monitoring = child.monitoring;
                    return (
                      <div className="case-child" key={child.id}>
                        <PersonRow
                          name={child.fullName}
                          subtitle={
                            <>
                              <span className="num">{new Intl.NumberFormat("fa-IR").format(child.age)}</span>{" "}
                              ساله
                              <span className="dot-sep">·</span>
                              {child.gender === "girl" ? "دختر" : "پسر"}
                            </>
                          }
                          badge={
                            monitoring ? (
                              <WellbeingBadge wellbeing={monitoring.wellbeing} />
                            ) : (
                              <Badge tone="neutral">در مرحله معرفی</Badge>
                            )
                          }
                        />
                        {monitoring ? (
                          <div className="case-child__monitor">
                            <div className="case-child__monitor-row">
                              <span>
                                دوره پایش{" "}
                                <span className="num">{monitoring.cycle}</span> از{" "}
                                <span className="num">{monitoring.totalCycles}</span>
                              </span>
                              {monitoring.nextVisitAt ? (
                                <span className="case-child__next">
                                  <CalendarClock size={12} aria-hidden="true" />
                                  {formatDayLabel(parse(monitoring.nextVisitAt), now)}
                                </span>
                              ) : null}
                            </div>
                            <CycleDots
                              cycle={monitoring.cycle}
                              total={monitoring.totalCycles}
                              tone={
                                monitoring.wellbeing === "needs_attention"
                                  ? "warning"
                                  : monitoring.wellbeing === "critical"
                                    ? "danger"
                                    : "brand"
                              }
                            />
                            {monitoring.note ? (
                              <p className="case-child__note">{monitoring.note}</p>
                            ) : null}
                          </div>
                        ) : null}
                      </div>
                    );
                  })}
                </div>
              ) : (
                <p className="panel__body">
                  این پرونده در مرحله ارزیابی متقاضیان است و تاکنون کودکی به آن واگذار نشده است.
                </p>
              )}
            </section>

            <section className="panel">
              <div className="panel__head">
                <h2 className="panel__title">سرپرستان و متقاضیان</h2>
                <span className="panel__caption num">{familyCase.guardians.length}</span>
              </div>
              <div className="case-guardians">
                {familyCase.guardians.map((guardian) => (
                  <PersonRow
                    key={guardian.id}
                    name={guardian.fullName}
                    subtitle={
                      <>
                        {guardian.relation === "mother"
                          ? "مادر"
                          : guardian.relation === "father"
                            ? "پدر"
                            : "سرپرست"}
                        {guardian.job ? <span className="dot-sep">·</span> : null}
                        {guardian.job}
                      </>
                    }
                    phone={guardian.phone}
                  />
                ))}
              </div>
            </section>

            <section className="panel">
              <h2 className="panel__title">مشخصات پرونده</h2>
              <div className="kv-list">
                <div className="kv-row">
                  <span className="kv-row__label">شماره پرونده</span>
                  <span className="kv-row__value num">{familyCase.caseNumber}</span>
                </div>
                <div className="kv-row">
                  <span className="kv-row__label">آخرین فعالیت</span>
                  <span className="kv-row__value">
                    {formatRelative(parse(familyCase.lastActivityAt), now)}
                  </span>
                </div>
                <div className="kv-row">
                  <span className="kv-row__label">بازدید بعدی</span>
                  <span className="kv-row__value">
                    {nextVisit
                      ? `${formatDayLabel(parse(nextVisit.startsAt), now)}، ${formatTime(parse(nextVisit.startsAt))}`
                      : "برنامه‌ریزی نشده"}
                  </span>
                </div>
              </div>
            </section>

            <LocationCard address={familyCase.address} />

            <Alert tone="neutral" icon={<ShieldCheck size={15} />} title="دسترسی محدود">
              شما بر اساس مسئولیت فعلی به اطلاعات این پرونده دسترسی دارید. تصمیم‌گیری نهایی درباره
              ارزیابی و واگذاری در اختیار مراجع مجاز است.
            </Alert>
          </div>
        ) : null}

        {tab === "visits" ? (
          <div className="case-section">
            {nextVisit ? (
              <Link to={`/visits/${nextVisit.id}`} className="next-visit">
                <span className="next-visit__icon" aria-hidden="true">
                  <CalendarClock size={18} />
                </span>
                <span className="next-visit__text">
                  <span className="next-visit__label">بازدید بعدی</span>
                  <span className="next-visit__value">
                    {formatDayLabel(parse(nextVisit.startsAt), now)}، ساعت{" "}
                    <span className="num">{formatTime(parse(nextVisit.startsAt))}</span>
                  </span>
                </span>
                <ChevronLeft size={17} className="next-visit__chev" aria-hidden="true" />
              </Link>
            ) : null}

            {caseVisits.length > 0 ? (
              <ul className="case-visit-list">
                {caseVisits.map((visit) => (
                  <li key={visit.id}>
                    <VisitRow visit={visit} compact />
                  </li>
                ))}
              </ul>
            ) : (
              <EmptyState
                icon={<CalendarCheck2 size={22} />}
                title="بازدیدی برای این پرونده ثبت نشده"
                description="پس از زمان‌بندی بازدید توسط سرپرست، در این بخش نمایش داده می‌شود."
                compact
              />
            )}
          </div>
        ) : null}

        {tab === "reports" ? (
          <div className="case-section">
            {caseReports.length > 0 ? (
              <ul className="case-report-list">
                {caseReports.map((report) => (
                  <li key={report.id}>
                    <ReportCard report={report} now={now} />
                  </li>
                ))}
              </ul>
            ) : (
              <EmptyState
                icon={<FileText size={22} />}
                title="گزارشی برای این پرونده ثبت نشده"
                description="گزارش‌های بازدید و ارزیابی این پرونده در این بخش جمع می‌شوند."
                compact
              />
            )}
          </div>
        ) : null}

        {tab === "history" ? (
          <div className="case-section">
            <ol className="history">
              {[...familyCase.events]
                .sort((a, b) => parse(b.at).getTime() - parse(a.at).getTime())
                .map((event) => {
                  const Icon = eventIcons[event.kind];
                  return (
                    <li className="history__item" key={event.id}>
                      <span className="history__icon" aria-hidden="true">
                        <Icon size={15} />
                      </span>
                      <div className="history__content">
                        <p className="history__title">{event.title}</p>
                        {event.note ? <p className="history__note">{event.note}</p> : null}
                        <p className="history__meta">
                          {formatDate(parse(event.at))}، ساعت{" "}
                          <span className="num">{formatTime(parse(event.at))}</span>
                          {event.actor ? (
                            <>
                              <span className="dot-sep" aria-hidden="true">·</span>
                              {event.actor}
                            </>
                          ) : null}
                        </p>
                      </div>
                    </li>
                  );
                })}
            </ol>
          </div>
        ) : null}
      </Page>
    </>
  );
}
