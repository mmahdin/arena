# Theia AI Core — Workflow & Orchestration v1.3

**مالک بخش:** S01 در سطح AI Core؛ Backend موجود مالک رویدادهای authoritative و تحویل است. این سند semantics اجرا را مشخص می‌کند، نه روش نوشتن کد Go.

## ۱. سه جریان مستقل

| جریان | آغازگر | هدف | مالک side effects |
|---|---|---|---|
| A. پاسخ عادی/ایمن | رویداد پیام ثبت‌شده کاربر | ساخت پاسخ معتبر و ثبت نتیجه | Backend + S03 برای transitions مجاز |
| B. کنترل کاربر | درخواست واقعی stop/revoke/delete/correction | تغییر scope/State، توقف و invalidation | Backend و S02/S03 معتبر؛ بدون انتظار برای P09 |
| C. استخراج حافظه | رویداد مجاز و committed | پیشنهاد و بررسی تغییر M2–M4 | P09 proposal؛ S02 commit کنترل‌شده |

## ۲. مسیر A — یک نوبت پاسخ

```mermaid
sequenceDiagram
  participant App as Existing Backend
  participant O as S01
  participant P as P01
  participant T as P02/P03
  participant R as P04/P05/P06?
  participant W as P07/P08
  participant M as P09
  App->>App: Persist authenticated user message
  App->>O: TurnContextSeed C01 + authoritative event
  O->>P: Safety assessment + approved D7
  P-->>O: C02 route and constraints
  alt normal
    O->>T: Plan and retrieve within scope
    T-->>O: C04 + C06 with context_revision
    O->>R: Understand, choose action, optional intervention
    R-->>O: C07 + C09 + optional C10/C11
    O->>W: Compose draft then independent validation
    W-->>O: C12 + C13
  else safety_only or hold
    O->>O: Approved safe path or hold (no improvised crisis protocol)
  end
  O->>App: Recheck current scope/state and request commit
  App-->>O: C14 committed / rejected / conflict / unknown
  opt committed authoritative event and permitted retention
    App->>M: Event for post-turn curation
    M-->>App: C15 candidate changes -> S02 policy/version checks
  end
  App->>App: Deliver via existing channel; separate delivery event
```

**الزامات:**

1. C01 قبل از Safety ساخته می‌شود؛ `TurnContext` کامل C03 **پس از** C02 ایجاد می‌شود تا وابستگی دوری نداشته باشیم.
2. S01 برای هر stage فقط payload لازم و نسخه‌دار را می‌دهد؛ دسترسی آزاد مدل به همه حافظه ممنوع است.
3. P03 حتی بدون retrieval بلندمدت C06 کمینه می‌سازد؛ `not_attempted` با `not_found` برابر نیست.
4. P04 ممکن است C08 درخواست بازیابی تکمیلی محدود بدهد؛ S01 سقف و scope را کنترل و context_revision را زیاد می‌کند؛ artifactهای وابسته قبلی invalid هستند.
5. P05 action و intent موضوع فعال را مشخص می‌کند؛ P06 فقط وقتی درخواست یا Thread معتبر نیاز دارد فعال می‌شود.
6. P07 متن می‌نویسد؛ P08 مستقل متن، evidence، consent و stage را بررسی می‌کند. `repairable` فقط repair محدود؛ `requires_redecision` بازگشت محدود به P05 است.
7. پیش از commit، scope_version، thread_version، state_version و اعتبار knowledge حساس recheck می‌شوند. Approval P08 مجوز *درخواست commit* است، نه خود commit.
8. commit نامعلوم با idempotency key reconcile می‌شود؛ تکرار بی‌قید commit ممنوع. Delivery رویدادی جداست.
9. P09 وابسته به رخداد معتبر پس از نوبت است و نمی‌تواند یک Draft ردشده یا یک event خیالی را مبنا قرار دهد.

## ۳. جدول Routing

| وضعیت | شرط | رفتار کنترل‌کننده |
|---|---|---|
| `normal` | P01 و D7 معتبر، scope لازم موجود | مسیر P02→…→P08 |
| `safety_only` | P01 نیاز به مسیر محدود تشخیص دهد | فقط مسیر پاسخ ایمنی از پیش تأییدشده + validator/commit متناسب |
| `hold` | Safety policy حیاتی ناموجود/نامعتبر یا مسیر مجاز نامشخص | عدم تولید/ارسال پاسخ حساسِ حدسی؛ هندلینگ از پیش تعریف‌شده محصول |
| `minimal` | P02 درخواستی برای retrieval ندارد | ContextBundle پایه بدون جست‌وجوی بلندمدت |
| `targeted` | reference/topic مرتبط وجود دارد | جست‌وجوی scoped و محدود |
| `expanded` | درخواست مشخص با دامنه معنایی گسترده‌تر | همچنان با سقف/مجوز، نه full-history dump |
| `intervention` | P05 درخواست معتبر داده | P06 با D5/D7 و consent/state checks |
| `blocked/defer` | P06 entry/consent/prerequisite نامعتبر | بازتصمیم/پاسخ محدود؛ بدون auto-start |

`Fast/Standard/Extended` فقط برچسب‌های توضیحی برای همین مسیر هستند؛ مقدار عددی سقف‌ها تا بنچمارک باز است.

