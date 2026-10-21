import { useMemo, useState } from 'react';
import { CheckCircle2, Inbox } from 'lucide-react';
import type { TaskType } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { SearchBar } from '@/components/ui/SearchBar';
import { ListGroup } from '@/components/ui/ListRow';
import { EmptyState } from '@/components/ui/EmptyState';
import { TaskRow } from '@/features/_shared/TaskRow';
import { isOpenTask, sortTasks, taskBucket, TASK_BUCKET_LABEL, TASK_BUCKET_ORDER } from '@/lib/domain';
import { fa } from '@/lib/digits';

type Scope = 'open' | 'revision' | 'done';

const TYPE_FILTERS: { value: TaskType | 'all'; label: string }[] = [
  { value: 'all', label: 'همه' },
  { value: 'visit', label: 'بازدید' },
  { value: 'report', label: 'گزارش' },
  { value: 'assessment', label: 'ارزیابی' },
  { value: 'monitoring', label: 'پایش' },
  { value: 'document', label: 'مستندات' },
  { value: 'meeting', label: 'جلسه' },
];

/**
 * Work — one queue for everything assigned to this social worker.
 *
 * Visits, reports, assessments and follow-ups are all "work items" here. They
 * are not split into separate destinations, because in the field the question
 * is never "which module is this in?" — it is "what is due, and what is late?".
 */
export function WorkScreen() {
  const { tasks } = useApp();
  const [scope, setScope] = useState<Scope>('open');
  const [type, setType] = useState<TaskType | 'all'>('all');
  const [query, setQuery] = useState('');

  const counts = useMemo(
    () => ({
      open: tasks.filter(isOpenTask).length,
      revision: tasks.filter((t) => t.status === 'revision').length,
      done: tasks.filter((t) => t.status === 'done').length,
    }),
    [tasks],
  );

  const visible = useMemo(() => {
    let rows = tasks.slice();
    if (scope === 'open') rows = rows.filter((t) => isOpenTask(t) && t.status !== 'revision');
    if (scope === 'revision') rows = rows.filter((t) => t.status === 'revision');
    if (scope === 'done') rows = rows.filter((t) => t.status === 'done');
    if (type !== 'all') rows = rows.filter((t) => t.type === type);
    if (query.trim()) {
      const needle = query.trim().toLowerCase();
      rows = rows.filter((t) =>
        [t.title, t.caseCode ?? '', t.assignedBy, t.description ?? ''].join(' ').toLowerCase().includes(needle),
      );
    }
    return sortTasks(rows);
  }, [tasks, scope, type, query]);

  const groups = useMemo(() => {
    if (scope === 'done') return [{ key: 'done', label: 'انجام‌شده', items: visible }];
    return TASK_BUCKET_ORDER.map((bucket) => ({
      key: bucket,
      label: TASK_BUCKET_LABEL[bucket],
      items: visible.filter((t) => taskBucket(t) === bucket),
    })).filter((g) => g.items.length > 0);
  }, [visible, scope]);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="کارها"
        subtitle={`${fa(counts.open)} مورد باز · مرتب‌شده بر اساس مهلت`}
      >
        <div className="mt-3 space-y-3">
          <SegmentedControl<Scope>
            ariaLabel="محدوده کارها"
            value={scope}
            onChange={setScope}
            items={[
              { value: 'open', label: 'فعال', count: counts.open },
              { value: 'revision', label: 'اصلاح', count: counts.revision },
              { value: 'done', label: 'انجام‌شده', count: counts.done },
            ]}
          />
        </div>
      </ScreenHeader>

      <div className="pt-3">
        <SearchBar value={query} onChange={setQuery} placeholder="جست‌وجو در کارها، پرونده یا ارجاع‌دهنده" />
        <div className="mt-3">
          <ChipRail>
            {TYPE_FILTERS.map((f) => (
              <Chip key={f.value} selected={type === f.value} onClick={() => setType(f.value)}>
                {f.label}
              </Chip>
            ))}
          </ChipRail>
        </div>
      </div>

      {visible.length === 0 ? (
        <EmptyState
          className="mt-4"
          icon={scope === 'done' ? <CheckCircle2 className="size-6" /> : <Inbox className="size-6" />}
          title={scope === 'done' ? 'هنوز کاری را کامل نکرده‌اید' : 'موردی یافت نشد'}
          description={
            scope === 'done'
              ? 'کارهای کامل‌شده اینجا بایگانی می‌شوند.'
              : 'فیلترها را تغییر دهید یا عبارت دیگری را جست‌وجو کنید.'
          }
        />
      ) : (
        <div className="pb-8">
          {groups.map((group) => (
            <section key={group.key} className="mt-6 first:mt-4">
              <div className="mb-2 flex items-baseline justify-between">
                <h2
                  className={`text-sm font-bold ${group.key === 'overdue' ? 'text-danger' : 'text-ink-2'}`}
                >
                  {group.label}
                </h2>
                <span className="text-2xs text-ink-3 tabular-nums">{fa(group.items.length)} مورد</span>
              </div>
              <ListGroup>
                {group.items.map((t, i) => (
                  <TaskRow key={t.id} task={t} divider={i < group.items.length - 1} />
                ))}
              </ListGroup>
              {group.key === 'revision' && (
                <p className="mt-2 px-1 text-xs leading-relaxed text-ink-3">
                  این گزارش‌ها توسط سرپرست بازگردانده شده‌اند. پس از اصلاح، دوباره برای بازبینی ارسال
                  کنید.
                </p>
              )}
            </section>
          ))}

          <p className="mt-6 px-2 text-center text-2xs leading-relaxed text-ink-3">
            کارها توسط سامانه یا سرپرست واحد به شما محول می‌شوند.
          </p>
        </div>
      )}
    </Screen>
  );
}
