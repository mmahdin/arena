package domain

import (
	"strings"
	"testing"
)

func TestBO10InspectorSafeReadLabels(t *testing.T) {
	for _, input := range []string{"", "PRIVATE DO NOT LEAK", "\nsecret", "a/b", "µsecret", strings.Repeat("x", 81)} {
		if actual := SafeInspectorLabel(input); actual != "unclassified" {
			t.Fatalf("unsafe label %q reflected as %q", input, actual)
		}
	}
	for _, input := range []string{"model-v1", "provider_alpha", "brain", "small", "gpt-5.6-sol"} {
		if got := SafeInspectorLabel(input); got != input {
			t.Fatalf("valid label changed: %q => %q", input, got)
		}
	}
	if SafeTurnStatus("PRIVATE DO NOT LEAK") != "unknown" || SafeTurnStatus("failed") != "failed" ||
		SafeStageStatus("PRIVATE DO NOT LEAK") != "unknown" || SafeStageStatus("degraded") != "degraded" ||
		SafeAttemptKind("PRIVATE DO NOT LEAK") != "unknown" || SafeAttemptKind("stream") != "stream" ||
		SafeAttemptStatus("PRIVATE DO NOT LEAK") != "unknown" || SafeAttemptStatus("incomplete") != "incomplete" ||
		SafeCostStatus("PRIVATE DO NOT LEAK") != "unknown" || SafeCostStatus("unpriced") != "unpriced" ||
		SafeWorkerKind("PRIVATE DO NOT LEAK") != "unclassified" || SafeWorkerKind("memory") != "memory" {
		t.Fatal("status/kind allowlist accepted untrusted values or rejected known values")
	}
}
