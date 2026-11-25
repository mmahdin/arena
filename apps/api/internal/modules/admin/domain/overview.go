package domain

import (
	"errors"
	"time"
)

// Overview represents metadata-only operational evidence. Counts are measured
// within [from,to); worker health is the latest known state as of AsOf.
// Optional metrics remain null when the underlying evidence is unavailable.
var ErrInvalidOverview = errors.New("invalid overview window")

const MaxOverviewWindow = 31 * 24 * time.Hour

type OverviewFilter struct {
	From time.Time
	To   time.Time
	AsOf time.Time
}

func NewOverviewFilter(now time.Time) OverviewFilter {
	at := now.UTC()
	return OverviewFilter{From: at.Add(-24 * time.Hour), To: at, AsOf: at}
}

func (f OverviewFilter) Validate() error {
	if f.From.IsZero() || f.To.IsZero() || f.AsOf.IsZero() ||
		!f.To.After(f.From) || f.To.Sub(f.From) > MaxOverviewWindow ||
		f.To.After(f.AsOf.Add(1*time.Second)) {
		return ErrInvalidOverview
	}
	return nil
}

type TurnSummary struct {
	// Counts reflect CURRENT state of turns created in the window, not
	// the number of transitions that happened inside the window.
	TotalV1        int64    `json:"totalV1"`
	Queued         int64    `json:"queued"`
	Running        int64    `json:"running"`
	Completed      int64    `json:"completed"`
	Failed         int64    `json:"failed"`
	Cancelled      int64    `json:"cancelled"`
	ExcludedLegacy int64    `json:"excludedLegacy"`
	FailureRate    *float64 `json:"failureRate"`  // failed / (failed+completed); null when no terminal sample
	P95ElapsedMs   *int64   `json:"p95ElapsedMs"` // created_at -> completed_at of complete v1 turns
	LatencySamples int64    `json:"latencySamples"`
}

type LLMSummary struct {
	Attempts           int64      `json:"attempts"`
	Started            int64      `json:"started"`
	Succeeded          int64      `json:"succeeded"`
	Failed             int64      `json:"failed"`
	Incomplete         int64      `json:"incomplete"`
	UsageKnown         int64      `json:"usageKnown"`
	UnknownUsage       int64      `json:"unknownUsage"`
	EstimatedCostCount int64      `json:"estimatedCostCount"`
	UnpricedCostCount  int64      `json:"unpricedCostCount"`
	UnknownCostCount   int64      `json:"unknownCostCount"`
	InputTokens        *int64     `json:"inputTokens"`
	OutputTokens       *int64     `json:"outputTokens"`
	CachedTokens       *int64     `json:"cachedTokens"`
	EstimatedMicroUSD  *int64     `json:"estimatedMicroUsd"` // estimate, NOT billing/invoice cost
	LastAttemptAt      *time.Time `json:"lastAttemptAt"`
	// CostPartial is true if any attempt lacks an estimated price or usage.
	CostPartial bool `json:"costPartial"`
}

type StageSummary struct {
	Events          int64      `json:"events"`
	Failed          int64      `json:"failed"`
	Degraded        int64      `json:"degraded"`
	DurationSamples int64      `json:"durationSamples"`
	LastEventAt     *time.Time `json:"lastEventAt"`
	// Stage coverage cannot be proven from best-effort writes.
}

type JobSummary struct {
	Queued    int64 `json:"queued"`
	Running   int64 `json:"running"`
	Failed    int64 `json:"failed"`
	DueNow    int64 `json:"dueNow"`
	Scheduled int64 `json:"scheduled"`
}

type WorkerSummary struct {
	Kind         string     `json:"kind"`
	Status       string     `json:"status"` // running, failed, stopped, stale, unknown
	FreshRunning int64      `json:"freshRunning"`
	FreshFailed  int64      `json:"freshFailed"`
	FreshStopped int64      `json:"freshStopped"`
	LastSeenAt   *time.Time `json:"lastSeenAt"`
}

type Overview struct {
	From     time.Time       `json:"from"`
	To       time.Time       `json:"to"`
	AsOf     time.Time       `json:"asOf"`
	Turns    TurnSummary     `json:"turns"`
	LLM      LLMSummary      `json:"llm"`
	Stages   StageSummary    `json:"stages"`
	Jobs     JobSummary      `json:"jobs"`
	Workers  []WorkerSummary `json:"workers"`
	Warnings []string        `json:"warnings"` // stable public codes, not internal exceptions
}

// Complete supplies derived metrics and explicit missing-evidence warnings.
// SQL always supplies the raw counters, so the application never invents zeros.
func (o *Overview) Complete() {
	terminal := o.Turns.Completed + o.Turns.Failed
	if terminal > 0 {
		v := float64(o.Turns.Failed) / float64(terminal)
		o.Turns.FailureRate = &v
	}
	o.LLM.UnknownUsage = o.LLM.Attempts - o.LLM.UsageKnown
	o.LLM.CostPartial = o.LLM.Attempts == 0 || o.LLM.EstimatedCostCount != o.LLM.Attempts
	o.Warnings = make([]string, 0)
	if o.Turns.TotalV1 == 0 {
		o.Warnings = append(o.Warnings, "NO_V1_TURNS_IN_WINDOW")
	}
	if o.Turns.Completed > o.Turns.LatencySamples {
		o.Warnings = append(o.Warnings, "INCOMPLETE_TURN_LATENCY")
	}
	if o.LLM.Attempts == 0 {
		o.Warnings = append(o.Warnings, "NO_LLM_ATTEMPTS_IN_WINDOW")
	} else if o.LLM.CostPartial {
		o.Warnings = append(o.Warnings, "PARTIAL_LLM_COST_COVERAGE")
	}
	if o.Stages.Events == 0 {
		o.Warnings = append(o.Warnings, "NO_STAGE_EVENTS_IN_WINDOW")
	}
	for _, worker := range o.Workers {
		if worker.Status != "running" {
			o.Warnings = append(o.Warnings, "WORKER_"+worker.Kind+"_"+worker.Status)
		}
	}
}
