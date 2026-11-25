package mssql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

// Run only against a throwaway SQL Server 2022 database with migrations through
// 0030. Tests intentionally leave synthetic user/auth rows behind. Never run
// against shared/staging/production data.
func TestAUTH06FirstLoginAndReturningUserDisposableSQL(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_USER_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_USER_SQL_ACK") != "YES_DESTROY_AFTER_TEST" {
		t.Skip("requires explicitly acknowledged disposable SQL Server and migration 0030")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 45*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	repo := New(db, true)
	issuer := "https://auth.example.invalid/realms/theia-users"
	firstSubject := uuid.NewString()
	const attempts = 3
	var ids [attempts]uuid.UUID
	var errs [attempts]error
	var wg sync.WaitGroup
	for i := 0; i < attempts; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			ids[index], errs[index] = repo.ResolveIdentity(ctx, auth.User, issuer, firstSubject)
		}(i)
	}
	wg.Wait()
	for i := 0; i < attempts; i++ {
		if errs[i] != nil || ids[i] == uuid.Nil || ids[i] != ids[0] {
			t.Fatalf("concurrent first-login diverged: %v %v", ids, errs)
		}
	}
	id := ids[0]
	var isTest bool
	var onboarding sql.NullTime
	if err := db.QueryRowContext(ctx, `SELECT u.is_test,p.onboarding_completed_at FROM dbo.users AS u JOIN dbo.user_profiles AS p ON p.user_id=u.id WHERE u.id=CONVERT(uniqueidentifier,@p1)`, id.String()).Scan(&isTest, &onboarding); err != nil {
		t.Fatal(err)
	}
	if !isTest || onboarding.Valid {
		t.Fatal("new synthetic-only User should start onboarding and never become a real account")
	}
	other, err := repo.ResolveIdentity(ctx, auth.User, issuer, uuid.NewString())
	if err != nil || other == id {
		t.Fatalf("distinct Keycloak subject reused owner: %v", err)
	}
	token := sha256.Sum256([]byte("synthetic-token-" + uuid.NewString()))
	now := time.Now().UTC()
	s := auth.Session{TokenHash: token[:], Kind: auth.User, PrincipalID: id, Issuer: issuer, Subject: firstSubject, CreatedAt: now, IdleTTL: auth.UserIdle, IdleExpiresAt: now.Add(auth.UserIdle), ExpiresAt: now.Add(auth.UserAbsolute)}
	if err := repo.SaveSession(ctx, s); err != nil {
		t.Fatal(err)
	}
	active, err := repo.ActiveSession(ctx, auth.User, token[:])
	if err != nil || active.PrincipalID != id {
		t.Fatalf("User session owner mismatch: %v", err)
	}
	if _, err := repo.ActiveSession(ctx, auth.Staff, token[:]); err == nil {
		t.Fatal("User session accepted as Staff")
	}
	if _, err := db.ExecContext(ctx, `UPDATE dbo.users SET deleted_at=SYSUTCDATETIME() WHERE id=CONVERT(uniqueidentifier,@p1)`, id.String()); err != nil {
		t.Fatal(err)
	}
	if _, err := repo.ActiveSession(ctx, auth.User, token[:]); err == nil {
		t.Fatal("deleted User still authorized")
	}
	if _, err := repo.ResolveIdentity(ctx, auth.User, issuer, firstSubject); err != auth.ErrInactive {
		t.Fatalf("deleted identity was revived: %v", err)
	}
}
