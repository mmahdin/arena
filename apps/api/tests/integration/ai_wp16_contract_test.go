package integration

import (
	"encoding/json"
	"errors"
	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"
	"os"
	"strings"
	"sync"
	"testing"
)

var wp16Schemas struct {
	sync.Once
	values map[string]*jsonschema.Schema
	err    error
}

type wp16NoExternal struct{}

func (wp16NoExternal) Load(string) (any, error) {
	return nil, errors.New("external OpenAPI reference forbidden")
}
func wp16Compiled() (map[string]*jsonschema.Schema, error) {
	wp16Schemas.Do(func() {
		raw, e := os.ReadFile("../../../../contracts/openapi/openapi.yaml")
		if e != nil {
			wp16Schemas.err = e
			return
		}
		var root map[string]any
		if e = yaml.Unmarshal(raw, &root); e != nil {
			wp16Schemas.err = e
			return
		}
		encoded, e := json.Marshal(root)
		if e != nil {
			wp16Schemas.err = e
			return
		}
		var doc any
		_ = json.Unmarshal(encoded, &doc)
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.AssertFormat()
		compiler.UseLoader(wp16NoExternal{})
		const uri = "https://synthetic.theia.invalid/openapi.json"
		if e = compiler.AddResource(uri, doc); e != nil {
			wp16Schemas.err = e
			return
		}
		wp16Schemas.values = map[string]*jsonschema.Schema{}
		for name := range root["components"].(map[string]any)["schemas"].(map[string]any) {
			s, e := compiler.Compile(uri + "#/components/schemas/" + name)
			if e != nil {
				wp16Schemas.err = e
				return
			}
			wp16Schemas.values[name] = s
		}
	})
	return wp16Schemas.values, wp16Schemas.err
}
func TestWP16OpenAPIResponseSchemasCompile(t *testing.T) {
	if _, e := wp16Compiled(); e != nil {
		t.Fatal(e)
	}
}
func wp16AssertResponse(t *testing.T, method, route string, status int, data map[string]any) {
	t.Helper()
	if status == 204 {
		return
	}
	schema := ""
	switch {
	case status >= 400:
		schema = "Problem"
	case strings.Contains(route, "/turns") || route == "/api/v1/me/ai/recover":
		schema = "TurnResult"
	case strings.HasPrefix(route, "/api/v1/me/ai/controls"):
		schema = "AIControls"
	case strings.HasPrefix(route, "/api/v1/me/ai/export"):
		schema = "AIExport"
	case strings.HasPrefix(route, "/api/v1/me/memories"):
		if method == "GET" {
			schema = "MemoryPage"
		} else if method == "PATCH" {
			schema = "MemoryItem"
		}
	}
	if schema == "" {
		return
	}
	schemas, e := wp16Compiled()
	if e != nil {
		t.Fatal(e)
	}
	if e = schemas[schema].Validate(data); e != nil {
		t.Fatalf("%s %s HTTP %d violates authored %s response schema", method, route, status, schema)
	}
}
