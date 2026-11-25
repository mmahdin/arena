// Package httpadapter maps authenticated WP-16 JSON onto typed core use cases.
// It never accepts owner/origin fields or exposes internal token streams.
package httpadapter

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	mem "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/memory"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/product"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/httpx"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/observability"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/principal"
)

type Handler struct{ s *product.Service }

func New(s *product.Service) *Handler { return &Handler{s} }
func (h *Handler) Available(ctx *gin.Context) bool {
	if h.s == nil {
		httpx.Failure(ctx, 503, "AI_CORE_UNAVAILABLE")
		return false
	}
	return true
}
func id(ctx *gin.Context, name string) (c.ID, bool) {
	v, e := uuid.Parse(ctx.Param(name))
	if e != nil || !c.ValidID(v) {
		httpx.Failure(ctx, 400, "INVALID_ID")
		return c.ID{}, false
	}
	return v, true
}
func command(ctx *gin.Context) (c.ID, bool) {
	v, e := uuid.Parse(ctx.GetHeader("Idempotency-Key"))
	if e != nil || !c.ValidID(v) {
		httpx.Failure(ctx, 400, "INVALID_IDEMPOTENCY_KEY")
		return c.ID{}, false
	}
	return v, true
}
func failure(ctx *gin.Context, e error) {
	status, code := 503, "AI_CORE_UNAVAILABLE"
	var ce *c.Error
	var boundary interface{ PreservePublicCode() bool }
	preserveCode := errors.As(e, &boundary) && boundary.PreservePublicCode()
	switch {
	case !preserveCode && errors.Is(e, httpx.ErrOnboardingRequired):
		status, code = 409, "ONBOARDING_REQUIRED"
	case !preserveCode && (errors.Is(e, httpx.ErrNotFound) || errors.Is(e, mem.ErrDenied)):
		status, code = 404, "RESOURCE_NOT_FOUND"
	case !preserveCode && errors.Is(e, httpx.ErrInvalid):
		status, code = 400, "VALIDATION_ERROR"
	case !preserveCode && (errors.Is(e, mem.ErrConflict) || errors.Is(e, httpx.ErrConflict)):
		status, code = 409, "STATE_CONFLICT"
	case !preserveCode && (errors.Is(e, context.Canceled) || errors.Is(e, context.DeadlineExceeded)):
		status, code = 503, "TURN_INTERRUPTED"
	case errors.As(e, &ce):
		switch ce.Code {
		case c.ErrValidation:
			status, code = 400, "VALIDATION_ERROR"
		case c.ErrForbidden:
			status, code = 404, "RESOURCE_NOT_FOUND"
		case c.ErrDuplicate:
			status, code = 409, "IDEMPOTENCY_CONFLICT"
		case c.ErrStateConflict:
			status, code = 409, "STATE_CONFLICT"
		case c.ErrRateLimited:
			status, code = 429, "RATE_LIMITED"
		}
	}
	if status == 503 || status == 429 {
		ctx.Header("Retry-After", "2")
	}
	if ce == nil {
		_ = errors.As(e, &ce)
	}
	if status == 404 {
		classification := "resource_not_visible"
		if errors.Is(e, mem.ErrDenied) || ce != nil && ce.Code == c.ErrForbidden {
			classification = "authorization_concealed"
		}
		ctx.Set("http_error_kind", classification)
	}
	if ce != nil && ce.Code.Valid() {
		ctx.Set("ai_code", string(ce.Code))
		// Step is an internal fixed stage; never log free-form error strings.
		switch ce.Step {
		case "generator_input", "generator_render", "generator_locale", "generator_crisis", "brain_render", "generator_identity", "generator_sources", "generator_context", "generator_stream", "generator_guard", "generator_format", "brain_context", "policy_gate", "retrieval_hop", "state_validate", "admit", "recent", "safety", "retrieval", "assembler", "route", "brain", "generator", "validator", "fallback", "finalizer", "recovery", "cancel", "bundle", "state_read", "protocol_shortlist", "brain_dependency", "memory_correction", "protocol_integration", "protocol_requirements", "protocol_safety", "risk_delta", "protocol_commit", "generator_integration", "brain_integration", "crisis", "proactive", "outage_fallback", "stream_guard", "stream_size", "validator_bundle", "generator_bundle", "protocol_bundle", "reflection_bundle", "memory_bundle", "wakeup_contract", "deadline", "turn", "persistence":
			ctx.Set("ai_step", ce.Step)
		}
	}
	httpx.FailureWithError(ctx, status, code, e)
}

