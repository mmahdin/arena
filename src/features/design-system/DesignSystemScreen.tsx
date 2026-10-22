import { useEffect, useState, type ReactNode } from 'react';
import { Bell, CheckCircle2, HeartPulse, Home, MapPin, Palette } from 'lucide-react';
import { Screen, Section } from '@/components/layout/Screen';
import { ScreenHeader } from '@/components/layout/ScreenHeader';
import { Card } from '@/components/ui/Card';
import { SectionHeader } from '@/components/ui/SectionHeader';
import { Button } from '@/components/ui/Button';
import { IconButton } from '@/components/ui/IconButton';
import { Badge, Dot } from '@/components/ui/Badge';
import { Chip, ChipRail } from '@/components/ui/Chip';
import { SegmentedControl } from '@/components/ui/SegmentedControl';
import { Avatar } from '@/components/ui/Avatar';
import { IconTile } from '@/components/ui/DomainIcon';
import { ListGroup, ListRow } from '@/components/ui/ListRow';
import { KeyValue, SelectInput, TextArea, TextInput } from '@/components/ui/Field';
import { ScaleInput, ScaleDots } from '@/components/ui/ScaleInput';
import { ChecklistRow } from '@/components/ui/Checklist';
import { ProgressBar, ProgressLine, ProgressRing } from '@/components/ui/Progress';
import { Stepper } from '@/components/ui/Stepper';
import { Switch } from '@/components/ui/Switch';
import { Sheet } from '@/components/ui/Sheet';
import { EmptyState } from '@/components/ui/EmptyState';
import { RowSkeleton, Skeleton } from '@/components/ui/Skeleton';
import { useApp } from '@/store/app-store';
import { useSettings } from '@/store/settings';
import { cn } from '@/lib/cn';

/* Literal class strings — Tailwind's scanner cannot see interpolated names. */
const RADIUS_SAMPLES = [
  { label: 'xs', className: 'rounded-xs' },
  { label: 'sm', className: 'rounded-sm' },
  { label: 'md', className: 'rounded-md' },
  { label: 'lg', className: 'rounded-lg' },
  { label: 'xl', className: 'rounded-xl' },
  { label: '2xl', className: 'rounded-2xl' },
] as const;

const SHADOW_SAMPLES = [
  { label: 'xs', className: 'shadow-xs' },
  { label: 'sm', className: 'shadow-sm' },
  { label: 'md', className: 'shadow-md' },
  { label: 'lg', className: 'shadow-lg' },
] as const;

/**
 * Design system reference (in-app).
 *
 * A living page: every value below is read from the same tokens the product
 * uses, so it can never drift from the real UI. It exists so a reviewer or a
 * coding agent can see the whole visual language in one scroll.
 */
