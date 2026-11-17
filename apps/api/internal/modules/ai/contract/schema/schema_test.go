package schema_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"
	"sync"
	"testing"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract/schema"
)

type fixture struct {
	Name    string          `json:"name"`
	Schema  string          `json:"schema"`
	Valid   bool            `json:"valid"`
	Payload json.RawMessage `json:"payload"`
	Changes []fixtureChange `json:"changes"`
}

type fixtureChange struct {
	Path   []string `json:"path"`
	Remove bool     `json:"remove"`
	Value  any      `json:"value"`
}

func vocabulary(t *testing.T) schema.Vocabulary {
	t.Helper()
	data, err := os.ReadFile("testdata/vocabulary.json")
	if err != nil {
		t.Fatal(err)
	}
	var vocab schema.Vocabulary
	if err := json.Unmarshal(data, &vocab); err != nil {
		t.Fatal(err)
	}
	return vocab
}
func load(t *testing.T) (*schema.Set, []fixture) {
	t.Helper()
	s, err := schema.New(vocabulary(t))
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []fixture
	if err := json.Unmarshal(data, &fixtures); err != nil {
		t.Fatal(err)
	}
	// Each case lists only its changes to a complete positive document, keeping
	// individual rejection cases reviewable without duplicating whole payloads.
	for i := range fixtures {
		f := &fixtures[i]
		base, err := os.ReadFile("testdata/" + f.Schema + ".json")
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(base, &doc); err != nil {
			t.Fatal(err)
		}
		for _, change := range f.Changes {
			target := doc
			for _, key := range change.Path[:len(change.Path)-1] {
				switch v := target.(type) {
				case map[string]any:
					target = v[key]
				case []any:
					index, err := strconv.Atoi(key)
					if err != nil {
						t.Fatal(err)
					}
					target = v[index]
				default:
					t.Fatalf("bad fixture path: %s", f.Name)
				}
			}
			key := change.Path[len(change.Path)-1]
			switch v := target.(type) {
			case map[string]any:
				if change.Remove {
					delete(v, key)
				} else {
					v[key] = change.Value
				}
			case []any:
				index, err := strconv.Atoi(key)
				if err != nil || change.Remove {
					t.Fatal("bad array fixture")
				}
				v[index] = change.Value
			default:
				t.Fatalf("bad fixture target: %s", f.Name)
			}
		}
		f.Payload, err = json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
	}
	return s, fixtures
}
func decode(s *schema.Set, name string, data []byte) (any, error) {
	switch name {
	case contract.BrainSchemaVersion:
		return s.DecodeBrain(data)
	case contract.MemoryWriterSchemaVersion:
		return s.DecodeMemoryWriter(data)
	case contract.ReflectionSchemaVersion:
		return s.DecodeReflection(data)
	case contract.SafetySchemaVersion:
		return s.DecodeSafety(data)
	default:
		return nil, schema.ErrConfiguration
	}
}

// Compile the resolved schema independently of the typed decoder, then test the
// same positive/negative documents at both boundaries. Accepted Go mirrors must
// round-trip back through the schema, including every optional field.
func TestSchemaAndGoMirrorFixtures(t *testing.T) {
	s, fixtures := load(t)
	independent := map[string]*jsonschema.Schema{}
	for _, name := range []string{contract.BrainSchemaVersion, contract.MemoryWriterSchemaVersion, contract.ReflectionSchemaVersion, contract.SafetySchemaVersion} {
		raw, err := s.JSON(name)
		if err != nil {
			t.Fatal(err)
		}
		var doc any
		if err := json.Unmarshal(raw, &doc); err != nil {
			t.Fatal(err)
		}
		compiler := jsonschema.NewCompiler()
		compiler.DefaultDraft(jsonschema.Draft2020)
		compiler.AssertFormat()

		if err := compiler.AddResource(name+".json", doc); err != nil {
			t.Fatal(err)
		}
		independent[name], err = compiler.Compile(name + ".json")
		if err != nil {
			t.Fatal(err)
		}
	}
	for _, f := range fixtures {
		t.Run(f.Name, func(t *testing.T) {
			var doc any
			if err := json.Unmarshal(f.Payload, &doc); err != nil {
				t.Fatal(err)
			}
			if accepted := independent[f.Schema].Validate(doc) == nil; accepted != f.Valid {
				t.Fatalf("schema accepted=%v; expected %v", accepted, f.Valid)
			}
			value, err := decode(s, f.Schema, f.Payload)
			if (err == nil) != f.Valid {
				t.Fatalf("Go decoder error=%v; expected valid=%v", err, f.Valid)
			}
			if !f.Valid {
				if !errors.Is(err, schema.ErrInvalid) || strings.Contains(err.Error(), "private_synthetic") {
					t.Fatalf("unsanitized error: %v", err)
				}
				return
			}
			roundTrip, err := json.Marshal(value)
			if err != nil {
				t.Fatal(err)
			}
			if err := s.Validate(f.Schema, roundTrip); err != nil {
				t.Fatalf("Go mirror no longer matches schema: %v", err)
			}
			if f.Name == "memwriter.v1_valid" {
				memory := value.(contract.MemoryWriterOutput)
				if memory.Ops[0].Importance == nil || *memory.Ops[0].Importance != 0 || !bytes.Contains(roundTrip, []byte(`"importance":0`)) {
					t.Fatal("explicit optional zero was lost")
				}
			}
		})
	}
}

