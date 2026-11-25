package llmhttp

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"hash"
	"io"
	"sync"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// contentBody observes only bytes which the existing response consumer reads.
// It never drains the HTTP response, and retains a bounded development capture.
// The same bytes still flow untouched to JSON decoding and the SSE scanner.
type contentBody struct {
	next     io.ReadCloser
	ctx      context.Context
	secret   string
	stream   bool
	mu       sync.Mutex
	once     sync.Once
	retained []byte
	digest   hash.Hash
	count    int64
	eof      bool
	closeErr bool
}

const maxRetainedWireBytes = telemetry.MaxWireCaptureBytes

func newContentBody(next io.ReadCloser, ctx context.Context, secret string, stream bool) *contentBody {
	return &contentBody{next: next, ctx: ctx, secret: secret, stream: stream, digest: sha256.New()}
}

func (b *contentBody) Read(p []byte) (int, error) {
	n, err := b.next.Read(p)
	b.mu.Lock()
	if n > 0 {
		b.count += int64(n)
		_, _ = b.digest.Write(p[:n])
		if len(b.retained) < maxRetainedWireBytes {
			keep := n
			if keep > maxRetainedWireBytes-len(b.retained) {
				keep = maxRetainedWireBytes - len(b.retained)
			}
			b.retained = append(b.retained, p[:keep]...)
		}
	}
	if err == io.EOF {
		b.eof = true
	}
	b.mu.Unlock()
	return n, err
}

func (b *contentBody) emit(streamComplete bool) {
	b.once.Do(func() {
		b.mu.Lock()
		raw := append([]byte(nil), b.retained...)
		state := telemetry.ContentDisposition{
			Bytes:    b.count,
			SHA256:   hex.EncodeToString(b.digest.Sum(nil)),
			Complete: !b.closeErr && ((b.stream && streamComplete) || (!b.stream && b.eof)),
			Stream:   b.stream,
		}
		b.mu.Unlock()
		telemetry.CaptureResponse(b.ctx, raw, state, b.secret)
	})
}

func (b *contentBody) Close() error {
	err := b.next.Close()
	if err != nil {
		b.mu.Lock()
		b.closeErr = true
		b.mu.Unlock()
	}
	b.emit(false)
	return err
}

// The metered stream finishes its OTel span when Recv reaches [DONE] or fails.
// Flush the wire capture at exactly that moment, not at a potentially later
// caller Close, so it is visible before the physical-attempt span ends.
type contentStream struct {
	next ports.Stream
	body *contentBody
}

func (s *contentStream) Recv() (ports.Chunk, error) {
	chunk, err := s.next.Recv()
	if err != nil {
		s.body.emit(err == io.EOF)
	}
	return chunk, err
}

func (s *contentStream) Close() error {
	return s.next.Close() // body.Close emits incomplete if Recv did not finish
}

var _ ports.Stream = (*contentStream)(nil)
