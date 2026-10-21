import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CalendarPlus, CheckCircle2, FileText, FolderHeart, MoreVertical, Undo2 } from 'lucide-react';
import type { Task } from '@/types';
import { ListRow } from '@/components/ui/ListRow';
import { IconTile, TASK_ICON } from '@/components/ui/DomainIcon';
import { IconButton } from '@/components/ui/IconButton';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { Badge } from '@/components/ui/Badge';
import { PRIORITY, TASK_STATUS, TASK_TYPE } from '@/lib/labels';
import { taskBucket } from '@/lib/domain';
import { deadlineLabel } from '@/lib/jalali';
import { useApp } from '@/store/app-store';
import { Meta, MetaTone, PriorityRail } from './Meta';

/**
 * Work item row — the atom of the whole product.
 *
 * Priority is a colour rail, type is an icon, urgency is the deadline text.
 * Three independent channels, so a queue can be scanned by any of them
 * without the row getting busier.
 */
export function TaskRow({
  task,
  divider,
  compact,
  onOpen,
}: {
  task: Task;
  divider?: boolean;
  compact?: boolean;
  onOpen?: () => void;
}) {
  const { setTaskStatus, postponeTask, toast } = useApp();
  const navigate = useNavigate();
  const [menuOpen, setMenuOpen] = useState(false);
  const done = task.status === 'done';
  const bucket = taskBucket(task);

  const href = task.reportId
    ? `/reports/${task.reportId}`
    : task.visitId
      ? `/visits/${task.visitId}`
      : task.caseId
        ? `/cases/${task.caseId}`
        : '/work';

  const deadlineTone = done
    ? 'neutral'
    : bucket === 'overdue'
      ? 'danger'
      : bucket === 'today'
        ? 'warning'
        : 'neutral';

  return (
    <>
      <ListRow
        to={href}
        onClick={onOpen}
        divider={divider}
        muted={done}
        dense={compact}
        accent={
          !done && (task.priority === 'critical' || task.priority === 'high') ? (
            <PriorityRail tone={task.priority === 'critical' ? 'danger' : 'warning'} />
          ) : undefined
        }
        leading={<IconTile icon={TASK_ICON[task.type]} tone={done ? 'neutral' : TASK_TYPE[task.type].tone} />}
        title={task.title}
        meta={
          <Meta>
            {task.caseCode && <span className="tabular-nums">{task.caseCode}</span>}
            <MetaTone tone={deadlineTone}>{deadlineLabel(task.dueDate)}</MetaTone>
          </Meta>
        }
        trailing={
          <>
            {task.status === 'revision' && <Badge tone="warning">اصلاح</Badge>}
            {task.status === 'blocked' && <Badge tone="danger">مسدود</Badge>}
            {done && <CheckCircle2 className="size-5 text-success" strokeWidth={2.25} aria-label="انجام‌شده" />}
            <IconButton
              label="گزینه‌های کار"
              size="sm"
              onClick={(e) => {
                e.preventDefault();
                e.stopPropagation();
                setMenuOpen(true);
              }}
            >
              <MoreVertical className="size-4" />
            </IconButton>
          </>
        }
      />

      <Sheet
        open={menuOpen}
        onClose={() => setMenuOpen(false)}
        title={task.title}
        description={`${TASK_TYPE[task.type].label} · ${TASK_STATUS[task.status].label} · اولویت ${PRIORITY[task.priority].label}`}
      >
        <div className="space-y-2 pb-4">
          {!done ? (
            <SheetOption
              icon={<CheckCircle2 className="size-[18px]" />}
              title="علامت‌گذاری به‌عنوان انجام‌شده"
              description="کار از فهرست فعال خارج می‌شود."
              onClick={async () => {
                setMenuOpen(false);
                await setTaskStatus(task.id, 'done');
                toast({ title: 'کار انجام‌شده ثبت شد', description: task.title, tone: 'success' });
              }}
            />
          ) : (
            <SheetOption
              icon={<Undo2 className="size-[18px]" />}
              title="بازگرداندن به فهرست فعال"
              onClick={async () => {
                setMenuOpen(false);
                await setTaskStatus(task.id, 'todo');
                toast({ title: 'کار دوباره فعال شد', tone: 'info' });
              }}
            />
          )}
          <SheetOption
            icon={<CalendarPlus className="size-[18px]" />}
            title="موکول به فردا"
            description="مهلت کار یک روز جابه‌جا می‌شود."
            onClick={async () => {
              setMenuOpen(false);
              await postponeTask(task.id, 1);
              toast({ title: 'به فردا موکول شد', description: task.title, tone: 'info' });
            }}
          />
          {task.reportId && (
            <SheetOption
              icon={<FileText className="size-[18px]" />}
              title="باز کردن گزارش مرتبط"
              onClick={() => {
                setMenuOpen(false);
                navigate(`/reports/${task.reportId}`);
              }}
            />
          )}
          {task.caseId && (
            <SheetOption
              icon={<FolderHeart className="size-[18px]" />}
              title="مشاهده پرونده"
              onClick={() => {
                setMenuOpen(false);
                navigate(`/cases/${task.caseId}`);
              }}
            />
          )}
        </div>
      </Sheet>
    </>
  );
}
