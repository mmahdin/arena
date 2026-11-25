package llmhttp

import (
	"io"
	"sync"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

// The HTTP span stays open while the caller consumes the actual response
// body, including streamed SSE. A response is never drained for observability.
// Abort/early Close is explicitly marked incomplete. The wrapper is transparent
// to provider parsing and changes neither error classification nor bytes.
type httpObservedBody struct {
	next    io.ReadCloser
	span    *telemetry.Span
	once    sync.Once
	mu      sync.Mutex
	stream  bool
	eof     bool
	readErr error
}

func observeHTTPBody(body io.ReadCloser, span *telemetry.Span, stream bool) io.ReadCloser {
	if span == nil {
		return body
	}
	return &httpObservedBody{next: body, span: span, stream: stream}
}

func (b *httpObservedBody) Read(p []byte) (int, error) {
	n, err := b.next.Read(p)
	if err != nil {
		b.mu.Lock()
		if err == io.EOF {
			b.eof = true
		} else {
			b.readErr = err
		}
		b.mu.Unlock()
	}
	return n, err
}

func (b *httpObservedBody) Close() error {
	// The inner body emits its wire event on Close. End the span afterward
	// so that a non-streaming response can actually be exported.
	err := b.next.Close()
	b.mu.Lock()
	complete := !b.stream && b.eof && b.readErr == nil && err == nil
	readErr := b.readErr
	b.mu.Unlock()
	b.once.Do(func() {
		if !complete {
			telemetry.HTTPIncomplete(b.span)
		}
		if readErr != nil {
			b.span.End(readErr)
		} else {
			b.span.End(err)
		}
	})
	return err
}

// SSE completion is signaled by [DONE], which can precede transport EOF.
// Preserve OBS-04A's explicit complete=true wire disposition even though the
// HTTP response is additionally wrapped for timing/transport diagnostics.
func capturedContentBody(body io.ReadCloser) *contentBody {
	if capture, ok := body.(*contentBody); ok {
		return capture
	}
	if observed, ok := body.(*httpObservedBody); ok {
		capture, _ := observed.next.(*contentBody)
		return capture
	}
	return nil
}

// SSE [DONE] is a protocol-level completion even if the transport did not
// produce a final io.EOF read. Only the validated stream parser may signal it.
func (b *httpObservedBody) completeStream() {
	b.once.Do(func() { b.span.End(nil) })
}

func (b *httpObservedBody) failStream(err error) {
	b.once.Do(func() {
		telemetry.HTTPIncomplete(b.span)
		b.span.End(err)
	})
}

// A streaming body can be fully consumed via [DONE] without reading the
// transport EOF. Observe the parser's real result; do not parse SSE twice.
type httpObservedStream struct {
	next ports.Stream
	body *httpObservedBody
}

func (s *httpObservedStream) Recv() (ports.Chunk, error) {
	chunk, err := s.next.Recv()
	if err == io.EOF {
		s.body.completeStream()
	} else if err != nil {
		s.body.failStream(err)
	}
	return chunk, err
}
func (s *httpObservedStream) Close() error { return s.next.Close() }

func observeHTTPStream(stream ports.Stream, body io.ReadCloser) ports.Stream {
	if observed, ok := body.(*httpObservedBody); ok {
		return &httpObservedStream{next: stream, body: observed}
	}
	return stream
}
