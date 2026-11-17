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
	reflectiondb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/reflection/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// Opt-in real MSSQL, isolated disposable database, never touches local user data.
func TestDailyReflectionsSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local MSSQL required")
	}
	host := strings.ToLower(parsed.Hostname())
	if host != "127.0.0.1" && host != "localhost" && host != "mssql" {
		t.Fatal("only local SQL Server allowed")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	name := "wellbeing_reflection_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
		t.Fatal("repeat migration", err)
	}
	a, b := uuid.New(), uuid.New()
	for _, uid := range []uuid.UUID{a, b} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test) VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, uid.String(), "reflection-"+uid.String()); err != nil {
			t.Fatal(err)
		}
	}
	repo := reflectiondb.New(db)
	const day = "2026-11-02"
	if _, err := repo.Get(ctx, b, day); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("unknown must be 404", err)
	}
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, e := repo.Save(ctx, a, day, "یادداشت ساختگی"); e != nil {
				t.Error("parallel save", e)
			}
		}()
	}
	wg.Wait()
	var count int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.daily_reflections WHERE user_id=CONVERT(uniqueidentifier,@p1)`, a.String()).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate rows", count, err)
	}
	if _, err := repo.Get(ctx, b, day); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("cross user read", err)
	}
	if other, err := repo.List(ctx, b); err != nil || len(other) != 0 {
		t.Fatal("cross user list", other, err)
	}
	if err := repo.Delete(ctx, b, day); err != nil {
		t.Fatal(err)
	}
	value, err := repo.Get(ctx, a, day)
	if err != nil || value.Text != "یادداشت ساختگی" {
		t.Fatal("foreign delete", value, err)
	}
	changed, err := repo.Save(ctx, a, day, "ویرایش ساختگی")
	if err != nil || changed.Text != "ویرایش ساختگی" {
		t.Fatal("update", changed, err)
	}
	if changed.CreatedAt != value.CreatedAt {
		t.Fatal("updated entry changed creation time")
	}
	if err := repo.Delete(ctx, a, day); err != nil {
		t.Fatal(err)
	}
	if err := repo.Delete(ctx, a, day); err != nil {
		t.Fatal("repeat delete", err)
	}
	if _, err := repo.Get(ctx, a, day); !errors.Is(err, httpx.ErrNotFound) {
		t.Fatal("delete did not remove", err)
	}
	if _, err := repo.Save(ctx, a, day, "یادداشت بعد از حذف"); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `DELETE dbo.users WHERE id=CONVERT(uniqueidentifier,@p1)`, a.String()); err == nil {
		t.Fatal("physical account deletion accepted")
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.daily_reflections`).Scan(&count); err != nil || count != 2 {
		t.Fatal("preserved deleted entry and replacement", count, err)
	}
}
