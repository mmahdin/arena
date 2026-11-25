package mssql

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"testing"
	"time"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"

	_ "github.com/microsoft/go-mssqldb"
)

func TestProductPolicyDoesNotRequireChatConsentMapping(t *testing.T) {
	db, err := sql.Open("sqlserver", "sqlserver://localhost?database=synthetic")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	for _, chatKey := range []string{"", "obsolete_chat_key"} {
		cfg := ProductPolicySettings{
			ReviewID:       "synthetic-test",
			ConsentKeys:    []string{"message_use"},
			MessageUseKey:  "message_use",
			ChatConsentKey: chatKey,
		}
		if _, err := NewProductUsePolicy(db, cfg); err != nil {
			t.Fatal("legacy chat key blocked policy configuration", err)
		}
		cfg.MessageUseKey = "unmapped_data_use"
		if _, err := NewProductUsePolicy(db, cfg); err == nil {
			t.Fatal("optional data use accepted an unmapped consent key")
		}
	}
}

func TestProductPolicyMemoryDefaultsWithoutConsentMappings(t *testing.T) {
	db := diagnosticDB(t)
	if _, err := NewProductUsePolicy(db, ProductPolicySettings{ReviewID: "synthetic"}); err != nil {
		t.Fatal("memory required consent configuration", err)
	}
}

func TestProductPolicyMemoryActionsIgnoreConsentAndPause(t *testing.T) {
	for _, action := range []mem.Action{mem.Reflect, mem.Extract, mem.UseSource, mem.Store, mem.Sensitive, mem.ExplicitChange, mem.Endorse} {
		for _, consent := range []map[string]bool{nil, {}, {"memory": false}} {
			db := diagnosticDB(t, diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "WHERE id=@p1 AND deleted_at IS NULL"})
			p, err := NewProductUsePolicy(db, ProductPolicySettings{ReviewID: "synthetic"})
			if err != nil {
				t.Fatal(err)
			}
			ok, err := p.Allow(context.Background(), mem.Permission{Action: action, User: c.UserState{UserID: c.NewID(), MemoryPaused: true, Consent: consent}})
			if err != nil || !ok {
				t.Fatalf("%s denied without consent: %v", action, err)
			}
		}
	}
}

func TestProductPolicySearchAndSourcesRequireVisibleOwner(t *testing.T) {
	for _, kind := range []retrieval.SourceKind{retrieval.SourceMessage, retrieval.SourceMemory, retrieval.SourceSession, retrieval.SourceThread, retrieval.SourceFormulation, retrieval.SourceProtocol, retrieval.SourceOpenLoop} {
		for _, visible := range []int64{0, 1} {
			query := "s.user_id=@p2"
			if kind == retrieval.SourceMessage {
				query = "v.user_id=@p2 AND m.deleted_at IS NULL AND v.deleted_at IS NULL"
			}
			queries := []diagnosticQuery{{values: []driver.Value{visible}, requireSQL: query}}
			if visible == 1 {
				queries = append(queries, diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "deleted_at IS NULL"})
			}
			p, _ := NewProductUsePolicy(diagnosticDB(t, queries...), ProductPolicySettings{ReviewID: "synthetic"})
			ok, err := p.CanUse(context.Background(), c.NewID(), kind, c.NewID())
			if err != nil || ok != (visible == 1) {
				t.Fatalf("source %s visibility=%d: %v %v", kind, visible, ok, err)
			}
		}
	}
	for _, visible := range []int64{0, 1} {
		queries := []diagnosticQuery{{values: []driver.Value{visible}, requireSQL: "t.user_id=@p2 AND v.deleted_at IS NULL"}}
		if visible == 1 {
			queries = append(queries, diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "deleted_at IS NULL"})
		}
		p, _ := NewProductUsePolicy(diagnosticDB(t, queries...), ProductPolicySettings{ReviewID: "synthetic"})
		ok, err := p.CanSearch(context.Background(), c.NewID(), c.NewID())
		if err != nil || ok != (visible == 1) {
			t.Fatalf("search visibility=%d: %v %v", visible, ok, err)
		}
	}
}

func TestProductPolicyKeepsAccountAndRetainedSourceSafeguards(t *testing.T) {
	for _, q := range []diagnosticQuery{{values: []driver.Value{int64(0)}}, {err: errors.New("synthetic failure")}} {
		p, _ := NewProductUsePolicy(diagnosticDB(t, q), ProductPolicySettings{ReviewID: "synthetic"})
		ok, _ := p.Allow(context.Background(), mem.Permission{Action: mem.Store, User: c.UserState{UserID: c.NewID()}})
		if ok {
			t.Fatal("unavailable or hidden account allowed")
		}
	}
	p, _ := NewProductUsePolicy(diagnosticDB(t), ProductPolicySettings{ReviewID: "synthetic"})
	for _, action := range []mem.Action{mem.Proactive, mem.UseRetainedSource, mem.Action("unknown")} {
		ok, err := p.Allow(context.Background(), mem.Permission{Action: action, User: c.UserState{UserID: c.NewID()}})
		if err != nil || ok {
			t.Fatalf("unguarded action allowed: %s %v", action, err)
		}
	}
	ok, err := p.Allow(context.Background(), mem.Permission{Action: mem.Store})
	if err != nil || ok {
		t.Fatal("invalid identity allowed", err)
	}
}

func TestProductControlsAlwaysReportMemoryEnabled(t *testing.T) {
	for _, paused := range []driver.Value{nil, true, false} {
		db := diagnosticDB(t, diagnosticQuery{values: []driver.Value{paused, `{}`, `{}`}, requireSQL: "u.id=@p1 AND u.deleted_at IS NULL"})
		controls, err := (&ProductRepository{db: db}).Controls(context.Background(), c.NewID())
		if err != nil || controls.MemoryPaused {
			t.Fatalf("legacy pause=%v reported disabled: %+v %v", paused, controls, err)
		}
	}
}

func TestEmbeddingInputNeedsNoConsentOrUnpausedState(t *testing.T) {
	owner, id := c.NewID(), c.NewID()
	db := diagnosticDB(t,
		diagnosticQuery{values: []driver.Value{int64(7), `{}`}, requireSQL: "u.deleted_at IS NULL"},
		diagnosticQuery{values: []driver.Value{"memory", id.String(), owner.String(), "fact", "Synthetic memory", nil, "reported", "active", "normal", time.Now(), "0000000000000001", int64(1), nil, ""}, requireSQL: "s.user_id=@p1 AND s.source_kind=@p2 AND s.id=@p3"},
		diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "s.user_id=@p2"},
		diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "deleted_at IS NULL"},
		diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "deleted_at IS NULL"},
		diagnosticQuery{values: []driver.Value{int64(1)}, requireSQL: "deleted_at IS NULL"},
	)
	policy, _ := NewProductUsePolicy(db, ProductPolicySettings{ReviewID: "synthetic"})
	repo := &MemoryRepository{db: db, policy: policy, searchUse: policy, model: "synthetic", dimension: 2}
	payload, _ := json.Marshal(mem.EmbeddingTask{Schema: "embedding_job.v2", SourceKind: "memory", SourceID: id, SourceVersion: "0000000000000001"})
	input, eligible, err := repo.EmbeddingInput(context.Background(), mem.Lease{UserID: owner, Kind: "memory_embed", Payload: payload})
	if err != nil || !eligible || input.Text != "Synthetic memory" || input.UserVersion != 7 {
		t.Fatalf("embedding required consent: %+v %v %v", input, eligible, err)
	}
}
