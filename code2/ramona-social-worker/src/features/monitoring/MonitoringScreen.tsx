import { useMemo, useState } from 'react';
import { HeartPulse, ShieldCheck } from 'lucide-react';
import type { Wellbeing } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Card } from '@/components/ui/Card';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { EmptyState } from '@/components/ui/EmptyState';
import { ChildRow } from '@/features/_shared/ChildRow';
import { fa } from '@/lib/digits';
import { daysUntil } from '@/lib/jalali';
import { cn } from '@/lib/cn';

type Filter = 'all' | Wellbeing | 'week';

/**
 * Monitoring.
 *
 * Post-placement follow-up is the part of the job where a slow drift matters
 * more than a crisis. The screen therefore leads with *who needs attention*
 * and *when the next visit falls due*, and treats the stable cases as the
 * quiet majority rather than hiding them.
 */
export function MonitoringScreen() {
  const { cases } = useApp();
  const [filter, setFilter] = useState<Filter>('all');

  const children = useMemo(
    () =>
      cases
        .filter((c) => c.stage !== 'closed')
        .flatMap((c) => c.children.map((ch) => ({ ...ch, caseCode: c.code, familyName: c.familyName, caseId: c.id }))),
    [cases],
  );

  const counts = useMemo(
    () => ({
      all: children.length,
      stable: children.filter((c) => c.wellbeing === 'stable').length,
      attention: children.filter((c) => c.wellbeing === 'attention').length,
      critical: children.filter((c) => c.wellbeing === 'critical').length,
      week: children.filter((c) => daysUntil(c.nextFollowUp) <= 7).length,
    }),
    [children],
  );

  const visible = useMemo(() => {
    const order = { critical: 0, attention: 1, stable: 2 } as const;
    const rows =
      filter === 'all'
        ? children
        : filter === 'week'
          ? children.filter((c) => daysUntil(c.nextFollowUp) <= 7)
          : children.filter((c) => c.wellbeing === filter);
    return rows.slice().sort((a, b) => order[a.wellbeing] - order[b.wellbeing] || a.nextFollowUp.localeCompare(b.nextFollowUp));
  }, [children, filter]);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader title="پایش کودکان" subtitle={`${fa(counts.all)} کودک تحت پایش فعال`} />

      <div className="pt-4">
        <div className="grid grid-cols-3 gap-2">
          <StatTile tone="danger" value={counts.critical} label="نیازمند مداخله" />
          <StatTile tone="warning" value={counts.attention} label="نیازمند توجه" />
          <StatTile tone="success" value={counts.stable} label="پایدار" />
        </div>

        <div className="mt-4">
          <ChipRail flush>
            <Chip selected={filter === 'all'} count={counts.all} onClick={() => setFilter('all')}>
              همه
            </Chip>
            <Chip selected={filter === 'week'} count={counts.week} onClick={() => setFilter('week')}>
              پیگیری این هفته
            </Chip>
            <Chip selected={filter === 'attention'} count={counts.attention} onClick={() => setFilter('attention')}>
              نیازمند توجه
            </Chip>
            <Chip selected={filter === 'critical'} count={counts.critical} onClick={() => setFilter('critical')}>
              بحرانی
            </Chip>
            <Chip selected={filter === 'stable'} count={counts.stable} onClick={() => setFilter('stable')}>
              پایدار
            </Chip>
          </ChipRail>
        </div>
      </div>

      <div className="pt-4 pb-8">
        {visible.length === 0 ? (
          <EmptyState
            icon={<HeartPulse className="size-6" />}
            title="موردی در این دسته نیست"
            description="همه کودکان تحت پایش در وضعیت پایدار قرار دارند."
          />
        ) : (
          <Card padded={false} className="divide-y divide-line-soft">
            {visible.map((c, i) => (
              <ChildRow key={c.id} child={c} divider={i < visible.length - 1} />
            ))}
          </Card>
        )}

        <p className="mt-5 flex items-start gap-2 px-1 text-2xs leading-relaxed text-ink-3">
          <ShieldCheck className="mt-0.5 size-4 shrink-0" strokeWidth={2.25} aria-hidden />
          ارزیابی وضعیت کودک بر اساس مشاهدات میدانی شما ثبت می‌شود و تصمیم‌گیری درباره تغییر در
          وضعیت جایگذاری بر عهده کمیته تخصصی است.
        </p>
      </div>
    </Screen>
  );
}

function StatTile({ tone, value, label }: { tone: 'danger' | 'warning' | 'success'; value: number; label: string }) {
  const skin = {
    danger: 'border-danger/25 bg-danger-soft text-danger',
    warning: 'border-warning/25 bg-warning-soft text-warning',
    success: 'border-success/25 bg-success-soft text-success',
  }[tone];
  return (
    <div className={cn('rounded-lg border px-3 py-3 text-center', skin)}>
      <p className="text-xl font-extrabold tabular-nums">{fa(value)}</p>
      <p className="mt-0.5 text-2xs font-medium opacity-85">{label}</p>
    </div>
  );
}
