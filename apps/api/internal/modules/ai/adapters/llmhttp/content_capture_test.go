package llmhttp

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

type countBody struct {
	io.Reader
	reads  int
	closed int
}

func (b *countBody) Read(out []byte) (int, error) { b.reads++; return b.Reader.Read(out) }
func (b *countBody) Close() error                 { b.closed++; return nil }

func TestContentReaderCapturesOnlyConsumedBytes(t *testing.T) {
	source := &countBody{Reader: strings.NewReader(`{"reply":"synthetic"} EXTRA UNREAD BYTES`)}
	wrapper := newContentBody(source, context.Background(), "private-secret", false)
	buf := make([]byte, len(`{"reply":`))
	n, err := wrapper.Read(buf)
	if err != nil || n != len(buf) || string(buf) != `{"reply":` {
		t.Fatalf("read changed: %d %q %v", n, buf, err)
	}
	if wrapper.count != int64(len(buf)) || source.reads != 1 {
		t.Fatal("read counters mismatch")
	}
	if err := wrapper.Close(); err != nil {
		t.Fatal(err)
	}
	if source.reads != 1 || source.closed != 1 || wrapper.eof {
		t.Fatal("capture drained or invented EOF")
	}
	if wrapper.count != int64(len(buf)) || string(wrapper.retained) != `{"reply":` {
		t.Fatal("capture differs from read bytes")
	}
}

func TestContentReaderBoundsBufferAndHashesAllNaturallyReadBytes(t *testing.T) {
	original := bytes.Repeat([]byte("a"), maxRetainedWireBytes+4096)
	source := &countBody{Reader: bytes.NewReader(original)}
	wrapper := newContentBody(source, context.Background(), "secret", false)
	copied, err := io.Copy(io.Discard, wrapper)
	if err != nil || copied != int64(len(original)) {
		t.Fatalf("changed reader behavior %d %v", copied, err)
	}
	if len(wrapper.retained) != maxRetainedWireBytes || wrapper.count != int64(len(original)) || !wrapper.eof {
		t.Fatal("capture bound, byte count or true EOF incorrect")
	}
	expected := sha256.Sum256(original)
	if hex.EncodeToString(wrapper.digest.Sum(nil)) != hex.EncodeToString(expected[:]) {
		t.Fatal("hash not over consumed bytes")
	}
	if err := wrapper.Close(); err != nil {
		t.Fatal(err)
	}
}

type fakeChunkStream struct {
	index  int
	closed bool
}

func (f *fakeChunkStream) Recv() (ports.Chunk, error) {
	f.index++
	switch f.index {
	case 1:
		return ports.Chunk{Text: "partial"}, nil
	case 2:
		return ports.Chunk{FinishReason: "stop"}, nil
	default:
		return ports.Chunk{}, io.EOF
	}
}
func (f *fakeChunkStream) Close() error { f.closed = true; return nil }

func TestContentStreamDoesNotChangeRecvsOrClose(t *testing.T) {
	source := &fakeChunkStream{}
	body := newContentBody(io.NopCloser(strings.NewReader("")), context.Background(), "secret", true)
	stream := &contentStream{next: source, body: body}
	c, err := stream.Recv()
	if err != nil || c.Text != "partial" {
		t.Fatal("first chunk changed")
	}
	c, err = stream.Recv()
	if err != nil || c.FinishReason != "stop" {
		t.Fatal("finish reason changed")
	}
	_, err = stream.Recv()
	if !errors.Is(err, io.EOF) {
		t.Fatal("SSE DONE no longer reaches caller")
	}
	if source.index != 3 {
		t.Fatal("synthetic stream was drained further")
	}
	if err = stream.Close(); err != nil || !source.closed {
		t.Fatal("stream close changed")
	}
}
