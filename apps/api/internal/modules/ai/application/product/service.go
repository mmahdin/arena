// Package product defines authenticated WP-16 use cases. It owns no SQL,
// model prompts or clinical policy; existing committers retain write authority.
package product

import (
	"context"
	"github.com/google/uuid"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"golang.org/x/text/language"
	"golang.org/x/text/unicode/norm"
)

type Turns interface {
	ports.Orchestrator
	RetryFailed(context.Context, c.ID, c.ID) error
	RecoverNext(context.Context, c.ID) (c.TurnResult, error)
}
type Memory interface {
	Reference() c.BundleRef
	Items(context.Context, c.ID, []c.ID) ([]c.MemoryItem, error)
	Edit(context.Context, c.ID, c.ID, c.MemoryOpProposal) (ports.MemoryApplyReport, error)
	Correct(context.Context, c.ID, c.ID, ports.CorrectionKind) error
	Delete(context.Context, c.ID, ports.DeletionScope) (ports.DeletionReport, error)
	SetControls(context.Context, c.ID, *bool, map[string]bool, *c.PrefsView) error
}
type Controls struct {
	Initialized  bool            `json:"-"`
	MemoryPaused bool            `json:"memoryPaused"`
	Consent      map[string]bool `json:"consent"`
	Preferences  c.PrefsView     `json:"-"`
}
type ExportMessage struct {
	ID             c.ID      `json:"id"`
	ConversationID c.ID      `json:"conversationId"`
	TurnID         c.ID      `json:"turnId"`
	Role           c.Role    `json:"role"`
	Content        string    `json:"content"`
	CreatedAt      time.Time `json:"createdAt"`
}
type Repository interface {
	Ready(context.Context, c.ID) error
	ValidateOwner(context.Context, c.ID, c.ID) error
	TurnInConversation(context.Context, c.ID, c.ID, c.ID) error
	MessageVisible(context.Context, c.ID, c.ID) error
	MemoryIDs(context.Context, c.ID, string, int) ([]c.ID, string, error)
	MemoryOwned(context.Context, c.ID, c.ID) error // includes retained hidden items for idempotent delete
	GetTurn(context.Context, c.ID, c.ID) (c.TurnResult, error)
	Controls(context.Context, c.ID) (Controls, error)
	EnsureState(context.Context, c.ID) error
	ExportMessages(context.Context, c.ID, string, int) ([]ExportMessage, string, error)
}
type Accounts interface {
	HideAccount(context.Context, c.ID) error
}
type Options struct {
	EmbeddingInvalidator ports.EmbeddingInvalidator
	ConsentKeys          []string
	// Deprecated: accepted for compatibility; AI chat requires no separate consent flag.
	ChatConsentKey string
}
type Service struct {
	embeddings ports.EmbeddingInvalidator
	turns      Turns
	memory     Memory
	repo       Repository
	accounts   Accounts
	keys       []string
}

func New(turns Turns, m Memory, r Repository, a Accounts, opt Options) (*Service, error) {
	if turns == nil || m == nil || r == nil || a == nil || len(opt.ConsentKeys) > 32 {
		return nil, mem.ErrAssets
	}
	keys := append([]string{}, opt.ConsentKeys...)
	sort.Strings(keys)
	for i, k := range keys {
		if strings.TrimSpace(k) != k || k == "" || len(k) > 80 || i > 0 && keys[i-1] == k {
			return nil, mem.ErrAssets
		}
	}
	return &Service{turns: turns, memory: m, repo: r, accounts: a, keys: keys, embeddings: opt.EmbeddingInvalidator}, nil
}
func contains(xs []string, s string) bool {
	for _, x := range xs {
		if x == s {
			return true
		}
	}
	return false
}
func (s *Service) Keys() []string { return append([]string{}, s.keys...) }
func (s *Service) meta(ctx context.Context, u, op c.ID) (context.Context, error) {
	if !c.ValidID(u) || !c.ValidID(op) {
		return nil, httpx.ErrInvalid
	}
	return mem.WithMeta(ctx, mem.Meta{UserID: u, OperationID: op, Bundle: s.memory.Reference()}), nil
}
func (s *Service) Consent(ctx context.Context, u c.ID) (Controls, error) {
	return s.repo.Controls(ctx, u)
}
func (s *Service) authorizeTurn(ctx context.Context, u, conv c.ID) error {
	if !c.ValidID(u) || !c.ValidID(conv) {
		return httpx.ErrInvalid
	}
	if e := s.repo.ValidateOwner(ctx, u, conv); e != nil {
		return e
	}
	if e := s.repo.Ready(ctx, u); e != nil {
		return e
	}
	return nil
}

