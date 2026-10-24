package migration

type CreateMigrationRequest struct {
	SqlUp string `json:"sql_up" binding:"required"`
}

type MigrationResponse struct {
	ID         uint   `json:"id"`
	DatabaseID uint   `json:"database_id"`
	Version    string `json:"version"`
	Status     string `json:"status"`
	SqlUp      string `json:"sql_up"`
	SqlDown    string `json:"sql_down"`
	UserID     uint   `json:"user_id"`
}
