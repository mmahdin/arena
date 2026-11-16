package integration

import (
	"context"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	convdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/conversation/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Run with WELLBEING_INTEGRATION=true on local SQL Server 2022. It creates a
// disposable database, never truncates or migrates the user's development DB.
func TestConversationTitlesSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server required")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "localhost" && host != "127.0.0.1" && host != "mssql" {
		t.Fatal("refuse nonlocal SQL Server")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	name := "wellbeing_titles_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := parsed.Query()
	q.Set("database", name)
	parsed.RawQuery = q.Encode()
	if err := database.EnsureLocalDatabase(ctx, parsed.String()); err != nil {
		t.Fatal(err)
	}
	masterURL := *parsed
	q.Set("database", "master")
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		if _, e := master.ExecContext(ctx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); e != nil {
			t.Error(e)
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
		t.Fatal("repeat migrations", err)
	}
	owner, other := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{owner, other} {
		if _, err := db.ExecContext(ctx, `INSERT INTO dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, id.String(), "title-"+id.String()); err != nil {
			t.Fatal(err)
		}
	}
	repo := convdb.New(db)
	conversation, err := repo.Create(ctx, owner)
	if err != nil {
		t.Fatal(err)
	}
	if conversation.Title != "گفت‌وگوی جدید" {
		t.Fatal("new conversation title", conversation.Title)
	}
	if _, err := repo.Rename(ctx, other, conversation.ID, "متن حساب دیگر"); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("wrong owner rename", err)
	}
	if _, err := repo.Messages(ctx, other, conversation.ID, 100); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("wrong owner read", err)
	}
	first := uuid.New()
	claim, err := repo.ClaimTurn(ctx, owner, conversation.ID, first, "برنامه آخر هفته. سایر جزئیات ساختگی")
	if err != nil || claim.LeaseID == uuid.Nil {
		t.Fatal("claim first turn", err)
	}
	// Before successful completion, the placeholder must not change.
	before, err := repo.List(ctx, owner)
	if err != nil || len(before) != 1 || before[0].Title != "گفت‌وگوی جدید" {
		t.Fatal("premature title", before, err)
	}
	if _, err := repo.CompleteTurn(ctx, owner, conversation.ID, first, claim.LeaseID, "برنامه آخر هفته. سایر جزئیات ساختگی", "پاسخ ساختگی", "supportive-test"); err != nil {
		t.Fatal(err)
	}
	after, err := repo.List(ctx, owner)
	if err != nil || len(after) != 1 || after[0].Title != "برنامه آخر هفته" {
		t.Fatal("auto title", after, err)
	}
	renamed, err := repo.Rename(ctx, owner, conversation.ID, "نام انتخابی من")
	if err != nil || renamed.Title != "نام انتخابی من" {
		t.Fatal("manual rename", renamed, err)
	}
	second := uuid.New()
	next, err := repo.ClaimTurn(ctx, owner, conversation.ID, second, "پیام دوم با محتوای تازه")
	if err != nil || next.LeaseID == uuid.Nil {
		t.Fatal("second claim", err)
	}
	if _, err := repo.CompleteTurn(ctx, owner, conversation.ID, second, next.LeaseID, "پیام دوم با محتوای تازه", "پاسخ دوم", "supportive-test"); err != nil {
		t.Fatal("second complete", err)
	}
	final, err := repo.List(ctx, owner)
	if err != nil || final[0].Title != "نام انتخابی من" {
		t.Fatal("manual title overwritten", final, err)
	}
	replay, err := repo.ClaimTurn(ctx, owner, conversation.ID, first, "برنامه آخر هفته. سایر جزئیات ساختگی")
	if err != nil || !replay.Existing {
		t.Fatal("idempotent replay", replay, err)
	}
	none, err := repo.List(ctx, other)
	if err != nil || len(none) != 0 {
		t.Fatal("cross-account list", none, err)
	}
}
