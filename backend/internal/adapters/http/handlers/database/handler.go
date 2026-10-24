package database

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ports.DatabaseService
}

func NewHandler(service ports.DatabaseService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	databases, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	response := make([]DatabaseResponse, 0, len(databases))

	for _, database := range databases {
		response = append(response, toDatabaseResponse(database))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Get(c *gin.Context) {
	databaseID, err := parseID(c, "databaseId")
	if err != nil {
		writeError(c, err)
		return
	}

	database, err := h.service.Get(c.Request.Context(), databaseID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toDatabaseResponse(database))
}

func (h *Handler) Create(c *gin.Context) {
	var request CreateDatabaseRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	input := domain.CreateDatabaseInput{
		Port:     request.Port,
		Host:     request.Host,
		User:     request.User,
		Password: request.Password,
		Type:     request.Type,
		Name:     request.Name,
	}

	createdDatabase, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toDatabaseResponse(createdDatabase))
}

func (h *Handler) Delete(c *gin.Context) {
	databaseID, err := parseID(c, "databaseId")
	if err != nil {
		writeError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), databaseID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context, parameter string) (uint, error) {
	rawID := c.Param(parameter)

	id, err := strconv.ParseUint(rawID, 10, 64)
	if err != nil || id == 0 {
		return 0, domain.ErrInvalidInput
	}

	return uint(id), nil
}

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrDatabaseNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrDatabaseAlreadyExists),
		errors.Is(err, domain.ErrDatabaseInUse):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
	}
}