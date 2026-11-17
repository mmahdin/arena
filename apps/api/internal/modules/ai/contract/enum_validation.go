package contract

func (v Code) Valid() bool {
	switch v {
	case ErrQueued, ErrDuplicate, ErrBudget, ErrSafetyDegraded, ErrRetrievalDegraded, ErrBrainInvalid, ErrBrainTimeout, ErrGenFailed, ErrValidation, ErrStateConflict, ErrStateCommit, ErrBundle, ErrProvider, ErrInternal, ErrRateLimited, ErrForbidden:
		return true
	default:
		return false
	}
}

func (v ConfidenceClass) Valid() bool {
	switch v {
	case ConfLow, ConfMedium, ConfHigh:
		return true
	default:
		return false
	}
}

func (v ContextKind) Valid() bool {
	switch v {
	case ContextEntity, ContextTopic, ContextTimeWindow, ContextItem:
		return true
	default:
		return false
	}
}

func (v ContextReason) Valid() bool {
	switch v {
	case ContextReferenceUnresolved, ContextMissingBackground, ContextVerifyFact:
		return true
	default:
		return false
	}
}

func (v Epistemic) Valid() bool {
	switch v {
	case EpReported, EpInferred, EpSystemObserved, EpUnknown:
		return true
	default:
		return false
	}
}

func (v EventType) Valid() bool {
	switch v {
	case EvTurnCompleted, EvSessionStarted, EvSessionClosed, EvRiskLevelChanged, EvProtocolStatusChanged, EvMemoryDeleted, EvMemoryCorrected, EvFollowUpScheduled, EvReviewFlagged, EvDeletionCompleted:
		return true
	default:
		return false
	}
}

func (v HypothesisOp) Valid() bool {
	switch v {
	case HypSupport, HypContradict, HypPropose, HypUserEndorsed, HypUserRejected:
		return true
	default:
		return false
	}
}

func (v Intent) Valid() bool {
	switch v {
	case IntentShareDistress, IntentSeekAdvice, IntentAnswerQuestion, IntentReportExercise, IntentAskRecall, IntentAskInfo, IntentSmallTalk, IntentDisagreeCorrect, IntentRequestTechnique, IntentDiscloseEvent, IntentOther:
		return true
	default:
		return false
	}
}

func (v Length) Valid() bool {
	switch v {
	case LengthMinimal, LengthShort, LengthMedium, LengthLong:
		return true
	default:
		return false
	}
}

func (v MemoryKind) Valid() bool {
	switch v {
	case MemFact, MemPreference, MemEpisode, MemHypothesis, MemGoal, MemEvent, MemSafetyEvt:
		return true
	default:
		return false
	}
}

func (v MemoryOp) Valid() bool {
	switch v {
	case OpAdd, OpUpdate, OpConfirm, OpContradict, OpSupersede, OpReject, OpNoop:
		return true
	default:
		return false
	}
}

func (v MemoryStatus) Valid() bool {
	switch v {
	case MemActive, MemContested, MemSuperseded, MemRejected, MemStale, MemProposed, MemSupported:
		return true
	default:
		return false
	}
}

func (v MessageType) Valid() bool {
	switch v {
	case MessageDisclosure, MessageQuestion, MessageAnswer, MessageReport, MessageRequest, MessageMetaType, MessageGreeting:
		return true
	default:
		return false
	}
}

func (v Objective) Valid() bool {
	switch v {
	case ObjReflect, ObjValidateSupport, ObjClarify, ObjExplore, ObjProvideInsight, ObjProvideGuidance, ObjOfferIntervention, ObjContinueIntervention, ObjCloseOrPause, ObjReviewProgress, ObjSummarize, ObjRepair, ObjAnswerDirect, ObjBoundary, ObjSafetySupport:
		return true
	default:
		return false
	}
}

func (v ProtocolAction) Valid() bool {
	switch v {
	case PAAdvance, PARepeat, PAAdapt, PAPause, PAStop, PAComplete, PAOffer, PAStart, PADecline:
		return true
	default:
		return false
	}
}

