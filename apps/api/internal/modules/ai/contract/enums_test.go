package contract_test

import (
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"testing"
)

func TestClosedEnums(t *testing.T) {
	t.Run("Code", func(t *testing.T) {
		for _, value := range []c.Code{c.ErrQueued, c.ErrDuplicate, c.ErrBudget, c.ErrSafetyDegraded, c.ErrRetrievalDegraded, c.ErrBrainInvalid, c.ErrBrainTimeout, c.ErrGenFailed, c.ErrValidation, c.ErrStateConflict, c.ErrStateCommit, c.ErrBundle, c.ErrProvider, c.ErrInternal, c.ErrRateLimited, c.ErrForbidden} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Code{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ConfidenceClass", func(t *testing.T) {
		for _, value := range []c.ConfidenceClass{c.ConfLow, c.ConfMedium, c.ConfHigh} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ConfidenceClass{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ContextKind", func(t *testing.T) {
		for _, value := range []c.ContextKind{c.ContextEntity, c.ContextTopic, c.ContextTimeWindow, c.ContextItem} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ContextKind{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ContextReason", func(t *testing.T) {
		for _, value := range []c.ContextReason{c.ContextReferenceUnresolved, c.ContextMissingBackground, c.ContextVerifyFact} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ContextReason{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Epistemic", func(t *testing.T) {
		for _, value := range []c.Epistemic{c.EpReported, c.EpInferred, c.EpSystemObserved, c.EpUnknown} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Epistemic{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("EventType", func(t *testing.T) {
		for _, value := range []c.EventType{c.EvTurnCompleted, c.EvSessionStarted, c.EvSessionClosed, c.EvRiskLevelChanged, c.EvProtocolStatusChanged, c.EvMemoryDeleted, c.EvMemoryCorrected, c.EvFollowUpScheduled, c.EvReviewFlagged, c.EvDeletionCompleted} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.EventType{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("HypothesisOp", func(t *testing.T) {
		for _, value := range []c.HypothesisOp{c.HypSupport, c.HypContradict, c.HypPropose, c.HypUserEndorsed, c.HypUserRejected} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.HypothesisOp{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Intent", func(t *testing.T) {
		for _, value := range []c.Intent{c.IntentShareDistress, c.IntentSeekAdvice, c.IntentAnswerQuestion, c.IntentReportExercise, c.IntentAskRecall, c.IntentAskInfo, c.IntentSmallTalk, c.IntentDisagreeCorrect, c.IntentRequestTechnique, c.IntentDiscloseEvent, c.IntentOther} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Intent{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Length", func(t *testing.T) {
		for _, value := range []c.Length{c.LengthMinimal, c.LengthShort, c.LengthMedium, c.LengthLong} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Length{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("MemoryKind", func(t *testing.T) {
		for _, value := range []c.MemoryKind{c.MemFact, c.MemPreference, c.MemEpisode, c.MemHypothesis, c.MemGoal, c.MemEvent, c.MemSafetyEvt} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.MemoryKind{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("MemoryOp", func(t *testing.T) {
		for _, value := range []c.MemoryOp{c.OpAdd, c.OpUpdate, c.OpConfirm, c.OpContradict, c.OpSupersede, c.OpReject, c.OpNoop} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.MemoryOp{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("MemoryStatus", func(t *testing.T) {
		for _, value := range []c.MemoryStatus{c.MemActive, c.MemContested, c.MemSuperseded, c.MemRejected, c.MemStale, c.MemProposed, c.MemSupported} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.MemoryStatus{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("MessageType", func(t *testing.T) {
		for _, value := range []c.MessageType{c.MessageDisclosure, c.MessageQuestion, c.MessageAnswer, c.MessageReport, c.MessageRequest, c.MessageMetaType, c.MessageGreeting} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.MessageType{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Objective", func(t *testing.T) {
		for _, value := range []c.Objective{c.ObjReflect, c.ObjValidateSupport, c.ObjClarify, c.ObjExplore, c.ObjProvideInsight, c.ObjProvideGuidance, c.ObjOfferIntervention, c.ObjContinueIntervention, c.ObjCloseOrPause, c.ObjReviewProgress, c.ObjSummarize, c.ObjRepair, c.ObjAnswerDirect, c.ObjBoundary, c.ObjSafetySupport} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Objective{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ProtocolAction", func(t *testing.T) {
		for _, value := range []c.ProtocolAction{c.PAAdvance, c.PARepeat, c.PAAdapt, c.PAPause, c.PAStop, c.PAComplete, c.PAOffer, c.PAStart, c.PADecline} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ProtocolAction{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ProtocolStatus", func(t *testing.T) {
		for _, value := range []c.ProtocolStatus{c.PIOffered, c.PIActive, c.PIPaused, c.PICompleted, c.PIStopped, c.PIDeclined} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ProtocolStatus{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("RetrievalStrategy", func(t *testing.T) {
		for _, value := range []c.RetrievalStrategy{c.StratTemporal, c.StratThread, c.StratHybrid, c.StratAnaphora, c.StratEntity} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.RetrievalStrategy{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("RiskCategory", func(t *testing.T) {
		for _, value := range []c.RiskCategory{c.RiskSuicidalIdeation, c.RiskSelfHarm, c.RiskHarmToOthers, c.RiskAbuseDisclosure, c.RiskMedicalEmergency, c.RiskSubstanceCrisis, c.RiskOther} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.RiskCategory{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("RiskLevel", func(t *testing.T) {
		for _, value := range []c.RiskLevel{c.RiskNone, c.RiskLow, c.RiskElevated, c.RiskHigh, c.RiskImminent} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.RiskLevel{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Role", func(t *testing.T) {
		for _, value := range []c.Role{c.RoleUser, c.RoleAssistant, c.RoleSystem} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Role{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Route", func(t *testing.T) {
		for _, value := range []c.Route{c.RouteNormal, c.RouteFast, c.RouteCrisis, c.RouteProactive} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Route{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("SafetyContextFlag", func(t *testing.T) {
		for _, value := range []c.SafetyContextFlag{c.ContextQuoted, c.ContextFictional, c.ContextHypothetical, c.ContextThirdPerson, c.ContextPastEvent} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.SafetyContextFlag{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Sensitivity", func(t *testing.T) {
		for _, value := range []c.Sensitivity{c.SensNormal, c.SensSensitive, c.SensHighly} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Sensitivity{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("SessionMode", func(t *testing.T) {
		for _, value := range []c.SessionMode{c.ModeVenting, c.ModeExploring, c.ModeProblem, c.ModeExercise, c.ModeCheckIn, c.ModeSafety, c.ModeSmallTalk} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.SessionMode{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("StepOutcome", func(t *testing.T) {
		for _, value := range []c.StepOutcome{c.OutSuccess, c.OutPartial, c.OutStuck, c.OutDistress, c.OutDecline, c.OutOffTopic} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.StepOutcome{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ThreadPhase", func(t *testing.T) {
		for _, value := range []c.ThreadPhase{c.PhaseExploring, c.PhaseUnderstanding, c.PhaseWorking, c.PhaseMaintaining, c.PhaseResolved, c.PhasePaused, c.PhaseDormant} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ThreadPhase{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ThreadRelation", func(t *testing.T) {
		for _, value := range []c.ThreadRelation{c.RelContinues, c.RelShifts, c.RelNew, c.RelReturns} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ThreadRelation{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("Tone", func(t *testing.T) {
		for _, value := range []c.Tone{c.ToneWarmSteady, c.ToneGentle, c.ToneCalmDirect, c.ToneLight, c.ToneFormal} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.Tone{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("TurnOrigin", func(t *testing.T) {
		for _, value := range []c.TurnOrigin{c.OriginUser, c.OriginWakeUp} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.TurnOrigin{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("TurnStatus", func(t *testing.T) {
		for _, value := range []c.TurnStatus{c.TurnQueued, c.TurnRunning, c.TurnCompleted, c.TurnFailed, c.TurnCancelled} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.TurnStatus{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
	t.Run("ValidationAction", func(t *testing.T) {
		for _, value := range []c.ValidationAction{c.VAAccept, c.VARegenerate, c.VAFallback, c.VACrisis} {
			if !value.Valid() {
				t.Errorf("valid enum rejected: %s", value)
			}
		}
		for _, value := range []c.ValidationAction{"", "unknown_synthetic_value", "NONE", " none "} {
			if value.Valid() {
				t.Errorf("invalid enum accepted: %s", value)
			}
		}
	})
}
