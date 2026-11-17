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
	aiapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/policy"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	convapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/application"
	devdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/devsession/adapters/mssql"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	userdomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Each run owns a new database; it never resets the developer's wellbeing data.
func TestSQLServerFoundation(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("set WELLBEING_INTEGRATION=true with local DATABASE_URL to test SQL Server")
	}
	ctx := context.Background()
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || u.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("integration requires APP_ENV=local and a sqlserver DATABASE_URL")
	}
	name := "wellbeing_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := u.Query()
	q.Set("database", name)
	u.RawQuery = q.Encode()
	if err := database.EnsureLocalDatabase(ctx, u.String()); err != nil {
		t.Fatal(err)
	}
	q.Set("database", "master")
	masterURL := *u
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, err := master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); err != nil {
			t.Errorf("cleanup isolated test database: %v", err)
		}
	}()
	db, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir := t.TempDir()
	files, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil || len(files) == 0 {
		t.Fatal("missing migrations", err)
	}
	for _, file := range files {
		raw, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, filepath.Base(file)), raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("upgrade pending turn preserves existing history", func(t *testing.T) {
		previous := t.TempDir()
		for _, file := range files {
			if filepath.Base(file) >= "0006_" {
				continue
			}
			raw, err := os.ReadFile(file)
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(previous, filepath.Base(file)), raw, 0600); err != nil {
				t.Fatal(err)
			}
		}
		if err := migrations.Apply(ctx, db, previous); err != nil {
			t.Fatal(err)
		}
		turn := uuid.New()
		if _, err := db.ExecContext(ctx, `INSERT INTO dbo.turns(id,conversation_id,user_id,input_hash,status)
            SELECT CONVERT(uniqueidentifier,@p1),id,user_id,HASHBYTES('SHA2_256','upgrade synthetic'),N'processing'
            FROM dbo.conversations WHERE id='a0000000-0000-4000-8000-000000000001'`, turn.String()); err != nil {
			t.Fatal(err)
		}
		if err := migrations.Apply(ctx, db, dir); err != nil {
			t.Fatal(err)
		}
		var reclaimable int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns WHERE id=CONVERT(uniqueidentifier,@p1)
            AND lease_id IS NOT NULL AND lease_expires_at<=SYSUTCDATETIME()`, turn.String()).Scan(&reclaimable); err != nil || reclaimable != 1 {
			t.Fatalf("upgrade did not make old processing turn reclaimable: count=%d error=%v", reclaimable, err)
		}
		var invalidCompleted int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns WHERE status=N'completed'
            AND (lease_id IS NOT NULL OR lease_expires_at IS NOT NULL)`).Scan(&invalidCompleted); err != nil || invalidCompleted != 0 {
			t.Fatalf("upgrade changed completed turns: count=%d error=%v", invalidCompleted, err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE dbo.turns SET lease_expires_at=CONVERT(datetime2(3),'2000-01-01') WHERE id=CONVERT(uniqueidentifier,@p1)`, turn.String()); err != nil {
			t.Fatal(err)
		}
	})
	t.Run("fresh and repeat migrations", func(t *testing.T) {
		for i := 0; i < 2; i++ {
			if err := migrations.Apply(ctx, db, dir); err != nil {
				t.Fatal(err)
			}
		}
		var users, conversations, messages int
		if err := db.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM dbo.users),(SELECT COUNT(*) FROM dbo.conversations),(SELECT COUNT(*) FROM dbo.messages)").Scan(&users, &conversations, &messages); err != nil {
			t.Fatal(err)
		}
		if users != 3 || conversations != 3 || messages != 6 {
			t.Fatalf("unexpected seed counts: %d users, %d conversations, %d messages", users, conversations, messages)
		}
	})
	t.Run("onboarding profile lifecycle, persisted immutability and user isolation", func(t *testing.T) {
		repo := userdb.New(db)
		svc := userapp.New(repo)
		a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
		beforeA, err := svc.Get(ctx, a)
		if err != nil || beforeA.OnboardingCompletedAt == nil {
			t.Fatalf("seeded tester A missing: %+v %v", beforeA, err)
		}
		beforeB, err := svc.Get(ctx, b)
		if err != nil || beforeB.OnboardingCompletedAt != nil || beforeB.DisplayName != "" {
			t.Fatalf("tester B must start onboarding: %+v %v", beforeB, err)
		}
		if _, err := svc.Update(ctx, b, " ", userdomain.Balanced); err == nil {
			t.Fatal("empty name completed onboarding")
		}
		if current, err := svc.Get(ctx, b); err != nil || current.OnboardingCompletedAt != nil {
			t.Fatalf("invalid request mutated profile: %+v %v", current, err)
		}
		first, err := svc.Update(ctx, b, "  می\u200cنا  ", userdomain.Guidance)
		if err != nil || first.DisplayName != "می\u200cنا" || first.ConversationStyle != userdomain.Guidance || first.OnboardingCompletedAt == nil {
			t.Fatalf("first onboarding save: %+v %v", first, err)
		}
		second, err := svc.Update(ctx, b, "نیلو", userdomain.Listening)
		if err != nil || !second.OnboardingCompletedAt.Equal(*first.OnboardingCompletedAt) {
			t.Fatalf("editing reset onboarding: %+v %v", second, err)
		}
		// Simulate two stale concurrent first saves; SQL should preserve the first completion.
		later := *first.OnboardingCompletedAt
		later = later.Add(time.Hour)
		second.OnboardingCompletedAt = &later
		if err := repo.Save(ctx, second); err != nil {
			t.Fatal(err)
		}
		persisted, err := svc.Get(ctx, b)
		if err != nil || !persisted.OnboardingCompletedAt.Equal(*first.OnboardingCompletedAt) {
			t.Fatalf("SQL did not preserve first completion: %+v %v", persisted, err)
		}
		afterA, err := svc.Get(ctx, a)
		if err != nil || afterA.DisplayName != beforeA.DisplayName || !afterA.OnboardingCompletedAt.Equal(*beforeA.OnboardingCompletedAt) {
			t.Fatalf("cross-user profile modification: %+v %v", afterA, err)
		}
		if _, err := svc.Get(ctx, uuid.New()); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatalf("unknown user: %v", err)
		}
	})
	t.Run("local session persistence, expiry and revocation", func(t *testing.T) {
		repo := devdb.New(db)
		a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
		sid := uuid.New()
		if ok, err := repo.CreateSession(ctx, uuid.New(), uuid.New(), time.Now().Add(time.Hour)); err != nil || ok {
			t.Fatalf("unknown user issued session: %v %v", ok, err)
		}
		if ok, err := repo.CreateSession(ctx, a, sid, time.Now().Add(time.Hour)); err != nil || !ok {
			t.Fatalf("test user session not created: %v %v", ok, err)
		}
		if ok, err := repo.IsActive(ctx, a, sid); err != nil || !ok {
			t.Fatalf("valid session rejected: %v %v", ok, err)
		}
		if ok, err := repo.IsActive(ctx, b, sid); err != nil || ok {
			t.Fatalf("foreign session accepted: %v %v", ok, err)
		}
		if err := repo.RevokeSession(ctx, b, sid); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.IsActive(ctx, a, sid); err != nil || !ok {
			t.Fatalf("wrong-owner revoke affected session: %v %v", ok, err)
		}
		if err := repo.RevokeSession(ctx, a, sid); err != nil {
			t.Fatal(err)
		}
		if ok, err := repo.IsActive(ctx, a, sid); err != nil || ok {
			t.Fatalf("revoked session accepted: %v %v", ok, err)
		}
		past := uuid.New()
		if ok, err := repo.CreateSession(ctx, a, past, time.Now().Add(-time.Minute)); err != nil || !ok {
			t.Fatalf("expired session fixture: %v %v", ok, err)
		}
		if ok, err := repo.IsActive(ctx, a, past); err != nil || ok {
			t.Fatalf("expired session accepted: %v %v", ok, err)
		}
	})
	t.Run("ownership replay ordering and soft deletion", func(t *testing.T) {
		repo := convdb.New(db)
		a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
		c, err := repo.Create(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := repo.Messages(ctx, b, c.ID, 100); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatalf("foreign read: %v", err)
		}
		if err := repo.Delete(ctx, b, c.ID); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatalf("foreign delete: %v", err)
		}
		turn := uuid.New()
		if _, err := repo.ClaimTurn(ctx, b, c.ID, turn, "synthetic"); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatalf("foreign write: %v", err)
		}
		firstClaim, err := repo.ClaimTurn(ctx, a, c.ID, turn, "synthetic")
		if err != nil || firstClaim.LeaseID == uuid.Nil {
			t.Fatalf("first claim: %+v %v", firstClaim, err)
		}
		claim, err := repo.ClaimTurn(ctx, a, c.ID, turn, "synthetic")
		if err != nil || !claim.Pending {
			t.Fatalf("pending retry: %+v %v", claim, err)
		}
		// SQL Server renders uniqueidentifier text with uppercase hex. Compare
		// lease UUID values, not the casing of their textual representations.
		firstClaim.LeaseID = uuid.MustParse("abcdefab-cdef-4abc-8def-abcdefabcdef")
		if _, err := db.ExecContext(ctx, `UPDATE dbo.turns SET lease_id=CONVERT(uniqueidentifier,@p1) WHERE id=CONVERT(uniqueidentifier,@p2)`, firstClaim.LeaseID.String(), turn.String()); err != nil {
			t.Fatal(err)
		}
		result, err := repo.CompleteTurn(ctx, a, c.ID, turn, firstClaim.LeaseID, "synthetic", "synthetic reply", "supportive-v1")
		if err != nil {
			t.Fatal(err)
		}
		claim, err = repo.ClaimTurn(ctx, a, c.ID, turn, "synthetic")
		if err != nil || !claim.Existing || claim.AssistantMessageID != result.AssistantMessageID {
			t.Fatalf("completed retry: %+v %v", claim, err)
		}
		if _, err := repo.ClaimTurn(ctx, a, c.ID, turn, "changed"); !errors.Is(err, httpx.ErrInvalid) {
			t.Fatalf("changed payload retry: %v", err)
		}
		// Force the millisecond tie that made UUID-based ordering nondeterministic.
		if _, err := db.ExecContext(ctx, "UPDATE dbo.messages SET created_at='2026-01-01' WHERE conversation_id=CONVERT(uniqueidentifier,@p1)", c.ID.String()); err != nil {
			t.Fatal(err)
		}
		messages, err := repo.Messages(ctx, a, c.ID, 100)
		if err != nil || len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
			t.Fatalf("message ordering: %+v %v", messages, err)
		}
		if err := repo.Delete(ctx, a, c.ID); err != nil {
			t.Fatal(err)
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM dbo.turns WHERE id=CONVERT(uniqueidentifier,@p1))+(SELECT COUNT(*) FROM dbo.messages WHERE turn_id=CONVERT(uniqueidentifier,@p1))", turn.String()).Scan(&count); err != nil || count != 3 {
			t.Fatalf("preserved turn and messages: count=%d error=%v", count, err)
		}
	})
	t.Run("expired processing turn recovery is fenced and replayed", func(t *testing.T) {
		repo := convdb.New(db)
		a := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		b := uuid.MustParse("22222222-2222-4222-8222-222222222222")
		c, err := repo.Create(ctx, a)
		if err != nil {
			t.Fatal(err)
		}
		defer repo.Delete(ctx, a, c.ID)
		tid := uuid.New()
		first, err := repo.ClaimTurn(ctx, a, c.ID, tid, "synthetic recovery")
		if err != nil || first.LeaseID == uuid.Nil {
			t.Fatalf("initial lease: %+v %v", first, err)
		}
		if _, err := repo.ClaimTurn(ctx, b, c.ID, tid, "synthetic recovery"); !errors.Is(err, httpx.ErrNotFound) {
			t.Fatalf("foreign claim: %v", err)
		}
		if _, err := repo.ClaimTurn(ctx, a, c.ID, tid, "different input"); !errors.Is(err, httpx.ErrInvalid) {
			t.Fatalf("changed input: %v", err)
		}
		if _, err := db.ExecContext(ctx, `UPDATE dbo.turns SET lease_expires_at=DATEADD(second,-1,SYSUTCDATETIME()) WHERE id=CONVERT(uniqueidentifier,@p1)`, tid.String()); err != nil {
			t.Fatal(err)
		}
		type attempt struct {
			leaseID uuid.UUID
			pending bool
			err     error
		}
		// Two simultaneous retry requests may race; exactly one wins the
		// expired lease. The other must observe a pending turn.
		start := make(chan struct{})
		var wg sync.WaitGroup
		results := make(chan attempt, 2)
		for i := 0; i < 2; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				claim, e := repo.ClaimTurn(ctx, a, c.ID, tid, "synthetic recovery")
				results <- attempt{leaseID: claim.LeaseID, pending: claim.Pending, err: e}
			}()
		}
		close(start)
		wg.Wait()
		close(results)
		var recovered uuid.UUID
		winners, pending := 0, 0
		for r := range results {
			if r.err != nil {
				t.Fatal(r.err)
			}
			if r.leaseID != uuid.Nil {
				winners++
				recovered = r.leaseID
			} else if r.pending {
				pending++
			}
		}
		if winners != 1 || pending != 1 || recovered == first.LeaseID {
			t.Fatalf("reclaim race: winners=%d pending=%d", winners, pending)
		}
		if err := repo.ReleaseTurn(ctx, a, c.ID, tid, first.LeaseID); err != nil {
			t.Fatal(err)
		}
		if _, err := repo.CompleteTurn(ctx, a, c.ID, tid, first.LeaseID, "synthetic recovery", "old reply", "v1"); !errors.Is(err, httpx.ErrConflict) {
			t.Fatalf("stale worker completed: %v", err)
		}
		stillPending, err := repo.ClaimTurn(ctx, a, c.ID, tid, "synthetic recovery")
		if err != nil || !stillPending.Pending {
			t.Fatalf("stale release removed new lease: %+v %v", stillPending, err)
		}
		completed, err := repo.CompleteTurn(ctx, a, c.ID, tid, recovered, "synthetic recovery", "new reply", "v2")
		if err != nil {
			t.Fatal(err)
		}
		again, err := repo.ClaimTurn(ctx, a, c.ID, tid, "synthetic recovery")
		if err != nil || !again.Existing || again.AssistantMessageID != completed.AssistantMessageID || again.Reply != "new reply" {
			t.Fatalf("replay mismatch: %+v %v", again, err)
		}
		var count int
		if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages WHERE turn_id=CONVERT(uniqueidentifier,@p1)`, tid.String()).Scan(&count); err != nil || count != 2 {
			t.Fatalf("duplicated messages: %d %v", count, err)
		}
		var remainingLease sql.NullString
		if err := db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),lease_id) FROM dbo.turns WHERE id=CONVERT(uniqueidentifier,@p1)`, tid.String()).Scan(&remainingLease); err != nil || remainingLease.Valid {
			t.Fatalf("completed turn kept lease: %+v %v", remainingLease, err)
		}
	})
	t.Run("concurrent first claims and completions persist exactly one pair", func(t *testing.T) {
		repo := convdb.New(db)
		uid := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		c, err := repo.Create(ctx, uid)
		if err != nil {
			t.Fatal(err)
		}
		defer repo.Delete(ctx, uid, c.ID)
		turn := uuid.New()
		type attempt struct {
			claim convapp.ClaimedTurn
			err   error
		}
		start := make(chan struct{})
		results := make(chan attempt, 8)
		for i := 0; i < cap(results); i++ {
			go func() {
				<-start
				claim, err := repo.ClaimTurn(ctx, uid, c.ID, turn, "synthetic concurrent turn")
				results <- attempt{claim, err}
			}()
		}
		close(start)
		winners, pending := 0, 0
		var lease uuid.UUID
		for i := 0; i < cap(results); i++ {
			result := <-results
			if result.err != nil {
				t.Error(result.err)
			} else if result.claim.LeaseID != uuid.Nil {
				winners++
				lease = result.claim.LeaseID
			} else if result.claim.Pending {
				pending++
			}
		}
		if winners != 1 || pending != 7 {
			t.Fatalf("first claim race: winners=%d pending=%d", winners, pending)
		}
		completed := make(chan error, 2)
		for i := 0; i < cap(completed); i++ {
			go func() {
				_, err := repo.CompleteTurn(ctx, uid, c.ID, turn, lease, "synthetic concurrent turn", "synthetic reply", policy.Version)
				completed <- err
			}()
		}
		successes, conflicts := 0, 0
		for i := 0; i < cap(completed); i++ {
			if err := <-completed; err == nil {
				successes++
			} else if errors.Is(err, httpx.ErrConflict) {
				conflicts++
			} else {
				t.Error(err)
			}
		}
		if successes != 1 || conflicts != 1 {
			t.Fatalf("completion race: successes=%d conflicts=%d", successes, conflicts)
		}
		messages, err := repo.Messages(ctx, uid, c.ID, 100)
		if err != nil || len(messages) != 2 {
			t.Fatalf("duplicate completion persisted messages: count=%d error=%v", len(messages), err)
		}
	})
	t.Run("provider failure cancellation and unsafe output preserve atomicity", func(t *testing.T) {
		repo := convdb.New(db)
		uid := uuid.MustParse("11111111-1111-4111-8111-111111111111")
		profiles := integrationProfiles{userapp.New(userdb.New(db))}
		for _, scenario := range []string{"provider failure", "late cancellation", "invalid output", "unsafe output", "crisis bypass"} {
			t.Run(scenario, func(t *testing.T) {
				c, err := repo.Create(ctx, uid)
				if err != nil {
					t.Fatal(err)
				}
				defer repo.Delete(ctx, uid, c.ID)
				turn := uuid.New()
				turnCtx, cancel := context.WithCancel(ctx)
				defer cancel()
				calls := 0
				gateway := integrationGateway(func(context.Context, []aiapp.ChatMessage) (string, error) {
					calls++
					switch scenario {
					case "provider failure":
						return "", errors.New("synthetic provider failure")
					case "late cancellation":
						cancel()
						return "late synthetic reply", nil
					case "invalid output":
						return string([]byte{0xff}), nil
					case "unsafe output":
						return "I am your therapist", nil
					}
					return "synthetic reply", nil
				})
				service := convapp.New(repo, profiles, aiapp.New(gateway, 16000), 20)
				input := "synthetic atomicity test"
				if scenario == "crisis bypass" {
					input = "I want to kill myself" // synthetic policy fixture only
				}
				result, err := service.Turn(turnCtx, uid, c.ID, turn, input)
				if scenario == "unsafe output" || scenario == "crisis bypass" {
					want := policy.SafetyFailure
					if scenario == "crisis bypass" {
						want = policy.SupportiveFallback
						if calls != 0 {
							t.Fatal("crisis route called provider")
						}
					}
					if err != nil || result.Reply != want {
						t.Fatalf("safe response not returned: %v", err)
					}
				} else {
					if !errors.Is(err, httpx.ErrUnavailable) {
						t.Fatalf("failed generation accepted: %v", err)
					}
					var count int
					if err := db.QueryRowContext(ctx, `SELECT (SELECT COUNT(*) FROM dbo.turns WHERE id=CONVERT(uniqueidentifier,@p1))+
                        (SELECT COUNT(*) FROM dbo.messages WHERE turn_id=CONVERT(uniqueidentifier,@p1))`, turn.String()).Scan(&count); err != nil || count != 1 {
						t.Fatalf("failed generation must retain only its reservation: count=%d error=%v", count, err)
					}
					// Same UUID/input can be retried after failure or cancellation.
					working := aiapp.New(integrationGateway(func(context.Context, []aiapp.ChatMessage) (string, error) {
						return "synthetic retry reply", nil
					}), 16000)
					if _, err := convapp.New(repo, profiles, working, 20).Turn(ctx, uid, c.ID, turn, input); err != nil {
						t.Fatal(err)
					}
				}
				messages, err := repo.Messages(ctx, uid, c.ID, 100)
				if err != nil || len(messages) != 2 || messages[0].Role != "user" || messages[1].Role != "assistant" {
					t.Fatalf("expected one complete ordered pair: count=%d error=%v", len(messages), err)
				}
			})
		}
	})
	t.Run("failed migration rolls back schema and ledger", func(t *testing.T) {
		file := filepath.Join(dir, "9999_rollback_probe.sql")
		if err := os.WriteFile(file, []byte("CREATE TABLE dbo.rollback_probe(id int); INSERT INTO dbo.users(id,test_key) VALUES(NEWID(),NULL);"), 0600); err != nil {
			t.Fatal(err)
		}
		defer os.Remove(file)
		if err := migrations.Apply(ctx, db, dir); err == nil {
			t.Fatal("expected constraint failure")
		}
		var count int
		if err := db.QueryRowContext(ctx, "SELECT (SELECT COUNT(*) FROM sys.tables WHERE name='rollback_probe')+(SELECT COUNT(*) FROM dbo.schema_migrations WHERE name='9999_rollback_probe.sql')").Scan(&count); err != nil || count != 0 {
			t.Fatalf("rollback: count=%d error=%v", count, err)
		}
	})
	t.Run("applied checksum cannot change", func(t *testing.T) {
		file := filepath.Join(dir, filepath.Base(files[0]))
		f, err := os.OpenFile(file, os.O_APPEND|os.O_WRONLY, 0600)
		if err != nil {
			t.Fatal(err)
		}
		_, err = f.WriteString("\n-- checksum probe\n")
		f.Close()
		if err != nil {
			t.Fatal(err)
		}
		if err := migrations.Apply(ctx, db, dir); err == nil || !strings.Contains(err.Error(), "applied migration changed") {
			t.Fatalf("checksum verification: %v", err)
		}
	})
}

type integrationGateway func(context.Context, []aiapp.ChatMessage) (string, error)

func (f integrationGateway) Generate(ctx context.Context, messages []aiapp.ChatMessage) (string, error) {
	return f(ctx, messages)
}

type integrationProfiles struct{ service *userapp.Service }

func (p integrationProfiles) ForUser(ctx context.Context, uid uuid.UUID) (convapp.Profile, error) {
	profile, err := p.service.Get(ctx, uid)
	return convapp.Profile{DisplayName: profile.DisplayName, ConversationStyle: string(profile.ConversationStyle), OnboardingComplete: profile.OnboardingCompletedAt != nil}, err
}
