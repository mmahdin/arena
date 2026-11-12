# Theia AI Core — راهنمای تحویل به Agent توسعه Go

**این بسته یک *design handoff* است، نه درخواست اجرای خودکار تغییرات Backend.** هدف آن است که Agent توسعه بتواند قابلیت‌های Backend موجود را با نیازهای AI Core تطبیق دهد و فقط با مجوز کاربر وارد کدنویسی شود.

## ۱. ترتیب مطالعه

`README` → `AI_CORE_ARCHITECTURE` → `MODULE_SPECIFICATIONS` → `DATA_CONTRACTS` → `WORKFLOW` → `MEMORY_ARCHITECTURE` → `PSYCHOLOGY_INTEGRATION` → `OPEN_DECISIONS` → `CONTRACT_COMPATIBILITY` → `SCENARIO_ACCEPTANCE`.

## ۲. کار اولیه Agent با **دسترسی به Backend موجود**

1. نقشه واقعی ماژول‌های Go، auth، message store، consent، event/commit/delivery و observability را از **خود سورس** استخراج کن؛ این بسته هیچ نام فایل/پکیج یا storage را فرض نمی‌کند.
2. برای پورت‌های منطقی S01–S06، وضعیت `existing|adapter_needed|missing|unknown` گزارش کن. هیچ ماژولی را صرفاً به‌خاطر نامش به Microservice مستقل تبدیل نکن.
3. جدول `CONTRACT_COMPATIBILITY.md` را با `actual Go field`, version, authorization check, test coverage پر کن؛ اگر mapping lossy است، Delta بنویس.
4. در هر ماژول مشخص کن کدام بخش rule/DB read، کدام LLM task و کدام prompt/knowledge placeholder است؛ تعداد call را **با فرض** انتخاب نکن.
5. برای هر P0 تصمیم باز، پیش‌نیاز، وضعیت owner و حالت bounded/fail-closed را ثبت کن. Agent حق ندارد D5/D7، رضایت، retention یا تشخیص بالینی را به دلخواه تصویب کند.
6. قبل از تغییر کد، یک Implementation Plan در ساختار واقعی پروژه ارائه کن: modified files, interface adapters, risks, rollback, tests, feature flags و وابستگی‌ها.

## ۳. نمونه قالب Integration Delta

```text
ID: GO-DELTA-001
Canonical requirement: C14 separate commit/delivery + idempotent unknown reconciliation
Existing Go capability: UNKNOWN (repository inspection pending)
Missing semantic behavior: UNKNOWN
Affected modules/contracts: S01, S03, C11, C14
Severity: P0 until proven for sensitive State transitions
Choices: use existing transaction/event mechanism OR hold dependent transition
Decision authority: Backend owner + Product/Safety for sensitive effect
Evidence/tests needed: source refs + T6/T7
Status: OPEN
```

**نام‌های خطی بالا نمونه‌اند، نه ادعای بررسی کد فعلی.**

## ۴. مواردی که Agent مجاز نیست خودکار انجام دهد

- ایجاد Backend جدید، انتخاب ناخواسته DB/queue/LLM provider، تغییر معماری Go یا schema migration بدون بررسی و تأیید.
- نوشتن متن «تکنیک درمانی مصوب»، ساخت پروتکل بحران، استفاده از D5/D7 draft به‌عنوان approved، ادعای تأیید تخصصی.
- تلقی M4 به‌عنوان Fact، بازسازی نقل‌قول دقیق از خلاصه، بازکردن scope حافظه، نگهداری داده حساس بدون سیاست داده.
- تأیید commit/delivery/consent/delete از روی خروجی مدل یا trace، و auto-start تمرین صرفاً از روی پیشنهاد.
- ارسال Draft نامعتبر، تکرار commit با نتیجه نامعلوم، و عبور از `version_conflict` با overwrite.

## ۵. معیار آماده‌بودن برای توسعه کنترل‌شده

- قابلیت‌های لازم Backend audit شده‌اند؛ شکاف‌ها و P0های وابسته مشخص‌اند.
- `C01–C19` به interfaceهای واقعی و version fieldها map شده‌اند.
- حداقل شاخه‌های P01، حریم/رضایت و event/commit طراحی و توسط صاحبان سیاست تعیین شده‌اند.
- T1–T12 برای تست قراردادی به taskهای قابل اجرا تبدیل شده‌اند؛ **کیفیت روان‌شناختی** جداگانه با معیار D8 و بررسی متخصص سنجیده می‌شود.
- فعال‌سازی هر قابلیت حساس به approval و feature gate روشن متصل است.

## ۶. کار بعدی پس از تحویل اسناد

این مرحله **اسناد معماری را نهایی‌سازی کرد، نه محصول را**. قدم بعدی برای کاربر: تحویل همین بسته به Agent بررسی Backend Go تا `Integration Assessment` بسازد؛ سپس با تصمیم خود کاربر، بخش‌های قابل‌ساخت برنامه‌ریزی و پیاده‌سازی شوند. این بسته بدون code ownership خارجی یا ابزار deployment خاص است.