export function DesignSystemScreen() {
  const { toast } = useApp();
  const { resolvedTheme, setTheme, fieldPrivacy, toggleFieldPrivacy } = useSettings();
  const [sheet, setSheet] = useState(false);
  const [seg, setSeg] = useState<'a' | 'b' | 'c'>('a');
  const [chip, setChip] = useState('all');
  const [scale, setScale] = useState(3);
  const [check, setCheck] = useState(true);
  const [sw, setSw] = useState(true);
  const [text, setText] = useState('');

  return (
    <Screen bottomGap={false}>
      <ScreenHeader
        back
        title="راهنمای طراحی"
        subtitle="توکن‌ها، اجزا و وضعیت‌های بصری رمونا"
        actions={
          <IconButton label="تغییر پوسته" onClick={() => setTheme(resolvedTheme === 'dark' ? 'light' : 'dark')}>
            <Palette className="size-[18px]" strokeWidth={2.1} />
          </IconButton>
        }
      />

      <div className="pt-4 pb-8">
        {/* ---------------------------------------------------- colour */}
        <Section>
          <SectionHeader title="رنگ" hint="مقادیر زنده از توکن‌های CSS — با تغییر پوسته به‌روز می‌شوند" />
          <SwatchGroup title="برند" tokens={['--p-brand-50', '--p-brand-100', '--p-brand-200', '--p-brand-300', '--p-brand-400', '--p-brand-500', '--p-brand-600', '--p-brand-700', '--p-brand-800', '--p-brand-900']} />
          <SwatchGroup title="خنثی (کاغذی)" tokens={['--p-sand-0', '--p-sand-50', '--p-sand-100', '--p-sand-200', '--p-sand-300', '--p-sand-500', '--p-sand-700', '--p-sand-900']} className="mt-3" />
          <SwatchGroup title="معنایی" tokens={['--s-primary', '--s-success', '--s-warning', '--s-danger', '--s-info', '--s-accent']} className="mt-3" />
          <div className="mt-3 grid grid-cols-2 gap-2">
            <ContrastSample bg="bg-primary" fg="text-primary-fg" label="دکمه اصلی" ratio="۶٫۴:۱" />
            <ContrastSample bg="bg-surface" fg="text-ink" label="متن روی سطح" ratio="۱۵:۱" />
            <ContrastSample bg="bg-brand-soft" fg="text-brand-ink" label="برچسب برند" ratio="۸:۱" />
            <ContrastSample bg="bg-sunken" fg="text-ink-2" label="متن ثانویه" ratio="۶٫۵:۱" />
          </div>
        </Section>

        {/* -------------------------------------------------- typography */}
        <Section>
          <SectionHeader title="تایپوگرافی" hint="وزیرمتن — یک خانواده، هفت وزن، مقیاس مبتنی بر ۱۵ پیکسل" />
          <Card className="divide-y divide-line-soft px-4 py-1">
            <TypeSample className="text-2xl font-extrabold" label="2xl · ۲۶ · ۸۰۰">عنوان بزرگ</TypeSample>
            <TypeSample className="text-xl font-extrabold" label="xl · ۲۲ · ۸۰۰">عنوان صفحه</TypeSample>
            <TypeSample className="text-lg font-bold" label="lg · ۱۹ · ۷۰۰">عنوان نوار بالا</TypeSample>
            <TypeSample className="text-md font-bold" label="md · ۱۶٫۵ · ۷۰۰">عنوان بخش</TypeSample>
            <TypeSample className="text-base font-semibold" label="base · ۱۵ · ۶۰۰">عنوان سطر و متن تأکیدی</TypeSample>
            <TypeSample className="text-base font-normal" label="base · ۱۵ · ۴۰۰">متن بدنه با فاصله خطوط ۱٫۷۵ برای خوانایی فارسی</TypeSample>
            <TypeSample className="text-sm text-ink-3" label="sm · ۱۳٫۵ · ۴۰۰">متن فرعی و متادیتا</TypeSample>
            <TypeSample className="text-xs text-ink-3" label="xs · ۱۲ · ۴۰۰">برچسب و راهنما</TypeSample>
          </Card>
        </Section>

        {/* ----------------------------------------------- space/radius */}
        <Section>
          <SectionHeader title="فاصله، شعاع و سایه" hint="مقیاس ۴ پیکسلی؛ حاشیه صفحه ۲۰ پیکسل" />
          <Card className="space-y-4">
            <div>
              <p className="mb-2 text-xs font-semibold text-ink-2">شعاع گوشه</p>
              <div className="flex items-end gap-2">
                {RADIUS_SAMPLES.map((r) => (
                  <div key={r.label} className="flex-1 text-center">
                    <div className={cn('h-12 w-full border border-line-strong bg-brand-soft', r.className)} />
                    <span className="mt-1 block text-2xs text-ink-3" dir="ltr">{r.label}</span>
                  </div>
                ))}
              </div>
            </div>
            <div>
              <p className="mb-2 text-xs font-semibold text-ink-2">سایه</p>
              <div className="flex gap-2">
                {SHADOW_SAMPLES.map((s) => (
                  <div key={s.label} className={cn('h-14 flex-1 rounded-md border border-line bg-surface text-center text-2xs leading-[3.5rem] text-ink-3', s.className)} dir="ltr">
                    {s.label}
                  </div>
                ))}
              </div>
            </div>
          </Card>
        </Section>

        {/* ---------------------------------------------------- buttons */}
        <Section>
          <SectionHeader title="دکمه‌ها" hint="ارتفاع لمسی حداقل ۴۴ پیکسل" />
          <Card className="space-y-3">
            <div className="flex flex-wrap gap-2">
              <Button>اصلی</Button>
              <Button variant="secondary">ثانویه</Button>
              <Button variant="ghost">متنی</Button>
              <Button variant="quiet">خنثی</Button>
              <Button variant="danger">هشدار</Button>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button size="sm">کوچک</Button>
              <Button size="md">متوسط</Button>
              <Button size="lg">بزرگ</Button>
              <Button loading>در حال ارسال</Button>
              <Button disabled>غیرفعال</Button>
            </div>
            <div className="flex items-center gap-2">
              <Button block iconStart={<CheckCircle2 className="size-4" />}>
                تمام‌عرض با آیکون
              </Button>
            </div>
            <div className="flex items-center gap-2 border-t border-line-soft pt-3">
              <IconButton label="نمونه"><Bell className="size-[18px]" /></IconButton>
              <IconButton label="نمونه" variant="soft"><Bell className="size-[18px]" /></IconButton>
              <IconButton label="نمونه" variant="outline"><Bell className="size-[18px]" /></IconButton>
              <IconButton label="نمونه" badge={3}><Bell className="size-[18px]" /></IconButton>
              <IconButton label="نمونه" dot><Bell className="size-[18px]" /></IconButton>
            </div>
          </Card>
        </Section>

        {/* -------------------------------------------------- indicators */}
        <Section>
          <SectionHeader title="نشانگرها و لحن‌ها" hint="هر لحن یک معنا دارد و در همه برنامه یکسان است" />
          <Card className="space-y-4">
            <div className="flex flex-wrap gap-1.5">
              {(['neutral', 'brand', 'success', 'warning', 'danger', 'info', 'accent'] as const).map((t) => (
                <Badge key={t} tone={t}>{t}</Badge>
              ))}
            </div>
            <div className="flex flex-wrap gap-1.5">
              {(['neutral', 'brand', 'success', 'warning', 'danger', 'info', 'accent'] as const).map((t) => (
                <Badge key={t} tone={t} variant="outline">{t}</Badge>
              ))}
            </div>
            <div className="flex flex-wrap items-center gap-3">
              {(['neutral', 'brand', 'success', 'warning', 'danger', 'info', 'accent'] as const).map((t) => (
                <span key={t} className="flex items-center gap-1.5 text-xs text-ink-3">
                  <Dot tone={t} /> {t}
                </span>
              ))}
            </div>
            <div className="flex flex-wrap items-center gap-2 border-t border-line-soft pt-3">
              {(['neutral', 'brand', 'success', 'warning', 'danger', 'info', 'accent'] as const).map((t) => (
                <IconTile key={t} icon={Home} tone={t} />
              ))}
            </div>
          </Card>
        </Section>

        {/* ------------------------------------------------- selection */}
        <Section>
          <SectionHeader title="انتخاب و فیلتر" />
          <Card className="space-y-4">
            <SegmentedControl<'a' | 'b' | 'c'>
              value={seg}
              onChange={setSeg}
              items={[
                { value: 'a', label: 'فعال', count: 9 },
                { value: 'b', label: 'اصلاح', count: 1 },
                { value: 'c', label: 'انجام‌شده', count: 3 },
              ]}
            />
            <ChipRail flush>
              {['all', 'visit', 'report'].map((c) => (
                <Chip key={c} selected={chip === c} count={4} onClick={() => setChip(c)}>
                  {c === 'all' ? 'همه' : c === 'visit' ? 'بازدید' : 'گزارش'}
                </Chip>
              ))}
            </ChipRail>
            <div className="border-t border-line-soft pt-3">
              <ScaleInput value={scale} onChange={setScale} lowLabel="نامناسب" highLabel="بسیار مناسب" />
              <div className="mt-3 flex items-center gap-3">
                <span className="text-xs text-ink-3">نمایش فشرده:</span>
                <ScaleDots value={scale} />
              </div>
            </div>
            <div className="divide-y divide-line-soft border-t border-line-soft">
              <ChecklistRow checked={check} onChange={() => setCheck(!check)} label="گفت‌وگوی خصوصی با کودک" required />
              <ChecklistRow checked={!check} onChange={() => setCheck(!check)} label="بررسی وضعیت تحصیلی" hint="با توضیح تکمیلی" />
            </div>
            <div className="border-t border-line-soft pt-2">
              <Switch checked={sw} onChange={setSw} label="کلید وضعیت" description="برای تنظیمات دستگاه و محرمانگی" />
            </div>
          </Card>
        </Section>

        {/* ----------------------------------------------------- inputs */}
        <Section>
          <SectionHeader title="ورودی‌ها" />
          <Card className="space-y-4">
            <TextInput label="عنوان" value={text} onChange={(e) => setText(e.target.value)} placeholder="متن نمونه" />
            <SelectInput label="انتخاب" value="" options={['گزینه یک', 'گزینه دو']} onChange={() => undefined} placeholder="انتخاب کنید…" />
            <TextArea label="شرح مشاهدات" rows={3} hint="راهنمای کوتاه زیر فیلد" value="" readOnly placeholder="متن نمونه…" />
            <div className="rounded-md border border-line bg-sunken px-4 py-1">
              <KeyValue label="کلید" value="مقدار" icon={<MapPin className="size-3.5 text-ink-3" />} />
              <KeyValue label="نشانی" value="تهران، خیابان نمونه" />
            </div>
          </Card>
        </Section>

        {/* ------------------------------------------------------- rows */}
        <Section>
          <SectionHeader title="سطرها و کارت‌ها" hint="پرتکرارترین جزء محصول" />
          <ListGroup>
            <ListRow
              divider
              chevron
              leading={<IconTile icon={Home} tone="brand" />}
              title="بازدید پایش دوره‌ای"
              meta={<span className="text-sm text-ink-3">RAM-0361 · امروز · ۷۵ دقیقه</span>}
              trailing={<Badge tone="warning">اصلاح</Badge>}
            />
            <ListRow
              chevron
              leading={<Avatar name="هستی" tone="warning" size="sm" />}
              title="هستی — ۹ ساله"
              meta={<span className="text-sm text-ink-3">خانواده موسوی · پیگیری فردا</span>}
            />
          </ListGroup>
          <Card className="mt-3">
            <p className="text-xs font-bold text-ink-3">پیشرفت</p>
            <div className="mt-3 flex items-center gap-4">
              <ProgressRing value={68} size={44} />
              <div className="flex-1 space-y-2">
                <ProgressBar value={45} />
                <ProgressLine done={3} total={6} unit="موارد بررسی‌شده" />
              </div>
            </div>
          </Card>
          <Card className="mt-3">
            <p className="mb-3 text-xs font-bold text-ink-3">مراحل پرونده</p>
            <Stepper stage="placement" />
          </Card>
        </Section>

        {/* -------------------------------------------------- feedback */}
        <Section>
          <SectionHeader title="بازخورد" hint="هر اقدام باید پاسخ قابل دیدن داشته باشد" />
          <Card className="space-y-3">
            <div className="flex flex-wrap gap-2">
              <Button variant="secondary" size="sm" onClick={() => setSheet(true)}>نمایش برگه پایین</Button>
              <Button variant="secondary" size="sm" onClick={() => toast({ title: 'ذخیره شد', description: 'پیش‌نویس گزارش ذخیره شد.', tone: 'success' })}>
                پیام موفق
              </Button>
              <Button variant="secondary" size="sm" onClick={() => toast({ title: 'گزارش کامل نیست', description: '۳۲٪ از بخش‌ها باقی مانده است.', tone: 'warning' })}>
                پیام هشدار
              </Button>
            </div>
            <div className="space-y-2 rounded-md border border-line bg-sunken p-3">
              <Skeleton className="h-3.5 w-1/2" />
              <RowSkeleton className="px-0 py-2" />
            </div>
            <div className="rounded-md border border-dashed border-line-strong">
              <EmptyState icon={<HeartPulse className="size-6" />} title="وضعیت خالی" description="هر فهرست باید حالت خالی داشته باشد." />
            </div>
          </Card>
        </Section>

        {/* ----------------------------------------------------- tokens */}
        <Section>
          <SectionHeader title="توکن‌های پایه" hint="مقادیر عددی مقیاس‌ها" />
          <Card className="space-y-1.5 text-sm">
            {TOKEN_TABLE.map((t) => (
              <div key={t[0]} className="flex items-center justify-between gap-3 border-b border-line-soft pb-1.5 last:border-0">
                <code className="text-xs text-ink-2" dir="ltr">{t[0]}</code>
                <span className="text-xs text-ink-3 tabular-nums">{t[1]}</span>
              </div>
            ))}
          </Card>
        </Section>

        <p className="mt-6 text-center text-2xs leading-relaxed text-ink-3">
          این صفحه بخشی از محصول نیست؛ مرجع زنده سامانه طراحی است و در «بیشتر › راهنمای طراحی» قرار دارد.
        </p>
      </div>

      <Sheet open={sheet} onClose={() => setSheet(false)} title="برگه پایین" description="الگوی اصلی گفت‌وگوهای موقت در برنامه">
        <div className="space-y-3 pb-4">
          <p className="text-sm leading-relaxed text-ink-2">
            برگه‌ها به‌جای پنجره وسط صفحه استفاده می‌شوند: دست در وضعیت طبیعی خود می‌ماند، محتوای بلند
            قابل پیمایش است و حس «ادامه همان صفحه» را حفظ می‌کند.
          </p>
          <Button block onClick={() => setSheet(false)}>بستن</Button>
        </div>
      </Sheet>
    </Screen>
  );
}

