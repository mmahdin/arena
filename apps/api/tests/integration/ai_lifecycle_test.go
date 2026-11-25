package integration

import (
	"context"
	"database/sql"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	aidb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/adapters/mssql"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports/contracttest"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

func aiDatabase(t *testing.T) *sql.DB {
	t.Helper()
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("local SQL Server integration opt-in")
	}
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || u.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server required")
	}
	if h := u.Hostname(); h != "127.0.0.1" && h != "localhost" && h != "mssql" {
		t.Fatal("refusing non-local database")
	}
	name := "wellbeing_ai_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := u.Query()
	q.Set("database", name)
	u.RawQuery = q.Encode()
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	if err = database.EnsureLocalDatabase(ctx, u.String()); err != nil {
		t.Fatal(err)
	}
	db, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Close()
		q.Set("database", "master")
		u.RawQuery = q.Encode()
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		master, err := database.Connect(ctx, u.String())
		if err != nil {
			t.Error(err)
			return
		}
		defer master.Close()
		if _, err = master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); err != nil {
			t.Error(err)
		}
	})
	var major int
	if err = db.QueryRowContext(ctx, `SELECT CONVERT(int,SERVERPROPERTY('ProductMajorVersion'))`).Scan(&major); err != nil || major != 16 {
		t.Fatal("requires SQL Server 2022", major, err)
	}
	return db
}
func aiExec(t *testing.T, db *sql.DB, q string, args ...any) {
	t.Helper()
	if _, err := db.ExecContext(context.Background(), q, args...); err != nil {
		t.Fatal(err)
	}
}
func aiCount(t *testing.T, db *sql.DB, q string, args ...any) int {
	t.Helper()
	var n int
	if err := db.QueryRowContext(context.Background(), q, args...).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}
