import { useMemo, useState } from 'react';
import { useNavigate } from 'react-router-dom';
import { Search, SearchX, Baby } from 'lucide-react';
import { useApp } from '@/store/app-store';
import { Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { SearchBar } from '@/components/ui/SearchBar';
import { Card } from '@/components/ui/Card';
import { Chip } from '@/components/ui/Chip';
import { ListRow } from '@/components/ui/ListRow';
import { IconTile, REPORT_TYPE_ICON, TASK_ICON } from '@/components/ui/DomainIcon';
import { Avatar } from '@/components/ui/Avatar';
import { EmptyState } from '@/components/ui/EmptyState';
import { CASE_TYPE, REPORT_STATUS, REPORT_TYPE, TASK_TYPE, WELLBEING } from '@/lib/labels';
import { fa } from '@/lib/digits';
import { faDate } from '@/lib/jalali';
import { Meta } from '@/features/_shared/Meta';

const RECENT = ['موسوی', 'ارزیابی اولیه', 'نجفی', 'هستی'];

/**
 * Global search.
 *
 * One field, all record types. In the field a social worker rarely remembers
 * whether a name belongs to a case, a child or a report — so results are
 * grouped by kind instead of asking the user to choose first.
 */
export function SearchScreen() {
  const navigate = useNavigate();
  const { cases, reports, tasks } = useApp();
  const [query, setQuery] = useState('');
  const q = query.trim().toLowerCase();

  const results = useMemo(() => {
    if (!q) return { cases: [], children: [], reports: [], tasks: [] };
    const matchedCases = cases.filter((c) =>
      [c.familyName, c.code, c.city, c.district ?? '', c.headOfHousehold].join(' ').toLowerCase().includes(q),
    );
    const matchedChildren = cases.flatMap((c) =>
      c.children.filter((ch) => ch.firstName.toLowerCase().includes(q)).map((ch) => ({ child: ch, caseFile: c })),
    );
    const matchedReports = reports.filter((r) => [r.title, r.caseCode, r.summary].join(' ').toLowerCase().includes(q));
    const matchedTasks = tasks.filter((t) => [t.title, t.caseCode ?? '', t.assignedBy].join(' ').toLowerCase().includes(q));
    return { cases: matchedCases, children: matchedChildren, reports: matchedReports, tasks: matchedTasks };
  }, [q, cases, reports, tasks]);

  const total = results.cases.length + results.children.length + results.reports.length + results.tasks.length;

  return (
    <Screen bottomGap={false}>
      <ScreenHeader back title="جست‌وجو">
        <div className="mt-3">
          <SearchBar value={query} onChange={setQuery} autoFocus placeholder="نام خانواده، کودک، کد پرونده یا گزارش" />
        </div>
      </ScreenHeader>

      <div className="pt-3 pb-8">
        {!q ? (
          <Section>
            <p className="mb-2 px-1 text-xs font-bold text-ink-3">جست‌وجوهای اخیر</p>
            <div className="flex flex-wrap gap-1.5">
              {RECENT.map((r) => (
                <Chip key={r} onClick={() => setQuery(r)}>
                  {r}
                </Chip>
              ))}
            </div>

            <Card className="mt-5 flex items-start gap-3">
              <IconTile icon={Search} tone="brand" />
              <div>
                <p className="text-base font-semibold text-ink">جست‌وجوی سراسری</p>
                <p className="mt-1 text-sm leading-relaxed text-ink-3">
                  نتیجه‌ها بر اساس پرونده، کودک، گزارش و کار دسته‌بندی می‌شوند. فقط مواردی که در حوزه
                  دسترسی شما هستند نمایش داده می‌شوند.
                </p>
              </div>
            </Card>
          </Section>
        ) : total === 0 ? (
          <EmptyState
            icon={<SearchX className="size-6" />}
            title="نتیجه‌ای یافت نشد"
            description={`موردی مطابق «${query}» در حوزه دسترسی شما وجود ندارد.`}
          />
        ) : (
          <div className="space-y-6">
            <p className="px-1 text-xs text-ink-3">{fa(total)} نتیجه برای «{query}»</p>

            {results.cases.length > 0 && (
              <Section>
                <GroupTitle>پرونده‌ها</GroupTitle>
                <Card padded={false} className="divide-y divide-line-soft">
                  {results.cases.map((c) => (
                    <ListRow
                      key={c.id}
                      to={`/cases/${c.id}`}
                      chevron
                      leading={<Avatar name={c.familyName} tone={CASE_TYPE[c.type].tone} size="sm" />}
                      title={c.familyName}
                      meta={<span className="text-sm text-ink-3">{CASE_TYPE[c.type].label} · {c.district ?? c.city}</span>}
                    />
                  ))}
                </Card>
              </Section>
            )}

            {results.children.length > 0 && (
              <Section>
                <GroupTitle>کودکان</GroupTitle>
                <Card padded={false} className="divide-y divide-line-soft">
                  {results.children.map(({ child, caseFile }) => (
                    <ListRow
                      key={child.id}
                      to={`/children/${child.id}`}
                      chevron
                      leading={<Avatar name={child.firstName} tone={WELLBEING[child.wellbeing].tone} size="sm" icon={<Baby className="size-4" />} />}
                      title={`${child.firstName} — ${fa(child.age)} ساله`}
                      meta={<Meta><span>{caseFile.familyName}</span><span>{WELLBEING[child.wellbeing].label}</span></Meta>}
                    />
                  ))}
                </Card>
              </Section>
            )}

            {results.reports.length > 0 && (
              <Section>
                <GroupTitle>گزارش‌ها</GroupTitle>
                <Card padded={false} className="divide-y divide-line-soft">
                  {results.reports.map((r) => (
                    <ListRow
                      key={r.id}
                      to={`/reports/${r.id}`}
                      chevron
                      leading={<IconTile icon={REPORT_TYPE_ICON[r.type]} tone={REPORT_TYPE[r.type].tone} size="sm" />}
                      title={r.title}
                      meta={<Meta><span className="tabular-nums">{r.caseCode}</span><span>{REPORT_STATUS[r.status].label}</span><span>{faDate(r.updatedAt, { year: false })}</span></Meta>}
                    />
                  ))}
                </Card>
              </Section>
            )}

            {results.tasks.length > 0 && (
              <Section>
                <GroupTitle>کارها</GroupTitle>
                <Card padded={false} className="divide-y divide-line-soft">
                  {results.tasks.map((t) => (
                    <ListRow
                      key={t.id}
                      onClick={() => navigate(t.visitId ? `/visits/${t.visitId}` : t.reportId ? `/reports/${t.reportId}` : t.caseId ? `/cases/${t.caseId}` : `/visits?date=${t.dueDate}`)}
                      chevron
                      leading={<IconTile icon={TASK_ICON[t.type]} tone={TASK_TYPE[t.type].tone} size="sm" />}
                      title={t.title}
                      meta={<Meta><span>{TASK_TYPE[t.type].label}</span><span className="tabular-nums">{faDate(t.dueDate, { year: false })}</span></Meta>}
                    />
                  ))}
                </Card>
              </Section>
            )}
          </div>
        )}
      </div>
    </Screen>
  );
}

function GroupTitle({ children }: { children: string }) {
  return <h2 className="mb-2 px-1 text-sm font-bold text-ink-2">{children}</h2>;
}
