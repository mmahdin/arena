import { useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { CalendarPlus, CheckCircle2, FileText, FolderHeart, MoreVertical, Undo2 } from 'lucide-react';
import type { Task } from '@/types';
import { ListRow } from '@/components/ui/ListRow';
import { IconTile, TASK_ICON } from '@/components/ui/DomainIcon';
import { IconButton } from '@/components/ui/IconButton';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { Badge } from '@/components/ui/Badge';
import { TASK_STATUS, TASK_TYPE } from '@/lib/labels';
import { taskBucket } from '@/lib/domain';
import { dayKey, deadlineLabel, faDate } from '@/lib/jalali';
import { useApp } from '@/store/app-store';
import { Meta, MetaTone } from './Meta';

/** Compact queue row; task actions never sit inside a navigation link. */
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
  const [menuOpen, setMenuOpen] = useState(false);
  const done = task.status === 'done';
  const bucket = taskBucket(task);

  const href = task.reportId
    ? `/reports/${task.reportId}`
    : task.visitId
      ? `/visits/${task.visitId}`
      : task.caseId
        ? `/cases/${task.caseId}`
        : `/calendar?date=${task.dueDate}`;

  const deadlineTone = done ? 'neutral' : bucket === 'overdue' ? 'danger' : bucket === 'today' ? 'warning' : 'neutral';

  return (
    <>
      <div className={`relative flex min-w-0 items-center ${divider ? 'border-b border-line-soft' : ''}`}>
        <div className="min-w-0 flex-1">
          <ListRow
            to={href}
            onClick={onOpen}
            muted={done}
            dense={compact}
            leading={<IconTile icon={TASK_ICON[task.type]} tone={done ? 'neutral' : TASK_TYPE[task.type].tone} />}
            title={task.title}
            meta={
              <Meta>
                {task.caseCode && <span className="tabular-nums">{task.caseCode}</span>}
                <MetaTone tone={deadlineTone}>
                  {done && task.completedAt ? faDate(task.completedAt, { year: false }) : deadlineLabel(task.dueDate)}
                </MetaTone>
              </Meta>
            }
            trailing={
              task.status === 'revision' ? <Badge tone="warning">اصلاح</Badge> :
              task.status === 'blocked' ? <Badge tone="danger">متوقف</Badge> :
              done ? <CheckCircle2 className="size-5 text-success" strokeWidth={2.25} aria-label="انجام‌شده" /> : undefined
            }
          />
        </div>
        <div className="pe-2">
          <IconButton label="گزینه‌های کار" size="sm" onClick={() => setMenuOpen(true)}>
            <MoreVertical className="size-4" />
          </IconButton>
        </div>
      </div>

      <TaskMenu task={task} open={menuOpen} onClose={() => setMenuOpen(false)} />
    </>
  );
}

/** One set of task actions, available from the selected calendar day too. */
export function TaskMenu({ task, open, onClose }: { task: Task; open: boolean; onClose: () => void }) {
  const { setTaskStatus, postponeTask, toast } = useApp();
  const navigate = useNavigate();
  const done = task.status === 'done';
  const requiresWorkflow = !!task.reportId || !!task.visitId || task.type === 'report' || task.type === 'visit';
  const canCompleteHere = !requiresWorkflow && task.status !== 'blocked';
  const canPostpone = !done && !requiresWorkflow && task.status !== 'blocked';

  return (
    <Sheet
      open={open}
      onClose={onClose}
      title={task.title}
      description={`${TASK_TYPE[task.type].label} · ${TASK_STATUS[task.status].label}`}
    >
      <div className="space-y-2 pb-4">
        {task.description && <p className="px-2 pb-2 text-sm leading-relaxed text-ink-2">{task.description}</p>}
        {canCompleteHere && (!done ? (
          <SheetOption
            icon={<CheckCircle2 className="size-[18px]" />}
            title="ثبت به‌عنوان انجام‌شده"
            onClick={async () => {
              onClose();
              await setTaskStatus(task.id, 'done');
              toast({ title: 'کار انجام‌شده ثبت شد', tone: 'success' });
            }}
          />
        ) : (
          <SheetOption
            icon={<Undo2 className="size-[18px]" />}
            title="بازگرداندن به کارهای باز"
            onClick={async () => {
              onClose();
              await setTaskStatus(task.id, 'todo');
              toast({ title: 'کار دوباره فعال شد', tone: 'info' });
            }}
          />
        ))}
        {canPostpone && (
          <SheetOption
            icon={<CalendarPlus className="size-[18px]" />}
            title={task.dueDate < dayKey(new Date()) ? 'انتقال مهلت به فردا' : 'یک روز تمدید مهلت'}
            onClick={async () => {
              onClose();
              await postponeTask(task.id, 1);
              toast({ title: 'مهلت کار به فردا منتقل شد', tone: 'info' });
            }}
          />
        )}
        {task.visitId && (
          <SheetOption
            icon={<CalendarPlus className="size-[18px]" />}
            title="ثبت نتیجه در صفحه بازدید"
            onClick={() => { onClose(); navigate(`/visits/${task.visitId}`); }}
          />
        )}
        {task.reportId && (
          <SheetOption
            icon={<FileText className="size-[18px]" />}
            title="باز کردن گزارش مرتبط"
            onClick={() => { onClose(); navigate(`/reports/${task.reportId}`); }}
          />
        )}
        {task.caseId && (
          <SheetOption
            icon={<FolderHeart className="size-[18px]" />}
            title="اطلاعات پرونده"
            onClick={() => { onClose(); navigate(`/cases/${task.caseId}`); }}
          />
        )}
        {requiresWorkflow && !task.reportId && !task.visitId && (
          <p className="px-2 py-2 text-sm leading-relaxed text-ink-3">وضعیت این کار از طریق فرایند مربوط ثبت می‌شود.</p>
        )}
      </div>
    </Sheet>
  );
}
