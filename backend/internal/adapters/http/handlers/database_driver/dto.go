package database_driver

type DatabaseQueryRequest struct {
	Query string `json:"query" binding:"required"`
}