func aiMigrate(t *testing.T, db *sql.DB) {
	t.Helper()
	if err := migrations.Apply(context.Background(), db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
}

type faultState struct {
	*aidb.StateStore
	fail                 bool
	rollbackBeforeCommit bool
}

func (s *faultState) Apply(ctx context.Context, tx *sql.Tx, a c.Admission, p c.PreparedTurn) error {
	if err := s.StateStore.Apply(ctx, tx, a, p); err != nil {
		return err
	}
	if s.fail {
		s.fail = false
		return errors.New("synthetic late state failure")
	}
	return nil
}

type aiFixture struct {
	db                *sql.DB
	store             *convdb.TurnStore
	state             *faultState
	a, b, ca, ca2, cb c.ID
}

func aiSetup(t *testing.T, db *sql.DB) aiFixture {
	f := aiFixture{db: db, a: c.NewID(), b: c.NewID(), ca: c.NewID(), ca2: c.NewID(), cb: c.NewID(), state: &faultState{StateStore: aidb.NewStateStore()}}
	for _, user := range []c.ID{f.a, f.b} {
		aiExec(t, db, `INSERT dbo.users(id,test_key,is_test) VALUES(@p1,@p2,1)`, user, "ai-"+user.String())
	}
	for _, pair := range [][2]c.ID{{f.ca, f.a}, {f.ca2, f.a}, {f.cb, f.b}} {
		aiExec(t, db, `INSERT dbo.conversations(id,user_id) VALUES(@p1,@p2)`, pair[0], pair[1])
	}
	f.store = convdb.NewTurnStore(db, f.state, 2*time.Minute)
	return f
}
func (f aiFixture) req() c.TurnRequest {
	return c.TurnRequest{UserID: f.a, ConversationID: f.ca, ClientTurnID: c.NewID(), Text: "Synthetic input", Origin: c.OriginUser}
}
func prepared(a c.Admission) c.PreparedTurn {
	return c.PreparedTurn{TurnID: a.TurnID, UserMessageID: a.UserMessageID, Text: "Validated synthetic answer", Route: c.RouteNormal, Objective: c.ObjReflect, State: c.StateDelta{TurnID: a.TurnID, UserID: a.UserID, ExpectedVersions: a.Versions, Session: c.SessionPatch{LastObjective: c.ObjReflect, TurnCountInc: 1}}, Trace: c.TurnTraceSummary{BundleVersion: "kb@synthetic"}, Jobs: []c.AsyncJobProposal{{Kind: "memory_write", DedupeKey: a.TurnID.String(), Payload: []byte(`{"schema":"job.v1"}`)}}}
}
func admit(t *testing.T, f aiFixture, r c.TurnRequest) c.Admission {
	t.Helper()
	a, err := f.store.AdmitUserMessage(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	return a
}
func TestAICoreSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	aiMigrate(t, db)
	factory := func(t *testing.T) contracttest.Fixture {
		f := aiSetup(t, db)
		leaseTurns := map[c.ID]c.ID{}
		return contracttest.Fixture{Store: f.store, Finalizer: f.store, Serialiser: f.store, UserA: f.a, UserB: f.b, ConversationA: f.ca, OtherConversationA: f.ca2, ConversationB: f.cb,
			AdvancePastLease: func() {
				aiExec(t, db, `UPDATE dbo.ai_user_turn_leases SET lease_expires_at=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id IN(@p1,@p2)`, f.a, f.b)
			},
			FailNextCompletion: func() { f.state.fail = true },
			LeaseTurn: func(user c.ID) c.ID {
				if id, ok := leaseTurns[user]; ok {
					return id
				}
				r := f.req()
				r.UserID = user
				if user == f.b {
					r.ConversationID = f.cb
				}
				a := admit(t, f, r)
				if err := f.store.Release(context.Background(), ports.LeaseHandle{UserID: user, TurnID: a.TurnID, Token: a.LeaseToken}); err != nil {
					t.Fatal(err)
				}
				leaseTurns[user] = a.TurnID
				return a.TurnID
			},
		}
	}
	contracttest.CheckLifecycle(t, factory)
	contracttest.CheckLeases(t, factory)
	t.Run("queue_recovery_renewal_failure_and_cancel", func(t *testing.T) {
		ctx := context.Background()
		f := aiSetup(t, db)
		r := f.req()
		a := admit(t, f, r)
		q := f.req()
		q.ConversationID = f.ca2
		b := admit(t, f, q)
		if !b.Queued || c.ValidID(b.LeaseToken) {
			t.Fatal("second conversation was not queued")
		}
		duplicate := admit(t, f, r)
		if c.ValidID(duplicate.LeaseToken) {
			t.Fatal("live token disclosed to duplicate worker")
		}
		h := ports.LeaseHandle{UserID: f.a, TurnID: a.TurnID, Token: a.LeaseToken}
		if _, err := f.store.Renew(ctx, h); err != nil {
			t.Fatal(err)
		}
		aiExec(t, db, `UPDATE dbo.ai_user_turn_leases SET lease_expires_at=DATEADD(second,-1,SYSUTCDATETIME()) WHERE user_id=@p1`, f.a)
		if _, err := f.store.Acquire(ctx, f.a, b.TurnID); err == nil {
			t.Fatal("queue bypassed oldest abandoned turn")
		}
		next, err := f.store.Next(ctx, f.a)
		if err != nil || next.ClientTurnID != r.ClientTurnID || next.Text != r.Text {
			t.Fatal("durable recovery request missing", err)
		}
		recovered := admit(t, f, next)
		if recovered.LeaseToken == a.LeaseToken || !c.ValidID(recovered.LeaseToken) {
			t.Fatal("recovery failed")
		}
		if _, err := f.store.Complete(ctx, a, prepared(a)); err == nil {
			t.Fatal("stale worker completed")
		}
		if err := f.store.Fail(ctx, recovered, c.ErrGenFailed); err != nil {
			t.Fatal(err)
		}
		if c.ValidID(admit(t, f, r).LeaseToken) {
			t.Fatal("failed duplicate implicitly retried")
		}
		b = admit(t, f, q)
		if b.Queued {
			t.Fatal("queue did not advance")
		}
		if err := f.store.Cancel(ctx, b); err != nil {
			t.Fatal(err)
		}
		if err := f.store.Retry(ctx, f.a, r.ClientTurnID); err != nil {
			t.Fatal(err)
		}
		a = admit(t, f, r)
		result, err := f.store.Complete(ctx, a, prepared(a))
		if err != nil {
			t.Fatal(err)
		}
		again, err := f.store.Complete(ctx, a, prepared(a))
		if err != nil || again.Text != result.Text || *again.AssistantMessageID != *result.AssistantMessageID {
			t.Fatal("completion acknowledgement replay failed", err)
		}
	})
	t.Run("atomic_state_and_late_rollback", func(t *testing.T) {
		f := aiSetup(t, db)
		a := admit(t, f, f.req())
		p := prepared(a)
		thread := c.NewID()
		p.State.Thread = &c.ThreadPatch{ThreadID: thread, Create: &c.ThreadInit{Topic: "Synthetic topic"}, Slots: []c.SlotUpdate{{Slot: "situation", Value: "Synthetic", Status: c.EpReported, Evidence: []c.ID{a.UserMessageID}}}}
		p.State.Session.FocusThread = &thread
		pi := c.NewID()
		status := c.PIActive
		p.State.Protocol = &c.ProtocolPatch{InstanceID: pi, Create: &c.ProtocolInstanceInit{ProtocolID: "synthetic", Version: "1", ThreadID: &thread, OfferedAt: time.Now()}, NewStatus: &status, ConsentAt: ptr(time.Now()), ConsentMessageID: &a.UserMessageID}
		p.State.OpenLoops = []c.OpenLoopOp{{Op: "add", Loop: c.OpenLoop{ID: c.NewID(), Kind: "synthetic", Text: "synthetic"}}}
		p.State.FollowUps = []c.FollowUpOp{{Op: "schedule", FollowUpID: c.NewID(), Reason: "synthetic", DueAt: time.Now().Add(time.Hour)}}
		admissionJobs := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a)
		f.state.fail = true
		if _, err := f.store.Complete(context.Background(), a, p); err == nil {
			t.Fatal("injected late failure accepted")
		}
		for _, table := range []string{"ai_threads", "ai_protocol_instances", "ai_turn_events", "ai_turn_context", "ai_jobs", "ai_turn_traces", "ai_outbox", "ai_open_loops", "ai_follow_ups"} {
			want := 0
			if table == "ai_jobs" {
				want = admissionJobs
			}
			if aiCount(t, db, "SELECT COUNT(*) FROM dbo."+table+" WHERE user_id=@p1", f.a) != want {
				t.Fatal("partial write", table)
			}
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_sessions WHERE id=@p1 AND version=1 AND turn_count=0`, a.SessionID) != 1 {
			t.Fatal("session update escaped rollback")
		}
		bad := p
		bad.State.ExpectedVersions.User++
		if _, err := f.store.Complete(context.Background(), a, bad); err == nil {
			t.Fatal("stale version accepted")
		}
		if _, err := f.store.Complete(context.Background(), a, p); err != nil {
			t.Fatal(err)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_follow_ups WHERE user_id=@p1 AND status=N'skipped' AND skip_reason=N'not_opted_in'`, f.a) != 1 {
			t.Fatal("unconsented proactive followup scheduled")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_protocol_instances WHERE id=@p1 AND status=N'active' AND consent_message_id=@p2`, pi, a.UserMessageID) != 1 {
			t.Fatal("protocol not persisted")
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_turn_traces WHERE turn_id=@p1 AND summary_json NOT LIKE N'%synthetic answer%' AND payload_enc IS NULL`, a.TurnID) != 1 {
			t.Fatal("trace leaked text")
		}
	})
	t.Run("two_users_can_complete_concurrently", func(t *testing.T) {
		f := aiSetup(t, db)
		r := f.req()
		a := admit(t, f, r)
		r.UserID = f.b
		r.ConversationID = f.cb
		r.ClientTurnID = c.NewID()
		b := admit(t, f, r)
		var wg sync.WaitGroup
		for _, a := range []c.Admission{a, b} {
			wg.Add(1)
			go func(a c.Admission) {
				defer wg.Done()
				if _, err := f.store.Complete(context.Background(), a, prepared(a)); err != nil {
					var ce *c.Error
					if errors.As(err, &ce) {
						t.Errorf("%v: %v", err, ce.Cause)
					} else {
						for cause := err; cause != nil; cause = errors.Unwrap(cause) {
							t.Logf("%T: %v", cause, cause)
						}
						t.Error("completion failed")
					}
				}
			}(a)
		}
		wg.Wait()
	})
	t.Run("owner_constraints_and_privacy", func(t *testing.T) { testAIPrivacy(t, db) })
}
func ptr[T any](v T) *T { return &v }

