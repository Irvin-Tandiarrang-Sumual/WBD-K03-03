package database_executor

import (
	"errors"
	"net/http"
	"strconv"

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
	// TODO: replace with the authenticated user ID from the auth middleware.
	return 0, domain.ErrCurrentUserUnavailable
}

func writeDatabaseExecutorErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidDatabaseQuery):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrDatabaseNotFound),
		errors.Is(err, domain.ErrDatabaseAccessDenied):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusNotImplemented
		message = err.Error()
	}
	c.JSON(status, response.Error(message))
}
