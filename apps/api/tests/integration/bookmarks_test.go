package integration

import (
	"context"
	"errors"
	"github.com/google/uuid"
	saveddb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/bookmark/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// Never touch the developer database: create and destroy a disposable local SQL database only.
func TestSavedMessagesSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server required")
	}
	if host := strings.ToLower(parsed.Hostname()); host != "127.0.0.1" && host != "localhost" && host != "mssql" {
		t.Fatal("local-only SQL Server required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Second)
	defer cancel()
	name := "wellbeing_bookmark_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
		t.Fatal("repeat migration", err)
	}
	a, b, conv, turn, msg := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{a, b} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1);
  INSERT dbo.user_profiles(user_id,conversation_style) VALUES(CONVERT(uniqueidentifier,@p1),N'balanced')`, id.String(), "saved-"+id.String()); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.conversations(id,user_id,title) VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'گفتگوی ساختگی')`, conv.String(), a.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.turns(id,conversation_id,user_id,input_hash,status)
 VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),@p4,N'completed')`, turn.String(), conv.String(), a.String(), make([]byte, 32)); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.messages(id,conversation_id,turn_id,role,content)
 VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),CONVERT(uniqueidentifier,@p3),N'assistant',N'متن ساختگی')`, msg.String(), conv.String(), turn.String()); err != nil {
		t.Fatal(err)
	}
	repo := saveddb.New(db)
	if err := repo.Save(ctx, b, msg); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatalf("cross owner save: %v", err)
	}
	if err := repo.Remove(ctx, b, msg); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatalf("cross owner remove: %v", err)
	}
	foreign, err := repo.List(ctx, b)
	if err != nil || len(foreign) != 0 {
		t.Fatal("foreign items leaked", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if e := repo.Save(ctx, a, msg); e != nil {
				t.Error("parallel save", e)
			}
		}()
	}
	wg.Wait()
	saved, err := repo.List(ctx, a)
	if err != nil || len(saved) != 1 || saved[0].ID != msg || saved[0].ConversationID != conv || saved[0].Content != "متن ساختگی" {
		t.Fatal("saved row incorrect", saved, err)
	}
	if err := repo.Remove(ctx, a, msg); err != nil {
		t.Fatal(err)
	}
	if err := repo.Remove(ctx, a, msg); err != nil {
		t.Fatal("idempotent unsave", err)
	}
	saved, err = repo.List(ctx, a)
	if err != nil || len(saved) != 0 {
		t.Fatal("unsave failed", err)
	}
	if err := repo.Save(ctx, a, msg); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE dbo.conversations WHERE id=CONVERT(uniqueidentifier,@p1) AND user_id=CONVERT(uniqueidentifier,@p2)`, conv.String(), a.String()); err != nil {
		t.Fatal(err)
	}
	saved, err = repo.List(ctx, a)
	if err != nil || len(saved) != 0 {
		t.Fatal("cascade delete failed", err)
	}
	if err := repo.Save(ctx, a, msg); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("removed message accepted", err)
	}
}
