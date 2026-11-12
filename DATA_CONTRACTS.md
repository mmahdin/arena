# Theia AI Core — Data Contracts & Interface Registry

**مرجع قراردادهای معنایی بسته نهایی v1.3 (wire/schema implementation هنوز باز است)**  
این سند **جایگزین ناگهانی** JSON Schemaهای A/B/C/D/Phase6 نیست. حداقل تعهد معنایی را در یک محل جمع می‌کند تا در مرحله انطباق با Backend به قراردادهای wire/schema سازگار با Backend موجود نگاشت شود.

## ۱. دستور خواندن قرارداد

`!` = اجباری، `?` = اختیاری، `*` = شرطی، `[]` = می‌تواند تهی باشد؛ `ref` = شناسه‌ای که باید به یک artifact/رویداد واقعی resolve شود. فیلدهای optional را نباید برای پرکردن فرم با حدس ساخت. **هر فیلد اجباریِ دارای داده ناموجود باید با status/gap صریح یا مسیر fail-closed مدیریت شود، نه مقدار ساختگی.**

**Common Envelope (server-side):** `contract_version!`, `run_id!` (برای artifactهای یک run)، `conversation_id!`, `user_message_id!`, `artifact_id!`, `created_at!` و `producer_module!`. شناسه `subject_ref` و `access_scope_ref/scope_version` در سرور نگه داشته می‌شود و لازم نیست به مدل فاش شود. برای jobهای مستقل حافظه، `curation_job_id` و `source_event_refs` جایگزین binding کامل run می‌شوند. اگر نمونه‌ای در سندهای قبلی این metadata را داخل body داشته، adapter آن را بی‌تغییر حفظ می‌کند.

**Version axisها را یکی نکنیم:** `contract_version` (شکل داده)، `context_revision` (شواهد همان run)، `state_version` (M6)، `thread_version` (M5)، `scope_version` (مجوز)، `knowledge_version/pack_digest` (دانش). تازه‌بودن هرکدام مستقل بررسی می‌شود.

## ۲. Source & evidence / قواعد مرجع

| مفهوم | حداقل فیلد | الزام |
|---|---|---|
| `EvidenceRef` | `kind!`, `ref_id!`، `version?`, `representation?` | `ref_id` باید در مرجع معتبر resolve شود؛ AI حق ساخت منبع جعلی ندارد. |
| `EvidenceItem` | `evidence_id!`, `source_type!`, `content!`, `representation!`, `epistemic_type!`, `record_status!`, `source_refs[]!`, `record_version!` | برای fact شخصی لازم است گوینده/منبع مشخص باشد؛ در صورت legacy missing-source نشانه کمبود منبع ضروری است. |
| `Source Type` | `M1`, `M2`, `M3`, `M4`, `M5`, `M6`, `current_message` | فقط M1 معتبر `verbatim` می‌تواند مبنای exact quote باشد. |
| `Epistemic Type` | `user_reported`, `assistant_authored`, `system_inferred`, `operational` | M4 از `system_inferred` به `user_reported` ارتقا داده نمی‌شود. |
| `Representation` | `verbatim`, `excerpt`, `summary`, `structured` | «summary» نباید به‌عنوان متن کلمه‌به‌کلمه ارائه شود. |
| `Coverage` | `request_id!`, `status!`, `returned_evidence_refs[]`, `has_more?` | هر درخواست retrieval حتی بدون نتیجه یک coverage result لازم دارد. |

`assistant_authored` به‌معنای آن است که تیا قبلاً چیزی نوشته است، **نه** اینکه آن گزاره درباره کاربر صحیح باشد. `operational` فقط با رویداد/State authoritative اعتبار دارد، نه حدس مدل.

### واژگان وضعیت بازیابی

`found`, `partial`, `not_found`, `restricted`, `error`, `unavailable` و `not_attempted` وضعیت‌های معنایی جدا هستند. سرویس قبلی S02 از `found/partial/not_found/restricted/error` پشتیبانی می‌کرد؛ `unavailable/not_attempted` در `ContextBundle` یا wrapper اجرا ثبت می‌شود تا لازم نباشد wire contract خواندن S02 را بدون دلیل تغییر دهیم. `restricted` نباید با عبارت افشاگر وجود رکورد محرمانه به کاربر منتقل شود.

## ۳. رجیستری قراردادهای اصلی

