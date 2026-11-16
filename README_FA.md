# راهنمای سریع فارسی — Backend Foundation

این ریپو برای **MVP داخلی اپ همراه هوشمند سلامت روان** ساخته شده و ساختارش از معماری ماژولار بک‌اند Yaran Clinic الهام گرفته است. **برای انتشار عمومی، اطلاعات واقعی کاربران یا ادعای خدمات روان‌شناسی آماده نیست.**

## درون ریپو چه داریم؟

- `MVP_SCOPE.md`: سند اسکوپ توافق‌شدهٔ MVP (فارسی).
- `AGENTS.md`: نقطهٔ شروع Agent جدید و ترتیب خواندن فایل‌ها.
- `TECHNICAL_CONSTITUTION.md` و `PRODUCT_CONSTITUTION.md`: قوانین فنی و مرزهای محصول.
- `REPO_MAP.md`: مالکیت ماژول‌ها و مسیر فایل‌ها.
- `.agents/skills/`: شش Skill، شامل `wellbeing-backend-module`، `wellbeing-api-contract`، `wellbeing-database-migration`، `wellbeing-permission-review`، `wellbeing-ai-prompt-safety` و `wellbeing-dev-session`.
- `apps/api/internal/modules/`: ماژول‌های `devsession`، `user`، `conversation` و `ai`.
- `contracts/openapi/openapi.yaml`: قرارداد API.
- `apps/api/db/migrations/`: ساختار دیتابیس SQL Server و سه کاربر کاملاً ساختگی.
- `Dockerfile`، `compose.yaml`، `.env.example` و اسکریپت‌های تست ساختاری.

## راه‌اندازی لوکال

1. `.env.example` را در `.env` کپی کن.
2. برای `DEV_SESSION_SECRET` یک مقدار تصادفی حداقل ۳۲ بایتی تولید کن؛ مقدار نمونه **نباید** استفاده شود.
3. `docker compose up --build -d` را اجرا کن (به اینترنت برای دریافت dependencyها و imageها نیاز دارد).
4. `http://127.0.0.1:8080/health/live` را بررسی کن.
5. `python3 scripts/smoke_api.py` را برای تست داخلی با مدل Mock اجرا کن.

**محدودیت محیط ساخت این فایل:** اعتبارسنجی ساختار و Skillها انجام شده، ولی نصب کتابخانه‌های Go و تست Integration به‌دلیل بسته‌بودن شبکه در محیط تولید آرشیو انجام نشده‌اند. برای جزئیات `VALIDATION_REPORT.md` را بخوان.

## ادامه توسعه با Agent دیگر

از Agent بخواه ابتدا `AGENTS.md` را بخواند، سپس `MVP_SCOPE.md` و Constitutionها را مطالعه کند و بر اساس Skill مربوطه Featureها را پیش ببرد. **نباید** بدون تصمیم جدید OIDC، Memory Engine، Back Office یا مدل‌های روان‌شناختی استنباطی اضافه کند.


### تصمیم دیتابیس

**Microsoft SQL Server 2022** موتور قطعی پروژه است. تمام Adapterها در `adapters/mssql` قرار دارند و `apps/api/go.mod` از `github.com/microsoft/go-mssqldb` استفاده می‌کند. Docker Compose از SQL Server Developer و Migrationهای T-SQL استفاده می‌کند. قبل از اجرای `docker compose up --build -d` مقدار `MSSQL_SA_PASSWORD` و `DEV_SESSION_SECRET` را در `.env` تنظیم کنید. این محیط فقط برای داده‌های ساختگی و تست داخلی است.

## وضعیت راه‌اندازی در ۲۰۲۶-۱۱-۰۲

وابستگی‌ها نصب و قفل شدند؛ SQL Server و API با Docker اجرا هستند. چهار migration، سه کاربر ساختگی، سه گفتگوی نمایشی و شش پیام ساختگی وجود دارد. خطای ترتیب پیام‌ها با migration جدید رفع شد. گزارش تست‌های واقعی در `VALIDATION_REPORT.md` است.

اجرای مجدد بدون حذف داده:

```sh
docker compose up --no-build -d
make smoke
make integration-test
```

اگر نیاز به ساخت مجدد بود و Go proxy خطای 403 داد:

```sh
docker compose build --build-arg GOPROXY=https://goproxy.io,direct
```

برای اجرای native از `make migrate` و `make api` استفاده کنید؛ این دستورها `.env` را امن و بدون اجرای shell می‌خوانند. API native و Docker را هم‌زمان روی پورت 8080 اجرا نکنید.


### Dev Session Task 02 update

