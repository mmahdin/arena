// Package productevents is the transactional, server-only event vocabulary for
// synthetic product analytics. It must never accept arbitrary user properties.
// The owning business adapter calls Record in its existing SQL transaction.
package productevents

import (
	"context"
	"database/sql"
	"errors"

	"github.com/google/uuid"
)

type Name string

const (
	UserRegistered      Name = "user_registered"
	OnboardingStarted   Name = "onboarding_started"
	OnboardingCompleted Name = "onboarding_completed"
	ConversationStarted Name = "conversation_started"
	AITurnAdmitted      Name = "ai_turn_admitted"
	AITurnCompleted     Name = "ai_turn_completed"
	AITurnFailed        Name = "ai_turn_failed"
	BookmarkSaved       Name = "bookmark_saved"
)

var ErrInvalidEvent = errors.New("invalid product event")

func (n Name) Valid() bool {
	switch n {
	case UserRegistered, OnboardingStarted, OnboardingCompleted,
		ConversationStarted, AITurnAdmitted, AITurnCompleted,
		AITurnFailed, BookmarkSaved:
		return true
	default:
		return false
	}
}

func userSubject(n Name) bool {
	return n == UserRegistered || n == OnboardingStarted || n == OnboardingCompleted
}

type executor interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
}

// Record inserts one event exactly once per (type, subject) in the caller's
// transaction. SQL filters out real/inactive users; only synthetic, active
// users are eligible. A duplicate is not another activity event. No payload,
// URL, device metadata or arbitrary string ever reaches the event table.
//
// It deliberately returns insertion errors to the caller: a business write
// must roll back if the durable event cannot be recorded. Migrate first.
func Record(ctx context.Context, tx *sql.Tx, user uuid.UUID, name Name, subject uuid.UUID) error {
	if tx == nil {
		return ErrInvalidEvent
	}
	return record(ctx, tx, user, name, subject)
}

func record(ctx context.Context, exec executor, user uuid.UUID, name Name, subject uuid.UUID) error {
	if exec == nil || user == uuid.Nil || subject == uuid.Nil || !name.Valid() ||
		(userSubject(name) && subject != user) {
		return ErrInvalidEvent
	}
	// UPDLOCK,HOLDLOCK on the uniqueness key prevents concurrent duplicate
	// inserts. The owner eligibility is checked by SQL, never by a caller bit.
	_, err := exec.ExecContext(ctx, `INSERT INTO dbo.product_events (id,user_id,event_name,subject_id)
        SELECT @p1,u.id,@p3,CONVERT(uniqueidentifier,@p4)
        FROM dbo.users AS u
        WHERE u.id=CONVERT(uniqueidentifier,@p2) AND u.is_test=1 AND u.deleted_at IS NULL
          AND NOT EXISTS (
            SELECT 1 FROM dbo.product_events WITH (UPDLOCK,HOLDLOCK)
            WHERE event_name=@p3 AND subject_id=CONVERT(uniqueidentifier,@p4)
          )`, uuid.NewString(), user.String(), string(name), subject.String())
	return err
}
