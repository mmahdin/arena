package llmhttp

import (
	"bytes"
	"io"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type observedReadCloser struct {
	io.Reader
	closes int
}

func (b *observedReadCloser) Close() error { b.closes++; return nil }

func TestHTTPObservationDisabledIsTransparent(t *testing.T) {
	in := &observedReadCloser{Reader: bytes.NewBufferString("complete synthetic HTTP response")}
	wrapped := observeHTTPBody(in, nil, false)
	if wrapped != in {
		t.Fatal("metadata-disabled wrapper must not change the response body")
	}
	data, err := io.ReadAll(wrapped)
	if err != nil || string(data) != "complete synthetic HTTP response" {
		t.Fatalf("data changed: %s / %v", data, err)
	}
	if err = wrapped.Close(); err != nil || in.closes != 1 {
		t.Fatalf("close changed: %v, %d", err, in.closes)
	}
}

func TestHTTPWrapperPreservesSSEContentCaptureOwnership(t *testing.T) {
	capture := newContentBody(&observedReadCloser{Reader: bytes.NewBufferString("data: [DONE]\n\n")}, nil, "", true)
	wrapped := &httpObservedBody{next: capture}
	if capturedContentBody(wrapped) != capture {
		t.Fatal("HTTP body wrapper suppressed SSE completeness capture")
	}
	if capturedContentBody(capture) != capture {
		t.Fatal("direct content body not detected")
	}
	if capturedContentBody(&observedReadCloser{Reader: bytes.NewBuffer(nil)}) != nil {
		t.Fatal("non-content body misidentified")
	}
}

func TestHTTPStreamObserverDisabledIsTransparent(t *testing.T) {
	source := &observedReadCloser{Reader: bytes.NewBufferString("data: [DONE]\n\n")}
	stream := ports.Stream(newSSEStream(source))
	if observeHTTPStream(stream, source) != stream {
		t.Fatal("disabled OTel unexpectedly changed streaming behavior")
	}
}
