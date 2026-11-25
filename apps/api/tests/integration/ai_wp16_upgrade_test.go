package integration

import (
	"context"
	"database/sql"
	"encoding/json"
	userdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/user/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
	"os"
	"path/filepath"
	"testing"
)

func TestWP16SQLAccountUpgradePreservesOriginalAndDerivedData(t *testing.T) {
	db := aiDatabase(t)
	dir := t.TempDir()
	files, e := filepath.Glob("../../db/migrations/*.sql")
	if e != nil {
		t.Fatal(e)
	}
	for _, file := range files {
		if filepath.Base(file) >= "0023_" {
			continue
		}
		raw, e := os.ReadFile(file)
		if e != nil {
			t.Fatal(e)
		}
		if e = os.WriteFile(filepath.Join(dir, filepath.Base(file)), raw, 0600); e != nil {
			t.Fatal(e)
		}
	}
	if e = migrations.Apply(context.Background(), db, dir); e != nil {
		t.Fatal(e)
	}
	f := aiSetup(t, db)
	legacy := legacyChatNew(db)
	req := f.req()
	claim, e := legacy.ClaimTurn(context.Background(), f.a, f.ca, req.ClientTurnID, req.Text)
	if e != nil {
		t.Fatal(e)
	}
	if _, e = legacy.CompleteTurn(context.Background(), f.a, f.ca, req.ClientTurnID, claim.LeaseID, req.Text, "Synthetic original answer", "historical-fixture"); e != nil {
		t.Fatal(e)
	}
	aiExec(t, db, `INSERT dbo.daily_reflections(user_id,entry_date,body) VALUES(@p1,'2026-11-18',N'Synthetic retained diary'); INSERT dbo.ai_user_state(user_id,consent_json) VALUES(@p1,N'{"synthetic_memory":true}'); INSERT dbo.ai_memory_items(id,user_id,kind,content,epistemic,status,source_json) VALUES(NEWID(),@p1,N'fact',N'Synthetic retained derivative',N'reported',N'active',N'{}')`, f.a)
	originals := map[string]string{}
	for _, table := range []string{"messages", "daily_reflections", "ai_memory_items", "ai_user_state"} {
		if table == "messages" {
			originals[table] = retainedJSON(t, db, `SELECT m.* FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id=@p1 ORDER BY m.message_order`, f.a)
		} else {
			originals[table] = retainedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a)
		}
	}
	aiMigrate(t, db)
	aiMigrate(t, db)
	for table, before := range originals {
		var after string
		if table == "messages" {
			after = retainedJSON(t, db, `SELECT m.* FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id=@p1 ORDER BY m.message_order`, f.a)
		} else {
			after = retainedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a)
		}
		if before != after {
			t.Fatal("migration altered retained data", table)
		}
	}
	accounts := userdb.New(db)
	if e = accounts.HideAccount(context.Background(), f.a); e != nil {
		t.Fatal(e)
	}
	before := retainedJSON(t, db, `SELECT deleted_at,deleted_by,deletion_reason FROM dbo.users WHERE id=@p1`, f.a)
	if e = accounts.HideAccount(context.Background(), f.a); e != nil {
		t.Fatal(e)
	}
	if before != retainedJSON(t, db, `SELECT deleted_at,deleted_by,deletion_reason FROM dbo.users WHERE id=@p1`, f.a) {
		t.Fatal("repeated hide replaced metadata")
	}
	for table, old := range originals {
		var now string
		if table == "messages" {
			now = retainedJSON(t, db, `SELECT m.* FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE v.user_id=@p1 ORDER BY m.message_order`, f.a)
		} else {
			now = retainedJSON(t, db, "SELECT * FROM dbo."+table+" WHERE user_id=@p1", f.a)
		}
		if now != old {
			t.Fatal("account hide changed payload/consent", table)
		}
	}
	if _, e = db.Exec(`DELETE dbo.users WHERE id=@p1`, f.a); e == nil {
		t.Fatal("physical account deletion allowed")
	}
	var retired int
	if e = db.QueryRowContext(context.Background(), `SELECT CASE WHEN OBJECT_ID(N'dbo.dev_sessions',N'U') IS NULL AND OBJECT_ID(N'dbo.dev_phone_identities',N'U') IS NULL THEN 0 ELSE 1 END`).Scan(&retired); e != nil || retired != 0 {
		t.Fatal("legacy session schema survived AUTH-09", e, retired)
	}
}

// Index revisions are additive metadata. Compare every original field while
// allowing a rebuildable index revision to appear during a forward migration.
func retainedJSON(t *testing.T, db *sql.DB, query string, args ...any) string {
	t.Helper()
	raw := storedJSON(t, db, query, args...)
	var rows []map[string]json.RawMessage
	if e := json.Unmarshal([]byte(raw), &rows); e != nil {
		t.Fatal(e)
	}
	for _, row := range rows {
		delete(row, "search_version")
	}
	encoded, e := json.Marshal(rows)
	if e != nil {
		t.Fatal(e)
	}
	return string(encoded)
}
