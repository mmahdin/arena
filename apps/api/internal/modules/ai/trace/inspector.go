package trace

// The local inspector extends the trace owner without changing the metadata-only
// Tracer contract. No payload is sent to logs, metrics, SQL traces or HTTP clients.
import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/platform/telemetry"
)

const InspectorSchema = "local_ai_trace.v1"

type InspectorOptions struct {
	Directory   string
	Environment string
	Address     string
	// OnError must accept only the fixed diagnostics below, never payloads or paths.
	OnError func(error)
}

type Inspector struct {
	directory string
	onError   func(error)
	mu        sync.Mutex
	secrets   []string
	// Only active turn/job roots stay in memory. Finished turns are reloaded
	// from their private checkpoint when a durable worker claims a later job.
	active map[c.ID]*inspection
}

var ErrInspectorWrite = errors.New("local AI inspector checkpoint failed")

// NewInspector is deliberately unavailable outside a loopback local server.
// A nil Inspector is the disabled mode: no files, serialization or clocks.
func NewInspector(o InspectorOptions) (*Inspector, error) {
	if o.Environment != "local" || !loopback(o.Address) {
		return nil, errors.New("AI_LOCAL_INSPECTOR requires APP_ENV=local and a loopback HTTP_ADDR")
	}
	if o.Directory == "" {
		o.Directory = ".local/ai-inspector"
	}
	dir, err := filepath.Abs(o.Directory)
	if err != nil {
		return nil, ErrInspectorWrite
	}
	if err = os.MkdirAll(dir, 0700); err != nil {
		return nil, ErrInspectorWrite
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 || info.Mode().Perm()&0077 != 0 {
		return nil, errors.New("AI_LOCAL_INSPECTOR_DIR must be a private directory (0700)")
	}
	i := &Inspector{directory: dir, onError: o.OnError, active: make(map[c.ID]*inspection)}
	f, err := os.CreateTemp(dir, ".probe-*")
	if err != nil {
		return nil, ErrInspectorWrite
	}
	name := f.Name()
	err = f.Close()
	_ = os.Remove(name)
	if err != nil {
		return nil, ErrInspectorWrite
	}
	return i, nil
}

type Document struct {
	// ResponseComplete is independent of queued, retrying or running background work.
	ResponseComplete bool       `json:"response_complete"`
	ResponseEndedAt  *time.Time `json:"response_ended_at,omitempty"`
	BackgroundStatus string     `json:"background_status,omitempty"`
	Jobs             []JobState `json:"jobs,omitempty"`

	Complete    bool       `json:"complete"`
	Schema      string     `json:"schema"`
	TraceID     string     `json:"trace_id"`
	Kind        string     `json:"kind"`
	TurnID      c.ID       `json:"turn_id,omitempty"`
	JobID       c.ID       `json:"job_id,omitempty"`
	Attempt     int        `json:"attempt,omitempty"`
	StartedAt   time.Time  `json:"started_at"`
	EndedAt     *time.Time `json:"ended_at,omitempty"`
	Status      string     `json:"status"`
	Nodes       []*Node    `json:"nodes"`
	Edges       []Edge     `json:"edges"`
	Totals      Totals     `json:"totals"`
	Diagnostics []string   `json:"diagnostics"`
}

type Node struct {
	Embedding     *EmbeddingInfo  `json:"embedding,omitempty"`
	ID            string          `json:"id"`
	Operation     string          `json:"operation"`
	ParentID      string          `json:"parent_id,omitempty"`
	Sequence      int             `json:"sequence"`
	StartedAt     time.Time       `json:"started_at"`
	EndedAt       *time.Time      `json:"ended_at,omitempty"`
	DurationNS    int64           `json:"duration_ns"`
	Status        string          `json:"status"`
	Reason        string          `json:"reason,omitempty"`
	Input         json.RawMessage `json:"input"`
	Output        json.RawMessage `json:"output"`
	Artifacts     []Artifact      `json:"artifacts"`
	Error         *Diagnostic     `json:"error,omitempty"`
	ProviderUsage []ProviderUsage `json:"provider_usage"`
	Estimates     []Estimate      `json:"estimates"`
}
type Artifact struct {
	Name   string          `json:"name"`
	Kind   string          `json:"kind"`
	Value  json.RawMessage `json:"value"`
	SHA256 string          `json:"sha256"`
	Source string          `json:"source,omitempty"`
}
type Edge struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Kind     string `json:"kind"`
	FromPath string `json:"from_path,omitempty"`
	ToPath   string `json:"to_path,omitempty"`
	// Exact value matches are evidence of shared data, not a claim of causality.
	Evidence string `json:"evidence,omitempty"`
	Relation string `json:"relation,omitempty"`
}
type Diagnostic struct {
	Type string `json:"type"`
	Code string `json:"code,omitempty"`
	Step string `json:"step,omitempty"`
}
type Estimate struct {
	Source string `json:"source"`
	Tokens int    `json:"tokens"`
	Method string `json:"method"`
	Bytes  int    `json:"bytes"`
}
type ProviderUsage struct {
	Provider              string `json:"provider"`
	Model                 string `json:"model"`
	InputTokens           int    `json:"input_tokens"`
	OutputTokens          int    `json:"output_tokens"`
	CachedTokens          int    `json:"cached_tokens"`
	EstimatedCostMicroUSD *int64 `json:"estimated_cost_micro_usd,omitempty"`
	LatencyNS             int64  `json:"latency_ns"`
}
type Totals struct {
	EstimatedInputTokens  int                   `json:"estimated_input_tokens"`
	ProviderCalls         int                   `json:"provider_calls"`
	ReportedCalls         int                   `json:"reported_calls"`
	UnknownUsageCalls     int                   `json:"unknown_usage_calls"`
	InputTokens           int                   `json:"actual_input_tokens"`
	OutputTokens          int                   `json:"actual_output_tokens"`
	CachedTokens          int                   `json:"actual_cached_tokens"`
	EstimatedCostMicroUSD int64                 `json:"estimated_cost_micro_usd"`
	ByStage               map[string]StageUsage `json:"by_stage"`
}
type StageUsage struct {
	EstimatedInputTokens  int   `json:"estimated_input_tokens"`
	Calls                 int   `json:"calls"`
	InputTokens           int   `json:"actual_input_tokens"`
	OutputTokens          int   `json:"actual_output_tokens"`
	CachedTokens          int   `json:"actual_cached_tokens"`
	EstimatedCostMicroUSD int64 `json:"estimated_cost_micro_usd"`
}

