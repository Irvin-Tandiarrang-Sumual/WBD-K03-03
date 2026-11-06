package migration

import "encoding/json"

type CreateMigrationRequest struct {
	SqlUp string `json:"sqlUp" binding:"required"`
}

type RollbackMigrationRequest struct {
	DatabaseID        uint `json:"databaseId" binding:"required"`
	TargetMigrationID uint `json:"targetMigrationId" binding:"required"`
}

type MigrationComparisonResponse struct {
	FirstMigration  MigrationResponse `json:"firstMigration"`
	SecondMigration MigrationResponse `json:"secondMigration"`
	SQLUpEqual      bool              `json:"sqlUpEqual"`
	SQLDownEqual    bool              `json:"sqlDownEqual"`
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
