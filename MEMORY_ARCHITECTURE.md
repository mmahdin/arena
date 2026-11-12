# Theia AI Core — Memory & State Architecture v1.3

## ۱. تفاوت حافظه با State و Context

| منبع | معنی/نمونه | مالک منطقی | خواندن | نوشتن |
|---|---|---|---|---|
| M1 Raw Conversations | متن واقعی پیام کاربر/تیا و timestamp | Backend existing | projection مجاز از S02/Backend | فقط Backend معتبر |
| M2 Facts & Preferences | کاربر گفته دانشجوست، پاسخ کوتاه دوست دارد | S02 | query scoped | کاندید P09 + بررسی S02/permission |
| M3 Episodic | «هفته قبل درباره اختلاف با دوست صحبت شد» به همراه refs | S02 | query scoped | کاندید P09 + بررسی S02 |
| M4 Inferred Hypotheses | «ممکن است از مقایسه‌شدن بترسد» | S02 | query scoped با status/uncertainty | کاندید P09 + بررسی سخت‌گیرانه |
| M5 Active Threads/Plans | تمرین، هدف، مرحله، بازخورد، توقف | S03 | StateRead با version | StateCommit authoritative با preconditions |
| M6 Conversation State | نسخه/وضعیت اجرایی turn و موضوع فعال | S03 | snapshot/brief نسخه‌دار | transitions معتبر S03/Backend |
| Working Context | برش اطلاعاتی برای پاسخ همین turn | P03 output C06 | P04/P05/P07/P08 | منبع حقیقت تازه نیست |

**نکته:** یک جعبه داده مشترک در فیزیک Backend ممکن است M1–M6 را نگه دارد؛ مرز منطقی را با فیزیک دیتابیس اشتباه نگیریم. هیچ دسته جدیدی بدون سناریوی نیازمند آن اضافه نمی‌شود.

## ۲. قواعد معرفتی (Epistemic)

- `user_reported`: کاربر گفته است، نه لزوماً حقیقت تأییدشده بیرونی.
- `assistant_authored`: تیا چیزی نوشته است؛ به‌خودی‌خود گزاره‌ای درباره کاربر نیست.
- `system_inferred`: فرضیه استخراج‌شده از داده و **قابل‌تصحیح/رد**، نه diagnosis.
- `operational`: فقط با event/State معتبر پشتیبانی می‌شود؛ متن مدل یا حافظه توصیفی منبع آن نیست.
- `representation=verbatim|excerpt|summary|structured` و `source_refs` باید حفظ شوند. `summary` هرگز «نقل‌قول دقیق» نیست.

**مثال:** «کاربر گفته از مقایسه‌شدن ناراحت می‌شود» می‌تواند M2 با provenance باشد؛ «کاربر از مقایسه شدن می‌ترسد» M4 tentative است تا با داده و بازبینی اصلاح شود. اگر خود کاربر خلاف آن را گفت، M4 باید با وضعیت disputed/superseded قابل‌ردیابی اصلاح شود و نباید به زور حفظ شود.

## ۳. Memory Read — P02 → P03 → S02/S03

1. P02 بر اساس پیام و brief کوتاه، برای retrieval `request_id`, `purpose`, `sources[]`, `match_mode`, `priority`, `max_items`, `allow_summary` تعیین می‌کند؛ درخواست **مجوز نیست**.
2. S01 access scope و budget را از Backend می‌گیرد/تأیید می‌کند؛ P03 به S02 و S03 query محدود می‌فرستد.
3. هر درخواست حتی بدون نتیجه، coverage دارد؛ حالت‌ها `found`, `partial`, `not_found`, `restricted`, `error`, `unavailable`, `not_attempted` از هم جدا می‌مانند.
4. P03 یک `ContextBundle` دارای `context_revision`, `context_items`, `retrieval_coverage`, `retrieval_gaps`, `state_snapshot_ref` می‌سازد.
5. P04 اگر داده‌ای **احتمالاً در حافظه** باشد درخواست C08 محدود می‌دهد؛ اگر فقط خود کاربر می‌تواند روشن کند، gap را به P05 می‌سپارد. دو نوع gap معادل نیستند.

**Exact recall:** سؤال «هفته پیش دقیقاً چی گفتم؟» باید فقط از M1 `verbatim` مجاز پاسخ داده شود. اگر متن پیدا نشد یا دسترسی محدود بود، تیا متن بازسازی‌شده را به‌عنوان نقل لفظی ارائه نمی‌کند؛ همچنان می‌تواند (در محدوده مجاز) بگوید اطلاعات دقیق در دسترس نیست.

## ۴. Memory Write — P09 → S02

P09 از پیام/پاسخ **رویداددار** و در صورت نیاز شواهد P04، یک `MemoryChanges` کاندید ایجاد می‌کند. هر کاندید حداقل: نوع عملیات، target M2/M3/M4، محتوا یا patch، نوع معرفتی، source refs، حساسیت، مجوز لازم، نسخه هدف و limitations را دارد.

