package handlers

import (
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
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
//	@Param			id	path	int	true	"Group ID"
//	@Param			user_id	path	int	true	"User ID"
//	@Success		204		"User added to group"
//	@Failure		400		"Invalid input"
//	@Failure		401		"Unauthorized"
//	@Failure		403		"Forbidden (not admin)"
//	@Failure		404		"Group or user not found"
//	@Failure		409		"Membership already exists"
//	@Failure		500		"Internal server error"
//	@Router			/admin/groups/{id}/users/{user_id} [post]
func (h *UserGroupHandler) AddUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "user_id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if h.service != nil {
		if err := h.service.AddUser(c.Request.Context(), groupID, userID); err != nil {
			c.Error(err)
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
//	@Param			id	path	int	true	"Group ID"
//	@Param			user_id	path	int	true	"User ID"
//	@Success		204		"User removed from group"
//	@Failure		400		"Invalid input"
//	@Failure		401		"Unauthorized"
//	@Failure		403		"Forbidden (not admin)"
//	@Failure		404		"Group or user not found"
//	@Failure		409		"Membership not found / user must remain in a group"
//	@Failure		500		"Internal server error"
//	@Router			/admin/groups/{id}/users/{user_id} [delete]
func (h *UserGroupHandler) RemoveUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "user_id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	if h.service != nil {
		if err := h.service.RemoveUser(c.Request.Context(), groupID, userID); err != nil {
			c.Error(err)
			return
		}
	}

	c.Status(http.StatusNoContent)
}
