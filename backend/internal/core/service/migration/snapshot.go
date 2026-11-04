package migration

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
)

type SchemaColumnSnapshot struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type SchemaIndexSnapshot struct {
	Name    string `json:"name"`
	Unique  bool   `json:"unique"`
	Primary bool   `json:"primary"`
}

type SchemaTableSnapshot struct {
	Schema  string                 `json:"schema,omitempty"`
	Columns []SchemaColumnSnapshot `json:"columns"`
	Indexes []SchemaIndexSnapshot  `json:"indexes,omitempty"`
}

type SchemaSnapshot struct {
	Tables map[string]SchemaTableSnapshot `json:"tables"`
}

func (s *SchemaSnapshot) ToJSON() ([]byte, error) {
	return json.Marshal(s)
}

func captureDatabaseSnapshot(
	ctx context.Context, inspector ports.SchemaInspectorRepository,
	database domain.Database,
) ([]byte, *SchemaSnapshot, error) {
	if inspector == nil {
		return nil, nil, nil
	}

	metadata, err := inspector.GetDatabaseMetadata(ctx, database)
	if err != nil || metadata == nil {
		return nil, nil, err
	}

	snapshot := buildSchemaSnapshot(metadata)
	rawBytes, err := snapshot.ToJSON()
	if err != nil {
		return nil, nil, err
	}

	return rawBytes, snapshot, nil
}

func buildSchemaSnapshot(metadata *ports.DatabaseMetadata) *SchemaSnapshot {
	tables := make(map[string]SchemaTableSnapshot)
	if metadata == nil {
		return &SchemaSnapshot{Tables: tables}
	}

	populateTables(tables, metadata.Tables)
	populateColumns(tables, metadata.Columns)
	populateIndexes(tables, metadata.Indexes)

	return &SchemaSnapshot{Tables: tables}
}

func populateTables(tables map[string]SchemaTableSnapshot, metaTables []ports.DatabaseTableMetadata) {
	for _, tbl := range metaTables {
		if tbl.Name == "" {
			continue
		}
		tables[tbl.Name] = ensureTableSnapshot(tables, tbl.Name, tbl.Schema)
	}
}

func populateColumns(tables map[string]SchemaTableSnapshot, metaColumns []ports.DatabaseColumnMetadata) {
	for _, col := range metaColumns {
		if col.Table == "" || col.Name == "" {
			continue
		}

		entry := ensureTableSnapshot(tables, col.Table, col.Schema)
		entry.Columns = append(entry.Columns, SchemaColumnSnapshot{
			Name:     col.Name,
			Type:     col.DataType,
			Nullable: col.IsNullable,
		})
		tables[col.Table] = entry
	}
}

func populateIndexes(tables map[string]SchemaTableSnapshot, metaIndexes []ports.DatabaseIndexMetadata) {
	for _, idx := range metaIndexes {
		if idx.Table == "" || idx.Name == "" {
			continue
		}

		entry := ensureTableSnapshot(tables, idx.Table, idx.Schema)
		if hasIndex(entry.Indexes, idx.Name) {
			continue
		}

		entry.Indexes = append(entry.Indexes, SchemaIndexSnapshot{
			Name:    idx.Name,
			Unique:  idx.IsUnique,
			Primary: idx.IsPrimary,
		})
		tables[idx.Table] = entry
	}
}

func ensureTableSnapshot(tables map[string]SchemaTableSnapshot, tableName, schema string) SchemaTableSnapshot {
	entry, exists := tables[tableName]
	if !exists {
		return SchemaTableSnapshot{
			Schema:  schema,
			Columns: make([]SchemaColumnSnapshot, 0),
			Indexes: make([]SchemaIndexSnapshot, 0),
		}
	}
	return entry
}

func hasIndex(indexes []SchemaIndexSnapshot, indexName string) bool {
	for _, idx := range indexes {
		if idx.Name == indexName {
			return true
		}
	}
	return false
}

func formatSnapshotAsBaselineSQL(snapshot *SchemaSnapshot) string {
	if snapshot == nil || len(snapshot.Tables) == 0 {
		return "-- Initial Schema Snapshot (Empty Database)"
	}

	sortedNames := getSortedTableNames(snapshot.Tables)
	tableStatements := make([]string, 0, len(sortedNames))

	for _, name := range sortedNames {
		tableStatements = append(tableStatements, formatCreateTableSQL(name, snapshot.Tables[name]))
	}

	var sb strings.Builder
	sb.WriteString("-- Initial Schema Snapshot (Migration #1)\n")
	sb.WriteString(strings.Join(tableStatements, "\n\n"))
	return sb.String()
}

func getSortedTableNames(tables map[string]SchemaTableSnapshot) []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func formatCreateTableSQL(tableName string, table SchemaTableSnapshot) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))

	columnLines := make([]string, 0, len(table.Columns))
	for _, col := range table.Columns {
		nullability := "NOT NULL"
		if col.Nullable {
			nullability = "NULL"
		}
		columnLines = append(columnLines, fmt.Sprintf("  %s %s %s", col.Name, col.Type, nullability))
	}

	sb.WriteString(strings.Join(columnLines, ",\n"))
	sb.WriteString("\n);")

	return sb.String()
}
