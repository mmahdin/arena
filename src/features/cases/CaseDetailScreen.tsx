import { useMemo } from 'react';
import { Link, useNavigate, useParams } from 'react-router-dom';
import { ChevronDown, ChevronLeft, MapPin, Phone } from 'lucide-react';
import type { CaseFile, FamilyMember, Report, Visit } from '@/types';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { EmptyState } from '@/components/ui/EmptyState';
import { PrivacyText } from '@/components/ui/PrivacyText';
import { Button } from '@/components/ui/Button';
import { visitDisplayState } from '@/lib/domain';
import { nextVisit, visitChronological, visitKindLabel, visitProgress } from '@/lib/visit-workflow';
import { fa, toLatinDigits } from '@/lib/digits';
import { faDate, faTime } from '@/lib/jalali';

/** Family details: essential contacts first, followed by the next visit and visit history. */
export function CaseDetailScreen() {
  const { caseId } = useParams();
  const navigate = useNavigate();
  const { cases, visits, reports } = useApp();
  const { fieldPrivacy } = useSettings();
  const family = cases.find((c) => c.id === caseId);

  const familyVisits = useMemo(() => visits.filter((v) => v.caseId === caseId), [visits, caseId]);
  const scheduled = familyVisits.filter((v) => visitDisplayState(v) === 'scheduled').sort(visitChronological);
  const history = familyVisits.filter((v) => visitDisplayState(v) !== 'scheduled')
    .sort((a, b) => visitChronological(b, a));
  const next = nextVisit(scheduled);

  if (!family) return (
    <Screen>
      <ScreenHeader back title="خانواده" />
      <EmptyState title="خانواده یافت نشد" description="اطلاعات این خانواده در دسترس نیست." />
      <Button block onClick={() => navigate('/cases')}>بازگشت به خانواده‌ها</Button>
    </Screen>
  );

  const contacts = getPrimaryContacts(family);
  const fullAddress = [family.city, family.district, family.address].filter(Boolean).join('، ');
  const mapHref = `https://www.google.com/maps/search/?api=1&query=${encodeURIComponent(fullAddress)}`;

  return (
    <Screen>
      {/* Keep the short app-bar title clear of the phone notch. The full family name sits below. */}
      <ScreenHeader back title="خانواده" />

      <main className="pb-7 pt-3">
        <h2 className="mb-3 text-lg font-extrabold leading-7 text-ink [overflow-wrap:anywhere]">
          <PrivacyText value={family.familyName} interactive={false} />
        </h2>

        <section aria-label="اطلاعات خانواده">
          <div className="overflow-hidden rounded-lg border border-line bg-surface px-4 shadow-xs">
            <div className="divide-y divide-line-soft">
              {contacts.map((contact) => (
                <ContactRow key={contact.id} person={contact} privateMode={fieldPrivacy} />
              ))}
            </div>

            <div className="flex items-center gap-3 border-t border-line-soft py-3.5 first:border-0">
              <div className="min-w-0 flex-1">
                <p className="text-xs text-ink-3">نشانی منزل</p>
                <p className="mt-1 text-sm leading-6 text-ink [overflow-wrap:anywhere]">
                  {fullAddress ? <PrivacyText value={fullAddress} interactive={false} /> : 'نشانی ثبت نشده'}
                </p>
              </div>
              {fullAddress && !fieldPrivacy ? (
                <a href={mapHref} target="_blank" rel="noopener noreferrer"
                  className="press grid size-11 shrink-0 place-items-center rounded-md border border-line bg-sunken/60 text-ink-2 hover:bg-sunken hover:text-ink"
                  aria-label="نمایش نشانی روی نقشه" title="نمایش نشانی روی نقشه">
                  <MapPin className="size-[19px]" aria-hidden />
                </a>
              ) : null}
            </div>
          </div>
        </section>

        {next && (
          <section className="mt-6" aria-label="بازدید بعدی">
            <h3 className="mb-2.5 text-sm font-bold text-ink">بازدید بعدی</h3>
            <Link to={`/visits/${next.id}`}
              className="press flex min-h-16 items-center gap-3 rounded-lg border border-line bg-surface px-4 py-3 shadow-xs hover:border-line-strong">
              <span className="min-w-0 flex-1">
                <span className="block text-sm font-semibold text-ink">{visitKindLabel(next)}</span>
                <span className="mt-1 block text-xs text-ink-3">
                  {faDate(next.date, { weekday: true, year: false })} · {faTime(next.time)}
                </span>
              </span>
              <ChevronLeft className="size-4 text-ink-3" aria-hidden />
            </Link>
          </section>
        )}

        <section className="mt-6" aria-label="سوابق بازدید">
          <div className="mb-2 flex items-center justify-between gap-3">
            <h3 className="text-sm font-bold text-ink">سوابق بازدید</h3>
            {history.length > 0 && <span className="text-xs text-ink-3">{fa(history.length)} مورد</span>}
          </div>
          {history.length ? (
            <ol>
              {history.map((visit, i) => (
                <VisitHistoryItem key={visit.id} visit={visit} reports={reports} last={i === history.length - 1} />
              ))}
            </ol>
          ) : (
            <p className="py-3 text-sm text-ink-3">هنوز بازدیدی ثبت نشده است.</p>
          )}
        </section>

        <FamilyExtras family={family} />
      </main>
    </Screen>
  );
}