type inspectionKey struct{}
type inspectorKey struct{}
type scope struct {
	session *inspection
	node    *Node
}
type inspection struct {
	mu       sync.Mutex
	owner    *Inspector
	doc      Document
	path     string
	secrets  []string
	sources  map[string]sourceRef
	bindings []binding
	// Protected by owner.mu. Includes in-flight child spans, so a worker
	// cannot reopen the file before a parallel operation finishes writing.
	activeRoots int
	requestKey  c.ID
}
type sourceRef struct{ node, path string }
type Span struct {
	otelCtx      context.Context
	scope        scope
	started      time.Time
	once         sync.Once
	ownerRoot    bool
	responseRoot bool
	otel         *telemetry.Span
	root         *telemetry.Span
}

func WithInspector(ctx context.Context, i *Inspector) context.Context {
	if i == nil {
		return ctx
	}
	return context.WithValue(ctx, inspectorKey{}, i)
}
func Enabled(ctx context.Context) bool {
	s := scopeFrom(ctx)
	return s.session != nil
}
func (i *Inspector) StartTurn(ctx context.Context, req c.TurnRequest) (context.Context, *Span) {
	// An AI Turn is an independent trace: no public browser traceparent is
	// accepted as an authoritative parent. Never inspect req in metadata mode.
	ctx, root := telemetry.StartRoot(ctx, "theia.ai.turn")
	if i == nil {
		i = InspectorFromContext(ctx)
	}
	if i == nil {
		if root == nil {
			return ctx, nil
		}
		return ctx, &Span{otel: root, otelCtx: ctx}
	}
	// The durable admission key also identifies replay/retry invocations. Reuse
	// its checkpoint instead of generating another file for the same user Turn.
	if c.ValidID(req.UserID) && c.ValidID(req.ClientTurnID) {
		key := uuid.NewSHA1(req.UserID, []byte("local-inspector-turn:"+req.ClientTurnID.String()))
		i.mu.Lock()
		s := i.active[key]
		if s == nil {
			s = i.loadPathLocked(filepath.Join(i.directory, key.String()+".json"))
		}
		if s == nil {
			s = i.newInspectionLocked("turn", c.ID{}, c.ID{}, 0, key.String())
		}
		s.requestKey = key
		i.active[key] = s
		s.activeRoots++
		s.mu.Lock()
		var parent *Node
		if len(s.doc.Nodes) > 0 {
			parent = s.doc.Nodes[0]
		}
		if c.ValidID(s.doc.TurnID) {
			i.active[s.doc.TurnID] = s
		}
		s.doc.ResponseComplete = false
		s.mu.Unlock()
		i.mu.Unlock()
		ctx = context.WithValue(WithInspector(ctx, i), inspectionKey{}, scope{session: s, node: parent})
		defer i.releaseRoot(s)
	} else {
		ctx = i.start(ctx, "turn", c.ID{}, c.ID{}, 0)
	}
	ctx, span := Start(ctx, "orchestrator.ProcessTurn", func() any { return map[string]any{"request": req} })
	span.ownerRoot = true
	span.responseRoot = true
	span.root = root
	return ctx, span
}

