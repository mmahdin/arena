package retrieval

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
)

func normalized(vector []float32) ([]float32, error) {
	if len(vector) == 0 || len(vector) > 8192 {
		return nil, errors.New("invalid embedding length")
	}
	var sum float64
	for _, v := range vector {
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) {
			return nil, errors.New("nonfinite embedding")
		}
		sum += float64(v) * float64(v)
	}
	if sum <= 0 {
		return nil, errors.New("zero embedding")
	}
	norm := math.Sqrt(sum)
	out := make([]float32, len(vector))
	for i, v := range vector {
		out[i] = float32(float64(v) / norm)
	}
	return out, nil
}
func cosine(a, b []float32) (float64, bool) {
	if len(a) == 0 || len(a) != len(b) {
		return 0, false
	}
	var dot, as, bs float64
	for i, v := range a {
		w := b[i]
		if math.IsNaN(float64(v)) || math.IsInf(float64(v), 0) || math.IsNaN(float64(w)) || math.IsInf(float64(w), 0) {
			return 0, false
		}
		dot += float64(v) * float64(w)
		as += float64(v) * float64(v)
		bs += float64(w) * float64(w)
	}
	if as == 0 || bs == 0 {
		return 0, false
	}
	sim := dot / math.Sqrt(as*bs)
	if sim < 0 {
		return 0, true
	}
	if sim > 1 {
		return 1, true
	}
	return sim, true
}

func approximateTokens(s string) int { return (len(s) + 3) / 4 }
func truncate(s string, tokens int) string {
	s = strings.TrimSpace(s)
	if s == "" || tokens <= 0 {
		return ""
	}
	max := tokens * 4
	if len(s) <= max {
		return s
	}
	if max <= len("…") {
		return ""
	}
	limit := max - len("…")
	n := 0
	for i := range s {
		if i > limit {
			break
		}
		n = i
	}
	if n == 0 {
		return ""
	}
	return strings.TrimSpace(s[:n]) + "…"
}
func age(now, at time.Time) string {
	if at.IsZero() {
		return "unknown"
	}
	d := now.Sub(at)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd", int(d.Hours()/24))
	}
}
