package mssql

import (
	"database/sql"
	"testing"

	c "github.com/wellbeing-companion/wellbeing-server/apps/api/internal/modules/ai/contract"
)

func TestWP08SQLReaderRejectsNilPool(t *testing.T) {
	if _, err := NewContextReader(nil); err == nil {
		t.Fatal("nil SQL pool accepted")
	}
}
func TestWP08NullableIDsAndJSONState(t *testing.T) {
	if id, err := nullableID(sql.NullString{}); err != nil || id != nil {
		t.Fatalf("null UUID: %v %v", id, err)
	}
	want := c.NewID()
	got, err := nullableID(sql.NullString{String: want.String(), Valid: true})
	if err != nil || got == nil || *got != want {
		t.Fatalf("valid UUID: %v %v", got, err)
	}
	if _, err = nullableID(sql.NullString{String: "not a uuid", Valid: true}); err == nil {
		t.Fatal("bad UUID accepted")
	}
	var xs []string
	if err = fromJSON(`["one"]`, &xs); err != nil || len(xs) != 1 || xs[0] != "one" {
		t.Fatalf("JSON state read: %+v %v", xs, err)
	}
	if err = fromJSON("", &xs); err == nil {
		t.Fatal("missing JSON state accepted")
	}
}
