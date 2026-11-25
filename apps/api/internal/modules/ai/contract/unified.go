package contract

import "encoding/json"

// UnifiedTurn is the active conversational output. Optional intents are no-ops
// when absent; IDs and expected versions are resolved by the application.
const UnifiedSchemaVersion = "unified_turn.v2"

type StateProposal struct {
	ID        string          `json:"id"`
	Kind      string          `json:"kind"`
	Value     json.RawMessage `json:"value"`
	DependsOn []string        `json:"depends_on"`
}
type BackgroundScope struct {
	Kind      string `json:"kind"`
	ThreadIDs []ID   `json:"thread_ids"`
}
type BackgroundRequest struct {
	Scope       *BackgroundScope `json:"scope,omitempty"`
	ID          string           `json:"id"`
	Kind        string           `json:"kind"`
	Objective   string           `json:"objective"`
	Topic       string           `json:"topic"`
	Queries     []string         `json:"queries"`
	EvidenceIDs []ID             `json:"evidence_ids"`
	DependsOn   []string         `json:"depends_on"`
}
type HintProposal struct {
	Topic       string   `json:"topic"`
	ResourceIDs []string `json:"resource_ids"`
	TTLTurns    int      `json:"ttl_turns"`
}
type UnifiedReply struct {
	Text         string             `json:"text"`
	Conversation ConversationIntent `json:"conversation"`
	UsedContext  []ID               `json:"used_context_ids"`
	DependsOn    []string           `json:"depends_on"`
}
type UnifiedEnvelope struct {
	SchemaVersion      string            `json:"schema_version"`
	Reply              UnifiedReply      `json:"reply"`
	StateProposals     []json.RawMessage `json:"state_proposals"`
	BackgroundRequests []json.RawMessage `json:"background_requests"`
	FutureHints        []json.RawMessage `json:"future_hints"`
}
type UnifiedTurn struct {
	Proposals          []StateProposal     `json:"-"`
	BackgroundRequests []BackgroundRequest `json:"-"`
	FutureHints        []HintProposal      `json:"-"`
	Rejected           []DeltaRejection    `json:"-"`
	ReplyDependencies  []string            `json:"-"`
	SchemaVersion      string              `json:"schema_version"`
	Reply              string              `json:"reply"`
	Conversation       ConversationIntent  `json:"conversation"`
	Session            *SessionIntent      `json:"session_intent,omitempty"`
	Thread             *ThreadIntent       `json:"thread_intent,omitempty"`
	Protocol           *ProtocolIntent     `json:"protocol_intent,omitempty"`
	OpenLoops          []OpenLoopIntent    `json:"open_loops,omitempty"`
	Preferences        *PrefsPatch         `json:"preferences,omitempty"`
	MemorySignals      []string            `json:"memory_signals"`
	Reflection         bool                `json:"reflection_signal"`
	UsedContext        []ID                `json:"used_context_ids"`
	RiskRead           *BrainRiskRead      `json:"risk_read,omitempty"`
}
type ConversationIntent struct {
	Intent    Intent    `json:"intent"`
	Objective Objective `json:"objective"`
}
type SessionIntent struct {
	Mode            *SessionMode `json:"mode,omitempty"`
	Summary         *string      `json:"summary,omitempty"` // bounded rolling replacement, not append
	AnsweredPending bool         `json:"answered_pending"`  // Unified's semantic judgment; independent of reply_to_id
}
type ThreadIntent struct {
	Relation   ThreadRelation     `json:"relation"`
	Target     *ID                `json:"target_id,omitempty"`
	Topic      *string            `json:"topic,omitempty"`
	Goal       *string            `json:"goal,omitempty"`
	Phase      *ThreadPhase       `json:"phase,omitempty"`
	Slots      []SlotUpdate       `json:"slots"`
	Hypotheses []HypothesisSignal `json:"hypotheses"`
	Related    []ID               `json:"related_ids,omitempty"`
	Operations []ThreadOp         `json:"operations,omitempty"`
}
type ProtocolIntent struct {
	Directive      ProtocolDirective `json:"directive"`
	UserAffirmed   bool              `json:"user_affirmed"`
	ExpectedStatus ProtocolStatus    `json:"expected_status"`
	ExpectedStep   string            `json:"expected_step"` // empty when no step should be delivered
}
type OpenLoopIntent struct {
	Op   string `json:"op"`
	ID   *ID    `json:"id,omitempty"` // required for close/expire; server assigns add IDs
	Kind string `json:"kind,omitempty"`
	Text string `json:"text,omitempty"`
	Ref  *ID    `json:"ref,omitempty"`
}

