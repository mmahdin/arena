import { useEffect, useMemo, useState } from "react";
import { useNavigate, useParams, useSearchParams } from "react-router-dom";
import {
  Check,
  Cloud,
  CloudOff,
  FileWarning,
  Lock,
  Send,
  ShieldCheck,
  Star,
} from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import { formatTime, parse, toISO } from "@/lib/date";
import { reportTypeLabels } from "@/lib/labels";
import { cn } from "@/lib/cn";
import type { Report, ReportStatus, ReportType, VisitKind } from "@/types";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { Alert } from "@/components/ui/Alert";
import { Button } from "@/components/ui/Button";
import { Field, Input, Select, Textarea } from "@/components/ui/Field";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { EmptyState } from "@/components/ui/EmptyState";
import { useToast } from "@/components/ui/Toast";
import "./ReportEditorPage.css";

type Section = { id: string; label: string; value: string };

const templates: Record<ReportType, { id: string; label: string }[]> = {
  monitoring: [
    { id: "housing", label: "شرایط مسکن و محیط زندگی" },
    { id: "health", label: "وضعیت سلامت و تغذیه" },
    { id: "education", label: "شرایط تحصیلی" },
    { id: "attachment", label: "تعامل عاطفی و پیوند خانوادگی" },
    { id: "conclusion", label: "جمع‌بندی و پیشنهاد" },
  ],
  social_assessment: [
    { id: "housing", label: "شرایط مسکن و محیط زندگی" },
    { id: "economy", label: "وضعیت اقتصادی و اشتغال" },
    { id: "motivation", label: "انگیزه و انتظارات متقاضیان" },
    { id: "capacity", label: "توانمندی‌های مراقبتی خانواده" },
    { id: "conclusion", label: "جمع‌بندی و پیشنهاد" },
  ],
  placement: [
    { id: "session", label: "شرح جلسه آشنایی" },
    { id: "observation", label: "مشاهدات تعامل کودک و خانواده" },
    { id: "readiness", label: "آمادگی خانواده و کودک" },
    { id: "plan", label: "اقدامات بعدی" },
  ],
  home_visit: [
    { id: "observation", label: "مشاهدات بازدید" },
    { id: "environment", label: "شرایط محیط و ایمنی" },
    { id: "interview", label: "گفت‌وگو با اعضای خانواده" },
    { id: "conclusion", label: "جمع‌بندی و پیشنهاد" },
  ],
};

const typeOptions: ReportType[] = ["monitoring", "social_assessment", "placement", "home_visit"];

const ratingLabels: Record<number, string> = {
  1: "نامناسب",
  2: "نیازمند توجه",
  3: "قابل قبول",
  4: "مطلوب",
  5: "بسیار مطلوب",
};

function typeFromVisit(kind: VisitKind): ReportType {
  if (kind === "monitoring" || kind === "follow_up") return "monitoring";
  if (kind === "initial_assessment") return "social_assessment";
  if (kind === "introduction" || kind === "placement") return "placement";
  return "home_visit";
}

function buildSections(type: ReportType, current: Section[] = []): Section[] {
  return templates[type].map((template) => {
    const existing = current.find((s) => s.id === template.id);
    return { ...template, value: existing?.value ?? "" };
  });
}

