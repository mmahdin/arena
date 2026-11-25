package bundle

import (
	"bytes"
	"fmt"
	"io"

	"gopkg.in/yaml.v3"
)

func strictYAML(data []byte, out any) error {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(out); err != nil {
		return err
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return fmt.Errorf("multiple YAML documents not allowed")
		}
		return err
	}
	return nil
}

type Skill struct {
	Schema         string   `yaml:"schema"`
	ID             string   `yaml:"id"`
	Name           string   `yaml:"name"`
	Category       string   `yaml:"category"`
	Descriptor     string   `yaml:"descriptor"`
	Purpose        string   `yaml:"purpose"`
	UseWhen        []string `yaml:"use_when"`
	DoNotUseWhen   []string `yaml:"do_not_use_when"`
	CompatibleWith []string `yaml:"compatible_with"`
	ConflictsWith  []string `yaml:"conflicts_with"`
	Form           struct {
		MaxSentences       int      `yaml:"max_sentences"`
		MayEndWithQuestion bool     `yaml:"may_end_with_question"`
		Avoid              []string `yaml:"avoid"`
	} `yaml:"form"`
	CardText string `yaml:"card_text"`
	Examples struct {
		Good []struct {
			User  string `yaml:"user"`
			Reply string `yaml:"reply"`
		} `yaml:"good"`
		Poor []struct {
			User  string `yaml:"user"`
			Reply string `yaml:"reply"`
			Why   string `yaml:"why"`
		} `yaml:"poor"`
	} `yaml:"examples"`
	Evidence []struct {
		Ref  string `yaml:"ref"`
		Note string `yaml:"note"`
	} `yaml:"evidence"`
	Locales map[string]struct {
		Descriptor string `yaml:"descriptor"`
		CardText   string `yaml:"card_text"`
	} `yaml:"locales"`
}

type Protocol struct {
	Schema     string `yaml:"schema"`
	ID         string `yaml:"id"`
	Version    string `yaml:"version"`
	Name       string `yaml:"name"`
	Family     string `yaml:"family"`
	OneLiner   string `yaml:"one_liner"`
	EstMinutes int    `yaml:"est_minutes"`
	Evidence   []struct {
		Ref   string `yaml:"ref"`
		Level string `yaml:"level"`
	} `yaml:"evidence"`
	FitTags     []string `yaml:"fit_tags"`
	Eligibility struct {
		RequirePhase        []string `yaml:"require_phase"`
		RequireSlotsFilled  []string `yaml:"require_slots_filled"`
		RequireRiskAtMost   string   `yaml:"require_risk_at_most"`
		MinUserReadiness    string   `yaml:"min_user_readiness"`
		ForbidFlags         []string `yaml:"forbid_flags"`
		DeclineCooldownDays int      `yaml:"decline_cooldown_days"`
		MinDaysBetweenRuns  int      `yaml:"min_days_between_runs"`
		MaxRunsPerThread    int      `yaml:"max_runs_per_thread"`
	} `yaml:"eligibility"`
	Steps []ProtocolStep `yaml:"steps"`
	Offer struct {
		WhyTemplates []string `yaml:"why_templates"`
		MustSay      []string `yaml:"must_say"` // legacy intent references
		Intents      []string `yaml:"intents"`
	} `yaml:"offer"`
	StopRules []struct {
		When         string `yaml:"when"`
		Action       string `yaml:"action"`
		Note         string `yaml:"note"`
		FollowUpDays int    `yaml:"follow_up_days"`
	} `yaml:"stop_rules"`
	Completion struct {
		RecordOutcome bool `yaml:"record_outcome"`
		AskRating     bool `yaml:"ask_rating"`
		FollowUpDays  int  `yaml:"follow_up_days"`
		MemoryWrite   struct {
			Kind string   `yaml:"kind"`
			Tags []string `yaml:"tags"`
		} `yaml:"memory_write"`
	} `yaml:"completion"`
	SafetyNotes string `yaml:"safety_notes"`
}
type ProtocolStep struct {
	ID              string   `yaml:"id"`
	Goal            string   `yaml:"goal"`
	Instruction     string   `yaml:"instruction"`
	ExpectedInputs  []string `yaml:"expected_inputs"`
	SuccessCriteria string   `yaml:"success_criteria"`
	UI              struct {
		Widget string `yaml:"widget"`
	} `yaml:"ui"`
	OnSuccess string `yaml:"on_success"`
	OnStuck   struct {
		Adapt string `yaml:"adapt"`
		After int    `yaml:"after"`
		Then  string `yaml:"then"`
	} `yaml:"on_stuck"`
	OnDistress  string `yaml:"on_distress"`
	OnDecline   string `yaml:"on_decline"`
	Adaptations []struct {
		ID          string `yaml:"id"`
		Instruction string `yaml:"instruction"`
	} `yaml:"adaptations"`
}

