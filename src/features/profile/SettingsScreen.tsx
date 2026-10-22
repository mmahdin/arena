import type { ReactNode } from 'react';
import { EyeOff, Moon, Sun, Monitor } from 'lucide-react';
import { Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Card, ToneCard } from '@/components/ui/Card';
import { Switch } from '@/components/ui/Switch';
import { SectionHeader } from '@/components/ui/SectionHeader';
import { useSettings, type ThemeMode } from '@/store/settings';
import { Logo } from '@/components/layout/Logo';
import { cn } from '@/lib/cn';

/**
 * Settings.
 *
 * Only settings that change how the app behaves for this professional:
 * appearance, field privacy and notification scope. Everything else belongs to
 * the platform's admin tools, which a social worker should never see.
 */
export function SettingsScreen() {
  const { theme, setTheme, fieldPrivacy, toggleFieldPrivacy, notifyAssignments, notifySchedule, notifyReviews, setNotify } =
    useSettings();

  return (
    <Screen bottomGap={false}>
      <ScreenHeader title="تنظیمات" back />

      <Section className="pt-4">
        <SectionHeader title="ظاهر" hint="پوسته روشن برای محیط اداری، پوسته تیره برای کار شبانه" />
        <Card padded={false} className="p-1.5">
          <div className="grid grid-cols-3 gap-1.5">
            <ThemeOption active={theme === 'light'} onClick={() => setTheme('light')} icon={<Sun className="size-[18px]" />} label="روشن" />
            <ThemeOption active={theme === 'dark'} onClick={() => setTheme('dark')} icon={<Moon className="size-[18px]" />} label="تیره" />
            <ThemeOption active={theme === 'system'} onClick={() => setTheme('system')} icon={<Monitor className="size-[18px]" />} label="خودکار" />
          </div>
        </Card>
      </Section>

      <Section>
        <SectionHeader title="محرمانگی میدانی" />
        <Card padded={false}>
          <Switch
            checked={fieldPrivacy}
            onChange={toggleFieldPrivacy}
            label="حالت محرمانگی"
            description="نام خانواده، نشانی و شماره تماس روی صفحه محو نمایش داده می‌شود؛ برای استفاده در فضای عمومی."
          />
        </Card>
        <div className="mt-2.5">
          <ToneCard tone={fieldPrivacy ? 'accent' : 'info'} className="flex items-start gap-3">
            <EyeOff className="mt-0.5 size-5 shrink-0" strokeWidth={2.25} aria-hidden />
            <p className="text-xs leading-relaxed opacity-90">
              {fieldPrivacy
                ? 'حالت محرمانگی فعال است. برای دیدن هر مقدار می‌توانید روی آن لمس کنید؛ مقدار پس از چند ثانیه دوباره محو می‌شود.'
                : 'با فعال کردن این حالت، اطلاعات هویتی به‌صورت محو نمایش داده می‌شوند و با یک لمس، موقتاً قابل مشاهده‌اند.'}
            </p>
          </ToneCard>
        </div>
      </Section>

      <Section>
        <SectionHeader title="اعلان‌ها" hint="چه رویدادهایی برای شما اعلان شود" />
        <Card padded={false} className="divide-y divide-line-soft">
          <Switch
            checked={notifyAssignments}
            onChange={(v) => setNotify('notifyAssignments', v)}
            label="مأموریت و پرونده جدید"
            description="ارجاع پرونده یا افزودن کار جدید به فهرست شما"
          />
          <Switch
            checked={notifySchedule}
            onChange={(v) => setNotify('notifySchedule', v)}
            label="تغییر برنامه بازدید"
            description="جابه‌جایی زمان، لغو یا هماهنگی مجدد بازدیدها"
          />
          <Switch
            checked={notifyReviews}
            onChange={(v) => setNotify('notifyReviews', v)}
            label="بازبینی گزارش‌ها"
            description="تأیید، بازگشت برای اصلاح یا درخواست تکمیل توسط سرپرست"
          />
        </Card>
      </Section>

      <Section>
        <SectionHeader title="درباره" />
        <Card className="divide-y divide-line-soft px-4 py-1">
          <Row label="برنامه" value="رمونا — مددکار اجتماعی" />
          <Row label="نسخه" value="۰٫۱٫۰ (نمونه طراحی)" />
          <Row label="زبان و جهت" value="فارسی · راست‌به‌چپ" />
          <Row label="تقویم" value="هجری شمسی" />
        </Card>
      </Section>

      <div className="flex items-center justify-center gap-2 pb-8 pt-4 text-ink-3">
        <Logo size={18} className="text-brand" />
        <span className="text-xs">Ramona Platform — Social Worker App</span>
      </div>
    </Screen>
  );
}

function ThemeOption({
  active,
  onClick,
  icon,
  label,
}: {
  active: boolean;
  onClick: () => void;
  icon: ReactNode;
  label: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={active}
      className={cn(
        'press flex h-16 flex-col items-center justify-center gap-1.5 rounded-md border text-sm font-semibold',
        active ? 'border-brand bg-brand-soft text-brand-ink' : 'border-line bg-surface text-ink-2 hover:bg-sunken',
      )}
    >
      {icon}
      {label}
    </button>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex items-center justify-between gap-4 py-3">
      <span className="text-sm text-ink-3">{label}</span>
      <span className="text-sm font-medium text-ink">{value}</span>
    </div>
  );
}
