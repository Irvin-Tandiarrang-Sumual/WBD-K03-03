package database_driver

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
)

type DatabaseDriverHandler struct {
	databaseDriverService ports.DatabaseDriverService
}

func NewDatabaseDriverHandler(databaseDriverService ports.DatabaseDriverService) *DatabaseDriverHandler {
	return &DatabaseDriverHandler{databaseDriverService: databaseDriverService}
}

func (h *DatabaseDriverHandler) ExecuteDatabaseQuery(c *gin.Context) {
	databaseIDParam := c.Param("database_id")
	databaseID, err := strconv.ParseUint(databaseIDParam, 10, 64)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := currentDatabaseDriverUserID(c)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(domain.ErrInvalidInput.Error()))
		return
	}

	result, err := h.databaseDriverService.ExecuteDatabaseQuery(
		c.Request.Context(),
		uint(databaseID),
		userID,
		request.Query,
	)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query executed successfully", result))
}

func (h *DatabaseDriverHandler) ExplainDatabaseQuery(c *gin.Context) {
	databaseIDParam := c.Param("database_id")
	databaseID, err := strconv.ParseUint(databaseIDParam, 10, 64)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := currentDatabaseDriverUserID(c)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	var request DatabaseQueryRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(domain.ErrInvalidInput.Error()))
		return
	}

	analyze := strings.EqualFold(c.Query("analyze"), "true")

	result, err := h.databaseDriverService.ExplainDatabaseQuery(
		c.Request.Context(),
		uint(databaseID),
		userID,
		request.Query,
		analyze,
	)
	if err != nil {
		writeDatabaseDriverErrorResponse(c, err)
		return
	}

	c.JSON(http.StatusOK, response.Success("query execution plan generated successfully", result))
}

func currentDatabaseDriverUserID(c *gin.Context) (uint, error) {
	value, exists := c.Get("user_id")
	if !exists {
		return 0, domain.ErrCurrentUserUnavailable
	}
	userID, ok := value.(uint)
	if !ok || userID == 0 {
		return 0, domain.ErrCurrentUserUnavailable
	}
	return userID, nil
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