type TagVocabulary struct {
	Schema string   `yaml:"schema"`
	Values []string `yaml:"values"`
}
type UserInformation struct {
	Schema string `yaml:"schema"`
	Fields []struct {
		ID          string `yaml:"id"`
		Group       string `yaml:"group"`
		Purpose     string `yaml:"purpose"`
		Sensitivity string `yaml:"sensitivity"`
		Collect     struct {
			Proactive          bool   `yaml:"proactive"`
			Condition          string `yaml:"condition"`
			InferFromBehavior  bool   `yaml:"infer_from_behavior"`
			ConfirmBeforeStore bool   `yaml:"confirm_before_store"`
		} `yaml:"collect"`
		UpdateRule string  `yaml:"update_rule"`
		AskCost    float64 `yaml:"ask_cost"`
	} `yaml:"fields"`
}
type ProblemUnderstanding struct {
	Schema string `yaml:"schema"`
	Slots  []struct {
		ID         string             `yaml:"id"`
		Importance map[string]float64 `yaml:"importance"`
		AskCost    float64            `yaml:"ask_cost"`
		AskHint    string             `yaml:"ask_hint"`
	} `yaml:"slots"`
	ReadinessRule struct {
		RequireFilled        []string `yaml:"require_filled"`
		AndAnyOf             []string `yaml:"and_any_of"`
		MinUserTurnsInThread int      `yaml:"min_user_turns_in_thread"`
	} `yaml:"readiness_rule"`
	InformationNeeds struct {
		VMin               float64  `yaml:"v_min"`
		RecentlyAskedHours int      `yaml:"recently_asked_hours"`
		MaxReturned        int      `yaml:"max_returned"`
		SuppressIf         []string `yaml:"suppress_if"`
	} `yaml:"information_needs"`
}
type Formulation struct {
	Schema     string `yaml:"schema"`
	Framework  string `yaml:"framework"`
	SelectWhen struct {
		ProblemTagsAny []string `yaml:"problem_tags_any"`
	} `yaml:"select_when"`
	Fields []struct {
		ID            string   `yaml:"id"`
		AllowedStatus []string `yaml:"allowed_status"`
		MinEvidence   int      `yaml:"min_evidence"`
		MinSessions   int      `yaml:"min_sessions"`
		Wording       string   `yaml:"wording"`
		Voiceable     *bool    `yaml:"voiceable"`
	} `yaml:"fields"`
	Rules     []string `yaml:"rules"`
	Paragraph struct {
		MaxWords         int  `yaml:"max_words"`
		MustMarkInferred bool `yaml:"must_mark_inferred"`
	} `yaml:"paragraph"`
}

type Lexicon struct {
	Schema  string `yaml:"schema"`
	Version string `yaml:"version"`
	Locale  string `yaml:"locale"`
	Rules   []struct {
		ID       string `yaml:"id"`
		Category string `yaml:"category"`
		Level    string `yaml:"level"`
		Match    struct {
			Type    string `yaml:"type"`
			Pattern string `yaml:"pattern"`
		} `yaml:"match"`
		SecondLook bool `yaml:"second_look"`
	} `yaml:"rules"`
}

