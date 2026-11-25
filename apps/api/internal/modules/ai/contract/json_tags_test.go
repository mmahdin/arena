package contract_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

func TestContractFieldsHaveExplicitJSONNames(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	namePattern := regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	for _, path := range files {
		if strings.HasSuffix(path, "_test.go") {
			continue
		}
		file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(node ast.Node) bool {
			s, ok := node.(*ast.StructType)
			if !ok {
				return true
			}
			for _, field := range s.Fields.List {
				for _, name := range field.Names {
					if !name.IsExported() {
						continue
					}
					if field.Tag == nil {
						t.Errorf("%s: %s lacks a JSON tag", path, name.Name)
						continue
					}
					literal, err := strconv.Unquote(field.Tag.Value)
					if err != nil {
						t.Error(err)
						continue
					}
					tag, ok := reflect.StructTag(literal).Lookup("json")
					key := strings.Split(tag, ",")[0]
					if !ok || (key != "-" && !namePattern.MatchString(key)) {
						t.Errorf("%s: %s has an invalid JSON name", path, name.Name)
					}
				}
			}
			return true
		})
	}
}
