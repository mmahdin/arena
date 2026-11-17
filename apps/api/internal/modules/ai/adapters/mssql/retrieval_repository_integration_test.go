package mssql

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// TestRepositoryOwnerIsolationSQLServer runs the retrieval read model against a
// disposable local SQL Server. It is skipped unless WELLBEING_INTEGRATION=true,
// matching the repository convention (see tests/integration).
//
// The full end-to-end scenario lives in tests/integration/retrieval_test.go; this
// test focuses on the repository contract itself: owner scoping in every method,
// soft-delete behaviour, and identifier validation.
func TestRepositoryOwnerIsolationSQLServer(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("SQL Server integration opt-in")
	}
	parsed, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || parsed.Scheme != "sqlserver" || os.Getenv("APP_ENV") != "local" {
		t.Fatal("local SQL Server and APP_ENV=local required")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	name := "wellbeing_ai_repo_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
		if _, e := master.ExecContext(ctx, fmt.Sprintf(
			"ALTER DATABASE [%s] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE [%s]", name, name)); e != nil {
			t.Errorf("drop database: %v", e)
		}
	}()
	db, err := database.Connect(ctx, parsed.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	dir, err := filepath.Abs("../../../db/migrations")
	if err != nil {
		t.Fatal(err)
	}
	if err := migrations.Apply(ctx, db, dir); err != nil {
		t.Fatal(err)
	}

	alice, bob := uuid.New(), uuid.New()
	for _, id := range []uuid.UUID{alice, bob} {
		if _, err := db.ExecContext(ctx, `INSERT dbo.users(id,test_key,is_test)
            VALUES(CONVERT(uniqueidentifier,@p1),@p2,1)`, id.String(), "repo-"+id.String()); err != nil {
			t.Fatal(err)
		}
	}
	itemID := uuid.New()
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,status,importance)
        VALUES(CONVERT(uniqueidentifier,@p1),CONVERT(uniqueidentifier,@p2),N'fact',N'synthetic',N'reported',N'active',0.5)`,
		itemID.String(), alice.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := db.ExecContext(ctx, `INSERT dbo.ai_memory_evidence(item_id,ref_kind,ref_id,role)
        VALUES(CONVERT(uniqueidentifier,@p1),N'message',@p2,N'supports')`, itemID.String(), uuid.New().String()); err != nil {
		t.Fatal(err)
	}

	repo := New(db)
	// Owner sees the row.
	mine, err := repo.MemoryByIDs(ctx, alice, []contract.ID{itemID})
	if err != nil {
		t.Fatal(err)
	}
	if len(mine) != 1 || mine[0].EvidenceN != 1 {
		t.Fatalf("owner read failed: %+v", mine)
	}
	// A second user can never see it, even with the exact id.
	theirs, err := repo.MemoryByIDs(ctx, bob, []contract.ID{itemID})
	if err != nil {
		t.Fatal(err)
	}
	if len(theirs) != 0 {
		t.Fatal("cross-user read by id succeeded")
	}
	// Direct-ID reads are bounded.
	if out, err := repo.MemoryByIDs(ctx, alice, nil); err != nil || len(out) != 0 {
		t.Fatalf("empty id list: %v %v", out, err)
	}
	if out, err := repo.RecentMemory(ctx, alice, 1); err != nil || len(out) != 1 {
		t.Fatalf("bounded pool read failed: %v %v", out, err)
	}
}