func (s *Service) Process(ctx context.Context, r c.TurnRequest) (c.TurnResult, error) {
	if e := s.authorizeTurn(ctx, r.UserID, r.ConversationID); e != nil {
		return c.TurnResult{}, e
	}
	if r.Origin != c.OriginUser || r.WakeUp != nil || !c.ValidID(r.ClientTurnID) || !utf8.ValidString(r.Text) || utf8.RuneCountInString(r.Text) > 4000 || strings.TrimSpace(r.Text) == "" {
		return c.TurnResult{}, httpx.ErrInvalid
	}
	r.Text = norm.NFC.String(r.Text)
	if r.Locale == "" {
		r.Locale = "fa"
	}
	if r.TZ == "" {
		r.TZ = "UTC"
	}
	if _, e := language.Parse(r.Locale); e != nil || len(r.Locale) > 24 {
		return c.TurnResult{}, httpx.ErrInvalid
	}
	if _, e := time.LoadLocation(r.TZ); e != nil || len(r.TZ) > 100 {
		return c.TurnResult{}, httpx.ErrInvalid
	}
	for _, ch := range r.Text {
		if ch == 0 || ch < 32 && ch != '\n' && ch != '\r' && ch != '\t' {
			return c.TurnResult{}, httpx.ErrInvalid
		}
	}
	r.Channel = "app_chat"
	if r.ReplyToID != nil {
		if !c.ValidID(*r.ReplyToID) {
			return c.TurnResult{}, httpx.ErrInvalid
		}
		if e := s.repo.MessageVisible(ctx, r.UserID, *r.ReplyToID); e != nil {
			return c.TurnResult{}, e
		}
	}
	return s.turns.ProcessTurn(ctx, r)
}
func (s *Service) Status(ctx context.Context, u, conv, client c.ID) (c.TurnResult, error) {
	if e := s.repo.TurnInConversation(ctx, u, conv, client); e != nil {
		return c.TurnResult{}, e
	}
	return s.repo.GetTurn(ctx, u, client)
}
func (s *Service) Retry(ctx context.Context, u, conv, client c.ID) (c.TurnResult, error) {
	if e := s.authorizeTurn(ctx, u, conv); e != nil {
		return c.TurnResult{}, e
	}
	prev, e := s.Status(ctx, u, conv, client)
	if e != nil {
		return prev, e
	}
	if prev.Status == c.TurnFailed || prev.Status == c.TurnCancelled {
		if e = s.turns.RetryFailed(ctx, u, client); e != nil {
			return c.TurnResult{}, e
		}
	}
	return s.Status(ctx, u, conv, client)
}
func (s *Service) Recover(ctx context.Context, u c.ID) (c.TurnResult, error) {
	return s.turns.RecoverNext(ctx, u)
}
func (s *Service) SetControls(ctx context.Context, u, op c.ID, pause *bool, consent map[string]bool, prefs *c.PrefsView) error {
	ctx, e := s.meta(ctx, u, op)
	if e != nil {
		return e
	}
	if pause == nil && len(consent) == 0 && prefs == nil {
		return httpx.ErrInvalid
	}
	for k := range consent {
		if !contains(s.keys, k) {
			return httpx.ErrInvalid
		}
	}
	if prefs != nil {
		if prefs.Language != "" {
			if _, e := language.Parse(prefs.Language); e != nil || len(prefs.Language) > 24 {
				return httpx.ErrInvalid
			}
		}
		if len(prefs.AddressForm) > 80 {
			return httpx.ErrInvalid
		}
		for _, v := range append(append([]string{}, prefs.Avoid...), prefs.Likes...) {
			if !utf8.ValidString(v) || utf8.RuneCountInString(v) > 160 {
				return httpx.ErrInvalid
			}
		}
	}
	if e = s.repo.EnsureState(ctx, u); e != nil {
		return e
	}
	return s.memory.SetControls(ctx, u, pause, consent, prefs)
}

