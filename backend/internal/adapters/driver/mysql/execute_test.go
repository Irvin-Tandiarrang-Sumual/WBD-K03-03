package mysql

import (
	"reflect"
	"testing"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

func TestMapTables(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{
		{"table_schema": "shop", "table_name": "orders"},
		{"table_schema": nil, "table_name": "products"},
	}}

	got := mapTables(result)
	want := []ports.DatabaseTableMetadata{
		{Schema: "shop", Name: "orders"},
		{Schema: "", Name: "products"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mapTables() = %#v, want %#v", got, want)
	}
}

func TestMapColumns(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{
		{
			"table_schema": "shop",
			"table_name":   "orders",
			"column_name":  "id",
			"data_type":    "bigint",
			"is_nullable":  true,
		},
		{
			"table_schema": "shop",
			"table_name":   "orders",
			"column_name":  "note",
			"data_type":    "text",
			"is_nullable":  []byte("false"),
		},
	}}

	got := mapColumns(result)
	want := []ports.DatabaseColumnMetadata{
		{Schema: "shop", Table: "orders", Name: "id", DataType: "bigint", IsNullable: true},
		{Schema: "shop", Table: "orders", Name: "note", DataType: "text", IsNullable: false},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mapColumns() = %#v, want %#v", got, want)
	}
}

func TestMapIndexes(t *testing.T) {
	result := &ports.DatabaseQueryResult{Rows: []map[string]any{
		{
			"table_schema": "shop",
			"table_name":   "orders",
			"index_name":   "PRIMARY",
			"is_unique":    true,
			"is_primary":   []byte("true"),
		},
	}}

	got := mapIndexes(result)
	want := []ports.DatabaseIndexMetadata{
		{Schema: "shop", Table: "orders", Name: "PRIMARY", IsUnique: true, IsPrimary: true},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("mapIndexes() = %#v, want %#v", got, want)
	}
}

func TestValueHelpers(t *testing.T) {
	if got := stringValue(nil); got != "" {
		t.Fatalf("stringValue(nil) = %q, want empty", got)
	}
	if got := stringValue(42); got != "42" {
		t.Fatalf("stringValue(42) = %q, want 42", got)
	}

	cases := []struct {
		name  string
		value any
		want  bool
	}{
		{name: "bool true", value: true, want: true},
		{name: "bool false", value: false, want: false},
		{name: "byte true", value: []byte("true"), want: true},
		{name: "byte false", value: []byte("false"), want: false},
		{name: "invalid bytes", value: []byte("not-bool"), want: false},
		{name: "unsupported type", value: 1, want: false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := boolValue(tc.value); got != tc.want {
				t.Fatalf("boolValue(%v) = %v, want %v", tc.value, got, tc.want)
			}
		})
	}
}
