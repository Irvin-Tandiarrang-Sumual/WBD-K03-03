package database

import (
	"errors"
	"net/http"
	"strings"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type DatabaseHandler struct {
	databaseService ports.DatabaseService
}

func NewDatabaseHandler(databaseService ports.DatabaseService) *DatabaseHandler {
	return &DatabaseHandler{databaseService: databaseService}
}

// GetDatabaseByID retrieves a database by its ID (admin only).
//
//	@Summary		Get database (admin)
//	@Description	Returns a database by its ID. Requires admin privileges.
//	@Tags			Admin - Databases
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		int											true	"Database ID"
//	@Success		200		{object}	response.BaseResponse[DatabaseResponse]		"Database retrieved"
//	@Failure		400		{object}	response.BaseResponse[any]					"Invalid database ID"
//	@Failure		401		{object}	response.BaseResponse[any]					"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]					"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]					"Database not found"
//	@Failure		500		{object}	response.BaseResponse[any]					"Internal server error"
//	@Router			/admin/databases/{id} [get]
func (h *DatabaseHandler) GetDatabaseByID(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		writeDatabaseErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}

	database, err := h.databaseService.GetDatabaseByID(c.Request.Context(), databaseID)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database retrieved successfully", toResponse(database)))
}

// GetAllDatabases retrieves a paginated list of all databases (admin only).
//
//	@Summary		List databases (admin)
//	@Description	Returns a paginated list of all databases. Requires admin privileges.
//	@Tags			Admin - Databases
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int												false	"Page number (default: 1)"
//	@Param			limit	query		int												false	"Items per page (default: 10)"
//	@Success		200		{object}	response.PagedResponse[DatabaseResponse]		"Databases retrieved"
//	@Failure		400		{object}	response.BaseResponse[any]						"Invalid pagination"
//	@Failure		401		{object}	response.BaseResponse[any]						"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]						"Forbidden (not admin)"
//	@Failure		500		{object}	response.BaseResponse[any]						"Internal server error"
//	@Router			/admin/databases [get]
func (h *DatabaseHandler) GetAllDatabases(c *gin.Context) {
	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeDatabaseErrorResponse(c, domain.ErrInvalidPagination)
		return
	}

	databases, totalData, err := h.databaseService.GetAllDatabases(c.Request.Context(), params)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}

	data := make([]DatabaseResponse, 0, len(databases))
	for _, database := range databases {
		data = append(data, toResponse(database))
	}
	c.JSON(http.StatusOK, response.NewPagedResponse(http.StatusOK, "databases retrieved successfully", data, params, totalData))
}

// GetDatabasesByUserID retrieves databases accessible by the current authenticated user.
//
//	@Summary		List my databases
//	@Description	Returns a paginated list of databases the authenticated user has access to.
//	@Tags			Me - Databases
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int												false	"Page number (default: 1)"
//	@Param			limit	query		int												false	"Items per page (default: 10)"
//	@Success		200		{object}	response.PagedResponse[DatabaseResponse]		"Databases retrieved"
//	@Failure		400		{object}	response.BaseResponse[any]						"Invalid pagination"
//	@Failure		401		{object}	response.BaseResponse[any]						"Unauthorized"
//	@Failure		500		{object}	response.BaseResponse[any]						"Internal server error"
//	@Router			/me/databases [get]
func (h *DatabaseHandler) GetDatabasesByUserID(c *gin.Context) {
	userID, err := currentUserID(c)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}

	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeDatabaseErrorResponse(c, domain.ErrInvalidPagination)
		return
	}

	databases, totalData, err := h.databaseService.GetDatabasesByUserID(c.Request.Context(), userID, params)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}

	data := make([]DatabaseResponse, 0, len(databases))
	for _, database := range databases {
		data = append(data, toResponse(database))
	}
	c.JSON(http.StatusOK, response.NewPagedResponse(http.StatusOK, "databases retrieved successfully", data, params, totalData))
}