## ۴. مدیریت data/control priority

درخواست صریح کاربر برای توقف تمرین، لغو رضایت، حذف حافظه یا اعتراض به یک برداشت باید **از کانال معتبر Backend و پیش از ادامه عملیات وابسته به داده** منعکس شود. متن LLM به‌تنهایی سند revoke یا deletion success نیست. در زمان revoke:

- اجرای جاریِ وابسته به scope قدیمی متوقف/بی‌اعتبار یا با scope جدید بازبرنامه‌ریزی شود.
- S02 داده/مشتقات و indexها را براساس قرارداد حذف/اصلاح، نامعتبر یا حذف کند؛ S03 وضعیتی را که رضایتش باطل شده بازبینی کند.
- trace نیز از نگهداری محتوای حساس خارج از مجوز جلوگیری کند.
- پاسخ به کاربر درباره نتیجه حذف/توقف فقط بر اساس رویداد authoritative باشد؛ عملیات pending/unknown شفاف است.

## ۵. چرخه Supplementary Retrieval و تصمیم مجدد

```text
P03(Context r) → P04(gap potential_memory)
→ S01(check scope + budget + duplication)
→ P03(Context r+1) → P04(new understanding)
→ P05(new decision) → P07(new draft) → P08(new validation)
```

- اگر gap از نوع `user_only` است، به بازیابی حافظه نامحدود نرود؛ P05 می‌تواند با هدف مناسب سؤال بپرسد.
- اگر retrieval دوباره `restricted/error` بود، سیستم باید محدودیتش را حفظ و اقدام کم‌خطرتر انتخاب کند.
- هیچ خروجی P05/P07/P08 نباید با نسخه قدیمی پس از refresh شواهد مصرف شود.

## ۶. Commit, State و Delivery

| رخداد/وضعیت | صاحب حقیقت | آیا معادل «ارسال شد» است؟ | شرط |
|---|---|---|---|
| Draft تولید شد | P07 | خیر | C12 |
| Draft تأیید شد | P08 | خیر | C13 برای همان نسخه و scope |
| پاسخ ثبت شد | Backend message store | خیر | C14 با `committed` و `response_message_id` واقعی |
| تغییر State وابسته ثبت شد | S03/Backend authoritative | خیر | transition معتبر + versions + consent/event |
| تحویل ثبت شد | Backend delivery | گزارش delivery، نه read | رویداد delivery مستقل |
| Memory بلندمدت تغییر کرد | S02 | خیر | C15 فقط کاندید، سپس C16 نتیجه معتبر |

اگر یک transition State **به ثبت همین پاسخ وابسته است** (مثلاً مرحله offer ارسال‌شده)، باید با response commit **اتمیک یا با تضمین معادل قابل اثبات** هماهنگ باشد؛ اگر Backend فعلی آن خواص را ندارد، آن transition حساس hold می‌ماند. State مستقل از پاسخ که بر اساس رویداد معتبر کاربر تغییر می‌کند، می‌تواند در مسیر کنترل مستقل اعمال شود.

**Unknown outcome:** اگر timeout هنگام ثبت رخ داد، `unknown` یعنی نتیجه مشخص نیست؛ جست‌وجوی نتیجه با شناسه درخواست idempotent انجام می‌شود و نباید یک پاسخ یا تمرین دوباره ثبت شود. `committed` نیز تا delivery، خوانده‌شدن را اثبات نمی‌کند.

## ۷. خطا و fallback

| محل | خروجی نامطلوب | پاسخ معماری |
|---|---|---|
| P01 / knowledge D7 | unavailable/revoked/incompatible | safe-only طبق پروتکل مصوب یا hold؛ نه normal حساس |
| P02 / P03 | invalid plan / retrieval error | retry محدود یا کانتکست ناقص علامت‌خورده؛ عدم جعل داده |
| P04 | gap user_only / memory | سؤال هدفمند یا C08 محدود؛ نه تشخیص مطمئن |
| P05/P06 | conflict / blocked | redecision محدود یا defer؛ no silent intervention start |
| P07 | mismatch plan / unsupported claim | return repair issue؛ عدم ارسال مستقیم |
| P08 | failure / requires_redecision | repair/redecision محدود، یا hold/fallback مصوب؛ هرگز implicit approval |
| commit | conflict / unknown | recheck/replan یا idempotent reconcile |
| P09/S02 | curation failure/version conflict | log status و retry مجاز؛ response قبلاً ثبت‌شده rollback نشود |

## ۸. داده‌هایی که trace ثبت می‌کند

`run_id`, stage, outcome, contract_version, knowledge digest/version, context_revision, model ref, latency/usage و refs لازم. **متن حساس و raw chat به‌صورت پیش‌فرض در log عمومی کپی نمی‌شود.** trace برای اثبات consent یا commit معتبر نیست.

## ۹. Acceptance

مسیرها با **T1–T12** در `SCENARIO_ACCEPTANCE.md` بررسی اسنادی می‌شوند. عدد واقعی timeout/retry، مکانیزم atomicity و قابلیت مسیر ایمنی باید توسط مالک Backend و تأییدکننده ایمنی تعیین شود؛ نبودشان به معنی انجام‌شدن فرض نشود.
