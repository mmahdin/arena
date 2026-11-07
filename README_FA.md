> **هویت فعال — ۲۰۲۶-۱۱-۰۷:** نام محصول **تیا / Theia** است. تصویرسازی‌های گل‌مانندٔ قبلی با هویت نور/بازتاب جایگزین شده‌اند، اما چیدمان، تایپوگرافی، RTL، ترتیب صفحات و عملکرد موجود حفظ شده است. یادداشت‌های تاریخی همراه در پایین برای مرزهای بک‌اند نگه داشته شده‌اند.

## اتصال واقعی Chat و History — HAMRAH-INTEGRATION-03

این مرحله **بعد از اعمال Patch ورود آزمایشی v2** فعال می‌شود. از طراحی جدید همراه استفاده می‌کند، بدون تغییر تم و تصویرسازی.

`GET/POST /api/v1/conversations`، `GET /api/v1/conversations/{id}/messages`، `POST /api/v1/conversations/{id}/turns` و `DELETE /api/v1/conversations/{id}` به بک‌اند واقعی وصل شده‌اند. پیام تکراری بعد از خطای شبکه با همان شناسهٔ نوبت و متن ارسال می‌شود. پیام پاسخ نمایشی حذف شده است.

در کنار Go/MSSQL و `LLM_PROVIDER=mock`:

```bash
npm ci
npm test
npm run build
npm run dev
# With real Go + MSSQL started and Playwright installed in test environment:
npm run test:e2e:chat
```

لیست و تاریخچه به ۱۰۰ رکورد اخیر محدودند؛ پاسخ واقعی LLM و قابلیت‌های بینش/صدا هنوز تست یا پیاده‌سازی نشده‌اند. فقط دادهٔ ساختگی.


---

# همراه — ریپازیتوری فرانت‌اند با مرجع طراحی جدید

این ZIP از همان UI آپلودشدهٔ `ai-therapist-frontend(1).zip` مشتق شده؛ **کد JSX/CSS، متن‌های نمایشی، ترتیب صفحات، تصاویر و ظاهر آن بدون تغییر** باقی مانده‌اند. فایل‌های توسعهٔ Agent، OpenAPI Snapshot و ابزار بررسی استاتیک اضافه شده‌اند.

## اجرا
```bash
npm ci
npm run dev
```
فرانت: `http://127.0.0.1:5173`؛ پروکسی Vite مسیر `/api/v1/*` را به بک‌اند محلی `http://127.0.0.1:8080` می‌فرستد. صرفاً وجود پروکسی به معنی متصل‌بودن صفحات نیست.

## قبل از توسعه
1. `AGENTS.md` سپس `FRONTEND_DESIGN_SPEC.md`، `MVP_SCOPE.md` و `REPO_MAP.md`.
2. `docs/integration/UI_BACKEND_GAP_MATRIX.md` و `docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md`.
3. مهارت مرتبط در `.agents/skills/`؛ تست `npm run check:foundation && npm test && npm run build`.

## هشدار
این کد فعلاً UI نمایشی با پاسخ ساختگی و داده‌های موقتی است. با وجود ظاهر کامل، Chat/History/Insight/Privacy/Voice/Questionnaire به Backend واقعی متصل نیستند. ادعاهای پژوهشی/درمانی/محرمانگیِ موجود در متن نمایشی اعتبارسنجی نشده‌اند؛ تنها تست داخلی با دادهٔ ساختگی مجاز است.
