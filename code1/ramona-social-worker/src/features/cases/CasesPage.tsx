import { useMemo, useState } from "react";
import { useSearchParams } from "react-router-dom";
import { FolderSearch, Search, ShieldCheck, Users } from "lucide-react";
import { useNow, useWorkspace } from "@/app/store";
import { caseStageShort } from "@/lib/labels";
import type { CaseStage } from "@/types";
import { cn } from "@/lib/cn";
import { Page, PageHeader } from "@/components/app/Page";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilterChips } from "@/components/ui/FilterChips";
import { SearchInput } from "@/components/ui/SearchInput";
import { EmptyState } from "@/components/ui/EmptyState";
import { CaseCard } from "@/components/domain/CaseCard";
import { MonitoringCard } from "@/components/domain/MonitoringCard";
import { monitoringChildren } from "@/lib/selectors";
import "./CasesPage.css";

type Tab = "cases" | "monitoring";
type StageFilter = "all" | Extract<CaseStage, "assessment" | "introduction" | "monitoring">;

const stageFilters: { value: StageFilter; label: string }[] = [
  { value: "all", label: "همه" },
  { value: "assessment", label: caseStageShort.assessment },
  { value: "introduction", label: caseStageShort.introduction },
  { value: "monitoring", label: caseStageShort.monitoring },
];

export function CasesPage() {
  const data = useWorkspace();
  const now = useNow();
  const [searchParams, setSearchParams] = useSearchParams();
  const [query, setQuery] = useState("");
  const [stage, setStage] = useState<StageFilter>("all");

  const tab: Tab = searchParams.get("tab") === "monitoring" ? "monitoring" : "cases";
  const children = monitoringChildren(data);
  const attentionCount = children.filter(
    (item) => item.child.monitoring?.wellbeing !== "stable"
  ).length;

  const stageCounts = useMemo(() => {
    const counts: Record<StageFilter, number> = {
      all: data.cases.length,
      assessment: 0,
      introduction: 0,
      monitoring: 0,
    };
    data.cases.forEach((familyCase) => {
      if (familyCase.stage === "assessment") counts.assessment += 1;
      if (familyCase.stage === "introduction") counts.introduction += 1;
      if (familyCase.stage === "monitoring") counts.monitoring += 1;
    });
    return counts;
  }, [data.cases]);

  const filteredCases = useMemo(() => {
    const normalized = query.trim();
    return data.cases
      .filter((familyCase) => (stage === "all" ? true : familyCase.stage === stage))
      .filter((familyCase) => {
        if (!normalized) return true;
        const haystack = [
          familyCase.title,
          familyCase.caseNumber,
          ...familyCase.children.map((child) => child.fullName),
          ...familyCase.guardians.map((guardian) => guardian.fullName),
        ].join(" ");
        return haystack.includes(normalized);
      })
      .sort((a, b) => (b.nextVisitAt ?? "").localeCompare(a.nextVisitAt ?? ""));
  }, [data.cases, query, stage]);

  const setTab = (next: Tab) =>
    setSearchParams(next === "monitoring" ? { tab: "monitoring" } : {}, { replace: true });

  return (
    <Page>
      <PageHeader eyebrow="پرونده‌های در اختیار شما" title="پرونده‌ها" />

      <SegmentedControl<Tab>
        ariaLabel="بخش پرونده‌ها"
        value={tab}
        onChange={setTab}
        items={[
          { value: "cases", label: "پرونده‌ها", count: data.cases.length },
          { value: "monitoring", label: "پایش کودکان", count: children.length },
        ]}
      />

      {tab === "cases" ? (
        <div className="cases-view">
          <SearchInput
            value={query}
            onChange={setQuery}
            ariaLabel="جست‌وجو در پرونده‌ها"
            placeholder="جست‌وجوی خانواده، کودک یا شماره پرونده"
          />

          <FilterChips<StageFilter>
            ariaLabel="فیلتر مرحله پرونده"
            items={stageFilters.map((item) => ({ ...item, count: stageCounts[item.value] }))}
            value={stage}
            onChange={setStage}
          />

          {filteredCases.length > 0 ? (
            <div className="case-list">
              {filteredCases.map((familyCase) => (
                <CaseCard key={familyCase.id} familyCase={familyCase} now={now} />
              ))}
            </div>
          ) : (
            <EmptyState
              icon={<FolderSearch size={22} />}
              title="پرونده‌ای پیدا نشد"
              description={
                query
                  ? "عبارت دیگری را جست‌وجو کنید یا فیلتر مرحله را بردارید."
                  : "در این مرحله پرونده‌ای وجود ندارد."
              }
              compact
            />
          )}

          <p className="cases-privacy">
            <ShieldCheck size={14} aria-hidden="true" />
            دسترسی شما به پرونده‌ها بر اساس ارجاع و سطح مجوز تعیین‌شده است.
          </p>
        </div>
      ) : (
        <div className="monitoring-view">
          <div className={cn("monitoring-summary", attentionCount > 0 && "has-alert")}>
            <div className="monitoring-summary__stat">
              <span className="monitoring-summary__value num">
                {new Intl.NumberFormat("fa-IR").format(children.length)}
              </span>
              <span className="monitoring-summary__label">
                <Users size={13} aria-hidden="true" />
                کودک در پایش
              </span>
            </div>
            <span className="monitoring-summary__divider" aria-hidden="true" />
            <div className="monitoring-summary__stat">
              <span
                className={cn(
                  "monitoring-summary__value num",
                  attentionCount > 0 && "is-warning"
                )}
              >
                {new Intl.NumberFormat("fa-IR").format(attentionCount)}
              </span>
              <span className="monitoring-summary__label">
                <Search size={13} aria-hidden="true" />
                نیازمند توجه
              </span>
            </div>
          </div>

          <div className="monitor-list">
            {children.map(({ child, familyCase }) => (
              <MonitoringCard key={child.id} child={child} familyCase={familyCase} now={now} />
            ))}
          </div>

          <p className="cases-privacy">
            <ShieldCheck size={14} aria-hidden="true" />
            نتایج پایش برای پیگیری وضعیت کودک و کیفیت مراقبت در خانواده استفاده می‌شود.
          </p>
        </div>
      )}
    </Page>
  );
}
