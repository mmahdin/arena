package contract

import "encoding/json"

// Required collections encode as [] rather than null. Optional collections use
// omitempty. Value receivers leave the caller's slices unchanged.
func (v Admission) MarshalJSON() ([]byte, error) {
	type plain Admission
	if v.Recent == nil {
		v.Recent = []MsgView{}
	}
	return json.Marshal(plain(v))
}

func (v SafetyBanner) MarshalJSON() ([]byte, error) {
	type plain SafetyBanner
	if v.Resources == nil {
		v.Resources = []CrisisResource{}
	}
	return json.Marshal(plain(v))
}

func (v StateDelta) MarshalJSON() ([]byte, error) {
	type plain StateDelta
	if v.ThreadOps == nil {
		v.ThreadOps = []ThreadOp{}
	}
	if v.OpenLoops == nil {
		v.OpenLoops = []OpenLoopOp{}
	}
	if v.FollowUps == nil {
		v.FollowUps = []FollowUpOp{}
	}
	if v.Events == nil {
		v.Events = []DomainEvent{}
	}
	return json.Marshal(plain(v))
}

func (v SessionPatch) MarshalJSON() ([]byte, error) {
	type plain SessionPatch
	if v.EmotionLabels == nil {
		v.EmotionLabels = []string{}
	}
	return json.Marshal(plain(v))
}