| ID | قرارداد | سازنده معتبر | مصرف‌کننده مستقیم | وضعیت الزام |
|---|---|---|---|---|
| C01 | `TurnContextSeed` | Backend/S01 | P01 | required |
| C02 | `SafetyResult` | P01 | S01/P02/P07/P08 | required |
| C03 | `TurnContext` | S01 | P02/P04/P05/P07 | required after safety |
| C04 | `TurnPlan` | P02 | S01/P03/P04/P05 | required normal |
| C05 | `MemoryReadResult` + `StateReadResult` | S02/S03 | P03/S01 | required per query |
| C06 | `ContextBundle` | P03 | P04/P05/P07/P08 | required normal |
| C07 | `CurrentUnderstanding` | P04 | P05/P06/P09 | required normal |
| C08 | `ContextGapRequest` | P04 | S01→P03 | conditional |
| C09 | `DialogueDecision` | P05 | P06/P07/P08/S01 | required normal |
| C10 | `InterventionPlan` | P06 | P07/P08/S01 | conditional |
| C11 | `StateChangeProposal` | P05/P06 | S01→S03 | conditional |
| C12 | `ResponseDraft` | P07 | P08 | required response |
| C13 | `ValidationResult` | P08 | S01 | required before ordinary commit |
| C14 | `ResponseCommitResult/Event` | Backend authoritative | S01/S03/P09 | required on commit |
| C15 | `MemoryChanges` | P09 | S02 | conditional post-event |
| C16 | `MemoryWriteResult` + `StateCommitResult` | S02/S03 | S01/next run | conditional on write |
| C17 | `KnowledgeSliceResult` | S04 | S01/Pxx/S05 | per task |
| C18 | `ModelTaskResult` | S05 | calling Pxx/S01 | per model call |
| C19 | `TraceEvent` | S01/S06 | S06 | minimal, scoped |

C01–C16 از سند یکپارچه v1.1 به‌دست آمده‌اند؛ C17–C19 نقش‌های فاز ۵ را در رجیستری مشترک قابل مشاهده می‌کنند. این رجیستری ۱۹ ردیف *logical contract* دارد، نه ۱۹ API یا schema اجباری جداگانه.

## ۴. حداقل شکل قراردادی هر artifact

### C01 — `TurnContextSeed`

`run_id!`, `conversation_id!`, `user_message_id!`, `message_text!`, `locale!`, `message_persist_event_ref!`, `access_scope_ref!`, `scope_version!`; `recent_messages_brief?`. **عمداً فاقد `SafetyResult` است.** Backend با اعتبار identity پیام را ثبت کرده؛ S01 حق تولید event جعلی ندارد.

### C02 — `SafetyResult`

`run_id!`, `route!` (`normal|safety_only|hold` به‌عنوان taxonomy حداقلی)، `constraints[]!`, `policy_ref!`, `assessment_status!`; `signal_evidence_refs[]?`, `recheck_needed?`. P01 برای مسیر safe-only باید به بسته پاسخ/پروتکل approved رجوع کند؛ این فایل متن safety protocol را تولید نمی‌کند.

### C03 — `TurnContext`

C01 + `safety_result!` + `recent_messages[]!` (ممکن است خالی) + `state_brief!` (با `state_version`) + `thread_brief?`, `essential_preferences?`. metadata دسترسی در سرور باقی می‌ماند؛ projection برای هر مدل متفاوت است.

### C04 — `TurnPlan`

`run_id!`, `primary_intent!`, `topic_hints[]!`, `reference_hints[]!`, `retrieval_mode!` (`none|targeted|expanded`)، `retrieval_requests[]!`, `unresolved_reference!`, `secondary_intents[]?`. هر `RetrievalRequest` حداقل `request_id`, `purpose`, `sources[]`, `match_mode`, `priority`, `max_items`, `allow_summary` دارد؛ `query_text/reference` شرطی. درخواست منبع، **مجوز** نیست.

### C05 — `MemoryReadResult` / `StateReadResult`

`request_id!`, `status!`, `returned_items[]!`, `read_at!`; `has_more?`, `issue_code?`, `snapshot_version?`, `origin_request_id?`. برای S02 records به M1–M4 و برای S03 snapshot/brief به M5–M6 محدود است؛ مالکیت فیزیکی می‌تواند مشترک باشد ولی این قرارداد نمی‌تواند از مخزن نامربوط بخواند. در wire legacy، نام فیلدها ممکن است `records`, `state_brief`, `thread_briefs`, `thread_details` باشد؛ adapter این تفاوت را پوشش می‌دهد.

### C06 — `ContextBundle`