S02 قبل از نوشتن بررسی می‌کند:

- آیا رکورد/مشتقات در محدوده مجاز retention قرار می‌گیرند و رضایت/اجازه معتبر وجود دارد؟
- آیا source واقعاً وجود دارد و نسخه آن هنوز معتبر است؟
- آیا با حافظه قبلی تکراری/متناقض است؟ اصلاح/ادغام/رد با نتیجه صریح انجام می‌شود.
- آیا نوع معرفتی درست حفظ شده و inference در مسیر Fact ارتقا داده نشده است؟
- آیا نسخه رکورد/ردیف و scope برای نوشتن هنوز سازگار هستند؟

`MemoryWriteResult` باید `committed`, `already_applied`, `version_conflict`, `rejected_policy`, `restricted`, `error`, `unknown` را در حد نیاز تفکیک کند. **شکست P09/S02 نمی‌تواند پاسخ قبلاً committed را حذف/rollback کند.** الزام به نوشتن حافظه در هر turn وجود ندارد.

## ۵. State/Threads — P05/P06 → S01 → S03

- P05 `thread_intent` و در صورت نیاز `StateChangeProposal` می‌دهد؛ P06 ممکن است پیشنهاد stage/consent بدهد.
- این proposalها **State نیستند**؛ S01 تعارض‌هایشان را بررسی و به S03 می‌دهد.
- S03 `expected_state_version`, `expected_thread_version`, preconditions، event refs، رضایت و مجازبودن transition را recheck می‌کند.
- `eligible_to_offer` فقط «قابل پیشنهاد» است؛ `explicitly_agreed` رضایت معتبر می‌خواهد؛ `started` فقط پس از رویداد/State transition معتبر است.
- `active_thread` لزوماً درمان نیست؛ ممکن است موضوعی معمولی و هدفی غیرمداخله‌ای باشد.
- نمونه stateهای تمرین (`proposed`, `awaiting_consent`, `active`, `reviewing_feedback`, `paused`, `declined`, `stopped`, `completed`) **taxonomy مفهومی** هستند، نه enum نهایی approved.

## ۶. Correction, Revoke & Delete

درخواست کاربر مبنی بر «این برداشت غلط است»، «این تمرین را نمی‌خواهم»، «این داده را حذف کن» یا «از این اطلاعات استفاده نکن» باید از مسیر معتبر کنترل کاربر عبور کند؛ P09 وظیفه تصمیم‌گیری درباره اجرای فوری آن را ندارد.

- اصلاح فرضیه: برچسب/نسخه و ارجاع به اصلاح کاربر ذخیره یا حافظه نامعتبر می‌شود؛ فرضیه قدیمی نباید برای پاسخ جدید بازیابی شود.
- stop: transition Thread/Intervention توسط S03 و با event معتبر؛ پاسخ تیا نباید بی‌پشتوانه ادعای توقف ثبت‌شده کند.
- revoke: scope جدید بلافاصله بر retrieval و write معتبر اثر می‌گذارد و اجراهای وابسته به نسخه قدیمی invalid می‌شوند.
- deletion: M1 canonical و مشتقات M3/M4، index/embedding، cache و logهایی که policy شامل می‌شود، مطابق عملیات Backend/Privacy حذف یا نامعتبر می‌شوند؛ نتیجه pending/unknown باید از committed متمایز باشد.

**نکته:** این سند سیاست حقوقی retention یا زمان‌بندی دقیق deletion را تعیین نمی‌کند؛ در O04/O06 باز است. وجود مسیر منطقی به معنی امکان فنیِ احرازشده در Go نیست.

## ۷. حداقل مرز API منطقی برای Go agent

| Port | سؤال قرارداد | بدون پیش‌فرض روش استقرار |
|---|---|---|
| `read_memory(query, scope)` | کدام M1–M4 مجاز و منبع‌دار است؟ | ممکن است method داخلی باشد |
| `propose_memory_write(changes)` | چه کاندیدهایی به بررسی S02 رفت؟ | P09 خودش DB را نمی‌نویسد |
| `commit_memory_changes(...)` | چه چیز واقعاً با نسخه/مجوز ثبت شد؟ | event/result واقعی لازم |
| `read_state(scope)` | snapshot کدام version/thread داریم؟ | S03/Backend existing |
| `commit_state_proposals(...)` | کدام transition با event و gate مجاز بود؟ | CAS/transaction/equivalent در Backend |
| `handle_privacy_control(...)` | stop/revoke/delete از چه مرجع معتبر اعمال شد؟ | مسیر مستقل از LLM و P09 |

اسامی بالا صرفاً توضیح **port منطقی** هستند، نه API الزام‌آور Go. معیارهای سناریویی مرتبط: T2, T3, T4, T5, T7, T9.
