package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestTraceReferenceValidationRejectsMalformedAndMixedMetadata(t *testing.T) {
	root := TraceReference{ID: uuid.New(), Kind: "turn", TraceID: "0123456789abcdef0123456789abcdef", RootSpanID: "0123456789abcdef", RecordedAt: time.Now()}
	if err := root.Validate(); err != nil {
		t.Fatal(err)
	}
	for _, value := range []string{"", "xyz", "0123456789ABCDEF0123456789ABCDEF", "00000000000000000000000000000000", "0123456789abcdef0123456789abcdeg"} {
		invalid := root
		invalid.TraceID = value
		if invalid.Validate() == nil {
			t.Fatalf("bad trace id accepted: %q", value)
		}
	}
	invalid := root
	invalid.Kind = "reflection"
	if invalid.Validate() == nil {
		t.Fatal("job without job id")
	}
	id, attempt := uuid.New(), 2
	validJob := root
	validJob.Kind = "reflection"
	validJob.JobID = &id
	validJob.JobAttempt = &attempt
	if validJob.Validate() != nil {
		t.Fatal("valid job rejected")
	}
	invalid = validJob
	invalid.Kind = "turn"
	if invalid.Validate() == nil {
		t.Fatal("turn with job data")
	}
	f := TraceReferenceFilter{TurnID: uuid.New(), Limit: 25}
	if f.Validate() != nil {
		t.Fatal("valid filter")
	}
	for _, v := range []TraceReferenceFilter{{TurnID: uuid.Nil, Limit: 25}, {TurnID: uuid.New(), Limit: 0}, {TurnID: uuid.New(), Limit: 51}} {
		if v.Validate() == nil {
			t.Fatal("invalid paging accepted")
		}
	}
}
