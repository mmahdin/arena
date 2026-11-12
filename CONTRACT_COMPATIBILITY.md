# Theia AI Core — Contract Compatibility & Migration Notes v1.3

## ۱. وضعیت این سند

این جدول **نگاشت معنایی طراحی** است، نه گواه compatibility runtime یا مهاجرت اجراشده. مبنای فعلی `DATA_CONTRACTS.md` با ۱۹ گروه منطقی است. اسناد گروه A v0.2، گروه B/C/D v0.1، فاز ۵، و ۳۹ Schema آزمایشی فاز ۶ صرفاً برای تطبیق نام/فیلد نگه داشته شده‌اند؛ هیچ‌کدام به‌تنهایی قرارداد wire مصوب Backend Go نیستند. قبل از اجرا باید `actual Go field → canonical meaning → legacy field` بر اساس کد واقعی کامل شود.

## ۲. نگاشت قراردادهای C01–C19

| Canonical ID / meaning | نام/منشأ نسخه قدیمی | وضعیت و قاعده تبدیل |
|---|---|---|
| C01 `TurnContextSeed` | در Phase6 فقط `common/TurnContext` پیش از Safety مبهم بود؛ در v1.1 تفکیک شد | **جدید در سطح معنا**؛ seed بدون `safety_result`؛ ساخت Backend/S01. |
| C02 `SafetyResult` | Phase5 / `execution/SafetyResult.schema.json` | route/status/constraints حفظ شود؛ نگاشت `safety_only/hold` بدون bypass. |
| C03 `TurnContext` | Group A / `common/TurnContext.schema.json` | **معنای جدید بعد از P01**؛ حامل SafetyResult و snapshot نسخه‌دار. اتصال P01 به C03 ممنوع. |
| C04 `TurnPlan` | Group A `TurnPlan`, `RetrievalRequest`, `MemoryQuery` | intent و retrieval_requests حفظ؛ درخواست ≠ مجوز. |
| C05 `MemoryReadResult` / `StateReadResult` | Group A `MemoryReadResult`, Group D `StateRead` | وضعیت‌های read و state جدا؛ `unavailable/not_attempted` می‌تواند در coverage wrapper باشد؛ بدون دستکاری خام S02. |
| C06 `ContextBundle` | Group A `ContextBundle`, `ContextItem`, `MemoryRecord` | source/epistemic/representation/coverage/revision حفظ؛ همه itemها ref معتبر. |
| C07 `CurrentUnderstanding` | Group B `CurrentUnderstanding` | observation/hypothesis/gaps/sufficiency؛ mapping جزییات `for_reflection` و مشابه آن ضروری است. |
| C08 `ContextGapRequest` | Group B output/section supplementary retrieval | اگر wire schema جدا ندارد، adapter مسئول payload معتبر است؛ فقط memory-eligible gaps. |
| C09 `DialogueDecision` | Group B `DialogueDecision` | primary action، goal، thread intent؛ فیلدهای legacy موردنیاز schema با مقدار default صریح/سازگار، نه حذف بی‌خبر. |
| C10 `InterventionPlan` | Group B `InterventionPlan` | eligibility/consent/stage/status جدا؛ selected=null در blocked/defer معتبر. |
| C11 `StateChangeProposal` | Group B `StateChangeProposal` + Group D `StateCommitRequest` | `expected_state_version` **و** `expected_thread_version` برای Thread transition حفظ؛ proposal ≠ event. |
| C12 `ResponseDraft` | Group C `ResponseDraft`, `ResponseCompositionInput` | متن و annotationها + decision/context refs؛ حذف annotation فقط اگر P08 مستقل check متناظر داشته باشد. |
| C13 `ValidationResult` | Group C `ValidationResult`, `ResponseValidationInput` | binding draft+decision+revision و چهار route؛ خطا ≠ approved. |
| C14 `ResponseCommitResult/Event` | Phase5 `ResponseCommitCommand`, `ResponseCommitResult` | user/response IDs جدا؛ `unknown` و delivery separate؛ event را فقط Backend می‌سازد. |
| C15 `MemoryChanges` | Group D `MemoryChanges`, `MemoryCurationInput` | کاندیدهای منبع‌دار، scope/consent/version؛ no direct write. |
| C16 `MemoryWriteResult` / `StateCommitResult` | Group D `MemoryWriteResult`, `StateCommitResult` | outcome/item partial/unknown/already-applied و version حفظ؛ دو مرجع حقیقت را یکی نکن. |
| C17 `KnowledgeSliceResult` | Phase5 `KnowledgeSliceRequest`, `KnowledgeSliceResult`, `KnowledgePackManifest` | pack/version/digest/approval/revoked روشن؛ فقط briefer approved sections. |
| C18 `ModelTaskResult` | Phase5 `ModelTaskRequest`, `ModelTaskResult` | success تکنیکی ≠ approval؛ output contract ref و knowledge refs حفظ. |
| C19 `TraceEvent` | Phase5 `TraceEvent`, `EvaluationResult` | metadata و redaction؛ evaluation نتیجه مستقل دارد؛ trace ≠ consent/event. |

