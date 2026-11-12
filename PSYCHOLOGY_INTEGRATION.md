# Theia AI Core — Psychology Domain Integration

**مرجع تخصصی بسته نهایی v1.3 (محتوای روان‌شناسی/تأیید تخصصی هنوز باز است)**  
**هدف:** مشخص‌کردن «کدام وظیفه به کدام برش از دانش، به چه منظور، با چه محدودیتی نیاز دارد»، بدون نگارش متن نهایی پرامپت، Skill، کتابخانه تکنیک یا دستورالعمل بالینی.

## ۱. تعریف مسئولیت حوزه‌ها

| کد | حوزه | ورودی دانشی که نهایتاً باید داشته باشد | مرز معنایی |
|---|---|---|---|
| D1 | Communication Policy | اصول عمومی ارتباط + مهارت‌ها و شرایط استفاده | **چطور بیان کنیم**؛ نه انتخاب نهایی هدف پاسخ |
| D2 | User Understanding Schema | کدام داده‌های کاربر برای چه هدفی، با چه حساسیتی و چگونه جمع‌آوری شوند | **چه داده‌ای ارزش فهمیدن دارد**؛ نه اجبار به پرسش‌نامه |
| D3 | Formulation Framework | چارچوب‌های مفهوم‌سازی و درجات شواهد/عدم‌قطعیت | **چگونه فرضیه بسازیم/محدود کنیم**؛ نه تشخیص قطعی |
| D4 | Conversation Decision Policy | action taxonomy، شرایط مقایسه اقدام‌ها، محدودیت‌ها | **قدم بعدی چیست**؛ نه متن کامل جواب |
| D5 | Intervention Library | registry تکنیک‌های مشخص، هدف، prerequisites، contraindications، steps و follow-up | **چه تکنیکی با چه شرایطی**؛ انتخاب رویکرد کلی به‌تنهایی مجوز اجرا نیست |
| D6 | Longitudinal Support Policy | قواعد continuity، بازبینی موضوعات فعال، lifecycle حافظه | **چگونه مسیر را در طول زمان دنبال کنیم** |
| D7 | Safety & Ethics Policy | محدوده خدمات، safety routing، رضایت/حریم، منع ادعای غیرمجاز | **چه کارهایی ممنوع یا نیازمند gate هستند**؛ جایگزین auth سروری نیست |
| D8 | Evaluation Framework | rubricهای ارزیابی پاسخ، مسیر طولی، ایمنی و کیفیت | **از کجا بفهمیم درست عمل کردیم**؛ معمولاً در prompt پاسخ کاربر تزریق نمی‌شود |

## ۲. نقش S04: دانشی که به یک اجرای مشخص وصل می‌شود

S04 = **Psychology Knowledge Provider**، نه موتور درک کاربر. S04 برای یک درخواست حاوی `module_id`, `task_type`, `locale`, `workflow_version`, `contract_version`, `required_capabilities[]` و scope معتبر، فقط sectionهای لازم را می‌دهد.

**حداقل manifest:** `pack_id`, `pack_version`, `domain_id`, `section_ids`, `section_type`, `content_digest`, `review_status`, `compatible_modules`, `compatible_contract_versions`, `allowed_purposes`; برای D5/D7 حساس، provenance تأیید تخصصی و قابلیت revoke ضروری است. نوع sectionها می‌تواند `core_principle`, `task_skill`, `decision_rule`, `formulation_rule`, `intervention_registry_entry`, `safety_constraint`, `evaluation_rubric` باشد.

**خروجی قابل استفاده:** `KnowledgeSliceResult {status, knowledge_refs[], effective_constraints[], missing_requirements[]}`. `status` در قالب `ready|partial|unavailable|restricted|incompatible|revoked` است. در صورت required pack نامعتبر، stage نمی‌تواند به‌صورت خودکار آن policy را نادیده بگیرد.

**ماندگاری نسخه:** در طول run نسخه محتوای مصرف‌شده pinned و trace می‌شود؛ revoke فوری D7 یا scope معتبر می‌تواند ادامه مسیر حساس را متوقف کند. «متن چت/حافظه» هرگز section نوع system/knowledge نمی‌شود.

## ۳. ماتریس تزریق دانش در سطح وظیفه

