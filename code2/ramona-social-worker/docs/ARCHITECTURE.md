# معماری رمونا · فضای کاری مددکار

هدف: بنیان فرانت‌اندی که بتوان incrementally بزرگ شد — صفحه‌ها و قابلیت‌ها اضافه شوند بدون
بازنویسی لایه‌های زیرین، و اتصال به بک‌اند واقعی فقط بدنهٔ `services/` را عوض کند.

## لایه‌ها و جهت وابستگی

```
types/  ←  lib/  ←  data/(mock+db)  ←  services/  ←  store/  ←  features/ ← app/
                          ▲
                      components/ (ui + layout)  ← features/
```

- هر لایه فقط به لایه‌های سمت چپ خود وابسته است؛ حلقهٔ import ممنوع
  (`lib/jalali → lib/digits` یک‌طرفه است).
- `features/*` هیچ‌گاه `data/mock` را import نمی‌کند؛ منبع داده همیشه `services` است.
- `components/ui` به دامنه وابسته نیست؛ اجزای دامنه‌ای مشترک در `features/_shared`اند.

## جریان داده

1. `store/app-store.tsx` در mount با `reload()` همه مجموعه‌ها را از `services` می‌گیرد
   (`status: loading → ready | error`) و در context نگه می‌دارد.
2. `db.ts` یک دیتابیس درون‌حافظه است که **پیش از resolve شدن تأخیر شبیه‌سازی‌شده**
   جهش می‌یابد؛ بنابراین `useMemo(..., [store arrays])` در صفحه‌ها همیشه با آخرین وضعیت
   همگام است و نیازی به refetch دستی پس هر intent نیست.
3. جهش‌ها فقط از طریق **intentهای store** انجام می‌شوند:
   `setTaskStatus`, `postponeTask`, `toggleChecklistItem`, `recordVisitOutcome`,
   `editReportField`, `editReportMeta`, `submitReport`, `createReportDraft`,
   `markNotificationRead/All`. هر intent: صدا زدن سرویس → به‌روز کردن state → toast
   (+ عوارض جانبی دامنه مثل ساخت پیش‌نویس گزارش یا اعلان).
4. صفحه‌ها داده مشتقی را با توابع خالص `lib/domain.ts` می‌سازند (`taskBucket`،
   `visitDisplayState`، `reportCompletion`، `buildToday` در `services/dashboard`)؛
   هیچ منطق دامنه‌ای داخل JSX نوشته نمی‌شود.

## قوانین دامنه که در کد جاری‌اند

- کارها توسط سامانه/سرپرست assign می‌شوند؛ مددکار فقط وضعیت را عوض می‌کند یا مهلت را
   با `postponeTask` جابه‌جا می‌کند (ثبت دلیل در نمونهٔ فعلی اختیاری است).
- `recordVisitOutcome` یک تراکنش دامنه‌ای است: بستن بازدید + بستن کار مرتبط + رویداد
   خط زمانی پرونده + ساخت خودکار پیش‌نویس گزارش بازدید؛ toast آن action «ادامه گزارش» دارد.
- گزارش پس از `submitReport` به وضعیت `in-review` می‌رود و اعلان برای سرپرست ساخته
   می‌شود؛ بازگشت برای اصلاح = وضعیت `revision` + کامنت سرپرست (در صفحه ویرایش دیده می‌شود).
- دسترسی: فهرست‌ها فقط دادهٔ «حوزه مسئولیت» کاربر فعلی را برمی‌گردانند و رابط این را
   صریح می‌گوید («۷ پرونده فعال در حوزه مسئولیت شما»)؛ تصمیم‌های نهایی جای مددکار نیست
   (متن‌های Profile و Monitoring).

## مسیریابی و پوسته

- `AppShell` قاب را می‌سازد: `.stage > .device > .device-viewport` (در دسکتاپ قاب گوشی،
  در موبایل تمام‌صفحه) → ستون flex شامل `#app-scroll` (اسکرول اصلی)، `TabBar`
  (فقط در ۵ مسیر اصلی)، `#app-overlay` (مقصد portal Sheet/Toasts) و `ToastHost`.
- اسکرول با تغییر مسیر صفر می‌شود؛ هدرها/فوترها چسبان‌اند و با safe-area هماهنگ‌اند.
- overlayها هرگز به `document.body` portal نمی‌شوند — فقط `#app-overlay` تا داخل قاب
  دستگاه بمانند و قفل اسکرول درست کار کند.

## افزودن‌ها بدون شکستن لایه‌ها

| می‌خواهید | مسیر پیشنهادی |
| --- | --- |
| صفحه جدید | `features/x/XScreen.tsx` + ثبت مسیر در `app/App.tsx` + (اختیاری) ورودی در `MoreScreen` |
| موجودیت جدید | نوع در `types/` → نمونه در `data/mock/` → ثبت در `data/db.ts` → توابع خواندن/نوشتن در `services/x.ts` → intent در store اگر جهش دارد |
| فرم گزارش جدید | فقط `data/mock/report-templates.ts`؛ رندر خودکار با `ReportFieldControl` |
| وضعیت/رنگ جدید | `tokens.css` + `tailwind.config.ts` + `lib/labels.ts` (برچسب فارسی + tone) |
| رویداد تقویم جدید | الحاق به `services/calendar.ts#buildEvents` — تقویم و امروز خودکار می‌گیرند |

## قراردادها و ابزارها

- TypeScript strict؛ alias `@/` → `src/`.
- کلاس‌های Tailwind فقط semantic؛ کلاس dynamic ممنوع (نگاشت صریح، مثل `ui/tone.ts`).
- ارقام و تاریخ: `lib/digits.ts#fa` و `lib/jalali.ts` (جلالی، بازه‌ها، «امروز/فردا»،
  `buildMonthGrid` برای تقویم).
- برچسب‌های فارسی وضعیت‌ها فقط از `lib/labels.ts` (هر enum → `{label, tone}`).
- کامپوننت‌ها تابعی و کوچک؛ صفحه‌ها از `Screen/ScreenHeader/Section/ActionFooter`
  ترکیب می‌شوند تا ریتم فاصله یکسان بماند.
- خطاها: `status === 'error'` در store → `EmptyState` با اقدام تلاش مجدد.

## بررسی کیفیت (workflow توسعه با Codex)

1. `npm run typecheck` و `npm run build` همیشه سبز بمانند.
2. اسکرین‌شات Playwright از مسیرها در دو حالت mobile/desktop و دو پوسته
   (اسکریپت نمونه در ریشه workspace بیرون مخزن نگهداری می‌شود).
3. چک‌لیست هر صفحه: gutter یکسان، هدف لمسی ≥44px، متن دوم با `ink-2/3`، وضعیت خالی/
   بارگذاری دارد، RTL درست (آیکون جهت‌دار آینه‌ای)، در پوسته تاریک کنتراست حفظ شود.