// StartJob attaches a claimed worker attempt below the original user turn.
// Without an existing checkpoint, a correlated worker creates a Turn envelope.
// Only a worker with no Turn creates an independent job document.
// User-level analysis need not fabricate a conversational turn.
// On process restart the turn's private checkpoint is loaded from disk; this
// allows durable jobs to be inspected without keeping completed turns in RAM.
func StartJob(ctx context.Context, turn, job c.ID, kind string, attempt int) (context.Context, *Span) {
	ctx = context.WithValue(ctx, turnKey{}, turn)
	i := InspectorFromContext(ctx)
	if (i == nil && !telemetry.Enabled(ctx)) || !c.ValidID(job) {
		return ctx, nil
	}
	// OTLP jobs start independent roots while the offline Inspector links the
	// attempt to the original Turn document when a checkpoint is available.
	ctx, otelRoot := telemetry.StartRoot(ctx, "theia.ai."+kind+"_job",
		telemetry.String("theia.turn_id", turn.String()),
		telemetry.String("theia.job_id", job.String()),
		telemetry.String("theia.job_kind", kind), telemetry.Int("theia.job_attempt", int64(attempt)))
	if i == nil {
		if otelRoot == nil {
			return ctx, nil
		}
		return ctx, &Span{otel: otelRoot, otelCtx: ctx}
	}
	i.mu.Lock()
	s := i.active[turn]
	if s == nil {
		s = i.loadTurnLocked(turn)
		if s != nil {
			i.active[turn] = s
		}
	}
	if s == nil && c.ValidID(turn) {
		// A worker enabled after admission still belongs to a Turn. Keep one
		// reloadable envelope, with an explicit gap instead of fabricating a response.
		id := turn.String()
		now := time.Now().UTC()
		n := &Node{ID: "n000001", Sequence: 1, Operation: "turn.Unobserved", StartedAt: now, EndedAt: &now, Status: "skipped", Reason: "response_pipeline_not_observed", Input: json.RawMessage("null"), Output: json.RawMessage("null"), Artifacts: []Artifact{}, ProviderUsage: []ProviderUsage{}, Estimates: []Estimate{}}
		s = &inspection{owner: i, path: filepath.Join(i.directory, id+".json"), sources: map[string]sourceRef{}, secrets: append([]string(nil), i.secrets...), doc: Document{Schema: InspectorSchema, TraceID: id, Kind: "turn", TurnID: turn, StartedAt: now, Status: "running", Nodes: []*Node{n}, Edges: []Edge{}, Diagnostics: []string{"response_pipeline_not_observed"}}}
		s.checkpointLocked()
		i.active[turn] = s
	}
	if s == nil {
		i.mu.Unlock()
		ctx = i.start(ctx, "job", turn, job, attempt)
		ctx, root := Start(ctx, kind+".RunOnce", func() any { return map[string]any{"kind": kind, "job_id": job, "attempt": attempt} })
		root.ownerRoot = true
		root.root = otelRoot
		return ctx, root
	}
	s.activeRoots++ // reserve this session until Start registers its span
	i.mu.Unlock()
	s.mu.Lock()
	turnRoot := s.doc.Nodes[0]
	s.mu.Unlock()

	// Workers are asynchronous. The original root gives them a navigable
	// execution parent, while the actual attempt identity is in node.input.
	ctx = context.WithValue(WithInspector(ctx, i), inspectionKey{}, scope{session: s, node: turnRoot})
	ctx, root := Start(ctx, kind+".RunOnce", func() any {
		return map[string]any{"kind": kind, "turn_id": turn, "job_id": job, "attempt": attempt}
	})
	root.ownerRoot = true
	root.root = otelRoot
	i.releaseRoot(s) // the new span holds its own reference
	return ctx, root
}

// loadTurnLocked is called with Inspector.mu held. Do not follow symlinks or
// accept another turn's content. Invalid/partial files remain untouched.
func (i *Inspector) loadTurnLocked(turn c.ID) *inspection {
	if !c.ValidID(turn) {
		return nil
	}
	entries, err := os.ReadDir(i.directory)
	if err != nil {
		return nil
	}
	var selected *inspection
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		path := filepath.Join(i.directory, entry.Name())
		s := i.loadPathLocked(path)
		if s == nil || s.doc.TurnID != turn {
			continue
		}
		if selected != nil && !s.doc.StartedAt.Before(selected.doc.StartedAt) {
			continue
		}
		selected = s
	}
	return selected
}

