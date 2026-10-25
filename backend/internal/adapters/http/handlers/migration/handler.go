package migration

import (
	"encoding/json"
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

// GetMigrationsByDatabaseID returns paginated migrations for a database.
//
//	@Summary		List migrations
//	@Description	Returns a paginated list of migrations for a specific database. Requires view access.
//	@Tags			Me - Migrations
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int										true	"Database ID"
//	@Param			page		query		int										false	"Page number (default: 1)"
//	@Param			limit		query		int										false	"Items per page (default: 10)"
//	@Success		200			{object}	response.PagedResponse[MigrationResponse]	"Migrations retrieved"
//	@Failure		400			{object}	response.BaseResponse[any]				"Invalid database ID or pagination"
//	@Failure		401			{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]				"Forbidden (no view access)"
//	@Failure		500			{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/me/databases/{database_id}/migrations [get]
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

// CreateMigration runs a new SQL migration on a database.
//
//	@Summary		Create migration
//	@Description	Runs a SQL migration (sql_up) on the specified database and records the result. Requires edit access.
//	@Tags			Me - Migrations
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int										true	"Database ID"
//	@Param			body		body		CreateMigrationRequest					true	"SQL up migration"
//	@Success		201			{object}	response.BaseResponse[MigrationResponse]	"Migration created"
//	@Failure		400			{object}	response.BaseResponse[any]				"Invalid database ID or migration input"
//	@Failure		401			{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]				"Forbidden (no edit access)"
//	@Failure		409			{object}	response.BaseResponse[any]				"Migration version conflict"
//	@Failure		500			{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/me/databases/{database_id}/migrations [post]
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
	message := "migration created successfully"
	if migration.Status == "fail" {
		message = "migration failed and was recorded"
	}
	c.JSON(http.StatusCreated, response.Success(message, toResponse(migration)))
}

// GetMigrationByIDAndDatabaseID returns a single migration by ID.
//
//	@Summary		Get migration
//	@Description	Returns a specific migration by its ID within a database. Requires view access.
//	@Tags			Me - Migrations
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id		path		int										true	"Database ID"
//	@Param			migration_id	path		int										true	"Migration ID"
//	@Success		200				{object}	response.BaseResponse[MigrationResponse]	"Migration retrieved"
//	@Failure		400				{object}	response.BaseResponse[any]				"Invalid database or migration ID"
//	@Failure		401				{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403				{object}	response.BaseResponse[any]				"Forbidden (no view access)"
//	@Failure		404				{object}	response.BaseResponse[any]				"Migration not found"
//	@Failure		500				{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/me/databases/{database_id}/migrations/{migration_id} [get]
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

// CompareMigrations compares two migrations within a database.
//
//	@Summary		Compare migrations
//	@Description	Compares the sql_up and sql_down of two migrations within a database. Requires view access.
//	@Tags			Me - Migrations
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id		path		int											true	"Database ID"
//	@Param			migration_id_1	query		int											true	"First migration ID"
//	@Param			migration_id_2	query		int											true	"Second migration ID"
//	@Success		200				{object}	response.BaseResponse[MigrationComparisonResponse]	"Comparison result"
//	@Failure		400				{object}	response.BaseResponse[any]					"Invalid database ID or migration IDs"
//	@Failure		401				{object}	response.BaseResponse[any]					"Unauthorized"
//	@Failure		403				{object}	response.BaseResponse[any]					"Forbidden (no view access)"
//	@Failure		404				{object}	response.BaseResponse[any]					"Migration not found"
//	@Failure		500				{object}	response.BaseResponse[any]					"Internal server error"
//	@Router			/me/databases/{database_id}/migrations/compare [get]
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

// RollbackMigrations rolls back migrations to a target migration.
//
//	@Summary		Rollback migrations
//	@Description	Rolls back all migrations up to and including the target migration ID.
//	@Tags			Me - Migrations
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			body	body		RollbackMigrationRequest						true	"Rollback target"
//	@Success		200		{object}	response.BaseResponse[[]MigrationResponse]		"Migrations rolled back"
//	@Failure		400		{object}	response.BaseResponse[any]						"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]						"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]						"Forbidden"
//	@Failure		409		{object}	response.BaseResponse[any]						"Rollback unavailable or version conflict"
//	@Failure		500		{object}	response.BaseResponse[any]						"Internal server error"
//	@Router			/me/databases/migrations/rollback [post]
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
	var snapshot json.RawMessage
	if len(migration.Snapshot) > 0 && json.Valid(migration.Snapshot) {
		snapshot = json.RawMessage(migration.Snapshot)
	}
	return MigrationResponse{
		ID: migration.ID, DatabaseID: migration.DatabaseID, Version: migration.Version,
		Status: migration.Status, SqlUp: migration.SqlUp, SqlDown: migration.SqlDown,
		ErrorMessage: migration.ErrorMessage,
		Snapshot:     snapshot,
		UserID:       migration.UserID,
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
