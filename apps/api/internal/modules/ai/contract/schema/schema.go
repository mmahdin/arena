// Package schema validates model JSON at the AI Core boundary. It performs
// structural validation only: owner/evidence checks, policy repairs, consent,
// risk transitions, and formulation evidence thresholds belong to later WPs.
package schema

import (
	"bytes"
	"embed"
	"encoding/json"
	"errors"
	"io"
	"sort"
	"strings"
	"unicode/utf8"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

//go:embed *.json
var documents embed.FS

// Vocabulary is supplied by the pinned bundle. There is deliberately no default
// clinical vocabulary. Protocols may be empty; nil means missing configuration.
// Frameworks maps approved framework names to their permitted field names.
type Vocabulary struct {
	Emotions, Slots, Skills, Protocols, RationaleTags, AvoidTags, HypothesisTags []string
	Frameworks                                                                   map[string][]string
}

// Set is immutable after construction and safe for concurrent validation.
type Set struct {
	compiled map[string]*jsonschema.Schema
	json     map[string][]byte
}

var (
	ErrConfiguration = errors.New("AI schema configuration invalid")
	ErrInvalid       = errors.New("AI structured output invalid")
)

type noExternalResources struct{}

func (noExternalResources) Load(string) (any, error) { return nil, ErrConfiguration }

// New compiles draft 2020-12 schemas with asserted UUID/date formats and no
// network or filesystem resolution. No mutable references to vocab are kept.
func New(vocab Vocabulary) (*Set, error) {
	values := map[string][]string{
		"emotion_label": vocab.Emotions, "slot_name": vocab.Slots,
		"skill_id": vocab.Skills, "protocol_id": vocab.Protocols,
		"rationale_tag": vocab.RationaleTags, "avoid_tag": vocab.AvoidTags,
		"hypothesis_tag": vocab.HypothesisTags,
	}
	for key, list := range values {
		if list == nil || (len(list) == 0 && key != "protocol_id") || !validVocabulary(list) {
			return nil, ErrConfiguration
		}
	}
	frameworkNames := make([]string, 0, len(vocab.Frameworks))
	for name, fields := range vocab.Frameworks {
		if !validVocabulary([]string{name}) || len(fields) == 0 || !validVocabulary(fields) {
			return nil, ErrConfiguration
		}
		frameworkNames = append(frameworkNames, name)
	}
	sort.Strings(frameworkNames)
	set := &Set{compiled: make(map[string]*jsonschema.Schema), json: make(map[string][]byte)}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(noExternalResources{})

	names := []string{contract.BrainSchemaVersion, contract.MemoryWriterSchemaVersion, contract.ReflectionSchemaVersion, contract.SafetySchemaVersion}
	for _, name := range names {
		raw, err := documents.ReadFile(name + ".json")
		if err != nil {
			return nil, ErrConfiguration
		}
		var doc map[string]any
		if json.Unmarshal(raw, &doc) != nil {
			return nil, ErrConfiguration
		}
		if defs, ok := doc["$defs"].(map[string]any); ok {
			for key, list := range values {
				if _, exists := defs[key]; exists {
					defs[key] = vocabularySchema(list)
				}
			}
			if name == contract.ReflectionSchemaVersion {
				defs["framework"] = vocabularySchema(frameworkNames)
				form := defs["formulation"].(map[string]any)
				conditions := make([]any, 0, len(frameworkNames))
				for _, framework := range frameworkNames {
					conditions = append(conditions, map[string]any{
						"if":   map[string]any{"properties": map[string]any{"framework": map[string]any{"const": framework}}},
						"then": map[string]any{"properties": map[string]any{"fields": map[string]any{"propertyNames": vocabularySchema(vocab.Frameworks[framework])}}},
					})
				}
				if len(conditions) > 0 {
					form["allOf"] = conditions
				}
			}
		}
		raw, err = json.Marshal(doc)
		if err != nil || bytes.Contains(raw, []byte("__INJECT:")) {
			return nil, ErrConfiguration
		}
		url := "https://theia.app/schemas/" + name + ".json"
		if compiler.AddResource(url, doc) != nil {
			return nil, ErrConfiguration
		}
		compiled, err := compiler.Compile(url)
		if err != nil {
			return nil, ErrConfiguration
		}
		set.compiled[name], set.json[name] = compiled, raw
	}
	return set, nil
}

func validVocabulary(values []string) bool {
	seen := make(map[string]bool, len(values))
	for _, value := range values {
		if strings.TrimSpace(value) == "" || !utf8.ValidString(value) || strings.Contains(value, "__INJECT:") || seen[value] {
			return false
		}
		seen[value] = true
	}
	return true
}
func vocabularySchema(values []string) any {
	if len(values) == 0 {
		return false
	} // JSON Schema disallows an empty enum.
	copyValues := make([]any, len(values))
	for i, value := range values {
		copyValues[i] = value
	}
	return map[string]any{"enum": copyValues}
}

// JSON returns a copy of the resolved schema for provider structured output.
func (s *Set) JSON(name string) ([]byte, error) {
	if s == nil {
		return nil, ErrConfiguration
	}
	raw, ok := s.json[name]
	if !ok {
		return nil, ErrConfiguration
	}
	return append([]byte(nil), raw...), nil
}

// Validate rejects malformed UTF-8, duplicate keys, trailing values, missing
// required fields, unknown fields (including nested ones), nulls not allowed by
// the schema, invalid formats, and unknown catalog entries. Error messages never
// include model text, JSON keys, or schema validator diagnostics.
func (s *Set) Validate(name string, data []byte) error {
	if s == nil {
		return ErrConfiguration
	}
	compiled, ok := s.compiled[name]
	if !ok {
		return ErrConfiguration
	}
	if !utf8.Valid(data) || !uniqueJSON(data) {
		return ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	var value any
	if decoder.Decode(&value) != nil || compiled.Validate(value) != nil {
		return ErrInvalid
	}
	return nil
}

// Decode functions are the supported strict LLM ingress. encoding/json alone
// cannot enforce required fields or schemas and must not be used at that boundary.
func (s *Set) DecodeBrain(data []byte) (contract.BrainDecision, error) {
	return decode[contract.BrainDecision](s, contract.BrainSchemaVersion, data)
}
func (s *Set) DecodeMemoryWriter(data []byte) (contract.MemoryWriterOutput, error) {
	return decode[contract.MemoryWriterOutput](s, contract.MemoryWriterSchemaVersion, data)
}
func (s *Set) DecodeReflection(data []byte) (contract.ReflectionResult, error) {
	return decode[contract.ReflectionResult](s, contract.ReflectionSchemaVersion, data)
}
func (s *Set) DecodeSafety(data []byte) (contract.SafetyClassifierOutput, error) {
	return decode[contract.SafetyClassifierOutput](s, contract.SafetySchemaVersion, data)
}
func decode[T any](s *Set, name string, data []byte) (T, error) {
	var zero T
	if err := s.Validate(name, data); err != nil {
		return zero, err
	}
	// JSON Schema treats 1, 1.0 and 1e0 as the same integer. Normalize numeric
	// spelling after validation so Go int fields accept the same documents.
	// These LLM schemas contain only bounded counts/probabilities, not int64 IDs
	// or costs; those are internal contracts and never pass through this decoder.
	var document any
	if json.Unmarshal(data, &document) != nil {
		return zero, ErrInvalid
	}
	canonical, err := json.Marshal(document)
	if err != nil {
		return zero, ErrInvalid
	}
	decoder := json.NewDecoder(bytes.NewReader(canonical))
	decoder.DisallowUnknownFields()
	var value T
	if decoder.Decode(&value) != nil {
		return zero, ErrInvalid
	}
	return value, nil
}

func uniqueJSON(data []byte) bool {
	d := json.NewDecoder(bytes.NewReader(data))
	d.UseNumber()
	if readValue(d, 0) != nil {
		return false
	}
	_, err := d.Token()
	return err == io.EOF
}

func readValue(d *json.Decoder, depth int) error {
	if depth > 64 {
		return ErrInvalid
	}
	token, err := d.Token()
	if err != nil {
		return ErrInvalid
	}
	delim, ok := token.(json.Delim)
	if !ok {
		return nil
	}
	switch delim {
	case '{':
		keys := map[string]bool{}
		for d.More() {
			key, err := d.Token()
			if err != nil {
				return ErrInvalid
			}
			name, ok := key.(string)
			if !ok || keys[name] {
				return ErrInvalid
			}
			keys[name] = true
			if err := readValue(d, depth+1); err != nil {
				return err
			}
		}
	case '[':
		for d.More() {
			if err := readValue(d, depth+1); err != nil {
				return err
			}
		}
	default:
		return ErrInvalid
	}
	_, err = d.Token()
	return err
}

// CompileInline validates an already-injected, request-pinned JSON Schema without
// fetching external references. WP-04 uses it to validate EVERY structured model
// response, including responses from providers claiming native schema support.
// The returned function is safe for concurrent use. Failure details are always
// sanitized, since parser/validator diagnostics may quote private model output.
func CompileInline(raw []byte) (func([]byte) error, error) {
	if len(raw) == 0 || len(raw) > 512*1024 || !utf8.Valid(raw) || !uniqueJSON(raw) {
		return nil, ErrConfiguration
	}
	var document map[string]any
	if json.Unmarshal(raw, &document) != nil || len(document) == 0 {
		return nil, ErrConfiguration
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	compiler.UseLoader(noExternalResources{})
	const resource = "https://theia.app/schemas/llm-inline.json"
	if compiler.AddResource(resource, document) != nil {
		return nil, ErrConfiguration
	}
	compiled, err := compiler.Compile(resource)
	if err != nil {
		return nil, ErrConfiguration
	}
	return func(data []byte) error {
		if len(data) == 0 || len(data) > 1024*1024 || !utf8.Valid(data) || !uniqueJSON(data) {
			return ErrInvalid
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		var value any
		if decoder.Decode(&value) != nil || compiled.Validate(value) != nil {
			return ErrInvalid
		}
		return nil
	}, nil
}
