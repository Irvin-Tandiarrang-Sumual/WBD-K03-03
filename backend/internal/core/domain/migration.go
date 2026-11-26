package domain

import (
	"time"
)

const (
	MigrationStatusSnapshotPending = "snapshot_pending"
	MigrationStatusSuccess         = "success"
	MigrationStatusFail            = "fail"
)

type Migration struct {
	ParentMigrationID  *uint
	ForwardMigrationID *uint
	RevertsMigrationID *uint
	SourceMigrationID  *uint
	DraftID            *uint
	ExecutionID        uint
	IsReversible       bool
	IrreversibleReason string

	ID           uint
	DatabaseID   uint
	Version      string
	MigratedAt   time.Time
	Status       string
	SqlUp        string
	SqlDown      string
	ErrorMessage string
	Snapshot     []byte
	UserID       uint
}

type CreateMigrationInput struct {
	DatabaseID uint
	UserID     uint
	SqlUp      string
}

type SchemaColumnSnapshot struct {
	Name     string
	Type     string
	Nullable bool
}

type SchemaIndexSnapshot struct {
	Name    string
	Unique  bool
	Primary bool
}

type SchemaTableSnapshot struct {
	Schema  string
	Columns []SchemaColumnSnapshot
	Indexes []SchemaIndexSnapshot
}

type SchemaSnapshot struct {
	Tables map[string]SchemaTableSnapshot
}

type SnapshotDiffSummaryItem struct {
	Type       string
	TableName  string
	TargetName string
	Detail     string
	BadgeText  string
}

type ColumnDiffChange struct {
	Before *SchemaColumnSnapshot
	After  *SchemaColumnSnapshot
}

type TableDiffChange struct {
	TableName       string
	Status          string
	AddedColumns    []SchemaColumnSnapshot
	RemovedColumns  []SchemaColumnSnapshot
	ModifiedColumns []ColumnDiffChange
	AddedIndexes    []SchemaIndexSnapshot
	RemovedIndexes  []SchemaIndexSnapshot
}

type SchemaDiffResult struct {
	IsIdentical  bool
	SummaryItems []SnapshotDiffSummaryItem
	TableDiffs   []TableDiffChange
	CandidateSQL string
}

type MigrationComparison struct {
	FirstMigration  *Migration
	SecondMigration *Migration
	SQLUpEqual      bool
	SQLDownEqual    bool
	SchemaDiff      *SchemaDiffResult
}

func (d *CreateMigrationInput) Validate() error {
	if d.DatabaseID == 0 || d.UserID == 0 || d.SqlUp == "" {
		return ErrInvalidMigrationInput
	}
	return nil
}

type CompareMigrationsInput struct {
	DatabaseID        uint
	UserID            uint
	FirstMigrationID  uint
	SecondMigrationID uint
}

func (d *CompareMigrationsInput) Validate() error {
	if d.DatabaseID == 0 || d.UserID == 0 || d.FirstMigrationID == 0 || d.SecondMigrationID == 0 || d.FirstMigrationID == d.SecondMigrationID {
		return ErrInvalidMigrationComparison
	}
	return nil
}

type MigrationPosition struct {
	HeadID      *uint
	ExecutionID uint
}
