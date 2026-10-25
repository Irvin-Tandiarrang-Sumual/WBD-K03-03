package postgresql

import (
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestMapTables(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{
		{"table_schema": "public", "table_name": "users"},
		{"table_schema": []byte("audit"), "table_name": "logs"},
	}}
	got := mapTables(result)
	want := []ports.DatabaseTableMetadata{{Schema: "public", Name: "users"}, {Schema: "[97 117 100 105 116]", Name: "logs"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestMapColumns(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{{
		"table_schema": "public",
		"table_name":   "users",
		"column_name":  "email",
		"data_type":    "text",
		"is_nullable":  true,
	}}}
	got := mapColumns(result)
	want := []ports.DatabaseColumnMetadata{{Schema: "public", Table: "users", Name: "email", DataType: "text", IsNullable: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestMapIndexes(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{{
		"schemaname": "public",
		"tablename":  "users",
		"indexname":  "users_pkey",
		"is_unique":  true,
		"is_primary": true,
	}}}
	got := mapIndexes(result)
	want := []ports.DatabaseIndexMetadata{{Schema: "public", Table: "users", Name: "users_pkey", IsUnique: true, IsPrimary: true}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got=%+v want=%+v", got, want)
	}
}

func TestValueHelpers(t *testing.T) {
	if got := stringValue(nil); got != "" {
		t.Fatalf("stringValue(nil)=%q", got)
	}
	if got := stringValue(42); got != "42" {
		t.Fatalf("stringValue(42)=%q", got)
	}
	if !boolValue(true) {
		t.Fatal("boolValue(true)=false")
	}
	if boolValue("true") || boolValue(nil) {
		t.Fatal("non-bool values must map to false")
	}
}
