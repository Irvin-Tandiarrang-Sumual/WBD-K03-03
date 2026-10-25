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

// List returns a paginated list of users.
//
//	@Summary		List users
//	@Description	Returns a paginated list of all users. Requires admin privileges.
//	@Tags			Admin - Users
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int									false	"Page number (default: 1)"
//	@Param			limit	query		int									false	"Items per page (default: 10)"
//	@Success		200		{object}	response.PagedResponse[UserResponse]	"Users retrieved"
//	@Failure		400		{object}	response.BaseResponse[any]				"Invalid pagination"
//	@Failure		401		{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]				"Forbidden (not admin)"
//	@Failure		500		{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/admin/users [get]
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

// Get returns a single user by ID.
//
//	@Summary		Get user
//	@Description	Returns a single user by their ID. Requires admin privileges.
//	@Tags			Admin - Users
//	@Produce		json
//	@Security		CookieAuth
//	@Param			userId	path		int								true	"User ID"
//	@Success		200		{object}	UserResponse					"User found"
//	@Failure		400		{object}	response.BaseResponse[any]		"Invalid user ID"
//	@Failure		401		{object}	response.BaseResponse[any]		"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]		"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]		"User not found"
//	@Failure		500		{object}	response.BaseResponse[any]		"Internal server error"
//	@Router			/admin/users/{userId} [get]
func (h *UserHandler) Get(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	user, err := h.service.Get(c.Request.Context(), userID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toUserResponse(user))
}

// Create creates a new user.
//
//	@Summary		Create user
//	@Description	Creates a new user and assigns them to one or more groups. Requires admin privileges.
//	@Tags			Admin - Users
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			body	body		CreateUserRequest				true	"User data"
//	@Success		201		{object}	UserResponse					"User created"
//	@Failure		400		{object}	response.BaseResponse[any]		"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]		"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]		"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]		"Group not found"
//	@Failure		409		{object}	response.BaseResponse[any]		"Username already taken"
//	@Failure		500		{object}	response.BaseResponse[any]		"Internal server error"
//	@Router			/admin/users [post]
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

// Update updates a user's username and/or password.
//
//	@Summary		Update user
//	@Description	Updates an existing user's username and/or password. Requires admin privileges.
//	@Tags			Admin - Users
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			userId	path		int								true	"User ID"
//	@Param			body	body		UpdateUserRequest				true	"Fields to update (all optional)"
//	@Success		200		{object}	UserResponse					"User updated"
//	@Failure		400		{object}	response.BaseResponse[any]		"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]		"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]		"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]		"User not found"
//	@Failure		409		{object}	response.BaseResponse[any]		"Username already taken"
//	@Failure		500		{object}	response.BaseResponse[any]		"Internal server error"
//	@Router			/admin/users/{userId} [patch]
func (h *UserHandler) Update(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
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

// Delete deletes a user by ID.
//
//	@Summary		Delete user
//	@Description	Deletes a user by their ID. Requires admin privileges.
//	@Tags			Admin - Users
//	@Produce		json
//	@Security		CookieAuth
//	@Param			userId	path	int	true	"User ID"
//	@Success		204		"User deleted"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid user ID"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"User not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"User must have at least one group"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/users/{userId} [delete]
func (h *UserHandler) Delete(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	if err := h.service.Delete(c.Request.Context(), userID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

// SetRole sets the role of a user.
//
//	@Summary		Set user role
//	@Description	Updates the role of a user (e.g. "admin" or "user"). Requires admin privileges.
//	@Tags			Admin - Users
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			userId	path		int								true	"User ID"
//	@Param			body	body		SetRoleRequest					true	"Role to set"
//	@Success		200		{object}	UserResponse					"Role updated"
//	@Failure		400		{object}	response.BaseResponse[any]		"Invalid input or invalid role"
//	@Failure		401		{object}	response.BaseResponse[any]		"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]		"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]		"User not found"
//	@Failure		500		{object}	response.BaseResponse[any]		"Internal server error"
//	@Router			/admin/users/{userId}/role [patch]
func (h *UserHandler) SetRole(c *gin.Context) {
	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
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
		errors.Is(err, domain.ErrInvalidPagination),
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
