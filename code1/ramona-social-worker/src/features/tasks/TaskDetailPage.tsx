import { Link, useNavigate, useParams } from "react-router-dom";
import {
  CalendarCheck2,
  Check,
  ChevronLeft,
  ClipboardList,
  Clock3,
  FileText,
  FolderClosed,
  RotateCcw,
  ShieldCheck,
} from "lucide-react";
import { useAppActions, useNow, useWorkspace } from "@/app/store";
import { deadlineLabel, formatDate, formatTime, parse } from "@/lib/date";
import { priorityLabels, taskTypeLabels } from "@/lib/labels";
import { cn } from "@/lib/cn";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Alert } from "@/components/ui/Alert";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { EmptyState } from "@/components/ui/EmptyState";
import { useToast } from "@/components/ui/Toast";
import { PriorityBadge, TaskTypeIcon } from "@/components/domain/Badges";
import "./TaskDetailPage.css";

export function TaskDetailPage() {
  const { taskId } = useParams();
  const data = useWorkspace();
  const now = useNow();
  const actions = useAppActions();
  const toast = useToast();
  const navigate = useNavigate();

  const task = data.tasks.find((t) => t.id === taskId);

  if (!task) {
    return (
      <>
        <TopBar title="جزئیات کار" onBack={() => navigate("/tasks")} />
        <Page>
          <EmptyState
            icon={<ClipboardList size={22} />}
            title="این کار یافت نشد"
            description="ممکن است کار حذف شده باشد یا به مددکار دیگری ارجاع داده شده باشد."
            action={
              <Button variant="secondary" onClick={() => navigate("/tasks")}>
                بازگشت به کارتابل
              </Button>
            }
          />
        </Page>
      </>
    );
  }

  const done = task.status === "done";
  const deadline = deadlineLabel(task.dueAt, now);
  const dueDate = parse(task.dueAt);
  const familyCase = task.caseId ? data.cases.find((c) => c.id === task.caseId) : undefined;
  const report = task.relatedReportId
    ? data.reports.find((r) => r.id === task.relatedReportId)
    : undefined;
  const visit = task.relatedVisitId
    ? data.visits.find((v) => v.id === task.relatedVisitId)
    : undefined;
  const checklistDone = task.checklist?.filter((item) => item.done).length ?? 0;
  const checklistTotal = task.checklist?.length ?? 0;
  const progress = checklistTotal ? (checklistDone / checklistTotal) * 100 : 0;

  const toggleDone = () => {
    const next = done ? "todo" : "done";
    actions.setTaskStatus(task.id, next);
    toast.show({
      message: done ? "کار به حالت باز برگشت." : "کار انجام شد.",
      tone: done ? "default" : "success",
      action: {
        label: "بازگرداندن",
        onClick: () => actions.setTaskStatus(task.id, done ? "done" : "todo"),
      },
    });
  };

  return (
    <>
      <TopBar title="جزئیات کار" />
      <Page className="page--with-action-bar">
        <section className="task-hero">
          <div className="task-hero__badges">
            <Badge tone="neutral" icon={<TaskTypeIcon type={task.type} size={13} />}>
              {taskTypeLabels[task.type]}
            </Badge>
            <PriorityBadge priority={task.priority} />
            {done ? <Badge tone="success" dot>انجام‌شده</Badge> : null}
          </div>
          <h1 className={cn("task-hero__title", done && "is-done")}>{task.title}</h1>
          {task.caseTitle ? (
            <Link to={`/cases/${task.caseId}`} className="task-hero__case">
              <FolderClosed size={14} aria-hidden="true" />
              پرونده {task.caseTitle}
              <ChevronLeft size={14} aria-hidden="true" />
            </Link>
          ) : null}
        </section>

        {deadline.tone === "overdue" && !done ? (
          <Alert tone="danger" icon={<Clock3 size={15} />} title="مهلت این کار گذشته است">
            این کار {deadline.text} است. لطفاً در اولین فرصت آن را تکمیل کنید یا با سرپرست تیم
            هماهنگ کنید.
          </Alert>
        ) : null}

        <section className="panel">
          <div className="kv-list">
            <div className="kv-row">
              <span className="kv-row__label">
                <Clock3 size={13} aria-hidden="true" />
                مهلت انجام
              </span>
              <span className="kv-row__value">
                {formatDate(dueDate)}، ساعت <span className="num">{formatTime(dueDate)}</span>
                {!done ? (
                  <span className={cn("task-deadline-tag", `is-${deadline.tone}`)}>{deadline.text}</span>
                ) : null}
              </span>
            </div>
            <div className="kv-row">
              <span className="kv-row__label">اولویت</span>
              <span className="kv-row__value">{priorityLabels[task.priority].label}</span>
            </div>
            <div className="kv-row">
              <span className="kv-row__label">وضعیت</span>
              <span className="kv-row__value">{done ? "انجام‌شده" : "باز"}</span>
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

        {task.description ? (
          <section className="panel">
            <h2 className="panel__title">شرح کار</h2>
            <p className="panel__body">{task.description}</p>
          </section>
        ) : null}

        {task.checklist && task.checklist.length > 0 ? (
          <section className="panel">
            <div className="panel__head">
              <h2 className="panel__title">مراحل انجام</h2>
              <span className="panel__caption num">
                {new Intl.NumberFormat("fa-IR").format(checklistDone)} از{" "}
                {new Intl.NumberFormat("fa-IR").format(checklistTotal)}
              </span>
            </div>
            <ProgressBar value={progress} label="پیشرفت مراحل کار" />
            <ul className="checklist">
              {task.checklist.map((item, index) => (
                <li key={`${item.label}-${index}`}>
                  <button
                    type="button"
                    role="checkbox"
                    aria-checked={item.done}
                    className={cn("checklist__item", item.done && "is-done")}
                    onClick={() => actions.toggleTaskChecklist(task.id, index)}
                  >
                    <span className="checklist__box" aria-hidden="true">
                      <Check size={12} strokeWidth={3} />
                    </span>
                    <span>{item.label}</span>
                  </button>
                </li>
              ))}
            </ul>
          </section>
        ) : null}

        {visit || report ? (
          <section className="panel">
            <h2 className="panel__title">پیوندهای مرتبط</h2>
            <div className="task-links">
              {visit ? (
                <Link to={`/visits/${visit.id}`} className="task-link">
                  <span className="icon-tile icon-tile--info icon-tile--sm" aria-hidden="true">
                    <CalendarCheck2 size={15} />
                  </span>
                  <span className="task-link__text">
                    <span className="task-link__title">بازدید مرتبط</span>
                    <span className="task-link__sub">{visit.caseTitle}</span>
                  </span>
                  <ChevronLeft size={15} className="task-link__chev" aria-hidden="true" />
                </Link>
              ) : null}
              {report ? (
                <Link to={`/reports/${report.id}`} className="task-link">
                  <span className="icon-tile icon-tile--brand icon-tile--sm" aria-hidden="true">
                    <FileText size={15} />
                  </span>
                  <span className="task-link__text">
                    <span className="task-link__title">گزارش مرتبط</span>
                    <span className="task-link__sub">{report.title}</span>
                  </span>
                  <ChevronLeft size={15} className="task-link__chev" aria-hidden="true" />
                </Link>
              ) : null}
            </div>
          </section>
        ) : null}

        <p className="privacy-note">
          <ShieldCheck size={14} aria-hidden="true" />
          نتیجه این کار در سوابق پرونده ثبت می‌شود و برای سرپرست تیم قابل مشاهده است.
        </p>
      </Page>

      <div className="action-bar">
        <Button
          variant={done ? "secondary" : "primary"}
          className="action-bar__primary"
          leadingIcon={done ? <RotateCcw size={17} /> : <Check size={18} />}
          onClick={toggleDone}
        >
          {done ? "بازگرداندن به کارهای باز" : "ثبت انجام کار"}
        </Button>
      </div>
    </>
  );
}