`run_id!`, `current_message_id!`, `context_revision!`, `state_snapshot_ref!`, `context_items[]!`, `retrieval_coverage[]!`, `retrieval_gaps[]!`; `is_supplemental?`. هر `context_item` باید به رکورد موجود قابل حل باشد و `source_type`, `representation`, `epistemic_type`, `record_status`, `source_refs[]`, `record_version` را حفظ کند. quote دقیق فقط از item مجاز M1.

### C07 — `CurrentUnderstanding`

`understanding_id!`, `run_id!`, `context_revision!`, `state_snapshot_ref!`, `focus!`, `observations[]!`, `hypotheses[]!`, `information_gaps[]!`, `sufficiency_assessment!`, `thread_alignment!`, `retrieval_quality!`; `safety_escalation_hint?`.

- `Observation`: `statement`, `kind`, `source_refs[]`؛ بدون آن ادعای مشاهده معتبر نیست.
- `Hypothesis`: `statement`, `epistemic_status`, `supporting_evidence_refs[]`, `counter_evidence_refs[]`, `limitations[]`؛ حتی با اعتماد بیشتر *fact* نیست.
- `UnderstandingGap`: `gap_id`, `question_or_unknown`, `acquisition_source` (`potential_memory|user_only|mixed|not_needed_now`), `importance`, `retrieval_status`, `resolution_options[]`.
- `sufficiency_assessment`: برحسب `reflection`, `clarification`, `general_guidance`, `intervention_evaluation` (هرکدام sufficient/partial/insufficient). تمام فیلدهای دقیق legacy B در adapter ممکن است اجباری باشند؛ اینجا فقط حداقل معنا تعریف شده است.

### C08 — `ContextGapRequest`

`run_id!`, `origin_understanding_gap_id!`, `reason!`, `requested_sources[]!`, `priority!`, `attempted_request_ids[]!`; `reference?`. فقط `potential_memory` یا بخش قابل‌بازیابی `mixed`، بعد از بررسی مجوز و بودجه S01. یک `user_only` نباید به درخواست جست‌وجوی نامحدود تبدیل شود.

### C09 — `DialogueDecision`

`decision_id!`, `run_id!`, `understanding_ref!` (شناسه + context_revision)، `primary_action!`, `response_goal!`, `response_constraints!`, `thread_intent!`, `focus_gap_ids[]!`, `intervention_request*`, `state_change_proposals[]!`; `secondary_actions[]?`, `decision_rationale?`, `needs_safety_reroute?`.

`primary_action` با P02.intent یکی نیست. فهرست مرجع پیشنهادی گروه B: `social_reply`, `reflect`, `clarify`, `explore`, `provide_insight`, `provide_guidance`, `offer_intervention`, `review_progress`, `summarize`, `respond_exact_recall`, `acknowledge_correction`, `respect_boundary`. مسیر ایمنی P01 از این enum مستقل است. `primary_action=clarify` باید به یک gap شناسایی‌شده متصل باشد.

### C10 — `InterventionPlan`

فقط در صورت نیاز: `plan_id!`, `decision_id!`, `mode!`, `feasibility_status!` (`eligible_to_offer|awaiting_consent|continue_allowed|needs_information|blocked|defer`), `selected_intervention*`, `goal!`, `prerequisite_assessment[]!`, `consent_state!`, `next_intervention_step*`, `blocking_reasons[]!`; `state_change_proposals[]?`, `consent_evidence_refs[]?`, `registry_refs?`.

**تفاوت قطعی:** `eligible_to_offer` ≠ `explicitly_agreed` ≠ `started`. شروع/ادامه حساس، به registry approved، رضایت معتبر با target مشخص، event و State gate نیاز دارد. `blocked/defer` با selected_intervention=null ممکن است معتبر باشد.

### C11 — `StateChangeProposal`

`proposal_id!`, `origin_module!` (`P05|P06`), `target_type!`, `operation!`, `proposed_patch!`, `preconditions[]!`, `commit_gate!`, `evidence_refs[]!`, `status=proposed!`; `thread_id*`, `expected_state_version*`, `expected_thread_version*`, `origin_ref!`. نسخه‌های واقعی در commit server-side مجدداً بررسی می‌شوند؛ پیشنهاد تغییر، رویداد ثبت‌شده نیست.

### C12 — `ResponseDraft`

`draft_id!`, `run_id!`, `decision_ref!` (decision/understanding/context/state)، `response_text!`, `language!`, `primary_action_delivered!`, `questions[]!`, `claims[]!`, `intervention_usage!`; `secondary_moves_used[]?`, `composition_issues[]?`, `intervention_plan_ref?`.

ادعاهای `claims[]` سند حقیقت نیستند، بلکه راهنمای بررسی‌اند. عدم وجود annotation هرگز P08 را از بررسی خود متن معاف نمی‌کند.

