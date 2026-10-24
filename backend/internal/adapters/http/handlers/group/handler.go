package group

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/gin-gonic/gin"
)

type Handler struct {
	service ports.GroupService
}

func NewHandler(service ports.GroupService) *Handler {
	return &Handler{service: service}
}

func (h *Handler) List(c *gin.Context) {
	groups, err := h.service.List(c.Request.Context())
	if err != nil {
		writeError(c, err)
		return
	}

	response := make([]GroupResponse, 0, len(groups))

	for _, group := range groups {
		response = append(response, toGroupResponse(group))
	}

	c.JSON(http.StatusOK, response)
}

func (h *Handler) Get(c *gin.Context) {
	groupID, err := parseID(c, "groupId")
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

func (h *Handler) Create(c *gin.Context) {
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

func (h *Handler) Update(c *gin.Context) {
	groupID, err := parseID(c, "groupId")
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

func (h *Handler) Delete(c *gin.Context) {
	groupID, err := parseID(c, "groupId")
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

func (h *Handler) AddUser(c *gin.Context) {
	groupID, err := parseID(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	userID, err := parseID(c, "userId")
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

func (h *Handler) RemoveUser(c *gin.Context) {
	groupID, err := parseID(c, "groupId")
	if err != nil {
		writeError(c, err)
		return
	}

	userID, err := parseID(c, "userId")
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
		errors.Is(err, domain.ErrInvalidDatabaseAccess),
		errors.Is(err, domain.ErrInvalidGroupsIDs):
		c.JSON(http.StatusBadRequest, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrGroupNotFound),
		errors.Is(err, domain.ErrUserNotFound),
		errors.Is(err, domain.ErrDatabaseNotFound):
		c.JSON(http.StatusNotFound, errorResponse{Error: err.Error()})

	case errors.Is(err, domain.ErrGroupNameTaken),
		errors.Is(err, domain.ErrMembershipExists),
		errors.Is(err, domain.ErrMembershipNotFound),
		errors.Is(err, domain.ErrGroupMustHaveDatabase),
		errors.Is(err, domain.ErrUserMustHaveGroup):
		c.JSON(http.StatusConflict, errorResponse{Error: err.Error()})

	default:
		c.JSON(http.StatusInternalServerError, errorResponse{
			Error: "internal server error",
		})
	}
}