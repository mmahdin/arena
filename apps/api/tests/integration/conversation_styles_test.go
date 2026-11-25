package integration

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	userapp "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	userdomain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// This test changes only a randomly named disposable local SQL Server database.
// It verifies upgrade from 0010 and old guidance profile compatibility.
func TestFiveConversationStylesSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("requires local MSSQL integration opt-in")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server required")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "mssql" {
		t.Fatal("local SQL Server required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	dbName := "wellbeing_tones_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	params := parsed.Query()
	params.Set("database", dbName)
	parsed.RawQuery = params.Encode()
	if err := database.EnsureLocalDatabase(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	params.Set("database", "master")
	masterURL := *parsed
	masterURL.RawQuery = params.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, err := master.ExecContext(ctx, "ALTER DATABASE ["+dbName+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+dbName+"]"); err != nil {
			t.Error(err)
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	// Apply original 0001..0010 as if this were an existing user database.
	previous := t.TempDir()
	migrationFiles, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range migrationFiles {
		name := filepath.Base(file)
		if name >= "0011_" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(previous, name), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Apply(ctx, db, previous); err != nil {
		t.Fatal(err)
	}
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1);
     INSERT dbo.user_profiles(user_id,display_name,conversation_style,onboarding_completed_at)
     VALUES(CONVERT(uniqueidentifier,@p1),N'Synthetic',N'guidance',SYSUTCDATETIME())`, id.String(), "tone-"+id.String()); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}
	service := userapp.New(userdb.New(db))
	original, err := service.Get(ctx, a)
	if err != nil || original.ConversationStyle != userdomain.Guidance || original.OnboardingCompletedAt == nil {
		t.Fatalf("legacy profile lost on upgrade: %+v %v", original, err)
	}
	for _, style := range []userdomain.Style{userdomain.Balanced, userdomain.Listening, userdomain.Teaching, userdomain.GentleGuidance, userdomain.Collaborative} {
		p, err := service.Update(ctx, a, "Synthetic", style)
		if err != nil || p.ConversationStyle != style {
			t.Fatalf("persist style %s: %+v %v", style, p, err)
		}
		loaded, err := service.Get(ctx, a)
		if err != nil || loaded.ConversationStyle != style || !loaded.OnboardingCompletedAt.Equal(*original.OnboardingCompletedAt) {
			t.Fatalf("reload style %s failed: %+v %v", style, loaded, err)
		}
	}
	other, err := service.Get(ctx, b)
	if err != nil || other.ConversationStyle != userdomain.Guidance {
		t.Fatal("second profile changed")
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.user_profiles SET conversation_style=N'not-a-style' WHERE user_id=CONVERT(uniqueidentifier,@p1)`, a.String()); err == nil {
		t.Fatal("SQL check accepted unknown style")
	}
}