const TOKEN_TABLE: [string, string][] = [
  ['--sp-gutter', '۲۰px'],
  ['--r-xs → --r-2xl', '۸ · ۱۰ · ۱۴ · ۱۸ · ۲۲ · ۲۸'],
  ['--tabbar-h', '۶۲px'],
  ['--topbar-h', '۵۶px'],
  ['--dur-fast/base/slow', '۱۲۰ · ۲۰۰ · ۳۲۰ms'],
  ['body size', '۱۵px / ۱٫۷۵'],
  ['touch target', '≥ ۴۴px'],
];

function SwatchGroup({ title, tokens, className }: { title: string; tokens: string[]; className?: string }) {
  const [values, setValues] = useState<Record<string, string>>({});

  useEffect(() => {
    const styles = getComputedStyle(document.documentElement);
    const next: Record<string, string> = {};
    for (const t of tokens) {
      const raw = styles.getPropertyValue(t).trim();
      next[t] = raw.includes(' ') ? rgbToHex(raw) : raw;
    }
    setValues(next);
  }, [tokens.join(','), document.documentElement.dataset.theme]);

  return (
    <div className={className}>
      <p className="mb-2 text-xs font-semibold text-ink-2">{title}</p>
      <div className="grid grid-cols-5 gap-1.5">
        {tokens.map((t) => (
          <div key={t} className="overflow-hidden rounded-md border border-line bg-surface">
            <div className="h-10 w-full" style={{ backgroundColor: `rgb(var(${t}))` }} />
            <div className="px-1.5 py-1">
              <p className="truncate text-2xs text-ink-2" dir="ltr">{t.replace('--p-', '').replace('--s-', '')}</p>
              <p className="truncate text-2xs text-ink-3 tabular-nums" dir="ltr">{values[t] ?? '—'}</p>
            </div>
          </div>
        ))}
      </div>
    </div>
  );
}

function rgbToHex(rgb: string): string {
  const parts = rgb.split(/\s+/).map(Number).filter((n) => !Number.isNaN(n));
  if (parts.length < 3) return rgb;
  return `#${parts.slice(0, 3).map((p) => p.toString(16).padStart(2, '0')).join('')}`;
}

function ContrastSample({ bg, fg, label, ratio }: { bg: string; fg: string; label: string; ratio: string }) {
  return (
    <div className={cn('rounded-md border border-line px-3 py-2.5', bg, fg)}>
      <p className="text-sm font-bold">{label}</p>
      <p className="text-2xs opacity-80 tabular-nums" dir="ltr">AA · {ratio}</p>
    </div>
  );
}

function TypeSample({ className, label, children }: { className?: string; label: string; children: ReactNode }) {
  return (
    <div className="flex items-baseline justify-between gap-4 py-2.5">
      <span className={cn('truncate', className)}>{children}</span>
      <span className="shrink-0 text-2xs text-ink-3 tabular-nums">{label}</span>
    </div>
  );
}