// ValidatorLexicon is the WP-07 locale rule asset carried by the same
// immutable, clinically reviewed WP-03 Knowledge Bundle as WP-06 safety.
type ValidatorLexicon struct {
	Schema         string              `yaml:"schema"`
	Locale         string              `yaml:"locale"`
	Hedges         []string            `yaml:"hedges"`
	Questions      []string            `yaml:"question_patterns"`
	ClinicalLabels []string            `yaml:"clinical_labels"`
	Patterns       map[string][]string `yaml:"patterns"`
}

type CrisisResources struct {
	Schema               string `yaml:"schema"`
	LastFullVerification string `yaml:"last_full_verification"`
	Locales              map[string][]struct {
		ID         string `yaml:"id"`
		Name       string `yaml:"name"`
		Kind       string `yaml:"kind"`
		Contact    string `yaml:"contact"`
		Hours      string `yaml:"hours"`
		VerifiedOn string `yaml:"verified_on"`
	} `yaml:"locales"`
	FallbackLocale string `yaml:"fallback_locale"`
	StaleAfterDays int    `yaml:"stale_after_days"`
}
type SafetyPolicy struct {
	Schema     string `yaml:"schema"`
	Categories map[string]struct {
		TemplateFamily string `yaml:"template_family"`
		EscalateTo     string `yaml:"escalate_to"`
	} `yaml:"categories"`
	TemplateFamilies map[string]struct {
		RequiredElements []string `yaml:"required_elements"`
		Forbidden        []string `yaml:"forbidden"`
	} `yaml:"template_families"`
	RiskFSM struct {
		Deescalate struct {
			FromHighToElevated struct {
				MinHoursSinceSignal int      `yaml:"min_hours_since_signal"`
				Require             []string `yaml:"require"`
			} `yaml:"from_high_to_elevated"`
			FromElevatedToLow struct {
				MinHoursSinceSignal int      `yaml:"min_hours_since_signal"`
				Require             []string `yaml:"require"`
			} `yaml:"from_elevated_to_low"`
		} `yaml:"deescalate"`
		FollowUpHours int `yaml:"follow_up_hours"`
	} `yaml:"risk_fsm"`
	SecondLook struct {
		Downgrade struct {
			From                string   `yaml:"from"`
			To                  string   `yaml:"to"`
			WhenContextFlagsAny []string `yaml:"when_context_flags_any"`
			NeverWhen           []string `yaml:"never_when"`
		} `yaml:"downgrade"`
	} `yaml:"second_look"`
}
type BoundaryPolicy struct {
	Schema string `yaml:"schema"`
	Modes  []struct {
		ID               string   `yaml:"id"`
		Detect           []string `yaml:"detect"`
		ResponseScriptID string   `yaml:"response_script_id"`
		AllowedContent   []string `yaml:"allowed_content"`
		ForbiddenContent []string `yaml:"forbidden_content"`
	} `yaml:"modes"`
}
type ContinuityPolicy struct {
	Schema          string `yaml:"schema"`
	ResumptionBands struct {
		SameMin    int `yaml:"same_min"`
		ShortHours int `yaml:"short_hours"`
		DayDays    int `yaml:"day_days"`
	} `yaml:"resumption_bands"`
	FollowUpIntervalsDays []int `yaml:"follow_up_intervals_days"`
	StaleAfterDays        int   `yaml:"stale_after_days"`
}
type Fallback struct {
	Schema    string `yaml:"schema"`
	Templates map[string]struct {
		ID                string            `yaml:"id"`
		Text              string            `yaml:"text"`
		Locales           map[string]string `yaml:"locales"`
		IncludesResources bool              `yaml:"includes_resources"`
	} `yaml:"templates"`
}
type CrisisTemplates struct {
	Schema    string `yaml:"schema"`
	Templates map[string]struct {
		Text    string            `yaml:"text"`
		Locales map[string]string `yaml:"locales"`
	} `yaml:"templates"`
}

