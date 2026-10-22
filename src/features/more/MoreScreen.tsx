import { HeartPulse, Bell, CircleUser, Settings, Palette, LifeBuoy, LogOut, ShieldCheck } from 'lucide-react';
import { useNavigate } from 'react-router-dom';
import { useApp } from '@/store/app-store';
import { Screen } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { ListGroup, ListRow } from '@/components/ui/ListRow';
import { IconTile } from '@/components/ui/DomainIcon';
import { Badge } from '@/components/ui/Badge';
import { Section } from '@/components/layout/Screen';
import { fa } from '@/lib/digits';
import { Logo } from '@/components/layout/Logo';

/**
 * More.
 *
 * The overflow tab is not a junk drawer: it holds the destinations that are
 * used less often: monitoring, notifications, profile and settings.
 * Reports have their own primary destination instead of appearing twice.
 */
export function MoreScreen() {
  const { myCases: cases, unreadCount, toast } = useApp();
  const navigate = useNavigate();

  const monitored = cases.filter((c) => c.stage !== 'closed').reduce((n, c) => n + c.children.length, 0);

  return (
    <Screen bottomGap={false}>
      <ScreenHeader back onBack={() => navigate('/profile')} title="امکانات دیگر" subtitle="بخش‌های تکمیلی فضای کاری" />

      <Section className="pt-4">
        <GroupLabel>پایش</GroupLabel>
        <ListGroup>
          <ListRow
            to="/monitoring"
            chevron
            leading={<IconTile icon={HeartPulse} tone="success" />}
            title="پایش کودکان"
            meta={`${fa(monitored)} کودک تحت پایش`}
          />
        </ListGroup>
      </Section>

      <Section>
        <GroupLabel>اطلاع‌رسانی</GroupLabel>
        <ListGroup>
          <ListRow
            to="/notifications"
            chevron
            leading={<IconTile icon={Bell} tone="brand" />}
            title="اعلان‌ها"
            meta={unreadCount > 0 ? `${fa(unreadCount)} اعلان خوانده‌نشده` : 'همه خوانده شده‌اند'}
            trailing={unreadCount > 0 ? <Badge tone="danger">{fa(unreadCount)}</Badge> : undefined}
          />
        </ListGroup>
      </Section>

      <Section>
        <GroupLabel>حساب کاربری</GroupLabel>
        <ListGroup>
          <ListRow to="/profile" chevron divider leading={<IconTile icon={CircleUser} tone="neutral" />} title="پروفایل حرفه‌ای" />
          <ListRow to="/settings" chevron leading={<IconTile icon={Settings} tone="neutral" />} title="تنظیمات" meta="پوسته، محرمانگی میدانی، اعلان‌ها" />
        </ListGroup>
      </Section>

      <Section>
        <GroupLabel>ابزارها</GroupLabel>
        <ListGroup>
          <ListRow
            to="/design-system"
            chevron
            divider
            leading={<IconTile icon={Palette} tone="accent" />}
            title="راهنمای طراحی"
            meta="توکن‌ها و نمونه اجزای رابط"
          />
          <ListRow
            chevron
            leading={<IconTile icon={LifeBuoy} tone="neutral" />}
            title="راهنما و پشتیبانی"
            onClick={() => toast({ title: 'پشتیبانی', description: 'مرکز پاسخ‌گویی در فاز بعدی فعال می‌شود.', tone: 'info' })}
          />
        </ListGroup>
      </Section>

      <Section>
        <button
          type="button"
          onClick={() => toast({ title: 'خروج از حساب', description: 'در نمونه طراحی، خروج غیرفعال است.', tone: 'warning' })}
          className="press flex w-full items-center justify-center gap-2 rounded-lg border border-line bg-surface py-3.5 text-base font-bold text-danger"
        >
          <LogOut className="size-[18px]" strokeWidth={2.25} aria-hidden />
          خروج از حساب کاربری
        </button>
      </Section>

      <div className="space-y-3 pb-8 pt-2">
        <div className="flex items-center justify-center gap-2 text-ink-3">
          <Logo size={16} className="text-brand" />
          <span className="text-2xs">رمونا · نسخه نمایشی ۰٫۱</span>
        </div>
        <p className="flex items-start gap-2 text-2xs leading-relaxed text-ink-3">
          <ShieldCheck className="mt-0.5 size-4 shrink-0" strokeWidth={2.25} aria-hidden />
          اطلاعات خانواده‌ها و کودکان محرمانه است و دسترسی بر اساس مسئولیت محوله محدود شده است.
        </p>
      </div>
    </Screen>
  );
}

function GroupLabel({ children }: { children: string }) {
  return <h2 className="mb-2 px-1 text-xs font-bold text-ink-3">{children}</h2>;
}
