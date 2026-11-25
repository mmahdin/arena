package ports_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/ports"
)

func TestPortEnums(t *testing.T) {
	for _, v := range []ports.ModelTier{ports.TierSmall, ports.TierStrong, ports.TierClassifier, ports.TierEmbedding} {
		if !v.Valid() {
			t.Fatal(v)
		}
	}
	for _, v := range []ports.TurnOutcome{ports.OutcomeCompleted, ports.OutcomeFailed, ports.OutcomeCancelled, ports.OutcomeQueued} {
		if !v.Valid() {
			t.Fatal(v)
		}
	}
	for _, v := range []ports.CorrectionKind{ports.CorrectionContest, ports.CorrectionReject, ports.CorrectionSupersedePending} {
		if !v.Valid() {
			t.Fatal(v)
		}
	}
	for _, value := range []string{"", "unknown", "COMPLETED"} {
		if ports.ModelTier(value).Valid() || ports.TurnOutcome(value).Valid() || ports.CorrectionKind(value).Valid() {
			t.Fatal("unknown enum accepted")
		}
	}
}

// New contract packages must remain independent of the prototype application,
// concrete providers, Gin, SQL and other modules. This guard reads actual Go
// imports rather than matching comments or documentation.
func TestContractDependencyBoundary(t *testing.T) {
	const module = "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/"
	for _, root := range []string{".", "../contract"} {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			file, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range file.Imports {
				name, err := strconv.Unquote(imp.Path.Value)
				if err != nil {
					return err
				}
				if name == "database/sql" || name == "net/http" {
					t.Errorf("infrastructure import in %s: %s", path, name)
				}
				if !strings.Contains(strings.Split(name, "/")[0], ".") {
					continue
				}
				allowed := name == "github.com/google/uuid" || name == module+"contract" || name == module+"ports"
				if strings.Contains(filepath.ToSlash(path), "contract/schema/") && name == "github.com/santhosh-tekuri/jsonschema/v6" {
					allowed = true
				}
				if !allowed {
					t.Errorf("unexpected dependency in %s: %s", path, name)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
}
