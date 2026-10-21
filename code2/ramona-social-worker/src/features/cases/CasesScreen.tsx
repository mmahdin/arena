import { useMemo, useState } from 'react';
import { EyeOff, FolderSearch, ShieldCheck, SlidersHorizontal } from 'lucide-react';
import type { CaseStage, CaseType } from '@/types';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { IconButton } from '@/components/ui/IconButton';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { SearchBar } from '@/components/ui/SearchBar';
import { EmptyState } from '@/components/ui/EmptyState';
import { Sheet, SheetOption } from '@/components/ui/Sheet';
import { CaseCard } from '@/features/_shared/CaseCard';
import { fa } from '@/lib/digits';

type QuickFilter = 'all' | CaseType | 'closed';

const FILTERS: { value: QuickFilter; label: string }[] = [
  { value: 'all', label: 'همه' },
  { value: 'hosting', label: 'میزبانی' },
  { value: 'adoption', label: 'فرزندخواندگی' },
  { value: 'closed', label: 'بایگانی' },
];

/**
 * Cases — the families this social worker is responsible for.
 *
 * Access is scoped server-side to assigned cases; the screen says so plainly,
 * because a user who wonders whether they are seeing everything stops trusting
 * what they do see.
 */
export function CasesScreen() {
  const { cases } = useApp();
  const { fieldPrivacy, toggleFieldPrivacy } = useSettings();
  const [filter, setFilter] = useState<QuickFilter>('all');
  const [query, setQuery] = useState('');
  const [stage, setStage] = useState<CaseStage | 'all'>('all');
  const [stageSheet, setStageSheet] = useState(false);

  const visible = useMemo(() => {
    let rows = cases.slice();
    if (filter === 'hosting' || filter === 'adoption') rows = rows.filter((c) => c.type === filter);
    if (filter === 'closed') rows = rows.filter((c) => c.stage === 'closed');
    if (filter !== 'closed') rows = rows.filter((c) => c.stage !== 'closed');
    if (stage !== 'all') rows = rows.filter((c) => c.stage === stage);
    if (query.trim()) {
      const needle = query.trim().toLowerCase();
      rows = rows.filter((c) =>
        [c.familyName, c.code, c.city, c.district ?? '', c.headOfHousehold, ...c.children.map((ch) => ch.firstName)]
          .join(' ')
          .toLowerCase()
          .includes(needle),
      );
    }
    return rows;
  }, [cases, filter, stage, query]);

  const activeCount = cases.filter((c) => c.stage !== 'closed').length;
  const stages: { value: CaseStage | 'all'; label: string }[] = [
    { value: 'all', label: 'همه مراحل' },
    { value: 'initial-assessment', label: 'ارزیابی اولیه' },
    { value: 'introduction', label: 'معرفی' },
    { value: 'placement', label: 'جایگذاری' },
    { value: 'post-placement', label: 'پایش' },
  ];

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="پرونده‌ها"
        subtitle={`${fa(activeCount)} پرونده فعال در حوزه مسئولیت شما`}
        actions={
          <>
            <IconButton label="فیلتر مرحله پرونده" onClick={() => setStageSheet(true)} dot={stage !== 'all'}>
              <SlidersHorizontal className="size-[20px]" strokeWidth={2} />
            </IconButton>
            <IconButton
              label={fieldPrivacy ? 'خاموش کردن حالت محرمانگی' : 'روشن کردن حالت محرمانگی'}
              variant={fieldPrivacy ? 'soft' : 'plain'}
              onClick={toggleFieldPrivacy}
            >
              <EyeOff className={fieldPrivacy ? 'size-[20px] text-accent-ink' : 'size-[20px]'} strokeWidth={2} />
            </IconButton>
          </>
        }
      >
        <div className="mt-3 space-y-2.5">
          <SearchBar value={query} onChange={setQuery} placeholder="جست‌وجوی نام خانواده، کد پرونده یا کودک" />
          <ChipRail flush>
            {FILTERS.map((f) => (
              <Chip key={f.value} selected={filter === f.value} onClick={() => { setFilter(f.value); setStage('all'); }}>
                {f.label}
              </Chip>
            ))}
          </ChipRail>
        </div>
      </ScreenHeader>

      <div className="pt-4">
        {visible.length === 0 ? (
          <EmptyState
            icon={<FolderSearch className="size-6" />}
            title="پرونده‌ای یافت نشد"
            description="فیلترها را تغییر دهید یا عبارت دیگری را جست‌وجو کنید."
          />
        ) : (
          <ul className="space-y-3 pb-6">
            {visible.map((c) => (
              <li key={c.id}>
                <CaseCard caseFile={c} />
              </li>
            ))}
          </ul>
        )}

        <Sheet
          open={stageSheet}
          onClose={() => setStageSheet(false)}
          title="فیلتر بر اساس مرحله"
          description="مرحله پرونده در فرایند میزبانی و فرزندخواندگی"
        >
          <div className="space-y-2 pb-4">
            {stages.map((s) => (
              <SheetOption
                key={s.value}
                selected={stage === s.value}
                title={s.label}
                onClick={() => {
                  setStage(s.value);
                  setStageSheet(false);
                }}
              />
            ))}
          </div>
        </Sheet>

        <p className="flex items-start gap-2 pb-8 text-2xs leading-relaxed text-ink-3">
          <ShieldCheck className="mt-0.5 size-4 shrink-0" strokeWidth={2.25} aria-hidden />
          فقط پرونده‌هایی که به شما محول شده‌اند نمایش داده می‌شوند. برخی جزئیات حساس ممکن است بر اساس
          سطح دسترسی شما پنهان باشد.
        </p>
      </div>
    </Screen>
  );
}
