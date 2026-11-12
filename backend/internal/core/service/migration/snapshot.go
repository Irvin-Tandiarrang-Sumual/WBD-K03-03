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

type SchemaColumnSnapshot = domain.SchemaColumnSnapshot
type SchemaIndexSnapshot = domain.SchemaIndexSnapshot
type SchemaTableSnapshot = domain.SchemaTableSnapshot
type SchemaSnapshot = domain.SchemaSnapshot

type schemaSnapshotJSON struct {
	Tables map[string]schemaTableSnapshotJSON `json:"tables"`
}

type schemaTableSnapshotJSON struct {
	Schema  string                     `json:"schema"`
	Columns []schemaColumnSnapshotJSON `json:"columns"`
	Indexes []schemaIndexSnapshotJSON  `json:"indexes"`
}

type schemaColumnSnapshotJSON struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type schemaIndexSnapshotJSON struct {
	Name    string `json:"name"`
	Unique  bool   `json:"unique"`
	Primary bool   `json:"primary"`
}

func ToSnapshotJSON(s *domain.SchemaSnapshot) ([]byte, error) {
	if s == nil {
		return nil, nil
	}
	dto := schemaSnapshotJSON{
		Tables: make(map[string]schemaTableSnapshotJSON, len(s.Tables)),
	}
	for tblName, tbl := range s.Tables {
		cols := make([]schemaColumnSnapshotJSON, 0, len(tbl.Columns))
		for _, c := range tbl.Columns {
			cols = append(cols, schemaColumnSnapshotJSON{
				Name:     c.Name,
				Type:     c.Type,
				Nullable: c.Nullable,
			})
		}
		idxs := make([]schemaIndexSnapshotJSON, 0, len(tbl.Indexes))
		for _, idx := range tbl.Indexes {
			idxs = append(idxs, schemaIndexSnapshotJSON{
				Name:    idx.Name,
				Unique:  idx.Unique,
				Primary: idx.Primary,
			})
		}
		dto.Tables[tblName] = schemaTableSnapshotJSON{
			Schema:  tbl.Schema,
			Columns: cols,
			Indexes: idxs,
		}
	}
	return json.Marshal(dto)
}

func ParseSchemaSnapshot(raw []byte) *domain.SchemaSnapshot {
	if len(raw) == 0 {
		return nil
	}
	var dto schemaSnapshotJSON
	if err := json.Unmarshal(raw, &dto); err != nil {
		return nil
	}
	tables := make(map[string]domain.SchemaTableSnapshot, len(dto.Tables))
	for tblName, tbl := range dto.Tables {
		cols := make([]domain.SchemaColumnSnapshot, 0, len(tbl.Columns))
		for _, c := range tbl.Columns {
			cols = append(cols, domain.SchemaColumnSnapshot{
				Name:     c.Name,
				Type:     c.Type,
				Nullable: c.Nullable,
			})
		}
		idxs := make([]domain.SchemaIndexSnapshot, 0, len(tbl.Indexes))
		for _, idx := range tbl.Indexes {
			idxs = append(idxs, domain.SchemaIndexSnapshot{
				Name:    idx.Name,
				Unique:  idx.Unique,
				Primary: idx.Primary,
			})
		}
		tables[tblName] = domain.SchemaTableSnapshot{
			Schema:  tbl.Schema,
			Columns: cols,
			Indexes: idxs,
		}
	}
	return &domain.SchemaSnapshot{Tables: tables}
}

func CaptureDatabaseSnapshot(
	ctx context.Context, inspector ports.SchemaInspectorRepository,
	database domain.Database,
) ([]byte, *domain.SchemaSnapshot, error) {
	if inspector == nil {
		return nil, nil, nil
	}

	metadata, err := inspector.GetDatabaseMetadata(ctx, database)
	if err != nil || metadata == nil {
		return nil, nil, err
	}

	snapshot := buildSchemaSnapshot(metadata)
	rawBytes, err := ToSnapshotJSON(snapshot)
	if err != nil {
		return nil, nil, err
	}

	return rawBytes, snapshot, nil
}

func buildSchemaSnapshot(metadata *ports.DatabaseMetadata) *domain.SchemaSnapshot {
	tables := make(map[string]domain.SchemaTableSnapshot)
	if metadata == nil {
		return &domain.SchemaSnapshot{Tables: tables}
	}

	populateTables(tables, metadata.Tables)
	populateColumns(tables, metadata.Columns)
	populateIndexes(tables, metadata.Indexes)

	return &domain.SchemaSnapshot{Tables: tables}
}

