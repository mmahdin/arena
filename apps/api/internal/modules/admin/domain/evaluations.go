package domain

import (
	"bytes"
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"math"
	"regexp"
	"sort"
	"time"

	"github.com/google/uuid"
)

const MaxEvalReportBytes = 1 << 20
const MaxEvalWindow = 31 * 24 * time.Hour
const DefaultEvalLimit = 25
const MaxEvalLimit = 100

var ErrInvalidEval = errors.New("invalid synthetic evaluation report or filter")
var ErrEvalNotFound = errors.New("evaluation run not found")
var ErrEvalConflict = errors.New("evaluation already imported with different provenance")

var evalIdentifier = regexp.MustCompile(`^[a-z][a-z0-9_-]{2,79}$`)
var evalCode = regexp.MustCompile(`^[A-Z][A-Z0-9_]{1,79}$`)
var evalAssertion = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_./\[\]-]{0,127}$`)

// WP-15's redacted report vocabulary is intentionally mirrored without
// importing the runtime evaluator into the Admin domain. Never add free text
// or raw traces to these DTOs. Unknown JSON fields are rejected at import.
type EvalDiagnostic struct {
	Code      string `json:"code"`
	Assertion string `json:"assertion"`
	Severity  string `json:"severity"`
}
type EvalTurn struct {
	Index            int              `json:"index"`
	Session          int              `json:"session"`
	Passed           bool             `json:"passed"`
	Assertions       int              `json:"assertions"`
	PassedAssertions int              `json:"passed_assertions"`
	Diagnostics      []EvalDiagnostic `json:"diagnostics,omitempty"`
}
type EvalScenario struct {
	ID           string     `json:"id"`
	Tags         []string   `json:"tags,omitempty"`
	Passed       bool       `json:"passed"`
	ReviewStatus string     `json:"review_status"`
	Turns        []EvalTurn `json:"turns"`
}
type EvalTotals struct {
	Scenarios        int `json:"scenarios"`
	PassedScenarios  int `json:"passed_scenarios"`
	FailedScenarios  int `json:"failed_scenarios"`
	Turns            int `json:"turns"`
	PassedTurns      int `json:"passed_turns"`
	Assertions       int `json:"assertions"`
	PassedAssertions int `json:"passed_assertions"`
	Failures         int `json:"failures"`
	Blockers         int `json:"blockers"`
}
type EvalMetric struct {
	Passed int     `json:"passed"`
	Total  int     `json:"total"`
	Rate   float64 `json:"rate"`
}
type EvalRegression struct {
	Scenario  string `json:"scenario,omitempty"`
	Assertion string `json:"assertion,omitempty"`
	Metric    string `json:"metric,omitempty"`
	Code      string `json:"code"`
}
type EvalReviewFlag struct {
	Scenario string `json:"scenario"`
	Turn     int    `json:"turn"`
	Reason   string `json:"reason"`
	Status   string `json:"status"`
}
type EvalReport struct {
	Schema        string                `json:"schema"`
	Method        string                `json:"method"`
	SyntheticOnly bool                  `json:"synthetic_only"`
	Passed        bool                  `json:"passed"`
	Totals        EvalTotals            `json:"totals"`
	Metrics       map[string]EvalMetric `json:"metrics"`
	Scenarios     []EvalScenario        `json:"scenarios"`
	Regressions   []EvalRegression      `json:"regressions,omitempty"`
	ReviewQueue   []EvalReviewFlag      `json:"review_queue,omitempty"`
}

//go:embed golden_registry.json
var goldenRegistryFS embed.FS

type goldenFixture struct {
	ID       string `json:"id"`
	SHA256   string `json:"sha256"`
	Language string `json:"language"`
}

// The embedded manifest is sourced from the 28 UNREVIEWED synthetic golden
// files in knowledge/evals/golden; it is not evidence of a model execution.
func GoldenRegistry() []string {
	b, err := goldenRegistryFS.ReadFile("golden_registry.json")
	if err != nil {
		return nil
	}
	var fixtures []goldenFixture
	if json.Unmarshal(b, &fixtures) != nil {
		return nil
	}
	out := make([]string, 0, len(fixtures))
	for _, f := range fixtures {
		out = append(out, f.ID)
	}
	return out
}

type EvalCoverage struct {
	ReferenceGoldenCount  int `json:"referenceGoldenCount"`
	DeclaredGoldenMatches int `json:"declaredGoldenMatches"`
	DeclaredPersian       int `json:"declaredPersian"`
	DeclaredEnglish       int `json:"declaredEnglish"`
	DeclaredOther         int `json:"declaredOther"`
	// No report-level fixture hashes exist in WP-15. ID matches are claims,
	// not verification that those exact golden bytes were executed.
	FixtureExecutionVerified bool `json:"fixtureExecutionVerified"`
}

func ReportCoverage(r EvalReport) EvalCoverage {
	registry := GoldenRegistry()
	allowed := make(map[string]bool, len(registry))
	for _, id := range registry {
		allowed[id] = true
	}
	out := EvalCoverage{ReferenceGoldenCount: len(registry)}
	for _, scenario := range r.Scenarios {
		if allowed[scenario.ID] {
			out.DeclaredGoldenMatches++
		}
		hasFA, hasEN := false, false
		for _, tag := range scenario.Tags {
			if tag == "language_fa" {
				hasFA = true
			}
			if tag == "language_en" {
				hasEN = true
			}
		}
		if hasFA {
			out.DeclaredPersian++
		} else if hasEN {
			out.DeclaredEnglish++
		} else {
			out.DeclaredOther++
		}
	}
	return out
}

func ReportEvidence(method string) string {
	switch method {
	case "synthetic_recorded":
		return "recorded_fixture"
	case "synthetic_trace_replay":
		return "partial_trace_replay"
	default:
		return "external_executor_claimed"
	}
}

func validRegression(r EvalRegression) bool {
	return evalCode.MatchString(r.Code) && (r.Scenario == "" || evalIdentifier.MatchString(r.Scenario)) &&
		(r.Assertion == "" || evalAssertion.MatchString(r.Assertion)) && (r.Metric == "" || evalIdentifier.MatchString(r.Metric))
}

// ParseEvalReport accepts only the exact WP-15 synthetic, redacted report
// shape. Client-provided string fields are treated as untrusted and must be
// stable identifiers/codes: a report must never serve as a PII storage path.
func ParseEvalReport(data []byte) (EvalReport, []byte, error) {
	var r EvalReport
	if len(data) == 0 || len(data) > MaxEvalReportBytes {
		return r, nil, ErrInvalidEval
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if dec.Decode(&r) != nil {
		return r, nil, ErrInvalidEval
	}
	var extra any
	if dec.Decode(&extra) != io.EOF {
		return r, nil, ErrInvalidEval
	}
	if r.Schema != "eval_report.v1" || !r.SyntheticOnly ||
		(r.Method != "synthetic_recorded" && r.Method != "synthetic_trace_replay" && r.Method != "external_executor") ||
		len(r.Scenarios) == 0 || len(r.Scenarios) > 512 || len(r.Metrics) > 128 ||
		len(r.Regressions) > 1024 || len(r.ReviewQueue) > 1024 {
		return r, nil, ErrInvalidEval
	}
	totals := r.Totals
	if totals.Scenarios != len(r.Scenarios) || totals.PassedScenarios < 0 || totals.FailedScenarios < 0 ||
		totals.PassedScenarios+totals.FailedScenarios != totals.Scenarios ||
		totals.Turns < 0 || totals.PassedTurns < 0 || totals.PassedTurns > totals.Turns ||
		totals.Assertions < 0 || totals.PassedAssertions < 0 || totals.PassedAssertions > totals.Assertions ||
		totals.Failures < 0 || totals.Blockers < 0 || totals.Turns > 5000 || totals.Assertions > 200000 {
		return r, nil, ErrInvalidEval
	}
	seen := make(map[string]bool, len(r.Scenarios))
	passed, countedTurns, countedPassedTurns, countedAssertions, countedPassedAssertions := 0, 0, 0, 0, 0
	observedFailures, observedBlockers := 0, 0
	for _, s := range r.Scenarios {
		if !evalIdentifier.MatchString(s.ID) || seen[s.ID] || len(s.Tags) > 32 || len(s.Turns) == 0 || len(s.Turns) > 256 {
			return r, nil, ErrInvalidEval
		}
		seen[s.ID] = true
		if s.ReviewStatus != "synthetic_unreviewed" && s.ReviewStatus != "metadata_declared_unverified" {
			// An unsigned JSON report can NEVER self-attest clinician labels.
			return r, nil, ErrInvalidEval
		}
		hasFA, hasEN := false, false
		for _, tag := range s.Tags {
			if !evalIdentifier.MatchString(tag) {
				return r, nil, ErrInvalidEval
			}
			hasFA = hasFA || tag == "language_fa"
			hasEN = hasEN || tag == "language_en"
		}
		if hasFA && hasEN {
			return r, nil, ErrInvalidEval
		}
		scenarioFailed := false
		for _, t := range s.Turns {
			if t.Index < 0 || t.Session < 0 || t.Assertions < 0 || t.PassedAssertions < 0 || t.PassedAssertions > t.Assertions || len(t.Diagnostics) > 100 {
				return r, nil, ErrInvalidEval
			}
			// Valid WP-15 steps are zero-indexed. A synthetic setup failure
			// can add an index-zero diagnostic WITHOUT executing a step.
			setupOnly := t.Index == 0 && !t.Passed && t.Assertions == 0 && len(t.Diagnostics) > 0 &&
				(t.Diagnostics[0].Code == "EVAL_SETUP" || t.Diagnostics[0].Code == "EVAL_CANCELLED")
			if !setupOnly {
				countedTurns++
				if t.Passed {
					countedPassedTurns++
				}
			}
			countedAssertions += t.Assertions
			countedPassedAssertions += t.PassedAssertions
			for _, d := range t.Diagnostics {
				if !evalCode.MatchString(d.Code) || !evalAssertion.MatchString(d.Assertion) || (d.Severity != "failure" && d.Severity != "blocker") {
					return r, nil, ErrInvalidEval
				}
				if d.Severity == "blocker" {
					observedBlockers++
				} else {
					observedFailures++
				}
			}
			if !t.Passed {
				scenarioFailed = true
			}
			if t.Passed && (len(t.Diagnostics) != 0 || t.PassedAssertions != t.Assertions) {
				return r, nil, ErrInvalidEval
			}
		}
		if s.Passed == scenarioFailed {
			return r, nil, ErrInvalidEval
		}
		if s.Passed {
			passed++
		}
	}
	if totals.Blockers < observedBlockers || totals.Failures < observedFailures ||
		passed != totals.PassedScenarios || len(r.Scenarios)-passed != totals.FailedScenarios ||
		countedTurns != totals.Turns || countedPassedTurns != totals.PassedTurns ||
		countedAssertions != totals.Assertions || countedPassedAssertions != totals.PassedAssertions ||
		r.Passed && (totals.FailedScenarios > 0 || totals.Failures > 0 || totals.Blockers > 0 || len(r.Regressions) > 0) {
		return r, nil, ErrInvalidEval
	}
	for k, m := range r.Metrics {
		if !evalIdentifier.MatchString(k) || m.Passed < 0 || m.Total < 0 || m.Passed > m.Total || math.IsNaN(m.Rate) || math.IsInf(m.Rate, 0) ||
			m.Rate < 0 || m.Rate > 1 || (m.Total == 0 && m.Rate != 0) ||
			(m.Total > 0 && math.Abs(m.Rate-float64(m.Passed)/float64(m.Total)) > 1e-6) {
			return r, nil, ErrInvalidEval
		}
	}
	for _, reg := range r.Regressions {
		if !validRegression(reg) {
			return r, nil, ErrInvalidEval
		}
	}
	for _, flag := range r.ReviewQueue {
		if !seen[flag.Scenario] || flag.Turn < 0 || flag.Reason != "rubric_review" && flag.Reason != "safety_review" && flag.Reason != "validator_review" || flag.Status != "pending" {
			return r, nil, ErrInvalidEval
		}
	}
	normalized, err := json.Marshal(r) // reserialize allowlisted fields only
	if err != nil || len(normalized) > MaxEvalReportBytes {
		return r, nil, ErrInvalidEval
	}
	return r, normalized, nil
}

func EvalStatus(r EvalReport, computedRegressions int) string {
	if r.Totals.Blockers > 0 {
		return "blocked"
	}
	if !r.Passed || r.Totals.FailedScenarios > 0 || r.Totals.Failures > 0 || len(r.Regressions) > 0 || computedRegressions > 0 {
		return "failed"
	}
	return "passed"
}

type EvalRun struct {
	ID                   uuid.UUID    `json:"id"`
	ReportSHA256         string       `json:"reportSha256"`
	ImportedAt           time.Time    `json:"importedAt"`
	ImportedByStaffID    uuid.UUID    `json:"importedByStaffId"`
	BaselineRunID        *uuid.UUID   `json:"baselineRunId"`
	Method               string       `json:"method"`
	EvidenceType         string       `json:"evidenceType"`
	Status               string       `json:"status"`
	Totals               EvalTotals   `json:"totals"`
	Coverage             EvalCoverage `json:"coverage"`
	ArtifactHashVerified bool         `json:"artifactHashVerified"`
	SourceAttested       bool         `json:"sourceAttested"`
	ClinicallyApproved   bool         `json:"clinicallyApproved"`
	RegressionCount      int          `json:"regressionCount"`
}
type EvalRunDetail struct {
	Run                 EvalRun          `json:"run"`
	Report              EvalReport       `json:"report"`
	BaselineRegressions []EvalRegression `json:"baselineRegressions"`
}
type EvalPage struct {
	Items      []EvalRun `json:"items"`
	NextCursor *string   `json:"nextCursor"`
}
type EvalFilter struct {
	From, To time.Time
	Limit    int
	Status   string
	Cursor   *InspectorCursor
}

func NewEvalFilter(now time.Time) EvalFilter {
	end := now.UTC()
	return EvalFilter{From: end.Add(-24 * time.Hour), To: end, Limit: DefaultEvalLimit}
}
func (f EvalFilter) Validate() error {
	if f.From.IsZero() || f.To.IsZero() || !f.To.After(f.From) || f.To.Sub(f.From) > MaxEvalWindow || f.Limit < 1 || f.Limit > MaxEvalLimit ||
		(f.Status != "" && f.Status != "passed" && f.Status != "failed" && f.Status != "blocked") ||
		(f.Cursor != nil && (f.Cursor.ID == uuid.Nil || f.Cursor.At.IsZero() || f.Cursor.At.Before(f.From) || !f.Cursor.At.Before(f.To))) {
		return ErrInvalidEval
	}
	return nil
}
func ValidateArtifactHash(raw []byte, expected string) (string, error) {
	actual := sha256.Sum256(raw)
	if len(expected) != 64 {
		return "", ErrInvalidEval
	}
	b, err := hex.DecodeString(expected)
	if err != nil || !bytes.Equal(b, actual[:]) {
		return "", ErrInvalidEval
	}
	return hex.EncodeToString(actual[:]), nil
}
func SortedRunIDs(r EvalReport) []string {
	ids := make([]string, 0, len(r.Scenarios))
	for _, s := range r.Scenarios {
		ids = append(ids, s.ID)
	}
	sort.Strings(ids)
	return ids
}

func ValidateEvalRegressions(items []EvalRegression) error {
	if len(items) > 1024 {
		return ErrInvalidEval
	}
	for _, item := range items {
		if !validRegression(item) {
			return ErrInvalidEval
		}
	}
	return nil
}
