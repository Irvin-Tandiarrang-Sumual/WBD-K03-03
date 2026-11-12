package handlers

import (
	"encoding/json"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
)

type CreateMigrationRequest struct {
	SqlUp string `json:"sqlUp" binding:"required"`
}

type RollbackMigrationRequest struct {
	DatabaseID        uint `json:"databaseId" binding:"required"`
	TargetMigrationID uint `json:"targetMigrationId" binding:"required"`
}

type SchemaColumnSnapshotResponse struct {
	Name     string `json:"name"`
	Type     string `json:"type"`
	Nullable bool   `json:"nullable"`
}

type SchemaIndexSnapshotResponse struct {
	Name    string `json:"name"`
	Unique  bool   `json:"unique"`
	Primary bool   `json:"primary"`
}

type ColumnDiffChangeResponse struct {
	Before *SchemaColumnSnapshotResponse `json:"before,omitempty"`
	After  *SchemaColumnSnapshotResponse `json:"after,omitempty"`
}

type TableDiffChangeResponse struct {
	TableName       string                         `json:"tableName"`
	Status          string                         `json:"status"`
	AddedColumns    []SchemaColumnSnapshotResponse `json:"addedColumns"`
	RemovedColumns  []SchemaColumnSnapshotResponse `json:"removedColumns"`
	ModifiedColumns []ColumnDiffChangeResponse     `json:"modifiedColumns"`
	AddedIndexes    []SchemaIndexSnapshotResponse  `json:"addedIndexes"`
	RemovedIndexes  []SchemaIndexSnapshotResponse  `json:"removedIndexes"`
}

type SnapshotDiffSummaryItemResponse struct {
	Type       string `json:"type"`
	TableName  string `json:"tableName"`
	TargetName string `json:"targetName"`
	Detail     string `json:"detail"`
	BadgeText  string `json:"badgeText"`
}

type SchemaDiffResultResponse struct {
	IsIdentical  bool                              `json:"isIdentical"`
	SummaryItems []SnapshotDiffSummaryItemResponse `json:"summaryItems"`
	TableDiffs   []TableDiffChangeResponse         `json:"tableDiffs"`
	CandidateSQL string                            `json:"candidateSql"`
}

type MigrationComparisonResponse struct {
	FirstMigration  MigrationResponse         `json:"firstMigration"`
	SecondMigration MigrationResponse         `json:"secondMigration"`
	SQLUpEqual      bool                      `json:"sqlUpEqual"`
	SQLDownEqual    bool                      `json:"sqlDownEqual"`
	SchemaDiff      *SchemaDiffResultResponse `json:"schemaDiff,omitempty"`
}

type MigrationResponse struct {
	ID           uint            `json:"id"`
	DatabaseID   uint            `json:"databaseId"`
	Version      string          `json:"version"`
	Status       string          `json:"status"`
	SqlUp        string          `json:"sqlUp"`
	SqlDown      string          `json:"sqlDown"`
	ErrorMessage string          `json:"errorMessage"`
	Snapshot     json.RawMessage `json:"snapshot,omitempty"`
	UserID       uint            `json:"userId"`
}

func toSchemaDiffResponse(diff *domain.SchemaDiffResult) *SchemaDiffResultResponse {
	if diff == nil {
		return nil
	}
	summary := make([]SnapshotDiffSummaryItemResponse, 0, len(diff.SummaryItems))
	for _, item := range diff.SummaryItems {
		summary = append(summary, SnapshotDiffSummaryItemResponse{
			Type:       item.Type,
			TableName:  item.TableName,
			TargetName: item.TargetName,
			Detail:     item.Detail,
			BadgeText:  item.BadgeText,
		})
	}

	tableDiffs := make([]TableDiffChangeResponse, 0, len(diff.TableDiffs))
	for _, td := range diff.TableDiffs {
		tableDiffs = append(tableDiffs, toTableDiffChangeResponse(td))
	}

	return &SchemaDiffResultResponse{
		IsIdentical:  diff.IsIdentical,
		SummaryItems: summary,
		TableDiffs:   tableDiffs,
		CandidateSQL: diff.CandidateSQL,
	}
}

func toTableDiffChangeResponse(td domain.TableDiffChange) TableDiffChangeResponse {
	addedCols := make([]SchemaColumnSnapshotResponse, 0, len(td.AddedColumns))
	for _, c := range td.AddedColumns {
		addedCols = append(addedCols, SchemaColumnSnapshotResponse{
			Name:     c.Name,
			Type:     c.Type,
			Nullable: c.Nullable,
		})
	}
	removedCols := make([]SchemaColumnSnapshotResponse, 0, len(td.RemovedColumns))
	for _, c := range td.RemovedColumns {
		removedCols = append(removedCols, SchemaColumnSnapshotResponse{
			Name:     c.Name,
			Type:     c.Type,
			Nullable: c.Nullable,
		})
	}
	modCols := make([]ColumnDiffChangeResponse, 0, len(td.ModifiedColumns))
	for _, mc := range td.ModifiedColumns {
		var before, after *SchemaColumnSnapshotResponse
		if mc.Before != nil {
			before = &SchemaColumnSnapshotResponse{
				Name:     mc.Before.Name,
				Type:     mc.Before.Type,
				Nullable: mc.Before.Nullable,
			}
		}
		if mc.After != nil {
			after = &SchemaColumnSnapshotResponse{
				Name:     mc.After.Name,
				Type:     mc.After.Type,
				Nullable: mc.After.Nullable,
			}
		}
		modCols = append(modCols, ColumnDiffChangeResponse{
			Before: before,
			After:  after,
		})
	}
	addedIdxs := make([]SchemaIndexSnapshotResponse, 0, len(td.AddedIndexes))
	for _, idx := range td.AddedIndexes {
		addedIdxs = append(addedIdxs, SchemaIndexSnapshotResponse{
			Name:    idx.Name,
			Unique:  idx.Unique,
			Primary: idx.Primary,
		})
	}
	removedIdxs := make([]SchemaIndexSnapshotResponse, 0, len(td.RemovedIndexes))
	for _, idx := range td.RemovedIndexes {
		removedIdxs = append(removedIdxs, SchemaIndexSnapshotResponse{
			Name:    idx.Name,
			Unique:  idx.Unique,
			Primary: idx.Primary,
		})
	}

	return TableDiffChangeResponse{
		TableName:       td.TableName,
		Status:          td.Status,
		AddedColumns:    addedCols,
		RemovedColumns:  removedCols,
		ModifiedColumns: modCols,
		AddedIndexes:    addedIdxs,
		RemovedIndexes:  removedIdxs,
	}
}
