package database_executor

type DatabaseQueryRequest struct {
	Query string `json:"query" binding:"required"`
}
