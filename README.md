# رهراه (Rahrah) — بازسازی اپلیکیشن «The Path»

بازسازی موبایل-فرستِ رابط کاربری اپلیکیشن The Path با وفاداری بصری بالا، به‌صورت یک
اپلیکیشن واقعی و قابل ویرایش با **React + Vite**، کاملاً **فارسی و RTL**.

مرجع: ۶۱ اسکرین‌شات داخل پوشهٔ `screens/` (ترتیب صحیح: `photo_61` → `photo_1`).

## اجرا

```bash
npm install
npm run dev      # http://localhost:5173
```

حداکثر عرض اپ ۴۳۰px است؛ روی موبایل تمام‌صفحه و روی دسکتاپ ستون وسط.

## ساختار

- `src/App.jsx` — مسیریاب پشته‌ای (stack router)، state سراسری mock، سیستم Toast
- `src/data.js` — تمام متن‌های فارسی، پرسش‌ها، راهنماها، آمارها
- `src/styles.css` — توکن‌های طراحی، تایپوگرافی (Estedad + Vazirmatn)، کامپوننت‌ها
- `src/components/` — آیکون‌ها و کامپوننت‌های مشترک (Pill, Stepper, Options, …)
- `src/screens/` — صفحه‌ها:
  - `Onboarding.jsx` — مسیر مارپیچ سبز (۴ گام) + صفحهٔ آغاز
  - `Intake.jsx` — کل جریان intake (نام، زبان، نقشهٔ راه، پرسش‌ها، اسلایدر، PHQ/GAD، آمارها، بازتاب‌ها)
  - `Guides.jsx` — انتخاب راهنما، جزئیات راهنما، مجوز حالت چهره، مدت جلسه
  - `Session.jsx` — جلسهٔ چت با راهنما
  - `Home.jsx` — خانه، کشو (منو)، جزئیات حوزهٔ تمرکز

## نگاشت ترتیب اسکرین‌شات‌ها به routeها

photo_61..58 → onboarding path · 57 splash · 56 name · 55 language · 54 roadmap ·
53 welcome · 52 gender · 51 age · 50 signup · 49 stat-orange · 48 country ·
47 free-about · 46 reflection · 45 therapy · 44 free-brings · 43 reflection ·
42 free-concerns · 41 carrying · 40 slider · 39 reflection · 38 stat-blue ·
37 eased · 36 interstitial · 35 less-heavy · 34 move-up · 33 stat-green62 ·
32 yourself · 31 person · 30 interstitial-private · 29 stat-green57 · 28/27 phq ·
26 found · 25/24 gad (+toast) · 23 which-ai · 22 home-empty · 21..10 guide-select ·
9/6 guide-detail · 8 expressions · 7 duration · 5/4 session · 3 home · 2 focus · 1 drawer

## یادداشت‌ها

- فونت‌ها self-host شده‌اند (`@fontsource-variable/estedad` و `vazirmatn`) تا بدون اینترنت هم درست نمایش داده شوند.
- چهرهٔ راهنماها تصویرسازی AI است (`public/guides/`)؛ دو مورد (مارکوس و تای) فعلاً جای‌گیر برداری SVG هستند.
- Toast «شناسایی نماگرفت» با کلید PrintScreen و Toast «جای‌گذاری از کلیپ‌بورد» با رویداد paste نمایش داده می‌شود.