func TestStrictJSON(t *testing.T) {
	s, _ := load(t)
	valid := `{"level":"low","categories":[],"confidence":0.5}`
	for name, data := range map[string]string{
		"trailing_object": valid + ` {}`, "trailing_garbage": valid + ` x`,
		"duplicate":         `{"level":"high","level":"none","categories":[],"confidence":1}`,
		"escaped_duplicate": `{"level":"high","\u006cevel":"none","categories":[],"confidence":1}`,
		"null":              `null`, "invalid_utf8": strings.Replace(valid, "low", "\xff", 1),
		"case_insensitive_go_key": strings.Replace(valid, "level", "Level", 1),
		"number_overflow":         strings.Replace(valid, "0.5", "1e999", 1),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := s.DecodeSafety([]byte(data)); !errors.Is(err, schema.ErrInvalid) {
				t.Fatalf("got %v", err)
			}
		})
	}
	_, fixtures := load(t)
	brain := fixtures[0].Payload
	brain = bytes.Replace(brain, []byte(`"allowed":false`), []byte(`"allowed":false,"allowed":true`), 1)
	if _, err := s.DecodeBrain(brain); !errors.Is(err, schema.ErrInvalid) {
		t.Fatal("nested duplicate accepted")
	}
}

func TestVocabularyIsolationAndEmptyProtocols(t *testing.T) {
	v := vocabulary(t)
	s, err := schema.New(v)
	if err != nil {
		t.Fatal(err)
	}
	before, _ := s.JSON(contract.BrainSchemaVersion)
	v.Skills[0] = "mutated"
	v.Frameworks["synthetic_framework"][0] = "mutated"
	copyJSON, _ := s.JSON(contract.BrainSchemaVersion)
	copyJSON[0] = '!'
	after, _ := s.JSON(contract.BrainSchemaVersion)
	if !bytes.Equal(before, after) || bytes.Contains(after, []byte("__INJECT:")) {
		t.Fatal("schema was mutable or unresolved")
	}
	v = vocabulary(t)
	v.Protocols = []string{}
	v.Frameworks = nil
	if _, err := schema.New(v); err != nil {
		t.Fatalf("empty optional catalogs: %v", err)
	}
	for _, mutate := range []func(*schema.Vocabulary){
		func(v *schema.Vocabulary) { v.Protocols = nil },
		func(v *schema.Vocabulary) { v.Skills = []string{} },
		func(v *schema.Vocabulary) { v.Skills = []string{"same", "same"} },
		func(v *schema.Vocabulary) { v.Emotions = []string{"__INJECT:bad__"} },
		func(v *schema.Vocabulary) { v.Frameworks = map[string][]string{"bad": {}} },
	} {
		v := vocabulary(t)
		mutate(&v)
		if _, err := schema.New(v); !errors.Is(err, schema.ErrConfiguration) {
			t.Fatalf("invalid vocabulary: %v", err)
		}
	}
}

func TestConcurrentValidation(t *testing.T) {
	s, fixtures := load(t)
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.DecodeBrain(fixtures[0].Payload); err != nil {
				t.Error(err)
			}
		}()
	}
	wg.Wait()
}

func TestSchemaIntegerSpellings(t *testing.T) {
	s, fixtures := load(t)
	for _, spelling := range []string{"0", "0.0", "0e0", "-0", "1", "1.0", "1e0"} {
		data := bytes.Replace(fixtures[0].Payload, []byte(`"max_count":0`), []byte(`"max_count":`+spelling), 1)
		if err := s.Validate(contract.BrainSchemaVersion, data); err != nil {
			t.Fatal(err)
		}
		if _, err := s.DecodeBrain(data); err != nil {
			t.Fatalf("schema-valid integer %s: %v", spelling, err)
		}
	}
}
