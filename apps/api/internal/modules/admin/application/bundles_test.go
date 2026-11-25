package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/admin/domain"
)

func TestBundlesAuditedImmutableAndFailClosed(t *testing.T) {
	audit := &inMemoryAudit{}
	synthetic := true
	source := domain.BundleInventory{Scope: "deployment_snapshot", AsOf: time.Now().UTC(),
		Active: &domain.BundleRecord{Slot: "active", Status: "synthetic_draft", Version: "kb@0.1.0", SyntheticOnly: &synthetic,
			ConversationLanguages: []string{"en", "fa"}, EmitsSchemas: map[string]string{"brain": "brain.v1"},
			Validation: domain.BundleValidation{Valid: true, WarningCodes: []string{"synthetic_only"}}}}
	svc := NewWithAudit(audit)
	svc.SetBundleInventory(source)
	source.Active.ConversationLanguages[0] = "secret"
	source.Active.EmitsSchemas["brain"] = "other"
	source.Active.Validation.WarningCodes[0] = "private"
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	req := uuid.New()
	report, err := svc.ListBundles(context.Background(), p, req)
	if err != nil || report.Active == nil || report.Active.Version != "kb@0.1.0" || report.Active.ConversationLanguages[0] != "en" || report.Active.EmitsSchemas["brain"] != "brain.v1" || report.Active.Validation.WarningCodes[0] != "synthetic_only" {
		t.Fatalf("invalid snapshot: %+v: %v", report, err)
	}
	if len(audit.events) != 1 || audit.events[0].ActionCode != domain.ActionBundlesRead || audit.events[0].ResourceKind != "bundle" {
		t.Fatal("missing sanitized read audit")
	}
	report.Active.ConversationLanguages[0] = "mutated"
	report.Active.EmitsSchemas["brain"] = "mutated"
	report.Active.Validation.WarningCodes[0] = "mutated"
	again, _ := svc.ListBundles(context.Background(), p, req)
	if again.Active.ConversationLanguages[0] != "en" || again.Active.Validation.WarningCodes[0] != "synthetic_only" || again.Active.EmitsSchemas["brain"] != "brain.v1" {
		t.Fatal("response escaped immutable catalog")
	}
	p.ExpiresAt = time.Now().Add(-time.Second)
	if _, err = svc.ListBundles(context.Background(), p, req); !errors.Is(err, ErrUnverifiedStaff) || len(audit.events) != 2 {
		t.Fatal("inactive staff read was recorded")
	}
	p.ExpiresAt = time.Now().Add(time.Hour)
	audit.appendErr = errors.New("private db error with credentials")
	if _, err = svc.ListBundles(context.Background(), p, req); !errors.Is(err, ErrAuditUnavailable) {
		t.Fatal("audit did not fail closed")
	}
}

func TestBundlesMissingRuntimeIsNotInvented(t *testing.T) {
	svc := NewWithAudit(&inMemoryAudit{})
	svc.SetBundleInventory(domain.BundleInventory{})
	p := domain.Principal{StaffID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour)}
	inventory, err := svc.ListBundles(context.Background(), p, uuid.New())
	if err != nil || inventory.Scope != "not_configured" || inventory.Active != nil || inventory.Candidates == nil || len(inventory.Candidates) != 0 || inventory.AsOf.IsZero() {
		t.Fatalf("unconfigured runtime misreported: %+v %v", inventory, err)
	}
}
