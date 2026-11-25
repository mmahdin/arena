package trace_test

import (
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/trace"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Every new exported runtime entrypoint in these pipeline modules must either
// carry context-scoped instrumentation or acquire a documented exemption here.
// This is a source/integration guard, not a manually maintained execution DAG.
func TestInspectorRuntimeBoundaryCoverage(t *testing.T) {
	dirs := []string{"application/unified", "application/knowledge", "application/retrieval", "application/assembler", "application/protocol", "safety", "validator"}
	exemptions := map[string]string{
		"RecentThrough":            "delegates to instrumented ConversationStore; no independent data transform",
		"Load":                     "startup loads immutable assets; actual runtime contents are captured at use",
		"New":                      "constructor; no conversation execution",
		"NewLLMClassifier":         "constructor; Scores/Classify are instrumented",
		"WithCallMeta":             "context metadata helper, no execution or IO",
		"PinBundle":                "context metadata helper, no execution or IO",
		"WithFinishReason":         "context metadata helper, no execution or IO",
		"WithProtocolRequirements": "context metadata helper, no execution or IO",
		"Shortlist":                "delegates immediately to instrumented ShortlistWithContext",
		"Resolve":                  "legacy entrypoint refuses execution; ResolveTurn is the implemented boundary",
		"ListApprovedResources":    "read-only display operation outside the turn pipeline",
	}
	for _, dir := range dirs {
		files, err := filepath.Glob(filepath.Join("..", dir, "*.go"))
		if err != nil {
			t.Fatal(err)
		}
		for _, path := range files {
			if strings.HasSuffix(path, "_test.go") {
				continue
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
			if err != nil {
				t.Fatal(err)
			}
			for _, decl := range file.Decls {
				fn, ok := decl.(*ast.FuncDecl)
				if !ok || !fn.Name.IsExported() || fn.Body == nil {
					continue
				}
				contextual := false
				for _, param := range fn.Type.Params.List {
					if typ, ok := param.Type.(*ast.SelectorExpr); ok && typ.Sel.Name == "Context" {
						contextual = true
					}
				}
				if !contextual {
					continue
				}
				if reason := exemptions[fn.Name.Name]; reason != "" {
					continue
				}
				instrumented := false
				ast.Inspect(fn.Body, func(n ast.Node) bool {
					call, ok := n.(*ast.CallExpr)
					if !ok {
						return true
					}
					sel, ok := call.Fun.(*ast.SelectorExpr)
					if !ok {
						return true
					}
					pkg, ok := sel.X.(*ast.Ident)
					if ok && pkg.Name == "trace" && (sel.Sel.Name == "Start" || sel.Sel.Name == "Call") {
						instrumented = true
					}
					return true
				})
				if !instrumented {
					t.Errorf("%s:%s needs an inspector boundary or an explicit exemption", path, fn.Name.Name)
				}
			}
		}
	}
}

func TestInspectorExamplesConformToContract(t *testing.T) {
	files, err := filepath.Glob(filepath.Join("..", "..", "..", "..", "..", "..", "docs", "ai-core", "inspector-examples", "*.json"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) < 4 {
		t.Fatal("missing generated inspector examples")
	}
	for _, path := range files {
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if err = trace.ValidateJSON(raw); err != nil {
			t.Errorf("%s: %v", path, err)
		}
	}
}
