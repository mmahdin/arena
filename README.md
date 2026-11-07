> **Active brand (2026-11-07):** Theia / تیا. The scoped visual identity is documented in `docs/design/THEIA_IDENTITY_2026_11.md`; historical Hamrah handoff notes below are retained for architecture/backend context.

## Connected chat/history (Task 03)

Apply this incremental change only **after Task 02 v2 synthetic phone login**. Chat and History now use the existing Go/MSSQL JSON conversation endpoints; backend updates/migrations are not required for this slice. The original Hamrah CSS and artwork remain the UI authority. See `VALIDATION_REPORT.md`.

Locally: `npm ci && npm test && npm run build && npm run dev`. With a real local Go/MSSQL Mock AI backend and Playwright available: `npm run test:e2e:chat`. No intercepted HTTP APIs in that test. Synthetic-only: no real phone numbers, private health data, or clinical validation.

---

# همراه

یک رابط موبایل‌محور فارسی و راست‌به‌چپ، بازسازی‌شده از توالی ۳۸ تصویر مرجع. رابط با React و Vite ساخته شده و هیچ بک‌اند یا اتصال هوش مصنوعی واقعی ندارد.

## اجرا

```bash
npm install
npm run dev
```

برای ساخت نسخه‌ی نهایی:

```bash
npm run build
npm run preview
```

## ساختار

- `src/screens/Onboarding.jsx`: خوش‌آمد، اعتماد و پژوهش، پرسش‌های چند/تک‌انتخابی، سن، صدا و سبک گفت‌وگو.
- `src/screens/Main.jsx`: خانه، پرسش‌نامه، تاریخچه، جزئیات روز، منو، گفت‌وگو و بینش.
- `src/components/UI.jsx`: اجزای مشترک و تصویرسازی‌های برداری.
- `src/data.js`: متن‌ها، گزینه‌ها و داده‌ی نمایشی گفت‌وگو.
- `src/styles.css`: توکن‌های رنگ، تایپوگرافی، چیدمان و حالت‌های واکنش‌گرا.
- `public/assets`: فقط تصویرسازی‌ها و عکس‌های جداشده از مراجع؛ هیچ اسکرین‌شاتی به‌عنوان صفحه استفاده نشده است.

فونت‌های Estedad، Vazirmatn و Noto Naskh Arabic به‌صورت محلی بارگذاری می‌شوند. مسیرها در hash مرورگر نگهداری می‌شوند و دکمه‌ی بازگشت مرورگر نیز کار می‌کند.

## تعاملات

- پاسخ‌های تک/چندانتخابی، محدودیت سه انتخاب، انتخاب سن با لمس یا اسکرول
- کاروسل صدا و سبک گفت‌وگو با نقاط و ژست کشیدن
- منوی کناری، تاریخچه و جزئیات روز
- ارسال پیام و دریافت پاسخ نمایشی، پیشنهادهای آماده و حالت ورودی صوتی
- منوی پیام با لمس طولانی یا کلیک راست: کپی، ذخیره، خواندن، بازتولید و گزارش
- بازخورد بینش و ذخیره‌ی وضعیت پاسخ‌ها در طول نشست فعلی

## Development handoff (new Hamrah UI baseline)

This exact JSX/CSS UI is the product design source of truth. Start at `AGENTS.md`, `FRONTEND_DESIGN_SPEC.md`, `docs/integration/UI_BACKEND_GAP_MATRIX.md`, and `docs/roadmap/BACKEND_FRONTEND_INTEGRATION_PLAN.md`. `npm run check:foundation` verifies original source byte preservation; `npm test` runs the Node handoff checks. A localhost-only Vite proxy has been added for future `/api/v1` integration, but screens **still use demo state and mock chat**. This is internal synthetic testing only; see `PRODUCT_CONSTITUTION.md` before connecting any screen.

### Hamrah synthetic phone-shaped login v2

The previous seeded-user selector is rejected. Enter a **fictional** `0900000XXXX` test identifier. The Go backend `POST /api/v1/dev/identity-session` creates or resumes that synthetic profile with a revocable HttpOnly session cookie, without OTP/SMS. The backend migration `0007` must be applied first. See `docs/integration/PHONE_IDENTITY_V2.md`. Never use actual phone numbers or real sensitive chats. Run `npm test` and, when local services are ready, `npm run test:e2e:session`.