export function ReportEditorPage({ mode }: { mode: "create" | "edit" }) {
  const { reportId } = useParams();
  const [searchParams] = useSearchParams();
  const data = useWorkspace();
  const now = useNow();
  const actions = useAppActions();
  const toast = useToast();
  const navigate = useNavigate();

  const existing = mode === "edit" ? data.reports.find((r) => r.id === reportId) : undefined;
  const visitId = searchParams.get("visitId") ?? existing?.visitId;
  const visit = visitId ? data.visits.find((v) => v.id === visitId) : undefined;
  const initialCaseId =
    existing?.caseId ?? visit?.caseId ?? searchParams.get("caseId") ?? "";
  const initialType: ReportType = existing?.type ?? (visit ? typeFromVisit(visit.kind) : "social_assessment");
  const initialCase = data.cases.find((c) => c.id === initialCaseId);

  const [caseId, setCaseId] = useState(initialCaseId);
  const [type, setType] = useState<ReportType>(initialType);
  const [title, setTitle] = useState(
    existing?.title ??
      (initialCase ? `${reportTypeLabels[initialType]} — ${initialCase.title}` : "")
  );
  const [sections, setSections] = useState<Section[]>(
    existing ? existing.sections.map(({ id, label, value }) => ({ id, label, value })) : buildSections(initialType)
  );
  const [rating, setRating] = useState<number>(existing?.rating ?? 0);
  const [status, setStatus] = useState<ReportStatus>(existing?.status ?? "draft");
  const [dirty, setDirty] = useState(false);
  const [savedAt, setSavedAt] = useState<string | null>(existing?.updatedAt ?? null);

  const selectedCase = data.cases.find((c) => c.id === caseId);
  const canEdit = mode === "create" || status === "draft" || status === "needs_revision";

  const filled = sections.filter((section) => section.value.trim().length > 24).length;
  const progress = sections.length ? Math.round((filled / sections.length) * 100) : 0;
  const showRating = type === "monitoring" || type === "social_assessment";
  const canSubmit = title.trim().length > 2 && progress >= 60 && Boolean(caseId);

  const patch = useMemo(
    () => ({
      title: title.trim(),
      type,
      sections,
      rating: rating || undefined,
      caseId,
      caseTitle: selectedCase?.title ?? existing?.caseTitle ?? "بدون پرونده",
    }),
    [title, type, sections, rating, caseId, selectedCase?.title, existing?.caseTitle]
  );

  /* Autosave drafts after a short pause, mirroring the eventual backend behaviour. */
  useEffect(() => {
    if (!existing || !dirty || !canEdit) return;
    const timer = window.setTimeout(() => {
      actions.saveReport(existing.id, patch);
      setDirty(false);
      setSavedAt(toISO(now));
    }, 1600);
    return () => window.clearTimeout(timer);
  }, [existing, dirty, canEdit, patch, actions, now]);

  if (mode === "edit" && !existing) {
    return (
      <>
        <TopBar title="ویرایش گزارش" onBack={() => navigate("/tasks?view=reports")} />
        <Page>
          <EmptyState
            icon={<FileWarning size={22} />}
            title="این گزارش یافت نشد"
            description="ممکن است گزارش حذف شده باشد یا دسترسی شما به آن محدود شده است."
          />
        </Page>
      </>
    );
  }

  if (mode === "edit" && existing && !canEdit) {
    return (
      <>
        <TopBar title="گزارش" />
        <Page>
          <Alert tone="info" icon={<Lock size={15} />} title="این گزارش قابل ویرایش نیست">
            گزارش در وضعیت «{status === "submitted" ? "در انتظار بررسی" : "تأییدشده"}» قرار دارد و
            پس از ارسال، ویرایش آن توسط مددکار امکان‌پذیر نیست.
          </Alert>
          <Button variant="secondary" onClick={() => navigate(`/reports/${existing.id}`)}>
            مشاهده گزارش
          </Button>
        </Page>
      </>
    );
  }

  const markDirty = () => setDirty(true);

  const handleType = (next: ReportType) => {
    setType(next);
    setSections((current) => buildSections(next, current));
    if (!title.trim() || title.includes("—")) {
      setTitle(selectedCase ? `${reportTypeLabels[next]} — ${selectedCase.title}` : "");
    }
    markDirty();
  };

  const handleCase = (nextCaseId: string) => {
    setCaseId(nextCaseId);
    const target = data.cases.find((c) => c.id === nextCaseId);
    if (target && (!title.trim() || title.includes("—"))) {
      setTitle(`${reportTypeLabels[type]} — ${target.title}`);
    }
    markDirty();
  };

  const baseReport = (nextStatus: ReportStatus): Report => ({
    id: existing?.id ?? `report-${Date.now()}`,
    title: title.trim() || "گزارش بدون عنوان",
    type,
    status: nextStatus,
    caseId,
    caseTitle: selectedCase?.title ?? "بدون پرونده",
    visitId: visit?.id ?? existing?.visitId,
    authorName: data.profile.fullName,
    updatedAt: toISO(now),
    submittedAt: nextStatus === "submitted" ? toISO(now) : existing?.submittedAt,
    reviewedBy: nextStatus === "submitted" ? undefined : existing?.reviewedBy,
    reviewNote: nextStatus === "submitted" ? undefined : existing?.reviewNote,
    rating: rating || undefined,
    sections,
    progress,
  });

  const handleSaveDraft = () => {
    const report = baseReport(status === "needs_revision" ? "needs_revision" : "draft");
    if (existing) {
      actions.saveReport(existing.id, {
        ...patch,
        status: report.status,
        updatedAt: report.updatedAt,
      });
      setSavedAt(report.updatedAt);
      setDirty(false);
      toast.show({ message: "پیش‌نویس ذخیره شد.", tone: "success" });
    } else {
      actions.createReport(report);
      toast.show({ message: "پیش‌نویس گزارش ایجاد شد.", tone: "success" });
      navigate(`/reports/${report.id}/edit`, { replace: true });
    }
  };

  const handleSubmit = () => {
    if (!canSubmit) return;
    const report = baseReport("submitted");
    if (existing) {
      actions.saveReport(existing.id, {
        ...patch,
        status: "submitted",
        submittedAt: report.submittedAt,
        reviewedBy: undefined,
        reviewNote: undefined,
        updatedAt: report.updatedAt,
      });
    } else {
      actions.createReport(report);
    }
    setStatus("submitted");
    toast.show({
      message: "گزارش برای بررسی سرپرست ارسال شد.",
      tone: "success",
      action: { label: "مشاهده", onClick: () => navigate(`/reports/${report.id}`) },
    });
    navigate(`/reports/${report.id}`);
  };

  return (
    <>
      <TopBar
        title={mode === "create" ? "گزارش جدید" : "ویرایش گزارش"}
        actions={
          <button type="button" className="editor-save" onClick={handleSaveDraft}>
            ذخیره
          </button>
        }
      />

      <Page className="page--with-action-bar">
        <div className="editor-status">
          <div className="editor-status__row">
            <span className="editor-status__label">تکمیل گزارش</span>
            <span className="editor-status__percent num">{new Intl.NumberFormat("fa-IR").format(progress)}٪</span>
          </div>
          <ProgressBar value={progress} label="میزان تکمیل گزارش" />
          <div className="editor-status__meta">
            <span className={cn("editor-save-state", dirty ? "is-dirty" : "is-saved")}>
              {dirty ? <CloudOff size={13} /> : <Cloud size={13} />}
              {dirty
                ? "تغییرات ذخیره نشده است"
                : savedAt
                  ? `ذخیره‌شده در ${formatTime(parse(savedAt))}`
                  : "پیش‌نویس خودکار ذخیره می‌شود"}
            </span>
            <span className="editor-status__hint">برای ارسال، دست‌کم ۶۰٪ محتوا لازم است</span>
          </div>
        </div>

        {status === "needs_revision" && existing?.reviewNote ? (
          <Alert
            tone="warning"
            icon={<FileWarning size={15} />}
            title="گزارش بازگردانده‌شده را اصلاح می‌کنید"
          >
            <p className="editor-review-note">«{existing.reviewNote}»</p>
            <p className="editor-review-by">— {existing.reviewedBy}</p>
          </Alert>
        ) : null}

        <section className="panel editor-panel">
          <Field label="پرونده" htmlFor="report-case" required>
            <Select
              id="report-case"
              value={caseId}
              onChange={(event) => handleCase(event.target.value)}
              disabled={Boolean(visit) || mode === "edit"}
            >
              <option value="">انتخاب پرونده…</option>
              {data.cases.map((familyCase) => (
                <option key={familyCase.id} value={familyCase.id}>
                  {familyCase.title} — {familyCase.caseNumber}
                </option>
              ))}
            </Select>
          </Field>

          <div className="field">
            <span className="field__label">نوع گزارش</span>
            <div className="editor-types" role="group" aria-label="نوع گزارش">
              {typeOptions.map((option) => (
                <button
                  key={option}
                  type="button"
                  className={cn("editor-type", type === option && "is-active")}
                  aria-pressed={type === option}
                  onClick={() => handleType(option)}
                >
                  {reportTypeLabels[option]}
                </button>
              ))}
            </div>
          </div>

          <Field label="عنوان گزارش" htmlFor="report-title" required>
            <Input
              id="report-title"
              value={title}
              onChange={(event) => {
                setTitle(event.target.value);
                markDirty();
              }}
              placeholder="مثلاً گزارش پایش دوره ۴ — آوا موسوی"
            />
          </Field>

          {visit ? (
            <p className="editor-linked-visit">
              گزارش مرتبط با بازدید {reportTypeLabels[typeFromVisit(visit.kind)]} ·{" "}
              {visit.caseTitle}
            </p>
          ) : null}
        </section>

        <section className="panel editor-panel">
          <div className="editor-panel__head">
            <h2 className="panel__title">محتوای گزارش</h2>
            <span className="panel__caption">هر بخش را جداگانه تکمیل کنید</span>
          </div>

          {sections.map((section, index) => {
            const completed = section.value.trim().length > 24;
            return (
              <Field
                key={section.id}
                label={section.label}
                htmlFor={`section-${section.id}`}
                counter={`${new Intl.NumberFormat("fa-IR").format(section.value.length)} کاراکتر`}
              >
                <div className="editor-textarea">
                  <Textarea
                    id={`section-${section.id}`}
                    value={section.value}
                    rows={4}
                    placeholder="مشاهدات، یافته‌ها و ارزیابی خود را بنویسید…"
                    onChange={(event) => {
                      const value = event.target.value;
                      setSections((current) =>
                        current.map((item, itemIndex) =>
                          itemIndex === index ? { ...item, value } : item
                        )
                      );
                      markDirty();
                    }}
                  />
                  {completed ? (
                    <span className="editor-textarea__done" aria-label="این بخش تکمیل شده است">
                      <Check size={12} strokeWidth={3} />
                    </span>
                  ) : null}
                </div>
              </Field>
            );
          })}
        </section>

        {showRating ? (
          <section className="panel editor-panel">
            <div className="editor-rating">
              <div>
                <h2 className="panel__title">ارزیابی کلی شرایط</h2>
                <p className="panel__caption">این امتیاز در جمع‌بندی گزارش ثبت می‌شود.</p>
              </div>
              <div className="editor-rating__stars" role="group" aria-label="ارزیابی کلی شرایط">
                {[1, 2, 3, 4, 5].map((value) => (
                  <button
                    key={value}
                    type="button"
                    className={cn("editor-star", value <= rating && "is-filled")}
                    aria-label={`${value} از ۵ — ${ratingLabels[value]}`}
                    aria-pressed={value === rating}
                    onClick={() => {
                      setRating(value);
                      markDirty();
                    }}
                  >
                    <Star size={24} aria-hidden="true" />
                  </button>
                ))}
              </div>
              <p className="editor-rating__label">
                {rating ? ratingLabels[rating] : "هنوز ارزیابی ثبت نشده است"}
              </p>
            </div>
          </section>
        ) : null}

        <p className="privacy-note">
          <ShieldCheck size={14} aria-hidden="true" />
          پیش‌نویس به‌صورت خودکار ذخیره می‌شود و تا زمان ارسال، برای سرپرست تیم قابل مشاهده نیست.
        </p>
      </Page>

      <div className="action-bar">
        <Button variant="secondary" className="editor-save-btn" onClick={handleSaveDraft}>
          ذخیره پیش‌نویس
        </Button>
        <Button
          className="action-bar__primary"
          leadingIcon={<Send size={16} />}
          disabled={!canSubmit}
          onClick={handleSubmit}
        >
          ارسال برای بررسی
        </Button>
      </div>
    </>
  );
}
