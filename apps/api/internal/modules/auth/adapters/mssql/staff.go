package mssql

import (
	"context"
	"database/sql"
	"errors"
	"net/url"

	"github.com/google/uuid"
	auth "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/auth"
)

// StaffProvisioner binds an existing Keycloak staff subject to the single
// fixed admin role. This repository is called only by a local operator CLI;
// it is NEVER exposed as a public or user-accessible HTTP API.
type StaffProvisioner struct{ db *sql.DB }

func NewStaffProvisioner(db *sql.DB) *StaffProvisioner { return &StaffProvisioner{db: db} }

// ValidateStaffBinding rejects malformed, wrong-realm and ambiguous subject
// identifiers. Keycloak's locally created user IDs are canonical UUIDs.
func ValidateStaffBinding(issuer, subject string) error {
	uri, err := url.Parse(issuer)
	if err != nil || uri.Host == "" || uri.User != nil || uri.RawQuery != "" || uri.Fragment != "" || uri.Opaque != "" || (uri.Scheme != "http" && uri.Scheme != "https") || uri.Path != "/realms/theia-staff" {
		return auth.ErrInvalid
	}
	id, err := uuid.Parse(subject)
	if err != nil || subject != id.String() || len(subject) != 36 {
		return auth.ErrInvalid
	}
	return nil
}

// StaffState describes only the local account allowlist; it is not evidence
// that the corresponding Keycloak user is enabled. The operator must check
// Keycloak separately before creating or reactivating this mapping.
type StaffState struct {
	ID     uuid.UUID
	Active bool
}

func lookupStaff(ctx context.Context, tx *sql.Tx, issuer, subject string) (StaffState, error) {
	var state StaffState
	var id sql.NullString
	var kind string
	err := tx.QueryRowContext(ctx, `SELECT principal_kind,CONVERT(varchar(36),staff_id)
      FROM dbo.auth_identities WITH (UPDLOCK,HOLDLOCK)
      WHERE issuer=@p1 AND subject=@p2`, issuer, subject).Scan(&kind, &id)
	if errors.Is(err, sql.ErrNoRows) {
		return state, auth.ErrNotFound
	}
	if err != nil {
		return state, err
	}
	if kind != string(auth.Staff) || !id.Valid {
		return state, auth.ErrInvalid
	}
	state.ID, err = uuid.Parse(id.String)
	if err != nil {
		return StaffState{}, auth.ErrInvalid
	}
	err = tx.QueryRowContext(ctx, `SELECT active FROM dbo.staff_accounts WITH (UPDLOCK,HOLDLOCK)
      WHERE id=CONVERT(uniqueidentifier,@p1)`, state.ID.String()).Scan(&state.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return StaffState{}, auth.ErrInvalid
	}
	return state, err
}

// State resolves a Keycloak staff subject without performing any mutation.
func (p *StaffProvisioner) State(ctx context.Context, issuer, subject string) (StaffState, error) {
	if err := ValidateStaffBinding(issuer, subject); err != nil {
		return StaffState{}, err
	}
	var state StaffState
	var idText string
	err := p.db.QueryRowContext(ctx, `SELECT CONVERT(varchar(36),i.staff_id),a.active
      FROM dbo.auth_identities AS i JOIN dbo.staff_accounts AS a ON a.id=i.staff_id
      WHERE i.issuer=@p1 AND i.subject=@p2 AND i.principal_kind='staff'`, issuer, subject).
		Scan(&idText, &state.Active)
	if errors.Is(err, sql.ErrNoRows) {
		return StaffState{}, auth.ErrNotFound
	}
	if err != nil {
		return StaffState{}, err
	}
	state.ID, err = uuid.Parse(idText)
	if err != nil {
		return StaffState{}, auth.ErrInvalid
	}
	return state, nil
}

