package retrieval

import (
	"strings"
	"time"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	domain "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/domain/retrieval"
)

// candidateBuilder accumulates candidates from several strategies into one
// deduplicated-by-id set.
//
// Merge rules:
//   - first producer records the Strategy (used in traces);
//   - the highest Similarity wins, so exact/temporal evidence is not diluted by
//     a weaker semantic hit for the same item;
//   - exclusion is sticky: once a rule excludes a candidate it stays excluded.
type candidateBuilder struct {
	userID contract.ID
	tun    domain.Tunables
	now    time.Time
	byID   map[string]int
	all    []domain.Candidate

	strategiesRun     []string
	temporalHits      int
	anaphoraMatched   int
	missingEmbeddings int
}

func newCandidateBuilder(userID contract.ID, tun domain.Tunables, now time.Time) *candidateBuilder {
	return &candidateBuilder{
		userID: userID, tun: tun, now: now,
		byID: map[string]int{},
	}
}

func (b *candidateBuilder) noteStrategy(name string) {
	for _, s := range b.strategiesRun {
		if s == name {
			return
		}
	}
	b.strategiesRun = append(b.strategiesRun, name)
}

// addMemory projects a stored record into a candidate and applies the structural
// eligibility rules: owner isolation, soft-delete, status, provenance,
// sensitivity and hypothesis surfacing (§3.4, §7.5, §7.6).
func (b *candidateBuilder) addMemory(rec contract.MemoryRecord, strategy string, similarity float32) {
	if rec.ID == (contract.ID{}) {
		return
	}
	// Owner isolation: a row belonging to another user is never a candidate,
	// even if a store bug returned it.
	if rec.UserID != (contract.ID{}) && rec.UserID != b.userID {
		return
	}
	c := domain.Candidate{
		ID:          rec.ID.String(),
		Kind:        string(rec.Kind),
		Epistemic:   string(rec.Epistemic),
		Status:      domain.Status(rec.Status),
		Text:        strings.TrimSpace(rec.Content),
		Age:         AgeLabel(b.now, rec.UpdatedAt),
		Sensitivity: string(rec.Sensitivity),
		Importance:  rec.Importance,
		ClusterID:   idString(rec.ClusterID),
		EvidenceN:   rec.EvidenceN,
		UpdatedAt:   rec.UpdatedAt,
		ValidFrom:   rec.ValidFrom,
		Tags:        rec.Tags,
		ThreadIDs:   idStrings(rec.ThreadIDs),
		Similarity:  similarity,
		Strategy:    strategy,
	}
	c.Excluded, c.ExcludeReason = b.eligibility(c, rec)
	if len(c.Text) == 0 {
		c.Excluded, c.ExcludeReason = true, "empty_content"
	}
	b.merge(c)
}

func (b *candidateBuilder) eligibility(c domain.Candidate, rec contract.MemoryRecord) (bool, string) {
	if rec.DeletedAt != nil {
		// Soft-delete policy: deleted rows stay in storage but never surface.
		return true, "soft_deleted"
	}
	switch rec.Status {
	case contract.MemRejected:
		return true, "rejected"
	case contract.MemSuperseded:
		// Superseded items are retained for provenance only (§7.3).
		return true, "superseded"
	}
	switch rec.Kind {
	case contract.MemSafetyEvt:
		// Safety events are for the safety pipeline, not conversational recall.
		return true, "safety_event"
	case contract.MemHypothesis:
		if rec.Epistemic != contract.EpInferred {
			// A hypothesis must never be stored as reported (§7.2 CK_ai_mem_hyp).
			return true, "hypothesis_epistemic_invalid"
		}
		allowed := false
		for _, st := range b.tun.HypothesisStatuses {
			if string(st) == string(rec.Status) {
				allowed = true
			}
		}
		if !allowed {
			return true, "hypothesis_status"
		}
	}
	if rec.EvidenceN < 1 {
		// Provenance invariant (§2.5 #3): no evidence, no retrieval.
		return true, "no_evidence"
	}
	if b.tun.ExcludeHighlySensitive && rec.Sensitivity == contract.SensHighly {
		return true, "highly_sensitive"
	}
	return false, ""
}