func TestAICoreUpgradeSQLServer(t *testing.T) {
	db := aiDatabase(t)
	dir := t.TempDir()
	files, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "0017_" {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, filepath.Base(file)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = migrations.Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	before := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`)
	user := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	conversationID := uuid.New()
	// Seed with 0016 SQL. The current conversation adapter requires the product event ledger.
	aiExec(t, db, `INSERT dbo.conversations(id,user_id) VALUES(@p1,@p2)`, conversationID, user)
	turn := c.NewID()
	aiExec(t, db, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status,lease_id,lease_expires_at) VALUES(@p1,@p2,@p3,HASHBYTES('SHA2_256','Synthetic unfinished legacy input'),N'processing',NEWID(),DATEADD(minute,2,SYSUTCDATETIME()))`, turn, conversationID, user)
	aiMigrate(t, db)
	aiMigrate(t, db)
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`) != before {
		t.Fatal("migration lost history")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.turns WHERE id=@p1 AND client_turn_id=id AND lifecycle_version=0 AND status=N'processing'`, turn) != 1 {
		t.Fatal("legacy in-flight reservation changed")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.tables WHERE name=N'daily_reflections'`) != 1 {
		t.Fatal("unapproved reflection data retirement")
	}
	// Checksum checks still cover every applied old file.
	raw, err := os.ReadFile(filepath.Join(dir, "0001_initial_schema.sql"))
	if err != nil {
		t.Fatal(err)
	}
	if err = os.WriteFile(filepath.Join(dir, "0001_initial_schema.sql"), append(raw, []byte("\n-- tampered\n")...), 0600); err != nil {
		t.Fatal(err)
	}
	if err = migrations.Apply(context.Background(), db, dir); err == nil {
		t.Fatal("checksum tamper accepted")
	}
}

func testAIPrivacy(t *testing.T, db *sql.DB) {
	ctx := context.Background()
	f := aiSetup(t, db)
	a := admit(t, f, f.req())
	if _, err := f.store.Complete(ctx, a, prepared(a)); err != nil {
		t.Fatal(err)
	}
	req := f.req()
	req.UserID = f.b
	req.ConversationID = f.cb
	b := admit(t, f, req)
	item := c.NewID()
	aiExec(t, db, `INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,source_json) VALUES(@p1,@p2,N'fact',N'synthetic retained snippet',N'reported',N'{}')`, item, f.a)
	for _, q := range []string{
		`INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,evidence_at) VALUES(@p1,@p2,N'message',@p3,N'supports',SYSUTCDATETIME())`,
		`INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes) VALUES(NEWID(),@p1,N'message',@p3,N'synthetic',1,0x0000803F)`,
	} {
		if _, err := db.ExecContext(ctx, q, f.a, item, b.UserMessageID); err == nil {
			t.Fatal("cross-owner polymorphic reference accepted")
		}
	}
	aiExec(t, db, `INSERT dbo.ai_memory_evidence(user_id,item_id,ref_kind,ref_id,role,session_id,evidence_at) VALUES(@p1,@p2,N'message',@p3,N'supports',@p4,SYSUTCDATETIME());
 INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes) VALUES(NEWID(),@p1,N'memory',@p2,N'synthetic',1,0x0000803F)`, f.a, item, a.UserMessageID, a.SessionID)
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_embeddings(id,user_id,ref_kind,ref_id,model_id,dim,vector_bytes) VALUES(NEWID(),@p1,N'memory',@p2,N'bad-dim',2,0x0000803F)`, f.a, item); err == nil {
		t.Fatal("bad embedding dimension accepted")
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_turn_context(turn_id,user_id,session_id) VALUES(@p1,@p2,@p3)`, b.TurnID, f.a, a.SessionID); err == nil {
		t.Fatal("cross-owner turn context accepted")
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_protocol_instances(id,user_id,protocol_id,protocol_version,status) VALUES(NEWID(),@p1,N'synthetic',N'1',N'active')`, f.a); err == nil {
		t.Fatal("active protocol without consent accepted")
	}
	if err := convdb.New(db).Delete(ctx, f.b, f.ca); err == nil {
		t.Fatal("foreign deletion accepted")
	}
	otherReq := f.req()
	otherReq.ConversationID = f.ca2
	other := admit(t, f, otherReq)
	if err := convdb.New(db).Delete(ctx, f.a, f.ca); err != nil {
		t.Fatal(err)
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_items WHERE id=@p1 AND deleted_at IS NULL`, item) != 1 ||
		aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_memory_evidence WHERE item_id=@p1`, item) != 1 ||
		aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_embeddings WHERE ref_id=@p1`, item) != 1 {
		t.Fatal("soft deletion lost derived data")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1`, b.UserMessageID) != 1 {
		t.Fatal("other user's data lost")
	}
	if _, err := f.store.Complete(ctx, a, prepared(a)); err == nil {
		t.Fatal("worker resurrected deleted conversation")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1`, other.UserMessageID) != 1 {
		t.Fatal("other raw history deleted")
	}
	if _, err := f.store.Complete(ctx, other, prepared(other)); err != nil {
		t.Fatal("unrelated active turn was affected", err)
	}
	// No account erasure is implemented; raw deletes cannot cascade away data.
	if _, err := db.ExecContext(ctx, `DELETE dbo.users WHERE id=@p1`, f.b); err == nil {
		t.Fatal("physical account deletion accepted")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_sessions WHERE user_id=@p1`, f.b) != 1 {
		t.Fatal("account data was lost")
	}
}