// Provision creates an ACTIVE staff allowlist record, but only for a subject
// that the operator has already created/verified in the Keycloak staff realm.
// Repeating an identical operation is idempotent; inactive identities require
// a separate explicit Activate operation rather than silent reactivation.
func (p *StaffProvisioner) Provision(ctx context.Context, issuer, subject string) (StaffState, bool, error) {
	if err := ValidateStaffBinding(issuer, subject); err != nil {
		return StaffState{}, false, err
	}
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return StaffState{}, false, err
	}
	defer tx.Rollback()
	existing, err := lookupStaff(ctx, tx, issuer, subject)
	if err == nil {
		if !existing.Active {
			return StaffState{}, false, auth.ErrInactive
		}
		return existing, false, nil // no duplicate staff identity or audit record
	}
	if !errors.Is(err, auth.ErrNotFound) {
		return StaffState{}, false, err
	}
	state := StaffState{ID: uuid.New(), Active: true}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbo.staff_accounts(id,active)
      VALUES(CONVERT(uniqueidentifier,@p1),1)`, state.ID.String())
	if err != nil {
		return StaffState{}, false, err
	}
	_, err = tx.ExecContext(ctx, `INSERT INTO dbo.auth_identities
      (id,issuer,subject,principal_kind,staff_id)
      VALUES(CONVERT(uniqueidentifier,@p1),@p2,@p3,'staff',CONVERT(uniqueidentifier,@p4))`,
		uuid.NewString(), issuer, subject, state.ID.String())
	if err != nil {
		return StaffState{}, false, err
	}
	if err = auditStaffChange(ctx, tx, state.ID, "staff_provisioned"); err != nil {
		return StaffState{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return StaffState{}, false, err
	}
	return state, true, nil
}

// SetActive disables/reactivates a pre-provisioned staff account. Disabling
// atomically revokes ALL its active Theia sessions. Reactivation never revives
// old sessions; a new Keycloak login is required. The Keycloak user's own
// enabled state must also be managed separately by the operator.
func (p *StaffProvisioner) SetActive(ctx context.Context, issuer, subject string, active bool) (StaffState, bool, error) {
	if err := ValidateStaffBinding(issuer, subject); err != nil {
		return StaffState{}, false, err
	}
	tx, err := p.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return StaffState{}, false, err
	}
	defer tx.Rollback()
	state, err := lookupStaff(ctx, tx, issuer, subject)
	if err != nil {
		return StaffState{}, false, err
	}
	if state.Active == active {
		return state, false, nil
	}
	action := "staff_deactivated"
	if active {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.staff_accounts SET active=1,deactivated_at=NULL
          WHERE id=CONVERT(uniqueidentifier,@p1)`, state.ID.String())
		action = "staff_reactivated"
	} else {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.staff_accounts SET active=0,deactivated_at=SYSUTCDATETIME()
          WHERE id=CONVERT(uniqueidentifier,@p1)`, state.ID.String())
	}
	if err != nil {
		return StaffState{}, false, err
	}
	if !active {
		_, err = tx.ExecContext(ctx, `UPDATE dbo.auth_sessions SET revoked_at=SYSUTCDATETIME()
          WHERE staff_id=CONVERT(uniqueidentifier,@p1) AND revoked_at IS NULL`, state.ID.String())
		if err != nil {
			return StaffState{}, false, err
		}
	}
	if err = auditStaffChange(ctx, tx, state.ID, action); err != nil {
		return StaffState{}, false, err
	}
	if err = tx.Commit(); err != nil {
		return StaffState{}, false, err
	}
	state.Active = active
	return state, true, nil
}

func auditStaffChange(ctx context.Context, tx *sql.Tx, target uuid.UUID, action string) error {
	// No authenticated actor is available to a CLI that directly owns DB
	// credentials. Do not misattribute operator actions to the target staff.
	// An approved admin workflow with verified actor identity is BO/P1 work.
	_, err := tx.ExecContext(ctx, `INSERT INTO dbo.auth_audit_events
      (id,principal_kind,actor_id,target_staff_id,action_code,outcome_code,request_id)
      VALUES(CONVERT(uniqueidentifier,@p1),'staff',NULL,CONVERT(uniqueidentifier,@p2),@p3,'success',@p4)`,
		uuid.NewString(), target.String(), action, uuid.NewString())
	return err
}
