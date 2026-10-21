import { useState } from "react";
import { Check, Lock, ShieldCheck } from "lucide-react";
import { Page } from "@/components/app/Page";
import { TopBar } from "@/components/app/TopBar";
import { Button } from "@/components/ui/Button";
import { Badge } from "@/components/ui/Badge";
import { Field, Input, Textarea } from "@/components/ui/Field";
import { Switch } from "@/components/ui/Switch";
import { SegmentedControl } from "@/components/ui/SegmentedControl";
import { FilterChips } from "@/components/ui/FilterChips";
import { ProgressBar } from "@/components/ui/ProgressBar";
import { Avatar } from "@/components/ui/Avatar";
import { Alert } from "@/components/ui/Alert";
import type { Tone } from "@/lib/labels";
import "./DesignSystemPage.css";

const brandScale = [
  { step: "25", label: "پس‌زمینه بسیار ملایم" },
  { step: "50", label: "پس‌زمینه ملایم" },
  { step: "100", label: "حاشیه و تأکید" },
  { step: "200", label: "خط تأکید" },
  { step: "500", label: "رنگ اصلی روشن" },
  { step: "600", label: "رنگ اصلی" },
  { step: "700", label: "متن روی روشن" },
  { step: "800", label: "متن پررنگ" },
];

const tones: Tone[] = ["neutral", "brand", "success", "warning", "danger", "info", "accent"];

const toneNames: Record<Tone, string> = {
  neutral: "خنثی",
  brand: "برند",
  success: "موفق",
  warning: "هشدار",
  danger: "بحرانی",
  info: "اطلاع",
  accent: "تأکید",
};

const typeScale = [
  { token: "3xl", size: "۲۸px", sample: "برنامه امروز", weight: "Bold / ۷۰۰", className: "ds-type-3xl" },
  { token: "2xl", size: "۲۲px", sample: "خانواده موسوی", weight: "Bold / ۷۰۰", className: "ds-type-2xl" },
  { token: "lg", size: "۱۶px", sample: "گزارش پایش دوره ۳", weight: "Semibold / ۶۰۰", className: "ds-type-lg" },
  { token: "md", size: "۱۴px", sample: "متن اصلی رابط کاربری و توضیحات کوتاه", weight: "Regular / ۴۰۰", className: "ds-type-md" },
  { token: "xs", size: "۱۲px", sample: "برچسب وضعیت · ۲۹ مهر ۱۴۰۵", weight: "Medium / ۵۰۰", className: "ds-type-xs" },
];

