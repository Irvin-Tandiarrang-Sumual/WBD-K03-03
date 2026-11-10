package handlers

type DatabaseQueryRequest struct {
	Query string `json:"query" binding:"required"`
}