type Router struct {
	Schema string `yaml:"schema"`
	Rules  []struct {
		ID    string           `yaml:"id"`
		When  []map[string]any `yaml:"when"`
		Route string           `yaml:"route"`
		Set   map[string]any   `yaml:"set"`
	} `yaml:"rules"`
	TrivialPatterns map[string]struct {
		Greetings []string `yaml:"greetings"`
		Thanks    []string `yaml:"thanks"`
		Acks      []string `yaml:"acks"`
		MaxChars  int      `yaml:"max_chars"`
	} `yaml:"trivial_patterns"`
	ShortlistRules struct {
		AllowedWhen struct {
			Any []map[string]any `yaml:"any"`
		} `yaml:"allowed_when"`
		BlockedWhen []string `yaml:"blocked_when"`
		MaxItems    int      `yaml:"max_items"`
	} `yaml:"shortlist_rules"`
}
type PolicyGate struct {
	Schema    string `yaml:"schema"`
	Questions struct {
		MaxConsecutiveQuestionTurns int `yaml:"max_consecutive_question_turns"`
		MaxPerReply                 int `yaml:"max_per_reply"`
	} `yaml:"questions"`
	Hypotheses struct {
		VoiceRequiresStatus string `yaml:"voice_requires_status"`
		VoiceMinGapHours    int    `yaml:"voice_min_gap_hours"`
		VoiceMaxPerSession  int    `yaml:"voice_max_per_session"`
	} `yaml:"hypotheses"`
	Protocols struct {
		OfferMinTurnsSinceLastOffer int `yaml:"offer_min_turns_since_last_offer"`
		MaxOffersPerSession         int `yaml:"max_offers_per_session"`
	} `yaml:"protocols"`
	Risk struct {
		BrainMayRaise bool `yaml:"brain_may_raise"`
		BrainMayLower bool `yaml:"brain_may_lower"`
	} `yaml:"risk"`
	Repairs struct {
		OnInvalidObjective string `yaml:"on_invalid_objective"`
		OnMissingEvidence  string `yaml:"on_missing_evidence"`
		OnUnknownSkill     string `yaml:"on_unknown_skill"`
	} `yaml:"repairs"`
}
type RetrievalDefaults struct {
	MaxItems       int `yaml:"max_items" json:"max_items"`
	HardCapWithHop int `yaml:"hard_cap_with_hop" json:"hard_cap_with_hop"`
	PerItemTokens  int `yaml:"per_item_tokens" json:"per_item_tokens"`
	TotalTokens    int `yaml:"total_tokens" json:"total_tokens"`
}
type Retrieval struct {
	Schema   string            `yaml:"schema"`
	Defaults RetrievalDefaults `yaml:"defaults"`
	Timeouts struct {
		TotalMS       int `yaml:"total_ms"`
		PerStrategyMS int `yaml:"per_strategy_ms"` // legacy deployment name: query embedding timeout
	} `yaml:"timeouts"`
}
type OperationalBudgets struct {
	SchedulerIntervalMS  int `yaml:"scheduler_interval_ms" json:"scheduler_interval_ms"`
	SchedulerRetryBaseMS int `yaml:"scheduler_retry_base_ms" json:"scheduler_retry_base_ms"`
	SchedulerRetryMaxMS  int `yaml:"scheduler_retry_max_ms" json:"scheduler_retry_max_ms"`
	SchedulerMaxFailures int `yaml:"scheduler_max_failures" json:"scheduler_max_failures"`
	HintReadMS           int `yaml:"hint_read_ms" json:"hint_read_ms"`
	AttemptWriteMS       int `yaml:"attempt_write_ms" json:"attempt_write_ms"`
	StageWriteMS         int `yaml:"stage_write_ms" json:"stage_write_ms"`
	TraceReferenceMS     int `yaml:"trace_reference_ms" json:"trace_reference_ms"`
	TraceJobsMS          int `yaml:"trace_jobs_ms" json:"trace_jobs_ms"`
	TurnLeaseMS          int `yaml:"turn_lease_ms" json:"turn_lease_ms"`
	TurnRenewMS          int `yaml:"turn_renew_ms" json:"turn_renew_ms"`
	TurnCleanupMS        int `yaml:"turn_cleanup_ms" json:"turn_cleanup_ms"`
	WorkerLeaseMS        int `yaml:"worker_lease_ms" json:"worker_lease_ms"`
	WorkerTimeoutMS      int `yaml:"worker_timeout_ms" json:"worker_timeout_ms"`
	WorkerCleanupMS      int `yaml:"worker_cleanup_ms" json:"worker_cleanup_ms"`
	HintVerificationMS   int `yaml:"hint_verification_ms" json:"hint_verification_ms"`
	HeartbeatMS          int `yaml:"heartbeat_ms" json:"heartbeat_ms"`
	HeartbeatIntervalMS  int `yaml:"heartbeat_interval_ms" json:"heartbeat_interval_ms"`
	RecentMessages       int `yaml:"recent_messages" json:"recent_messages"`
}