## ۳. Deliberate deltas (نباید بی‌سر‌و‌صدا اعمال شوند)

| Delta | مسئله | قاعده v1.3 | Action برای Agent |
|---|---|---|---|
| Δ01 | TurnContext vs Safety bootstrap | seed قبل از P01، full context بعد | مدل و adapter جدید فقط با تست dependency |
| Δ02 | MemoryRead statuses محدود vs Context coverage گسترده | `unavailable/not_attempted` در wrapper معتبر | وضعیت‌ها را lossless map کن |
| Δ03 | مفصل‌بودن schemaهای B | semantic minimum + حفظ فیلد لازم downstream | field mapping موردی و version bump |
| Δ04 | proposalهای همزمان P05/P06 | reconcile در S01 و CAS در S03 | no last-write-wins؛ collision tests |
| Δ05 | commit پاسخ و تغییر State ترتیبی/اتمیک | atomicity یا تضمین معادل، **مشروط به Backend** | اگر property ندارید، transition حساس gated |
| Δ06 | P07 خوداظهاری درباره شواهد | P08 بررسی مستقل متن/شواهد | test claims بدون annotation نیز |
| Δ07 | `expected_thread_version` جاافتاده در D | برای Thread transition اجباری معنایی | adaptation با recheck واقعی |
| Δ08 | Schemaهای Phase6 آزمایشی | source of truth نیستند | نه replace خام و نه ادعای تست Go |
| Δ09 | مسیر قدیمی «ساخت Backend جدید» | خارج scope | حذف از task plan |
| Δ10 | approved/committed/delivered خلط شده | status + authoritative events مستقل | prohibit false success |
| Δ11 | نقش Knowledge/Model/Trace در registry ۱۶تایی اولیه نبود | C17–C19 اضافه شد | preserve separation |
| Δ12 | task enumها در legacy متفاوت‌اند | معادل‌سازی موردی و حفظ forward compatibility | adapter map + ADR |

## ۴. قالب Gap/Field Mapping برای Backend موجود

| Canonical contract | Semantic field | Existing Go source/field | Mapping rule | Lossy? | Test | Owner / status |
|---|---|---|---|---|---|---|
| مثال: C14 | response_message_id | **UNKNOWN — inspect repository** | شناسه واقعی پیام خروجی | نباید | T6 | Backend owner / OPEN |
| مثال: C03 | scope_version | **UNKNOWN — inspect repository** | نسخه مجوز معتبر | نباید | T5/T7 | Backend owner / OPEN |
| مثال: C11 | expected_thread_version | **UNKNOWN — inspect repository** | compare-and-swap/معادل | نباید | T7 | Backend owner / OPEN |

این جدول **نباید** با فرض نام فیلدهای Go تکمیل شود. اگر Backend فعلأ تضمینی ندارد، `Integration Delta` بزنید و P0 وابسته را باز نگه دارید.

## ۵. قانون تغییر قرارداد

هر تغییرِ معنایی نیاز دارد به: `decision_id`, `affected contracts`, `producer/consumer`, `backward compatibility`, `evidence/test`, `approval role`, `new contract version`. در این بسته هیچ schema قدیمی بی‌اطلاع و بدون نگاشت منسوخ نشده است. `v1.3` شماره **اسناد** است، نه ادعای وجود API/schema در Go.