export function DesignSystemPage() {
  const [segment, setSegment] = useState<"a" | "b" | "c">("a");
  const [chip, setChip] = useState("all");
  const [switchOn, setSwitchOn] = useState(true);

  return (
    <>
      <TopBar title="سیستم طراحی رامونا" />

      <Page>
        <header className="ds-hero">
          <p className="ds-hero__eyebrow">Ramona Platform · Design System ۰.۱</p>
          <h1 className="ds-hero__title">یک زبان بصری آرام، دقیق و حرفه‌ای</h1>
          <p className="ds-hero__text">
            این صفحه مرجع زنده توکن‌ها و کامپوننت‌های مشترک کارتابل مددکار اجتماعی است. همه
            اجزا از همین توکن‌ها استفاده می‌کنند تا توسعه قابلیت‌های بعدی بدون تغییر زبان بصری
            ممکن باشد.
          </p>
          <div className="ds-hero__badges">
            <Badge tone="brand">فارسی و RTL</Badge>
            <Badge tone="neutral">موبایل‌اول</Badge>
            <Badge tone="neutral">دسترسی‌پذیر</Badge>
          </div>
        </header>

        <section className="ds-section">
          <h2 className="ds-section__title">رنگ · Colour</h2>
          <p className="ds-section__lead">
            سبز گیاهی برای اعتماد و آرامش، خاک رس برای توجه، و پالت کاغذی گرم برای خوانایی طولانی‌مدت.
          </p>
          <div className="ds-scale">
            {brandScale.map((item) => (
              <div className="ds-swatch" key={item.step}>
                <span className="ds-swatch__chip" style={{ background: `var(--brand-${item.step})` }} />
                <span className="ds-swatch__name num">{item.step}</span>
                <span className="ds-swatch__label">{item.label}</span>
              </div>
            ))}
          </div>

          <div className="ds-tone-grid">
            {tones.map((tone) => (
              <div className={`ds-tone ds-tone--${tone}`} key={tone}>
                <span className="ds-tone__dot" />
                <span className="ds-tone__name">{toneNames[tone]}</span>
              </div>
            ))}
          </div>

          <div className="ds-surface-strip">
            <div className="ds-surface ds-surface--canvas">بوم</div>
            <div className="ds-surface ds-surface--surface">سطح</div>
            <div className="ds-surface ds-surface--sunken">فرورفته</div>
            <div className="ds-surface ds-surface--inverse">معکوس</div>
          </div>
        </section>

        <section className="ds-section">
          <h2 className="ds-section__title">تایپوگرافی · Typography</h2>
          <p className="ds-section__lead">
            وزیرمتن با ارتفاع خط باز برای فارسی؛ اعداد به‌صورت جدولی و همیشه فارسی نمایش داده می‌شوند.
          </p>
          <div className="ds-type-list">
            {typeScale.map((item) => (
              <div className="ds-type-row" key={item.token}>
                <div className="ds-type-row__head">
                  <span className="ds-type-token">{item.token}</span>
                  <span className="ds-type-meta num">{item.size} · {item.weight}</span>
                </div>
                <p className={item.className}>{item.sample}</p>
              </div>
            ))}
          </div>
        </section>

        <section className="ds-section">
          <h2 className="ds-section__title">فاصله و شعاع · Space & Radius</h2>
          <p className="ds-section__lead">
            شبکه چهار پیکسلی، حداقل ناحیه لمسی ۴۴ پیکسل و شعاع ۱۰ تا ۲۲ پیکسل برای کارت‌ها.
          </p>
          <div className="ds-space">
            {[4, 8, 12, 16, 24, 32].map((size) => (
              <div className="ds-space__row" key={size}>
                <span className="ds-space__label num">{size}px</span>
                <span className="ds-space__bar" style={{ width: `${size * 3}px` }} />
              </div>
            ))}
          </div>
          <div className="ds-radii">
            {[
              { r: "8", token: "xs" },
              { r: "14", token: "md" },
              { r: "18", token: "lg" },
              { r: "28", token: "2xl" },
            ].map((item) => (
              <div className="ds-radius" key={item.token}>
                <span className="ds-radius__box" style={{ borderRadius: `${item.r}px` }} />
                <span className="ds-radius__label">{item.token} · {item.r}px</span>
              </div>
            ))}
          </div>
        </section>

        <section className="ds-section">
          <h2 className="ds-section__title">کامپوننت‌ها · Components</h2>

          <div className="ds-group">
            <h3 className="ds-group__title">دکمه‌ها</h3>
            <div className="ds-row ds-row--wrap">
              <Button size="sm">ثبت بازدید</Button>
              <Button size="sm" variant="secondary">انصراف</Button>
              <Button size="sm" variant="soft">ذخیره پیش‌نویس</Button>
              <Button size="sm" variant="ghost">بازگشت</Button>
              <Button size="sm" variant="danger">لغو بازدید</Button>
              <Button size="sm" disabled>غیرفعال</Button>
            </div>
          </div>

          <div className="ds-group">
            <h3 className="ds-group__title">وضعیت‌ها</h3>
            <div className="ds-row ds-row--wrap">
              <Badge tone="success" dot>انجام‌شده</Badge>
              <Badge tone="info" dot>در انتظار انجام</Badge>
              <Badge tone="warning" dot>نیازمند اصلاح</Badge>
              <Badge tone="danger" dot>فوری</Badge>
              <Badge tone="brand">پایش</Badge>
              <Badge tone="neutral" variant="outline">بسته‌شده</Badge>
            </div>
            <div className="ds-progress-list">
              <ProgressBar value={72} />
              <ProgressBar value={40} tone="warning" />
              <ProgressBar value={18} tone="danger" />
            </div>
          </div>

          <div className="ds-group">
            <h3 className="ds-group__title">فیلتر و ناوبری درون‌صفحه</h3>
            <SegmentedControl
              ariaLabel="نمونه کنترل بخش‌ها"
              value={segment}
              onChange={(value) => setSegment(value)}
              items={[
                { value: "a", label: "امروز" },
                { value: "b", label: "پیش رو", count: 4 },
                { value: "c", label: "انجام‌شده" },
              ]}
            />
            <FilterChips
              ariaLabel="نمونه فیلتر"
              value={chip}
              onChange={setChip}
              items={[
                { value: "all", label: "همه", count: 9 },
                { value: "urgent", label: "فوری", count: 2 },
                { value: "report", label: "گزارش" },
                { value: "admin", label: "اداری" },
              ]}
            />
          </div>

          <div className="ds-group">
            <h3 className="ds-group__title">فرم‌ها</h3>
            <Field label="عنوان گزارش" htmlFor="ds-input" required hint="حداکثر ۸۰ کاراکتر">
              <Input id="ds-input" placeholder="مثلاً گزارش پایش دوره ۴" />
            </Field>
            <Field label="مشاهدات" htmlFor="ds-textarea" counter="۰ از ۵۰۰">
              <Textarea id="ds-textarea" rows={3} placeholder="مشاهدات خود را بنویسید…" />
            </Field>
            <Switch
              id="ds-switch"
              checked={switchOn}
              onChange={setSwitchOn}
              label="اعلان بازدیدها"
              description="یک ساعت پیش از هر بازدید"
            />
          </div>

          <div className="ds-group">
            <h3 className="ds-group__title">آواتار و افراد</h3>
            <div className="ds-row">
              <Avatar name="سارا محمدی" size="lg" />
              <Avatar name="آوا موسوی" size="md" />
              <Avatar name="یسنا احمدی" size="sm" />
              <Avatar name="خانواده کریمی" size="md" shape="squircle" icon={<Lock size={16} />} />
            </div>
          </div>

          <div className="ds-group">
            <h3 className="ds-group__title">پیام‌ها</h3>
            <Alert tone="warning" icon={<ShieldCheck size={15} />} title="دسترسی محدود">
              اطلاعات این پرونده تنها برای افراد مجاز قابل مشاهده است.
            </Alert>
            <Alert tone="brand" icon={<Check size={15} />}>
              گزارش با موفقیت برای بررسی سرپرست ارسال شد.
            </Alert>
          </div>
        </section>

        <section className="ds-section">
          <h2 className="ds-section__title">دسترسی‌پذیری · Accessibility</h2>
          <ul className="ds-rules">
            <li>حداقل ناحیه لمسی ۴۴×۴۴ پیکسل برای همه اقدام‌ها.</li>
            <li>حلقه فوکوس مشخص با کنتراست کافی روی همه عناصر تعاملی.</li>
            <li>وضعیت‌ها هرگز تنها با رنگ منتقل نمی‌شوند و همیشه متن یا آیکون دارند.</li>
            <li>احترام به تنظیم کاهش حرکت در سیستم‌عامل.</li>
            <li>ساختار عنوان‌ها و برچسب‌های ARIA برای صفحه‌خوان.</li>
          </ul>
        </section>
      </Page>
    </>
  );
}
