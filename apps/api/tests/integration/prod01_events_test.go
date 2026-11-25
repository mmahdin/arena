package integration

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	aid "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	oidcUsers "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/mssql"
	bookmarks "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/adapters/mssql"
	conversations "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	profiles "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userdomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/productevents"
)

// Requires the existing guarded local disposable SQL Server 2022 harness.
// aiDatabase creates and removes a NEW test database; no shared DB is touched.
func TestPROD01SyntheticProductEventLedger(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	aiMigrate(t, db)
	ctx := context.Background()
	f := aiSetup(t, db)
	count := func(n productevents.Name, subject uuid.UUID) int {
		t.Helper()
		return aiCount(t, db, `SELECT COUNT(*) FROM dbo.product_events WHERE event_name=@p1 AND subject_id=@p2`, string(n), subject)
	}

	// Direct fixtures do not masquerade as registered users. Registration is
	// recorded when the Keycloak identity is first linked to a new user.
	if count(productevents.UserRegistered, f.a) != 0 {
		t.Fatal("seed fixture was counted as registration")
	}
	const issuer = "http://auth.localhost:8180/realms/theia-users"
	syntheticAccounts := oidcUsers.New(db, true)
	registered, err := syntheticAccounts.ResolveIdentity(ctx, auth.User, issuer, "prod01-synthetic-keycloak")
	if err != nil || count(productevents.UserRegistered, registered) != 1 {
		t.Fatal("synthetic Keycloak registration was not recorded", err)
	}
	replayed, err := syntheticAccounts.ResolveIdentity(ctx, auth.User, issuer, "prod01-synthetic-keycloak")
	if err != nil || replayed != registered || count(productevents.UserRegistered, registered) != 1 {
		t.Fatal("returning Keycloak login duplicated registration", err)
	}
	aiExec(t, db, `INSERT dbo.user_profiles(user_id,conversation_style) VALUES(@p1,N'balanced')`, f.a)
	progress := profiles.NewProgressRepository(db)
	if _, _, err := progress.AdvanceStep(ctx, f.a, 1); err != nil {
		t.Fatal(err)
	}
	if _, _, err := progress.AdvanceStep(ctx, f.a, 1); err != nil {
		t.Fatal(err)
	}
	if count(productevents.OnboardingStarted, f.a) != 1 {
		t.Fatal("progress retry counted twice")
	}
	now := time.Now().UTC()
	profile := userdomain.Profile{UserID: f.a, DisplayName: "Synthetic Person", ConversationStyle: userdomain.Balanced, OnboardingCompletedAt: &now}
	if err := profiles.New(db).Save(ctx, profile); err != nil {
		t.Fatal(err)
	}
	if err := profiles.New(db).Save(ctx, profile); err != nil {
		t.Fatal(err)
	}
	if count(productevents.OnboardingCompleted, f.a) != 1 {
		t.Fatal("completion edit counted twice")
	}

	conv, err := conversations.New(db).Create(ctx, f.a)
	if err != nil || count(productevents.ConversationStarted, conv.ID) != 1 {
		t.Fatal("conversation event not committed", err)
	}

	req := f.req()
	admission := admit(t, f, req)
	_ = admit(t, f, req) // duplicate clientTurnId must not emit a second event
	if count(productevents.AITurnAdmitted, admission.TurnID) != 1 {
		t.Fatal("duplicate admission event")
	}
	if _, err = f.store.Complete(ctx, admission, prepared(admission)); err != nil {
		t.Fatal(err)
	}
	if _, err = f.store.Complete(ctx, admission, prepared(admission)); err != nil {
		t.Fatal(err)
	}
	if count(productevents.AITurnCompleted, admission.TurnID) != 1 {
		t.Fatal("completion replay counted twice")
	}
	if err = bookmarks.New(db).Save(ctx, f.b, admission.UserMessageID); !errors.Is(err, httpx.ErrNotFound) || count(productevents.BookmarkSaved, admission.UserMessageID) != 0 {
		t.Fatal("wrong-owner bookmark emitted an event or disclosed the message", err)
	}
	if err = bookmarks.New(db).Save(ctx, f.a, admission.UserMessageID); err != nil {
		t.Fatal(err)
	}
	if err = bookmarks.New(db).Save(ctx, f.a, admission.UserMessageID); err != nil {
		t.Fatal(err)
	}
	if count(productevents.BookmarkSaved, admission.UserMessageID) != 1 {
		t.Fatal("bookmark retry counted twice")
	}

	failed := admit(t, f, f.req())
	if err = f.store.Fail(ctx, failed, aid.ErrGenFailed); err != nil {
		t.Fatal(err)
	}
	if err = f.store.Fail(ctx, failed, aid.ErrGenFailed); err != nil {
		t.Fatal(err)
	}
	if count(productevents.AITurnFailed, failed.TurnID) != 1 {
		t.Fatal("failure replay counted twice")
	}
	if count(productevents.AITurnCompleted, failed.TurnID) != 0 {
		t.Fatal("failed turn counted as complete")
	}

	// A rolled-back event must not become durable. The ledger is transactional.
	tx, err := db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	rolledBack := uuid.New()
	if err = productevents.Record(ctx, tx, f.a, productevents.ConversationStarted, rolledBack); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Rollback(); err != nil {
		t.Fatal(err)
	}
	if count(productevents.ConversationStarted, rolledBack) != 0 {
		t.Fatal("uncommitted event leaked")
	}

	// Real users are deliberately outside PROD-01's synthetic-only scope.
	realUser, err := oidcUsers.New(db, false).ResolveIdentity(ctx, auth.User, issuer, "prod01-non-synthetic-fixture")
	if err != nil {
		t.Fatal(err)
	}
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = productevents.Record(ctx, tx, realUser, productevents.UserRegistered, realUser); err != nil {
		tx.Rollback()
		t.Fatal(err)
	}
	if err = tx.Commit(); err != nil {
		t.Fatal(err)
	}
	if count(productevents.UserRegistered, realUser) != 0 {
		t.Fatal("real user analytics enabled without approval")
	}
	// Even a direct SQL writer must not bypass the synthetic-only boundary.
	_, err = db.ExecContext(ctx, `INSERT dbo.product_events(id,user_id,event_name,subject_id)
        VALUES(NEWID(),@p1,'user_registered',@p1)`, realUser)
	if err == nil {
		t.Fatal("direct SQL inserted a real-user event")
	}
	if count(productevents.UserRegistered, realUser) != 0 {
		t.Fatal("trigger retained invalid event")
	}
	_, err = db.ExecContext(ctx, `UPDATE dbo.product_events SET user_id=@p1,subject_id=@p1
        WHERE event_name='user_registered' AND subject_id=@p2`, realUser, registered)
	if err == nil || count(productevents.UserRegistered, registered) != 1 || count(productevents.UserRegistered, realUser) != 0 {
		t.Fatal("direct SQL moved a product event onto a non-synthetic user")
	}
	_, err = db.ExecContext(ctx, `DELETE dbo.product_events
        WHERE event_name='user_registered' AND subject_id=@p1`, registered)
	if err == nil || count(productevents.UserRegistered, registered) != 1 {
		t.Fatal("physical deletion of a product event was permitted")
	}

	// The read-only reporting view must return aggregates, not user/turn IDs.
	var events int64
	var users int
	err = db.QueryRowContext(ctx, `SELECT event_count,distinct_test_users FROM dbo.v_product_daily_events_synthetic
      WHERE activity_date_utc=CONVERT(date,SYSUTCDATETIME()) AND event_name='ai_turn_admitted'`).Scan(&events, &users)
	if err != nil || events < 2 || users < 1 {
		t.Fatal("aggregated daily reporting invalid", err, events, users)
	}
	var unsafeColumn int
	err = db.QueryRowContext(ctx, `SELECT COUNT(*) FROM sys.columns
      WHERE object_id=OBJECT_ID(N'dbo.v_product_daily_events_synthetic')
        AND name IN('user_id','subject_id','content','prompt','message','email','phone')`).Scan(&unsafeColumn)
	if err != nil || unsafeColumn != 0 {
		t.Fatal("reporting view exposes identifiers", err, unsafeColumn)
	}

	// Subject of a user-scoped event must be that user.
	tx, err = db.BeginTx(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if err = productevents.Record(ctx, tx, f.a, productevents.UserRegistered, uuid.New()); err == nil {
		tx.Rollback()
		t.Fatal("wrong user subject accepted")
	}
	tx.Rollback()

	// Force the event INSERT to fail in this disposable database. The business
	// row must roll back too, rather than committing without its event.
	aiExec(t, db, `CREATE TRIGGER dbo.prod01_test_reject_conversation_event
        ON dbo.product_events AFTER INSERT AS
        IF EXISTS (SELECT 1 FROM inserted WHERE event_name='conversation_started')
            THROW 51042, 'synthetic test forces product event rollback', 1;`)
	before := aiCount(t, db, `SELECT COUNT(*) FROM dbo.conversations WHERE user_id=@p1`, f.a)
	if _, err := conversations.New(db).Create(ctx, f.a); err == nil {
		t.Fatal("conversation creation succeeded despite a failed product event")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.conversations WHERE user_id=@p1`, f.a) != before {
		t.Fatal("business row committed despite a failed product event")
	}
}