func (s *faultState) Release(ctx context.Context, tx *sql.Tx, h ports.LeaseHandle) error {
	if err := s.StateStore.Release(ctx, tx, h); err != nil {
		return err
	}
	if s.rollbackBeforeCommit {
		s.rollbackBeforeCommit = false
		return tx.Rollback()
	}
	return nil
}
func TestAICoreRacesSQLServer(t *testing.T) {
	db := aiDatabase(t)
	aiMigrate(t, db)
	ctx := context.Background()
	t.Run("distinct_turns_across_repository_instances", func(t *testing.T) {
		f := aiSetup(t, db)
		second := convdb.NewTurnStore(db, aidb.NewStateStore(), time.Minute)
		var wg sync.WaitGroup
		admissions := make(chan c.Admission, 16)
		for i := 0; i < 16; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				req := f.req()
				store := f.store
				if i%2 == 1 {
					req.ConversationID = f.ca2
					store = second
				}
				a, err := store.AdmitUserMessage(ctx, req)
				if err != nil {
					t.Error(err)
					return
				}
				admissions <- a
			}(i)
		}
		wg.Wait()
		close(admissions)
		active := 0
		count := 0
		for a := range admissions {
			count++
			if c.ValidID(a.LeaseToken) {
				active++
			}
		}
		if active != 1 || count != 16 {
			t.Fatalf("active=%d admitted=%d", active, count)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations c ON c.id=m.conversation_id WHERE c.user_id=@p1`, f.a) != 16 {
			t.Fatal("lost queued raw input")
		}
	})
	t.Run("final_commit_error_does_not_report_success", func(t *testing.T) {
		f := aiSetup(t, db)
		r := f.req()
		a := admit(t, f, r)
		admissionJobs := aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a)
		f.state.rollbackBeforeCommit = true
		if _, err := f.store.Complete(ctx, a, prepared(a)); !errors.Is(err, sql.ErrTxDone) {
			t.Fatal("expected failed final commit", err)
		}
		result, err := f.store.GetTurn(ctx, f.a, r.ClientTurnID)
		if err != nil || result.Status != c.TurnRunning || result.AssistantMessageID != nil {
			t.Fatal("failed commit exposed success", err)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.ai_jobs WHERE user_id=@p1`, f.a) != admissionJobs {
			t.Fatal("failed commit left jobs")
		}
		if _, err = f.store.Complete(ctx, a, prepared(a)); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("queued_cancellation_preserves_input", func(t *testing.T) {
		f := aiSetup(t, db)
		admit(t, f, f.req())
		r := f.req()
		a := admit(t, f, r)
		if !a.Queued {
			t.Fatal("fixture not queued")
		}
		if err := f.store.CancelQueued(ctx, f.b, r.ClientTurnID); err == nil {
			t.Fatal("foreign cancel accepted")
		}
		if err := f.store.CancelQueued(ctx, f.a, r.ClientTurnID); err != nil {
			t.Fatal(err)
		}
		got, err := f.store.GetTurn(ctx, f.a, r.ClientTurnID)
		if err != nil || got.Status != c.TurnCancelled {
			t.Fatal(got.Status, err)
		}
		if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages WHERE id=@p1`, a.UserMessageID) != 1 {
			t.Fatal("cancel lost input")
		}
	})
}

func TestAICoreMigrationSafeguardSQLServer(t *testing.T) {
	db := aiDatabase(t)
	dir := t.TempDir()
	files, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "0017_" {
			continue
		}
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(filepath.Join(dir, filepath.Base(file)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err = migrations.Apply(context.Background(), db, dir); err != nil {
		t.Fatal(err)
	}
	// Deliberately inconsistent historical data must stop the migration, never be
	// deduplicated or deleted automatically to make the new unique index pass.
	aiExec(t, db, `INSERT dbo.messages(id,conversation_id,turn_id,role,content) SELECT TOP(1) NEWID(),conversation_id,turn_id,role,N'synthetic duplicate' FROM dbo.messages`)
	before := aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`)
	if err = migrations.Apply(context.Background(), db, "../../db/migrations"); err == nil {
		t.Fatal("invalid historical data accepted")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.messages`) != before {
		t.Fatal("migration destroyed historical rows")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM dbo.schema_migrations WHERE name LIKE N'0017_%'`) != 0 {
		t.Fatal("failed migration recorded as applied")
	}
	if aiCount(t, db, `SELECT COUNT(*) FROM sys.columns WHERE object_id=OBJECT_ID(N'dbo.turns') AND name=N'client_turn_id'`) != 0 {
		t.Fatal("migration expansion escaped rollback")
	}
}