func populateTables(tables map[string]domain.SchemaTableSnapshot, metaTables []ports.DatabaseTableMetadata) {
	for _, tbl := range metaTables {
		if tbl.Name == "" {
			continue
		}
		tables[tbl.Name] = ensureTableSnapshot(tables, tbl.Name, tbl.Schema)
	}
}

func populateColumns(tables map[string]domain.SchemaTableSnapshot, metaColumns []ports.DatabaseColumnMetadata) {
	for _, col := range metaColumns {
		if col.Table == "" || col.Name == "" {
			continue
		}

		entry := ensureTableSnapshot(tables, col.Table, col.Schema)
		entry.Columns = append(entry.Columns, domain.SchemaColumnSnapshot{
			Name:     col.Name,
			Type:     col.DataType,
			Nullable: col.IsNullable,
		})
		tables[col.Table] = entry
	}
}

func populateIndexes(tables map[string]domain.SchemaTableSnapshot, metaIndexes []ports.DatabaseIndexMetadata) {
	for _, idx := range metaIndexes {
		if idx.Table == "" || idx.Name == "" {
			continue
		}

		entry := ensureTableSnapshot(tables, idx.Table, idx.Schema)
		if hasIndex(entry.Indexes, idx.Name) {
			continue
		}

		entry.Indexes = append(entry.Indexes, domain.SchemaIndexSnapshot{
			Name:    idx.Name,
			Unique:  idx.IsUnique,
			Primary: idx.IsPrimary,
		})
		tables[idx.Table] = entry
	}
}

func ensureTableSnapshot(tables map[string]domain.SchemaTableSnapshot, tableName, schema string) domain.SchemaTableSnapshot {
	entry, exists := tables[tableName]
	if !exists {
		return domain.SchemaTableSnapshot{
			Schema:  schema,
			Columns: make([]domain.SchemaColumnSnapshot, 0),
			Indexes: make([]domain.SchemaIndexSnapshot, 0),
		}
	}
	return entry
}

func hasIndex(indexes []domain.SchemaIndexSnapshot, indexName string) bool {
	for _, idx := range indexes {
		if idx.Name == indexName {
			return true
		}
	}
	return false
}

