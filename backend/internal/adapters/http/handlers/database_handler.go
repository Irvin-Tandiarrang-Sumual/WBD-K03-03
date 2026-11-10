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
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}

	database, err := h.databaseService.GetDatabaseByID(c.Request.Context(), databaseID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database retrieved successfully", toDatabaseResponse(database)))
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
		c.Error(domain.ErrInvalidPagination)
		return
	}

	databases, totalData, err := h.databaseService.GetAllDatabases(c.Request.Context(), params)
	if err != nil {
		c.Error(err)
		return
	}

	data := make([]DatabaseResponse, 0, len(databases))
	for _, database := range databases {
		data = append(data, toDatabaseResponse(database))
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

	databases, totalData, err := h.databaseService.GetDatabasesByUserID(c.Request.Context(), userID, params)
	if err != nil {
		c.Error(err)
		return
	}

	data := make([]DatabaseResponse, 0, len(databases))
	for _, database := range databases {
		data = append(data, toDatabaseResponse(database))
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
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}
	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	database, err := h.databaseService.GetDatabaseByIDAndUserID(c.Request.Context(), databaseID, userID)
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response.Success("database retrieved successfully", toDatabaseResponse(database)))
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
		c.Error(err)
		return
	}

	database, err := h.databaseService.CreateDatabase(c.Request.Context(), &ports.DatabaseInput{
		Port: uint(request.Port), Host: request.Host, User: request.User,
		Password: request.Password, Type: request.Type, Name: request.Name,
	})
	if err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusCreated, response.Success("database created successfully", toDatabaseResponse(database)))
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
		c.Error(domain.ErrInvalidDatabaseID)
		return
	}
	if err := h.databaseService.DeleteDatabase(c.Request.Context(), databaseID); err != nil {
		c.Error(err)
		return
	}
	c.JSON(http.StatusOK, response.Success[any]("database deleted successfully", nil))
}

func toDatabaseResponse(database *domain.Database) DatabaseResponse {
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