func (v ThreadPatch) MarshalJSON() ([]byte, error) {
	type plain ThreadPatch
	if v.Slots == nil {
		v.Slots = []SlotUpdate{}
	}
	if v.HypothesisSignals == nil {
		v.HypothesisSignals = []HypothesisSignal{}
	}
	if v.RelatedThreadIDs == nil {
		v.RelatedThreadIDs = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v PrefsPatch) MarshalJSON() ([]byte, error) {
	type plain PrefsPatch
	if v.AvoidAdd == nil {
		v.AvoidAdd = []string{}
	}
	if v.AvoidRemove == nil {
		v.AvoidRemove = []string{}
	}
	return json.Marshal(plain(v))
}

func (v CommitResult) MarshalJSON() ([]byte, error) {
	type plain CommitResult
	if v.Rejected == nil {
		v.Rejected = []DeltaRejection{}
	}
	return json.Marshal(plain(v))
}

func (v WorkingContext) MarshalJSON() ([]byte, error) {
	type plain WorkingContext
	if v.Recent == nil {
		v.Recent = []MsgView{}
	}
	return json.Marshal(plain(v))
}

func (v ThreadView) MarshalJSON() ([]byte, error) {
	type plain ThreadView
	if v.Slots == nil {
		v.Slots = []SlotView{}
	}
	return json.Marshal(plain(v))
}

func (v ProtocolView) MarshalJSON() ([]byte, error) {
	type plain ProtocolView
	if v.AllowedActions == nil {
		v.AllowedActions = []ProtocolAction{}
	}
	return json.Marshal(plain(v))
}

func (v RetrievalMeta) MarshalJSON() ([]byte, error) {
	type plain RetrievalMeta
	if v.Strategies == nil {
		v.Strategies = []string{}
	}
	return json.Marshal(plain(v))
}

func (v FormulationBrief) MarshalJSON() ([]byte, error) {
	type plain FormulationBrief
	if v.Gaps == nil {
		v.Gaps = []string{}
	}
	return json.Marshal(plain(v))
}

func (v ProtocolBrief) MarshalJSON() ([]byte, error) {
	type plain ProtocolBrief
	if v.FitTags == nil {
		v.FitTags = []string{}
	}
	return json.Marshal(plain(v))
}

func (v SafetySignal) MarshalJSON() ([]byte, error) {
	type plain SafetySignal
	if v.Categories == nil {
		v.Categories = []RiskCategory{}
	}
	if v.Triggers == nil {
		v.Triggers = []SafetyTrigger{}
	}
	return json.Marshal(plain(v))
}

func (v CrisisPlan) MarshalJSON() ([]byte, error) {
	type plain CrisisPlan
	if v.RequiredElements == nil {
		v.RequiredElements = []string{}
	}
	if v.Resources == nil {
		v.Resources = []CrisisResource{}
	}
	if v.Forbidden == nil {
		v.Forbidden = []string{}
	}
	return json.Marshal(plain(v))
}

func (v RetrievalRequest) MarshalJSON() ([]byte, error) {
	type plain RetrievalRequest
	if v.Strategies == nil {
		v.Strategies = []RetrievalStrategy{}
	}
	if v.Exclude == nil {
		v.Exclude = []ID{}
	}
	if v.IncludeSensitivity == nil {
		v.IncludeSensitivity = []Sensitivity{}
	}
	return json.Marshal(plain(v))
}

func (v RetrievalResult) MarshalJSON() ([]byte, error) {
	type plain RetrievalResult
	if v.Items == nil {
		v.Items = []MemoryView{}
	}
	return json.Marshal(plain(v))
}

func (v ProtocolStepSpec) MarshalJSON() ([]byte, error) {
	type plain ProtocolStepSpec
	if v.ExpectedInputs == nil {
		v.ExpectedInputs = []string{}
	}
	if v.AllowedAdaptations == nil {
		v.AllowedAdaptations = []string{}
	}
	return json.Marshal(plain(v))
}

func (v OfferSpec) MarshalJSON() ([]byte, error) {
	type plain OfferSpec
	if v.Why == nil {
		v.Why = []string{}
	}
	return json.Marshal(plain(v))
}

func (v SessionRecord) MarshalJSON() ([]byte, error) {
	type plain SessionRecord
	if v.EmotionTrajectory == nil {
		v.EmotionTrajectory = []string{}
	}
	return json.Marshal(plain(v))
}

func (v ThreadRecord) MarshalJSON() ([]byte, error) {
	type plain ThreadRecord
	if v.HypothesisIDs == nil {
		v.HypothesisIDs = []ID{}
	}
	if v.RelatedThreadIDs == nil {
		v.RelatedThreadIDs = []ID{}
	}
	if v.ProblemTags == nil {
		v.ProblemTags = []string{}
	}
	return json.Marshal(plain(v))
}

func (v ThreadInit) MarshalJSON() ([]byte, error) {
	type plain ThreadInit
	if v.ProblemTags == nil {
		v.ProblemTags = []string{}
	}
	return json.Marshal(plain(v))
}

func (v UserState) MarshalJSON() ([]byte, error) {
	type plain UserState
	if v.ProtocolHistory == nil {
		v.ProtocolHistory = []ProtocolHistoryBrief{}
	}
	return json.Marshal(plain(v))
}

func (v TurnAnalysis) MarshalJSON() ([]byte, error) {
	type plain TurnAnalysis
	if v.Emotions == nil {
		v.Emotions = []EmotionRead{}
	}
	if v.PastRefs == nil {
		v.PastRefs = []PastRef{}
	}
	if v.Ambiguities == nil {
		v.Ambiguities = []Ambiguity{}
	}
	return json.Marshal(plain(v))
}

func (v PastRef) MarshalJSON() ([]byte, error) {
	type plain PastRef
	if v.ResolvedTo == nil {
		v.ResolvedTo = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v UnderstandingUpdate) MarshalJSON() ([]byte, error) {
	type plain UnderstandingUpdate
	if v.Slots == nil {
		v.Slots = []SlotUpdate{}
	}
	if v.Hypotheses == nil {
		v.Hypotheses = []HypothesisSignal{}
	}
	if v.Unknowns == nil {
		v.Unknowns = []string{}
	}
	return json.Marshal(plain(v))
}

func (v SlotUpdate) MarshalJSON() ([]byte, error) {
	type plain SlotUpdate
	if v.Evidence == nil {
		v.Evidence = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v HypothesisSignal) MarshalJSON() ([]byte, error) {
	type plain HypothesisSignal
	if v.Evidence == nil {
		v.Evidence = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v Decision) MarshalJSON() ([]byte, error) {
	type plain Decision
	if v.Rationale == nil {
		v.Rationale = []string{}
	}
	if v.Alternatives == nil {
		v.Alternatives = []Objective{}
	}
	return json.Marshal(plain(v))
}

func (v ResponsePlan) MarshalJSON() ([]byte, error) {
	type plain ResponsePlan
	if v.PermittedContext == nil {
		v.PermittedContext = []ID{}
	}
	if v.MustAvoid == nil {
		v.MustAvoid = []string{}
	}
	return json.Marshal(plain(v))
}

func (v SkillSelection) MarshalJSON() ([]byte, error) {
	type plain SkillSelection
	if v.Secondary == nil {
		v.Secondary = []string{}
	}
	return json.Marshal(plain(v))
}

func (v BrainRiskRead) MarshalJSON() ([]byte, error) {
	type plain BrainRiskRead
	if v.Categories == nil {
		v.Categories = []RiskCategory{}
	}
	return json.Marshal(plain(v))
}

func (v ValidationReport) MarshalJSON() ([]byte, error) {
	type plain ValidationReport
	if v.Violations == nil {
		v.Violations = []Violation{}
	}
	return json.Marshal(plain(v))
}

func (v SafetyClassifierOutput) MarshalJSON() ([]byte, error) {
	type plain SafetyClassifierOutput
	if v.Categories == nil {
		v.Categories = []RiskCategory{}
	}
	return json.Marshal(plain(v))
}

func (v MemoryItem) MarshalJSON() ([]byte, error) {
	type plain MemoryItem
	if v.Evidence == nil {
		v.Evidence = []EvidenceRef{}
	}
	if v.ThreadIDs == nil {
		v.ThreadIDs = []ID{}
	}
	if v.Entities == nil {
		v.Entities = []EntityRef{}
	}
	if v.Tags == nil {
		v.Tags = []string{}
	}
	return json.Marshal(plain(v))
}

func (v MemoryOpProposal) MarshalJSON() ([]byte, error) {
	type plain MemoryOpProposal
	if v.EvidenceMsg == nil {
		v.EvidenceMsg = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v MemoryWriterOutput) MarshalJSON() ([]byte, error) {
	type plain MemoryWriterOutput
	if v.Ops == nil {
		v.Ops = []MemoryOpProposal{}
	}
	return json.Marshal(plain(v))
}

func (v ReflectionResult) MarshalJSON() ([]byte, error) {
	type plain ReflectionResult
	if v.HypothesisOps == nil {
		v.HypothesisOps = []MemoryOpProposal{}
	}
	if v.ThreadProposals == nil {
		v.ThreadProposals = []ThreadProposal{}
	}
	if v.InformationGaps == nil {
		v.InformationGaps = []string{}
	}
	return json.Marshal(plain(v))
}

func (v FormulationField) MarshalJSON() ([]byte, error) {
	type plain FormulationField
	if v.Evidence == nil {
		v.Evidence = []ID{}
	}
	return json.Marshal(plain(v))
}

func (v EpisodeDoc) MarshalJSON() ([]byte, error) {
	type plain EpisodeDoc
	if v.Topics == nil {
		v.Topics = []string{}
	}
	if v.Emotions == nil {
		v.Emotions = []Emotion{}
	}
	if v.FollowUps == nil {
		v.FollowUps = []string{}
	}
	return json.Marshal(plain(v))
}
