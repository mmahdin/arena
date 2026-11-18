package retrieval

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/bundle"
	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
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

var stop = map[string]bool{}

func init() {
	for _, word := range strings.Fields(`a about again ago all and are as at be been before by can did do does for from had has have how i in into is it last me month my of on or our said say she should since so that the their them there they this to us was we were what when where which who why with would yesterday you your
        am an another also ask asked earlier here him her his herself himself myself talked tell told just know remember today day days week weeks past previous time i m re s first second someone some one same
        از است با به برای بود بوده در را رو که کرد کردم کردی کردیم گفتم گفتی گفته گفتمون دوباره یادته یادم یادمون درباره راجع همون اون این چی چیه کی کجا چه چرا من تو ما شما دیروز پریروز امروز هفته پیش قبل دو سه چهار یه یک می کنه کردم
        `) {
		stop[word] = true
	}
}
func tokens(s string) []string {
	var words []string
	var word []rune
	flush := func() {
		if len(word) > 0 {
			w := strings.ToLower(string(word))
			if !stop[w] && utf8.RuneCountInString(w) > 1 {
				words = append(words, w)
			}
			word = nil
		}
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			word = append(word, r)
		} else {
			flush()
		}
	}
	flush()
	return words
}
func firstTerms(text string, n int) []string {
	seen := map[string]bool{}
	var out []string
	for _, term := range tokens(text) {
		if !seen[term] {
			seen[term] = true
			out = append(out, term)
			if len(out) >= n {
				break
			}
		}
	}
	return out
}
func similarity(a, b string) float64 {
	ta, tb := tokens(a), tokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	aset, bset := map[string]bool{}, map[string]bool{}
	for _, t := range ta {
		aset[t] = true
	}
	for _, t := range tb {
		bset[t] = true
	}
	common := 0
	for t := range aset {
		if bset[t] {
			common++
		}
	}
	union := len(aset) + len(bset) - common
	if union == 0 {
		return 0
	}
	return float64(common) / float64(union)
}
func topicalMatch(a, b string) float64 {
	ta, tb := tokens(a), tokens(b)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	aset, bset := map[string]bool{}, map[string]bool{}
	for _, t := range ta {
		aset[t] = true
	}
	for _, t := range tb {
		bset[t] = true
	}
	n := 0
	for k := range aset {
		if bset[k] {
			n++
		}
	}
	if n == 0 {
		return 0
	}
	denom := len(aset)
	if len(bset) < denom {
		denom = len(bset)
	}
	return math.Max(similarity(a, b), float64(n)/float64(denom)*0.9)
}
func textMatches(text string, terms []string) bool {
	if len(terms) == 0 {
		return true
	}
	hay := map[string]bool{}
	for _, t := range tokens(text) {
		hay[t] = true
	}
	for _, term := range terms {
		if !hay[term] {
			return false
		}
	}
	return true
}
func subjectTerms(text, temporalPhrase string) []string {
	low := strings.ToLower(text)
	for _, marker := range []string{" about ", " regarding ", " concerning ", " درباره ", " راجع به "} {
		if i := strings.LastIndex(low, marker); i >= 0 {
			return firstTerms(low[i+len(marker):], 5)
		}
	}
	// Without an explicit 'about' qualifier there may be no topical filter;
	// exact recall is constrained by date, role, and authorization only.
	return nil
}
func containsAnaphora(text string) bool {
	low := strings.ToLower(text)
	for _, p := range []string{"that again", "that thing", "it again", "like before", "what happened before", "same problem", "همون", "اون موضوع", "دوباره", "قبلا", "مثل قبل", "یادته"} {
		if strings.Contains(low, p) {
			return true
		}
	}
	return false
}
func targetEntity(req c.RetrievalRequest) (role, label string) {
	words := map[string]bool{}
	for _, v := range tokens(queryText(req)) {
		words[v] = true
	}
	roles := map[string][]string{
		"mother":   {"mother", "mom", "mum", "مادر", "مامان"},
		"father":   {"father", "dad", "پدر", "بابا"},
		"boss":     {"boss", "manager", "رئیس", "رییس", "مدیر"},
		"partner":  {"partner", "spouse", "husband", "wife", "همسر", "پارتنر"},
		"sister":   {"sister", "خواهر"},
		"brother":  {"brother", "برادر"},
		"friend":   {"friend", "دوست"},
		"coworker": {"coworker", "colleague", "همکار"},
		"child":    {"child", "son", "daughter", "فرزند", "بچه"},
	}
	var found []string
	for key, aliases := range roles {
		for _, alias := range aliases {
			if words[alias] {
				found = append(found, key)
				break
			}
		}
	}
	if len(found) == 1 {
		return found[0], ""
	}
	if req.Extra != nil && req.Extra.Kind == c.ContextEntity && len(found) == 0 {
		// Explicit label only. Do not guess a person's identity/aliases.
		return "", strings.TrimSpace(req.Extra.Query)
	}
	return "", ""
}
func scoreCandidate(v Candidate, query string, vector []float32, focus *c.ID, now time.Time, cfg bundle.Retrieval, exact bool) float64 {
	if exact {
		return 1
	}
	sim := topicalMatch(query, v.Text)
	if emb, valid := cosine(vector, v.Vector); valid {
		sim = math.Max(sim, emb)
	}
	delta := now.Sub(v.CreatedAt).Hours() / 24
	if delta < 0 {
		delta = 0
	}
	recency := math.Pow(0.5, delta/float64(cfg.Recency.HalfLifeDays))
	importance := math.Max(0, math.Min(1, v.Importance))
	match := 0.0
	if threadContains(v.ThreadIDs, focus) {
		match = 1
	}
	score := cfg.Weights.Similarity*sim + cfg.Weights.Recency*recency + cfg.Weights.Importance*importance + cfg.Weights.ThreadMatch*match
	if score > 1 {
		return 1
	}
	return score
}
func isNovel(text string, recent []string, threshold float64) bool {
	if threshold <= 0 {
		return false
	}
	for _, other := range recent {
		if similarity(text, other) >= threshold {
			return true
		}
	}
	return false
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

func entityContains(entities []c.EntityRef, role, label string) bool {
	if role == "" && label == "" {
		return false
	}
	for _, e := range entities {
		if role != "" && e.Role == role {
			return true
		}
		if label != "" && e.Label != nil && strings.EqualFold(*e.Label, label) {
			return true
		}
	}
	return false
}
func maxFloat(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// Exact reuse of explicitly written emotion words is a retrieval signal,
// never an inference of a diagnosis, risk state, or unreported user feeling.
func explicitEmotionOverlap(a, b string) bool {
	emotion := map[string]bool{"sad": true, "angry": true, "worried": true, "afraid": true, "lonely": true,
		"anxious": true, "happy": true, "grief": true, "stress": true, "sadness": true,
		"غمگین": true, "ناراحت": true, "نگران": true, "عصبانی": true, "ترس": true, "تنها": true, "استرس": true}
	aWords := map[string]bool{}
	for _, w := range tokens(a) {
		if emotion[w] {
			aWords[w] = true
		}
	}
	for _, w := range tokens(b) {
		if aWords[w] {
			return true
		}
	}
	return false
}
