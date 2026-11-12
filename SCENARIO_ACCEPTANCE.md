# Theia AI Core — سناریوهای پذیرش معماری v1.3

این‌ها **نمونه مصنوعی و معیار پذیرش پیشنهادی** هستند؛ در این گام هیچ تماس واقعی مدل، تست Backend Go، یا بازبینی تخصصی سلامت روان اجرا نشده است. ترتیب خروجی‌ها معنایی است، نه تضمین تعداد فراخوانی LLM.

## قواعد آزمون مشترک

هر run باید هویت و scope معتبر داشته باشد؛ اگر stage غایب شد یا `blocked` بود، نتیجه قابل‌ردیابی باشد. خروجی P04/P05/P07/P08 با `context_revision` همسان، و side effects فقط توسط Backend/S02/S03 ثبت شوند. در متن مثال‌ها، هیچ جمله ساختگی منسوب به کاربر را نباید Fact قطعی نگه داشت.

| ID | ورودی/شرایط نمونه | قراردادها و جریان انتظار | معیار قبولی / خطای ممنوع |
|---|---|---|---|
| **T1** | «سلام» بدون ارجاع | C01→C02 normal→C04 no retrieval→C06 base→C07/C09→C12/C13→C14 | بدون بازیابی بلندمدت بی‌دلیل، فرضیه‌سازی یا تکنیک بی‌نیاز |
| **T2** | «همون حرف هفته پیش» | P02 retrieval M1/M3→C05 coverage→C06→P04/P05 | اگر not_found/error شد، «قبلاً هیچ‌وقت نگفتی» یا نقل‌قول جعلی ممنوع |
| **T3** | «تمرین هفته پیش کمکم نکرد» با Thread معتبر | S03 M5/M6→P04→P05 review_progress→P06 conditional | ثبت feedback و بررسی علت؛ پیشنهاد خودکار تکنیک جدید ممنوع |
| **T4** | کاربر تمرین می‌خواهد، consent ندارد | P05 intervention request→P06 `awaiting_consent`→P07/P08 | پیشنهاد توضیح‌دار ممکن؛ state `started` و اجرای مرحله حساس ممنوع |
| **T5** | «دیگه نمی‌خوام/از این داده استفاده نکن» | priority Backend control→scope/State/M2–M4 invalidation | منتظر P09 یا approval متن پاسخ نمی‌ماند؛ ادعای deletion قبل از event ممنوع |
| **T6** | timeout هنگام ثبت پاسخ | C13 approved→commit `unknown`→lookup by idempotency | `unknown` را rejected/committed فرض نکند؛ پاسخ تکراری ثبت نکند |
| **T7** | دو پیام هم‌زمان درباره یک Thread | concurrent snapshots→C11 expected_version→S03 check | version conflict صریح، بدون last-write-wins و بدون overwrite |
| **T8** | نشانه خطر تازه در P04/P08 | escalation hint→S01→P01 recheck | مسیر normal متوقف؛ تنها پروتکل ایمنی مصوب یا hold |
| **T9** | کاربر: «نه، من از مقایسه نمی‌ترسم؛ فقط قضاوت خانواده‌ام نگرانم می‌کنه» | correction event→M4 dispute/update→new context | hypothesis قدیمی Fact تلقی نشود؛ در پاسخ بعدی بدون بررسی دوباره تکرار نشود |
| **T10** | P07 نقل‌قولی از M3 summary به‌عنوان متن دقیق تولید می‌کند | C12 claims + independent P08→C13 not approved | unsupported exact quote رد/اصلاح شود؛ annotation تولیدکننده سند صحت نیست |
| **T11** | P04 «تاریخچه خاص شاید کمک کند» ولی retrieval `restricted` است | C08 bounded→coverage restricted→P05 limited action | scope گسترش خودکار نیابد؛ وجود رکورد restricted افشا نشود |
| **T12** | P05 تکنیک مشخص پیشنهاد می‌کند ولی D5 approved entry موجود نیست | P06 `blocked/defer`→P05 limited redecision | ایجاد تکنیک از مدل/ادعای شروع ممنوع؛ fallback محدود |

## حداقل artifactهای قابل بررسی برای هر سناریو

1. `TurnContextSeed`, `SafetyResult`, `TurnPlan`, `ContextBundle` با `context_revision` و retrieval coverage (حتی empty).
2. `CurrentUnderstanding` با observations/hypotheses/gaps و status منابع.
3. `DialogueDecision` و در صورت نیاز `InterventionPlan`/`StateChangeProposal` با consent و version references.
4. `ResponseDraft` با متن/ادعا/سؤال و `ValidationResult` بر همان نسخه؛ یا توضیح route ایمن/hold.
5. ثبت/عدم ثبت واقعی پاسخ، State/Memory و delivery با **eventهای جدا**؛ هیچ state ساختگی.
6. `TraceEvent` فقط metadata لازم، نه محتوای حساس در log عادی.

## چگونه با این سناریوها معماری را تغییر دهیم؟

اگر یکی از سناریوها شکست طراحی پیدا کرد: (الف) منبع/مصرف‌کننده ناقص را شناسایی کن؛ (ب) مرز مالکیت را در `MODULE_SPECIFICATIONS.md` اصلاح کن؛ (ج) معنی قرارداد را در `DATA_CONTRACTS.md` تغییر بده و Delta را ثبت کن؛ (د) اثر روی D1–D8 و open decision را بررسی کن؛ (هـ) سناریوی متناظر را به‌روزرسانی کن. رفع شکست با افزودن پنهانی API Call یا permission ساختگی مجاز نیست.
