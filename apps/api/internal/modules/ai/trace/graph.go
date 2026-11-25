package trace

// References and composition indexes reuse captured values; they never contain
// a second copy of a prompt or introduce an architectural/predeclared DAG.
import (
	"bytes"
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
	"unicode/utf8"
)

type Reference struct {
	NodeID  string `json:"node_id"`
	Path    string `json:"path"` // RFC 6901, relative to the referenced node
	session *inspection
}

func Ref(ctx context.Context, path string) Reference {
	s := scopeFrom(ctx)
	if s.node == nil {
		return Reference{}
	}
	return Reference{NodeID: s.node.ID, Path: path, session: s.session}
}

func (p *Span) Ref(path string) Reference {
	if p == nil || p.scope.node == nil {
		return Reference{}
	}
	return Reference{NodeID: p.scope.node.ID, Path: path, session: p.scope.session}
}

// ChildRef is for a call site immediately after a synchronous invocation of a
// known instrumented boundary. Restricting it to completed direct children
// avoids selecting another branch's nested operation. Uninstrumented ports
// return no reference; callers must not use this for parallel identical calls.
func ChildRef(ctx context.Context, operation, path string) Reference {
	s := scopeFrom(ctx)
	if s.node == nil {
		return Reference{}
	}
	s.session.mu.Lock()
	defer s.session.mu.Unlock()
	for i := len(s.session.doc.Nodes) - 1; i >= 0; i-- {
		n := s.session.doc.Nodes[i]
		if n.ParentID == s.node.ID && n.Operation == operation && n.Status != "running" {
			return Reference{NodeID: n.ID, Path: path, session: s.session}
		}
	}
	return Reference{}
}

// ArtifactRef refers to the most recent artifact of this name on this node.
func ArtifactRef(ctx context.Context, name string) Reference {
	s := scopeFrom(ctx)
	if s.node == nil {
		return Reference{}
	}
	s.session.mu.Lock()
	defer s.session.mu.Unlock()
	for i := len(s.node.Artifacts) - 1; i >= 0; i-- {
		if s.node.Artifacts[i].Name == name {
			return Reference{NodeID: s.node.ID, Path: "/artifacts/" + strconv.Itoa(i) + "/value", session: s.session}
		}
	}
	return Reference{}
}

type binding struct {
	from, to Reference
	relation string
}

// Bind is called where code actually passes/transforms a value. Unlike automatic
// matches, it records an observed dependency. Output pointers can be registered
// before return; they are exported only when both endpoints have been captured.
// A copy binding additionally requires equal JSON values. Cross-turn links and
// intra-node mappings are excluded (the latter live in prompt composition).
func Bind(ctx context.Context, from, to Reference, relation string) {
	s := scopeFrom(ctx).session
	if s == nil || from.session != s || to.session != s || from.NodeID == to.NodeID {
		return
	}
	if relation != "copy" && relation != "transform" && relation != "regeneration" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bindings = append(s.bindings, binding{from, to, relation})
}

func (s *inspection) bindingsLocked() {
	pending := s.bindings[:0]
	for _, b := range s.bindings {
		from, okFrom := resolveRef(s.doc, b.from)
		to, okTo := resolveRef(s.doc, b.to)
		if !okFrom || !okTo {
			pending = append(pending, b)
			continue
		}
		if b.relation == "copy" && !reflect.DeepEqual(from, to) {
			continue // a transformation must never be advertised as an exact copy
		}
		edge := Edge{From: b.from.NodeID, To: b.to.NodeID, Kind: "data", FromPath: b.from.Path, ToPath: b.to.Path, Evidence: "instrumented_binding", Relation: b.relation}
		duplicate := false
		for _, e := range s.doc.Edges {
			if e == edge {
				duplicate = true
				break
			}
		}
		if !duplicate {
			s.doc.Edges = append(s.doc.Edges, edge)
		}
	}
	s.bindings = pending
}