The session selector remains internal-only. Migration `0005_dev_sessions.sql` adds revocable SQL Server dev sessions: a browser logout or test-user switch invalidates the old cookie on the server; existing cookies require a new test login after the upgrade. Run migration before starting the updated API. See `docs/adr/0003-revocable-dev-sessions.md` and `VALIDATION_REPORT.md` for tests and limitations.

### تسک ۰۳: پروفایل و Onboarding

مسیر `GET /api/v1/me` وضعیت `onboardingCompletedAt` را برمی‌گرداند؛ برای کاربر تستی جدید `null` است. اولین `PATCH /api/v1/me/profile` معتبر با نام و سبک گفتگو، Onboarding را تکمیل می‌کند. زمان تکمیل در ویرایش‌های بعدی ثابت می‌ماند. ساخت گفتگو یا ارسال پیام قبل از تکمیل با `409 ONBOARDING_REQUIRED` متوقف می‌شود؛ درخواست برای مکالمهٔ کاربر دیگر همچنان `404` است. این مرحله نیاز به Migration جدید یا تغییر فرم درخواست فرانت ندارد. نتایج واقعی تست‌ها را بعد از اعمال Patch ثبت کنید.

### تسک ۰۴: تاریخچه و بازیابی پردازش مکالمه

Migration جدید `0006_turn_recovery_lease.sql` برای هر Turn در حال پردازش یک Lease دو دقیقه‌ای می‌سازد. در صورت قطع ناگهانی پردازش، درخواست تکراری **با همان `clientTurnId` و متن یکسان** ابتدا `409` می‌گیرد و پس از انقضای Lease می‌تواند پردازش را امن ادامه دهد؛ Worker قدیمی اجازهٔ ذخیرهٔ پاسخ یا حذف Lease جدید را ندارد. پاسخ تکمیل‌شده با همان شناسه‌های پیام بازپخش می‌شود. فرم API عوض نشده است. پیش از اعمال Migration، نسخهٔ قبلی API را متوقف کنید؛ بعد Migration را اجرا و سرویس جدید را راه‌اندازی کنید. تست‌های `make test`، `make integration-test` و `make smoke` باید در محیط MSSQL واقعی اجرا شوند.

### تسک ۰۵: اتصال اختیاری مدل واقعی و Context محدود

مدل پیش‌فرض همچنان `mock` است. برای تست **صرفاً داخلی و با دادهٔ ساختگی**، مقادیر `LLM_PROVIDER=openai-compatible`، `MODEL_NAME` و `MODEL_API_KEY` را فقط در تنظیمات سرور قرار دهید و سرویس را restart کنید. کلید هرگز به فرانت ارسال نمی‌شود.

`supportive-v2` سبک‌های گفتگو را با Prompt نسخه‌دار اعمال می‌کند و فقط آخرین پیام‌های همان گفتگو را (با سقف تعداد و مجموع کاراکترها) برای مدل می‌فرستد. Timeout حداکثر ۹۰ ثانیه است؛ پاسخ ناقص، خالی یا بسیار طولانی رد می‌شود و خطای Provider با `503 AI_UNAVAILABLE` برمی‌گردد. تغییر API یا Migration نداریم. آزمایش با Provider واقعی و تأیید ایمنی برای دادهٔ واقعی انجام **نشده** است. جزئیات در `apps/api/internal/modules/ai/README.md` است.

### تسک ۰۶: ایمنی آزمایشی AI و پایداری پاسخ

Prompt جدید `supportive-v3` محدودیت‌های توصیهٔ دارویی، تشخیص، وابستگی به AI و ادعای کمک اضطراری را صریح‌تر می‌کند. الگوهای محدودی از متن فارسی/انگلیسی (با یکسان‌سازی حروف و فاصله‌ها) برای تست ساختگی بررسی می‌شوند و پاسخ مدل پس از لغو درخواست پذیرفته نمی‌شود. قرارداد API و دیتابیس MSSQL عوض نشده‌اند و Migration تازه‌ای نداریم. **این کنترل‌ها ارزیابی بالینی یا تشخیص قابل‌اتکای بحران نیستند.** ماتریس تست و موارد پوشش‌نداده‌شده در `docs/quality/ai-safety-evaluation.md` آمده است. تست‌های Go و MSSQL باید روی محیط محلی اجرا شوند.

### تسک ۰۷ — آزمون پذیرش MVP داخلی

برای آزمایش کامل HTTP روی یک دیتابیس موقت و مجزای MSSQL دستور `make acceptance-http` و برای اجرای همهٔ کنترل‌های بک‌اند دستور `make acceptance-backend` را اجرا کنید. الزامات تست مرورگر واقعی، وضعیت نتایج و محدودیت استفادهٔ صرفاً داخلی در `docs/quality/mvp-acceptance.md` آمده است.
