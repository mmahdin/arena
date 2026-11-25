package mssql

import (
	"context"
	"database/sql"
	"encoding/json"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"strings"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

// Settings retain legacy consent mappings for deployment compatibility.
// Ordinary memory use requires no consent; retained hidden sources remain gated.
type ProductPolicySettings struct {
	ReviewID    string   `json:"reviewId"`
	ConsentKeys []string `json:"consentKeys"`
	// Deprecated: accepted for deployment compatibility; never gates AI chat.
	ChatConsentKey string                `json:"chatConsentKey"`
	SearchKey      string                `json:"searchKey"`
	MessageUseKey  string                `json:"messageUseKey"`
	MemoryUseKey   string                `json:"memoryUseKey"`
	Actions        map[mem.Action]string `json:"actions"`
}
type ProductUsePolicy struct {
	db  *sql.DB
	cfg ProductPolicySettings
}

func NewProductUsePolicy(db *sql.DB, cfg ProductPolicySettings) (*ProductUsePolicy, error) {
	if db == nil || strings.TrimSpace(cfg.ReviewID) == "" || len(cfg.ConsentKeys) > 32 {
		return nil, mem.ErrAssets
	}
	keys := map[string]bool{}
	for _, k := range cfg.ConsentKeys {
		if k == "" || len(k) > 80 || strings.TrimSpace(k) != k || keys[k] {
			return nil, mem.ErrAssets
		}
		keys[k] = true
	}
	for _, k := range []string{cfg.MessageUseKey, cfg.SearchKey, cfg.MemoryUseKey} {
		if k != "" && !keys[k] {
			return nil, mem.ErrAssets
		}
	}
	for action, k := range cfg.Actions {
		switch action {
		case mem.Reflect, mem.Extract, mem.UseSource, mem.UseRetainedSource, mem.Store, mem.Sensitive, mem.ExplicitChange, mem.Endorse:
		default:
			return nil, mem.ErrAssets
		}
		if !keys[k] {
			return nil, mem.ErrAssets
		}
	}
	raw, _ := json.Marshal(cfg)
	_ = json.Unmarshal(raw, &cfg)
	return &ProductUsePolicy{db, cfg}, nil
}
func (p *ProductUsePolicy) grant(ctx context.Context, u c.ID, key string) (bool, error) {
	allowed, _, err := p.explainGrant(ctx, u, key)
	return allowed, err
}
func (p *ProductUsePolicy) explainGrant(ctx context.Context, u c.ID, key string) (bool, string, error) {
	if key == "" {
		return false, "policy_mapping_missing", nil
	}
	if !c.ValidID(u) {
		return false, "identity_mismatch", nil
	}
	var raw sql.NullString
	e := p.db.QueryRowContext(ctx, `SELECT s.consent_json FROM dbo.users u LEFT JOIN dbo.ai_user_state s ON s.user_id=u.id WHERE u.id=@p1 AND u.deleted_at IS NULL`, u).Scan(&raw)
	if e == sql.ErrNoRows {
		return false, "resource_not_visible", nil
	}
	if e != nil {
		return false, "policy_query_failed", observability.Detail(e, "policy_query_failed", "consent_state", "consent_read")
	}
	return consentDecision(raw, key)
}
func consentDecision(raw sql.NullString, key string) (bool, string, error) {
	if !raw.Valid {
		return false, "consent_missing", nil
	}
	v := map[string]bool{}
	if json.Unmarshal([]byte(raw.String), &v) != nil {
		return false, "consent_invalid", mem.ErrDenied
	}
	allowed, exists := v[key]
	if !exists {
		return false, "consent_missing", nil
	}
	// A persisted false is not proof of revocation; the state has no history.
	if !allowed {
		return false, "consent_not_granted", nil
	}
	return true, "", nil
}
func (p *ProductUsePolicy) accountActive(ctx context.Context, u c.ID) (bool, error) {
	if !c.ValidID(u) {
		return false, nil
	}
	var n int
	err := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.users WHERE id=@p1 AND deleted_at IS NULL`, u).Scan(&n)
	return n == 1 && err == nil, err
}
func (p *ProductUsePolicy) CanSearch(ctx context.Context, u, turn c.ID) (bool, error) {
	var n int
	e := p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.turns t JOIN dbo.conversations v ON v.id=t.conversation_id AND v.user_id=t.user_id WHERE t.id=@p1 AND t.user_id=@p2 AND v.deleted_at IS NULL`, turn, u).Scan(&n)
	if e != nil || n != 1 {
		return false, e
	}
	return p.accountActive(ctx, u)
}
func (p *ProductUsePolicy) CanUse(ctx context.Context, u c.ID, kind retrieval.SourceKind, id c.ID) (bool, error) {
	allowed, _, err := p.ExplainUse(ctx, u, kind, id)
	return allowed, err
}
func (p *ProductUsePolicy) ExplainUse(ctx context.Context, u c.ID, kind retrieval.SourceKind, id c.ID) (bool, string, error) {
	var n int
	var e error
	switch kind {
	case retrieval.SourceMessage:
		e = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.messages m JOIN dbo.conversations v ON v.id=m.conversation_id WHERE m.id=@p1 AND v.user_id=@p2 AND m.deleted_at IS NULL AND v.deleted_at IS NULL`, id, u).Scan(&n)
	case retrieval.SourceMemory, retrieval.SourceSession, retrieval.SourceThread, retrieval.SourceFormulation, retrieval.SourceProtocol, retrieval.SourceOpenLoop:
		e = p.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM dbo.ai_search_sources s WHERE s.source_kind=@p3 AND s.id=@p1 AND s.user_id=@p2`, id, u, kind).Scan(&n)
	default:
		return false, "policy_denied", nil
	}
	if e != nil {
		return false, "policy_query_failed", observability.Detail(e, "policy_query_failed", "source_visibility", "source_use")
	}
	if n != 1 {
		return false, "resource_not_visible", nil
	}
	active, err := p.accountActive(ctx, u)
	if err != nil {
		return false, "policy_query_failed", observability.Detail(err, "policy_query_failed", "account_visibility", "source_use")
	}
	if !active {
		return false, "resource_not_visible", nil
	}
	return true, "", nil
}
func (p *ProductUsePolicy) Allow(ctx context.Context, v mem.Permission) (bool, error) {
	switch v.Action {
	case mem.Reflect, mem.Extract, mem.UseSource, mem.Store, mem.Sensitive, mem.ExplicitChange, mem.Endorse:
		return p.accountActive(ctx, v.User.UserID)
	case mem.UseRetainedSource:
		// Soft-hidden evidence must not gain an implicit reuse grant.
		key := p.cfg.Actions[v.Action]
		if key == "" || !v.User.Consent[key] {
			return false, nil
		}
		return p.grant(ctx, v.User.UserID, key)
	default:
		return false, nil
	}
}
