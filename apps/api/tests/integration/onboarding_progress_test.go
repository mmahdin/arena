package integration

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// This test runs only against a disposable LOCAL SQL Server database.
// It never truncates, upgrades or drops the developer's existing database.
func TestOnboardingProgressSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
	}
	raw := os.Getenv("DATABASE_URL")
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server required")
	}
	if host := strings.ToLower(parsed.Hostname()); host != "127.0.0.1" && host != "localhost" && host != "mssql" {
		t.Fatal("local-only database host required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 80*time.Second)
	defer cancel()
	name := "wellbeing_cursor_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := parsed.Query()
	q.Set("database", name)
	parsed.RawQuery = q.Encode()
	if err := database.EnsureLocalDatabase(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	q.Set("database", "master")
	masterURL := *parsed
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, err := master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); err != nil {
			t.Error(err)
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal("reapplication failed", err)
	}
	first, second := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{first, second} {
		_, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1);
            INSERT dbo.user_profiles(user_id,conversation_style) VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, id.String(), "cursor-"+id.String())
		if err != nil {
			t.Fatal(err)
		}
	}
	service := application.NewProgressService(userdb.NewProgressRepository(db))
	current, err := service.Get(ctx, first)
	if err != nil || current.FurthestStep != 0 {
		t.Fatal(current, err)
	}
	if _, err = service.Advance(ctx, first, 25); !errors.Is(err, application.ErrProgressConflict) {
		t.Fatal("accepted jump", err)
	}
	for i := 1; i <= 25; i++ {
		current, err = service.Advance(ctx, first, i)
		if err != nil || current.FurthestStep != i {
			t.Fatalf("step %d: %+v %v", i, current, err)
		}
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := service.Advance(ctx, first, 19); e != nil {
				t.Error(e)
			}
		}()
	}
	wg.Wait()
	after, err := service.Get(ctx, first)
	if err != nil || after.FurthestStep != 25 {
		t.Fatal("cursor regressed", after, err)
	}
	isolated, err := service.Get(ctx, second)
	if err != nil || isolated.FurthestStep != 0 {
		t.Fatal("other user progressed", isolated, err)
	}
	// SQL rejects out-of-range values without altering the profile.
	if _, err := db.ExecContext(ctx, `UPDATE dbo.user_profiles SET onboarding_step=26 WHERE user_id=CONVERT(uniqueidentifier,@p1)`, second.String()); err == nil {
		t.Fatal("range constraint did not reject 26")
	}
	var next sync.WaitGroup
	for i := 0; i < 8; i++ {
		next.Add(1)
		go func() {
			defer next.Done()
			if _, err := service.Advance(ctx, second, 1); err != nil {
				t.Error(err)
			}
		}()
	}
	next.Wait()
	if got, err := service.Get(ctx, second); err != nil || got.FurthestStep != 1 {
		t.Fatal("concurrent next step inconsistent", err)
	}
	// Existing account that completes onboarding retains its cursor; later PUT is rejected.
	_, err = db.ExecContext(ctx, `UPDATE dbo.user_profiles SET onboarding_completed_at=SYSUTCDATETIME() WHERE user_id=CONVERT(uniqueidentifier,@p1)`, first.String())
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Advance(ctx, first, 25); !errors.Is(err, application.ErrProgressCompleted) {
		t.Fatal("completed account advanced", err)
	}
}
