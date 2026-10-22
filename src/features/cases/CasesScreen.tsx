import { useMemo, useState } from 'react';
import { FolderSearch } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { SearchBar } from '@/components/ui/SearchBar';
import { EmptyState } from '@/components/ui/EmptyState';
import { CaseCard } from '@/features/_shared/CaseCard';
import { fa } from '@/lib/digits';

type Scope = 'active' | 'closed';

/** Families are the reference point for recurring visits, not a second task list. */
export function CasesScreen() {
  const { cases, visits } = useApp();
  const [scope, setScope] = useState<Scope>('active');
  const [query, setQuery] = useState('');

  const visible = useMemo(() => cases.filter((c) => {
    if ((c.stage === 'closed') !== (scope === 'closed')) return false;
    if (!query.trim()) return true;
    const needle = query.trim().toLowerCase();
    return [c.familyName, c.code, c.city, c.district ?? '', c.headOfHousehold].join(' ').toLowerCase().includes(needle);
  }), [cases, scope, query]);
  const active = cases.filter((c) => c.stage !== 'closed').length;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader title="خانواده‌ها" subtitle={`${fa(active)} خانواده فعال در حوزه مسئولیت شما`}>
        <div className="mt-3 space-y-2.5">
          <SearchBar value={query} onChange={setQuery} placeholder="جست‌وجوی نام خانواده یا کد پرونده" />
          <SegmentedControl<Scope>
            value={scope}
            onChange={setScope}
            ariaLabel="وضعیت خانواده‌ها"
            items={[
              { value: 'active', label: 'فعال', count: active },
              { value: 'closed', label: 'بایگانی', count: cases.length - active },
            ]}
          />
        </div>
      </ScreenHeader>

      <div className="pb-8 pt-4">
        {!visible.length ? (
          <EmptyState icon={<FolderSearch className="size-6" />} title="خانواده‌ای یافت نشد" description="عبارت جست‌وجو یا وضعیت را تغییر دهید." />
        ) : (
          <ul className="space-y-2.5">
            {visible.map((c) => (
              <li key={c.id}>
                <CaseCard caseFile={c} visits={visits.filter((v) => v.caseId === c.id)} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </Screen>
  );
}
