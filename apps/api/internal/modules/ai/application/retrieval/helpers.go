package retrieval

import (
	"github.com/google/uuid"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func strategyNames(in []contract.RetrievalStrategy) []string {
	if len(in) == 0 {
		return nil
	}
	out := make([]string, 0, len(in))
	for _, s := range in {
		out = append(out, string(s))
	}
	return out
}

func containsStrategy(in []contract.RetrievalStrategy, want contract.RetrievalStrategy) bool {
	for _, s := range in {
		if s == want {
			return true
		}
	}
	return false
}

func needsKind(n *contract.NeedsContext) string {
	if n == nil {
		return ""
	}
	return n.Kind
}

func idString(id *contract.ID) string {
	if id == nil {
		return ""
	}
	return id.String()
}

func idStrings(ids []contract.ID) []string {
	out := make([]string, 0, len(ids))
	for _, id := range ids {
		out = append(out, id.String())
	}
	return out
}

func parseID(s string) contract.ID {
	if s == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(s)
	if err != nil {
		return uuid.Nil
	}
	return id
}

// ranStrategy reports whether a strategy appears in the executed list.
func ranStrategy(executed []string, name string) bool {
	for _, s := range executed {
		if s == name {
			return true
		}
	}
	return false
}
