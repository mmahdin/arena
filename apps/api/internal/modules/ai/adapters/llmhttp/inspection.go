package llmhttp

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"io"
	"sync"
)

// No request/response header maps, credentials, URLs or cookies are captured.
type inspectionBody struct {
	once sync.Once
	next io.ReadCloser
	ctx  context.Context
	span *trace.Span
	mu   sync.Mutex
	raw  []byte
	err  error
	eof  bool
}

func (b *inspectionBody) Read(p []byte) (int, error) {
	n, err := b.next.Read(p)
	b.mu.Lock()
	b.raw = append(b.raw, p[:n]...)
	if err == io.EOF {
		b.eof = true
	} else if err != nil {
		b.err = err
	}
	b.mu.Unlock()
	return n, err
}
func (b *inspectionBody) Close() error {
	err := b.next.Close()
	b.mu.Lock()
	defer b.mu.Unlock()
	if err != nil {
		b.err = err
	}
	b.once.Do(func() {
		trace.Capture(b.ctx, "raw_response_body", "wire_response", "http_response_body", string(b.raw))
		b.span.End(func() any { return map[string]any{"body_complete": b.eof, "bytes_read": len(b.raw)} }, b.err)
	})
	return err
}
