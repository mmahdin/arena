import { useMemo, useState } from 'react';
import { FolderSearch } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SearchBar } from '@/components/ui/SearchBar';
import { EmptyState } from '@/components/ui/EmptyState';
import { CaseCard } from '@/features/_shared/CaseCard';
import { fa } from '@/lib/digits';

/** One searchable directory of the families in my caseload; no archive or
 * status tabs. A family joins this list when one of its visits becomes mine
 * (assigned by the unit or claimed from «بازدیدهای آزاد») — until then it
 * exists only as an anonymous pool visit. */
export function CasesScreen() {
  const { myCases: cases, myVisits: visits, status } = useApp();
  const [query, setQuery] = useState('');

  const visible = useMemo(() => {
    const needle = query.trim().toLowerCase();
    if (!needle) return cases;
    // Keep case references searchable without occupying the family's card.
    return cases.filter((c) =>
      [c.familyName, c.code, c.city, c.district ?? '', c.headOfHousehold]
        .join(' ').toLowerCase().includes(needle),
    );
  }, [cases, query]);

  const visitsByCase = useMemo(() => {
    const grouped = new Map<string, typeof visits>();
    for (const visit of visits) {
      const rows = grouped.get(visit.caseId) ?? [];
      rows.push(visit);
      grouped.set(visit.caseId, rows);
    }
    return grouped;
  }, [visits]);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        title="خانواده‌ها"
        subtitle={status === 'ready' ? `${fa(cases.length)} خانواده در حوزه مسئولیت شما` : 'خانواده‌های حوزه مسئولیت شما'}
      >
        <div className="mt-3">
          <SearchBar value={query} onChange={setQuery} placeholder="جست‌وجوی خانواده یا محله" />
        </div>
      </ScreenHeader>

      <div className="pb-8 pt-4">
        {!visible.length ? (
          <EmptyState
            icon={<FolderSearch className="size-6" />}
            title={query.trim() ? 'خانواده‌ای یافت نشد' : 'خانواده‌ای ثبت نشده'}
            description={query.trim() ? 'نام خانواده یا محله را بررسی کنید.' : undefined}
          />
        ) : (
          <ul className="space-y-3">
            {visible.map((family) => (
              <li key={family.id}>
                <CaseCard caseFile={family} visits={visitsByCase.get(family.id) ?? []} />
              </li>
            ))}
          </ul>
        )}
      </div>
    </Screen>
  );
}
