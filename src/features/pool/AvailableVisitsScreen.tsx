import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Inbox } from 'lucide-react';
import type { Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { EmptyState } from '@/components/ui/EmptyState';
import { ClaimVisitSheet } from '@/features/_shared/ClaimVisitSheet';
import { PoolVisitCard } from './PoolVisitCard';
import { groupBy, visitsOnSameDay } from '@/lib/domain';
import { fa } from '@/lib/digits';
import { daysUntil, faDate, faTime } from '@/lib/jalali';
import { visitCategory } from '@/lib/visit-workflow';

type KindFilter = 'all' | 'initial' | 'follow-up';

const KIND_LABEL: Record<KindFilter, string> = {
  all: 'همه',
  initial: 'بازدید اولیه',
  'follow-up': 'بازدید پیگیری',
};

/**
 * «بازدیدهای آزاد» — the pool of scheduled visits nobody owns yet.
 *
 * Grouped by day, nearest first, because the claiming decision is always
 * "what can I take on, and when". Each day header also states my own load
 * that day, so the pool is read against my schedule instead of in a vacuum.
 * Claiming moves the visit into «بازدیدهای من» and out of this list.
 */
export function AvailableVisitsScreen() {
  const { poolVisits, myVisits, cases, toast } = useApp();
  const navigate = useNavigate();
  const [kind, setKind] = useState<KindFilter>('all');
  const [candidate, setCandidate] = useState<Visit | null>(null);

  const counts = useMemo<Record<KindFilter, number>>(
    () => ({
      all: poolVisits.length,
      initial: poolVisits.filter((v) => visitCategory(v.purpose) === 'initial').length,
      'follow-up': poolVisits.filter((v) => visitCategory(v.purpose) !== 'initial').length,
    }),
    [poolVisits],
  );

  const byDay = useMemo(() => {
    const visible = poolVisits.filter((v) =>
      kind === 'all' ? true : kind === 'initial' ? visitCategory(v.purpose) === 'initial' : visitCategory(v.purpose) !== 'initial',
    );
    return [...groupBy(visible, (v) => v.date).entries()].sort(([a], [b]) => a.localeCompare(b));
  }, [poolVisits, kind]);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="بازدیدهای آزاد"
        subtitle={poolVisits.length > 0 ? `${fa(poolVisits.length)} بازدید زمان‌بندی‌شده بدون مددکار در حوزه شما` : 'بازدیدهای بدون مددکار در حوزه شما'}
      >
        <ChipRail flush className="mt-3">
          {(Object.keys(KIND_LABEL) as KindFilter[]).map((value) => (
            <Chip key={value} selected={kind === value} count={counts[value]} onClick={() => setKind(value)}>
              {KIND_LABEL[value]}
            </Chip>
          ))}
        </ChipRail>
      </ScreenHeader>

      <div className="pb-8 pt-4">
        <p className="px-1 text-xs leading-6 text-ink-3">
          تاریخ و ساعت همه این بازدیدها از پیش مشخص است. با پذیرش، بازدید به برنامه شما اضافه می‌شود، اطلاعات کامل خانواده در دسترس قرار می‌گیرد و از این فهرست خارج می‌شود.
        </p>

        {byDay.length === 0 ? (
          <EmptyState
            icon={<Inbox className="size-6" />}
            title={poolVisits.length === 0 ? 'بازدید آزادی نیست' : 'در این دسته بازدیدی نیست'}
            description={
              poolVisits.length === 0
                ? 'در حال حاضر همه بازدیدهای حوزه شما مددکار دارند. بازدیدهای تازه پس از زمان‌بندی واحد در اینجا دیده می‌شوند.'
                : 'دسته دیگری را انتخاب کنید.'
            }
          />
        ) : (
          <div className="mt-2 space-y-6">
            {byDay.map(([date, dayVisits]) => {
              const myCount = visitsOnSameDay(dayVisits[0], myVisits).length;
              const inDays = daysUntil(date);
              const relative = inDays === 0 ? 'امروز' : inDays === 1 ? 'فردا' : inDays > 1 && inDays <= 6 ? `${fa(inDays)} روز دیگر` : null;
              return (
                <section key={date} aria-label={faDate(date, { weekday: true })}>
                  <div className="mb-2.5 flex items-baseline justify-between gap-3 px-1">
                    <h2 className="text-sm font-bold text-ink">
                      {relative && <>{relative}<span className="mx-1.5 font-medium text-ink-3">·</span></>}
                      <span className={relative ? 'text-xs font-medium text-ink-3' : undefined}>
                        {faDate(date, { weekday: true, year: false })}
                      </span>
                    </h2>
                    <span className="shrink-0 text-2xs text-ink-3">
                      {myCount > 0 ? `برنامه شما: ${fa(myCount)} بازدید` : 'برنامه شما آزاد است'}
                    </span>
                  </div>
                  <div className="space-y-3">
                    {dayVisits.map((visit) => (
                      <PoolVisitCard
                        key={visit.id}
                        visit={visit}
                        family={cases.find((c) => c.id === visit.caseId)}
                        myVisits={myVisits}
                        onClaim={setCandidate}
                      />
                    ))}
                  </div>
                </section>
              );
            })}
          </div>
        )}
      </div>

      <ClaimVisitSheet
        visit={candidate}
        open={candidate !== null}
        onClose={() => setCandidate(null)}
        onClaimed={(visit) => {
          setCandidate(null);
          toast({
            title: 'بازدید به برنامه شما اضافه شد',
            description: `${faDate(visit.date, { weekday: true, year: false })} · ساعت ${visit.time.replace(/[0-9]/g, (d) => '۰۱۲۳۴۵۶۷۸۹'[Number(d)])}`,
            tone: 'success',
            action: { label: 'مشاهده برنامه', onClick: () => navigate(`/visits?date=${visit.date}`) },
          });
        }}
      />
    </Screen>
  );
}
