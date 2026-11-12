# Theia AI Core — Decisions, Compatibility Deltas & Risks

**مرجع تخصصی بسته نهایی v1.3 (محتوای روان‌شناسی/تأیید تخصصی هنوز باز است)**  
این فایل موضوعات نیازمند تصمیم را از الزام‌های پذیرفته‌شده جدا می‌کند؛ تا ایجنت Go مجبور نشود از روی ابهام یک سیاست محصول یا روان‌شناسی اختراع کند.

## ۱. برچسب وضعیت

- `LOCKED-LOGICAL`: در معماری منطقی مبنا قرار گرفته؛ تغییر فقط با ADR ثبت‌شده.
- `PROPOSED`: جهت پیشنهادی که باید در مرحله انطباق با Backend تصمیم یا ارزیابی شود.
- `OPEN`: اطلاعات/تأیید لازم هنوز موجود نیست.
- `GATED`: تا تعیین/تأیید، قابلیت وابسته نباید فعال شود.

## ۲. تصمیم‌های پایه که فعلاً مبنا هستند

| ID | تصمیم منطقی | چرا |
|---|---|---|
| L01 | P01–P09 **۹ مسئولیت** دارند، نه الزاماً ۹ API call | قابلیت ردیابی، امکان ادغام فیزیکی بدون مخدوش‌شدن interface |
| L02 | S01 کنترل branch و lifecycle؛ ماژول‌ها پیشنهاد/ artifact می‌دهند | جلوگیری از side effect آزاد و نامحدود مدل |
| L03 | M1 canonical متعلق به Backend؛ M2–M4 متعلق به S02؛ M5–M6 متعلق به S03 | جلوگیری از چند مرجع متناقض |
| L04 | `TurnContextSeed` پیش از P01 و `TurnContext` بعد از P01 | رفع وابستگی دوری در طراحی A |
| L05 | `ContextBundle` و P04/P05/P07/P08 به `context_revision` همسان متکی‌اند | تصمیم روی شواهد stale رد شود |
| L06 | «گپ بازیابی» و «گپ فهم/سؤال از کاربر» مستقل | جلوگیری از retrieval غیرضروری یا سؤال بیجا |
| L07 | User-report / Inference / Operational-event / Knowledge چهار مرتبه ادعای جدا دارند | امنیت استدلال و حافظه |
| L08 | P05 Action/هدف، P06 eligibility/stage، P07 متن، P08 اعتبارسنجی | جلوگیری از مالکیت مبهم تصمیم/مداخله |
| L09 | proposal تغییر State یا حافظه هرگز خودِ event/commit نیست | جلوگیری از تمرین فعال بدون رضایت یا write ناخواسته |
| L10 | دریافت دستور delete/revoke/stop مسیر priority مستقل دارد | نباید منتظر P09 یا موفقیت LLM باشد |
| L11 | response drafted/approved/committed/delivered متمایز هستند | صداقت در مورد عملیات واقعاً انجام‌شده |
| L12 | تعهدات معماری نباید Backend/DB/Queue جدید را به Go تحمیل کنند | سازگاری با پروژه موجود |

## ۳. تصمیم‌های باز با اولویت و پیش‌فرض ایمن