### C13 — `ValidationResult`

`validation_id!`, `run_id!`, `draft_id!`, `decision_id!`, `context_revision!`, `status!` (`approved|repairable|requires_redecision|safety_fallback`), `issues[]!`, `checks[]!`; `approved_draft_id*`, `repair_directive*`, `reroute_reason*`.

فقط `approved` برای همان draft+decision+context با State/scope معتبر مجوز درخواست commit است؛ حتی خود آن **commit نیست**. خطای validator نتیجه approved تولید نمی‌کند.

### C14 — `ResponseCommitResult/Event`

`commit_request_id!`, `run_id!`, `source_user_message_id!`, `response_message_id*`, `outcome!` (`committed|rejected|conflict|unknown`), `committed_event_ref*`, `context_revision!`, `decision_id!`, `approved_draft_id!`; `delivery_status?`, `state_outcome_refs[]?`.

فقط Backend رویداد authoritative را صادر می‌کند. `committed` ≠ `delivered`، `unknown` ≠ شکست قطعی. رویداد تحویل جداست. تغییر State حساس وابسته به پاسخ باید در همان commit یا **تضمین معادلِ قابل اثبات** باشد؛ روش دقیق در Backend موجود باز است.

### C15 — `MemoryChanges`

`change_set_id!`, `curation_job_id!`, `source_event_ids[]!`, `candidate_changes[]!`, `no_write_reason*`, `curation_warnings[]?`. هر candidate حداقل: `operation!`, `target_store!` (M2/M3/M4), `content_or_patch!`, `epistemic_type!`, `source_refs[]!`, `sensitivity!`, `consent_requirement!`, `expected_record_version*`, `confidence_or_limitations?`. **این خروجی نوشته نمی‌شود تا S02 آن را بررسی کند.**

### C16 — `MemoryWriteResult` / `StateCommitResult`

`request_id!`, `overall_status!`, `item_or_proposal_results[]!`, `outcome_known!` و برای هر موفقیت `record_or_state_version`, `authoritative_event_ref`. وضعیت‌های مهم: `committed`, `already_applied`, `version_conflict`, `rejected_policy`, `restricted`, `error`, `unknown`. در wire legacy حالت batch و وضعیت‌های جزئی نام‌های بیشتری دارند؛ تفاوت در adapter ثبت شود، نه با معادل‌سازی `unknown` و `error`.

### C17 — `KnowledgeSliceResult`

`status!` (`ready|partial|unavailable|restricted|incompatible|revoked`), `knowledge_refs[]!` با domain/pack/version/section IDs/digest/approval، `effective_constraints[]!`, `missing_requirements[]!`. درخواست S04 باید دارای `module_id`, `task_type`, `locale` و version/scope باشد. هیچ D5/D7 حساس و unapproved مجاز به استفاده فعال نیست.

### C18 — `ModelTaskResult`

`task_id!`, `task_status!`, `parsed_output*`, `model_ref!`, `prompt_or_knowledge_refs[]!`, `output_contract_ref!`, `usage_metrics?`, `error_code?`. `task_status=success` *فقط* موفقیت تکنیکی است، نه تأیید response یا مجوز side effect.

### C19 — `TraceEvent`

`trace_id!`, `run_id!`, `stage_id!`, `event_kind!`, `status!`, `contract_versions[]`, `knowledge_refs[]`, `result_refs[]`, `redaction_level!`; `duration/usage?`. Trace داده غیرمرجع است؛ برای اثبات رضایت یا ثبت واقعی پیام معتبر نیست.

## ۵. پنج ناوردایی اجرایی که هیچ adapter حق شکستن ندارد

1. **Identity:** شناسه پیام کاربر و پاسخ تیا متفاوت است؛ تمام artifactها به run درست متصل‌اند.
2. **Freshness:** P04/P05/P07/P08 باید `context_revision` هماهنگ داشته باشند؛ scope/thread/state پیش از commit حساس recheck می‌شوند.
3. **Evidence/epistemic:** هر ادعای شخصی مبتنی بر رکورد دارای منشأ است؛ فرضیه M4 با گفته کاربر هم‌ارز نیست؛ exact quote فقط M1.
4. **Authority:** consent/permission/commit/delivery/delete توسط مراجع معتبر Backend/S02/S03 اثبات می‌شود؛ متن مدل رویداد معتبر نیست.
5. **Bounded failure:** retrieval اضافی، repair، redecision، retry محدودند؛ محدودیت/خطا واضح می‌ماند و draft ردشده منتشر نمی‌شود.

