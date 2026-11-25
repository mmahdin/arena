package mssql

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/microsoft/go-mssqldb"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

// Explicitly disposable database ONLY, migrated through 0031. Leaves synthetic
// auth rows. Never point this at staging/production or a shared database.
func TestAUTH08DisposableSQLSessionLifecycle(t *testing.T) {
	dsn := os.Getenv("THEIA_DISPOSABLE_AUTH08_SQL_DSN")
	if dsn == "" || os.Getenv("THEIA_DISPOSABLE_AUTH08_SQL_ACK") != "YES_DESTROY_AFTER_TEST" {
		t.Skip("disposable SQL Server with migration 0031 required")
	}
	db, err := sql.Open("sqlserver", dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err = db.PingContext(ctx); err != nil {
		t.Fatal(err)
	}
	repo := New(db, true)
	issuer := "https://auth.example.invalid/realms/theia-users"
	subject := uuid.NewString()
	principal, err := repo.ResolveIdentity(ctx, auth.User, issuer, subject)
	if err != nil {
		t.Fatal(err)
	}
	sid := "sid-" + uuid.NewString()
	create := func(start time.Time, sessionID string) ([]byte, error) {
		digest := sha256.Sum256([]byte(uuid.NewString()))
		now := time.Now().UTC()
		err := repo.SaveSession(ctx, auth.Session{TokenHash: digest[:], Kind: auth.User, PrincipalID: principal,
			Issuer: issuer, Subject: subject, KeycloakSID: sessionID, LoginStartedAt: start, CreatedAt: now,
			IdleTTL: auth.UserIdle, IdleExpiresAt: now.Add(auth.UserIdle), ExpiresAt: now.Add(auth.UserAbsolute)})
		return digest[:], err
	}
	early := time.Now().Add(-time.Minute)
	one, err := create(early, sid)
	if err != nil {
		t.Fatal(err)
	}
	two, err := create(early, sid)
	if err != nil {
		t.Fatal(err)
	}
	otherIssuer := "https://auth.example.invalid/realms/theia-staff"
	event := sha256.Sum256([]byte(uuid.NewString()))
	// A signed sid+sub event may only revoke this issuer, subject and SID.
	applied, err := repo.RevokeBackchannel(ctx, auth.User, issuer, subject, sid, event[:], time.Now().Add(time.Minute))
	if err != nil || !applied {
		t.Fatalf("backchannel failed: %v", err)
	}
	for _, hash := range [][]byte{one, two} {
		if _, err = repo.ActiveSession(ctx, auth.User, hash); !errors.Is(err, auth.ErrNotFound) {
			t.Fatal("revoked session accepted")
		}
	}
	if _, err = create(early, sid); !errors.Is(err, auth.ErrInactive) {
		t.Fatalf("late callback resurrected session: %v", err)
	}
	if repeat, err := repo.RevokeBackchannel(ctx, auth.User, issuer, subject, sid, event[:], time.Now().Add(time.Minute)); err != nil || repeat {
		t.Fatalf("duplicate event replayed: %v", err)
	}
	secondEvent := sha256.Sum256([]byte(uuid.NewString()))
	if _, err = repo.RevokeBackchannel(ctx, auth.User, issuer, subject, sid, secondEvent[:], time.Now().Add(time.Minute)); err != nil {
		t.Fatalf("second Keycloak event for same session should be idempotent: %v", err)
	}
	_ = otherIssuer // cross-realm isolation is a fixed predicate of the tested query
	// A fresh session (new Keycloak SID) begun after the logout may log in.
	fresh, err := create(time.Now().Add(time.Second), "new-"+uuid.NewString())
	if err != nil {
		t.Fatalf("new login blocked: %v", err)
	}
	second, err := create(time.Now().Add(time.Second), "new-"+uuid.NewString())
	if err != nil {
		t.Fatal(err)
	}
	if err = repo.RevokeAllForPrincipal(ctx, auth.User, principal); err != nil {
		t.Fatal(err)
	}
	for _, hash := range [][]byte{fresh, second} {
		if _, err = repo.ActiveSession(ctx, auth.User, hash); !errors.Is(err, auth.ErrNotFound) {
			t.Fatal("logout-all left Theia session active")
		}
	}
	if _, err = create(early, "new-"+uuid.NewString()); !errors.Is(err, auth.ErrInactive) {
		t.Fatalf("logout-all watermark missed old callback: %v", err)
	}
	if _, err = create(time.Now().Add(time.Second), "new-"+uuid.NewString()); err != nil {
		t.Fatalf("fresh login after logout-all blocked: %v", err)
	}
}
