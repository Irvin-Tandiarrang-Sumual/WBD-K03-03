package database_executor

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type DatabaseExecutorHandler struct {
	databaseExecutorService ports.DatabaseExecutorService
}

func NewDatabaseExecutorHandler(databaseExecutorService ports.DatabaseExecutorService) *DatabaseExecutorHandler {
	return &DatabaseExecutorHandler{databaseExecutorService: databaseExecutorService}
}

// ExecuteDatabaseQuery executes a SQL query on a database.
//
//	@Summary		Execute query
//	@Description	Executes a raw SQL query on the specified database. Requires view access.
//	@Tags			Me - Databases
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int									true	"Database ID"
//	@Param			body		body		DatabaseQueryRequest				true	"SQL query"
//	@Success		200			{object}	response.BaseResponse[any]			"Query executed successfully"
//	@Failure		400			{object}	response.BaseResponse[any]			"Invalid database ID or query"
//	@Failure		401			{object}	response.BaseResponse[any]			"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]			"Forbidden (no view access)"
//	@Failure		404			{object}	response.BaseResponse[any]			"Database not found"
//	@Failure		500			{object}	response.BaseResponse[any]			"Internal server error"
//	@Router			/me/databases/{database_id}/query [post]
func (h *DatabaseExecutorHandler) ExecuteDatabaseQuery(c *gin.Context) {
	databaseID, err := strconv.ParseUint(c.Param("database_id"), 10, 64)
	if err != nil || databaseID == 0 {
		writeDatabaseExecutorErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := currentDatabaseExecutorUserID(c)
	if err != nil {
		writeDatabaseExecutorErrorResponse(c, err)
		return
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}

	result, err := h.databaseExecutorService.ExecuteDatabaseQuery(
		c.Request.Context(),
		uint(databaseID),
		userID,
		request.Query,
	)
	if err != nil {
		writeDatabaseExecutorErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database query executed successfully", result))
}

// ExplainDatabaseQuery explains a SQL query execution plan.
//
//	@Summary		Explain query
//	@Description	Returns the execution plan for a SQL query. Requires view access. Use `analyzer=true` for EXPLAIN ANALYZE.
//	@Tags			Me - Databases
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int								true	"Database ID"
//	@Param			body		body		DatabaseQueryRequest			true	"SQL query"
//	@Param			analyzer	query		bool							false	"Use EXPLAIN ANALYZE (default: false)"
//	@Success		200			{object}	response.BaseResponse[any]		"Query plan retrieved"
//	@Failure		400			{object}	response.BaseResponse[any]		"Invalid database ID, query, or analyzer param"
//	@Failure		401			{object}	response.BaseResponse[any]		"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]		"Forbidden (no view access)"
//	@Failure		404			{object}	response.BaseResponse[any]		"Database not found"
//	@Failure		500			{object}	response.BaseResponse[any]		"Internal server error"
//	@Router			/me/databases/{database_id}/explain [post]
func (h *DatabaseExecutorHandler) ExplainDatabaseQuery(c *gin.Context) {
	databaseID, err := strconv.ParseUint(c.Param("database_id"), 10, 64)
	if err != nil || databaseID == 0 {
		writeDatabaseExecutorErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := currentDatabaseExecutorUserID(c)
	if err != nil {
		writeDatabaseExecutorErrorResponse(c, err)
		return
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	analyze, err := strconv.ParseBool(c.DefaultQuery("analyzer", "false"))
	if err != nil {
		c.JSON(http.StatusBadRequest, response.Error("analyzer must be a boolean"))
		return
	}

	result, err := h.databaseExecutorService.ExplainDatabaseQuery(
		c.Request.Context(),
		uint(databaseID),
		userID,
		request.Query,
		analyze,
	)
	if err != nil {
		writeDatabaseExecutorErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database query explained successfully", result))
}

func currentDatabaseExecutorUserID(c *gin.Context) (uint, error) {
	userID, exists := middleware.AuthenticatedUserID(c)
	if !exists {
		return 0, domain.ErrCurrentUserUnavailable
	}
	return userID, nil
}

func writeDatabaseExecutorErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidDatabaseQuery):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrDatabaseNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, domain.ErrDatabaseAccessDenied):
		status = http.StatusForbidden
		message = err.Error()
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusNotImplemented
		message = err.Error()
	}
	c.JSON(status, response.Error(message))
}
