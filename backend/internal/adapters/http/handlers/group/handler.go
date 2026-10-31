package group

import (
	"errors"
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type GroupHandler struct {
	service ports.GroupService
}

func NewGroupHandler(service ports.GroupService) *GroupHandler {
	return &GroupHandler{service: service}
}

// List returns a paginated list of groups.
//
//	@Summary		List groups
//	@Description	Returns a paginated list of all groups. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			page	query		int									false	"Page number (default: 1)"
//	@Param			limit	query		int									false	"Items per page (default: 10)"
//	@Success		200		{object}	response.PagedResponse[GroupResponse]	"Groups retrieved"
//	@Failure		400		{object}	response.BaseResponse[any]				"Invalid pagination"
//	@Failure		401		{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]				"Forbidden (not admin)"
//	@Failure		500		{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/admin/groups [get]
func (h *GroupHandler) List(c *gin.Context) {
	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeError(c, domain.ErrInvalidPagination)
		return
	}

	groups, totalData, err := h.service.GetAllGroups(c.Request.Context(), params)
	if err != nil {
		writeError(c, err)
		return
	}

	data := make([]GroupResponse, 0, len(groups))

	for _, group := range groups {
		data = append(data, toGroupResponse(group))
	}

	c.JSON(http.StatusOK, response.NewPagedResponse(
		http.StatusOK,
		"groups retrieved successfully",
		data,
		params,
		totalData,
	))
}

// Get returns a single group by ID.
//
//	@Summary		Get group
//	@Description	Returns a single group by its ID. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			groupId	path		int							true	"Group ID"
//	@Success		200		{object}	GroupResponse				"Group found"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid group ID"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Group not found"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups/{groupId} [get]
func (h *GroupHandler) Get(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	group, err := h.service.GetGroupByID(c.Request.Context(), groupID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGroupResponse(group))
}

// Create creates a new group.
//
//	@Summary		Create group
//	@Description	Creates a new group with database access permissions. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			body	body		CreateGroupRequest			true	"Group data"
//	@Success		201		{object}	GroupResponse				"Group created"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid input or invalid database access type"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Database not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Group name already taken"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups [post]
func (h *GroupHandler) Create(c *gin.Context) {
	var request CreateGroupRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	input := domain.CreateGroupInput{
		Name:             request.Name,
		DatabaseAccesses: toDatabaseAccesses(request.DatabaseAccesses),
	}

	group, err := h.service.CreateGroup(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toGroupResponse(group))
}

// Update updates a group's name and/or database access permissions.
//
//	@Summary		Update group
//	@Description	Updates an existing group. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			groupId	path		int							true	"Group ID"
//	@Param			body	body		UpdateGroupRequest			true	"Fields to update (all optional)"
//	@Success		200		{object}	GroupResponse				"Group updated"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Group not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Group name already taken"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups/{groupId} [patch]
func (h *GroupHandler) Update(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	var request UpdateGroupRequest

	if err := c.ShouldBindJSON(&request); err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	input := domain.UpdateGroupInput{
		Name: request.Name,
	}

	if request.DatabaseAccesses != nil {
		accesses := toDatabaseAccesses(*request.DatabaseAccesses)
		input.DatabaseAccesses = &accesses
	}

	group, err := h.service.UpdateGroup(c.Request.Context(), groupID, input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGroupResponse(group))
}

// Delete deletes a group by ID.
//
//	@Summary		Delete group
//	@Description	Deletes a group by its ID. Requires admin privileges.
//	@Tags			Admin - Groups
//	@Produce		json
//	@Security		CookieAuth
//	@Param			groupId	path	int	true	"Group ID"
//	@Success		204		"Group deleted"
//	@Failure		400		{object}	response.BaseResponse[any]	"Invalid group ID"
//	@Failure		401		{object}	response.BaseResponse[any]	"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]	"Forbidden (not admin)"
//	@Failure		404		{object}	response.BaseResponse[any]	"Group not found"
//	@Failure		409		{object}	response.BaseResponse[any]	"Group still has users / must have database"
//	@Failure		500		{object}	response.BaseResponse[any]	"Internal server error"
//	@Router			/admin/groups/{groupId} [delete]
func (h *GroupHandler) Delete(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, domain.ErrInvalidInput)
		return
	}

	if err := h.service.DeleteGroup(c.Request.Context(), groupID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
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
func (h *GroupHandler) AddUser(c *gin.Context) {
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

	if err := h.service.AddUser(c.Request.Context(), groupID, userID); err != nil {
		writeError(c, err)
		return
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
func (h *GroupHandler) RemoveUser(c *gin.Context) {
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

	if err := h.service.RemoveUser(c.Request.Context(), groupID, userID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func writeError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidInput),
		errors.Is(err, domain.ErrInvalidPagination),
		errors.Is(err, domain.ErrInvalidDatabaseAccess),
		errors.Is(err, domain.ErrInvalidGroupsIDs):
		c.JSON(http.StatusBadRequest, response.Error(err.Error()))

	case errors.Is(err, domain.ErrGroupNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrDatabaseNotFound):
		c.JSON(http.StatusNotFound, response.Error(err.Error()))

	case errors.Is(err, domain.ErrGroupNameTaken),
		errors.Is(err, domain.ErrMembershipExists),
		errors.Is(err, domain.ErrMembershipNotFound),
		errors.Is(err, domain.ErrGroupMustHaveDatabase),
		errors.Is(err, domain.ErrUserMustHaveGroup):
		c.JSON(http.StatusConflict, response.Error(err.Error()))

	default:
		c.JSON(http.StatusInternalServerError, response.Error("internal server error"))
	}
}
