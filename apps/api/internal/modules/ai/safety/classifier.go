package safety

import (
	"context"
	"errors"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

// Authenticated turn metadata is deliberately not copied from user text.
// WP-05's SafetyInput currently lacks IDs. The authenticated composition
// caller must set this on the ProcessTurn context (no shared mutable state).
type callMetaKey struct{}

func WithCallMeta(ctx context.Context, meta contract.CallMeta) context.Context {
	return context.WithValue(ctx, callMetaKey{}, meta)
}

type LLMClassifier struct {
	client  ports.Client
	schemas *schema.Set
	assets  *Assets
	timeout time.Duration
}

var _ Classifier = (*LLMClassifier)(nil)

// NewLLMClassifier reuses WP-04's structured tier, retry and budgets. L1's
// strict schema and prompt come from WP-01 and the pinned WP-03 bundle.
// A synthetic placeholder cannot be sent to an LLM as an instruction.
func NewLLMClassifier(client ports.Client, schemas *schema.Set, assets *Assets, timeout time.Duration) (*LLMClassifier, error) {
	if client == nil || schemas == nil || assets == nil || strings.TrimSpace(assets.prompt) == "" ||
		strings.Contains(assets.prompt, "SYNTHETIC PLACEHOLDER") {
		return nil, ErrAssets
	}
	if timeout <= 0 {
		return nil, ErrAssets
	}
	return &LLMClassifier{client: client, schemas: schemas, assets: assets, timeout: timeout}, nil
}
func (c *LLMClassifier) Classify(ctx context.Context, in ports.SafetyInput) (result0 Assessment, result1 error) {
	ctx, inspectionSpan := trace.Start(ctx, "safety/classifier.Classify", func() any { return map[string]any{"in": in} })
	defer func() {
		inspectionSpan.Complete(func() any { return map[string]any{"result0": result0} }, result1, recover())
	}()

	if err := ctx.Err(); err != nil {
		return Assessment{}, err
	}
	if c == nil {
		return Assessment{}, ErrClassifier
	}
	meta, ok := ctx.Value(callMetaKey{}).(contract.CallMeta)
	if !ok || !contract.ValidID(meta.UserID) || !contract.ValidID(meta.TurnID) {
		return Assessment{}, ErrClassifier
	}
	if meta.Bundle.Version != "" && meta.Bundle.Version != c.assets.ref.Version {
		return Assessment{}, ErrClassifier
	}
	meta.Bundle = c.assets.Reference()
	definition, err := c.schemas.JSON(contract.SafetySchemaVersion)
	if err != nil {
		return Assessment{}, ErrClassifier
	}
	messages := make([]ports.ChatMessage, 0, 5)
	start := len(in.Recent) - 4
	if start < 0 {
		start = 0
	}
	for _, msg := range in.Recent[start:] {
		if msg.Role != contract.RoleAssistant && msg.Role != contract.RoleUser {
			continue
		}
		messages = append(messages, ports.ChatMessage{Role: msg.Role, Content: msg.Text})
	}
	messages = append(messages, ports.ChatMessage{Role: contract.RoleUser, Content: in.Text})
	trace.Capture(ctx, "classifier_template", "prompt_template", "prompts/safety_l1.system.tmpl", c.assets.prompt)
	resp, err := c.client.Structured(ctx, ports.StructuredRequest{
		Tier: ports.ComponentSafety, System: []ports.Block{{Text: c.assets.prompt, Name: "safety_l1.system"}},
		Messages: messages, Schema: definition, SchemaName: contract.SafetySchemaVersion,
		MaxTokens: 256, Temperature: 0.1, Timeout: c.timeout, Meta: meta,
		IdempotencyKey: meta.TurnID.String() + "/safety_l1",
	})
	if err != nil {
		return Assessment{}, errors.Join(ErrClassifier, err)
	}
	parsed, err := c.schemas.DecodeSafety(resp.JSON)
	if err != nil {
		return Assessment{}, ErrClassifier
	}
	a := Assessment{Output: parsed, Version: contract.SafetySchemaVersion + "/" + resp.Model}
	if !validL1(a) {
		return Assessment{}, ErrClassifier
	}
	return a, nil
}
