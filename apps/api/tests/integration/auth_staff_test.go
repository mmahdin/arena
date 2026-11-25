package integration

import (
	"context"
	"crypto/sha256"
	"errors"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
	authdb "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth/adapters/mssql"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/database"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/migrations"
)

// This creates and destroys its own disposable database. It NEVER inserts
// bootstrap identities into an existing developer or production database.
func TestAuth04StaffProvisioningSQL(t *testing.T) {
	if os.Getenv("WELLBEING_INTEGRATION") != "true" {
		t.Skip("requires opt-in SQL Server integration")
	}
	if os.Getenv("APP_ENV") != "local" {
		t.Fatal("integration requires APP_ENV=local")
	}
	u, err := url.Parse(os.Getenv("DATABASE_URL"))
	if err != nil || u.Scheme != "sqlserver" || (u.Hostname() != "localhost" && u.Hostname() != "127.0.0.1" && u.Hostname() != "mssql") {
		t.Fatal("integration needs a local SQL Server DATABASE_URL")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	name := "theia_staff_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	q := u.Query()
	q.Set("database", name)
	u.RawQuery = q.Encode()
	if err = database.EnsureLocalDatabase(ctx, u.String()); err != nil {
		t.Fatal(err)
	}
	q.Set("database", "master")
	masterURL := *u
	masterURL.RawQuery = q.Encode()
	master, err := database.Connect(ctx, masterURL.String())
	if err != nil {
		t.Fatal(err)
	}
	defer master.Close()
	defer func() {
		cleanupCtx, done := context.WithTimeout(context.Background(), 30*time.Second)
		defer done()
		if _, e := master.ExecContext(cleanupCtx, "ALTER DATABASE ["+name+"] SET SINGLE_USER WITH ROLLBACK IMMEDIATE; DROP DATABASE ["+name+"]"); e != nil {
			t.Errorf("test database cleanup: %v", e)
		}
	}()
	db, err := database.Connect(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err = migrations.Apply(ctx, db, "../../db/migrations"); err != nil {
		t.Fatal(err)
	}

	provisioner := authdb.NewStaffProvisioner(db)
	issuer := "http://auth.localhost:8180/realms/theia-staff"
	subject := uuid.NewString()
	if _, err = provisioner.State(ctx, issuer, subject); !errors.Is(err, auth.ErrNotFound) {
		t.Fatalf("unexpected initial state: %v", err)
	}
	if _, _, err = provisioner.Provision(ctx, "http://auth.localhost:8180/realms/theia-users", subject); !errors.Is(err, auth.ErrInvalid) {
		t.Fatal("wrong realm provision accepted")
	}
	state, created, err := provisioner.Provision(ctx, issuer, subject)
	if err != nil || !created || !state.Active {
		t.Fatalf("provision: %+v, %v, %v", state, created, err)
	}
	again, created, err := provisioner.Provision(ctx, issuer, subject)
	if err != nil || created || again.ID != state.ID {
		t.Fatalf("idempotent provision: %+v, %v, %v", again, created, err)
	}
	repo := authdb.New(db, true)
	tokenHash := sha256.Sum256([]byte("integration-staff-session"))
	now := time.Now().UTC().Add(-time.Second)
	sess := auth.Session{
		TokenHash: tokenHash[:], Kind: auth.Staff, PrincipalID: state.ID, Issuer: issuer, Subject: subject,
		CreatedAt: now, IdleExpiresAt: now.Add(30 * time.Minute), ExpiresAt: now.Add(12 * time.Hour), IdleTTL: 30 * time.Minute,
	}
	if err = repo.SaveSession(ctx, sess); err != nil {
		t.Fatalf("staff session create: %v", err)
	}
	if _, err = repo.ActiveSession(ctx, auth.User, tokenHash[:]); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("staff session accepted as user")
	}
	if _, err = repo.ActiveSession(ctx, auth.Staff, tokenHash[:]); err != nil {
		t.Fatalf("active staff session rejected: %v", err)
	}
	state, changed, err := provisioner.SetActive(ctx, issuer, subject, false)
	if err != nil || !changed || state.Active {
		t.Fatalf("deactivate: %+v %v %v", state, changed, err)
	}
	if _, err = repo.ActiveSession(ctx, auth.Staff, tokenHash[:]); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("deactivated staff session accepted")
	}
	if _, _, err = provisioner.Provision(ctx, issuer, subject); !errors.Is(err, auth.ErrInactive) {
		t.Fatal("provision silently reactivated staff")
	}
	state, changed, err = provisioner.SetActive(ctx, issuer, subject, true)
	if err != nil || !changed || !state.Active {
		t.Fatalf("reactivate: %+v %v %v", state, changed, err)
	}
	if _, err = repo.ActiveSession(ctx, auth.Staff, tokenHash[:]); !errors.Is(err, auth.ErrNotFound) {
		t.Fatal("reactivation revived old session")
	}
	var staffAccounts, identities, audit int
	if err = db.QueryRowContext(ctx, `SELECT
       (SELECT COUNT(*) FROM dbo.staff_accounts WHERE id=CONVERT(uniqueidentifier,@p1)),
       (SELECT COUNT(*) FROM dbo.auth_identities WHERE issuer=@p2 AND subject=@p3),
       (SELECT COUNT(*) FROM dbo.auth_audit_events WHERE target_staff_id=CONVERT(uniqueidentifier,@p1))`,
		state.ID.String(), issuer, subject).Scan(&staffAccounts, &identities, &audit); err != nil {
		t.Fatal(err)
	}
	if staffAccounts != 1 || identities != 1 || audit != 3 {
		t.Fatalf("unexpected persisted account/identity/audit: %d/%d/%d", staffAccounts, identities, audit)
	}
}
