package database_driver

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type DatabaseDriverHandler struct {
	databaseDriverService ports.DatabaseDriverService
}

func NewDatabaseDriverHandler(databaseDriverService ports.DatabaseDriverService) *DatabaseDriverHandler {
	return &DatabaseDriverHandler{databaseDriverService: databaseDriverService}
}

func (h *DatabaseDriverHandler) ExecuteDatabaseQuery(c *gin.Context) {
	databaseID, userID, query, ok := h.extractQueryContext(c)
	if !ok {
		return
	}

	result, err := h.databaseDriverService.ExecuteDatabaseQuery(
		c.Request.Context(),
		databaseID,
		userID,
		query,
	)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query executed successfully", result))
}

func (h *DatabaseDriverHandler) ExplainDatabaseQuery(c *gin.Context) {
	databaseID, userID, query, ok := h.extractQueryContext(c)
	if !ok {
		return
	}

	result, err := h.databaseDriverService.ExplainDatabaseQuery(
		c.Request.Context(),
		databaseID,
		userID,
		query,
	)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query execution plan generated successfully", result))
}

func (h *DatabaseDriverHandler) ExplainAnalyzeDatabaseQuery(c *gin.Context) {
	databaseID, userID, query, ok := h.extractQueryContext(c)
	if !ok {
		return
	}

	result, err := h.databaseDriverService.ExplainAnalyzeDatabaseQuery(
		c.Request.Context(),
		databaseID,
		userID,
		query,
	)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query execution plan with runtime analysis generated successfully", result))
}

func (h *DatabaseDriverHandler) extractQueryContext(c *gin.Context) (uint, uint, string, bool) {
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		writeDatabaseDriverErrorResponse(c, domain.ErrInvalidDatabaseID)
		return 0, 0, "", false
	}
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return 0, 0, "", false
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(domain.ErrInvalidInput.Error()))
		return 0, 0, "", false
	}

	return databaseID, userID, request.Query, true
}

func writeDatabaseDriverErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}

	switch {
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidDatabaseQuery),
		errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrDatabaseNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrDatabaseAccessDenied):
		status = http.StatusForbidden
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusUnauthorized
	}

	c.JSON(status, response.Error(message))
}
