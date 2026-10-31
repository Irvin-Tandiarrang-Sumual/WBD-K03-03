package usergroup

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type UserGroupHandler struct {
	service ports.UserGroupService
}

func NewUserGroupHandler(service ports.UserGroupService) *UserGroupHandler {
	return &UserGroupHandler{
		service: service,
	}
}

// AddUser adds a user to a group.
//
//	@Summary		Add user to group
//	@Description	Adds an existing user to an existing group. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			groupId	path	int	true	"Group ID"
//	@Param			userId	path	int	true	"User ID"
//	@Success		204		"User added to group"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Group or user not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Membership already exists"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups/{groupId}/users/{userId} [post]
func (h *UserGroupHandler) AddUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	if h.service != nil {
		if err := h.service.AddUser(c.Request.Context(), groupID, userID); err != nil {
			writeError(c, err)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

// RemoveUser removes a user from a group.
//
//	@Summary		Remove user from group
//	@Description	Removes an existing user from an existing group. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			groupId	path	int	true	"Group ID"
//	@Param			userId	path	int	true	"User ID"
//	@Success		204		"User removed from group"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Group or user not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Membership not found / user must remain in a group"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups/{groupId}/users/{userId} [delete]
func (h *UserGroupHandler) RemoveUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	if h.service != nil {
		if err := h.service.RemoveUser(c.Request.Context(), groupID, userID); err != nil {
			writeError(c, err)
			return
		}
	}

	c.Status(http.StatusNoContent)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput):
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))

	case errors.Is(err, domain.ErrGroupNotFound),
		errors.Is(err, domain.ErrUserNotFound):
		c.JSON(http.StatusNotFound, response.Error(err.Error()))

	case errors.Is(err, domain.ErrMembershipExists),
		errors.Is(err, domain.ErrMembershipNotFound),
		errors.Is(err, domain.ErrUserMustHaveGroup):
		c.JSON(http.StatusConflict, response.Error(err.Error()))

	default:
		c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}
}