| ID | اولویت | تصمیم باز و علت | چه مدرکی لازم است؟ | تصمیم‌گیرنده از نظر نقش | پیش‌فرض تا تعیین |
|---|---|---|---|---|---|
| O01 | **P0** | **دامنه خدمت:** تیا در V0 چه سطحی از حمایت/تکنیک ارائه می‌کند؟ | دامنه محصول، بازبینی متخصص سلامت روان و سناریوهای مرزی | Product + Clinical/Safety reviewer | مداخله ساختاریافته حساس غیرفعال |
| O02 | **P0** | متن و status واقعی D7 و پاسخ مسیرهای پرخطر | سیاست ایمنی مصوب + سناریوهای ارزیابی | Clinical/Safety reviewer | مسیر محدود/hold؛ عدم ساخت پروتکل بحرانی توسط مدل |
| O03 | **P0** | D5: کدام تکنیک، entry و stage مجاز و تأییدشده است؟ | رجیستری technique با evidence/reviewer/version | Clinical/Safety reviewer | P06: `blocked/defer` |
| O04 | **P0** | مجوز retention و پروفایل‌سازی M2–M4، به‌ویژه M4 | data policy محصول، کنترل‌های consent، قواعد حذف/اعتراض و بررسی حقوقی مناسب | Product/Privacy | عدم نوشتن حافظه حساس یا inference بلندمدت بدون مجوز |
| O05 | **P0** | در Backend Go موجود چه تضمین قابل اتکایی برای response commit و State transition وجود دارد؟ | قرارداد event store/transaction یا تضمین معادل و proof failure modes | Backend owner | تغییر حساس thread/تمرین hold تا تضمین معتبر |
| O06 | **P0** | کنترل دسترسی و revoke و حذف مشتقات M1→M3/M4 چگونه enforce می‌شود؟ | scope rev، فهرست مشتقات و adapter معتبر | Backend/Privacy | منع استفاده از داده نامعتبر/لغوشده؛ invalidation تا روشن شدن |
| O07 | **P1** | حداقل قابلیت‌های P08 واقعاً چیست و چه چیزی با مدل/قواعد قابل‌ارزیابی است؟ | rubric مستقل، تست‌کیس‌های adverse، نرخ خطا | Safety/Evaluation | در نبود validator معتبر، approval صادر نمی‌شود |
| O08 | **P1** | حد رسمی schema/wire و تغییر نسخه قراردادهای قدیمی چگونه انجام می‌شود؟ | تطبیق `legacy → canonical → wire` برای C01–C19 | Architecture/Backend | semantic interface گام ۲؛ schemaهای قدیمی بی‌صدا بازنویسی نشوند |
| O09 | **P1** | تعداد API Callها و مدل‌ها از جمله Jev | latency/cost/quality measurement روی چند سناریو | AI Engineering | انتخاب قطعی معلق؛ preserve logical outputs |
| O10 | **P1** | حد retrieval، retry، repair، redecision و timeouts | SLO محصول و آزمون فشار/کیفیت | AI Engineering/Product | همه حلقه‌ها bounded؛ عدد ثابت بدون آزمون قطعی نمی‌شود |
| O11 | **P1** | M4 confidence، وضعیت dispute و انقضای hypothesis | سیاست شناخت بلندمدت و قابلیت اعتراض کاربر | Product/Privacy/Clinical | فرضیه tentative، قابل حذف و عدم تبدیل به fact |
| O12 | **P1** | سطح جزئیات واقعی `ResponseDraft.claims[]` و ممیزی P08 | نمونه پاسخ‌ها و مقایسه هزینه/کیفیت annotation | AI Engineering/Evaluation | حداقل evidence و متن نگه داشته شود؛ P08 مستقل بررسی کند |
| O13 | **P1** | تکنیک‌های consent: کدام اقدام به موافقت صریح نیاز دارد؟ | mapping D5 stage + consent event policy | Product/Clinical/Privacy | شروع مرحله حساس بدون consent معتبر ممنوع |
| O14 | **P1** | D1–D4 و D6: محتوا، version و review status | متن دانش واقعی، مسیر تأیید، owner و benchmark | Psychology Content owner | تا محتوای لازم آماده نیست، رفتار محافظه‌کارانه/محدود |
| O15 | **P2** | بسته‌بندی ابزارهای observability و trace در Go | logs/metrics/redaction فعلی | Backend owner | metadata حداقلی؛ عدم ثبت متن حساس در لاگ عمومی |
| O16 | **P2** | تقسیم‌بندی `Fast/Standard/Extended` واقعاً لازم است یا یک مسیر با short-circuit کافی است؟ | latency/profiling و بنچمارک چند مسیر | AI Engineering | یک workflow منطقی مشترک و شاخه‌های اختیاری |
| O17 | **P2** | آیا P04+P05 یا P02+P04+P05+P07 را در یک Call اجرا کنیم؟ | مقایسه کیفیت artifactهای مستقل و هزینه | AI Engineering/Evaluation | ماژول‌ها منطقی جدا؛ ادغام اجرایی اختیاری |
| O18 | **P2** | آیا دسته‌بندی انواع memory فراتر از M1–M6 لازم است؟ | سناریوهای واقعی که توسط M1–M6 پوشش داده نمی‌شوند | Architecture/Product | فعلاً نوع تازه اضافه نشود |

**P0 = blocker برای فعال‌سازی قابلیت وابسته یا تحویل قطعی**؛ منظور نیست که تمام طراحی اسناد تا حل آن متوقف شود. P1 برای کیفیت و قرارداد final، P2 برای بهینه‌سازی است.

## ۴. Compatibility Deltaها: از اسناد قدیمی به گام دوم

