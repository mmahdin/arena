package mssql

import (
	"context"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/application/retrieval"
	"strings"
	"testing"
)

func TestHybridSQLValidation(t *testing.T) {
	if _, e := NewRetrievalRepository(nil, nil, nil); e == nil {
		t.Fatal("nil DB accepted")
	}
	if _, e := NewRetrievalRepositoryTx(nil, nil, nil); e == nil {
		t.Fatal("nil transaction accepted")
	}
	r := &RetrievalRepository{}
	if e := r.Scan(context.Background(), retrieval.Query{}, nil); e == nil {
		t.Fatal("invalid query accepted")
	}
}
func TestCorpusHasNoPreselection(t *testing.T) {
	if strings.Contains(corpusSQL, "TOP") || strings.Contains(corpusSQL, "importance") {
		t.Fatal("preselection limits vector recall")
	}
	for _, guard := range []string{"s.user_id=@p1", "trigger_conv.user_id=@p1", "trigger_msg.deleted_at IS NULL", "e.source_version=s.source_version"} {
		if !strings.Contains(corpusSQL, guard) {
			t.Fatal("missing isolation/revision guard", guard)
		}
	}
}
