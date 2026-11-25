package trace_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
)

func TestInspectorGraphBindingsWaitForOutputsAndIsolateTurns(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic source"})
	_, producer := trace.Start(ctx, "producer", nil)
	consumerCtx, consumer := trace.Start(ctx, "consumer", func() any { return map[string]any{"copied": "synthetic value", "changed": "new value"} })
	trace.Bind(ctx, producer.Ref("/output/value"), consumer.Ref("/input/copied"), "copy")
	trace.Bind(ctx, producer.Ref("/output/value"), consumer.Ref("/input/changed"), "transform")
	trace.Bind(ctx, producer.Ref("/output/value"), consumer.Ref("/input/changed"), "copy") // deliberately false
	other, otherRoot := i.StartTurn(context.Background(), c.TurnRequest{Text: "other synthetic turn"})
	trace.Bind(other, consumer.Ref("/input/copied"), otherRoot.Ref("/input/request/text"), "transform")
	otherRoot.End(nil, nil)
	// A checkpoint while producer is still open contains no dangling data edge.
	_, checkpoint := trace.Start(consumerCtx, "checkpoint", nil)
	checkpoint.End(nil, nil)
	for _, d := range readDocuments(t, dir) {
		for _, e := range d.Edges {
			if e.Evidence == "instrumented_binding" {
				t.Fatal("published an unresolved binding")
			}
		}
	}
	producer.End(func() any { return map[string]any{"value": "synthetic value"} }, nil)
	consumer.End(nil, nil)
	root.End(nil, nil)
	bound := 0
	for _, d := range readDocuments(t, dir) {
		for _, e := range d.Edges {
			if e.Evidence == "instrumented_binding" {
				bound++
			}
		}
	}
	if bound != 2 {
		t.Fatal("missing/incorrect/cross-turn bindings", bound)
	}
}

func TestInspectorGraphPromptRangesSourcesAndSecretOmission(t *testing.T) {
	i, dir := newInspector(t)
	secret := "synthetic-credential"
	i.RegisterSecret(secret)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "متن café " + secret})
	text := "<context><memory>متن café " + secret + " &amp; note</memory><recent><message>hello</message></recent></context>"
	trace.Capture(ctx, "rendered", "rendered_prompt", "test.renderer", text)
	part := trace.Compose("context", "user", trace.ArtifactRef(ctx, "rendered"), text, func(string, map[string]string) []trace.Reference {
		return []trace.Reference{trace.Ref(ctx, "/input/request/text")}
	})
	trace.Capture(ctx, "prompt_composition", "prompt_composition", "test.renderer", []trace.PromptPart{part})
	root.End(nil, nil)
	docs := readDocuments(t, dir)
	files, _ := filepath.Glob(filepath.Join(dir, "*.json"))
	raw, _ := os.ReadFile(files[0])
	if strings.Contains(string(raw), secret) {
		t.Fatal("credential survived composition capture")
	}
	var parts []trace.PromptPart
	if err := json.Unmarshal(docs[0].Nodes[0].Artifacts[1].Value, &parts); err != nil {
		t.Fatal(err)
	}
	var rendered string
	_ = json.Unmarshal(docs[0].Nodes[0].Artifacts[0].Value, &rendered)
	memory := parts[0].Components[1]
	if memory.Parent == nil || *memory.Parent != 0 || rendered[memory.StartByte:memory.EndByte] != "<memory>متن café [authentication secret omitted] &amp; note</memory>" {
		t.Fatal("nested UTF-8 ranges did not track exported text", parts)
	}
	// Reader validation rejects references/ranges that would mislead a viewer.
	parts[0].Components[1].EndByte = len(rendered) + 1
	docs[0].Nodes[0].Artifacts[1].Value, _ = json.Marshal(parts)
	if trace.ValidateDocument(docs[0]) == nil {
		t.Fatal("accepted corrupt prompt range")
	}
	parts[0].Components[1].EndByte = memory.EndByte
	parts[0].Components[1].Sources[0].Path = "/input/missing"
	docs[0].Nodes[0].Artifacts[1].Value, _ = json.Marshal(parts)
	if trace.ValidateDocument(docs[0]) == nil {
		t.Fatal("accepted nonexistent prompt source")
	}
}

func TestInspectorGraphPlainTextAndMalformedXMLRemainReadable(t *testing.T) {
	for _, text := range []string{"plain فارسی & text", "<broken>still readable", ""} {
		part := trace.Compose("instructions", "system", trace.Reference{}, text, nil)
		if len(part.Components) != 1 || part.Components[0].StartByte != 0 || part.Components[0].EndByte != len(text) {
			t.Fatal(part)
		}
	}
}

func TestInspectorGraphRejectsFalseCopyAndExecutionParent(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{Text: "synthetic original"})
	_, child := trace.Start(ctx, "consumer", func() any { return "different value" })
	child.End(nil, nil)
	root.End(nil, nil)
	d := readDocuments(t, dir)[0]
	d.Edges = append(d.Edges, trace.Edge{From: d.Nodes[0].ID, To: d.Nodes[1].ID, Kind: "data", FromPath: "/input/request/text", ToPath: "/input", Evidence: "instrumented_binding", Relation: "copy"})
	if trace.ValidateDocument(d) == nil {
		t.Fatal("accepted false copy")
	}
	d.Edges = d.Edges[:1]
	d.Edges[0].From, d.Edges[0].To = d.Edges[0].To, d.Edges[0].From
	if trace.ValidateDocument(d) == nil {
		t.Fatal("accepted inconsistent execution hierarchy")
	}
}

func TestInspectorGraphDisabledHelpersDoNotAllocate(t *testing.T) {
	ctx := context.Background()
	if n := testing.AllocsPerRun(100, func() {
		ref := trace.Ref(ctx, "/input")
		trace.Bind(ctx, ref, trace.ArtifactRef(ctx, "missing"), "copy")
		trace.BindInputs(ctx)
		trace.ChildRef(ctx, "missing", "/output")
	}); n != 0 {
		t.Fatal("disabled graph capture allocated", n)
	}
}

func TestInspectorGraphChildRefUsesCompletedDirectInvocation(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	childCtx, child := trace.Start(ctx, "call", nil)
	_, nested := trace.Start(childCtx, "call", nil)
	nested.End(nil, nil)
	if trace.ChildRef(ctx, "call", "/output").NodeID != "" {
		t.Fatal("selected nested or unfinished operation")
	}
	child.End(func() any { return "direct result" }, nil)
	if trace.ChildRef(ctx, "call", "/output").NodeID != child.Ref("/output").NodeID {
		t.Fatal("selected another invocation")
	}
	root.End(nil, nil)
	readDocuments(t, dir)
}

func TestInspectorGraphContainerMatchDoesNotDuplicateItsChildren(t *testing.T) {
	i, dir := newInspector(t)
	ctx, root := i.StartTurn(context.Background(), c.TurnRequest{})
	value := map[string]any{"message": "synthetic sufficiently long text", "nested": map[string]string{"field": "another sufficiently long value"}}
	_, producer := trace.Start(ctx, "producer", nil)
	producer.End(func() any { return value }, nil)
	_, consumer := trace.Start(ctx, "consumer", func() any { return value })
	consumer.End(nil, nil)
	root.End(nil, nil)
	count := 0
	for _, e := range readDocuments(t, dir)[0].Edges {
		if e.Kind == "data" {
			count++
		}
	}
	if count != 1 {
		t.Fatal("redundant descendant data edges", count)
	}
}
