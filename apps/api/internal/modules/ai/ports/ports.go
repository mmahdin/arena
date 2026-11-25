package ports

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"time"
)

// ---- Admission: NEW AI Core Gateway, backed by Conversation-owned storage ----
type Gateway interface {
	Admit(ctx context.Context, req contract.TurnRequest) (contract.Admission, error)
	Release(ctx context.Context, a contract.Admission, outcome TurnOutcome) error
}

// Consumer-owned storage port. One physical dbo.messages/dbo.turns source of truth.
type ConversationStore interface {
	ValidateOwner(ctx context.Context, userID, conversationID contract.ID) error
	AdmitUserMessage(ctx context.Context, req contract.TurnRequest) (contract.Admission, error)
	Recent(ctx context.Context, userID, conversationID contract.ID, limit int) ([]contract.Message, error)
	GetTurn(ctx context.Context, userID, clientTurnID contract.ID) (contract.TurnResult, error)
}

// Finalizing is an adapter-level SQL Server Unit of Work. Keep *sql.Tx
// private to persistence adapters; it does NOT run across LLM calls.
type TurnFinalizer interface {
	Complete(ctx context.Context, a contract.Admission, p contract.PreparedTurn) (contract.TurnResult, error)
	Fail(ctx context.Context, a contract.Admission, code contract.Code) error
}

// A leased per-user turn coordinator (may use MSSQL lease rows).
type TurnSerialiser interface {
	Acquire(ctx context.Context, userID contract.ID, turnID contract.ID) (LeaseHandle, error)
	Release(ctx context.Context, lease LeaseHandle) error
}
type TurnOutcome string // completed | failed | cancelled | queued
type LeaseHandle struct {
	UserID, TurnID, Token contract.ID
	ExpiresAt             time.Time
}

// ----- Safety -----
type SafetyGate interface {
	Assess(ctx context.Context, in SafetyInput) (contract.SafetySignal, error) // never returns an error that lowers risk; errors map to Degraded=true elevated
}
type SafetyInput struct {
	Text   string
	Recent []contract.MsgView
	Risk   contract.RiskState
	Lang   string
}

type CrisisPlanner interface {
	Plan(ctx context.Context, sig contract.SafetySignal, locale string, risk contract.RiskState) (contract.CrisisPlan, error)
}

// ---- Context ----
type ContextAssembler interface {
	Build(ctx context.Context, req contract.TurnRequest, sig contract.SafetySignal, retr contract.RetrievalResult) (contract.WorkingContext, StateSnapshot, error)
}

// StateSnapshot carries versions and raw state needed later by the committer (not sent to LLMs).
type StateSnapshot struct {
	Versions contract.Versions
	Session  contract.SessionRecord
	Focus    *contract.ThreadRecord
	Threads  []contract.ThreadRecord
	Protocol *contract.ProtocolInstance
	User     contract.UserState
}

type Retriever interface {
	Retrieve(ctx context.Context, req contract.RetrievalRequest) (contract.RetrievalResult, error) // errors => caller sets Degraded
}

// ProtocolEngine enforces persisted exercise state, consent and transitions.
type ProtocolEngine interface {
	Reference() contract.BundleRef
	ResolveTurn(ctx context.Context, in contract.ProtocolResolutionInput) (contract.ProtocolStepSpec, contract.ProtocolPatch, error)
}

type Validator interface {
	StreamGuard(plan contract.ResponsePlan, risk contract.RiskLevel) StreamGuard // per-turn, stateful
	Finalize(ctx context.Context, text string, plan contract.ResponsePlan, wc contract.WorkingContext, risk contract.RiskLevel, crisis *contract.CrisisPlan) (contract.ValidationReport, error)
}
type StreamGuard interface {
	Push(tok string) (allow bool, v *contract.Violation)
}

// ---- State ----
type StateCommitter interface {
	// Validates the AI state changes; final DB writes occur through TurnFinalizer
	// inside the same transaction as the validated assistant message.
	ValidateDelta(ctx context.Context, d contract.StateDelta) (contract.StateDelta, []contract.DeltaRejection, error)
	Reconcile(ctx context.Context, userID contract.ID) (ReconcileReport, error)
}

// The MSSQL TurnFinalizer applies this validated StateDelta with the same
// transaction as the assistant message and turn completion.

// ---- Memory ----
type MemoryWriter interface {
	Propose(ctx context.Context, in MemWriterInput) (contract.MemoryWriterOutput, contract.LLMUsage, error)
}
type MemWriterInput struct {
	Research  *BackgroundResearch
	UserID    contract.ID
	Messages  []contract.Message
	Analysis  contract.TurnAnalysis
	Neighbors []contract.MemoryItem
	Claims    []contract.UserClaim
	Bundle    contract.BundleRef
}

type MemoryCommitter interface {
	Apply(ctx context.Context, userID contract.ID, ops []contract.MemoryOpProposal, origin contract.ProvenanceSrc) (MemoryApplyReport, error)
	Correct(ctx context.Context, userID contract.ID, target contract.ID, kind CorrectionKind) error // synchronous path
	// Delete changes visibility only; retain original content and related records.
	// Permanent erasure and AI-use eligibility are separate future policies.
	Delete(ctx context.Context, userID contract.ID, scope DeletionScope) (DeletionReport, error)
	Maintain(ctx context.Context, userID contract.ID, now time.Time) (MaintenanceReport, error)
}
type CorrectionKind string // "contest" | "reject" | "supersede_pending"
type DeletionScope struct {
	ItemIDs, MessageIDs []contract.ID
	Category            string
	All                 bool
	Reason              string
}

type Reflector interface {
	Reflect(ctx context.Context, in ReflectionInput) (contract.ReflectionResult, contract.LLMUsage, error)
}
type ReflectionInput struct {
	Research         *BackgroundResearch
	UserID, ThreadID contract.ID
	Trigger          string
	PrevFormulation  *contract.FormulationDoc
	Items            []contract.MemoryItem
	Evidence         []contract.Message
	Framework        string
	Bundle           contract.BundleRef
}

// ---- Time ----
type Scheduler interface {
	Apply(ctx context.Context, ops []contract.FollowUpOp) error
	Due(ctx context.Context, now time.Time, limit int) ([]contract.FollowUp, error)
}

// ---- Cross-cutting ----
type BundleRegistry interface {
	Active(ctx context.Context, userID contract.ID) (Bundle, error)
} // pinned per user cohort/flag
type Tracer interface {
	Begin(turnID contract.ID, meta contract.CallMeta) TurnTrace
}
type TurnTrace interface {
	Record(step string, v any) // redaction is applied on write
	AddUsage(u contract.LLMUsage)
	SetRoute(r contract.Route)
	Fail(step string, err error)
	Finish(ctx context.Context)
}
type Clock interface{ Now() time.Time }

// TurnRecovery is the WP-02 persistence recovery contract. Authorization is
// supplied by the caller. Retry is explicit; ordinary duplicate admission does
// not restart a terminal turn. No background worker is started by these ports.
type TurnRecovery interface {
	Renew(context.Context, LeaseHandle) (LeaseHandle, error)
	Cancel(context.Context, contract.Admission) error
	CancelQueued(context.Context, contract.ID, contract.ID) error // user, client turn
	Retry(context.Context, contract.ID, contract.ID) error        // user, client turn
	Next(context.Context, contract.ID) (contract.TurnRequest, error)
}