type Contact = Pick<FamilyMember, 'id' | 'name' | 'relation' | 'phone'>;

function getPrimaryContacts(family: CaseFile): Contact[] {
  const head = family.members.find((m) => m.name === family.headOfHousehold)
    ?? family.members.find((m) => !m.relation.includes('همسر') && (m.relation.includes('متقاضی') || m.relation.includes('سرپرست')));
  const spouse = family.members.find((m) => m.relation.includes('همسر'));
  const primary: Contact[] = [];
  if (head) {
    primary.push({ ...head, phone: head.phone || family.phone });
  } else if (family.headOfHousehold) {
    primary.push({ id: 'primary', name: family.headOfHousehold, relation: 'سرپرست خانوار', phone: family.phone });
  }
  if (spouse && spouse.id !== head?.id) primary.push(spouse);
  return primary;
}

function ContactRow({ person, privateMode }: { person: Contact; privateMode: boolean }) {
  const phone = person.phone?.trim();
  const dial = phone ? toLatinDigits(phone).replace(/[^0-9+]/g, '') : '';

  return (
    <div className="flex min-h-[84px] items-center gap-3 py-3">
      <div className="min-w-0 flex-1">
        <p className="text-xs text-ink-3">{person.relation}</p>
        <p className="mt-0.5 whitespace-normal text-sm font-semibold leading-6 text-ink [overflow-wrap:anywhere]">
          <PrivacyText value={person.name} interactive={false} />
        </p>
        <p className="mt-0.5 text-xs text-ink-2">
          {phone
            ? <span dir="ltr" className="inline-block"><PrivacyText value={phone} /></span>
            : 'شماره ثبت نشده'}
        </p>
      </div>
      {dial && !privateMode ? (
        <a href={`tel:${dial}`}
          className="press grid size-11 shrink-0 place-items-center rounded-md border border-line bg-sunken/60 text-ink-2 hover:bg-sunken hover:text-ink"
          aria-label={`تماس با ${person.name}`} title={`تماس با ${person.name}`}>
          <Phone className="size-[19px]" aria-hidden />
        </a>
      ) : null}
    </div>
  );
}

function VisitHistoryItem({ visit, reports, last }: {
  visit: Visit;
  reports: Report[];
  last: boolean;
}) {
  const progress = visitProgress(visit, reports);
  const needsAction = progress.tone === 'warning' || progress.tone === 'danger';

  return (
    <li className="relative ps-5">
      {!last && <span className="absolute start-[5px] top-7 h-full w-px bg-line" aria-hidden />}
      <span className="absolute start-0.5 top-[23px] size-[7px] rounded-full bg-ink-3 ring-[3px] ring-bg" aria-hidden />
      <Link to={`/visits/${visit.id}`}
        className={`press flex min-h-[69px] min-w-0 items-center gap-3 py-3 text-start hover:bg-sunken/40 ${last ? '' : 'border-b border-line-soft'}`}>
        <span className="min-w-0 flex-1">
          <span className="block text-sm font-semibold text-ink">{visitKindLabel(visit)}</span>
          <span className="mt-0.5 flex flex-wrap items-center gap-x-2 gap-y-0.5 text-xs text-ink-3">
            <span>{faDate(visit.date, { year: false })} · {faTime(visit.time)}</span>
            <span className={needsAction ? 'font-medium text-ink-2' : undefined}>{progress.label}</span>
          </span>
        </span>
        <ChevronLeft className="size-4 text-ink-3" aria-hidden />
      </Link>
    </li>
  );
}

/** Additional family context is optional and never crowds the visit history. */
function FamilyExtras({ family }: { family: CaseFile }) {
  const hasExtra = !!(family.notes || family.housing || family.income || family.children.length || family.documents.length);
  if (!hasExtra) return null;
  return (
    <details className="group mt-6 border-t border-line-soft pt-1">
      <summary className="press flex min-h-12 cursor-pointer list-none items-center justify-between gap-2 text-sm font-semibold text-ink-2 [&::-webkit-details-marker]:hidden">
        اطلاعات بیشتر
        <ChevronDown className="size-4 text-ink-3 transition-transform group-open:rotate-180" aria-hidden />
      </summary>
      <div className="space-y-3 rounded-lg bg-surface p-4 text-sm leading-7 text-ink-2">
        {family.housing && <p><strong className="font-semibold text-ink">مسکن: </strong>{family.housing.tenure} · {fa(family.housing.rooms)} اتاق · {family.housing.condition}</p>}
        {family.income && <p><strong className="font-semibold text-ink">وضعیت اقتصادی: </strong>{family.income}</p>}
        {family.notes && <p>{family.notes}</p>}
        {family.children.length > 0 && (
          <div>
            <p className="font-semibold text-ink">کودکان مرتبط</p>
            {family.children.map((child) => (
              <Link key={child.id} to={`/children/${child.id}`} className="press flex min-h-11 items-center justify-between border-b border-line-soft text-ink-2 last:border-0 hover:text-ink">
                <PrivacyText value={child.firstName} interactive={false} />
                <ChevronLeft className="size-4" aria-hidden />
              </Link>
            ))}
          </div>
        )}
        {family.documents.length > 0 && (
          <div>
            <p className="font-semibold text-ink">مستندات</p>
            {family.documents.map((document) => (
              <p key={document.id} className="border-b border-line-soft py-1 last:border-0">{document.title}</p>
            ))}
          </div>
        )}
      </div>
    </details>
  );
}
