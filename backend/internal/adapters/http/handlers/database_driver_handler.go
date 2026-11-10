package handlers

import (
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
		c.Error(err)
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
		c.Error(err)
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
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query execution plan with runtime analysis generated successfully", result))
}

func (h *DatabaseDriverHandler) extractQueryContext(c *gin.Context) (uint, uint, string, bool) {
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		c.Error(domain.ErrInvalidDatabaseID)
		return 0, 0, "", false
	}
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return 0, 0, "", false
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.Error(err)
		return 0, 0, "", false
	}

	return databaseID, userID, request.Query, true
}
