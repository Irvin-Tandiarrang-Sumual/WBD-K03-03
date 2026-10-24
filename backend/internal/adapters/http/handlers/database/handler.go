package database

import (
	"errors"
	"net/http"

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
	// TODO: replace with the authenticated user ID from the auth middleware.
	return 0, domain.ErrCurrentUserUnavailable
}

func toResponse(database *domain.Database) DatabaseResponse {
	return DatabaseResponse{
		ID: database.ID, Port: database.Port, Host: database.Host,
		User: database.User, Type: database.Type, Name: database.Name,
	}
}

func writeDatabaseErrorResponse(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	message := "internal server error"
	switch {
	case errors.Is(err, domain.ErrDatabaseNotFound):
		status = http.StatusNotFound
		message = err.Error()
	case errors.Is(err, domain.ErrInvalidDatabaseID),
		errors.Is(err, domain.ErrInvalidUserID),
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrInvalidDatabaseInput),
		errors.Is(err, domain.ErrInvalidInput):
		status = http.StatusBadRequest
		message = err.Error()
	case errors.Is(err, domain.ErrDatabaseAlreadyExists),
		errors.Is(err, domain.ErrDatabaseInUse):
		status = http.StatusConflict
		message = err.Error()
	case errors.Is(err, domain.ErrCurrentUserUnavailable):
		status = http.StatusNotImplemented
		message = err.Error()
	}
	c.JSON(status, response.Error(message))
}