## ۶. دو نمونه اتصال میان قراردادها (داده ساختگی)

### ۶.۱. ارجاع به گفت‌وگوی قبلی

```json
{
  "turn_plan": {
    "run_id": "r_12",
    "primary_intent": "reference_past",
    "retrieval_mode": "targeted",
    "retrieval_requests": [{"request_id":"q_1","purpose":"resolve_reference","sources":["M1","M3"],"match_mode":"topic","query_text":"social comparison","priority":"essential","max_items":4,"allow_summary":true}]
  },
  "context_bundle": {
    "run_id":"r_12","current_message_id":"u_12","context_revision":1,
    "context_items":[{"evidence_id":"e_1","source_type":"M3","representation":"summary","epistemic_type":"user_reported","record_status":"active","record_version":2,"source_refs":[{"kind":"message","ref_id":"u_2"}]}],
    "retrieval_coverage":[{"request_id":"q_1","status":"found","returned_evidence_refs":["e_1"]}]
  },
  "understanding": {"understanding_id":"cu_12","context_revision":1,"observations":[{"statement":"کاربر قبلا درباره مقایسه صحبت کرده است","source_refs":[{"kind":"context_item","ref_id":"e_1"}]}]},
  "decision": {"decision_id":"d_12","understanding_ref":{"understanding_id":"cu_12","context_revision":1},"primary_action":"clarify","response_goal":"فهمیدن تجربه فعلی","focus_gap_ids":["g_1"]}
}
```

این مثال **برش توضیحی** است، نه JSON payload کامل مطابق wire schemaهای A/B. نکته فقط زنجیره‌ی `q_1 → e_1 → cu_12 → d_12` و حفظ نسخه است. سؤال کاربر برای استناد باید از gap واقعی `g_1` که در نمونه کوتاه نشده تعریف می‌شود؛ اگر در input واقعی این gap وجود نداشته باشد تصمیم باید رد شود.

### ۶.۲. پیشنهاد تمرین بدون موافقت

```json
{
  "intervention_plan": {
    "plan_id":"ip_7","decision_id":"d_7","mode":"evaluate_offer",
    "feasibility_status":"awaiting_consent",
    "selected_intervention":{"intervention_id":"approved_example","registry_version":"v1","stage_id":"offer"},
    "consent_state":"awaiting_explicit_response",
    "next_intervention_step":null,
    "state_change_proposals":[]
  },
  "forbidden_inference":"eligible_to_offer does not mean started"
}
```

این شناسه تکنیک صرفاً نمونه فرضی است؛ محتوای D5 هنوز تأیید نشده و هیچ technique production با آن مجاز نمی‌شود.

## ۷. تفاوت مشخصات حداقلی با Schemaهای قبلی و Migration Guide

- نام‌های `CurrentUnderstanding.sufficiency_assessment.for_reflection` و مشابه آن در Group B جزئیات بیشتری دارند؛ معنای assessment در این سند حفظ شده ولی API ناسازگار جدید تحمیل نمی‌شود.
- `MemoryReadResult.status` هنوز در Group A پنج مقدار دارد؛ `unavailable/not_attempted` در wrapper پوشش ContextBuilder بیان می‌شود.
- `TurnContextSeed` در v1.1 اضافه شد تا دور منطقی P01/TurnContext حذف شود؛ adapter Backend این مفهوم را تأمین می‌کند.
- `ResponseCommitEvent` در Group C و Phase5 فیلدهای جزئی‌تر دارد؛ event معنایی این سند خلاصه‌سازی آن است، نه تغییر رویداد مرجع.
- `ResponseDraft` در Group C annotationهای گسترده دارد؛ P08 همچنان مستقل از آن‌ها باید متن واقعی و شواهد را بررسی کند. حذف هر annotation تنها وقتی مجاز است که check متناظر در P08 حفظ شود.
- `StateCommitResult` legacy از نتیجه هر proposal استفاده می‌کند؛ فشرده‌کردن batch تا جایی مجاز است که failure/partial/unknown و version قابل‌تمایز بماند.

**قانون تکامل:** در `CONTRACT_COMPATIBILITY.md` نگاشت معنایی ۱۹ قرارداد به طراحی‌های قدیمی ثبت شده است. نگاشت نهایی `existing_go_field → canonical_meaning → wire_field` فقط پس از بررسی سورس واقعی Backend ممکن است و همچنان **باز** است. `v1.3` نسخه مستندات است، نه نسخه مستقر wire/schema.

*پایان DATA_CONTRACTS — مرجع معنایی v1.3؛ wire mapping در Backend باز است.*


