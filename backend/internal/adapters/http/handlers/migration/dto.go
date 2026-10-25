package migration

import "encoding/json"

type CreateMigrationRequest struct {
	SqlUp string `json:"sql_up" binding:"required"`
}

type RollbackMigrationRequest struct {
	DatabaseID        uint `json:"database_id" binding:"required"`
	TargetMigrationID uint `json:"target_migration_id" binding:"required"`
}

type MigrationComparisonResponse struct {
	FirstMigration  MigrationResponse `json:"first_migration"`
	SecondMigration MigrationResponse `json:"second_migration"`
	SQLUpEqual      bool              `json:"sql_up_equal"`
	SQLDownEqual    bool              `json:"sql_down_equal"`
}

type MigrationResponse struct {
	ID           uint            `json:"id"`
	DatabaseID   uint            `json:"database_id"`
	Version      string          `json:"version"`
	Status       string          `json:"status"`
	SqlUp        string          `json:"sql_up"`
	SqlDown      string          `json:"sql_down"`
	ErrorMessage string          `json:"error_message"`
	Snapshot     json.RawMessage `json:"snapshot,omitempty"`
	UserID       uint            `json:"user_id"`
}