func (i *Inspector) loadPathLocked(path string) *inspection {
	for _, active := range i.active {
		if active.path == path {
			return active
		}
	}
	info, err := os.Lstat(path)
	if err != nil || !info.Mode().IsRegular() {
		return nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var doc Document
	if json.Unmarshal(raw, &doc) != nil || doc.Kind != "turn" || ValidateDocument(doc) != nil {
		return nil
	}
	if !bytes.Contains(raw, []byte(`"response_complete"`)) && len(doc.Nodes) > 0 && doc.Nodes[0].EndedAt != nil {
		doc.ResponseComplete = true
		doc.ResponseEndedAt = doc.Nodes[0].EndedAt
	}
	s := &inspection{owner: i, path: path, doc: doc, sources: map[string]sourceRef{}, secrets: append([]string(nil), i.secrets...)}
	for _, n := range doc.Nodes {
		if n.Status != "running" {
			s.indexOutput(n, n.Output)
		}
		for j, a := range n.Artifacts {
			s.indexValue(n.ID, "/artifacts/"+fmt.Sprint(j)+"/value", a.Value)
		}
	}
	return s
}

func (i *Inspector) releaseRoot(s *inspection) {
	i.mu.Lock()
	defer i.mu.Unlock()
	s.activeRoots--
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.activeRoots == 0 {
		if i.active[s.doc.TurnID] == s {
			delete(i.active, s.doc.TurnID)
		}
		if i.active[s.requestKey] == s {
			delete(i.active, s.requestKey)
		}
	}
}

func (i *Inspector) start(ctx context.Context, kind string, turn, job c.ID, attempt int) context.Context {
	i.mu.Lock()
	s := i.newInspectionLocked(kind, turn, job, attempt, c.NewID().String())
	i.mu.Unlock()
	return context.WithValue(WithInspector(ctx, i), inspectionKey{}, scope{session: s})
}

func (i *Inspector) newInspectionLocked(kind string, turn, job c.ID, attempt int, id string) *inspection {
	return &inspection{secrets: append([]string(nil), i.secrets...), owner: i, path: filepath.Join(i.directory, id+".json"), sources: map[string]sourceRef{}, doc: Document{Schema: InspectorSchema, TraceID: id, Kind: kind, TurnID: turn, JobID: job, Attempt: attempt, StartedAt: time.Now().UTC(), Status: "running", Nodes: []*Node{}, Edges: []Edge{}, Diagnostics: []string{}, Totals: Totals{ByStage: map[string]StageUsage{}}}}
}

// Start uses lazy input capture so disabled execution never builds input maps.
func Start(ctx context.Context, operation string, input func() any) (context.Context, *Span) {
	// Preserve the old zero-allocation, zero-capture disabled path.
	parent := scopeFrom(ctx)
	if parent.session == nil && !telemetry.Enabled(ctx) {
		return ctx, nil
	}
	name := "theia.ai.stage"
	// Operations originate in code, but never treat an arbitrary dynamic
	// operation string as an unbounded span name or user-derived attribute.
	if len(operation) > 0 && len(operation) <= 80 {
		valid := true
		for _, r := range operation {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
				(r >= '0' && r <= '9') || r == '.' || r == '_' || r == '/' || r == '-' {
				continue
			}
			valid = false
			break
		}
		if valid {
			name = "theia.ai." + strings.ReplaceAll(operation, "/", ".")
		}
	}
	ctx, otel := telemetry.Start(ctx, name)
	if parent.session == nil {
		if otel == nil {
			return ctx, nil
		}
		// Evaluate the lazy input only when the private runtime permits content.
		if telemetry.ContentEnabled(ctx) && input != nil {
			captureOTelStage(ctx, "input", input)
		}
		return ctx, &Span{otel: otel, otelCtx: ctx}
	}
	s := parent.session
	s.owner.mu.Lock()
	s.activeRoots++
	s.owner.mu.Unlock()
	at := time.Now()
	var raw json.RawMessage
	if input != nil {
		raw = snapshot(input())
	} else {
		raw = json.RawMessage("null")
	}
	telemetry.StageEvent(ctx, "input", raw)
	s.mu.Lock()
	n := &Node{ID: fmt.Sprintf("n%06d", len(s.doc.Nodes)+1), Operation: operation, Sequence: len(s.doc.Nodes) + 1, StartedAt: at.UTC(), Status: "running", Input: raw, Output: json.RawMessage("null"), Artifacts: []Artifact{}, ProviderUsage: []ProviderUsage{}, Estimates: []Estimate{}}
	if parent.node != nil {
		n.ParentID = parent.node.ID
	} else if s.doc.Kind == "turn" && len(s.doc.Nodes) > 0 {
		n.ParentID = s.doc.Nodes[0].ID
	}
	if n.ParentID != "" {
		s.doc.Edges = append(s.doc.Edges, Edge{From: n.ParentID, To: n.ID, Kind: "execution"})
	}
	s.doc.Nodes = append(s.doc.Nodes, n)
	s.matchInput(n, raw)
	s.checkpointLocked()
	s.mu.Unlock()
	sc := scope{s, n}
	return context.WithValue(ctx, inspectionKey{}, sc), &Span{scope: sc, started: at, otel: otel, otelCtx: ctx}
}

func (p *Span) End(output func() any, err error) { p.end(output, err, nil) }
func (p *Span) end(output func() any, err error, skipped *string) {
	if p == nil {
		return
	}
	p.once.Do(func() {
		defer p.root.End(err)
		defer p.otel.End(err)
		if p.scope.session == nil {
			if telemetry.ContentEnabled(p.otelCtx) && output != nil {
				captureOTelStage(p.otelCtx, "output", output)
			}
			return
		}
		raw := json.RawMessage("null")
		if output != nil {
			raw = snapshot(output())
		}
		telemetry.StageEvent(p.otelCtx, "output", raw)
		s, n := p.scope.session, p.scope.node
		now := time.Now().UTC()
		s.mu.Lock()
		n.Output = raw
		n.EndedAt = &now
		n.DurationNS = time.Since(p.started).Nanoseconds()
		n.Status = "success"
		if err != nil {
			n.Status = "error"
			n.Error = diagnostic(err)
			if errors.Is(err, context.Canceled) {
				n.Status = "cancelled"
			}
		}
		if skipped != nil {
			n.Status = "skipped"
			n.Reason = *skipped
		}
		s.indexOutput(n, raw)
		if n.ParentID == "" || p.responseRoot {
			// Main turn outcome, independent of asynchronous job failures.
			if s.doc.Kind == "turn" {
				s.doc.ResponseComplete = true
				s.doc.ResponseEndedAt = &now
			}
			if s.doc.EndedAt == nil || now.After(*s.doc.EndedAt) {
				s.doc.EndedAt = &now
			}
			s.doc.Status = n.Status
		} else if p.ownerRoot && (s.doc.EndedAt == nil || now.After(*s.doc.EndedAt)) {
			// The trace's end time includes the last observed worker attempt.
			s.doc.EndedAt = &now
		}
		s.checkpointLocked()
		s.mu.Unlock()
		s.owner.releaseRoot(s)
	})
}

// A snapshot closure in the OTel-only path must not interrupt AI execution.
func captureOTelStage(ctx context.Context, direction string, value func() any) {
	defer func() {
		if recover() != nil {
			telemetry.StageEvent(ctx, direction, map[string]any{"capture_error": "snapshot_panicked"})
		}
	}()
	telemetry.StageEvent(ctx, direction, value())
}

func (p *Span) Finish(output func() any, err error) { p.End(output, err) }
func Associate(ctx context.Context, turn c.ID) {
	telemetry.Annotate(ctx, telemetry.String("theia.turn_id", turn.String()))
	sc := scopeFrom(ctx)
	if sc.session == nil || !c.ValidID(turn) {
		return
	}
	s := sc.session
	s.owner.mu.Lock()
	s.mu.Lock()
	if old := s.doc.TurnID; old != turn && s.owner.active[old] == s {
		delete(s.owner.active, old)
	}
	s.doc.TurnID = turn
	// Persist the association immediately: a worker can claim the job after
	// admission, even when the turn itself is still generating its answer.
	s.checkpointLocked()
	if existing := s.owner.active[turn]; existing == nil || existing == s {
		s.owner.active[turn] = s
	}
	s.mu.Unlock()
	s.owner.mu.Unlock()
}
func Secret(ctx context.Context, value string) {
	telemetry.RegisterStageSecret(ctx, value)
	sc := scopeFrom(ctx)
	if sc.session == nil || value == "" {
		return
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	for _, v := range sc.session.secrets {
		if v == value {
			return
		}
	}
	sc.session.secrets = append(sc.session.secrets, value)
}

// Capture is generic so disabled calls do not box large stage values into heap-allocated interfaces.
func Capture[T any](ctx context.Context, name, kind, source string, value T) {
	sc := scopeFrom(ctx)
	if sc.node == nil && !telemetry.ContentEnabled(ctx) {
		return
	}
	telemetry.StageArtifact(ctx, name, kind, value)
	if sc.node == nil {
		return
	}
	raw := snapshot(value)
	sum := sha256.Sum256(raw)
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	sc.node.Artifacts = append(sc.node.Artifacts, Artifact{Name: name, Kind: kind, Value: raw, SHA256: fmt.Sprintf("%x", sum), Source: source})
	sc.session.indexValue(sc.node.ID, "/artifacts/"+fmt.Sprint(len(sc.node.Artifacts)-1)+"/value", raw)
}
func Skip(ctx context.Context, operation, reason string) {
	_, p := Start(ctx, operation, nil)
	if p == nil {
		return
	}
	p.end(nil, nil, &reason)
}
func Mark(ctx context.Context, status, reason string) {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return
	}
	sc.session.mu.Lock()
	sc.node.Reason = reason
	sc.session.mu.Unlock()
	Capture(ctx, "decision", "control_flow", "", map[string]string{"status": status, "reason": reason})
}

// Usage is attached once at the actual provider attempt, never at aggregating
// Brain/Generator return values. This prevents double-counting retries/hops.
func Usage(ctx context.Context, u c.LLMUsage, cost *int64) {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return
	}
	if u.InTokens < 0 || u.OutTokens < 0 || u.CachedTokens < 0 || u.CachedTokens > u.InTokens {
		Capture(ctx, "invalid_provider_usage", "diagnostic", "provider", u)
		return
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	node := sc.node
	// Adapters can expose usage from invalid responses even if their normal
	// return value deliberately omits it. Attribute it to the provider attempt.
	for parent := node; parent != nil; {
		if strings.HasPrefix(parent.Operation, "provider.") {
			node = parent
			break
		}
		var next *Node
		for _, n := range sc.session.doc.Nodes {
			if n.ID == parent.ParentID {
				next = n
				break
			}
		}
		parent = next
	}
	if len(node.ProviderUsage) != 0 {
		return
	} // malformed duplicate usage is retained in raw artifacts, never double charged
	node.ProviderUsage = append(node.ProviderUsage, ProviderUsage{Provider: u.Provider, Model: u.Model, InputTokens: u.InTokens, OutputTokens: u.OutTokens, CachedTokens: u.CachedTokens, EstimatedCostMicroUSD: cost, LatencyNS: int64(u.Latency)})
}

func EstimateText(ctx context.Context, source, text string) {
	sc := scopeFrom(ctx)
	if sc.node == nil {
		return
	}
	sc.session.mu.Lock()
	defer sc.session.mu.Unlock()
	sc.node.Estimates = append(sc.node.Estimates, Estimate{Source: source, Tokens: (utf8.RuneCountInString(text) + 2) / 3, Method: "ceil_unicode_codepoints_div_3; excludes provider framing; not a tokenizer", Bytes: len(text)})
}
func diagnostic(err error) *Diagnostic {
	d := &Diagnostic{Type: fmt.Sprintf("%T", err)}
	var ce *c.Error
	if errors.As(err, &ce) {
		d.Code = string(ce.Code)
		d.Step = ce.Step
	}
	if errors.Is(err, context.DeadlineExceeded) {
		d.Code = "deadline_exceeded"
	}
	if errors.Is(err, context.Canceled) {
		d.Code = "cancelled"
	}
	return d
}
func snapshot(v any) json.RawMessage {
	// MarshalJSON on contract types remains authoritative. []byte fields in those
	// types retain Go's base64 encoding; dedicated raw/prompt artifacts are text.
	raw, err := json.Marshal(v)
	if err != nil {
		return json.RawMessage(`{"capture_error":"unsupported_value"}`)
	}
	var value any
	decoder := json.NewDecoder(strings.NewReader(string(raw)))
	decoder.UseNumber()
	if decoder.Decode(&value) == nil {
		scrubCredentialFields(value)
		if clean, e := json.Marshal(value); e == nil {
			return clean
		}
	}
	return raw
}
func (s *inspection) indexOutput(n *Node, raw json.RawMessage) { s.indexValue(n.ID, "/output", raw) }
func walkValue(path string, v any, fn func(string, any)) {
	fn(path, v)
	switch x := v.(type) {
	case map[string]any:
		for k, item := range x {
			walkValue(path+"/"+strings.ReplaceAll(strings.ReplaceAll(k, "~", "~0"), "/", "~1"), item, fn)
		}
	case []any:
		for i, item := range x {
			walkValue(path+"/"+fmt.Sprint(i), item, fn)
		}
	}
}
func valueKey(v any) string {
	switch x := v.(type) {
	case nil, bool, float64, json.Number:
		return ""
	case string:
		if len(x) < 12 {
			return ""
		}
	case []any:
		if len(x) == 0 {
			return ""
		}
	case map[string]any:
		if len(x) == 0 {
			return ""
		}
	}
	raw, _ := json.Marshal(v)
	h := sha256.Sum256(raw)
	return fmt.Sprintf("%x", h)
}
func (s *inspection) indexValue(node, path string, raw json.RawMessage) {
	var v any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&v) != nil {
		return
	}
	walkValue(path, v, func(path string, v any) {
		if key := valueKey(v); key != "" {
			s.sources[key] = sourceRef{node, path}
		}
	})
}
func (s *inspection) matchInput(n *Node, raw json.RawMessage) {
	var v any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if decoder.Decode(&v) != nil {
		return
	}
	var match func(string, any)
	match = func(path string, v any) {
		if src, ok := s.sources[valueKey(v)]; ok && src.node != n.ID {
			s.doc.Edges = append(s.doc.Edges, Edge{From: src.node, To: n.ID, Kind: "data", FromPath: src.path, ToPath: path, Evidence: "exact_json_value_match"})
			return // this match already covers every descendant of the same value
		}
		switch value := v.(type) {
		case map[string]any:
			for key, item := range value {
				match(path+"/"+strings.ReplaceAll(strings.ReplaceAll(key, "~", "~0"), "/", "~1"), item)
			}
		case []any:
			for i, item := range value {
				match(path+"/"+fmt.Sprint(i), item)
			}
		}
	}
	match("/input", v)
}
func (s *inspection) totalsLocked() {
	t := Totals{ByStage: map[string]StageUsage{}}
	byID := map[string]*Node{}
	for _, n := range s.doc.Nodes {
		byID[n.ID] = n
	}
	for _, n := range s.doc.Nodes {
		if strings.HasPrefix(n.Operation, "provider.") && n.Status != "skipped" {
			estimated := 0
			for _, e := range n.Estimates {
				if !strings.Contains(e.Source, "/section/") {
					estimated += e.Tokens
				}
			}
			t.EstimatedInputTokens += estimated
			stage := n.Operation
			for p := byID[n.ParentID]; p != nil; p = byID[p.ParentID] {
				if !strings.HasPrefix(p.Operation, "llm.") && !strings.HasPrefix(p.Operation, "llm/") {
					stage = p.Operation
					break
				}
			}
			su := t.ByStage[stage]
			su.EstimatedInputTokens += estimated
			su.Calls++
			t.ByStage[stage] = su
			t.ProviderCalls++
			if len(n.ProviderUsage) == 0 {
				t.UnknownUsageCalls++
			}
		}
		for _, u := range n.ProviderUsage {
			t.ReportedCalls++
			t.InputTokens += u.InputTokens
			t.OutputTokens += u.OutputTokens
			t.CachedTokens += u.CachedTokens
			if u.EstimatedCostMicroUSD != nil {
				t.EstimatedCostMicroUSD += *u.EstimatedCostMicroUSD
			}
			stage := n.Operation
			for p := byID[n.ParentID]; p != nil; p = byID[p.ParentID] {
				if !strings.HasPrefix(p.Operation, "llm.") && !strings.HasPrefix(p.Operation, "llm/") {
					stage = p.Operation
					break
				}
			}

			a := t.ByStage[stage]
			a.InputTokens += u.InputTokens
			a.OutputTokens += u.OutputTokens
			a.CachedTokens += u.CachedTokens
			if u.EstimatedCostMicroUSD != nil {
				a.EstimatedCostMicroUSD += *u.EstimatedCostMicroUSD
			}
			t.ByStage[stage] = a
		}
	}
	s.doc.Totals = t
}
func (s *inspection) checkpointLocked() {
	s.bindingsLocked()
	s.totalsLocked()
	s.doc.Complete = s.doc.EndedAt != nil
	if s.doc.Kind == "turn" && !s.doc.ResponseComplete {
		s.doc.Complete = false
	}
	if s.doc.BackgroundStatus != "" && s.doc.BackgroundStatus != "unknown" {
		s.doc.BackgroundStatus = "complete"
		for _, job := range s.doc.Jobs {
			if job.Status == "running" {
				s.doc.BackgroundStatus = "running"
				break
			}
			if job.Status == "queued" {
				s.doc.BackgroundStatus = "pending"
			}
		}
	}
	for _, n := range s.doc.Nodes {
		if n.Status == "running" {
			s.doc.Complete = false
			if n.ParentID != "" && (strings.HasSuffix(n.Operation, ".RunOnce") || n.Operation == "embedding.compute") && s.doc.BackgroundStatus != "unknown" {
				s.doc.BackgroundStatus = "running"
			}
		}
	}
	if s.doc.BackgroundStatus == "pending" || s.doc.BackgroundStatus == "running" || s.doc.BackgroundStatus == "unknown" {
		s.doc.Complete = false
	}
	raw, err := json.MarshalIndent(s.doc, "", "  ")
	if err == nil {
		if len(s.secrets) > 0 {
			var value any
			decoder := json.NewDecoder(bytes.NewReader(raw))
			decoder.UseNumber()
			if err = decoder.Decode(&value); err == nil && scrubSecrets(value, s.secrets) {
				raw, err = json.Marshal(value)
				var exported Document
				if err == nil {
					err = json.Unmarshal(raw, &exported)
				}
				if err == nil {
					refreshPromptRanges(&exported)
					for _, node := range exported.Nodes {
						for index := range node.Artifacts {
							artifact := &node.Artifacts[index]
							var compact bytes.Buffer
							if e := json.Compact(&compact, artifact.Value); e == nil {
								sum := sha256.Sum256(compact.Bytes())
								artifact.SHA256 = fmt.Sprintf("%x", sum)
							}
						}
					}
					raw, err = json.MarshalIndent(exported, "", "  ")
				}
			}
		}

		if err == nil {
			err = atomicWrite(s.path, raw)
		}
	}
	if err != nil {
		if len(s.doc.Diagnostics) == 0 {
			s.doc.Diagnostics = append(s.doc.Diagnostics, "checkpoint_write_failed")
		}
		if s.owner.onError != nil {
			s.owner.onError(ErrInspectorWrite)
		}
	}
}
func atomicWrite(path string, raw []byte) error {
	f, err := os.CreateTemp(filepath.Dir(path), ".trace-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	defer os.Remove(tmp)
	if _, err = f.Write(raw); err != nil {
		f.Close()
		return err
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return err
	}
	if err = f.Close(); err != nil {
		return err
	}
	if err = os.Rename(tmp, path); err != nil {
		return err
	}
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer dir.Close()
	return dir.Sync()
}

// ValidateDocument checks graph and timing invariants in addition to JSON Schema.
func ValidateDocument(d Document) error {
	if d.Schema != InspectorSchema || d.TraceID == "" || d.StartedAt.IsZero() || len(d.Nodes) == 0 {
		return errors.New("invalid inspector header")
	}
	ids := map[string]*Node{}
	roots := 0
	for i, n := range d.Nodes {
		if n == nil || n.ID == "" || ids[n.ID] != nil || n.Sequence != i+1 || n.Operation == "" || n.StartedAt.IsZero() || n.DurationNS < 0 || !json.Valid(n.Input) || !json.Valid(n.Output) {
			return errors.New("invalid inspector node")
		}
		if n.ParentID == "" {
			roots++
		} else if ids[n.ParentID] == nil {
			return errors.New("missing inspector parent")
		}
		switch n.Status {
		case "running":
		case "success", "error", "cancelled", "skipped":
			if n.EndedAt == nil || n.EndedAt.Before(n.StartedAt) {
				return errors.New("invalid inspector timing")
			}
		default:
			return errors.New("invalid inspector status")
		}
		ids[n.ID] = n
	}
	if roots != 1 {
		return errors.New("invalid inspector root")
	}
	for _, e := range d.Edges {
		if ids[e.From] == nil || ids[e.To] == nil || e.From == e.To {
			return errors.New("invalid inspector edge")
		}
	}
	return validateGraph(d)
}

// Complete observes panic unwinding without swallowing or replacing the panic.
// Call recover directly in the deferred closure at the instrumented boundary.
func (p *Span) Complete(output func() any, err error, panicked any) {
	if panicked != nil {
		p.End(output, errors.New("operation panicked"))
		panic(panicked)
	}
	p.End(output, err)
}
func InspectorFromContext(ctx context.Context) *Inspector {
	if ctx == nil {
		return nil
	}
	i, _ := ctx.Value(inspectorKey{}).(*Inspector)
	return i
}

func (i *Inspector) RegisterSecret(secret string) {
	if i == nil || secret == "" {
		return
	}
	i.mu.Lock()
	defer i.mu.Unlock()
	i.secrets = append(i.secrets, secret)
}
func scrubCredentialFields(value any) {
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			normalized := strings.ToLower(strings.ReplaceAll(key, "_", ""))
			switch normalized {
			case "authorization", "apikey", "password", "accesstoken", "refreshtoken", "leasetoken", "sessionsecret":
				v[key] = "[authentication secret omitted]"
			default:
				scrubCredentialFields(item)
			}
		}
	case []any:
		for _, item := range v {
			scrubCredentialFields(item)
		}
	}
}