func FormatSnapshotAsBaselineSQL(snapshot *domain.SchemaSnapshot) string {
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

func getSortedTableNames(tables map[string]domain.SchemaTableSnapshot) []string {
	names := make([]string, 0, len(tables))
	for name := range tables {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func formatCreateTableSQL(tableName string, table domain.SchemaTableSnapshot) string {
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

type diffCollector struct {
	summaryItems        []domain.SnapshotDiffSummaryItem
	tableDiffs          []domain.TableDiffChange
	candidateStatements []string
}

func newDiffCollector() *diffCollector {
	return &diffCollector{
		summaryItems:        make([]domain.SnapshotDiffSummaryItem, 0),
		tableDiffs:          make([]domain.TableDiffChange, 0),
		candidateStatements: make([]string, 0),
	}
}

func getTableMap(s *domain.SchemaSnapshot) map[string]domain.SchemaTableSnapshot {
	if s == nil || s.Tables == nil {
		return make(map[string]domain.SchemaTableSnapshot)
	}
	return s.Tables
}

func getAllSortedTableNames(left, right map[string]domain.SchemaTableSnapshot) []string {
	namesMap := make(map[string]struct{}, len(left)+len(right))
	for name := range left {
		namesMap[name] = struct{}{}
	}
	for name := range right {
		namesMap[name] = struct{}{}
	}

	names := make([]string, 0, len(namesMap))
	for name := range namesMap {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func DiffSchemaSnapshots(left, right *domain.SchemaSnapshot) *domain.SchemaDiffResult {
	leftTables := getTableMap(left)
	rightTables := getTableMap(right)
	allTableNames := getAllSortedTableNames(leftTables, rightTables)

	c := newDiffCollector()

	for _, tableName := range allTableNames {
		lTable, lExists := leftTables[tableName]
		rTable, rExists := rightTables[tableName]

		switch {
		case !lExists && rExists:
			c.handleTableAdded(tableName, rTable)
		case lExists && !rExists:
			c.handleTableRemoved(tableName, lTable)
		case lExists && rExists:
			c.diffExistingTable(tableName, lTable, rTable)
		}
	}

	return c.toResult()
}

func (c *diffCollector) handleTableAdded(tableName string, rTable domain.SchemaTableSnapshot) {
	c.tableDiffs = append(c.tableDiffs, domain.TableDiffChange{
		TableName:       tableName,
		Status:          "added",
		AddedColumns:    rTable.Columns,
		RemovedColumns:  make([]domain.SchemaColumnSnapshot, 0),
		ModifiedColumns: make([]domain.ColumnDiffChange, 0),
		AddedIndexes:    rTable.Indexes,
		RemovedIndexes:  make([]domain.SchemaIndexSnapshot, 0),
	})

	c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
		Type:       "table_added",
		TableName:  tableName,
		TargetName: tableName,
		Detail:     fmt.Sprintf("Tabel baru \"%s\" ditambahkan (%d kolom)", tableName, len(rTable.Columns)),
		BadgeText:  "+ TABEL BARU",
	})

	for _, col := range rTable.Columns {
		nullText := "NOT NULL"
		if col.Nullable {
			nullText = "NULL"
		}
		c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
			Type:       "column_added",
			TableName:  tableName,
			TargetName: col.Name,
			Detail:     fmt.Sprintf("Kolom baru di tabel %s: %s (%s, %s)", tableName, col.Name, col.Type, nullText),
			BadgeText:  "+ KOLOM BARU",
		})
	}

	c.candidateStatements = append(c.candidateStatements, formatCreateTableSQL(tableName, rTable))
}

func (c *diffCollector) handleTableRemoved(tableName string, lTable domain.SchemaTableSnapshot) {
	c.tableDiffs = append(c.tableDiffs, domain.TableDiffChange{
		TableName:       tableName,
		Status:          "removed",
		AddedColumns:    make([]domain.SchemaColumnSnapshot, 0),
		RemovedColumns:  lTable.Columns,
		ModifiedColumns: make([]domain.ColumnDiffChange, 0),
		AddedIndexes:    make([]domain.SchemaIndexSnapshot, 0),
		RemovedIndexes:  lTable.Indexes,
	})

	c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
		Type:       "table_removed",
		TableName:  tableName,
		TargetName: tableName,
		Detail:     fmt.Sprintf("Tabel \"%s\" dihapus (%d kolom)", tableName, len(lTable.Columns)),
		BadgeText:  "- TABEL DIHAPUS",
	})

	c.candidateStatements = append(c.candidateStatements, fmt.Sprintf("DROP TABLE %s;", tableName))
}

func (c *diffCollector) diffExistingTable(tableName string, lTable, rTable domain.SchemaTableSnapshot) {
	addedCols, removedCols, modifiedCols := c.diffColumns(tableName, lTable.Columns, rTable.Columns)
	addedIdxs, removedIdxs := c.diffIndexes(tableName, lTable.Indexes, rTable.Indexes)

	if len(addedCols) > 0 || len(removedCols) > 0 || len(modifiedCols) > 0 || len(addedIdxs) > 0 || len(removedIdxs) > 0 {
		c.tableDiffs = append(c.tableDiffs, domain.TableDiffChange{
			TableName:       tableName,
			Status:          "modified",
			AddedColumns:    addedCols,
			RemovedColumns:  removedCols,
			ModifiedColumns: modifiedCols,
			AddedIndexes:    addedIdxs,
			RemovedIndexes:  removedIdxs,
		})
	}
}