type ContextTokenBudgets struct {
	Active     int `yaml:"active"`
	Persistent int `yaml:"persistent"`
	Profile    int `yaml:"profile"`
	Retrieved  int `yaml:"retrieved"`
	Recent     int `yaml:"recent"`
}
type Budgets struct {
	Context    ContextTokenBudgets `yaml:"context"`
	Operations OperationalBudgets  `yaml:"operations"`
	Schema     string              `yaml:"schema"`
	Turn       struct {
		JEVMS           int `yaml:"jev_ms,omitempty"`
		UnifiedMS       int `yaml:"unified_ms,omitempty"`
		TotalMS         int `yaml:"total_ms"`
		SafetyMS        int `yaml:"safety_ms"`
		RouterMS        int `yaml:"router_ms"`
		RetrievalMS     int `yaml:"retrieval_ms"`
		BrainMS         int `yaml:"brain_ms"`
		GenFirstTokenMS int `yaml:"gen_first_token_ms"`
		GenTotalMS      int `yaml:"gen_total_ms"`
	} `yaml:"turn"`
	Limits struct {
		MaxSyncLLMCalls  int `yaml:"max_sync_llm_calls"`
		MaxRegenerations int `yaml:"max_regenerations"`
		MaxBrainHops     int `yaml:"max_brain_hops"`
	} `yaml:"limits"`
	Tokens struct {
		JEVIn            int `yaml:"jev_in"`
		KnowledgeContent int `yaml:"knowledge_content"`
		MemoryWriterIn   int `yaml:"memory_writer_in"`
		ReflectionIn     int `yaml:"reflection_in"`
		UnifiedIn        int `yaml:"unified_in,omitempty"`
		UnifiedOut       int `yaml:"unified_out,omitempty"`
		BrainIn          int `yaml:"brain_in"`
		BrainOut         int `yaml:"brain_out"`
		GenIn            int `yaml:"gen_in"`
		GenOut           int `yaml:"gen_out"`
		WCtxTarget       int `yaml:"wctx_target"`
	} `yaml:"tokens"`
	Quotas struct {
		ReflectionPerUserPerDay    int `yaml:"reflection_per_user_per_day"`
		MemoryWriterPerUserPerHour int `yaml:"memory_writer_per_user_per_hour"`
		UserMsgsPerMinute          int `yaml:"user_msgs_per_minute"`
	} `yaml:"quotas"`
}
type Thresholds struct {
	Schema     string             `yaml:"schema"`
	Validator  map[string]float32 `yaml:"validator"` // optional until classifier is enabled
	Hypothesis struct {
		SupportedMinSupport  int `yaml:"supported_min_support"`
		SupportedMinSessions int `yaml:"supported_min_sessions"`
		ContestedWhen        struct {
			ContradictGTE           int  `yaml:"contradict_gte"`
			AndContradictGTESupport bool `yaml:"and_contradict_gte_support"`
		} `yaml:"contested_when"`
		StaleAfterDays        int `yaml:"stale_after_days"`
		RejectionCooldownDays int `yaml:"rejection_cooldown_days"`
	} `yaml:"hypothesis"`
	Memory struct {
		DedupeCosineGTE                       float64 `yaml:"dedupe_cosine_gte"`
		ContradictionRequiresUserConfirmation bool    `yaml:"contradiction_requires_user_confirmation"`
		StaleAfterDays                        int     `yaml:"stale_after_days"`
		StaleImportanceLT                     float64 `yaml:"stale_importance_lt"`
		HotSetMaxItems                        int     `yaml:"hot_set_max_items"`
	} `yaml:"memory"`
	Session struct {
		IdleTimeoutMin int `yaml:"idle_timeout_min"`
	} `yaml:"session"`
	Thread struct {
		DormantAfterDays int `yaml:"dormant_after_days"`
		MaxActive        int `yaml:"max_active"`
	} `yaml:"thread"`
	ResumptionBands struct {
		SameMin    int `yaml:"same_min"`
		ShortHours int `yaml:"short_hours"`
		DayDays    int `yaml:"day_days"`
	} `yaml:"resumption_bands"`
	Safety struct {
		L1MinConfidenceForHigh float64 `yaml:"l1_min_confidence_for_high"`
		Deescalate             struct {
			MinHoursSinceSignal  int    `yaml:"min_hours_since_signal"`
			RequireUserStatement bool   `yaml:"require_user_statement"`
			ThenLevel            string `yaml:"then_level"`
		} `yaml:"deescalate"`
	} `yaml:"safety"`
	Engagement struct {
		LateNightWindow   string `yaml:"late_night_window"`
		DependencyWatchIf struct {
			LateNightSessions7DGTE int `yaml:"late_night_sessions_7d_gte"`
			OrAvgDailyMinutesGTE   int `yaml:"or_avg_daily_minutes_gte"`
		} `yaml:"dependency_watch_if"`
	} `yaml:"engagement"`
	Reflection struct {
		Triggers struct {
			SessionClosed          bool `yaml:"session_closed"`
			MinNewItems            int  `yaml:"min_new_items"`
			ThreadTurnsGTE         int  `yaml:"thread_turns_gte"`
			ContradictionEventsGTE int  `yaml:"contradiction_events_gte"`
		} `yaml:"triggers"`
		MinEvidenceForFormulation struct {
			Sessions   int `yaml:"sessions"`
			Situations int `yaml:"situations"`
		} `yaml:"min_evidence_for_formulation"`
	} `yaml:"reflection"`
}
type MemoryConfig struct {
	Schema                      string   `yaml:"schema"`
	AllowedKinds                []string `yaml:"allowed_kinds"`
	RequireEvidence             bool     `yaml:"require_evidence"`
	PreserveDerivedOnSoftDelete bool     `yaml:"preserve_derived_on_soft_delete"`
	AllowHardDelete             bool     `yaml:"allow_hard_delete"`
}
type Sampling struct {
	Schema           string             `yaml:"schema"`
	Always           []string           `yaml:"always"`
	Random           map[string]float64 `yaml:"random"`
	ReviewQueueEvent string             `yaml:"review_queue_event"`
}
type Golden struct {
	Schema        string   `yaml:"schema"`
	ID            string   `yaml:"id"`
	Title         string   `yaml:"title"`
	Tags          []string `yaml:"tags"`
	SyntheticOnly bool     `yaml:"synthetic_only"`
	// Evaluations are not loaded at runtime. The complete scenario body is
	// intentionally owned by WP-15; WP-03 only checks the packaging envelope.
	Setup    yaml.Node `yaml:"setup"`
	Turn     yaml.Node `yaml:"turn"`
	Turns    yaml.Node `yaml:"turns"`
	Sessions yaml.Node `yaml:"sessions"`
	Expect   yaml.Node `yaml:"expect"`
}

// Ancillary static scripts are versioned assets, never model-generated content.
type BoundaryScripts struct {
	Schema  string `yaml:"schema"`
	Scripts map[string]struct {
		Text    string            `yaml:"text"`
		Locales map[string]string `yaml:"locales"`
	} `yaml:"scripts"`
}
type ProtocolOfferTexts struct {
	Schema       string `yaml:"schema"`
	WhyTemplates map[string]struct {
		Text    string            `yaml:"text"`
		Locales map[string]string `yaml:"locales"`
	} `yaml:"why_templates"`
	MustSay map[string]struct {
		Text    string            `yaml:"text"`
		Locales map[string]string `yaml:"locales"`
	} `yaml:"must_say"`
}
type ForbiddenClaims struct {
	Schema  string              `yaml:"schema"`
	Locales map[string][]string `yaml:"locales"`
}
