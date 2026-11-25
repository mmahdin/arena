package integration

import (
	"context"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/application"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/domain"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/config"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Only a uniquely named disposable local database is migrated or dropped.
func TestOnboardingAnswersSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("requires local SQL Server integration opt-in")
	}
	raw, err := config.LoadDatabaseURL()
	if err != nil {
		t.Fatal("local database configuration required")
	}
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" || (parsed.Hostname() != "127.0.0.1" && parsed.Hostname() != "localhost") {
		t.Fatal("loopback SQL Server required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()
	name := "wellbeing_answers_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := parsed.Query()
	q.Set("database", name)
	parsed.RawQuery = q.Encode()
	if database.EnsureLocalDatabase(ctx, parsed.String()) != nil {
		t.Fatal("could not create disposable local database")
	}
	masterURL := *parsed
	q.Set("database", "master")
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal("local master connection failed")
	}
	defer master.Close()
	defer func() {
		if _, err := master.ExecContext(context.Background(), "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); err != nil {
			t.Error("disposable database cleanup failed")
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal("disposable database connection failed")
	}
	defer db.Close()
	previous := t.TempDir()
	files, err := filepath.Glob("../../db/migrations/*.sql")
	if err != nil {
		t.Fatal(err)
	}
	for _, file := range files {
		if filepath.Base(file) >= "0016_" {
			continue
		}
		data, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(previous, filepath.Base(file)), data, 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := migrations.Apply(ctx, db, previous); err != nil {
		t.Fatal(err)
	}
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1);
		INSERT dbo.user_profiles(user_id,conversation_style) VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, id.String(), "answers-"+id.String()); err != nil {
			t.Fatal("synthetic profile creation failed")
		}
	}
	for i := 0; i < 2; i++ {
		if err := migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
			t.Fatal(err)
		}
	}
	service := application.New(userdb.New(db))
	old, err := service.Get(ctx, a)
	if err != nil || old.OnboardingAnswers != nil {
		t.Fatal("upgrade did not preserve an unanswered profile")
	}
	zero := 0
	answers := &domain.OnboardingAnswers{Identity: "زن", Mood: &zero, Concerns: []string{"کار یا درس"}, Focus: []string{"بهتر بخوابم"}, Recharge: []string{"رفتن به دل طبیعت"}}
	saved, err := service.UpdateWithOnboarding(ctx, a, "Synthetic A", domain.Listening, answers)
	if err != nil || !reflect.DeepEqual(saved.OnboardingAnswers, answers) {
		t.Fatal("onboarding answer round trip failed")
	}
	completion := *saved.OnboardingCompletedAt
	saved, err = service.Update(ctx, a, "Synthetic edit", domain.Balanced)
	if err != nil || !reflect.DeepEqual(saved.OnboardingAnswers, answers) || !saved.OnboardingCompletedAt.Equal(completion) {
		t.Fatal("ordinary edit changed answers or completion time")
	}
	other, err := service.Get(ctx, b)
	if err != nil || other.OnboardingAnswers != nil || other.OnboardingCompletedAt != nil {
		t.Fatal("another user's answers were changed")
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.user_profiles SET onboarding_answers_json=N'invalid' WHERE user_id=CONVERT(uniqueidentifier,@p1)`, a.String()); err == nil {
		t.Fatal("database accepted invalid JSON")
	}
	saved, err = service.UpdateWithOnboarding(ctx, a, "Synthetic edit", domain.Balanced, &domain.OnboardingAnswers{})
	if err != nil || saved.OnboardingAnswers == nil || len(saved.OnboardingAnswers.Concerns) != 0 {
		t.Fatal("explicit empty answers failed to clear saved context")
	}
}
