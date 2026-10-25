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

func captureDatabaseSnapshot(
	ctx context.Context,
	executor ports.DatabaseExecutor,
	database domain.Database,
) ([]byte, *SchemaSnapshot, error) {
	if executor == nil {
		return nil, nil, nil
	}

	metadata, err := executor.GetDatabaseMetadata(ctx, database)
	if err != nil || metadata == nil {
		return nil, nil, err
	}

	snapshot := buildSchemaSnapshot(metadata)
	raw, err := json.Marshal(snapshot)
	if err != nil {
		return nil, nil, err
	}
	return raw, snapshot, nil
}

func buildSchemaSnapshot(metadata *ports.DatabaseMetadata) *SchemaSnapshot {
	tables := make(map[string]SchemaTableSnapshot)
	if metadata == nil {
		return &SchemaSnapshot{Tables: tables}
	}

	for _, tbl := range metadata.Tables {
		if tbl.Name == "" {
			continue
		}
		tables[tbl.Name] = SchemaTableSnapshot{
			Schema:  tbl.Schema,
			Columns: []SchemaColumnSnapshot{},
			Indexes: []SchemaIndexSnapshot{},
		}
	}

	for _, col := range metadata.Columns {
		if col.Table == "" || col.Name == "" {
			continue
		}
		entry, exists := tables[col.Table]
		if !exists {
			entry = SchemaTableSnapshot{
				Schema:  col.Schema,
				Columns: []SchemaColumnSnapshot{},
				Indexes: []SchemaIndexSnapshot{},
			}
		}
		entry.Columns = append(entry.Columns, SchemaColumnSnapshot{
			Name:     col.Name,
			Type:     col.DataType,
			Nullable: col.IsNullable,
		})
		tables[col.Table] = entry
	}

	for _, idx := range metadata.Indexes {
		if idx.Table == "" || idx.Name == "" {
			continue
		}
		entry, exists := tables[idx.Table]
		if !exists {
			entry = SchemaTableSnapshot{
				Schema:  idx.Schema,
				Columns: []SchemaColumnSnapshot{},
				Indexes: []SchemaIndexSnapshot{},
			}
		}
		// Avoid duplicate index names when multi-column index rows are returned
		alreadyAdded := false
		for _, existing := range entry.Indexes {
			if existing.Name == idx.Name {
				alreadyAdded = true
				break
			}
		}
		if !alreadyAdded {
			entry.Indexes = append(entry.Indexes, SchemaIndexSnapshot{
				Name:    idx.Name,
				Unique:  idx.IsUnique,
				Primary: idx.IsPrimary,
			})
		}
		tables[idx.Table] = entry
	}

	return &SchemaSnapshot{Tables: tables}
}

func formatSnapshotAsBaselineSQL(snapshot *SchemaSnapshot) string {
	if snapshot == nil || len(snapshot.Tables) == 0 {
		return "-- Initial Schema Snapshot (Empty Database)"
	}

	tableNames := make([]string, 0, len(snapshot.Tables))
	for name := range snapshot.Tables {
		tableNames = append(tableNames, name)
	}
	sort.Strings(tableNames)

	var sb strings.Builder
	sb.WriteString("-- Initial Schema Snapshot (Migration #1)\n")
	for i, tableName := range tableNames {
		tbl := snapshot.Tables[tableName]
		sb.WriteString(fmt.Sprintf("CREATE TABLE %s (\n", tableName))
		for cIdx, col := range tbl.Columns {
			nullStr := "NOT NULL"
			if col.Nullable {
				nullStr = "NULL"
			}
			comma := ","
			if cIdx == len(tbl.Columns)-1 {
				comma = ""
			}
			sb.WriteString(fmt.Sprintf("  %s %s %s%s\n", col.Name, col.Type, nullStr, comma))
		}
		sb.WriteString(");")
		if i < len(tableNames)-1 {
			sb.WriteString("\n\n")
		}
	}
	return sb.String()
}

