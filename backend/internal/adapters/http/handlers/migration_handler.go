package handlers

import (
	"encoding/json"
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
//
// @Param hide_rollback query bool false "Hide successful rollback pairs"
//
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
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}

	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	params, err := ginutil.ParsePagination(c)
	if err != nil {
		c.Error(domain.ErrInvalidPagination)
		return
	}

	hideRollback, err := strconv.ParseBool(c.DefaultQuery("hide_rollback", "false"))
	if err != nil {
		c.Error(domain.ErrInvalidMigrationInput)
		return
	}
	migrations, totalData, err := h.service.GetMigrationsByDatabaseID(c.Request.Context(), databaseID, userID, params, hideRollback)
	if err != nil {
		c.Error(err)
		return
	}

	data := make([]MigrationResponse, 0, len(migrations))
	for _, migration := range migrations {
		data = append(data, toMigrationResponse(migration))
	}
	c.JSON(http.StatusOK, response.NewPagedResponse(http.StatusOK, "migrations retrieved successfully", data, params, totalData))
}

// CreateMigration submits a SQL migration for approval.
//
//	@Summary		Submit migration draft
//	@Description	Submits a SQL migration as a draft for unanimous approval. Requires edit access.
//	@Tags			Me - Migrations
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int										true	"Database ID"
//	@Param			body		body		CreateDraftRequest					true	"SQL up migration"
//	@Success		201			{object}	response.BaseResponse[DraftResponse]	"Draft created"
//	@Failure		400			{object}	response.BaseResponse[any]				"Invalid database ID or migration input"
//	@Failure		401			{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]				"Forbidden (no edit access)"
//	@Failure		409			{object}	response.BaseResponse[any]				"Migration version conflict"
//	@Failure		500			{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/me/databases/{database_id}/migrations [post]
func (h *DraftHandler) CreateMigration(c *gin.Context) {
	h.CreateDraft(c)
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
		c.Error(domain.ErrInvalidMigrationInput)
		return
	}
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}
	migration, _, err := h.service.GetMigrationByIDAndDatabaseID(c.Request.Context(), migrationID, databaseID, userID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response.Success("migration retrieved successfully", toMigrationResponse(migration)))
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
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}
	firstMigrationID, err := strconv.ParseUint(c.Query("migration_id_1"), 10, 64)
	if err != nil || firstMigrationID == 0 {
		c.Error(domain.ErrInvalidMigrationComparison)
		return
	}
	secondMigrationID, err := strconv.ParseUint(c.Query("migration_id_2"), 10, 64)
	if err != nil || secondMigrationID == 0 {
		c.Error(domain.ErrInvalidMigrationComparison)
		return
	}
	comparison, err := h.service.CompareMigrations(
		c.Request.Context(),
		domain.CompareMigrationsInput{
			DatabaseID:        databaseID,
			UserID:            userID,
			FirstMigrationID:  uint(firstMigrationID),
			SecondMigrationID: uint(secondMigrationID),
		},
	)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response.Success("migrations compared successfully", MigrationComparisonResponse{
		FirstMigration:  toMigrationResponse(comparison.FirstMigration),
		SecondMigration: toMigrationResponse(comparison.SecondMigration),
		SQLUpEqual:      comparison.SQLUpEqual,
		SQLDownEqual:    comparison.SQLDownEqual,
		SchemaDiff:      toSchemaDiffResponse(comparison.SchemaDiff),
	}))
}

func toMigrationResponse(migration *domain.Migration) MigrationResponse {
	var snapshot json.RawMessage
	if len(migration.Snapshot) > 0 && json.Valid(migration.Snapshot) {
		snapshot = json.RawMessage(migration.Snapshot)
	}
	return MigrationResponse{
		ExecutionID:        migration.ExecutionID,
		RevertsMigrationID: migration.RevertsMigrationID,
		SourceMigrationID:  migration.SourceMigrationID,
		IsReversible:       migration.IsReversible,
		IrreversibleReason: migration.IrreversibleReason,

		ID:           migration.ID,
		DatabaseID:   migration.DatabaseID,
		Version:      migration.Version,
		Status:       migration.Status,
		SqlUp:        migration.SqlUp,
		SqlDown:      migration.SqlDown,
		ErrorMessage: migration.ErrorMessage,
		Snapshot:     snapshot,
		UserID:       migration.UserID,
		MigratedAt:   migration.MigratedAt,
	}
}
