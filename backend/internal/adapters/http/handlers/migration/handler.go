package migration

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type MigrationHandler struct {
	service ports.MigrationService
}

func NewMigrationHandler(service ports.MigrationService) *MigrationHandler {
	return &MigrationHandler{service: service}
}

func (h *MigrationHandler) GetMigrationsByDatabaseID(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		writeErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}

	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeErrorResponse(c, domain.ErrInvalidPagination)
		return
	}

	migrations, totalData, err := h.service.GetMigrationsByDatabaseID(c.Request.Context(), databaseID, params)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}

	data := make([]MigrationResponse, 0, len(migrations))
	for _, migration := range migrations {
		data = append(data, toResponse(migration))
	}
	c.JSON(http.StatusOK, response.NewPagedResponse(http.StatusOK, "migrations retrieved successfully", data, params, totalData))
}

func (h *MigrationHandler) CreateMigration(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		writeErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}

	userID, err := currentUserID(c)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}

	var req CreateMigrationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}

	migration, err := h.service.CreateMigration(c.Request.Context(), databaseID, userID, req.SqlUp)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Success("migration created successfully", toResponse(migration)))
}

func (h *MigrationHandler) GetMigrationByIDAndDatabaseID(c *gin.Context) {
	migrationID, err := ginutil.ParseUintParam(c, "migration_id")
	if err != nil {
		writeErrorResponse(c, domain.ErrInvalidMigrationInput)
		return
	}

	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		writeErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}

	migration, _, err := h.service.GetMigrationByIDAndDatabaseID(c.Request.Context(), migrationID, databaseID)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("migration retrieved successfully", toResponse(migration)))
}

func currentUserID(c *gin.Context) (uint, error) {
	// TODO: pake dari auth middleware
	return 0, domain.ErrCurrentUserUnavailable
}

func toResponse(migration *domain.Migration) MigrationResponse {
	return MigrationResponse{
		ID: migration.ID, DatabaseID: migration.DatabaseID, Version: migration.Version,
		Status: migration.Status, SqlUp: migration.SqlUp, SqlDown: migration.SqlDown,
		UserID: migration.UserID,
	}
}

func writeErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrMigrationNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidMigrationInput),
		errors.Is(err, domain.ErrInvalidPagination):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusNotImplemented
		message = err.Error()
	}
	c.JSON(status, response.Error(message))
}