type turnOutput struct {
	TurnID             c.ID         `json:"turnId"`
	UserMessageID      c.ID         `json:"userMessageId"`
	AssistantMessageID *c.ID        `json:"assistantMessageId,omitempty"`
	Text               string       `json:"text,omitempty"`
	Status             c.TurnStatus `json:"status"`
	Route              c.Route      `json:"route,omitempty"`
	Degraded           bool         `json:"degraded"`
	Safety             any          `json:"safety,omitempty"`
	UI                 any          `json:"protocolUi,omitempty"`
}

func result(ctx *gin.Context, r c.TurnResult, poll bool) {
	if !r.Status.Valid() || r.Route == c.RouteProactive {
		httpx.Failure(ctx, 503, "AI_CORE_UNAVAILABLE")
		return
	}
	status := 200
	if !poll && (r.Status == c.TurnQueued || r.Status == c.TurnRunning) {
		status = 202
		ctx.Header("Retry-After", "2")
	}
	out := turnOutput{TurnID: r.TurnID, UserMessageID: r.UserMessageID, AssistantMessageID: r.AssistantMessageID, Text: r.Text, Status: r.Status, Route: r.Route, Degraded: r.Degraded}
	if r.Status != c.TurnCompleted {
		out.Text = ""
		out.AssistantMessageID = nil
		out.Route = ""
	} // no drafts even from a failed adapter
	if r.Safety != nil {
		resources := []gin.H{}
		for _, v := range r.Safety.Resources {
			resources = append(resources, gin.H{"id": v.ID, "name": v.Name, "kind": v.Kind, "contact": v.Contact, "locale": v.Locale, "hours": v.Hours, "verifiedOn": v.VerifiedOn})
		}
		out.Safety = gin.H{"level": r.Safety.Level, "resources": resources}
	}
	if r.UI != nil {
		ui := gin.H{"protocolInstanceId": r.UI.ProtocolInstanceID, "stepId": r.UI.StepID, "widget": r.UI.Widget}
		if r.UI.Params != nil {
			ui["params"] = r.UI.Params
		}
		out.UI = ui
	}
	ctx.JSON(status, out)
}
func (h *Handler) Turn(ctx *gin.Context) {
	conv, ok := id(ctx, "id")
	if !ok || !h.Available(ctx) {
		return
	}
	var b struct {
		ClientID c.ID       `json:"clientTurnId"`
		Message  string     `json:"message"`
		Locale   string     `json:"locale"`
		TZ       string     `json:"timeZone"`
		TS       *time.Time `json:"clientTimestamp"`
		Reply    *c.ID      `json:"replyToId"`
	}
	if !httpx.StrictJSON(ctx, &b) {
		return
	}
	r, e := h.s.Process(ctx.Request.Context(), c.TurnRequest{UserID: principal.UserID(ctx), ConversationID: conv, ClientTurnID: b.ClientID, Text: b.Message, Locale: b.Locale, TZ: b.TZ, ClientTS: b.TS, ReplyToID: b.Reply, Origin: c.OriginUser})
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.Set("turn_ref", observability.TurnRef(r.TurnID))
	result(ctx, r, false)
}
func (h *Handler) Status(ctx *gin.Context) {
	conv, ok := id(ctx, "id")
	if !ok {
		return
	}
	client, ok := id(ctx, "clientTurnId")
	if !ok || !h.Available(ctx) {
		return
	}
	r, e := h.s.Status(ctx.Request.Context(), principal.UserID(ctx), conv, client)
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.Set("turn_ref", observability.TurnRef(r.TurnID))
	result(ctx, r, true)
}
func (h *Handler) Retry(ctx *gin.Context) {
	conv, ok := id(ctx, "id")
	if !ok {
		return
	}
	client, ok := id(ctx, "clientTurnId")
	if !ok || !h.Available(ctx) {
		return
	}
	if !emptyBody(ctx) {
		return
	}
	r, e := h.s.Retry(ctx.Request.Context(), principal.UserID(ctx), conv, client)
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.Header("Retry-After", "2")
	ctx.Set("turn_ref", observability.TurnRef(r.TurnID))
	ctx.JSON(202, turnOutput{TurnID: r.TurnID, UserMessageID: r.UserMessageID, Status: r.Status, Degraded: r.Degraded})
}
func (h *Handler) Recover(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	if !emptyBody(ctx) {
		return
	}
	r, e := h.s.Recover(ctx.Request.Context(), principal.UserID(ctx))
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.Set("turn_ref", observability.TurnRef(r.TurnID))
	result(ctx, r, false)
}

