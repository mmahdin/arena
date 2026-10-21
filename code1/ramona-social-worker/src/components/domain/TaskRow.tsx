import { Link } from "react-router-dom";
import { Clock } from "lucide-react";
import type { Task } from "@/types";
import { deadlineLabel, formatRelative, parse } from "@/lib/date";
import { taskTypeLabels } from "@/lib/labels";
import { cn } from "@/lib/cn";
import { Checkbox } from "@/components/ui/Checkbox";
import { TaskTypeIcon } from "./Badges";
import "./TaskRow.css";

export function TaskRow({
  task,
  now,
  onToggle,
  showCase = true,
}: {
  task: Task;
  now: Date;
  onToggle: (task: Task) => void;
  showCase?: boolean;
}) {
  const done = task.status === "done";
  const deadline = deadlineLabel(task.dueAt, now);
  const urgent = (task.priority === "urgent" || task.priority === "high") && !done;

  return (
    <li className={cn("task-item", done && "is-done")}>
      <Checkbox
        checked={done}
        onChange={() => onToggle(task)}
        label={done ? `بازگرداندن «${task.title}» به کارهای باز` : `تکمیل «${task.title}»`}
      />
      <Link to={`/tasks/${task.id}`} className="task-item__main">
        <span className="task-item__row">
          <span className="task-item__title">{task.title}</span>
          {urgent ? (
            <span className={cn("task-item__flag", task.priority === "urgent" && "is-urgent")}>
              {task.priority === "urgent" ? "فوری" : "مهم"}
            </span>
          ) : null}
        </span>
        <span className="task-item__meta">
          <span className="task-item__type">
            <TaskTypeIcon type={task.type} size={13} />
            {taskTypeLabels[task.type]}
          </span>
          {showCase && task.caseTitle ? (
            <>
              <span className="task-item__sep" aria-hidden="true">·</span>
              <span className="truncate">{task.caseTitle}</span>
            </>
          ) : null}
          <span className="task-item__sep" aria-hidden="true">·</span>
          <span
            className={cn(
              "task-item__due",
              !done && `is-${deadline.tone}`
            )}
          >
            {done ? (
              <>انجام‌شده {formatRelative(parse(task.dueAt), now)}</>
            ) : (
              <>
                <Clock size={11} aria-hidden="true" />
                {deadline.text}
              </>
            )}
          </span>
        </span>
      </Link>
    </li>
  );
}