// Call observes an invocation at a consumer-owned port, including substituted
// implementations. It neither decorates the interface nor hides marker methods.
func Call[T any](ctx context.Context, operation string, input func() any, call func(context.Context) (T, error)) (result T, err error) {
	ctx, span := Start(ctx, operation, input)
	defer func() { span.Complete(func() any { return result }, err, recover()) }()
	return call(ctx)
}

func scopeFrom(ctx context.Context) scope {
	if ctx == nil {
		return scope{}
	}
	s, _ := ctx.Value(inspectionKey{}).(scope)
	return s
}

// Go byte fields are base64 on the wire. An echoed credential must not survive
// encoded in a structured response/checkpoint after the readable copy is scrubbed.
func scrubSecrets(value any, secrets []string) bool {
	changed := false
	clean := func(text string) string {
		for _, secret := range secrets {
			text = strings.ReplaceAll(text, secret, "[authentication secret omitted]")
		}
		if decoded, err := base64.StdEncoding.DecodeString(text); err == nil && len(decoded) > 0 {
			old := string(decoded)
			safe := old
			for _, secret := range secrets {
				safe = strings.ReplaceAll(safe, secret, "[authentication secret omitted]")
			}
			if safe != old {
				text = base64.StdEncoding.EncodeToString([]byte(safe))
			}
		}
		return text
	}
	switch v := value.(type) {
	case map[string]any:
		for key, item := range v {
			if text, ok := item.(string); ok {
				safe := clean(text)
				if safe != text {
					v[key] = safe
					changed = true
				}
			} else if scrubSecrets(item, secrets) {
				changed = true
			}
		}
	case []any:
		for index, item := range v {
			if text, ok := item.(string); ok {
				safe := clean(text)
				if safe != text {
					v[index] = safe
					changed = true
				}
			} else if scrubSecrets(item, secrets) {
				changed = true
			}
		}
	}
	return changed
}

// ErrorInfo exposes diagnostic identity without serializing arbitrary vendor error fields.
func ErrorInfo(err error) *Diagnostic {
	if err == nil {
		return nil
	}
	return diagnostic(err)
}

// Optional identities stay absent for standalone user-level background work.
func (d Document) MarshalJSON() ([]byte, error) {
	type documentAlias Document
	var turn, job *c.ID
	if c.ValidID(d.TurnID) {
		id := d.TurnID
		turn = &id
	}
	if c.ValidID(d.JobID) {
		id := d.JobID
		job = &id
	}
	return json.Marshal(struct {
		documentAlias
		Turn *c.ID `json:"turn_id,omitempty"`
		Job  *c.ID `json:"job_id,omitempty"`
	}{documentAlias(d), turn, job})
}