func (c *diffCollector) diffColumns(
	tableName string,
	leftCols, rightCols []domain.SchemaColumnSnapshot,
) ([]domain.SchemaColumnSnapshot, []domain.SchemaColumnSnapshot, []domain.ColumnDiffChange) {
	added := make([]domain.SchemaColumnSnapshot, 0)
	removed := make([]domain.SchemaColumnSnapshot, 0)
	modified := make([]domain.ColumnDiffChange, 0)

	leftMap := make(map[string]domain.SchemaColumnSnapshot, len(leftCols))
	for _, col := range leftCols {
		leftMap[strings.ToLower(col.Name)] = col
	}

	rightMap := make(map[string]domain.SchemaColumnSnapshot, len(rightCols))
	for _, col := range rightCols {
		rightMap[strings.ToLower(col.Name)] = col
	}

	for _, rCol := range rightCols {
		lCol, exists := leftMap[strings.ToLower(rCol.Name)]
		if !exists {
			added = append(added, rCol)
			nullText := "NOT NULL"
			if rCol.Nullable {
				nullText = "NULL"
			}
			c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
				Type:       "column_added",
				TableName:  tableName,
				TargetName: rCol.Name,
				Detail:     fmt.Sprintf("Kolom baru di tabel %s: %s (%s, %s)", tableName, rCol.Name, rCol.Type, nullText),
				BadgeText:  "+ KOLOM BARU",
			})
			c.candidateStatements = append(c.candidateStatements,
				fmt.Sprintf("ALTER TABLE %s ADD COLUMN %s %s %s;", tableName, rCol.Name, rCol.Type, nullText))
		} else if !strings.EqualFold(lCol.Type, rCol.Type) || lCol.Nullable != rCol.Nullable {
			b := lCol
			a := rCol
			modified = append(modified, domain.ColumnDiffChange{
				Before: &b,
				After:  &a,
			})
			lNull := "NOT NULL"
			if lCol.Nullable {
				lNull = "NULL"
			}
			rNull := "NOT NULL"
			if rCol.Nullable {
				rNull = "NULL"
			}
			c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
				Type:       "column_modified",
				TableName:  tableName,
				TargetName: rCol.Name,
				Detail:     fmt.Sprintf("Kolom diubah di tabel %s.%s: %s (%s) → %s (%s)", tableName, rCol.Name, lCol.Type, lNull, rCol.Type, rNull),
				BadgeText:  "~ KOLOM DIUBAH",
			})
		}
	}

	for _, lCol := range leftCols {
		if _, exists := rightMap[strings.ToLower(lCol.Name)]; !exists {
			removed = append(removed, lCol)
			c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
				Type:       "column_removed",
				TableName:  tableName,
				TargetName: lCol.Name,
				Detail:     fmt.Sprintf("Kolom dihapus di tabel %s: %s (%s)", tableName, lCol.Name, lCol.Type),
				BadgeText:  "- KOLOM DIHAPUS",
			})
			c.candidateStatements = append(c.candidateStatements,
				fmt.Sprintf("ALTER TABLE %s DROP COLUMN %s;", tableName, lCol.Name))
		}
	}

	return added, removed, modified
}

func (c *diffCollector) diffIndexes(
	tableName string,
	leftIdxs, rightIdxs []domain.SchemaIndexSnapshot,
) ([]domain.SchemaIndexSnapshot, []domain.SchemaIndexSnapshot) {
	added := make([]domain.SchemaIndexSnapshot, 0)
	removed := make([]domain.SchemaIndexSnapshot, 0)

	leftMap := make(map[string]domain.SchemaIndexSnapshot, len(leftIdxs))
	for _, idx := range leftIdxs {
		leftMap[strings.ToLower(idx.Name)] = idx
	}

	rightMap := make(map[string]domain.SchemaIndexSnapshot, len(rightIdxs))
	for _, idx := range rightIdxs {
		rightMap[strings.ToLower(idx.Name)] = idx
	}

	for _, rIdx := range rightIdxs {
		if _, exists := leftMap[strings.ToLower(rIdx.Name)]; !exists {
			added = append(added, rIdx)
			c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
				Type:       "index_added",
				TableName:  tableName,
				TargetName: rIdx.Name,
				Detail:     fmt.Sprintf("Index baru di tabel %s: %s", tableName, rIdx.Name),
				BadgeText:  "+ INDEX BARU",
			})
		}
	}

	for _, lIdx := range leftIdxs {
		if _, exists := rightMap[strings.ToLower(lIdx.Name)]; !exists {
			removed = append(removed, lIdx)
			c.summaryItems = append(c.summaryItems, domain.SnapshotDiffSummaryItem{
				Type:       "index_removed",
				TableName:  tableName,
				TargetName: lIdx.Name,
				Detail:     fmt.Sprintf("Index dihapus di tabel %s: %s", tableName, lIdx.Name),
				BadgeText:  "- INDEX DIHAPUS",
			})
		}
	}

	return added, removed
}

func (c *diffCollector) toResult() *domain.SchemaDiffResult {
	return &domain.SchemaDiffResult{
		IsIdentical:  len(c.summaryItems) == 0,
		SummaryItems: c.summaryItems,
		TableDiffs:   c.tableDiffs,
		CandidateSQL: strings.Join(c.candidateStatements, "\n\n"),
	}
}
