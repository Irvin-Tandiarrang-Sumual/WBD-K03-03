package handlers

import (
	"net/http"

	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/adapters/http/middleware"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/domain"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/internal/core/ports"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/ginutil"
	"github.com/Irvin-Tandiarrang-Sumual/WBD-K03-03/pkg/response"
	"github.com/gin-gonic/gin"
)

type DraftHandler struct {
	service ports.DraftService
}

func NewDraftHandler(service ports.DraftService) *DraftHandler {
	return &DraftHandler{service: service}
}

// CreateDraft creates a new migration draft for a database.
//
//	@Summary		Create draft
//	@Description	Creates a new migration draft on the specified database and initializes approvals for all connected group leaders. Requires edit access.
//	@Tags			Me - Drafts
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		int									true	"Database ID"
//	@Param			body	body		CreateDraftRequest					true	"Draft details"
//	@Success		201		{object}	response.BaseResponse[DraftResponse]	"Draft created successfully"
//	@Failure		400		{object}	response.BaseResponse[any]			"Invalid input"
//	@Failure		401		{object}	response.BaseResponse[any]			"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]			"Forbidden (no edit access)"
//	@Failure		500		{object}	response.BaseResponse[any]			"Internal server error"
//	@Router			/databases/{id}/drafts [post]
func (h *DraftHandler) CreateDraft(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		databaseID, err = ginutil.ParseUintParam(c, "database_id")
		if err != nil {
			c.Error(domain.ErrInvalidDatabaseID)
			return
		}
	}

	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req CreateDraftRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	input := domain.CreateDraftInput{
		DatabaseID:         databaseID,
		UserID:             userID,
		SqlUp:              req.SqlUp,
		SqlDown:            req.SqlDown,
		IsReversible:       req.IsReversible,
		IrreversibleReason: req.IrreversibleReason,
		IsBatch:            req.IsBatch,
		BatchColumn:        req.BatchColumn,
		BatchSize:          req.BatchSize,
	}

	createdDraft, err := h.service.CreateDraft(c.Request.Context(), input)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusCreated, response.Success("draft created successfully", toDraftResponse(createdDraft)))
}

// GetDraftsByDatabaseID returns paginated draft history for a database.
//
//	@Summary		List database drafts
//	@Description	Returns a paginated list of draft history for a specific database. Requires view access.
//	@Tags			Me - Drafts
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		int										true	"Database ID"
//	@Param			page	query		int										false	"Page number (default: 1)"
//	@Param			limit	query		int										false	"Items per page (default: 10)"
//	@Success		200		{object}	response.PagedResponse[DraftResponse]	"Drafts retrieved successfully"
//	@Failure		400		{object}	response.BaseResponse[any]				"Invalid database ID or pagination"
//	@Failure		401		{object}	response.BaseResponse[any]				"Unauthorized"
//	@Failure		403		{object}	response.BaseResponse[any]				"Forbidden (no view access)"
//	@Failure		500		{object}	response.BaseResponse[any]				"Internal server error"
//	@Router			/databases/{id}/drafts [get]
func (h *DraftHandler) GetDraftsByDatabaseID(c *gin.Context) {
	databaseID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		databaseID, err = ginutil.ParseUintParam(c, "database_id")
		if err != nil {
			c.Error(domain.ErrInvalidDatabaseID)
			return
		}
	}

	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	params, err := ginutil.ParsePagination(c)
	if err != nil {
		c.Error(domain.ErrInvalidPagination)
		return
	}

	drafts, totalData, err := h.service.GetDraftsByDatabaseID(c.Request.Context(), databaseID, userID, params)
	if err != nil {
		c.Error(err)
		return
	}

	data := make([]DraftResponse, 0, len(drafts))
	for _, d := range drafts {
		data = append(data, toDraftResponse(d))
	}

	c.JSON(http.StatusOK, response.NewPagedResponse(http.StatusOK, "drafts retrieved successfully", data, params, totalData))
}

// GetDraftByID returns details of a single draft including vote status of group leaders.
//
//	@Summary		Get draft details
//	@Description	Returns draft details and leader approval voting statuses. Requires view access to the database.
//	@Tags			Me - Drafts
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id	path		int									true	"Draft ID"
//	@Success		200	{object}	response.BaseResponse[DraftResponse]	"Draft retrieved successfully"
//	@Failure		400	{object}	response.BaseResponse[any]			"Invalid draft ID"
//	@Failure		401	{object}	response.BaseResponse[any]			"Unauthorized"
//	@Failure		403	{object}	response.BaseResponse[any]			"Forbidden"
//	@Failure		404	{object}	response.BaseResponse[any]			"Draft not found"
//	@Failure		500	{object}	response.BaseResponse[any]			"Internal server error"
//	@Router			/drafts/{id} [get]
func (h *DraftHandler) GetDraftByID(c *gin.Context) {
	draftID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	d, err := h.service.GetDraftByID(c.Request.Context(), draftID, userID)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success("draft retrieved successfully", toDraftResponse(d)))
}

// SubmitApproval submits an approval or rejection vote by a connected group leader.
//
//	@Summary		Submit draft approval
//	@Description	Action voting by a connected group leader (status: "approved" | "rejected").
//	@Tags			Me - Drafts
//	@Accept			json
//	@Produce		json
//	@Security		CookieAuth
//	@Param			id		path		int									true	"Draft ID"
//	@Param			body	body		SubmitApprovalRequest				true	"Approval vote"
//	@Success		200		{object}	response.BaseResponse[DraftResponse]	"Vote submitted successfully"
//	@Failure		400		{object}	response.BaseResponse[any]			"Invalid input or status"
//	@Failure		401		{object}	response.BaseResponse[any]			"Unauthorized (voter not group leader)"
//	@Failure		404		{object}	response.BaseResponse[any]			"Draft not found"
//	@Failure		409		{object}	response.BaseResponse[any]			"Conflict (draft not in pending_approval state)"
//	@Failure		500		{object}	response.BaseResponse[any]			"Internal server error"
//	@Router			/drafts/{id}/approvals [post]
func (h *DraftHandler) SubmitApproval(c *gin.Context) {
	draftID, err := ginutil.ParseUintParam(c, "id")
	if err != nil {
		c.Error(domain.ErrInvalidInput)
		return
	}

	userID, err := middleware.CurrentUserID(c)
	if err != nil {
		c.Error(err)
		return
	}

	var req SubmitApprovalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(err)
		return
	}

	input := domain.SubmitApprovalInput{
		DraftID: draftID,
		UserID:  userID,
		Status:  req.Status,
	}

	updatedDraft, err := h.service.SubmitApproval(c.Request.Context(), input)
	if err != nil {
		c.Error(err)
		return
	}

	c.JSON(http.StatusOK, response.Success("approval submitted successfully", toDraftResponse(updatedDraft)))
}