func (v ProtocolStatus) Valid() bool {
	switch v {
	case PIOffered, PIActive, PIPaused, PICompleted, PIStopped, PIDeclined:
		return true
	default:
		return false
	}
}

func (v RetrievalStrategy) Valid() bool {
	switch v {
	case StratTemporal, StratThread, StratHybrid, StratAnaphora, StratEntity:
		return true
	default:
		return false
	}
}

func (v RiskCategory) Valid() bool {
	switch v {
	case RiskSuicidalIdeation, RiskSelfHarm, RiskHarmToOthers, RiskAbuseDisclosure, RiskMedicalEmergency, RiskSubstanceCrisis, RiskOther:
		return true
	default:
		return false
	}
}

func (v RiskLevel) Valid() bool {
	switch v {
	case RiskNone, RiskLow, RiskElevated, RiskHigh, RiskImminent:
		return true
	default:
		return false
	}
}

func (v Role) Valid() bool {
	switch v {
	case RoleUser, RoleAssistant, RoleSystem:
		return true
	default:
		return false
	}
}

func (v Route) Valid() bool {
	switch v {
	case RouteNormal, RouteFast, RouteCrisis, RouteProactive:
		return true
	default:
		return false
	}
}

func (v SafetyContextFlag) Valid() bool {
	switch v {
	case ContextQuoted, ContextFictional, ContextHypothetical, ContextThirdPerson, ContextPastEvent:
		return true
	default:
		return false
	}
}

func (v Sensitivity) Valid() bool {
	switch v {
	case SensNormal, SensSensitive, SensHighly:
		return true
	default:
		return false
	}
}

func (v SessionMode) Valid() bool {
	switch v {
	case ModeVenting, ModeExploring, ModeProblem, ModeExercise, ModeCheckIn, ModeSafety, ModeSmallTalk:
		return true
	default:
		return false
	}
}

func (v StepOutcome) Valid() bool {
	switch v {
	case OutSuccess, OutPartial, OutStuck, OutDistress, OutDecline, OutOffTopic:
		return true
	default:
		return false
	}
}

func (v ThreadPhase) Valid() bool {
	switch v {
	case PhaseExploring, PhaseUnderstanding, PhaseWorking, PhaseMaintaining, PhaseResolved, PhasePaused, PhaseDormant:
		return true
	default:
		return false
	}
}

func (v ThreadRelation) Valid() bool {
	switch v {
	case RelContinues, RelShifts, RelNew, RelReturns:
		return true
	default:
		return false
	}
}

func (v Tone) Valid() bool {
	switch v {
	case ToneWarmSteady, ToneGentle, ToneCalmDirect, ToneLight, ToneFormal:
		return true
	default:
		return false
	}
}

func (v TurnOrigin) Valid() bool {
	switch v {
	case OriginUser, OriginWakeUp:
		return true
	default:
		return false
	}
}

func (v TurnStatus) Valid() bool {
	switch v {
	case TurnQueued, TurnRunning, TurnCompleted, TurnFailed, TurnCancelled:
		return true
	default:
		return false
	}
}

func (v ValidationAction) Valid() bool {
	switch v {
	case VAAccept, VARegenerate, VAFallback, VACrisis:
		return true
	default:
		return false
	}
}

// Rank returns -1 for invalid values; callers must validate inputs first.
func (r RiskLevel) Rank() int {
	switch r {
	case RiskNone:
		return 0
	case RiskLow:
		return 1
	case RiskElevated:
		return 2
	case RiskHigh:
		return 3
	case RiskImminent:
		return 4
	default:
		return -1
	}
}

// MaxRisk fails closed to imminent for malformed inputs. A missing classifier
// must be represented by the caller as RiskElevated, never as the zero value.
func MaxRisk(a, b RiskLevel) RiskLevel {
	if !a.Valid() || !b.Valid() {
		return RiskImminent
	}
	if a.Rank() >= b.Rank() {
		return a
	}
	return b
}