| Delta | مشاهده/تعارض | تصمیم گام دوم | اثر قابل اقدام در مرحله انطباق با Backend |
|---|---|---|---|
| Δ01 | A: `TurnContext` حاوی `safety_result` و در عین حال ورودی P01 فرض شده | `TurnContextSeed` پیش از safety؛ `TurnContext` پس از آن | contract mapping برای Adapter |
| Δ02 | A: `MemoryReadResult` پنج status دارد، ولی نیاز orchestration به `unavailable/not_attempted` هم هست | statusهای اضافی در coverage/wrapper؛ بدون تغییر الزامی S02 | تعیین mapping دقیق error domain |
| Δ03 | A/B: فیلدهای طولانی برای context، understanding و gaps | حداقل معنا در DATA_CONTRACTS؛ حفظ جزئیات لازم در نسخه wire | تهیه جدول legacy→canonical؛ حذف فیلد فقط با اثبات عدم نیاز downstream |
| Δ04 | B: P05 و P06 هر دو proposal می‌دهند | S01 reconcile و S03 validate/commit؛ no last-write-wins | تعریف اولویت و collision tests |
| Δ05 | C در بعضی نمودارها commit→State را متوالی نشان می‌دهد، فاز ۵ خواص هماهنگ را خواسته | شرط atomicity یا ضمانت معادل برای تغییر وابسته به پاسخ | تطبیق با backend موجود؛ اگر ناممکن hold تغییر حساس |
| Δ06 | C: self-annotation P07 برای ممیزی مطرح است | annotation صرفاً کمک؛ independent evidence لازم در P08 | تعیین validator بدون اعتماد به producer |
| Δ07 | D: `StateChangeProposal` فقط expected_state_version داشت و Thread version جداست | thread version در adapter/commit guard اجباری برای تغییر Thread | جلوگیری از stale update |
| Δ08 | فاز ۶ JSON schemaهای ماشینی را منبع نهایی تصور کردن ممکن است | آن‌ها مکمل‌اند؛ semantic contract و constraints مرجع جاری‌اند | schema hardening در زمان تحویل به Go |
| Δ09 | نسخه‌های master قدیمی ساخت Backend تازه را پیشنهاد می‌کنند | این درخواست بیرون از دامنه است | حذف از نقشه راه نهایی |
| Δ10 | واژه‌های `approved`, `committed`, `delivered` گاهی برای مراحل مختلف به کار رفته | statusهای صریح و eventهای مستقل | مانع ادعای عملیات انجام‌نشده |
| Δ11 | `KnowledgeSection` و `ModelTask` در لیست ۱۶ contract اصلی v1.1 نبودند | C17–C19 فقط برای registry کامل نقش‌های S04/S05/S06 اضافه شدند | مشخص‌کردن mapping بدون ایجاد سرویس تازه |
| Δ12 | فازهای قدیمی مقادیر task و enumهای مختلف داشته‌اند | enumهای کامل legacy تا تصویب نهایی حفظ شوند؛ interface مختصر معادل معنایی | test coverage قبل از تغییر wire |

## ۵. معیار بسته‌شدن هر تصمیم

برای هر تصمیم باید ثبت شود: `decision_id`, `options considered`, `selected option`, `rationale`, `source/evidence`, `approver_role`, `affected contracts/modules`, `contract versions`, `reversible?`, `effective_from`. اگر evidence موجود نیست، تصمیم به شکل `OPEN`/`GATED` باقی بماند؛ صرفاً با نوشتن `approved` بسته نمی‌شود.

## ۶. وضعیت پس از تکمیل گام سوم (بسته v1.3)

- **انجام شده در سطح مستندات:** مرجع یکپارچه، شناسنامه ماژول‌ها، مسیر اجرا، حافظه و State، mapping معنایی C01–C19، D1–D8، سناریوهای T1–T12 و دستور تحویل به Agent توسعه در `Theia_AI_Core_Final_Documentation_v1.3` تنظیم شد.
- **هنوز انجام نشده:** بررسی مخزن واقعی Go، mapping واقعی wire fields، انتخاب مدل/API/Jev، نوشتن متن پرامپت‌ها و محتوای مصوب D5/D7، تأیید retention/consent و تست عملی/بالینی.
- **قدم بعدی بیرون از پروژه مستندسازی:** تحویل بسته به Agent برای **Integration Assessment** در Backend موجود؛ هر خلأ یا عدم‌امکان با `Integration Delta` و مسئول/تصمیم مشخص ثبت شود، نه اصلاح ضمنی معماری.
- **معیار خروج:** Developer بداند چه Interface لازم است و چه موضوعی بسته نشده؛ مسئول Product/Privacy/Safety حق تصمیم‌های gated را دارد و agent آن‌ها را اختراع نمی‌کند.

*پایان OPEN_DECISIONS — وضعیت و تصمیم‌های باز v1.3.*
