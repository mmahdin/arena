package composition

import (
	"context"
	"testing"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type capturedClient struct {
	structured ports.StructuredRequest
	stream     ports.StreamRequest
	embed      ports.EmbeddingRequest
}

func (f *capturedClient) Structured(_ context.Context, req ports.StructuredRequest) (ports.StructuredResponse, error) {
	f.structured = req
	return ports.StructuredResponse{}, nil
}
func (f *capturedClient) Stream(_ context.Context, req ports.StreamRequest) (ports.Stream, error) {
	f.stream = req
	return nil, nil
}
func (f *capturedClient) Embed(_ context.Context, req ports.EmbeddingRequest) (ports.EmbeddingResponse, error) {
	f.embed = req
	return ports.EmbeddingResponse{}, nil
}

func TestComponentClientUsesRuntimeSettingsAndRespectsTighterTurnLimits(t *testing.T) {
	fake := &capturedClient{}
	adapter := configuredComponentClient{next: fake, components: map[ports.ModelTier]runtimeComponent{
		ports.ComponentSafety:    {TimeoutMS: 5000, MaxTokens: 256, Temperature: 0.1},
		ports.ComponentValidator: {TimeoutMS: 4000, MaxTokens: 128, Temperature: 0},
		ports.ComponentGenerator: {TimeoutMS: 30000, MaxTokens: 600, Temperature: 0.7},
		ports.ComponentEmbedding: {TimeoutMS: 5000},
	}}
	_, err := adapter.Structured(context.Background(), ports.StructuredRequest{Tier: ports.ComponentSafety, MaxTokens: 300, Timeout: time.Millisecond})
	if err != nil || fake.structured.Timeout != time.Millisecond || fake.structured.MaxTokens != 256 || fake.structured.Temperature != 0.1 {
		t.Fatal(fake.structured, err)
	}
	_, err = adapter.Structured(context.Background(), ports.StructuredRequest{Tier: ports.ComponentValidator, MaxTokens: 90, Temperature: 1})
	if err != nil || fake.structured.MaxTokens != 90 || fake.structured.Temperature != 0 {
		t.Fatal(fake.structured, err)
	}
	_, err = adapter.Stream(context.Background(), ports.StreamRequest{Tier: ports.ComponentGenerator, MaxTokens: 150})
	if err != nil || fake.stream.Timeout != 30*time.Second || fake.stream.MaxTokens != 150 || fake.stream.Temperature != 0.7 {
		t.Fatal(fake.stream, err)
	}
	_, err = adapter.Embed(context.Background(), ports.EmbeddingRequest{Tier: ports.ComponentEmbedding})
	if err != nil || fake.embed.Timeout != 5*time.Second {
		t.Fatal(fake.embed, err)
	}
	if _, err = adapter.Structured(context.Background(), ports.StructuredRequest{Tier: ports.TierClassifier}); err == nil {
		t.Fatal("legacy tier bypassed runtime")
	}
}