func resolveRef(d Document, ref Reference) (any, bool) {
	for _, n := range d.Nodes {
		if n.ID != ref.NodeID {
			continue
		}
		parts := strings.Split(ref.Path, "/")
		if len(parts) < 2 || parts[0] != "" {
			return nil, false
		}
		var raw []byte
		switch parts[1] {
		case "input":
			raw = n.Input
		case "output":
			raw = n.Output
		case "artifacts":
			raw, _ = json.Marshal(n.Artifacts)
		default:
			return nil, false
		}
		if parts[1] == "output" && n.Status == "running" {
			return nil, false
		}
		var value any
		decoder := json.NewDecoder(bytes.NewReader(raw))
		decoder.UseNumber()
		if decoder.Decode(&value) != nil {
			return nil, false
		}
		for _, part := range parts[2:] {
			part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
			switch v := value.(type) {
			case map[string]any:
				var ok bool
				value, ok = v[part]
				if !ok {
					return nil, false
				}
			case []any:
				i, err := strconv.Atoi(part)
				if err != nil || i < 0 || i >= len(v) {
					return nil, false
				}
				value = v[i]
			default:
				return nil, false
			}
		}
		return value, true
	}
	return nil, false
}

type PromptPart struct {
	Name       string            `json:"name"`
	Role       string            `json:"role"`
	Rendered   Reference         `json:"rendered"`
	Source     string            `json:"source,omitempty"` // loaded bundle path, or named code-derived source
	Components []PromptComponent `json:"components"`
}
type PromptComponent struct {
	Name      string      `json:"name"`
	Parent    *int        `json:"parent,omitempty"` // index in this part's components
	StartByte int         `json:"start_byte"`
	EndByte   int         `json:"end_byte"` // half-open UTF-8 bytes in the decoded JSON string
	Sources   []Reference `json:"sources"`
}

// Compose indexes the actual rendered string, including nested XML elements.
// Sources are supplied by the renderer that knows the transformation, not by
// matching prose. Plain/non-XML blocks remain readable as a single component.
// Invoke only while inspection is enabled; it does not render or alter prompts.
func Compose(name, role string, rendered Reference, text string, sources func(string, map[string]string) []Reference) PromptPart {
	p := PromptPart{Name: name, Role: role, Rendered: rendered, Components: []PromptComponent{}}
	refs := func(name string, attrs map[string]string) []Reference {
		out := []Reference{}
		if sources != nil {
			for _, r := range sources(name, attrs) {
				if r.NodeID != "" {
					out = append(out, r)
				}
			}
		}
		return out
	}
	d := xml.NewDecoder(strings.NewReader(text))
	stack := []int{}
	for {
		start := int(d.InputOffset())
		token, err := d.Token()
		if err != nil {
			if err != io.EOF || len(stack) != 0 {
				p.Components = nil
			}
			break
		}
		switch v := token.(type) {
		case xml.StartElement:
			attrs := map[string]string{}
			for _, a := range v.Attr {
				attrs[a.Name.Local] = a.Value
			}
			component := PromptComponent{Name: v.Name.Local, StartByte: start, Sources: refs(v.Name.Local, attrs)}
			if len(stack) > 0 {
				parent := stack[len(stack)-1]
				component.Parent = &parent
			}
			stack = append(stack, len(p.Components))
			p.Components = append(p.Components, component)
		case xml.EndElement:
			index := stack[len(stack)-1]
			stack = stack[:len(stack)-1]
			p.Components[index].EndByte = int(d.InputOffset())
		}
	}
	if len(p.Components) == 0 {
		p.Components = []PromptComponent{{Name: name, StartByte: 0, EndByte: len(text), Sources: refs(name, nil)}}
	}
	return p
}

type InputBinding struct {
	Source Reference
	Path   string // relative to the provider's request snapshot
}
type inputBindingsKey struct{}

