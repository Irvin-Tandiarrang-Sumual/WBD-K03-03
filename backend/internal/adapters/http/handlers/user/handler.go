package user

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type UserHandler struct {
	service ports.UserService
}

func NewUserHandler(service ports.UserService) *UserHandler {
	return &UserHandler{service: service}
}

func (h *UserHandler) List(c *gin.Context) {
	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeError(c, domain.ErrInvalidPagination)
		return
	}

	users, totalData, err := h.service.List(c.Request.Context(), params)
	if err != nil {
		writeError(c, err)
		return
	}

	data := make([]UserResponse, 0, len(users))

	for _, user := range users {
		data = append(data, toUserResponse(user))
	}

	c.JSON(http.StatusOK, response.NewPagedResponse(
		http.StatusOK,
		"users retrieved successfully",
		data,
		params,
		totalData,
	))
}

func (h *UserHandler) Get(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
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

func (h *UserHandler) Create(c *gin.Context) {
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

func (h *UserHandler) Update(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
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

func (h *UserHandler) Delete(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
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

func (h *UserHandler) SetRole(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
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

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidRole),
		errors.Is(err, domain.ErrInvalidGroupsIDs):
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))

	case errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrGroupNotFound):
		c.JSON(http.StatusNotFound, response.Error(err.Error()))

	case errors.Is(err, domain.ErrUsernameTaken),
		errors.Is(err, domain.ErrUserMustHaveGroup):
		c.JSON(http.StatusConflict, response.Error(err.Error()))

	default:
		c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}
}
