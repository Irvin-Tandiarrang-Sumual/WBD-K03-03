package user

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ports.UserService
}

func NewHandler(service ports.UserService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	users, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	response := make([]UserResponse, 0, len(users))

	for _, user := range users {
		response = append(response, toUserResponse(user))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Get(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		writeError(c, err)
		return
	}

	user, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

func (h *Handler) Create(c *gin.Context) {
	var request CreateUserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	input := domain.CreateUserInput{
		Username: request.Username,
		Password: request.Password,
		GroupIDs: request.GroupIDs,
	}

	createdUser, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toUserResponse(createdUser))
}

func (h *Handler) Update(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		writeError(c, err)
		return
	}

	var request UpdateUserRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	input := domain.UpdateUserInput{
		Username: request.Username,
		Password: request.Password,
	}

	updatedUser, err := h.service.Update(c.Request.Context(), userID, input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(updatedUser))
}

func (h *Handler) Delete(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		writeError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *Handler) SetRole(c *gin.Context) {
	userID, err := parseID(c, "userId")
	if err != nil {
		writeError(c, err)
		return
	}

	var request SetRoleRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	updatedUser, err := h.service.SetRole(c.Request.Context(), userID, request.Role)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(updatedUser))
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
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidGroupsIDs):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrGroupNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrUserMustHaveGroup):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
	}
}