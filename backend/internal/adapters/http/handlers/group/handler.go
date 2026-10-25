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

func (h *GroupHandler) List(c *gin.Context) {
	params, err := ginutil.ParsePagination(c)
	if err != nil {
		writeError(c, domain.ErrInvalidPagination)
		return
	}

	groups, totalData, err := h.service.List(c.Request.Context(), params)
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

func (h *GroupHandler) Get(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	group, err := h.service.Get(c.Request.Context(), groupID)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGroupResponse(group))
}

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

	group, err := h.service.Create(c.Request.Context(), input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusCreated, toGroupResponse(group))
}

func (h *GroupHandler) Update(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, err)
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

	group, err := h.service.Update(c.Request.Context(), groupID, input)
	if err != nil {
		writeError(c, err)
		return
	}

	c.JSON(http.StatusOK, toGroupResponse(group))
}

func (h *GroupHandler) Delete(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	if err := h.service.Delete(c.Request.Context(), groupID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *GroupHandler) AddUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, err)
		return
	}

	if err := h.service.AddUser(c.Request.Context(), groupID, userID); err != nil {
		writeError(c, err)
		return
	}

	c.Status(http.StatusNoContent)
}

func (h *GroupHandler) RemoveUser(c *gin.Context) {
	groupID, err := ginutil.ParseUintParam(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	userID, err := ginutil.ParseUintParam(c, "userId")
	if err != nil {
		writeError(c, err)
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