func (s *Service) Correct(ctx context.Context, u, id, op c.ID, kind ports.CorrectionKind) error {
	if !kind.Valid() {
		return httpx.ErrInvalid
	}
	if e := s.repo.MemoryOwned(ctx, u, id); e != nil {
		return e
	}
	ctx, e := s.meta(ctx, u, op)
	if e != nil {
		return e
	}
	return s.memory.Correct(ctx, u, id, kind)
}
func (s *Service) HideMemory(ctx context.Context, u, id, op c.ID, all bool) error {
	if !all {
		if e := s.repo.MemoryOwned(ctx, u, id); e != nil {
			return e
		}
	}
	ctx, e := s.meta(ctx, u, op)
	if e != nil {
		return e
	}
	scope := ports.DeletionScope{All: all, Reason: "user_request"}
	if !all {
		scope.ItemIDs = []c.ID{id}
	}
	_, e = s.memory.Delete(ctx, u, scope)
	return e
}
func (s *Service) HideMessage(ctx context.Context, u, id, op c.ID) error {
	// SQL ownership includes hidden resources so repeating a delete remains safe.
	ctx, e := s.meta(ctx, u, op)
	if e != nil {
		return e
	}
	_, e = s.memory.Delete(ctx, u, ports.DeletionScope{MessageIDs: []c.ID{id}, Reason: "user_request"})
	return e
}
func (s *Service) Edit(ctx context.Context, u, id, op c.ID, content string, ids []c.ID) (c.MemoryItem, error) {
	if len(ids) < 1 || len(ids) > 6 || !utf8.ValidString(content) || strings.TrimSpace(content) == "" || len(content) > 1280 {
		return c.MemoryItem{}, httpx.ErrInvalid
	}
	ctx, e := s.meta(ctx, u, op)
	if e != nil {
		return c.MemoryItem{}, e
	}
	items, e := s.memory.Items(ctx, u, []c.ID{id})
	if e != nil {
		return c.MemoryItem{}, e
	}
	var target *c.MemoryItem
	for i := range items {
		if items[i].ID == id {
			target = &items[i]
			break
		}
	}
	if target == nil {
		return c.MemoryItem{}, httpx.ErrNotFound
	}
	for _, evidence := range ids {
		if !c.ValidID(evidence) {
			return c.MemoryItem{}, httpx.ErrInvalid
		}
		if e = s.repo.MessageVisible(ctx, u, evidence); e != nil {
			return c.MemoryItem{}, e
		}
	}
	report, e := s.memory.Edit(ctx, u, id, c.MemoryOpProposal{Kind: target.Kind, Content: norm.NFC.String(content), Epistemic: target.Epistemic, EvidenceMsg: ids, Sensitivity: target.Sensitivity, Tags: target.Tags, Entities: target.Entities, Importance: &target.Importance})
	if e != nil {
		return c.MemoryItem{}, e
	}
	if len(report.Rejected) > 0 {
		return c.MemoryItem{}, httpx.ErrInvalid
	}
	items, e = s.memory.Items(ctx, u, report.Applied)
	if e != nil {
		return c.MemoryItem{}, e
	}
	for _, item := range items {
		for _, newID := range report.Applied {
			if item.ID == newID {
				return item, nil
			}
		}
	}
	return c.MemoryItem{}, httpx.ErrConflict
}
func (s *Service) HideAccount(ctx context.Context, u c.ID) error {
	if e := s.accounts.HideAccount(ctx, u); e != nil {
		return e
	}
	if s.embeddings != nil {
		s.embeddings.InvalidateEmbeddingUser(u)
	}
	return nil
}
func (s *Service) ExportMessages(ctx context.Context, u c.ID, after string, limit int) ([]ExportMessage, string, error) {
	return s.repo.ExportMessages(ctx, u, after, limit)
}

func (s *Service) MemoryPage(ctx context.Context, u c.ID, cursor string, n int) ([]c.MemoryItem, string, error) {
	if n < 1 || n > 100 {
		return nil, "", httpx.ErrInvalid
	}
	if cursor != "" {
		id, e := uuid.Parse(cursor)
		if e != nil || !c.ValidID(id) {
			return nil, "", httpx.ErrInvalid
		}
		cursor = id.String()
	}
	state, e := s.repo.Controls(ctx, u)
	if e != nil {
		return nil, "", e
	}
	if !state.Initialized {
		return []c.MemoryItem{}, "", nil
	}
	ids, next, e := s.repo.MemoryIDs(ctx, u, cursor, n)
	if e != nil {
		return nil, "", e
	}
	if len(ids) == 0 {
		return []c.MemoryItem{}, "", nil
	}
	ctx, e = s.meta(ctx, u, c.NewID())
	if e != nil {
		return nil, "", e
	}
	items, e := s.memory.Items(ctx, u, ids)
	return items, next, e
}
