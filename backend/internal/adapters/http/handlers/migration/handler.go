package migration

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
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

func (h *MigrationHandler) CompareMigrations(c *gin.Context) {
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
	firstMigrationID, err := strconv.ParseUint(c.Query("migration_id_1"), 10, 64)
	if err != nil || firstMigrationID == 0 {
		writeErrorResponse(c, domain.ErrInvalidMigrationComparison)
		return
	}
	secondMigrationID, err := strconv.ParseUint(c.Query("migration_id_2"), 10, 64)
	if err != nil || secondMigrationID == 0 {
		writeErrorResponse(c, domain.ErrInvalidMigrationComparison)
		return
	}
	comparison, err := h.service.CompareMigrations(
		c.Request.Context(),
		databaseID,
		userID,
		uint(firstMigrationID),
		uint(secondMigrationID),
	)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("migrations compared successfully", MigrationComparisonResponse{
		FirstMigration:  toResponse(comparison.FirstMigration),
		SecondMigration: toResponse(comparison.SecondMigration),
		SQLUpEqual:      comparison.SQLUpEqual,
		SQLDownEqual:    comparison.SQLDownEqual,
	}))
}

func (h *MigrationHandler) RollbackMigrations(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}
	var request RollbackMigrationRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}
	rolledBack, err := h.service.RollbackMigrations(
		c.Request.Context(),
		request.DatabaseID,
		userID,
		request.TargetMigrationID,
	)
	if err != nil {
		writeErrorResponse(c, err)
		return
	}
	data := make([]MigrationResponse, 0, len(rolledBack))
	for _, migration := range rolledBack {
		data = append(data, toResponse(migration))
	}
	c.JSON(http.StatusOK, response.Success("migrations rolled back successfully", data))
}

func currentUserID(c *gin.Context) (uint, error) {
	userID, exists := middleware.AuthenticatedUserID(c)
	if !exists {
		return 0, domain.ErrCurrentUserUnavailable
	}
	return userID, nil
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
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrInvalidMigrationComparison):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrMigrationRollbackUnavailable):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, domain.ErrMigrationVersionUnavailable):
		status = http.StatusConflict
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
