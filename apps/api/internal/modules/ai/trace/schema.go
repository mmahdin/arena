package trace

import (
	"bytes"
	_ "embed"
	"encoding/json"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"sync"
)

//go:embed local_ai_trace.v1.schema.json
var JSONSchema []byte
var compiledSchema struct {
	once   sync.Once
	schema *jsonschema.Schema
	err    error
}

// ValidateJSON is intended for integration checks and offline consumers. The
// recorder's hot path uses typed data and never compiles schemas per event.
func ValidateJSON(raw []byte) error {
	compiledSchema.once.Do(func() {
		var definition any
		if compiledSchema.err = json.Unmarshal(JSONSchema, &definition); compiledSchema.err != nil {
			return
		}
		compiler := jsonschema.NewCompiler()
		if compiledSchema.err = compiler.AddResource("local_ai_trace.v1.schema.json", definition); compiledSchema.err != nil {
			return
		}
		compiledSchema.schema, compiledSchema.err = compiler.Compile("local_ai_trace.v1.schema.json")
	})
	if compiledSchema.err != nil {
		return compiledSchema.err
	}
	var value any
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&value); err != nil {
		return err
	}
	if err := compiledSchema.schema.Validate(value); err != nil {
		return err
	}
	var d Document
	if err := json.Unmarshal(raw, &d); err != nil {
		return err
	}
	return ValidateDocument(d)
}