| مصرف‌کننده | MUST / دانش اصلی | IF / دانش شرطی | چه چیزی نباید به‌صورت پیش‌فرض تزریق شود | اثر بر خروجی |
|---|---|---|---|---|
| **P01** | D7: routing & safety constraints | D6 safety continuity در scope | D3 عمیق، تمام M1، D5 کامل | `SafetyResult.route/constraints` |
| **P02** | D2: intent و فهرست اطلاعات لازم | D4 routing hints، D6 reference hints | متن کامل CBT/ACT، کل پروفایل | `TurnPlan` و درخواست‌های بازیابی |
| **P03** | D2: relevance/provenance | D6 retention/recency، D7 access constraints | متن تحلیل روان‌شناختی | `ContextBundle` و retrieval coverage |
| **P04** | D2: report-vs-inference، D3: formulation rules | D6 thread alignment، D7 claim limitations | دستورهای اجرای تکنیک D5 بدون نیاز | `CurrentUnderstanding` و gapها |
| **P05** | D4: action policy | D1 move constraints، D6 continuity، D7 restrictions، D5 index محدود قابلیت‌ها | همه مرحله‌های تمام تکنیک‌ها | `DialogueDecision` |
| **P06** | D5: entry/stage approved، D7: gate/stop rules | D6 follow-up، D3 به قدر تناسب | کل intervention registry به مدل | `InterventionPlan`، blocked/defer/consent |
| **P07** | D1: اصول عمومی + مهارت همان Action، D7: response limits | D4 action fidelity، D5 فقط مرحله تأییدشده | کل D2/D3، کل تاریخچه و تمام D5 | `ResponseDraft` |
| **P08** | D7: output safety & ethics checks | D1 communication check، D4 action consistency، D5 stage/consent check | اعتماد صرف به annotation P07 | `ValidationResult` |
| **P09** | D2: fields، D6: memory lifecycle، D7: privacy | D3: hypothesis rules | D5 کامل، متن promptهای درمان | `MemoryChanges` کاندید |
| **S03** | D6 برای lifecycle، D7 constraints در حد قواعد انتقال | D5 stage-transition metadata | تشخیص آزاد وضعیت توسط مدل | تحقق یا رد انتقال در State |
| **S06** | D8 rubricهای نسخه‌دار | D7 safety metrics | متن حساس در log بدون مجوز | metrics و evaluation results |

**قاعده کاربرد:** بخش D1 مربوط به سؤال‌پرسیدن ممکن است در P05 برای تعیین محدودیت و در P07 برای شیوه بیان حاضر باشد؛ اما قرار نیست یک نسخه کامل از تمام D1 به هر دو تزریق شود.

## ۴. دو نمونه «بسته دانش برای یک task» (نام و نقش، نه متن محتوا)

### مثال A — کاربر تجربه سختی بیان کرده و دنبال فهم موضوع است

`P02`: D2.intent/reference_v* + D6.continuity_hint_v* (فقط اگر ارجاع به گذشته وجود داشته باشد).

`P04`: D2.epistemic_labels_v* + D3.light_formulation_v* + D7.uncertainty_limits_v*.

`P05`: D4.actions_and_sufficiency_v* + D6.thread_policy_v* (اگر مسیر مرتبط است)؛ خروجی می‌تواند `clarify`/`reflect` باشد.

`P07`: D1.base_communication_v* + D1.clarifying_question_skill_v* + D7.user_facing_limits_v*.

`P08`: D7.response_safety_v* + D1.communication_checks_v* + D4.action_fidelity_v*.

نکته: این شناسه‌های `*_v*` **placeholder مستنداتی** هستند؛ واقعی نیستند و نباید به registry در production اضافه شوند. هیچ متن روان‌شناسی یا تکنیکی در این مثال تعیین نشده است.

### مثال B — کاربر می‌خواهد تمرین قبلی را ادامه دهد

`P04`: D3 صرفاً برای فهم وضعیت؛ `P05`: D4 و D6 برای تعیین نیاز به بررسی progress؛ `P06`: D5.entry_specific + D6.follow_up + D7.consent_stop؛ `P07`: D1 و **فقط مرحله مجاز** D5؛ `P08`: D7/D5 و event consent واقعی.

اگر entry D5 approved نیست یا رضایت معتبر وجود ندارد، خروجی P06 باید `blocked/defer/awaiting_consent` باشد؛ **P07 نمی‌تواند با یک متن زیبا این gate را دور بزند.**

