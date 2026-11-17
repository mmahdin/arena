package contract

type EventType string

const (
	EvTurnCompleted         EventType = "turn_completed"
	EvSessionStarted        EventType = "session_started"
	EvSessionClosed         EventType = "session_closed"
	EvRiskLevelChanged      EventType = "risk_level_changed"
	EvProtocolStatusChanged EventType = "protocol_status_changed"
	EvMemoryDeleted         EventType = "memory_deleted"
	EvMemoryCorrected       EventType = "memory_corrected"
	EvFollowUpScheduled     EventType = "follow_up_scheduled"
	EvReviewFlagged         EventType = "review_flagged" // clinician review queue
	EvDeletionCompleted     EventType = "deletion_completed"
)

type TraceFlags uint32