type preferences struct {
	Length   c.Length `json:"length,omitempty"`
	Tone     c.Tone   `json:"tone,omitempty"`
	Language string   `json:"language,omitempty"`
	Address  string   `json:"addressForm,omitempty"`
	Avoid    []string `json:"avoid,omitempty"`
	Likes    []string `json:"likes,omitempty"`
}

func pref(v c.PrefsView) preferences {
	return preferences{v.Length, v.Tone, v.Language, v.AddressForm, v.Avoid, v.Likes}
}
func (h *Handler) controls(ctx *gin.Context) (gin.H, error) {
	v, e := h.s.Consent(ctx.Request.Context(), principal.UserID(ctx))
	return gin.H{"memoryPaused": v.MemoryPaused, "consent": v.Consent, "allowedConsentKeys": h.s.Keys(), "preferences": pref(v.Preferences), "proactiveEnabled": false}, e
}
func (h *Handler) Controls(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	v, e := h.controls(ctx)
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.JSON(200, v)
}
func (h *Handler) SetControls(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	var b struct {
		Operation c.ID            `json:"operationId"`
		Pause     *bool           `json:"memoryPaused"`
		Consent   map[string]bool `json:"consent"`
		Prefs     *preferences    `json:"preferences"`
	}
	if !httpx.StrictJSON(ctx, &b) {
		return
	}
	var p *c.PrefsView
	if b.Prefs != nil {
		p = &c.PrefsView{Length: b.Prefs.Length, Tone: b.Prefs.Tone, Language: b.Prefs.Language, AddressForm: b.Prefs.Address, Avoid: b.Prefs.Avoid, Likes: b.Prefs.Likes}
	}
	if e := h.s.SetControls(ctx.Request.Context(), principal.UserID(ctx), b.Operation, b.Pause, b.Consent, p); e != nil {
		failure(ctx, e)
		return
	}
	h.Controls(ctx)
}
func memoryItem(v c.MemoryItem) gin.H {
	ev := []gin.H{}
	for _, e := range v.Evidence {
		ev = append(ev, gin.H{"kind": e.Kind, "ref": e.Ref, "role": e.Role, "at": e.At})
	}
	out := gin.H{"id": v.ID, "kind": v.Kind, "content": v.Content, "language": v.Lang, "epistemic": v.Epistemic, "status": v.Status, "confidence": v.Confidence, "supportCount": v.SupportN, "contradictCount": v.ContradictN, "sessionCount": v.SessionsN, "evidence": ev, "source": gin.H{"origin": v.Source.Origin, "model": v.Source.Model, "promptVersion": v.Source.PromptVersion}, "createdAt": v.CreatedAt, "updatedAt": v.UpdatedAt}
	if v.Supersedes != nil {
		out["supersedesId"] = v.Supersedes
	}
	if v.SupersededBy != nil {
		out["supersededById"] = v.SupersededBy
	}
	return out
}
func page(ctx *gin.Context) (int, string, bool) {
	n := 50
	if q := ctx.Query("limit"); q != "" {
		v, e := strconv.Atoi(q)
		if e != nil || v < 1 || v > 100 {
			httpx.Failure(ctx, 400, "VALIDATION_ERROR")
			return 0, "", false
		}
		n = v
	}
	return n, ctx.Query("cursor"), true
}
func (h *Handler) memoryPage(ctx *gin.Context) (gin.H, error) {
	n, cursor, ok := page(ctx)
	if !ok {
		return nil, httpx.ErrInvalid
	}
	if cursor != "" {
		v, e := uuid.Parse(cursor)
		if e != nil || !c.ValidID(v) {
			return nil, httpx.ErrInvalid
		}
		cursor = v.String()
	}
	vs, next, e := h.s.MemoryPage(ctx.Request.Context(), principal.UserID(ctx), cursor, n)
	if e != nil {
		return nil, e
	}
	out := []gin.H{}
	for _, v := range vs {
		out = append(out, memoryItem(v))
	}
	r := gin.H{"items": out}
	if next != "" {
		r["nextCursor"] = next
	}
	return r, nil
}
func (h *Handler) Memories(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	v, e := h.memoryPage(ctx)
	if e != nil {
		if !ctx.Writer.Written() {
			failure(ctx, e)
		}
		return
	}
	ctx.JSON(200, v)
}
func (h *Handler) EditMemory(ctx *gin.Context) {
	v, ok := id(ctx, "memoryId")
	if !ok || !h.Available(ctx) {
		return
	}
	var b struct {
		Operation c.ID   `json:"operationId"`
		Content   string `json:"content"`
		Evidence  []c.ID `json:"evidenceMessageIds"`
	}
	if !httpx.StrictJSON(ctx, &b) {
		return
	}
	out, e := h.s.Edit(ctx.Request.Context(), principal.UserID(ctx), v, b.Operation, b.Content, b.Evidence)
	if e != nil {
		failure(ctx, e)
		return
	}
	ctx.JSON(200, memoryItem(out))
}
func (h *Handler) CorrectMemory(ctx *gin.Context) {
	v, ok := id(ctx, "memoryId")
	if !ok || !h.Available(ctx) {
		return
	}
	var b struct {
		Operation c.ID                 `json:"operationId"`
		Kind      ports.CorrectionKind `json:"kind"`
	}
	if !httpx.StrictJSON(ctx, &b) {
		return
	}
	if e := h.s.Correct(ctx.Request.Context(), principal.UserID(ctx), v, b.Operation, b.Kind); e != nil {
		failure(ctx, e)
		return
	}
	ctx.Status(204)
}
func (h *Handler) HideMemory(ctx *gin.Context) {
	v, ok := id(ctx, "memoryId")
	if !ok || !h.Available(ctx) {
		return
	}
	op, ok := command(ctx)
	if !ok {
		return
	}
	if e := h.s.HideMemory(ctx.Request.Context(), principal.UserID(ctx), v, op, false); e != nil {
		failure(ctx, e)
		return
	}
	ctx.Status(204)
}
func (h *Handler) HideMemories(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	op, ok := command(ctx)
	if !ok {
		return
	}
	if e := h.s.HideMemory(ctx.Request.Context(), principal.UserID(ctx), c.ID{}, op, true); e != nil {
		failure(ctx, e)
		return
	}
	ctx.Status(204)
}
func (h *Handler) HideMessage(ctx *gin.Context) {
	v, ok := id(ctx, "messageId")
	if !ok || !h.Available(ctx) {
		return
	}
	op, ok := command(ctx)
	if !ok {
		return
	}
	if e := h.s.HideMessage(ctx.Request.Context(), principal.UserID(ctx), v, op); e != nil {
		failure(ctx, e)
		return
	}
	ctx.Status(204)
}
func (h *Handler) HideAccount(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	if e := h.s.HideAccount(ctx.Request.Context(), principal.UserID(ctx)); e != nil {
		failure(ctx, e)
		return
	}
	ctx.Status(204)
}
func (h *Handler) Export(ctx *gin.Context) {
	if !h.Available(ctx) {
		return
	}
	var out gin.H
	var e error
	switch ctx.Query("resource") {
	case "memories":
		out, e = h.memoryPage(ctx)
	case "controls":
		var controls gin.H
		controls, e = h.controls(ctx)
		out = gin.H{"items": []gin.H{controls}}
	case "messages":
		n, cursor, ok := page(ctx)
		if !ok {
			return
		}
		var vs []product.ExportMessage
		var next string
		vs, next, e = h.s.ExportMessages(ctx.Request.Context(), principal.UserID(ctx), cursor, n)
		out = gin.H{"items": vs}
		if next != "" {
			out["nextCursor"] = next
		}
	default:
		httpx.Failure(ctx, 400, "VALIDATION_ERROR")
		return
	}
	if e != nil {
		if !ctx.Writer.Written() {
			failure(ctx, e)
		}
		return
	}
	out["resource"] = ctx.Query("resource")
	out["retention"] = "soft_delete_only"
	ctx.Header("Content-Disposition", `attachment; filename="theia-export.json"`)
	ctx.JSON(http.StatusOK, out)
}

func emptyBody(ctx *gin.Context) bool {
	var b [1]byte
	n, e := ctx.Request.Body.Read(b[:])
	if n > 0 || e != io.EOF {
		httpx.Failure(ctx, 400, "UNEXPECTED_BODY")
		return false
	}
	return true
}