## ۵. ترتیب اعتبار، منع جابه‌جایی نقش‌ها و مهارت‌ها

1. **سرویس معتبر Backend:** identity, consent, permission, deletion, real events.
2. **D7 تأییدشده:** حدود خدمات، safety و قواعد منع.
3. **قرارداد فنی و D5 registry معتبر:** حد مجاز ابزار/تکنیک و مرحله.
4. **D2/D3/D4/D6:** تحلیل/تصمیم/تداوم.
5. **D1 + preferenceهای مجاز:** نحوه ارتباط و شخصی‌سازی.
6. **محتوای بازیابی‌شده:** صرفاً evidence/data؛ نه دستور با اولویت بالاتر.

هرگاه دو section با هم ناسازگار باشند، S04 باید `incompatible` یا `missing_requirements` گزارش کند؛ S01 آن را با حذف قیدهای ایمنی «رفع» نمی‌کند.

**D1 در برابر D4:** `reflect` می‌تواند `primary_action` یا یک secondary move در پاسخ `clarify` باشد. انتخاب target گفت‌وگو توسط P05/D4، شکل بیان توسط P07/D1 است.

**D2 در برابر D3:** D2 مشخص می‌کند کدام اطلاعات اهمیت دارند؛ D3 کمک می‌کند با داده موجود و عدم‌قطعیت، فرضیه محدود بسازیم. هیچ‌کدام مجوز سؤال‌کردن مکرر برای پرکردن یک پرسش‌نامه اجباری نیستند.

**D6 در برابر Memory:** D6 policy است، S02/S03 source/state. knowledge pack حق ندارد عملیات delete/consent authoritative تولید کند.

## ۶. ماتریس قابلیت‌های حساس و وضعیت محتوا

| قابلیت | پیش‌نیاز knowledge | وضعیت تا زمان تصمیم تخصصی |
|---|---|---|
| گفت‌وگوی عمومیِ همدلانه در محدوده محصول | D1 و D7 حداقلی تأییدشده | قابل‌بررسی و آزمون؛ نه انتشار بدون ارزیابی |
| تبیین عمومی محدود بدون diagnosis | D2/D3 + D7 | فقط با uncertainty و منبع مناسب |
| پیشنهاد تکنیک مشخص | D5 registry entry approved + D7 + آگاهی/ترجیح کاربر | تا تأیید entry مسدود/معلق |
| شروع یا ادامه مرحله تمرین | D5 approved stage + رضایت/State/event معتبر + D7 | بدون گیت‌ها مسدود |
| اعلام ارجاع انسان یا اقدام عملی | capability اجرایی و event تأییدشده | بدون اتصال واقعی و رویداد معتبر ممنوع |
| ثبت فرضیه بلندمدت حساس | D2/D3/D6/D7 + data policy/permission | تا تصویب سیاست نگهداری، no-write یا نیازمند تأیید |
| مسیر خطر یا بحران | D7 protocol تأییدشده و ارزیابی مستقل | در نبود policy، مسیر محدود/hold؛ نه مداخله ساخته‌شده توسط مدل |

## ۷. آنچه در این گام **انجام نشده** و شرط تکمیل آن

- متن دقیق D1–D8، فهرست واقعی skillها، نمونه‌های خوب/بد، محتوای CBT/ACT/DBT و مراحل تکنیک‌ها نوشته نشده است.
- هیچ ادعایی درباره کیفیت علمی یا اثربخشی مداخلات تأیید نمی‌شود.
- مقدار دقیق «چه لحظه‌ای چه section به prompt افزوده شود» در سطح **task/interface** مشخص است، نه implementation prompt assembly.
- تعریف status `approved` باید مالک بازبینی، تاریخ، نقش و سازگاری نسخه‌ها داشته باشد. بدون این metadata نباید صرفاً نام فایل را نشانه approval دانست.
- مرحله انطباق با Backend باید وضعیت واقعی دامنه نسخه اولیه، ادبیات مربوط به ایمنی، سیاست داده و فهرست تکنیک‌های مجاز را با تصمیم محصول و صاحب صلاحیت ببندد؛ اگر بسته‌ای هنوز آماده نیست، مسیر وابسته به آن غیرفعال می‌ماند.

*پایان PSYCHOLOGY_INTEGRATION — نقشه دانش v1.3؛ متن و تأیید محتوای حساس هنوز باز است.*