func (b *candidateBuilder) addMessage(rec contract.MessageRecord, strategy string) {
	if rec.ID == (contract.ID{}) {
		return
	}
	if rec.UserID != (contract.ID{}) && rec.UserID != b.userID {
		return
	}
	c := domain.Candidate{
		ID:           rec.ID.String(),
		Kind:         "episode", // raw messages are the episodic source of truth
		Epistemic:    string(contract.EpReported),
		Status:       domain.StatusActive,
		Text:         strings.TrimSpace(rec.Content),
		Age:          AgeLabel(b.now, rec.CreatedAt),
		Sensitivity:  string(contract.SensNormal),
		Importance:   0.8, // verbatim user words outrank derived items
		UpdatedAt:    rec.CreatedAt,
		MessageAgeAt: &rec.CreatedAt,
		Verbatim:     true,
		EvidenceN:    1,
		Similarity:   1, // the temporal window already matched this message
		Strategy:     strategy,
	}
	if rec.DeletedAt != nil || len(c.Text) == 0 {
		c.Excluded = true
		if rec.DeletedAt != nil {
			c.ExcludeReason = "soft_deleted"
		} else {
			c.ExcludeReason = "empty_content"
		}
	}
	b.merge(c)
}

// merge folds a candidate into the set. A candidate is identified by (id, kind),
// because a message and a memory item may legitimately share an id space only by
// kind. Exclusions are permanent: they come from hard invariants (soft delete,
// provenance, sensitivity), so a second strategy may never resurrect a row.
func (b *candidateBuilder) merge(c domain.Candidate) {
	key := c.ID + "|" + c.Kind
	if idx, ok := b.byID[key]; ok {
		existing := &b.all[idx]
		if existing.Excluded {
			return
		}
		if c.Similarity > existing.Similarity {
			existing.Similarity = c.Similarity
		}
		return
	}
	b.byID[key] = len(b.all)
	b.all = append(b.all, c)
}

func (b *candidateBuilder) candidates() []domain.Candidate { return b.all }

// dropHypotheses marks every hypothesis candidate as excluded. Used when a
// reference is unresolved: an inference is the last thing to offer then (§3.4).
func (b *candidateBuilder) dropHypotheses() {
	for i := range b.all {
		if b.all[i].Kind == string(contract.MemHypothesis) && !b.all[i].Excluded {
			b.all[i].Excluded = true
			b.all[i].ExcludeReason = "ambiguous_reference"
		}
	}
}

// toView projects a ranked candidate into the wire MemoryView.
func toView(r domain.Ranked, now time.Time) contract.MemoryView {
	view := contract.MemoryView{
		ID:          parseID(r.ID),
		Kind:        contract.MemoryKind(r.Kind),
		Epistemic:   contract.Epistemic(r.Epistemic),
		Status:      contract.MemoryStatus(r.Status),
		Text:        r.Text,
		Age:         r.Age,
		EvidenceN:   r.EvidenceN,
		Sensitivity: contract.Sensitivity(r.Sensitivity),
		Verbatim:    r.Verbatim,
		Score:       r.Breakdown.Total,
	}
	if r.ValidFrom != nil && !r.ValidFrom.IsZero() {
		v := r.ValidFrom.UTC().Format(time.RFC3339)
		view.ValidAt = &v
	}
	if view.Age == "" {
		view.Age = AgeLabel(now, r.UpdatedAt)
	}
	return view
}

// AgeLabel renders a short relative age ("2m", "3h", "2d") per §4.3 MsgView.
func AgeLabel(now, then time.Time) string {
	if then.IsZero() {
		return ""
	}
	d := now.Sub(then)
	if d < 0 {
		d = 0
	}
	switch {
	case d < time.Minute:
		return "0m"
	case d < time.Hour:
		return itoa(int(d.Minutes())) + "m"
	case d < 24*time.Hour:
		return itoa(int(d.Hours())) + "h"
	case d < 90*24*time.Hour:
		return itoa(int(d.Hours()/24)) + "d"
	default:
		return itoa(int(d.Hours()/24/30)) + "mo"
	}
}

func itoa(v int) string {
	if v == 0 {
		return "0"
	}
	var buf [12]byte
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + v%10)
		v /= 10
	}
	return string(buf[i:])
}