// MarshalJSON is the v2 wire contract. Typed fields remain useful to state and
// background services; there is only one model-facing execution contract.
func (u UnifiedTurn) MarshalJSON() ([]byte, error) {
	e := UnifiedEnvelope{SchemaVersion: u.SchemaVersion, Reply: UnifiedReply{Text: u.Reply, Conversation: u.Conversation, UsedContext: u.UsedContext, DependsOn: u.ReplyDependencies}, StateProposals: []json.RawMessage{}, BackgroundRequests: []json.RawMessage{}, FutureHints: []json.RawMessage{}}
	if e.Reply.UsedContext == nil {
		e.Reply.UsedContext = []ID{}
	}
	if e.Reply.DependsOn == nil {
		e.Reply.DependsOn = []string{}
	}
	proposals := u.Proposals
	if proposals == nil {
		add := func(kind string, v any) {
			raw, _ := json.Marshal(v)
			proposals = append(proposals, StateProposal{ID: kind, Kind: kind, Value: raw, DependsOn: []string{}})
		}
		if u.Session != nil {
			add("session_intent", u.Session)
		}
		if u.Thread != nil {
			add("thread_intent", u.Thread)
		}
		if u.Protocol != nil {
			add("protocol_intent", u.Protocol)
		}
		if len(u.OpenLoops) > 0 {
			add("open_loops", u.OpenLoops)
		}
		if u.Preferences != nil {
			add("preferences", u.Preferences)
		}
		if u.RiskRead != nil {
			add("risk_read", u.RiskRead)
		}
	}
	for _, p := range proposals {
		if p.DependsOn == nil {
			p.DependsOn = []string{}
		}
		raw, _ := json.Marshal(p)
		e.StateProposals = append(e.StateProposals, raw)
	}
	for _, r := range u.BackgroundRequests {
		if r.DependsOn == nil {
			r.DependsOn = []string{}
		}
		if r.EvidenceIDs == nil {
			r.EvidenceIDs = []ID{}
		}
		raw, _ := json.Marshal(r)
		e.BackgroundRequests = append(e.BackgroundRequests, raw)
	}
	for _, h := range u.FutureHints {
		if h.ResourceIDs == nil {
			h.ResourceIDs = []string{}
		}
		raw, _ := json.Marshal(h)
		e.FutureHints = append(e.FutureHints, raw)
	}
	return json.Marshal(e)
}

// UnmarshalJSON reads recorded v2 output and historical v1 snapshots. Model
// ingress must still use schema.Set.DecodeUnified for independent validation.
func (u *UnifiedTurn) UnmarshalJSON(raw []byte) error {
	var version struct {
		Schema string `json:"schema_version"`
	}
	if err := json.Unmarshal(raw, &version); err != nil {
		return err
	}
	if version.Schema == "unified_turn.v1" {
		type archived UnifiedTurn
		return json.Unmarshal(raw, (*archived)(u))
	}
	var e UnifiedEnvelope
	if err := json.Unmarshal(raw, &e); err != nil {
		return err
	}
	*u = UnifiedTurn{SchemaVersion: e.SchemaVersion, Reply: e.Reply.Text, Conversation: e.Reply.Conversation, UsedContext: e.Reply.UsedContext, ReplyDependencies: e.Reply.DependsOn, Proposals: []StateProposal{}}
	for _, raw := range e.StateProposals {
		var p StateProposal
		if err := json.Unmarshal(raw, &p); err != nil {
			return err
		}
		u.Proposals = append(u.Proposals, p)
		var err error
		switch p.Kind {
		case "session_intent":
			u.Session = new(SessionIntent)
			err = json.Unmarshal(p.Value, u.Session)
		case "thread_intent":
			u.Thread = new(ThreadIntent)
			err = json.Unmarshal(p.Value, u.Thread)
		case "protocol_intent":
			u.Protocol = new(ProtocolIntent)
			err = json.Unmarshal(p.Value, u.Protocol)
		case "open_loops":
			err = json.Unmarshal(p.Value, &u.OpenLoops)
		case "preferences":
			u.Preferences = new(PrefsPatch)
			err = json.Unmarshal(p.Value, u.Preferences)
		case "risk_read":
			u.RiskRead = new(BrainRiskRead)
			err = json.Unmarshal(p.Value, u.RiskRead)
		}
		if err != nil {
			return err
		}
	}
	for _, raw := range e.BackgroundRequests {
		var r BackgroundRequest
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		u.BackgroundRequests = append(u.BackgroundRequests, r)
	}
	for _, raw := range e.FutureHints {
		var h HintProposal
		if err := json.Unmarshal(raw, &h); err != nil {
			return err
		}
		u.FutureHints = append(u.FutureHints, h)
	}
	return nil
}