// WithInputBindings carries renderer provenance through WP-04 retries/failover.
// A modified repair request only gets copy links for values it actually retains.
func WithInputBindings(ctx context.Context, bindings ...InputBinding) context.Context {
	if !Enabled(ctx) {
		return ctx
	}
	return context.WithValue(ctx, inputBindingsKey{}, bindings)
}
func BindInputs(ctx context.Context) {
	if !Enabled(ctx) {
		return
	}
	bindings, _ := ctx.Value(inputBindingsKey{}).([]InputBinding)
	for _, b := range bindings {
		Bind(ctx, b.Source, Ref(ctx, "/input/request"+b.Path), "copy")
	}
}

func validateGraph(d Document) error {
	execution := map[string]int{}
	for _, e := range d.Edges {
		if e.Kind == "execution" {
			for _, n := range d.Nodes {
				if n.ID == e.To && n.ParentID != e.From {
					return errors.New("execution edge disagrees with parent")
				}
			}
			execution[e.To]++
		}
		if e.Kind == "data" {
			from, a := resolveRef(d, Reference{NodeID: e.From, Path: e.FromPath})
			to, b := resolveRef(d, Reference{NodeID: e.To, Path: e.ToPath})
			if !a || !b {
				return errors.New("unresolved data edge pointer")
			}
			if (e.Evidence == "exact_json_value_match" || e.Relation == "copy") && !reflect.DeepEqual(from, to) {
				return errors.New("copy edge values differ")
			}
		}
	}
	for _, n := range d.Nodes {
		if n.ParentID != "" && execution[n.ID] != 1 {
			return errors.New("missing or duplicate execution edge")
		}
		for _, a := range n.Artifacts {
			if a.Kind != "prompt_composition" {
				continue
			}
			var parts []PromptPart
			if json.Unmarshal(a.Value, &parts) != nil {
				return errors.New("invalid prompt composition")
			}
			for _, p := range parts {
				value, ok := resolveRef(d, p.Rendered)
				text, stringValue := value.(string)
				if !ok || !stringValue {
					return errors.New("unresolved rendered prompt")
				}
				for i, c := range p.Components {
					if c.StartByte < 0 || c.EndByte < c.StartByte || c.EndByte > len(text) || !utf8.ValidString(text[:c.StartByte]) || !utf8.ValidString(text[:c.EndByte]) {
						return errors.New("invalid prompt byte range")
					}
					if c.Parent != nil {
						if *c.Parent < 0 || *c.Parent >= i {
							return errors.New("invalid prompt parent")
						}
						parent := p.Components[*c.Parent]
						if c.StartByte < parent.StartByte || c.EndByte > parent.EndByte {
							return errors.New("prompt child escapes parent")
						}
					}
					for _, r := range c.Sources {
						if _, ok := resolveRef(d, r); !ok {
							return fmt.Errorf("unresolved prompt source %s:%s", r.NodeID, r.Path)
						}
					}
				}
			}
		}
	}
	return nil
}

// Credential omission can change string lengths. Re-index the exported text so
// ranges remain valid after redaction; source references never copy that text.
func refreshPromptRanges(d *Document) {
	for _, n := range d.Nodes {
		for i := range n.Artifacts {
			a := &n.Artifacts[i]
			if a.Kind != "prompt_composition" {
				continue
			}
			var parts []PromptPart
			if json.Unmarshal(a.Value, &parts) != nil {
				continue
			}
			for j := range parts {
				p := &parts[j]
				value, _ := resolveRef(*d, p.Rendered)
				text, ok := value.(string)
				if !ok {
					continue
				}
				fresh := Compose(p.Name, p.Role, p.Rendered, text, nil)
				for k := range fresh.Components {
					if k < len(p.Components) && fresh.Components[k].Name == p.Components[k].Name {
						fresh.Components[k].Sources = p.Components[k].Sources
					}
				}
				p.Components = fresh.Components
			}
			a.Value, _ = json.Marshal(parts)
		}
	}
}
