package integration

import (
	"context"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	privacydb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Uses a disposable local SQL Server database. Never mutates the developer database.
func TestPrivacyPreferenceSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
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
	dbName := "wellbeing_privacy_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := parsed.Query()
	q.Set("database", dbName)
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
		if _, err := master.ExecContext(ctx, "ALTER DATABASE ["+dbName+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+dbName+"]"); err != nil {
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
		t.Fatal(err)
	}
	a, b := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1);
          INSERT dbo.user_profiles(user_id,conversation_style) VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, id.String(), "privacy-"+id.String()); err != nil {
			t.Fatal(err)
		}
	}
	repo := privacydb.NewPrivacyRepository(db)
	get := func(id uuid.UUID, want bool) {
		t.Helper()
		p, err := repo.GetPrivacy(ctx, id)
		if err != nil || p.ConversationSharingEnabled != want {
			t.Fatalf("get preference: %+v %v", p, err)
		}
	}
	get(a, true)
	get(b, true)
	if _, err := repo.SetPrivacy(ctx, a, false); err != nil {
		t.Fatal(err)
	}
	get(a, false)
	get(b, true)
	if _, err := repo.SetPrivacy(ctx, a, true); err != nil {
		t.Fatal(err)
	}
	get(a, true)
}
