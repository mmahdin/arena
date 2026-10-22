import type { CaseFile, CaseStage, CaseType, Child, Wellbeing } from '@/types';
import { db } from '@/data/db';
import { latency } from './_utils';

export interface ChildSummary extends Child {
  caseCode: string;
  familyName: string;
  caseId: string;
}

export interface CaseQuery {
  query?: string;
  stage?: CaseStage;
  type?: CaseType;
}

function matches(c: CaseFile, q: CaseQuery): boolean {
  if (q.stage && c.stage !== q.stage) return false;
  if (q.type && c.type !== q.type) return false;
  if (q.query) {
    const needle = q.query.trim().toLowerCase();
    if (!needle) return true;
    const haystack = [c.familyName, c.code, c.city, c.district ?? '', c.headOfHousehold, ...c.children.map((ch) => ch.firstName)]
      .join(' ')
      .toLowerCase();
    return haystack.includes(needle);
  }
  return true;
}

export function listCases(q: CaseQuery = {}): Promise<CaseFile[]> {
  const order = { critical: 0, high: 1, medium: 2, low: 3 } as const;
  const result = db.cases
    .filter((c) => matches(c, q))
    .sort((a, b) => {
      return order[a.priority] - order[b.priority] || b.lastActivityAt.localeCompare(a.lastActivityAt);
    });
  return latency(result);
}

export function getCase(id: string): Promise<CaseFile | undefined> {
  return latency(db.cases.find((c) => c.id === id));
}

export function listChildren(filter?: { wellbeing?: Wellbeing }): Promise<ChildSummary[]> {
  const rows: ChildSummary[] = db.cases.flatMap((c) =>
    c.children
      .filter((ch) => c.stage !== 'closed')
      .map((ch) => ({ ...ch, caseCode: c.code, familyName: c.familyName, caseId: c.id })),
  );
  const filtered = filter?.wellbeing ? rows.filter((r) => r.wellbeing === filter.wellbeing) : rows;
  const order = { critical: 0, attention: 1, stable: 2 } as const;
  return latency(filtered.sort((a, b) => order[a.wellbeing] - order[b.wellbeing] || a.nextFollowUp.localeCompare(b.nextFollowUp)));
}

export function getChild(id: string): Promise<ChildSummary | undefined> {
  return latency(listChildrenSync().find((c) => c.id === id));
}

/** Sync variant for internal use — avoids promise chains in small helpers. */
export function listChildrenSync(): ChildSummary[] {
  return db.cases.flatMap((c) =>
    c.children.map((ch) => ({ ...ch, caseCode: c.code, familyName: c.familyName, caseId: c.id })),
  );
}

export function casesForCaseIds(ids: string[]): CaseFile[] {
  return db.cases.filter((c) => ids.includes(c.id));
}