// GetDatabaseByIDAndUserID retrieves a specific database accessible by the current user.
//
//	@Summary		Get my database
//	@Description	Returns a specific database that the authenticated user has view access to.
//	@Tags			Me - Databases
//	@Produce		json
//	@Security		CookieAuth
//	@Param			database_id	path		int										true	"Database ID"
//	@Success		200			{object}	response.BaseResponse[DatabaseResponse]	"Database retrieved"
//	@Failure		400			{object}	response.BaseResponse[any]				"Invalid database ID"
//	@Failure		401			{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403			{object}	response.BaseResponse[any]				"Forbidden (no view access)"
//	@Failure		404			{object}	response.BaseResponse[any]				"Database not found"
//	@Failure		500			{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/me/databases/{database_id} [get]
func (h *DatabaseHandler) GetDatabaseByIDAndUserID(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "database_id")
	if err != nil {
		writeDatabaseErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := currentUserID(c)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}

	database, err := h.databaseService.GetDatabaseByIDAndUserID(c.Request.Context(), databaseID, userID)
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database retrieved successfully", toResponse(database)))
}

// CreateDatabase registers a new database connection (admin only).
//
//	@Summary		Create database (admin)
//	@Description	Registers a new database connection. Requires admin privileges.
//	@Tags			Admin - Databases
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			body	body		CreateDatabaseRequest						true	"Database connection details"
//	@Success		201		{object}	response.BaseResponse[DatabaseResponse]		"Database created"
//	@Failure		400		{object}	response.BaseResponse[any]					"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]					"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]					"Forbidden (not admin)"
//	@Failure		409		{object}	response.BaseResponse[any]					"Database already exists"
//	@Failure		500		{object}	response.BaseResponse[any]					"Internal server error"
//	@Router			/admin/databases [post]
func (h *DatabaseHandler) CreateDatabase(c *gin.Context) {
	var request CreateDatabaseRequest
	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))
		return
	}

	database, err := h.databaseService.CreateDatabase(c.Request.Context(), &ports.DatabaseInput{
		Port: uint(request.Port), Host: request.Host, User: request.User,
		Password: request.Password, Type: request.Type, Name: request.Name,
	})
	if err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusCreated, response.Success("database created successfully", toResponse(database)))
}

// DeleteDatabase deletes a database by ID (admin only).
//
//	@Summary		Delete database (admin)
//	@Description	Deletes a registered database by its ID. Requires admin privileges.
//	@Tags			Admin - Databases
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		int						true	"Database ID"
//	@Success		200		{object}	response.BaseResponse[any]	"Database deleted"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid database ID"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Database not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Database is in use"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/databases/{id} [delete]
func (h *DatabaseHandler) DeleteDatabase(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		writeDatabaseErrorResponse(c, domain.ErrInvalidDatabaseID)
		return
	}
	if err := h.databaseService.DeleteDatabase(c.Request.Context(), databaseID); err != nil {
		writeDatabaseErrorResponse(c, err)
		return
	}
	c.JSON(http.StatusOK, response.Success[any]("database deleted successfully", nil))
}

func currentUserID(c *gin.Context) (uint, error) {
	userID, exists := middleware.AuthenticatedUserID(c)
	if !exists {
		return 0, domain.ErrCurrentUserUnavailable
	}
	return userID, nil
}

func toResponse(database *domain.Database) DatabaseResponse {
	return DatabaseResponse{
		ID:         database.ID,
		Port:       database.Port,
		Host:       database.Host,
		User:       database.User,
		Type:       database.Type,
		Name:       database.Name,
		AccessType: string(database.AccessType),
	}
}

func writeDatabaseErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	if err != nil && err.Error() != "" {
		message = err.Error()
	}
	switch {
	case errors.Is(err, domain.ErrDatabaseNotFound):
		status = http.StatusNotFound
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrInvalidDatabaseInput),
		errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
	case errors.Is(err, domain.ErrDatabaseAlreadyExists),
		errors.Is(err, domain.ErrDatabaseInUse):
		status = http.StatusConflict
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusNotImplemented
	default:
		if err != nil {
			errLower := strings.ToLower(err.Error())
			if strings.Contains(errLower, "connection refused") ||
				strings.Contains(errLower, "no such host") ||
				strings.Contains(errLower, "dial tcp") ||
				strings.Contains(errLower, "failed to connect") ||
				strings.Contains(errLower, "authentication failed") ||
				strings.Contains(errLower, "access denied") {
				status = http.StatusBadRequest
			}
		}
	}
	c.JSON(status, response.Error(message))
}
